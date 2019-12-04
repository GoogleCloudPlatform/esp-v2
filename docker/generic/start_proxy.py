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
import subprocess
import sys
import threading

# Location of start proxy script
PROXY_STARTER = "apiproxy/start_proxy.sh"

# The command to generate Envoy bootstrap config
BOOTSTRAP_CMD = "apiproxy/bootstrap"

# bootstrap config file will write here.
# By default, envoy writes some logs to /tmp too
# If root file system is read-only, this folder should be
# mounted from tmpfs.
DEFAULT_CONFIG_DIR = "/tmp"

# Default Listener HTTP/1.x port
DEFAULT_LISTENER_HTTP1_PORT = 8080

# Default Backend HTTP/1.x port
DEFAULT_BACKEND_HTTP1_PORT = 80

# Default backend
DEFAULT_BACKEND = "127.0.0.1:8082"

# Default rollout_strategy
DEFAULT_ROLLOUT_STRATEGY = "fixed"

# Protocol prefixes
GRPC_PREFIX = "grpc://"
HTTP_PREFIX = "http://"
HTTPS_PREFIX = "https://"

# Google default application credentials environment variable
GOOGLE_CREDS_KEY = "GOOGLE_APPLICATION_CREDENTIALS"


def gen_bootstrap_conf(args):
    cmd = [BOOTSTRAP_CMD, "--logtostderr"]

    if args.disable_tracing:
        cmd.append("--disable_tracing")
    else:
        if args.tracing_project_id:
            cmd.extend(["--tracing_project_id", args.tracing_project_id])
        if args.tracing_sample_rate:
            cmd.extend(["--tracing_sample_rate", str(args.tracing_sample_rate)])
        if args.tracing_incoming_context:
            cmd.extend(
                ["--tracing_incoming_context", args.tracing_incoming_context])
        if args.tracing_outgoing_context:
            cmd.extend(
                ["--tracing_outgoing_context", args.tracing_outgoing_context])

    if args.http_request_timeout_s:
        cmd.extend(
            ["--http_request_timeout_s",
             str(args.http_request_timeout_s)])

    if args.enable_debug:
        os.environ["ENVOY_ARGS"] = "-l debug"
        cmd.append("--enable_admin")

    if args.enable_admin and not args.enable_debug:
        cmd.append("--enable_admin")

    bootstrap_file = DEFAULT_CONFIG_DIR + "/bootstrap.json"
    cmd.append(bootstrap_file)
    # Use environment variable to pass it to start_proxy.sh
    os.environ["BOOTSTRAP_FILE"] = bootstrap_file
    print(cmd)
    return cmd


def start_proxy(proxy_conf):
    try:
        os.execv(PROXY_STARTER, proxy_conf)
    except OSError as err:
        logging.error("Failed to launch ESP V2")
        logging.error(err.strerror)
        sys.exit(1)


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
ESP V2 start-up script. This script starts ConfigManager and Envoy.

The service name and config ID are optional. If not supplied, the ConfigManager
fetches the service name and the config ID from the metadata service as
attributes "service_name" and "service_config_id".

ESP V2 relies on the metadata service to fetch access tokens for Google
services. If you deploy ESP V2 outside of Google Cloud environment, you need
to provide a service account credentials file by setting "creds_key"
environment variable or by passing "-k" flag to this script.
            ''')

    parser.add_argument(
        '-s',
        '--service',
        default="",
        help=''' Set the name of the Endpoints service.  If omitted and -c not
        specified, ESP V2 contacts the metadata service to fetch the service
        name.  ''')

    parser.add_argument(
        '-v',
        '--version',
        default="",
        help=''' Set the service config ID of the Endpoints service.
        If omitted and -c not specified, ESP V2 contacts the metadata
        service to fetch the service config ID.  ''')

    parser.add_argument(
        '--service_json_path',
        default=None,
        help='''
        Specify a path for ApiProxy to load the endpoint service config.
        With this flag, ApiProxy will use "fixed" rollout strategy and following
        flags will be ignored:
           --service, --version, and --rollout_strategy.
        ''')

    parser.add_argument(
        '-a',
        '--backend',
        default=DEFAULT_BACKEND,
        help=''' Change the application server address to which ESP V2
        proxies requests. Default value: {backend}. For HTTPS backends,
        please use "https://" prefix, e.g. https://127.0.0.1:8082.
        For HTTP/1.x backends, prefix "http://" is optional.
        For GRPC backends, please use "grpc://" prefix,
        e.g. grpc://127.0.0.1:8082.'''.format(backend=DEFAULT_BACKEND))

    parser.add_argument(
        '--backend_protocol',
        default=None,
        help='''Backend Protocol. Overrides the protocol in --backend.
        Choices: [http1|http2|grpc].
        Default value: http1.''',
        choices=['http1', 'http2', 'grpc'])

    parser.add_argument('--http_port', default=None, type=int, help='''
       The port to accept HTTP/1.x connections.
       Default is {port}'''.format(port=DEFAULT_LISTENER_HTTP1_PORT))

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

    parser.add_argument(
        '--enable_backend_routing',
        action='store_true',
        default=False,
        help='''Enable apiproxy to route requests according to the
        "x-google-backend" or "backend" configurationn''')

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
        documentation for detailed information. The default value is 2.''')

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
        Set the timeout in second(eg. 10) for all the requests made by ConfigManager.
        Must be > 0 and the default is 5 seconds if not set.
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
        default=0.001,
        help="tracing sampling rate from 0.0 to 1.0")
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
        '--backend_dns_lookup_family',
        default=None,
        help='''
        Define the dns lookup family for all backends. The options are "auto", "v4only" and "v6only". The default is "auto".
        ''')
    parser.add_argument(
        '--compute_platform_override',
        default=None,
        help='''
        The overridden platform where the proxy is running on.
        ''')
    parser.add_argument('--enable_admin', action='store_true', default=False,
                        help='''
        Enables envoy's admin interface on port 8001.
        Not recommended for production use-cases, as the admin port is unauthenticated.
        ''')
    parser.add_argument('--enable_debug', action='store_true', default=False,
        help='''
        Enables a variety of debug features in both Config Manager and Envoy, such as:
        - Debug level per-request application logs in Envoy
        - Debug level service configuration logs in Config Manager
        - Admin interface in Envoy
        ''')
    return parser

# Check whether there are conflict flags. If so, return the error string. Otherwise returns None.
# This function also changes some default flag value.
def enforce_conflict_args(args):
    if args.rollout_strategy:
        if args.rollout_strategy not in {"fixed", "managed"}:
          return "Flag --R or  --rollout_strategy must be 'fixed' or 'managed'."
        if args.rollout_strategy != DEFAULT_ROLLOUT_STRATEGY:
          if args.version:
            return "Flag --version cannot be used together with -R or --rollout_strategy."
          if args.service_json_path:
            return "Flag --rollout_strategy must be fixed with --service_json_path."

    if args.service_json_path:
        if args.service:
            return "Flag --service cannot be used together with --service_json_path."
        if args.version:
            return "Flag --version cannot be used together with --service_json_path."

    # set non_gcp to True if service account key is provided.
    if args.service_account_key:
        args.non_gcp = True

    if args.non_gcp:
        if args.service_account_key is None and GOOGLE_CREDS_KEY not in os.environ:
            return "If --non_gcp is specified, --service_account_key has to be specified, or GOOGLE_APPLICATION_CREDENTIALS has to set in os.environ."
        if not args.tracing_project_id:
            # for non gcp case, disable tracing if tracing project id is not provided.
            args.disable_tracing = True

    if args.backend_dns_lookup_family and args.backend_dns_lookup_family not in {"auto", "v4only", "v6only"}:
        return "Flag --backend_dns_lookup_family must be 'auto', 'v4only' or 'v6only'."

    return None

def gen_proxy_config(args):
    check_conflict_result = enforce_conflict_args(args)
    if check_conflict_result:
        logging.error(check_conflict_result)
        sys.exit(1)

    if args.backend_protocol is None:
        if args.backend.startswith(GRPC_PREFIX):
            backend_protocol = "grpc"
            backends = args.backend[len(GRPC_PREFIX):]
        elif args.backend.startswith(HTTP_PREFIX):
            backend_protocol = "http1"
            backends = args.backend[len(HTTP_PREFIX):]
        elif args.backend.startswith(HTTPS_PREFIX):
            backend_protocol = "http2"
            backend = args.backend[len(HTTPS_PREFIX):]
            if not re.search(r':[0-9]+$', backend):
                backend = backend + ':443'
            backends = backend
        else:
            backend_protocol = "http1"
            backends = args.backend
    else:
        backend_protocol = args.backend_protocol
        backends = args.backend

    cluster_args = backends.split(':')
    if len(cluster_args) == 2:
        cluster_address = cluster_args[0]
        cluster_port = cluster_args[1]
    elif len(cluster_args) == 1:
        cluster_address = cluster_args[0]
        cluster_port = str(DEFAULT_BACKEND_HTTP1_PORT)
    else:
        print("incorrect backend")
        sys.exit(1)

    proxy_conf = [
       "-v", "--logtostderr", "--backend_protocol", backend_protocol,
        "--cluster_address", cluster_address, "--cluster_port", cluster_port,
        "--rollout_strategy", args.rollout_strategy,
    ]

    if args.enable_debug:
        proxy_conf.extend(["--v", "1"])
    else:
        proxy_conf.extend(["--v", "0"])

    if args.envoy_xff_num_trusted_hops:
         proxy_conf.extend(["--envoy_xff_num_trusted_hops", args.envoy_xff_num_trusted_hops])

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
        proxy_conf.append("--check_metadata", )

    if args.disable_tracing:
        proxy_conf.append("--disable_tracing", )

    if args.compute_platform_override:
        proxy_conf.extend([
            "--compute_platform_override", args.compute_platform_override])

    if args.enable_backend_routing:
        proxy_conf.append("--enable_backend_routing")

    if args.backend_dns_lookup_family:
        proxy_conf.extend(
            ["--backend_dns_lookup_family", args.backend_dns_lookup_family])

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
    print(proxy_conf)
    return proxy_conf

if __name__ == '__main__':
    parser = make_argparser()
    args = parser.parse_args()
    logging.basicConfig(format='%(levelname)s:%(message)s', level=logging.INFO)
    subprocess.call(gen_bootstrap_conf(args))
    start_proxy(gen_proxy_config(args))
