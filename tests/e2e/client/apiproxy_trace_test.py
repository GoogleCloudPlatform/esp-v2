#!/usr/bin/env python3

# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Automated Cloud Trace E2E assertion script for ESPv2.

Verifies end-to-end W3C trace context propagation (traceparent header)
through ESPv2 to the backend and into Google Cloud Trace v2.

Uses Python 3 standard library only without external pip dependencies.
"""

import argparse
import http.client
import json
import os
import ssl
import subprocess
import sys
import time
import unittest
import urllib.error
import urllib.parse
import urllib.request
import uuid


def generate_w3c_traceparent():
    """Generates a 32-hex-char trace ID and 16-hex-char span ID.

    Returns:
        tuple of (trace_id, span_id, traceparent_header):
            trace_id: 32 lowercase hex characters (128-bit ID).
            span_id: 16 lowercase hex characters (64-bit ID).
            traceparent_header: W3C traceparent string formatted as '00-${trace_id}-${span_id}-01'.
    """
    trace_id = uuid.uuid4().hex.lower()
    span_id = uuid.uuid4().hex[:16].lower()
    traceparent_header = f"00-{trace_id}-{span_id}-01"
    return trace_id, span_id, traceparent_header


def get_gcp_access_token() -> str:
    """Resolves an OAuth2 access token for Google APIs.

    Tries:
    1. 'gcloud auth print-access-token' via subprocess.
    2. GCE metadata server:
       http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token

    Returns:
        str: OAuth2 Bearer access token.

    Raises:
        RuntimeError: If neither gcloud nor the metadata server is reachable.
    """
    # 1. Try gcloud auth print-access-token
    try:
        res = subprocess.run(
            ['gcloud', 'auth', 'print-access-token'],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            check=True,
        )
        token = res.stdout.strip()
        if token:
            return token
    except (subprocess.SubprocessError, FileNotFoundError, OSError):
        pass

    # 2. Fallback: GCE metadata server
    metadata_url = (
        'http://metadata.google.internal/computeMetadata/v1/instance/'
        'service-accounts/default/token'
    )
    try:
        req = urllib.request.Request(
            metadata_url,
            headers={'Metadata-Flavor': 'Google'},
        )
        with urllib.request.urlopen(req, timeout=5) as resp:
            data = json.loads(resp.read().decode('utf-8'))
            token = data.get('access_token', '').strip()
            if token:
                return token
    except (urllib.error.URLError, OSError, ValueError, KeyError):
        pass

    raise RuntimeError(
        "Failed to resolve GCP access token via gcloud or GCE metadata server."
    )


def poll_cloud_trace(
    project_id: str,
    trace_id: str,
    access_token: str,
    timeout_sec: int = 60,
    poll_interval_sec: int = 5,
    verbose: bool = False,
) -> dict:
    """Polls Google Cloud Trace v2 REST API for the trace object.

    Handles 404s and empty span lists gracefully with sleep/retry until
    timeout or trace found.

    Args:
        project_id: GCP project ID.
        trace_id: 32-hex-char trace ID.
        access_token: OAuth2 access token.
        timeout_sec: Max duration to poll before raising TimeoutError.
        poll_interval_sec: Seconds to sleep between polling attempts.
        verbose: If True, prints polling debug info.

    Returns:
        dict: The Cloud Trace v2 Trace JSON object containing spans.

    Raises:
        TimeoutError: If no trace with spans is found before timeout_sec.
    """
    url = f"https://cloudtrace.googleapis.com/v2/projects/{project_id}/traces/{trace_id}"
    headers = {
        'Authorization': f"Bearer {access_token}",
        'Accept': 'application/json',
    }

    start_time = time.time()
    while True:
        try:
            req = urllib.request.Request(url, headers=headers)
            with urllib.request.urlopen(req, timeout=10) as resp:
                data = json.loads(resp.read().decode('utf-8'))
                spans = data.get('spans', [])
                if spans:
                    if verbose:
                        print(f"Found trace {trace_id} with {len(spans)} spans.")
                    return data
                if verbose:
                    print(f"Trace {trace_id} returned empty spans, retrying...")
        except urllib.error.HTTPError as e:
            if e.code == 404:
                if verbose:
                    print(f"Trace {trace_id} not found (404), retrying...")
            else:
                raise
        except (urllib.error.URLError, OSError) as e:
            if verbose:
                print(f"Transient error fetching trace {trace_id}: {e}, retrying...")

        elapsed = time.time() - start_time
        if elapsed >= timeout_sec:
            raise TimeoutError(
                f"Timed out after {timeout_sec}s polling Cloud Trace for trace ID "
                f"'{trace_id}' in project '{project_id}'."
            )

        sleep_duration = min(poll_interval_sec, max(0.1, timeout_sec - elapsed))
        time.sleep(sleep_duration)


def _extract_attribute_value(val):
    """Extracts scalar attribute value from Cloud Trace v2 AttributeValue structure."""
    if isinstance(val, dict):
        if 'intValue' in val:
            return val['intValue']
        if 'stringValue' in val:
            sv = val['stringValue']
            if isinstance(sv, dict):
                return sv.get('value')
            return sv
        if 'value' in val:
            return val['value']
    return val


def _has_http_status_attribute(span: dict, expected_status=200) -> bool:
    """Checks if span contains HTTP status code matching expected_status."""
    attrs = span.get('attributes', {})
    attr_map = attrs.get('attributeMap', attrs) if isinstance(attrs, dict) else {}
    if not isinstance(attr_map, dict):
        return False

    status_keys = {
        '/http/status_code',
        'http.status_code',
        'http.response.status_code',
        'status_code',
        '/http/status',
        'http.status',
        'g.co/http/status_code',
    }

    for k, v in attr_map.items():
        if k in status_keys or 'status_code' in k.lower():
            raw_val = _extract_attribute_value(v)
            if raw_val is not None:
                if expected_status is None:
                    return True
                try:
                    if int(raw_val) == int(expected_status):
                        return True
                except (ValueError, TypeError):
                    if str(raw_val) == str(expected_status):
                        return True
    return False


def verify_trace_spans(
    trace_json: dict,
    expected_trace_id: str,
    expected_client_span_id: str,
    expected_status=200,
    verbose: bool = False,
) -> bool:
    """Validates Cloud Trace v2 JSON structure.

    Checks:
    - Trace name ends with expected_trace_id.
    - ESPv2 ingress span exists with parentSpanId == expected_client_span_id.
    - Bookstore backend child span exists with parentSpanId == espv2_span_id.
    - HTTP status code attribute (e.g. 200) is present on spans.

    Args:
        trace_json: Parsed JSON from Cloud Trace v2 REST API.
        expected_trace_id: 32-hex-char trace ID.
        expected_client_span_id: 16-hex-char client span ID.
        expected_status: Expected HTTP status code on spans (default 200).
        verbose: If True, logs verification details.

    Returns:
        bool: True if all assertions hold, False otherwise.
    """
    if not isinstance(trace_json, dict):
        if verbose:
            print("Trace JSON is not a dictionary.")
        return False

    trace_name = str(trace_json.get('name', ''))
    if not trace_name.lower().endswith(expected_trace_id.lower()):
        if verbose:
            print(
                f"Trace name mismatch: got '{trace_name}', expected to end with "
                f"'{expected_trace_id}'."
            )
        return False

    spans = trace_json.get('spans', [])
    if not spans or not isinstance(spans, list):
        if verbose:
            print("No spans found in trace JSON.")
        return False

    def get_span_id(s):
        return str(s.get('spanId') or s.get('span_id') or '')

    def get_parent_span_id(s):
        return str(s.get('parentSpanId') or s.get('parent_span_id') or '')

    # 1. Find ESPv2 ingress span where parentSpanId == expected_client_span_id
    matched_pair = None
    for espv2_span in spans:
        parent_id = get_parent_span_id(espv2_span)
        if parent_id.lower() == expected_client_span_id.lower():
            espv2_span_id = get_span_id(espv2_span)
            if not espv2_span_id:
                continue

            # 2. Find Bookstore backend child span where parentSpanId == espv2_span_id
            for backend_span in spans:
                if backend_span is espv2_span:
                    continue
                backend_parent_id = get_parent_span_id(backend_span)
                if backend_parent_id.lower() == espv2_span_id.lower():
                    # 3. Check HTTP status code attribute on spans
                    if _has_http_status_attribute(
                        espv2_span, expected_status
                    ) and _has_http_status_attribute(backend_span, expected_status):
                        matched_pair = (espv2_span, backend_span)
                        break
            if matched_pair:
                break

    if not matched_pair:
        if verbose:
            print(
                f"Failed to find valid ESPv2 ingress and backend child span chain "
                f"matching client span '{expected_client_span_id}'."
            )
        return False

    if verbose:
        espv2_s, backend_s = matched_pair
        print("Trace verification succeeded:")
        print(f"  Trace ID: {expected_trace_id}")
        print(
            f"  ESPv2 span ID: {get_span_id(espv2_s)} (parent: "
            f"{get_parent_span_id(espv2_s)})"
        )
        print(
            f"  Backend span ID: {get_span_id(backend_s)} (parent: "
            f"{get_parent_span_id(backend_s)})"
        )
    return True


def send_traced_request(
    host: str,
    traceparent_header: str,
    path: str = '/shelves',
    method: str = 'GET',
    api_key: str = None,
    auth_token: str = None,
    host_header: str = None,
    timeout: int = 15,
    verbose: bool = False,
):
    """Sends an HTTP request with the W3C traceparent header to ESPv2."""
    if not host.startswith('http://') and not host.startswith('https://'):
        host = 'http://' + host
    url = host.rstrip('/') + path
    if api_key:
        delimiter = '&' if '?' in url else '?'
        url += f"{delimiter}key={api_key}"

    headers = {
        'traceparent': traceparent_header,
        'Accept': 'application/json',
    }
    if host_header:
        headers['Host'] = host_header
    if auth_token:
        headers['Authorization'] = f"Bearer {auth_token}"

    if verbose:
        print(f"Sending HTTP {method} to {url}")
        print(f"Headers: {headers}")

    req = urllib.request.Request(url, headers=headers, method=method)
    ssl_ctx = None
    if url.startswith('https://'):
        ssl_ctx = ssl.create_default_context()
        ssl_ctx.check_hostname = False
        ssl_ctx.verify_mode = ssl.CERT_NONE

    try:
        with urllib.request.urlopen(req, timeout=timeout, context=ssl_ctx) as resp:
            status = resp.status
            body = resp.read().decode('utf-8')
            if verbose:
                print(f"Received status {status}, body length: {len(body)}")
            return status, body
    except urllib.error.HTTPError as e:
        body = e.read().decode('utf-8') if e.fp else ''
        if verbose:
            print(f"Received HTTP error {e.code}, body: {body}")
        return e.code, body



def verify_trace_propagation_via_version_endpoint(
    host: str,
    trace_id: str,
    client_span_id: str,
    host_header: str = None,
    timeout: int = 15,
    verbose: bool = False,
) -> bool:
    """Verifies in-band W3C trace context extraction and propagation to backend.

    Sends GET /version to ESPv2 with the client traceparent header. The Bookstore
    backend's /version endpoint returns its received HTTP request headers as JSON.
    This function validates:
    1. HTTP response status is 200.
    2. Response body is valid JSON containing a 'traceparent' header.
    3. The propagated trace ID matches expected trace_id.
    4. The propagated span ID is a valid 16-hex character string and differs
       from client_span_id (confirming ESPv2 generated its own downstream span).

    Args:
        host: ESPv2 endpoint URL or hostname.
        trace_id: 32-hex-char client trace ID.
        client_span_id: 16-hex-char client span ID.
        host_header: Optional custom Host header.
        timeout: Request timeout in seconds.
        verbose: If True, prints debug output.

    Returns:
        bool: True if trace context propagation was verified, False otherwise.
    """
    traceparent_header = f"00-{trace_id}-{client_span_id}-01"
    status, body = send_traced_request(
        host=host,
        traceparent_header=traceparent_header,
        path='/version',
        method='GET',
        host_header=host_header,
        timeout=timeout,
        verbose=verbose,
    )
    if status != 200:
        if verbose:
            print(
                f"/version returned HTTP status {status}, expected 200. Body: {body}",
                file=sys.stderr,
            )
        return False

    try:
        headers_dict = json.loads(body)
    except (json.JSONDecodeError, ValueError) as e:
        if verbose:
            print(
                f"Failed to parse JSON response from /version: {e}. Body: {body}",
                file=sys.stderr,
            )
        return False

    if not isinstance(headers_dict, dict):
        if verbose:
            print(
                f"/version response is not a JSON object: {headers_dict}",
                file=sys.stderr,
            )
        return False

    # Find traceparent header (case-insensitive)
    echoed_traceparent = None
    for k, v in headers_dict.items():
        if k.lower() == 'traceparent':
            echoed_traceparent = v
            break

    if not echoed_traceparent:
        if verbose:
            print(
                f"/version response did not contain 'traceparent' header. Headers: {headers_dict}",
                file=sys.stderr,
            )
        return False

    if isinstance(echoed_traceparent, list):
        echoed_traceparent = echoed_traceparent[0]
    echoed_traceparent = str(echoed_traceparent).strip()

    parts = echoed_traceparent.split('-')
    if len(parts) != 4:
        if verbose:
            print(
                f"Malformed traceparent header format: '{echoed_traceparent}'",
                file=sys.stderr,
            )
        return False

    version, echoed_trace_id, echoed_span_id, _ = parts
    if version != '00':
        if verbose:
            print(
                f"Unexpected W3C traceparent version: '{version}' (expected '00')",
                file=sys.stderr,
            )
        return False

    if echoed_trace_id.lower() != trace_id.lower():
        if verbose:
            print(
                f"Trace ID mismatch in propagated header: got '{echoed_trace_id}', "
                f"expected '{trace_id}'",
                file=sys.stderr,
            )
        return False

    if len(echoed_span_id) != 16 or not all(c in '0123456789abcdefABCDEF' for c in echoed_span_id):
        if verbose:
            print(
                f"Invalid downstream span ID in propagated header: '{echoed_span_id}'",
                file=sys.stderr,
            )
        return False

    if echoed_span_id.lower() == client_span_id.lower():
        if verbose:
            print(
                f"Downstream span ID was not mutated by ESPv2: got '{echoed_span_id}', "
                f"matches client span ID '{client_span_id}'",
                file=sys.stderr,
            )
        return False

    if verbose:
        print(
            f"Successfully verified in-band trace propagation via /version: "
            f"trace_id={trace_id}, client_span_id={client_span_id} -> "
            f"downstream_span_id={echoed_span_id}"
        )
    return True


def run_trace_e2e_test(
    host: str,
    project_id: str,
    api_key: str = None,
    auth_token: str = None,
    host_header: str = None,
    timeout_sec: int = 60,
    delay_sec: int = 5,
    verbose: bool = False,
    path: str = '/shelves',
) -> bool:
    """Executes the full E2E trace assertion workflow with dual verification."""
    trace_id, client_span_id, traceparent = generate_w3c_traceparent()
    if verbose:
        print(
            f"Generated W3C trace: trace_id={trace_id}, "
            f"client_span_id={client_span_id}"
        )
        print(f"traceparent header: {traceparent}")

    # --- Verification 1: In-Band Header Propagation via /version ---
    if verbose:
        print("Starting Verification 1: In-band W3C trace context propagation via /version...")
    header_ok = verify_trace_propagation_via_version_endpoint(
        host=host,
        trace_id=trace_id,
        client_span_id=client_span_id,
        host_header=host_header,
        timeout=15,
        verbose=verbose,
    )
    if not header_ok:
        print(
            "Verification 1 FAILED: Direct trace context propagation via /version failed.",
            file=sys.stderr,
        )
        return False
    print("Verification 1 SUCCESS: In-band W3C trace context propagation verified via /version.")

    # --- Verification 2: Out-of-Band Cloud Trace Export ---
    if verbose:
        print("Starting Verification 2: Out-of-band Google Cloud Trace export...")
    status, _ = send_traced_request(
        host=host,
        traceparent_header=traceparent,
        path=path,
        api_key=api_key,
        auth_token=auth_token,
        host_header=host_header,
        verbose=verbose,
    )
    if verbose:
        print(f"ESPv2 response status for {path}: {status}")

    if delay_sec > 0:
        time.sleep(delay_sec)

    try:
        if auth_token and auth_token.startswith('ya29.'):
            access_token = auth_token
        else:
            access_token = get_gcp_access_token()
    except Exception as e:
        if auth_token:
            access_token = auth_token
        else:
            print(f"Error resolving GCP access token: {e}", file=sys.stderr)
            return False

    try:
        trace_json = poll_cloud_trace(
            project_id=project_id,
            trace_id=trace_id,
            access_token=access_token,
            timeout_sec=timeout_sec,
            poll_interval_sec=delay_sec,
            verbose=verbose,
        )
    except TimeoutError as e:
        print(f"Trace polling timed out: {e}", file=sys.stderr)
        return False
    except Exception as e:
        print(f"Error polling Cloud Trace: {e}", file=sys.stderr)
        return False

    valid = verify_trace_spans(
        trace_json=trace_json,
        expected_trace_id=trace_id,
        expected_client_span_id=client_span_id,
        expected_status=200 if status == 200 else status,
        verbose=verbose,
    )

    if valid:
        print(f"Verification 2 SUCCESS: Cloud Trace E2E export verified for trace {trace_id}.")
    else:
        print(f"Verification 2 FAILED: Cloud Trace span structure invalid for trace {trace_id}.", file=sys.stderr)
    return valid


class ApiProxyTraceTest(unittest.TestCase):
    """Unittest test case runner for live E2E trace propagation verification."""

    host = None
    project = None
    api_key = None
    auth_token = None
    host_header = None
    timeout = 60
    delay = 5
    verbose = False

    def test_e2e_trace_propagation(self):
        host = (
            self.host
            or os.environ.get('ESP_HOST')
            or os.environ.get('HOST')
        )
        project = (
            self.project
            or os.environ.get('GCP_PROJECT')
            or os.environ.get('PROJECT_ID')
            or os.environ.get('GOOGLE_CLOUD_PROJECT')
        )
        if not host or not project:
            self.skipTest(
                "Skipping live E2E test: --host and --project (or ESP_HOST and "
                "GCP_PROJECT env vars) must be specified."
            )

        success = run_trace_e2e_test(
            host=host,
            project_id=project,
            api_key=self.api_key or os.environ.get('API_KEY'),
            auth_token=self.auth_token,
            host_header=self.host_header or os.environ.get('HOST_HEADER'),
            timeout_sec=self.timeout,
            delay_sec=self.delay,
            verbose=self.verbose,
        )
        self.assertTrue(success, "E2E Cloud Trace validation failed")


def make_argparser():
    parser = argparse.ArgumentParser(
        description="Cloud Trace E2E assertion test for ESPv2."
    )
    parser.add_argument(
        '--host',
        default=None,
        help='Deployed application host name (e.g. http://localhost:8082).',
    )
    parser.add_argument(
        '--project',
        default=None,
        help='GCP Project ID for Cloud Trace.',
    )
    parser.add_argument(
        '--api_key',
        default=None,
        help='Project API key to access service.',
    )
    parser.add_argument(
        '--auth_token',
        default=None,
        help='OAuth2 / JWT auth token for backend or Cloud Trace.',
    )
    parser.add_argument(
        '--host_header',
        default=None,
        help='Deployed application host name or custom domain.',
    )
    parser.add_argument(
        '--timeout',
        type=int,
        default=60,
        help='Max timeout in seconds for polling Cloud Trace.',
    )
    parser.add_argument(
        '--delay',
        type=int,
        default=5,
        help='Delay/poll interval in seconds.',
    )
    parser.add_argument(
        '--verbose',
        action='store_true',
        help='Turn on verbose logging.',
    )
    parser.add_argument(
        '--unittest',
        action='store_true',
        help='Run via unittest test runner.',
    )
    return parser


if __name__ == '__main__':
    parser = make_argparser()
    args, unknown = parser.parse_known_args()

    ApiProxyTraceTest.host = args.host
    ApiProxyTraceTest.project = args.project
    ApiProxyTraceTest.api_key = args.api_key
    ApiProxyTraceTest.auth_token = args.auth_token
    ApiProxyTraceTest.host_header = args.host_header
    ApiProxyTraceTest.timeout = args.timeout
    ApiProxyTraceTest.delay = args.delay
    ApiProxyTraceTest.verbose = args.verbose

    if args.unittest or any(
        a.startswith('Test') or a.startswith('-v') for a in unknown
    ):
        unittest.main(argv=[sys.argv[0]] + [a for a in unknown if a != '--unittest'])
    else:
        if not args.host or not args.project:
            print(
                "Error: Both --host and --project must be specified when running directly.",
                file=sys.stderr,
            )
            parser.print_help(sys.stderr)
            sys.exit(1)

        success = run_trace_e2e_test(
            host=args.host,
            project_id=args.project,
            api_key=args.api_key,
            auth_token=args.auth_token,
            host_header=args.host_header,
            timeout_sec=args.timeout,
            delay_sec=args.delay,
            verbose=args.verbose,
        )
        sys.exit(0 if success else 1)
