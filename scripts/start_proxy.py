# Copyright 2018 Google Cloud Platform Proxy Authors

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
import threading
import sys
import re

# Location of start proxy script
PROXY_STARTER = "apiproxy/start_proxy.sh"

# Default HTTP/1.x port
DEFAULT_PORT = '8082'

# Default backend
DEFAULT_BACKEND = "127.0.0.1:8082"

# Default rollout_strategy
DEFAULT_ROLLOUT_STRATEGY = "fixed"

# Protocol prefixes
GRPC_PREFIX = "grpc://"
HTTP_PREFIX = "http://"
HTTPS_PREFIX = "https://"

# Management service
MANAGEMENT_ADDRESS = "https://servicemanagement.googleapis.com"

# Metadata service
METADATA_ADDRESS = "http://169.254.169.254"

def start_proxy(proxy_conf):
    try:
        os.execv(PROXY_STARTER, proxy_conf)
    except OSError as err:
        logging.error("Failed to launch Api Proxy")
        logging.error(err.strerror)
        sys.exit(3)

class ArgumentParser(argparse.ArgumentParser):
    def error(self, message):
        self.print_help(sys.stderr)
        self.exit(4, '%s: error: %s\n' % (self.prog, message))

# Notes: These flags should get aligned with that of ESP at
# https://github.com/cloudendpoints/esp/blob/master/start_esp/start_esp.py#L420
def make_argparser():
    parser = ArgumentParser(formatter_class=argparse.RawDescriptionHelpFormatter,
            description = '''
API Proxy start-up script. This script starts ConfigManager and Envoy.

The service name and config ID are optional. If not supplied, the ConfigManager
fetches the service name and the config ID from the metadata service as
attributes "service_name" and "service_config_id".

Api Proxy relies on the metadata service to fetch access tokens for Google
services. If you deploy API Proxy outside of Google Cloud environment, you need
to provide a service account credentials file by setting "creds_key"
environment variable or by passing "-k" flag to this script.
            ''')

    parser.add_argument('-s', '--service',
        default = "",
        help=''' Set the name of the Endpoints service.  If omitted and -c not
        specified, API proxy contacts the metadata service to fetch the service
        name.  ''')

    parser.add_argument('-v', '--version',
        default = "",
        help=''' Set the service config ID of the Endpoints service.
        If omitted and -c not specified, API proxy contacts the metadata
        service to fetch the service config ID.  ''')

    parser.add_argument('-a', '--backend', default=DEFAULT_BACKEND,
        help=''' Change the application server address to which API Proxy
        proxies requests. Default value: {backend}. For HTTPS backends,
        please use "https://" prefix, e.g. https://127.0.0.1:8082.
        For HTTP/1.x backends, prefix "http://" is optional.
        For GRPC backends, please use "grpc://" prefix,
        e.g. grpc://127.0.0.1:8082.'''.format(backend=DEFAULT_BACKEND))

    parser.add_argument('-R', '--rollout_strategy',
        default=None,
        help='''The service config rollout strategy, [fixed|managed],
        Default value: {strategy}'''.format(strategy=DEFAULT_ROLLOUT_STRATEGY),
        choices=['fixed', 'managed'])

    # Customize management service url prefix.
    parser.add_argument('-g', '--management',
        default=MANAGEMENT_ADDRESS,
        help=argparse.SUPPRESS)

    # Customize metadata service url prefix.
    parser.add_argument('-m', '--metadata',
        default=METADATA_ADDRESS,
        help=argparse.SUPPRESS)

    # CORS presets
    parser.add_argument('--cors_preset',
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
    parser.add_argument('--cors_allow_origin',
        default='*',
        help='''
        Only works when --cors_preset is 'basic'. Configures the CORS header
        Access-Control-Allow-Origin. Defaults to "*" which allows all origins.
        ''')
    parser.add_argument('--cors_allow_origin_regex',
        default='',
        help='''
        Only works when --cors_preset is 'cors_with_regex'. Configures the
        whitelists of CORS header Access-Control-Allow-Origin with regular
        expression.
        ''')
    parser.add_argument('--cors_allow_methods',
        default='GET, POST, PUT, PATCH, DELETE, OPTIONS',
        help='''
        Only works when --cors_preset is in use. Configures the CORS header
        Access-Control-Allow-Methods. Defaults to allow common HTTP
        methods.
        ''')
    parser.add_argument('--cors_allow_headers',
        default='DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization',
        help='''
        Only works when --cors_preset is in use. Configures the CORS header
        Access-Control-Allow-Headers. Defaults to allow common HTTP
        headers.
        ''')
    parser.add_argument('--cors_expose_headers',
        default='Content-Length,Content-Range',
        help='''
        Only works when --cors_preset is in use. Configures the CORS header
        Access-Control-Expose-Headers. Defaults to allow common response headers.
        ''')
    parser.add_argument('--cors_allow_credentials', action='store_true',
        help='''
        Only works when --cors_preset is in use. Enable the CORS header
        Access-Control-Allow-Credentials. By default, this header is disabled.
        ''')
    #TODO(jcwang) fetch access token from --service_account_key
    parser.add_argument('--check_metadata', action='store_true',
        help='''Enable fetching service name, service config ID and rollout
        strategy from the metadata service.''')

    return parser


if __name__ == '__main__':
    parser = make_argparser()
    args = parser.parse_args()
    logging.basicConfig(format='%(levelname)s:%(message)s', level=logging.INFO)

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

    cluster_args = backends.split(':')
    if len(cluster_args) == 2:
        cluster_address =  cluster_args[0]
        cluster_port = cluster_args[1]
    elif len(cluster_args) == 1:
        cluster_address =  cluster_args[0]
        cluster_port = DEFAULT_PORT
    else:
        print ("incorrect backend")
        sys.exit(0)

    if args.rollout_strategy is None or not args.rollout_strategy.strip():
        args.rollout_strategy = DEFAULT_ROLLOUT_STRATEGY

    proxy_conf = ["-v",
        "--backend_protocol", backend_protocol,
        "--service", args.service,
        "--version", args.version,
        "--cluster_address", cluster_address,
        "--cluster_port", cluster_port,
        "--service_management_url", args.management,
        "--check_metadata", str(args.check_metadata),
        "--rollout_strategy", args.rollout_strategy,
    ]

    if args.cors_preset:
        proxy_conf.append([
        "--cors_preset", args.cors_preset,
        "--cors_allow_origin", args.cors_allow_origin,
        "--cors_allow_origin_regex", args.cors_allow_origin_regex,
        "--cors_allow_methods", args.cors_allow_methods,
        "--cors_allow_headers", args.cors_allow_headers,
        "--cors_expose_headers", args.cors_expose_headers,
        "cors_allow_credentials", args.cors_allow_credentials,
        ])

    print (proxy_conf)
    start_proxy(proxy_conf)
