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

"""Hermetic unit tests for apiproxy_trace_test.py.

Verifies W3C traceparent generation, GCP access token resolution,
Cloud Trace polling and retries, span hierarchy and attribute validation,
and CLI argument handling using unittest and unittest.mock.
"""

import json
import os
import subprocess
import sys
import unittest
from unittest import mock
import urllib.error
import urllib.request

# Ensure tests/e2e/client is in sys.path
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
if SCRIPT_DIR not in sys.path:
    sys.path.insert(0, SCRIPT_DIR)

from apiproxy_trace_test import (
    generate_w3c_traceparent,
    get_gcp_access_token,
    make_argparser,
    normalize_span_id,
    poll_cloud_trace,
    run_trace_e2e_test,
    send_traced_request,
    verify_trace_propagation_via_version_endpoint,
    verify_trace_spans,
)


class ApiProxyTraceUnitTest(unittest.TestCase):
    """Unit test suite for apiproxy_trace_test."""

    def test_generate_w3c_traceparent(self):
        """Verifies correct 128-bit/64-bit hex length and W3C version 00 format."""
        trace_id, span_id, header = generate_w3c_traceparent()

        # 128-bit hex trace ID: 32 hex chars
        self.assertEqual(len(trace_id), 32)
        self.assertTrue(all(c in '0123456789abcdef' for c in trace_id))
        self.assertNotEqual(trace_id, '0' * 32)

        # 64-bit hex span ID: 16 hex chars
        self.assertEqual(len(span_id), 16)
        self.assertTrue(all(c in '0123456789abcdef' for c in span_id))
        self.assertNotEqual(span_id, '0' * 16)

        # W3C version 00 format: 00-{trace_id}-{span_id}-01
        expected_header = f"00-{trace_id}-{span_id}-01"
        self.assertEqual(header, expected_header)

        parts = header.split('-')
        self.assertEqual(len(parts), 4)
        self.assertEqual(parts[0], '00')
        self.assertEqual(parts[1], trace_id)
        self.assertEqual(parts[2], span_id)
        self.assertEqual(parts[3], '01')

        # Verify uniqueness
        trace_id2, span_id2, header2 = generate_w3c_traceparent()
        self.assertNotEqual(trace_id, trace_id2)
        self.assertNotEqual(span_id, span_id2)
        self.assertNotEqual(header, header2)

    @mock.patch('subprocess.run')
    def test_get_gcp_access_token_gcloud(self, mock_run):
        """Verifies subprocess token retrieval from gcloud."""
        mock_run.return_value = mock.Mock(
            returncode=0,
            stdout='mock-gcloud-access-token\n',
        )

        token = get_gcp_access_token()
        self.assertEqual(token, 'mock-gcloud-access-token')
        mock_run.assert_called_once_with(
            ['gcloud', 'auth', 'print-access-token'],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            check=True,
        )

    @mock.patch('urllib.request.urlopen')
    @mock.patch('subprocess.run')
    def test_get_gcp_access_token_metadata(self, mock_run, mock_urlopen):
        """Verifies metadata server fallback when gcloud is unavailable."""
        # gcloud is missing / fails
        mock_run.side_effect = FileNotFoundError('gcloud not found')

        # Metadata server returns access token json
        mock_resp = mock.MagicMock()
        mock_resp.read.return_value = json.dumps({
            'access_token': 'mock-metadata-server-token',
            'expires_in': 3600,
            'token_type': 'Bearer',
        }).encode('utf-8')
        mock_resp.__enter__.return_value = mock_resp
        mock_urlopen.return_value = mock_resp

        token = get_gcp_access_token()
        self.assertEqual(token, 'mock-metadata-server-token')

        # Verify metadata request headers
        args, _ = mock_urlopen.call_args
        req = args[0]
        self.assertIn('metadata.google.internal', req.full_url)
        self.assertEqual(req.headers.get('Metadata-flavor'), 'Google')

    @mock.patch('urllib.request.urlopen')
    @mock.patch('subprocess.run')
    def test_get_gcp_access_token_failure(self, mock_run, mock_urlopen):
        """Verifies exception raised when both gcloud and metadata server fail."""
        mock_run.side_effect = subprocess.SubprocessError('command failed')
        mock_urlopen.side_effect = urllib.error.URLError('network unreachable')

        with self.assertRaises(RuntimeError):
            get_gcp_access_token()

    @mock.patch('time.sleep')
    @mock.patch('urllib.request.urlopen')
    def test_poll_cloud_trace_success_retry(self, mock_urlopen, mock_sleep):
        """Verifies retry on 404s and empty span lists until success against Cloud Trace v1 URL."""
        # Call 1: 404 Not Found
        err_404 = urllib.error.HTTPError(
            url='https://cloudtrace.googleapis.com/v1/projects/test-project/traces/4bf92f3577b34da6a3ce929d0e0e4736',
            code=404,
            msg='Not Found',
            hdrs={},
            fp=None,
        )

        # Call 2: 200 with empty spans list
        resp_empty = mock.MagicMock()
        resp_empty.read.return_value = json.dumps({'spans': []}).encode('utf-8')
        resp_empty.__enter__.return_value = resp_empty

        # Call 3: 200 with trace and spans
        expected_trace = {
            'name': 'projects/test-project/traces/4bf92f3577b34da6a3ce929d0e0e4736',
            'spans': [{'spanId': '0020000000000001'}],
        }
        resp_success = mock.MagicMock()
        resp_success.read.return_value = json.dumps(expected_trace).encode('utf-8')
        resp_success.__enter__.return_value = resp_success

        mock_urlopen.side_effect = [err_404, resp_empty, resp_success]

        result = poll_cloud_trace(
            project_id='test-project',
            trace_id='4bf92f3577b34da6a3ce929d0e0e4736',
            access_token='mock-token',
            timeout_sec=30,
            poll_interval_sec=2,
        )

        self.assertEqual(result, expected_trace)
        self.assertEqual(mock_urlopen.call_count, 3)
        self.assertEqual(mock_sleep.call_count, 2)

        # Verify that the URL queried uses Cloud Trace REST API v1
        first_req = mock_urlopen.call_args_list[0][0][0]
        self.assertEqual(
            first_req.full_url,
            'https://cloudtrace.googleapis.com/v1/projects/test-project/traces/4bf92f3577b34da6a3ce929d0e0e4736',
        )

    @mock.patch('time.sleep')
    @mock.patch('time.time')
    @mock.patch('urllib.request.urlopen')
    def test_poll_cloud_trace_timeout(self, mock_urlopen, mock_time, mock_sleep):
        """Verifies timeout handling when trace is never ingested."""
        err_404 = urllib.error.HTTPError(
            url='https://cloudtrace.googleapis.com/v1/projects/test-project/traces/4bf92f3577b34da6a3ce929d0e0e4736',
            code=404,
            msg='Not Found',
            hdrs={},
            fp=None,
        )
        mock_urlopen.side_effect = err_404

        # Simulate elapsed time: 0.0, 0.0, 10.0, 10.0, 25.0
        mock_time.side_effect = [0.0, 0.0, 10.0, 10.0, 25.0]

        with self.assertRaises(TimeoutError):
            poll_cloud_trace(
                project_id='test-project',
                trace_id='4bf92f3577b34da6a3ce929d0e0e4736',
                access_token='mock-token',
                timeout_sec=20,
                poll_interval_sec=5,
            )

    def test_normalize_span_id(self):
        """Verifies normalization of decimal strings, hex strings, 0x hex, and ints."""
        # 1. Decimal strings (from Cloud Trace v1 proto3 JSON serialization of uint64)
        self.assertEqual(
            normalize_span_id("10577596205993833633"), "92cb2bb4f3c834a1"
        )
        self.assertEqual(
            normalize_span_id("10580979624564887553"), "92d730e879d61001"
        )
        self.assertEqual(
            normalize_span_id("4503599627370497"), "0010000000000001"
        )

        # 2. Hex strings (W3C headers & Cloud Trace v2)
        self.assertEqual(
            normalize_span_id("92cb2bb4f3c834a1"), "92cb2bb4f3c834a1"
        )
        self.assertEqual(
            normalize_span_id("0010000000000001"), "0010000000000001"
        )

        # 3. Hex with 0x prefix
        self.assertEqual(
            normalize_span_id("0x92cb2bb4f3c834a1"), "92cb2bb4f3c834a1"
        )
        self.assertEqual(
            normalize_span_id("0x1a"), "000000000000001a"
        )

        # 4. Integer input
        self.assertEqual(
            normalize_span_id(4503599627370497), "0010000000000001"
        )

        # 5. Empty / None
        self.assertEqual(normalize_span_id(None), "")
        self.assertEqual(normalize_span_id(""), "")

    def test_verify_trace_spans_v1_decimal_ids(self):
        """Mocks realistic Cloud Trace v1 JSON with decimal span IDs and verifies successful validation."""
        trace_id = "4bf92f3577b34da6a3ce929d0e0e4736"
        client_span_id = "0010000000000001"  # hex: 0x0010000000000001 = 4503599627370497 in decimal
        espv2_span_decimal = "4503599627370498"  # hex: 0x0010000000000002
        backend_span_decimal = "4503599627370499"  # hex: 0x0010000000000003

        trace_json_v1 = {
            "projectId": "test-project",
            "traceId": trace_id,
            "spans": [
                {
                    "spanId": espv2_span_decimal,
                    "parentSpanId": "4503599627370497",  # decimal representation of client_span_id
                    "name": "ingress router-backend",
                    "labels": {
                        "/http/status_code": "200",
                        "http.method": "GET",
                    },
                },
                {
                    "spanId": backend_span_decimal,
                    "parentSpanId": espv2_span_decimal,
                    "name": "Bookstore.ListShelves",
                    "labels": {
                        "http.status_code": "200",
                    },
                },
            ],
        }

        self.assertTrue(
            verify_trace_spans(trace_json_v1, trace_id, client_span_id)
        )

    def test_verify_trace_spans_v1_labels_and_trace_id_field(self):
        """Verifies matching traceId / trace_id field and v1 labels structure."""
        trace_id = "4bf92f3577b34da6a3ce929d0e0e4736"
        client_span_id = "92cb2bb4f3c834a1"
        client_decimal = str(int(client_span_id, 16))
        espv2_decimal = "111111111111111111"
        backend_decimal = "222222222222222222"

        trace_json_v1 = {
            "trace_id": trace_id,
            "spans": [
                {
                    "span_id": espv2_decimal,
                    "parent_span_id": client_decimal,
                    "name": "ESPv2 Span",
                    "labels": {
                        "status_code": "200",
                    },
                },
                {
                    "span_id": backend_decimal,
                    "parent_span_id": espv2_decimal,
                    "name": "Backend Span",
                    "labels": {
                        "/http/status": "200",
                    },
                },
            ],
        }

        self.assertTrue(
            verify_trace_spans(trace_json_v1, trace_id, client_span_id)
        )

    def test_verify_trace_spans_success(self):
        """Mocks realistic Cloud Trace v2 JSON (ESPv2 span + Bookstore child span)

        and verifies successful validation.
        """
        trace_id = '4bf92f3577b34da6a3ce929d0e0e4736'
        client_span_id = '0010000000000001'
        espv2_span_id = '0020000000000001'
        backend_span_id = '0030000000000001'

        trace_json = {
            'name': f'projects/test-project/traces/{trace_id}',
            'spans': [
                {
                    'name': (
                        f'projects/test-project/traces/{trace_id}/spans/'
                        f'{espv2_span_id}'
                    ),
                    'spanId': espv2_span_id,
                    'parentSpanId': client_span_id,
                    'displayName': {'value': 'ingress router-backend'},
                    'attributes': {
                        'attributeMap': {
                            '/http/status_code': {'intValue': '200'},
                            'http.method': {'stringValue': {'value': 'GET'}},
                        }
                    },
                },
                {
                    'name': (
                        f'projects/test-project/traces/{trace_id}/spans/'
                        f'{backend_span_id}'
                    ),
                    'spanId': backend_span_id,
                    'parentSpanId': espv2_span_id,
                    'displayName': {'value': 'Bookstore.ListShelves'},
                    'attributes': {
                        'attributeMap': {
                            'http.status_code': {'intValue': '200'},
                        }
                    },
                },
            ],
        }

        self.assertTrue(
            verify_trace_spans(trace_json, trace_id, client_span_id)
        )

    def test_verify_trace_spans_mismatch(self):
        """Asserts failure when trace ID, parentSpanId, or status code is incorrect."""
        trace_id = '4bf92f3577b34da6a3ce929d0e0e4736'
        client_span_id = '0010000000000001'
        espv2_span_id = '0020000000000001'
        backend_span_id = '0030000000000001'

        valid_trace = {
            'name': f'projects/test-project/traces/{trace_id}',
            'spans': [
                {
                    'spanId': espv2_span_id,
                    'parentSpanId': client_span_id,
                    'attributes': {
                        'attributeMap': {'/http/status_code': {'intValue': '200'}}
                    },
                },
                {
                    'spanId': backend_span_id,
                    'parentSpanId': espv2_span_id,
                    'attributes': {
                        'attributeMap': {'http.status_code': {'intValue': '200'}}
                    },
                },
            ],
        }

        # 1. Trace ID mismatch
        wrong_trace_id = '11111111111111111111111111111111'
        self.assertFalse(
            verify_trace_spans(valid_trace, wrong_trace_id, client_span_id)
        )

        # 2. ESPv2 parentSpanId mismatch
        wrong_client_span = 'ffffffffffffffff'
        self.assertFalse(
            verify_trace_spans(valid_trace, trace_id, wrong_client_span)
        )

        # 3. Backend parentSpanId mismatch (does not connect to ESPv2 span)
        broken_backend_trace = {
            'name': f'projects/test-project/traces/{trace_id}',
            'spans': [
                {
                    'spanId': espv2_span_id,
                    'parentSpanId': client_span_id,
                    'attributes': {
                        'attributeMap': {'/http/status_code': {'intValue': '200'}}
                    },
                },
                {
                    'spanId': backend_span_id,
                    'parentSpanId': '9999999999999999',  # Unrelated parent
                    'attributes': {
                        'attributeMap': {'http.status_code': {'intValue': '200'}}
                    },
                },
            ],
        }
        self.assertFalse(
            verify_trace_spans(broken_backend_trace, trace_id, client_span_id)
        )

        # 4. Missing HTTP status code attribute on ESPv2 span
        no_status_espv2 = {
            'name': f'projects/test-project/traces/{trace_id}',
            'spans': [
                {
                    'spanId': espv2_span_id,
                    'parentSpanId': client_span_id,
                    'attributes': {},
                },
                {
                    'spanId': backend_span_id,
                    'parentSpanId': espv2_span_id,
                    'attributes': {
                        'attributeMap': {'http.status_code': {'intValue': '200'}}
                    },
                },
            ],
        }
        self.assertFalse(
            verify_trace_spans(no_status_espv2, trace_id, client_span_id)
        )

        # 5. Non-matching HTTP status code (e.g. 500 instead of 200)
        error_status_trace = {
            'name': f'projects/test-project/traces/{trace_id}',
            'spans': [
                {
                    'spanId': espv2_span_id,
                    'parentSpanId': client_span_id,
                    'attributes': {
                        'attributeMap': {'/http/status_code': {'intValue': '500'}}
                    },
                },
                {
                    'spanId': backend_span_id,
                    'parentSpanId': espv2_span_id,
                    'attributes': {
                        'attributeMap': {'http.status_code': {'intValue': '200'}}
                    },
                },
            ],
        }
        self.assertFalse(
            verify_trace_spans(
                error_status_trace, trace_id, client_span_id, expected_status=200
            )
        )

        # 6. Malformed JSON / empty spans
        self.assertFalse(
            verify_trace_spans({}, trace_id, client_span_id)
        )
        self.assertFalse(
            verify_trace_spans(
                {'name': f'projects/test/traces/{trace_id}', 'spans': []},
                trace_id,
                client_span_id,
            )
        )

    @mock.patch('urllib.request.urlopen')
    def test_send_traced_request(self, mock_urlopen):
        """Verifies sending HTTP request with W3C traceparent and API key."""
        mock_resp = mock.MagicMock()
        mock_resp.status = 200
        mock_resp.read.return_value = b'{"shelves": []}'
        mock_resp.__enter__.return_value = mock_resp
        mock_urlopen.return_value = mock_resp

        status, body = send_traced_request(
            host='http://localhost:8082',
            traceparent_header='00-4bf92f3577b34da6a3ce929d0e0e4736-0010000000000001-01',
            path='/shelves',
            api_key='test-api-key',
            auth_token='test-jwt',
        )

        self.assertEqual(status, 200)
        self.assertEqual(body, '{"shelves": []}')

        args, _ = mock_urlopen.call_args
        req = args[0]
        self.assertEqual(req.full_url, 'http://localhost:8082/shelves?key=test-api-key')
        self.assertEqual(
            req.headers.get('Traceparent'),
            '00-4bf92f3577b34da6a3ce929d0e0e4736-0010000000000001-01',
        )
        self.assertEqual(req.headers.get('Authorization'), 'Bearer test-jwt')

    @mock.patch('urllib.request.urlopen')
    def test_send_traced_request_with_host_header(self, mock_urlopen):
        """Verifies custom Host header is set on the request when host_header is provided."""
        mock_resp = mock.MagicMock()
        mock_resp.status = 200
        mock_resp.read.return_value = b'{"shelves": []}'
        mock_resp.__enter__.return_value = mock_resp
        mock_urlopen.return_value = mock_resp

        status, body = send_traced_request(
            host='http://localhost:8082',
            traceparent_header='00-4bf92f3577b34da6a3ce929d0e0e4736-0010000000000001-01',
            path='/shelves',
            host_header='custom.example.com',
        )

        self.assertEqual(status, 200)
        args, _ = mock_urlopen.call_args
        req = args[0]
        self.assertEqual(req.headers.get('Host'), 'custom.example.com')

    @mock.patch('apiproxy_trace_test.send_traced_request')
    def test_verify_trace_propagation_via_version_endpoint_success(self, mock_send):
        """Verifies successful in-band trace context verification via /version."""
        mock_send.return_value = (
            200,
            json.dumps({
                'host': 'localhost:8082',
                'traceparent': '00-4bf92f3577b34da6a3ce929d0e0e4736-0020000000000002-01',
            }),
        )

        result = verify_trace_propagation_via_version_endpoint(
            host='http://localhost:8082',
            trace_id='4bf92f3577b34da6a3ce929d0e0e4736',
            client_span_id='0010000000000001',
            host_header='custom.example.com',
        )
        self.assertTrue(result)
        mock_send.assert_called_once_with(
            host='http://localhost:8082',
            traceparent_header='00-4bf92f3577b34da6a3ce929d0e0e4736-0010000000000001-01',
            path='/version',
            method='GET',
            host_header='custom.example.com',
            timeout=15,
            verbose=False,
        )

    @mock.patch('apiproxy_trace_test.send_traced_request')
    def test_verify_trace_propagation_via_version_endpoint_http_error(self, mock_send):
        """Verifies failure when /version returns a non-200 HTTP status code."""
        mock_send.return_value = (500, 'Internal Server Error')
        result = verify_trace_propagation_via_version_endpoint(
            host='http://localhost:8082',
            trace_id='4bf92f3577b34da6a3ce929d0e0e4736',
            client_span_id='0010000000000001',
        )
        self.assertFalse(result)

    @mock.patch('apiproxy_trace_test.send_traced_request')
    def test_verify_trace_propagation_via_version_endpoint_missing_header(self, mock_send):
        """Verifies failure when /version response contains no traceparent header."""
        mock_send.return_value = (200, json.dumps({'host': 'localhost:8082'}))
        result = verify_trace_propagation_via_version_endpoint(
            host='http://localhost:8082',
            trace_id='4bf92f3577b34da6a3ce929d0e0e4736',
            client_span_id='0010000000000001',
        )
        self.assertFalse(result)

    @mock.patch('apiproxy_trace_test.send_traced_request')
    def test_verify_trace_propagation_via_version_endpoint_trace_id_mismatch(self, mock_send):
        """Verifies failure when echoed trace ID does not match expected trace ID."""
        mock_send.return_value = (
            200,
            json.dumps({
                'traceparent': '00-ffffffffffffffffffffffffffffffff-0020000000000002-01'
            }),
        )
        result = verify_trace_propagation_via_version_endpoint(
            host='http://localhost:8082',
            trace_id='4bf92f3577b34da6a3ce929d0e0e4736',
            client_span_id='0010000000000001',
        )
        self.assertFalse(result)

    @mock.patch('apiproxy_trace_test.send_traced_request')
    def test_verify_trace_propagation_via_version_endpoint_span_not_mutated(self, mock_send):
        """Verifies failure when echoed span ID is identical to the client span ID."""
        mock_send.return_value = (
            200,
            json.dumps({
                'traceparent': '00-4bf92f3577b34da6a3ce929d0e0e4736-0010000000000001-01'
            }),
        )
        result = verify_trace_propagation_via_version_endpoint(
            host='http://localhost:8082',
            trace_id='4bf92f3577b34da6a3ce929d0e0e4736',
            client_span_id='0010000000000001',
        )
        self.assertFalse(result)

    @mock.patch('apiproxy_trace_test.send_traced_request')
    def test_verify_trace_propagation_via_version_endpoint_invalid_json(self, mock_send):
        """Verifies failure when /version response body cannot be parsed as JSON."""
        mock_send.return_value = (200, 'Not JSON')
        result = verify_trace_propagation_via_version_endpoint(
            host='http://localhost:8082',
            trace_id='4bf92f3577b34da6a3ce929d0e0e4736',
            client_span_id='0010000000000001',
        )
        self.assertFalse(result)

    @mock.patch('apiproxy_trace_test.verify_trace_spans')
    @mock.patch('apiproxy_trace_test.poll_cloud_trace')
    @mock.patch('apiproxy_trace_test.get_gcp_access_token')
    @mock.patch('apiproxy_trace_test.send_traced_request')
    @mock.patch('apiproxy_trace_test.verify_trace_propagation_via_version_endpoint')
    def test_run_trace_e2e_test_success(
        self, mock_version, mock_send, mock_token, mock_poll, mock_verify
    ):
        """Verifies end-to-end trace assertion execution flow with dual verification."""
        mock_version.return_value = True
        mock_send.return_value = (200, '{"shelves": []}')
        mock_token.return_value = 'mock-bearer-token'
        mock_poll.return_value = {'spans': [{'spanId': '0020000000000001'}]}
        mock_verify.return_value = True

        success = run_trace_e2e_test(
            host='http://localhost:8082',
            project_id='test-project',
            api_key='key123',
            host_header='custom.example.com',
            timeout_sec=10,
            delay_sec=0,
        )

        self.assertTrue(success)
        mock_version.assert_called_once()
        mock_send.assert_called_once_with(
            host='http://localhost:8082',
            traceparent_header=mock.ANY,
            path='/shelves',
            api_key='key123',
            auth_token=None,
            host_header='custom.example.com',
            verbose=False,
        )
        mock_token.assert_called_once()
        mock_poll.assert_called_once()
        mock_verify.assert_called_once()

    @mock.patch('apiproxy_trace_test.poll_cloud_trace')
    @mock.patch('apiproxy_trace_test.verify_trace_propagation_via_version_endpoint')
    def test_run_trace_e2e_test_version_propagation_failure(
        self, mock_version, mock_poll
    ):
        """Verifies that in-band /version verification failure blocks execution before Cloud Trace."""
        mock_version.return_value = False

        success = run_trace_e2e_test(
            host='http://localhost:8082',
            project_id='test-project',
            timeout_sec=10,
            delay_sec=0,
        )

        self.assertFalse(success)
        mock_version.assert_called_once()
        mock_poll.assert_not_called()

    def test_make_argparser(self):
        """Verifies CLI argument parser options and defaults."""
        parser = make_argparser()
        args = parser.parse_args([
            '--host', 'http://localhost:8082',
            '--project', 'test-project',
            '--api_key', 'my-key',
            '--auth_token', 'my-token',
            '--host_header', 'custom.example.com',
            '--timeout', '45',
            '--delay', '3',
            '--verbose',
        ])
        self.assertEqual(args.host, 'http://localhost:8082')
        self.assertEqual(args.project, 'test-project')
        self.assertEqual(args.api_key, 'my-key')
        self.assertEqual(args.auth_token, 'my-token')
        self.assertEqual(args.host_header, 'custom.example.com')
        self.assertEqual(args.timeout, 45)
        self.assertEqual(args.delay, 3)
        self.assertTrue(args.verbose)

        # Check default value for host_header
        default_args = parser.parse_args([])
        self.assertIsNone(default_args.host_header)


if __name__ == '__main__':
    unittest.main()
