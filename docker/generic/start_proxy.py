# Copyright 2019 Google LLC

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

import argparse
import logging
import os
import re
import signal
import subprocess
import sys
import threading
import time

# The command to generate Envoy bootstrap config
BOOTSTRAP_CMD = "bin/bootstrap"

# Location of Config Manager and Envoy binary
CONFIGMANAGER_BIN = "bin/configmanager"
ENVOY_BIN = "bin/envoy"

# Health check period in secs, for Config Manager and Envoy.
HEALTH_CHECK_PERIOD = 60

# bootstrap config file will write here.
# By default, envoy writes some logs to /tmp too
# If root file system is read-only, this folder should be
# mounted from tmpfs.
DEFAULT_CONFIG_DIR = "/tmp"

# bootstrap config file name.
BOOTSTRAP_CONFIG = "/bootstrap.json"

# Default Listener port
DEFAULT_LISTENER_PORT = 8080

# Default backend
DEFAULT_BACKEND = "http://127.0.0.1:8082"

# Default rollout_strategy
DEFAULT_ROLLOUT_STRATEGY = "fixed"

# Google default application credentials environment variable
GOOGLE_CREDS_KEY = "GOOGLE_APPLICATION_CREDENTIALS"

# Flag defaults when running on serverless.
SERVERLESS_PLATFORM = "Cloud Run(ESPv2)"
SERVERLESS_XFF_NUM_TRUSTED_HOPS = 0

def gen_bootstrap_conf(args):
    cmd = [BOOTSTRAP_CMD, "--logtostderr"]

    cmd.extend(["--admin_port", str(args.status_port)])
    if args.http_request_timeout_s:
        cmd.extend(
            ["--http_request_timeout_s",
             str(args.http_request_timeout_s)])


    bootstrap_file = DEFAULT_CONFIG_DIR + BOOTSTRAP_CONFIG
    cmd.append(bootstrap_file)
    print(cmd)
    return cmd


class ArgumentParser(argparse.ArgumentParser):
    def error(self, message):
        self.print_help(sys.stderr)
        self.exit(1, '%s: error: %s\n' % (self.prog, message))


# Notes: These flags should get aligned with that of ESP at
# https://github.com/cloudendpoints/esp/blob/master/start_esp/start_esp.py#L420
def make_argparser():
    parser = ArgumentParser(
        formatter_class=argparse.RawDescriptionHelpFormatter,
        description='''
ESPv2 start-up script. This script starts Config Manager and Envoy.

The service name and config ID are optional. If not supplied, the Config Manager
fetches the service name and the config ID from the metadata service as
attributes "service_name" and "service_config_id".

ESPv2 relies on the metadata service to fetch access tokens for Google
services. If you deploy ESPv2 outside of Google Cloud environment, you need
to provide a service account credentials file by setting "creds_key"
environment variable or by passing "-k" flag to this script.
            ''')

    parser.add_argument(
        '-s',
        '--service',
        default="",
        help=''' Set the name of the Endpoints service.  If omitted and -c not
        specified, ESPv2 contacts the metadata service to fetch the service
        name.  ''')

    parser.add_argument(
        '-v',
        '--version',
        default="",
        help=''' Set the service config ID of the Endpoints service.
        If omitted and -c not specified, ESPv2 contacts the metadata
        service to fetch the service config ID.  ''')

    parser.add_argument(
        '--service_json_path',
        default=None,
        help='''
        Specify a path for ESPv2 to load the endpoint service config.
        With this flag, ESPv2 will use "fixed" rollout strategy and following
        flags will be ignored:
           --service, --version, and --rollout_strategy.
        ''')

    parser.add_argument(
        '-a',
        '--backend',
        default=DEFAULT_BACKEND,
        help='''
        Specify the local backend application server address
        when using ESPv2 as a sidecar.

        Default value is {backend}. Follow the same format when setting
        manually. Valid schemes are `http`, `https`, `grpc`, and `grpcs`.
        '''.format(backend=DEFAULT_BACKEND))

    parser.add_argument('--listener_port', default=None, type=int, help='''
        The port to accept downstream connections.
        It supports HTTP/1.x, HTTP/2, and gRPC connections.
        Default is {port}'''.format(port=DEFAULT_LISTENER_PORT))

    parser.add_argument('-N', '--status_port', '--admin_port', default=0,
        type=int, help=''' Enable ESPv2 Envoy admin on this port. Please refer
        to https://www.envoyproxy.io/docs/envoy/latest/operations/admin.
        By default the admin port is disabled.''')

    parser.add_argument('--ssl_server_cert_path', default=None, help='''
        Proxy's server cert path. When configured, ESPv2 only accepts HTTP/1.x and
        HTTP/2 secure connections on listener_port. Requires the certificate and
        key files "server.crt" and "server.key" within this path.''')

    parser.add_argument('--ssl_backend_client_cert_path', default=None, help='''
        Proxy's client cert path. When configured, ESPv2 enables TLS mutual
        authentication for HTTPS backends. Requires the certificate and
        key files "client.crt" and "client.key" within this path.''')

    parser.add_argument('--ssl_backend_client_root_certs_file', default=None, help='''
        The file path of root certificates that ESPv2 uses to verify backend server certificate.
        If not specified, ESPv2 uses '/etc/ssl/certs/ca-certificates.crt' by default.''')

    parser.add_argument('--ssl_minimum_protocol', default=None,
        choices=['TLSv1.0', 'TLSv1.1', 'TLSv1.2', 'TLSv1.3'],
        help=''' Minimum TLS protocol version for client side connection.
        Please refer to https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/auth/cert.proto#common-tls-configuration.
        ''')

    parser.add_argument('--ssl_maximum_protocol', default=None,
        choices=['TLSv1.0', 'TLSv1.1', 'TLSv1.2', 'TLSv1.3'],
        help=''' Maximum TLS protocol version for client side connection.
        Please refer to https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/auth/cert.proto#common-tls-configuration.
        ''')

    parser.add_argument('--enable_strict_transport_security', action='store_true',
        help='''Enable HSTS (HTTP Strict Transport Security). "Strict-Transport-Security" response header
        with value "max-age=31536000; includeSubdomains;" is added for all responses from local backend.
        Not valid for remote backends.''')

    parser.add_argument('--generate_self_signed_cert', action='store_true',
        help='''Generate a self-signed certificate and key at start, then
        store them in /tmp/ssl/endpoints/server.crt and /tmp/ssl/endponts/server.key.
        This is useful when only a random self-sign cert is needed to serve
        HTTPS requests. Generated certificate will have Common Name
        "localhost" and valid for 10 years.
        ''')

    parser.add_argument('-z', '--healthz', default=None, help='''Define a
        health checking endpoint on the same ports as the application backend.
        For example, "-z healthz" makes ESPv2 return code 200 for location
        "/healthz", instead of forwarding the request to the backend. Please
        don't use any paths conflicting with your normal requests.
        Default: not used.''')

    parser.add_argument(
        '-R',
        '--rollout_strategy',
        default=DEFAULT_ROLLOUT_STRATEGY,
        help='''The service config rollout strategy, [fixed|managed],
        Default value: {strategy}'''.format(strategy=DEFAULT_ROLLOUT_STRATEGY),
        choices=['fixed', 'managed'])

    # Customize management service url prefix.
    parser.add_argument(
        '-g',
        '--management',
        default=None,
        help=argparse.SUPPRESS)

    # CORS presets
    parser.add_argument(
        '--cors_preset',
        default=None,
        help='''
        Enables setting of CORS headers. This is useful when using a GRPC
        backend, since a GRPC backend cannot set CORS headers.
        Specify one of available presets to configure CORS response headers
        in nginx. Defaults to no preset and therefore no CORS response
        headers. If no preset is suitable for the use case, use the
        --nginx_config arg to use a custom nginx config file.
        Available presets:
        - basic - Assumes all location paths have the same CORS policy.
          Responds to preflight OPTIONS requests with an empty 204, and the
          results of preflight are allowed to be cached for up to 20 days
          (1728000 seconds). See descriptions for args --cors_allow_origin,
          --cors_allow_methods, --cors_allow_headers, --cors_expose_headers,
          --cors_allow_credentials for more granular configurations.
        - cors_with_regex - Same as basic preset, except that specifying
          allowed origins in regular expression. See descriptions for args
          --cors_allow_origin_regex, --cors_allow_methods,
          --cors_allow_headers, --cors_expose_headers, --cors_allow_credentials
          for more granular configurations.
        ''')
    parser.add_argument(
        '--cors_allow_origin',
        default='*',
        help='''
        Only works when --cors_preset is 'basic'. Configures the CORS header
        Access-Control-Allow-Origin. Defaults to "*" which allows all origins.
        ''')
    parser.add_argument(
        '--cors_allow_origin_regex',
        default='',
        help='''
        Only works when --cors_preset is 'cors_with_regex'. Configures the
        whitelists of CORS header Access-Control-Allow-Origin with regular
        expression.
        ''')
    parser.add_argument(
        '--cors_allow_methods',
        default='GET, POST, PUT, PATCH, DELETE, OPTIONS',
        help='''
        Only works when --cors_preset is in use. Configures the CORS header
        Access-Control-Allow-Methods. Defaults to allow common HTTP
        methods.
        ''')
    parser.add_argument(
        '--cors_allow_headers',
        default=
        'DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization',
        help='''
        Only works when --cors_preset is in use. Configures the CORS header
        Access-Control-Allow-Headers. Defaults to allow common HTTP
        headers.
        ''')
    parser.add_argument(
        '--cors_expose_headers',
        default='Content-Length,Content-Range',
        help='''
        Only works when --cors_preset is in use. Configures the CORS header
        Access-Control-Expose-Headers. Defaults to allow common response headers.
        ''')
    parser.add_argument(
        '--cors_allow_credentials',
        action='store_true',
        help='''
        Only works when --cors_preset is in use. Enable the CORS header
        Access-Control-Allow-Credentials. By default, this header is disabled.
        ''')
    parser.add_argument(
        '--check_metadata',
        action='store_true',
        help='''Enable fetching service name, service config ID and rollout
        strategy from the metadata service.''')

    parser.add_argument('--underscores_in_headers', action='store_true',
        help='''Allow headers contain underscores to pass through. By default
        ESPv2 rejects requests that have headers with underscores.''')

    parser.add_argument(
        '--envoy_use_remote_address',
        action='store_true',
        default=False,
        help='''Envoy HttpConnectionManager configuration, please refer to envoy
        documentation for detailed information.''')

    parser.add_argument(
        '--envoy_xff_num_trusted_hops',
        default=None,
        help='''Envoy HttpConnectionManager configuration, please refer to envoy
        documentation for detailed information. The default value is 2 for
        sidecar deployments and 0 for serverless deployments.''')

    parser.add_argument(
        '--envoy_connection_buffer_limit_bytes', action=None,
        help='''
        Configure the maximum amount of data that is buffered for each 
        request/response body, in bytes. If not set, default is decided by
        Envoy.
        
        https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/listener/v3/listener.proto
        ''')

    parser.add_argument(
        '--log_request_headers',
        default=None,
        help='''Log corresponding request headers through
        service control, separated by comma. Example, when
        --log_request_headers=foo,bar, endpoint log will have
        request_headers: foo=foo_value;bar=bar_value if values are available;
        ''')

    parser.add_argument(
        '--log_response_headers',
        default=None,
        help='''Log corresponding response headers through
        service control, separated by comma. Example, when
        --log_response_headers=foo,bar, endpoint log will have
        response_headers: foo=foo_value;bar=bar_value if values are available;
        ''')

    parser.add_argument(
        '--log_jwt_payloads',
        default=None,
        help='''
        Log corresponding JWT JSON payload primitive fields through service control,
        separated by comma. Example, when --log_jwt_payload=sub,project_id, log
        will have jwt_payload: sub=[SUBJECT];project_id=[PROJECT_ID]
        if the fields are available. The value must be a primitive field,
        JSON objects and arrays will not be logged.
        ''')
    parser.add_argument(
        '--service_control_network_fail_open',
        default=True,
        action='store_true',
        help='''
        In case of network failures when connecting to Google service control,
        the requests will be allowed if this flag is on. The default is on.
        ''')
    parser.add_argument(
        '--jwks_cache_duration_in_s',
        default=None,
        help='''
        Specify JWT public key cache duration in seconds. The default is 5 minutes.'''
    )
    parser.add_argument(
        '--http_request_timeout_s',
        default=None, type=int,
        help='''
        Set the timeout in seconds for all requests made to all external services
        from ESPv2 (ie. Service Management, Instance Metadata Server, etc.).
        This timeout does not apply to requests proxied to the backend.
        Must be > 0 and the default is 30 seconds if not set.
        ''')
    parser.add_argument(
        '--service_control_check_timeout_ms',
        default=None,
        help='''
        Set the timeout in millisecond for service control Check request.
        Must be > 0 and the default is 1000 if not set. Default
        ''')
    parser.add_argument(
        '--service_control_quota_timeout_ms',
        default=None,
        help='''
        Set the timeout in millisecond for service control Quota request.
        Must be > 0 and the default is 1000 if not set.
        ''')
    parser.add_argument(
        '--service_control_report_timeout_ms',
        default=None,
        help='''
        Set the timeout in millisecond for service control Report request.
        Must be > 0 and the default is 2000 if not set.
        ''')
    parser.add_argument(
        '--service_control_check_retries',
        default=None,
        help='''
        Set the retry times for service control Check request.
        Must be >= 0 and the default is 3 if not set.
        ''')
    parser.add_argument(
        '--service_control_quota_retries',
        default=None,
        help='''
        Set the retry times for service control Quota request.
        Must be >= 0 and the default is 1 if not set.
        ''')
    parser.add_argument(
        '--service_control_report_retries',
        default=None,
        help='''
        Set the retry times for service control Report request.
        Must be >= 0 and the default is 5 if not set.
        ''')
    parser.add_argument(
        '--access_log',
        help='''
        Path to a local file to which the access log entries will be written.
        '''
    )
    parser.add_argument(
        '--access_log_format',
        help='''
        String format to specify the format of access log. If unset, the
        following format will be used.
        https://www.envoyproxy.io/docs/envoy/latest/configuration/observability/access_log#default-format-string
        For the detailed format grammar, please refer to the following document.
        https://www.envoyproxy.io/docs/envoy/latest/configuration/observability/access_log#format-strings
        '''
    )

    parser.add_argument(
        '--disable_tracing',
        action='store_true',
        default=False,
        help='''
        Disable Stackdriver tracing. By default, tracing is enabled with 1 out
        of 1000 requests being sampled. This sampling rate can be changed with
        the --tracing_sample_rate flag.
        '''
    )
    parser.add_argument(
        '--tracing_project_id',
        default="",
        help="The Google project id for Stack driver tracing")
    parser.add_argument(
        '--tracing_sample_rate',
        help='''
        Tracing sampling rate from 0.0 to 1.0.
        By default, 1 out of 1000 requests are sampled.
        Cloud trace can still be enabled from request HTTP headers with
        trace context regardless this flag value.
        '''
    )
    parser.add_argument(
        '--disable_cloud_trace_auto_sampling',
        action='store_true',
        default=False,
        help="An alias to override --tracing_sample_rate to 0")
    parser.add_argument(
        '--tracing_incoming_context',
        default="",
        help='''
        comma separated incoming trace contexts (traceparent|grpc-trace-bin|x-cloud-trace-context)'''
    )
    parser.add_argument(
        '--tracing_outgoing_context',
        default="",
        help='''
        comma separated outgoing trace contexts (traceparent|grpc-trace-bin|x-cloud-trace-context)'''
    )
    parser.add_argument(
        '--cloud_trace_url_override',
        default="",
        help='''
        By default, traces will be sent to production Stackdriver Tracing.
        If this is non-empty, ESPv2 will send traces to this gRPC service instead.
        
        The url must be in gRPC format.
        https://github.com/grpc/grpc/blob/master/doc/naming.md
        
        The gRPC service must implement the cloud trace v2 RPCs.
        https://github.com/googleapis/googleapis/tree/master/google/devtools/cloudtrace/v2
        '''
    )
    parser.add_argument(
        '--non_gcp',
        action='store_true',
        default=False,
        help='''
        By default, the proxy tries to talk to GCP metadata server to get VM
        location in the first few requests. Setting this flag to true to skip
        this step.
        ''')
    parser.add_argument(
        '--service_account_key',
        help='''
        Use the service account key JSON file to access the service control and the
        service management.  You can also set {creds_key} environment variable to
        the location of the service account credentials JSON file. If the option is
        omitted, the proxy contacts the metadata service to fetch an access token.
        '''.format(creds_key=GOOGLE_CREDS_KEY))

    parser.add_argument(
        '--dns_resolver_addresses',
        help='''
        The addresses of dns resolvers. Each address should be in format of
        IP_ADDR or IP_ADDR:PORT and they are separated by ';'. For the IP_ADDR
        case, the default DNS port 52 will be used. (e.g.,
        --dns_resolver_addresses=127.0.0.1;127.0.0.2;127.0.0.3:8000)

        If unset, will use the default resolver configured in /etc/resolv.conf.
        ''')

    parser.add_argument(
        '--backend_dns_lookup_family',
        default=None,
        choices=['auto', 'v4only', 'v6only'],
        help='''
        Define the dns lookup family for all backends. The options are "auto", "v4only" and "v6only". The default is "auto".
        ''')
    parser.add_argument('--enable_debug', action='store_true', default=False,
        help='''
        Enables a variety of debug features in both Config Manager and Envoy, such as:
        - Debug level per-request application logs in Envoy
        - Debug level service configuration logs in Config Manager
        - Debug HTTP response headers
        ''')

    parser.add_argument(
        '--transcoding_always_print_primitive_fields',
        action='store_true', help='''Whether to always print primitive fields
        for grpc-json transcoding. By default primitive fields with default
        values will be omitted in JSON output. For example, an int32 field set
        to 0 will be omitted. Setting this flag to true will override the
        default behavior and print primitive fields regardless of their values.
        Defaults to false
        ''')

    parser.add_argument(
        '--transcoding_always_print_enums_as_ints', action='store_true',
        help='''Whether to always print enums as ints for grpc-json transcoding.
        By default they are rendered as strings. Defaults to false.''')

    parser.add_argument(
        '--transcoding_preserve_proto_field_names', action='store_true',
        help='''Whether to preserve proto field names for grpc-json transcoding.
        By default protobuf will generate JSON field names using the json_name
        option, or lower camel case, in that order. Setting this flag will
        preserve the original field names. Defaults to false''')

    parser.add_argument(
        '--transcoding_ignore_query_parameters', action=None,
        help='''
         A list of query parameters(separated by comma) to be ignored for
         transcoding method mapping in grpc-json transcoding. By default, the
         transcoder filter will not transcode a request if there are any
         unknown/invalid query parameters.
         ''')

    parser.add_argument(
        '--transcoding_ignore_unknown_query_parameters', action='store_true',
        help='''
        Whether to ignore query parameters that cannot be mapped to a
        corresponding protobuf field in grpc-json transcoding. Use this if you
        cannot control the query parameters and do not know them beforehand.
        Otherwise use ignored_query_parameters. Defaults to false.
        ''')

    # Start Deprecated Flags Section

    parser.add_argument(
        '--enable_backend_routing',
        action='store_true',
        default=False,
        help='''
        ===
        DEPRECATED: This flag will automatically be enabled if needed, so it
        does NOT need to be set manually.
        ===
        Enable ESPv2 to route requests according to the
        "x-google-backend" or "backend" configuration
        ''')
    parser.add_argument(
        '--backend_protocol',
        default=None,
        help='''
        ===
        DEPRECATED: This flag will automatically be set based on the scheme
        specified in the --backend flag. Overrides are no longer needed.
        ===
        Backend Protocol. Overrides the protocol in --backend.
        Choices: [http1|http2|grpc].
        Default value: http1.''',
        choices=['http1', 'http2', 'grpc'])

    parser.add_argument('--http_port', default=None, type=int, help='''
        This flag is exactly same as --listener_port. It is added for
        backward compatible for ESPv1 and will be deprecated.
        Please use the flag --listener_port.''')

    parser.add_argument('--http2_port', default=None, type=int, help='''
        This flag is exactly same as --listener_port. It is added for
        backward compatible for ESPv1 and will be deprecated.
        Please use the flag --listener_port.''')

    parser.add_argument('--ssl_port', default=None, type=int, help='''
        This flag added for backward compatible for ESPv1 and will be deprecated.
        Please use the flags --listener_port and --ssl_server_cert_path instead. 
        When configured, ESPv2 accepts HTTP/1.x and HTTP/2 secure connections on this port,
        Requires the certificate and key files /etc/nginx/ssl/nginx.crt and
        /etc/nginx/ssl/nginx.key''')

    parser.add_argument('--dns',  help='''
        This flag is exactly same as --dns_resolver_addresses. This flag is added
        for backward compatible for ESPv1 and will be deprecated.
        Please use the flag --dns_resolver_addresses instead.''')

    parser.add_argument('-t', '--tls_mutual_auth', action='store_true', help='''
        This flag added for backward compatible for ESPv1 and will be deprecated.
        Please use the flag --ssl_backend_client_cert_path instead.
        Enable TLS mutual authentication for HTTPS backends.
        Default value: Not enabled. Please provide the certificate and key files
        /etc/nginx/ssl/backend.crt and /etc/nginx/ssl/backend.key.''')

    parser.add_argument('--ssl_protocols',
        default=None, action='append', help='''
        This flag added for backward compatible for ESPv1 and will be deprecated.
        Please use the flag --ssl_minimum_protocol and  --ssl_maximum_protocol
        instead.
        Enable the specified SSL protocols. Please refer to
        https://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_protocols.
        The "ssl_protocols" argument can be repeated multiple times to specify multiple
        SSL protocols (e.g., --ssl_protocols=TLSv1.1 --ssl_protocols=TLSv1.2).
        ''')

    parser.add_argument('--enable_grpc_backend_ssl',
        action='store_true', help='''
        This flag added for backward compatible for ESPv1 and will be deprecated.
        Enable SSL for gRPC backend. ESPv2 auto enables SSL if schema `grpcs` is
        detected.''')

    parser.add_argument('--grpc_backend_ssl_root_certs_file',
        default='/etc/nginx/trusted-ca-certificates.crt',
        help='''This flag added for backward compatible for ESPv1 and will be deprecated.
        ESPv2 uses `/etc/ssl/certs/ca-certificates.crt` by default.
        The file path for gRPC backend SSL root certificates.''')

    parser.add_argument('--ssl_client_cert_path', default=None, help='''
        This flag is renamed and deprecated for clarity. 
        Use `--ssl_backend_client_cert_path` instead.''')

    parser.add_argument('--ssl_client_root_certs_file', default=None, help='''
        This flag is renamed and deprecated for clarity. 
        Use `--ssl_backend_client_root_certs_file` instead.''')

    # End Deprecated Flags Section

    # Start internal flags section

    parser.add_argument(
        '--on_serverless',
        action='store_true',
        default=False,
        help='''
        When ESPv2 is started via the serverless image, this is true.
        ''')

    # End internal flags section

    return parser

# Check whether there are conflict flags. If so, return the error string.
# Otherwise returns None. This function also changes some default flag value.
def enforce_conflict_args(args):
    if args.rollout_strategy:
        if args.rollout_strategy != DEFAULT_ROLLOUT_STRATEGY:
          if args.version:
            return "Flag --version cannot be used together with -R or --rollout_strategy."
          if args.service_json_path:
            return "Flag -R or --rollout_strategy must be fixed with --service_json_path."

    if args.service_json_path:
        if args.service:
            return "Flag --service cannot be used together with --service_json_path."
        if args.version:
            return "Flag --version cannot be used together with --service_json_path."

    if args.non_gcp:
        if args.service_account_key is None and GOOGLE_CREDS_KEY not in os.environ:
            return "If --non_gcp is specified, --service_account_key has to be specified, or GOOGLE_APPLICATION_CREDENTIALS has to set in os.environ."
        if not args.tracing_project_id:
            # for non gcp case, disable tracing if tracing project id is not provided.
            args.disable_tracing = True

    if not args.access_log and args.access_log_format:
        return "Flag --access_log_format has to be used together with --access_log."

    if args.ssl_port and args.ssl_server_cert_path:
        return "Flag --ssl_port is going to be deprecated, please use --ssl_server_cert_path only."
    if args.tls_mutual_auth and (args.ssl_backend_client_cert_path or args.ssl_client_cert_path):
        return "Flag --tls_mutual_auth is going to be deprecated, please use --ssl_backend_client_cert_path only."
    if (args.ssl_backend_client_root_certs_file or args.ssl_client_root_certs_file) and args.enable_grpc_backend_ssl:
        return "Flag --enable_grpc_backend_ssl are going to be deprecated, please use --ssl_backend_client_root_certs_file only."
    if args.generate_self_signed_cert and args.ssl_server_cert_path:
         return "Flag --generate_self_signed_cert and --ssl_server_cert_path cannot be used simutaneously."

    port_flags = []
    port_num = DEFAULT_LISTENER_PORT
    if args.http_port:
        port_flags.append("--http_port")
        port_num = args.http_port
    if args.http2_port:
        port_flags.append("--http2_port")
        port_num = args.http2_port
    if args.listener_port:
        port_flags.append("--listener_port")
        port_num = args.listener_port
    if args.ssl_port:
        port_flags.append("--ssl_port")
        port_num = args.ssl_port

    if len(port_flags) > 1:
        return "Multiple port flags {} are not allowed, use only the --listener_port flag".format(",".join(port_flags))
    elif port_num < 1024:
        return "Port {} is a privileged port. " \
               "For security purposes, the ESPv2 container cannot bind to it. " \
               "Use any port above 1024 instead.".format(port_num)

    if args.ssl_protocols and (args.ssl_minimum_protocol or args.ssl_maximum_protocol):
        return "Flag --ssl_protocols is going to be deprecated, please use --ssl_minimum_protocol and --ssl_maximum_protocol."

    if args.transcoding_ignore_query_parameters \
        and args.transcoding_ignore_unknown_query_parameters:
        return "Flag --transcoding_ignore_query_parameters cannot be used" \
               " together with --transcoding_ignore_unknown_query_parameters."

    if args.dns_resolver_addresses and args.dns:
        return "Flag --dns_resolver_addresses cannot be used together with" \
               " together with --dns."

    if args.ssl_backend_client_cert_path and args.ssl_client_cert_path:
        return "Flag --ssl_client_cert_path is renamed to " \
               "--ssl_backend_client_cert_path, only use the latter flag."

    if args.ssl_backend_client_root_certs_file and args.ssl_client_root_certs_file:
        return "Flag --ssl_client_root_certs_file is renamed to " \
               "--ssl_backend_client_root_certs_file, only use the latter flag."

    return None

def gen_proxy_config(args):
    check_conflict_result = enforce_conflict_args(args)
    if check_conflict_result:
        logging.error(check_conflict_result)
        sys.exit(1)

    proxy_conf = [
        CONFIGMANAGER_BIN,
        "--logtostderr",
        "--rollout_strategy", args.rollout_strategy,
    ]

    if "://" not in args.backend:
      proxy_conf.extend(["--backend_address", "http://" + args.backend])
    else:
      proxy_conf.extend(["--backend_address", args.backend])

    if args.healthz:
      proxy_conf.extend(["--healthz", args.healthz])

    if args.enable_debug:
        proxy_conf.extend(["--v", "1"])
    else:
        proxy_conf.extend(["--v", "0"])

    if args.envoy_xff_num_trusted_hops:
        proxy_conf.extend(["--envoy_xff_num_trusted_hops",
                           args.envoy_xff_num_trusted_hops])
    elif args.on_serverless:
        proxy_conf.extend(["--envoy_xff_num_trusted_hops",
                           '{}'.format(SERVERLESS_XFF_NUM_TRUSTED_HOPS)])

    if args.jwks_cache_duration_in_s:
         proxy_conf.extend(["--jwks_cache_duration_in_s", args.jwks_cache_duration_in_s])

    if args.management:
        proxy_conf.extend(["--service_management_url", args.management])

    if args.log_request_headers:
        proxy_conf.extend(["--log_request_headers", args.log_request_headers])

    if args.log_response_headers:
        proxy_conf.extend(["--log_response_headers", args.log_response_headers])

    if args.log_jwt_payloads:
        proxy_conf.extend(["--log_jwt_payloads", args.log_jwt_payloads])

    if args.http_port:
        proxy_conf.extend(["--listener_port", str(args.http_port)])
    if args.http2_port:
        proxy_conf.extend(["--listener_port", str(args.http2_port)])
    if args.listener_port:
        proxy_conf.extend(["--listener_port", str(args.listener_port)])
    if args.ssl_server_cert_path:
        proxy_conf.extend(["--ssl_server_cert_path", str(args.ssl_server_cert_path)])
    if args.ssl_port:
        proxy_conf.extend(["--ssl_server_cert_path", "/etc/nginx/ssl"])
        proxy_conf.extend(["--listener_port", str(args.ssl_port)])

    if args.ssl_backend_client_cert_path:
        proxy_conf.extend(["--ssl_backend_client_cert_path", str(args.ssl_backend_client_cert_path)])
    if args.ssl_client_cert_path:
        proxy_conf.extend(["--ssl_backend_client_cert_path", str(args.ssl_client_cert_path)])

    if args.enable_grpc_backend_ssl and args.grpc_backend_ssl_root_certs_file:
        proxy_conf.extend(["--ssl_backend_client_root_certs_path", str(args.grpc_backend_ssl_root_certs_file)])

    if args.ssl_backend_client_root_certs_file:
        proxy_conf.extend(["--ssl_backend_client_root_certs_path", str(args.ssl_backend_client_root_certs_file)])
    if args.ssl_client_root_certs_file:
        proxy_conf.extend(["--ssl_backend_client_root_certs_path", str(args.ssl_client_root_certs_file)])

    if args.tls_mutual_auth:
        proxy_conf.extend(["--ssl_backend_client_cert_path", "/etc/nginx/ssl"])

    if args.ssl_minimum_protocol:
        proxy_conf.extend(["--ssl_minimum_protocol", args.ssl_minimum_protocol])
    if args.ssl_maximum_protocol:
        proxy_conf.extend(["--ssl_maximum_protocol", args.ssl_maximum_protocol])
    if args.ssl_protocols:
        args.ssl_protocols.sort()
        proxy_conf.extend(["--ssl_minimum_protocol", args.ssl_protocols[0]])
        proxy_conf.extend(["--ssl_maximum_protocol", args.ssl_protocols[-1]])

    # Generate self-signed cert if needed
    if args.generate_self_signed_cert:
        if not os.path.exists("/tmp/ssl/endpoints"):
            os.makedirs("/tmp/ssl/endpoints")
        logging.info("Generating self-signed certificate...")
        os.system(("openssl req -x509 -newkey rsa:2048"
                   " -keyout /tmp/ssl/endpoints/server.key -nodes"
                   " -out /tmp/ssl/endpoints/server.crt"
                   ' -days 3650 -subj "/CN=localhost"'))
        proxy_conf.extend(["--ssl_server_cert_path", "/tmp/ssl/endpoints"])

    if args.enable_strict_transport_security:
            proxy_conf.append("--enable_strict_transport_security")

    if args.service:
        proxy_conf.extend(["--service", args.service])

    if args.http_request_timeout_s:
        proxy_conf.extend( ["--http_request_timeout_s", str(args.http_request_timeout_s)])

    if args.service_control_check_retries:
        proxy_conf.extend([
            "--service_control_check_retries",
            args.service_control_check_retries
        ])

    if args.service_control_quota_retries:
        proxy_conf.extend([
            "--service_control_quota_retries",
            args.service_control_quota_retries
        ])

    if args.service_control_report_retries:
        proxy_conf.extend([
            "--service_control_report_retries",
            args.service_control_report_retries
        ])

    if args.service_control_check_timeout_ms:
        proxy_conf.extend([
            "--service_control_check_timeout_ms",
            args.service_control_check_timeout_ms
        ])

    if args.service_control_quota_timeout_ms:
        proxy_conf.extend([
            "--service_control_quota_timeout_ms",
            args.service_control_quota_timeout_ms
        ])

    if args.service_control_report_timeout_ms:
        proxy_conf.extend([
            "--service_control_report_timeout_ms",
            args.service_control_report_timeout_ms
        ])

    #  NOTE: It is true by default in configmangager's flags.
    if not args.service_control_network_fail_open:
        proxy_conf.extend(["--service_control_network_fail_open=false"])

    if args.version:
        proxy_conf.extend(["--service_config_id", args.version])

    if args.service_json_path:
        proxy_conf.extend(["--service_json_path", args.service_json_path])

    if args.check_metadata:
        proxy_conf.append("--check_metadata")

    if args.underscores_in_headers:
        proxy_conf.append("--underscores_in_headers")

    if args.access_log:
        proxy_conf.extend(["--access_log",
                           args.access_log])
    if args.access_log_format:
        proxy_conf.extend(["--access_log_format",
                           args.access_log_format])

    if args.disable_tracing:
        proxy_conf.append("--disable_tracing")
    else:
        if args.tracing_project_id:
            proxy_conf.extend(["--tracing_project_id", args.tracing_project_id])
        if args.tracing_incoming_context:
            proxy_conf.extend(
                ["--tracing_incoming_context", args.tracing_incoming_context])
        if args.tracing_outgoing_context:
            proxy_conf.extend(
                ["--tracing_outgoing_context", args.tracing_outgoing_context])
        if args.cloud_trace_url_override:
            proxy_conf.extend(["--tracing_stackdriver_address",
                        args.cloud_trace_url_override])

        if args.disable_cloud_trace_auto_sampling:
            proxy_conf.extend(["--tracing_sample_rate", "0"])
        elif args.tracing_sample_rate:
            proxy_conf.extend(["--tracing_sample_rate",
                               str(args.tracing_sample_rate)])

    if args.transcoding_always_print_primitive_fields:
        proxy_conf.append("--transcoding_always_print_primitive_fields")

    if args.transcoding_always_print_enums_as_ints:
        proxy_conf.append("--transcoding_always_print_enums_as_ints")

    if args.transcoding_preserve_proto_field_names:
        proxy_conf.append("--transcoding_preserve_proto_field_names")

    if args.transcoding_ignore_query_parameters:
        proxy_conf.extend(["--transcoding_ignore_query_parameters",
                           args.transcoding_ignore_query_parameters])

    if args.transcoding_ignore_unknown_query_parameters:
        proxy_conf.append("--transcoding_ignore_unknown_query_parameters")

    if args.on_serverless:
        proxy_conf.extend([
            "--compute_platform_override", SERVERLESS_PLATFORM])

    if args.backend_dns_lookup_family:
        proxy_conf.extend(
            ["--backend_dns_lookup_family", args.backend_dns_lookup_family])

    if args.dns_resolver_addresses:
        proxy_conf.extend(
            ["--dns_resolver_addresses", args.dns_resolver_addresses])
    if args.dns:
        proxy_conf.extend(
            ["--dns_resolver_addresses", args.dns]
        )

    if args.envoy_use_remote_address:
        proxy_conf.append("--envoy_use_remote_address")

    if args.cors_preset:
        proxy_conf.extend([
            "--cors_preset",
            args.cors_preset,
            "--cors_allow_origin",
            args.cors_allow_origin,
            "--cors_allow_origin_regex",
            args.cors_allow_origin_regex,
            "--cors_allow_methods",
            args.cors_allow_methods,
            "--cors_allow_headers",
            args.cors_allow_headers,
            "--cors_expose_headers",
            args.cors_expose_headers,
        ])
        if args.cors_allow_credentials:
            proxy_conf.append("--cors_allow_credentials")

    # Set credentials file from the environment variable
    if args.service_account_key is None and GOOGLE_CREDS_KEY in os.environ:
        args.service_account_key = os.environ[GOOGLE_CREDS_KEY]

    if args.service_account_key:
        proxy_conf.extend(["--service_account_key", args.service_account_key])
    if args.non_gcp:
        proxy_conf.append("--non_gcp")

    if args.enable_debug:
        proxy_conf.append("--suppress_envoy_headers=false")

    if args.envoy_connection_buffer_limit_bytes:
        proxy_conf.extend(["--connection_buffer_limit_bytes",
                           args.envoy_connection_buffer_limit_bytes])

    return proxy_conf

def gen_envoy_args(args):
    cmd = [ENVOY_BIN, "-c", DEFAULT_CONFIG_DIR + BOOTSTRAP_CONFIG,
           "--disable-hot-restart",
           # This will print logs in `glog` format.
           # Stackdriver logging integrates nicely with this format.
           "--log-format %L%m%d %T.%e %t envoy] [%t][%n]%v",
           "--log-format-escaped"]

    if args.enable_debug:
        # Enable debug logging, but not for everything... too noisy otherwise.
        cmd.append("-l debug")
        cmd.append("--component-log-level upstream:info,main:info")

    return cmd

def output_reader(proc):
    for line in iter(proc.stdout.readline, b''):
        sys.stdout.write(line.decode())

def start_config_manager(proxy_conf):
    print("Starting Config Manager with args: {}".format(proxy_conf))
    proc = subprocess.Popen(proxy_conf,
                            stdout=subprocess.PIPE,
                            stderr=subprocess.STDOUT)
    t = threading.Thread(target=output_reader, args=(proc,))
    t.start()
    return proc

def start_envoy(args):
    subprocess.call(gen_bootstrap_conf(args))

    cmd = gen_envoy_args(args)
    print("Starting Envoy with args: {}".format(cmd))

    proc = subprocess.Popen(cmd,
                            stdout=subprocess.PIPE,
                            stderr=subprocess.STDOUT)
    t = threading.Thread(target=output_reader, args=(proc,))
    t.start()
    return proc


if __name__ == '__main__':
    logging.basicConfig(format='%(levelname)s: %(message)s', level=logging.INFO)

    parser = make_argparser()
    args = parser.parse_args()

    cm_proc = start_config_manager(gen_proxy_config(args))
    envoy_proc = start_envoy(args)

    while True:
        time.sleep(HEALTH_CHECK_PERIOD)
        if not cm_proc or cm_proc.poll():
            logging.fatal("Config Manager is down, killing all processes.")
            if envoy_proc:
               os.kill(envoy_proc.pid, signal.SIGKILL)
            sys.exit(1)
        if not envoy_proc or envoy_proc.poll():
            logging.fatal("Envoy is down, killing all processes.")
            if cm_proc:
               os.kill(cm_proc.pid, signal.SIGKILL)
            sys.exit(1)

