# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import inspect
import json
import os
import sys
import tempfile
import unittest
from unittest import mock

currentdir = os.path.dirname(
    os.path.abspath(inspect.getfile(inspect.currentframe())))
sys.path.insert(0, currentdir + "/../../docker/generic")
from start_proxy import (
    gen_bootstrap_conf,
    make_argparser,
    gen_proxy_config,
    gen_envoy_args,
    GOOGLE_CREDS_KEY,
    fetch_project_id_from_metadata,
    fetch_project_id_from_service_account_key,
    setup_otel_resource_attributes,
)


class TestStartProxy(unittest.TestCase):

    def setUp(self):
        self.parser = make_argparser()

    def test_gen_bootstrap(self):
        testcases = [
            (["--http_request_timeout_s=1", "--disable_tracing", "--admin_port=8001"],
             ['bin/bootstrap', '--logtostderr', '--admin_port', '8001',
              '--http_request_timeout_s', '1',
              '/tmp/bootstrap.json']),
            (["--ads_named_pipe=@espv2-named-pipe-9", "--disable_tracing", "--admin_port=8001"],
             ['bin/bootstrap', '--logtostderr', '--admin_port', '8001',
              '--ads_named_pipe', '@espv2-named-pipe-9',
              '/tmp/bootstrap.json']),
            ([], ['bin/bootstrap',
                  '--logtostderr', '--admin_port', '0',
                  '/tmp/bootstrap.json']),
        ]

        for flags, wantedArgs in testcases:
            gotArgs = gen_bootstrap_conf(self.parser.parse_args(flags))
            self.assertEqual(gotArgs, wantedArgs)

    def test_gen_proxy_config(self):
        testcases = [
            # grpc backend with fixed version.
            (['--service=test_bookstore.gloud.run', '--version=2019-11-09r0',
              '--backend=grpc://127.0.0.1:8000', '--http_request_timeout_s=10',
              '--log_jwt_payloads=aud,exp', '--disable_tracing', '--healthz=/'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'grpc://127.0.0.1:8000','--healthz', '/',
              '--v', '0', '--log_jwt_payloads', 'aud,exp',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--http_request_timeout_s', '10',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing',
              ]),
            # backend with DNS address, no version.
            (['--service=echo.gloud.run', '--backend=http://echo:8080',
              '--log_request_headers=x-google-x', '--version=2019-11-09r0',
              '--service_control_check_timeout_ms=100', '-z=hc',
              '--backend_dns_lookup_family=v4only', '--disable_tracing',
              '--dns_resolver_addresses=127.0.0.1:53'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'http://echo:8080',
              '--healthz', 'hc', '--v', '0',
              '--log_request_headers', 'x-google-x',
              '--service', 'echo.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_check_timeout_ms', '100',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing',
              '--backend_dns_lookup_family', 'v4only',
              '--dns_resolver_addresses', '127.0.0.1:53'
              ]),
            (['--service=echo.gloud.run', '--backend=http://echo:8080',
              '--log_request_headers=x-google-x', '--version=2019-11-09r0',
              '--service_control_check_timeout_ms=100', '-z=hc',
              '--backend_dns_lookup_family=v4only', '--disable_tracing',
              '--dns=127.0.0.1:53'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'http://echo:8080',
              '--healthz', 'hc', '--v', '0',
              '--log_request_headers', 'x-google-x',
              '--service', 'echo.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_check_timeout_ms', '100',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing',
              '--backend_dns_lookup_family', 'v4only',
              '--dns_resolver_addresses', '127.0.0.1:53'
              ]),
            # Default backend
            (['-R=managed','--enable_strict_transport_security',
              '--http_port=8079', '--service_control_quota_retries=3',
              '--service_control_report_timeout_ms=300',
              '--check_metadata',
              '--disable_tracing', '--underscores_in_headers'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8079', '--enable_strict_transport_security',
              '--service_control_quota_retries', '3',
              '--service_control_report_timeout_ms', '300',
              '--service_control_enable_api_key_uid_reporting',
              '--check_metadata', '--underscores_in_headers',
              '--disable_tracing'
              ]),
            # disable_jwks_async_fetch
            (['-R=managed','--disable_jwks_async_fetch',
              '--http_port=8079', '--service_control_quota_retries=3',
              '--service_control_report_timeout_ms=300',
              '--check_metadata',
              '--disable_tracing', '--underscores_in_headers'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--disable_jwks_async_fetch',
              '--listener_port', '8079',
              '--service_control_quota_retries', '3',
              '--service_control_report_timeout_ms', '300',
              '--service_control_enable_api_key_uid_reporting',
              '--check_metadata', '--underscores_in_headers',
              '--disable_tracing'
              ]),
            # jwks_async_fetch_fast_listener
            (['-R=managed','--jwks_async_fetch_fast_listener',
              '--http_port=8079', '--service_control_quota_retries=3',
              '--service_control_report_timeout_ms=300',
              '--check_metadata',
              '--disable_tracing', '--underscores_in_headers'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--jwks_async_fetch_fast_listener',
              '--listener_port', '8079',
              '--service_control_quota_retries', '3',
              '--service_control_report_timeout_ms', '300',
              '--service_control_enable_api_key_uid_reporting',
              '--check_metadata', '--underscores_in_headers',
              '--disable_tracing'
              ]),
            # jwt_cache_size
            (['-R=managed','--jwt_cache_size=300',
              '--http_port=8079', '--service_control_quota_retries=3',
              '--service_control_report_timeout_ms=300',
              '--check_metadata',
              '--disable_tracing', '--underscores_in_headers'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--jwt_cache_size', '300',
              '--listener_port', '8079',
              '--service_control_quota_retries', '3',
              '--service_control_report_timeout_ms', '300',
              '--service_control_enable_api_key_uid_reporting',
              '--check_metadata', '--underscores_in_headers',
              '--disable_tracing'
              ]),
            # --disable_jwt_audience_service_name_check
            (['-R=managed','--disable_jwt_audience_service_name_check',
              '--http_port=8079', '--service_control_quota_retries=3',
              '--service_control_report_timeout_ms=300',
              '--check_metadata',
              '--disable_tracing', '--underscores_in_headers'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--disable_jwt_audience_service_name_check',
              '--listener_port', '8079',
              '--service_control_quota_retries', '3',
              '--service_control_report_timeout_ms', '300',
              '--service_control_enable_api_key_uid_reporting',
              '--check_metadata', '--underscores_in_headers',
              '--disable_tracing'
              ]),
            # jwks_fetch retry backoff
            (['-R=managed','--disable_jwks_async_fetch',
              '--jwks_fetch_num_retries=10',
              '--jwks_fetch_retry_back_off_base_interval=100',
              '--jwks_fetch_retry_back_off_max_interval=32000',
              '--jwt_pad_forward_payload_header',
              '--http_port=8079', '--service_control_quota_retries=3',
              '--service_control_report_timeout_ms=300',
              '--check_metadata',
              '--disable_tracing', '--underscores_in_headers'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--disable_jwks_async_fetch',
              '--jwks_fetch_num_retries', '10',
              '--jwks_fetch_retry_back_off_base_interval_ms', '100',
              '--jwks_fetch_retry_back_off_max_interval_ms', '32000',
              '--jwt_pad_forward_payload_header',
              '--listener_port', '8079',
              '--service_control_quota_retries', '3',
              '--service_control_report_timeout_ms', '300',
              '--service_control_enable_api_key_uid_reporting',
              '--check_metadata', '--underscores_in_headers',
              '--disable_tracing'
              ]),
            # service_control_network_fail_policy=open
            (['-R=managed','--enable_strict_transport_security',
              '--http_port=8079', '--service_control_quota_retries=3',
              '--service_control_report_timeout_ms=300',
              '--service_control_network_fail_policy=open', '--check_metadata',
              '--disable_tracing', '--underscores_in_headers'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8079', '--enable_strict_transport_security',
              '--service_control_quota_retries', '3',
              '--service_control_report_timeout_ms', '300',
              '--service_control_enable_api_key_uid_reporting',
              '--check_metadata', '--underscores_in_headers',
              '--disable_tracing'
              ]),
            # no-service_control_enable_api_key_uid_reporting
            (['-R=managed','--enable_strict_transport_security',
              '--http_port=8079', '--service_control_quota_retries=3',
              '--service_control_report_timeout_ms=300',
              '--service_control_network_fail_policy=open', '--check_metadata',
              '--no-service_control_enable_api_key_uid_reporting',
              '--disable_tracing', '--underscores_in_headers'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8079', '--enable_strict_transport_security',
              '--service_control_quota_retries', '3',
              '--service_control_report_timeout_ms', '300',
              '--check_metadata', '--underscores_in_headers',
              '--disable_tracing'
              ]),
              # service_control_url specified
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--service_control_url=https://non-default-servicecontrol.googleapis.com'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8080', '--service_control_url',
              'https://non-default-servicecontrol.googleapis.com',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing'
              ]),
            # service_control_network_fail_policy=close
            (['-R=managed','--enable_strict_transport_security',
              '--http_port=8079', '--service_control_quota_retries=3',
              '--service_control_report_timeout_ms=300',
              '--service_control_network_fail_policy=close', '--check_metadata',
              '--disable_tracing', '--underscores_in_headers'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8079', '--enable_strict_transport_security',
              '--service_control_quota_retries', '3',
              '--service_control_report_timeout_ms', '300',
              '--service_control_network_fail_open=false',
              '--service_control_enable_api_key_uid_reporting',
              '--check_metadata', '--underscores_in_headers',
              '--disable_tracing'
              ]),
            # ssl_server_cert_path specified
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--ssl_server_cert_path=/etc/endpoint/ssl'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8080', '--ssl_server_cert_path',
              '/etc/endpoint/ssl',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing'
              ]),
            # ssl_server_root_cert_path specified
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--ssl_server_root_cert_path=/etc/endpoint/ssl/root.cert'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8080', '--ssl_server_root_cert_path',
              '/etc/endpoint/ssl/root.cert',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing'
              ]),
            # legacy ssl_port specified
            (['-R=managed','--ssl_port=9000', '--disable_tracing'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--ssl_server_cert_path', '/etc/nginx/ssl',
              '--listener_port', '9000',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing',
              ]),
            # ssl_backend_client_cert_path specified
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--ssl_backend_client_cert_path=/etc/endpoint/ssl'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8080', '--ssl_backend_client_cert_path',
              '/etc/endpoint/ssl',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing'
              ]),
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--ssl_client_cert_path=/etc/endpoint/ssl'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8080', '--ssl_backend_client_cert_path',
              '/etc/endpoint/ssl',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing'
              ]),
            # ssl_backend_client_root_certs_file specified
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--ssl_backend_client_root_certs_file=/etc/endpoints/ssl/ca-certificates.crt' ],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8080', '--ssl_backend_client_root_certs_path',
              '/etc/endpoints/ssl/ca-certificates.crt',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing'
              ]),
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--ssl_client_root_certs_file=/etc/endpoints/ssl/ca-certificates.crt' ],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8080', '--ssl_backend_client_root_certs_path',
              '/etc/endpoints/ssl/ca-certificates.crt',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing'
              ]),
            # legacy enable_grpc_backend_ssl specified
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--enable_grpc_backend_ssl' ],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8080', '--ssl_backend_client_root_certs_path',
              '/etc/nginx/trusted-ca-certificates.crt',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing'
              ]),
            # legacy tls_mutual_auth specified
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--tls_mutual_auth'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8080', '--ssl_backend_client_cert_path',
              '/etc/nginx/ssl',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing'
              ]),
            # ssl_minimum_protocol and ssl_maximum_protocol specified
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--ssl_minimum_protocol=TLSv1.1',
              '--ssl_maximum_protocol=TLSv1.3'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8080', '--ssl_minimum_protocol',
              'TLSv1.1','--ssl_maximum_protocol','TLSv1.3',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing'
              ]),
            # ssl_server_cipher_suites and ssl_backend_client_cipher_suites specified
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--ssl_server_cipher_suites=AES128-SHA,AES256-GCM-SHA384',
              '--ssl_backend_client_cipher_suites=AES256-SHA'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8080',
              '--ssl_server_cipher_suites', 'AES128-SHA,AES256-GCM-SHA384',
              '--ssl_backend_client_cipher_suites', 'AES256-SHA',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing'
              ]),
            # legacy --ssl_protocols specified
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--ssl_protocols=TLSv1.3', '--ssl_protocols=TLSv1.2'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8080', '--ssl_minimum_protocol',
              'TLSv1.2','--ssl_maximum_protocol','TLSv1.3',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing'
              ]),
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--ssl_protocols=TLSv1.2'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8080', '--ssl_minimum_protocol',
              'TLSv1.2','--ssl_maximum_protocol','TLSv1.2',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing'
              ]),
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--generate_self_signed_cert'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8080', '--ssl_server_cert_path',
              '/tmp/ssl/endpoints',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing'
              ]),
            # http2_port specified.
            (['-R=managed',
              '--http2_port=8079', '--service_control_quota_retries=3',
              '--service_control_report_timeout_ms=300',
              '--check_metadata',
              '--disable_tracing'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8079',
              '--service_control_quota_retries', '3',
              '--service_control_report_timeout_ms', '300',
              '--service_control_enable_api_key_uid_reporting',
              '--check_metadata',
              '--disable_tracing'
              ]),
            # listener_port specified.
            (['-R=managed',
              '--listener_port=8079', '--service_control_quota_retries=3',
              '--service_control_report_timeout_ms=300',
              '--check_metadata',
              '--disable_tracing'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8079',
              '--service_control_quota_retries', '3',
              '--service_control_report_timeout_ms', '300',
              '--service_control_enable_api_key_uid_reporting',
              '--check_metadata',
              '--disable_tracing'
              ]),
            # Cors: test default CORS flag values
            (['--service=test_bookstore.gloud.run',
              '--backend=https://127.0.0.1', '--cors_preset=basic',
              '--non_gcp', '--version=2019-11-09r0',
              '--service_account_key', '/tmp/service_accout_key'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'https://127.0.0.1', '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing',
              '--cors_preset', 'basic',
              '--cors_allow_origin', '*', '--cors_allow_origin_regex', '',
              '--cors_allow_methods', 'GET, POST, PUT, PATCH, DELETE, OPTIONS',
              '--cors_allow_headers', 'DNT,User-Agent,X-User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization',
              '--cors_expose_headers', 'Content-Length,Content-Range',
              '--cors_max_age', "480h",
              '--service_account_key', '/tmp/service_accout_key', '--non_gcp',
              ]),
            # Cors: test CORS flag valus are passed to config_manager correctly
            (['--service=test_bookstore.gloud.run',
              '--backend=https://127.0.0.1', '--cors_preset=cors_with_regex',
              '--cors_allow_origin_regex=^https?://.+.google.com$',
              '--cors_allow_methods=GET,POST,OPTIONS',
              '--cors_allow_headers=X-Requested-With,Content-Type,Range,Authorization',
              '--cors_expose_headers=Content-Length',
              '--cors_allow_credentials',
              '--cors_max_age=1200m',
              '--non_gcp', '--version=2019-11-09r0',
              '--service_account_key', '/tmp/service_accout_key'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'https://127.0.0.1', '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing',
              '--cors_preset', 'cors_with_regex',
              '--cors_allow_origin', '*',
              '--cors_allow_origin_regex', '^https?://.+.google.com$',
              '--cors_allow_methods', 'GET,POST,OPTIONS',
              '--cors_allow_headers', 'X-Requested-With,Content-Type,Range,Authorization',
              '--cors_expose_headers', 'Content-Length',
              '--cors_max_age', "1200m",
              '--cors_allow_credentials',
              '--service_account_key', '/tmp/service_accout_key', '--non_gcp',
              ]),
            # backend routing (with deprecated flag)
            (['--backend=https://127.0.0.1:8000', '--enable_backend_routing',
              '--service_json_path=/tmp/service.json',
              '--on_serverless',
              '--disable_tracing'],
             ['bin/configmanager',  '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'https://127.0.0.1:8000', '--v', '0',
              '--envoy_xff_num_trusted_hops', '0',
              '--service_control_enable_api_key_uid_reporting',
              '--service_json_path', '/tmp/service.json',
              '--disable_tracing',
              '--compute_platform_override', 'Cloud Run(ESPv2)'
              ]),
            # grpc backend with fixed version and tracing
            (['--service=test_bookstore.gloud.run', '--version=2019-11-09r0',
              '--backend=grpc://127.0.0.1:8000', '--http_request_timeout_s=10',
              '--log_jwt_payloads=aud,exp', '--disable_tracing'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'grpc://127.0.0.1:8000', '--v', '0',
              '--log_jwt_payloads', 'aud,exp',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--http_request_timeout_s', '10',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing',
              ]),
            # json-grpc transcoder json print options
            (['--service=test_bookstore.gloud.run',
              '--backend=grpc://127.0.0.1:8000',
              '--transcoding_always_print_primitive_fields',
              '--transcoding_preserve_proto_field_names',
              '--transcoding_always_print_enums_as_ints',
              '--disable_tracing', '--version=2019-11-09r0',
              ],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'grpc://127.0.0.1:8000', '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing',
              '--transcoding_always_print_primitive_fields',
              '--transcoding_always_print_enums_as_ints',
              '--transcoding_preserve_proto_field_names',
              ]),
            # json-grpc transcoder ignore query parameters
            (['--service=test_bookstore.gloud.run',
              '--backend=grpc://127.0.0.1:8000',
              '--transcoding_ignore_query_parameters=foo,bar',
              '--disable_tracing',
              '--version=2019-11-09r0',
              ],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'grpc://127.0.0.1:8000', '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing',
              '--transcoding_ignore_query_parameters', 'foo,bar'
              ]),
            # json-grpc transcoder ignore unknown parameters
            (['--service=test_bookstore.gloud.run',
              '--backend=grpc://127.0.0.1:8000',
              '--transcoding_ignore_unknown_query_parameters',
              '--disable_tracing',
              '--version=2019-11-09r0',
              ],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'grpc://127.0.0.1:8000', '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing',
              '--transcoding_ignore_unknown_query_parameters'
              ]),
            # json-grpc transcoding_query_parameters_disable_unescape_plus
            (['--service=test_bookstore.gloud.run',
              '--backend=grpc://127.0.0.1:8000',
              '--transcoding_query_parameters_disable_unescape_plus',
              '--disable_tracing',
              '--version=2019-11-09r0',
              ],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'grpc://127.0.0.1:8000', '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing',
              '--transcoding_query_parameters_disable_unescape_plus'
              ]),
            # json-grpc transcoding_stream_newline_delimited
            (['--service=test_bookstore.gloud.run',
              '--backend=grpc://127.0.0.1:8000',
              '--transcoding_stream_newline_delimited',
              '--disable_tracing',
              '--version=2019-11-09r0',
              ],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'grpc://127.0.0.1:8000', '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing',
              '--transcoding_stream_newline_delimited'
              ]),
            # json-grpc transcoding_case_insensitive_enum_parsing
            (['--service=test_bookstore.gloud.run',
              '--backend=grpc://127.0.0.1:8000',
              '--transcoding_case_insensitive_enum_parsing',
              '--disable_tracing',
              '--version=2019-11-09r0',
              ],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'grpc://127.0.0.1:8000', '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing',
              '--transcoding_case_insensitive_enum_parsing'
              ]),
            # route_match disallow_colon_in_wildcard_path_segment
            (['--service=test_bookstore.gloud.run',
              '--backend=grpc://127.0.0.1:8000',
              '--disallow_colon_in_wildcard_path_segment',
              '--disable_tracing',
              '--version=2019-11-09r0',
              ],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'grpc://127.0.0.1:8000', '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing',
              '--disallow_colon_in_wildcard_path_segment'
              ]),
            # Connection buffer limit bytes
            (['--service=test_bookstore.gloud.run',
              '--backend=grpc://127.0.0.1:8000',
              '--envoy_connection_buffer_limit_bytes=1024',
              '--disable_tracing',
              '--version=2019-11-09r0',
              ],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'grpc://127.0.0.1:8000', '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing',
              '--connection_buffer_limit_bytes', '1024'
              ]),
            # --enable_debug, with default http schema
            (['--service=test_bookstore.gloud.run',
              '--backend=echo:8000',
              '--enable_debug',
              '--disable_tracing',
              '--version=2019-11-09r0',
              ],
             ['bin/configmanager', '--logtostderr',
              '--rollout_strategy', 'fixed',
              '--backend_address', 'http://echo:8000',
              '--v', '1',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing',
              '--suppress_envoy_headers=false'
              ]),
            (['--service=test_bookstore.gloud.run',
              '--backend=127.0.0.1:8000',
              '--access_log=/foo/bar', "--access_log_format=%START_TIME%",
              '--disable_tracing',
              '--version=2019-11-09r0',
              ],
             ['bin/configmanager', '--logtostderr',
              '--rollout_strategy', 'fixed',
              '--backend_address', 'http://127.0.0.1:8000',
              '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--access_log', '/foo/bar',
              '--access_log_format', '%START_TIME%',
              '--disable_tracing',
              ]),
            # Tracing disabled on non-gcp
            (['--service=test_bookstore.gloud.run',
              '--backend=http://127.0.0.1', '--version=2019-11-09r0',
              '--service_account_key', '/tmp/service_accout_key', '--non_gcp'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'http://127.0.0.1', '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing',
              '--service_account_key', '/tmp/service_accout_key', '--non_gcp',
              ]),
            # Use Application Default Credentials with non-gcp
            (['--service=test_bookstore.gloud.run',
              '--backend=http://127.0.0.1', '--version=2019-11-09r0',
              '--enable_application_default_credentials', '--non_gcp', '--disable_tracing'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'http://127.0.0.1', '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing', '--enable_application_default_credentials',
              '--non_gcp',
              ]),
            # Tracing enabled when manually specifying project id on non-gcp.
            (['--service=test_bookstore.gloud.run',
              '--backend=http://127.0.0.1', '--version=2019-11-09r0',
              '--service_account_key', '/tmp/service_accout_key', '--non_gcp',
              '--tracing_project_id=test_project_1234'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'http://127.0.0.1', '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--tracing_project_id', 'test_project_1234',
              '--service_account_key', '/tmp/service_accout_key', '--non_gcp',
              ]),
            # Tracing params preserved.
            (['--service=test_bookstore.gloud.run',
              '--backend=grpc://127.0.0.1:8000',
              '--tracing_sample_rate=1',
              '--cloud_trace_url_override=localhost:9990',
              '--tracing_incoming_context=fake-incoming-context',
              '--tracing_outgoing_context=fake-outgoing-context',
              '--version=2019-11-09r0',
              ],
             ['bin/configmanager', '--logtostderr',
              '--rollout_strategy', 'fixed',
              '--backend_address', 'grpc://127.0.0.1:8000',
              '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--tracing_incoming_context', 'fake-incoming-context',
              '--tracing_outgoing_context', 'fake-outgoing-context',
              '--tracing_stackdriver_address', 'localhost:9990',
              '--tracing_sample_rate', '1',
              ]),
            # Enable debug affects tracing.
            (['--service=test_bookstore.gloud.run',
              '--backend=grpc://127.0.0.1:8000',
              '--tracing_sample_rate=1',
              '--version=2019-11-09r0',
              '--enable_debug',
              ],
             ['bin/configmanager', '--logtostderr',
              '--rollout_strategy', 'fixed',
              '--backend_address', 'grpc://127.0.0.1:8000',
              '--v', '1',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--tracing_sample_rate', '1',
              # '--tracing_enable_verbose_annotations',
              '--suppress_envoy_headers=false',
              ]),
            # --disable_cloud_trace_auto_sampling overrides --tracing_sample_rate
            (['--service=test_bookstore.gloud.run',
              '--backend=grpc://127.0.0.1:8000',
              '--tracing_sample_rate=1',
              '--disable_cloud_trace_auto_sampling',
              '--version=2019-11-09r0',
              ],
             ['bin/configmanager', '--logtostderr',
              '--rollout_strategy', 'fixed',
              '--backend_address', 'grpc://127.0.0.1:8000',
              '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--tracing_sample_rate', '0',
              ]),
            # --disable_tracing overrides all other tracing flags
            (['--service=test_bookstore.gloud.run',
              '--backend=grpc://127.0.0.1:8000',
              '--tracing_sample_rate=1',
              '--disable_tracing',
              '--version=2019-11-09r0',
              ],
             ['bin/configmanager', '--logtostderr',
              '--rollout_strategy', 'fixed',
              '--backend_address', 'grpc://127.0.0.1:8000',
              '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing',
              ]),
            # backend retry setting
            (['-R=managed',
              '--http2_port=8079', '--backend_retry_ons=5xx',
              '--disable_tracing'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8079',
              '--service_control_enable_api_key_uid_reporting',
              '--backend_retry_ons', '5xx',
              '--disable_tracing'
              ]),
            (['-R=managed',
              '--http2_port=8079', '--backend_retry_num=10',
              '--disable_tracing'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8079',
              '--service_control_enable_api_key_uid_reporting',
              '--backend_retry_num', '10',
              '--disable_tracing'
              ]),
            (['-R=managed',
              '--http2_port=8079', '--backend_per_try_timeout=10s',
              '--disable_tracing'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8079',
              '--service_control_enable_api_key_uid_reporting',
              '--backend_per_try_timeout', '10s',
              '--disable_tracing'
              ]),
            (['-R=managed',
              '--http2_port=8079', '--backend_retry_on_status_codes=500,501',
              '--disable_tracing'],
             ['bin/configmanager', '--logtostderr', '--rollout_strategy', 'managed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--listener_port', '8079',
              '--service_control_enable_api_key_uid_reporting',
              '--backend_retry_on_status_codes', '500,501',
              '--disable_tracing'
              ]),
            # Service account key does not assume non-gcp
            # and does not disable tracing.
            (['--service=test_bookstore.gloud.run',
              '--backend=http://127.0.0.1',
              '--service_account_key', '/tmp/service_account_key',
              '--disable_tracing',
              '--version=2019-11-09r0',
              ],
             ['bin/configmanager', '--logtostderr',
              '--rollout_strategy', 'fixed',
              '--backend_address', 'http://127.0.0.1',
              '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--disable_tracing',
              '--service_account_key', '/tmp/service_account_key',
              ]),
            # Serverless sets compute platform and xff_num_trusted_hops.
            (['--on_serverless',
              '--http_port=8080',
              '--rollout_strategy=fixed',
              '--service_json_path=/tmp/service_config.json',
              '--disable_tracing',
              ],
             ['bin/configmanager',  '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--envoy_xff_num_trusted_hops', '0',
              '--listener_port', '8080',
              '--service_control_enable_api_key_uid_reporting',
              '--service_json_path', '/tmp/service_config.json',
              '--disable_tracing',
              '--compute_platform_override', 'Cloud Run(ESPv2)'
              ]),
            # Serverless with override for xff_num_trusted_hops.
            (['--on_serverless',
              '--http_port=8080',
              '--rollout_strategy=fixed',
              '--service_json_path=/tmp/service_config.json',
              '--envoy_xff_num_trusted_hops=3',
              '--disable_tracing',
              ],
             ['bin/configmanager',  '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--envoy_xff_num_trusted_hops', '3',
              '--listener_port', '8080',
              '--service_control_enable_api_key_uid_reporting',
              '--service_json_path', '/tmp/service_config.json',
              '--disable_tracing',
              '--compute_platform_override', 'Cloud Run(ESPv2)'
              ]),
            # Single header flag: --add_request_header
            (['--rollout_strategy=fixed',
              '--service_json_path=/tmp/service_config.json',
              '--add_request_header=k1=v1',
              ],
             ['bin/configmanager',  '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--add_request_headers', 'k1=v1',
              '--service_control_enable_api_key_uid_reporting',
              '--service_json_path', '/tmp/service_config.json',
              ]),
            # Double header flags: --add_request_header
            (['--rollout_strategy=fixed',
              '--service_json_path=/tmp/service_config.json',
              '--add_request_header=k1=v1',
              '--add_request_header=k2=v2',
              ],
             ['bin/configmanager',  '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--add_request_headers', 'k1=v1;k2=v2',
              '--service_control_enable_api_key_uid_reporting',
              '--service_json_path', '/tmp/service_config.json',
              ]),
            # Single header flag: --append_request_header
            (['--rollout_strategy=fixed',
              '--service_json_path=/tmp/service_config.json',
              '--append_request_header=k1=v1',
              ],
             ['bin/configmanager',  '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--append_request_headers', 'k1=v1',
              '--service_control_enable_api_key_uid_reporting',
              '--service_json_path', '/tmp/service_config.json',
              ]),
            # Double header flags: --append_request_header
            (['--rollout_strategy=fixed',
              '--service_json_path=/tmp/service_config.json',
              '--append_request_header=k1=v1',
              '--append_request_header=k2=v2',
              ],
             ['bin/configmanager',  '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--append_request_headers', 'k1=v1;k2=v2',
              '--service_control_enable_api_key_uid_reporting',
              '--service_json_path', '/tmp/service_config.json',
              ]),
            # Single header flag: --add_response_header
            (['--rollout_strategy=fixed',
              '--service_json_path=/tmp/service_config.json',
              '--add_response_header=k1=v1',
              ],
             ['bin/configmanager',  '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--add_response_headers', 'k1=v1',
              '--service_control_enable_api_key_uid_reporting',
              '--service_json_path', '/tmp/service_config.json',
              ]),
            # Double header flags: --add_response_header
            (['--rollout_strategy=fixed',
              '--service_json_path=/tmp/service_config.json',
              '--add_response_header=k1=v1',
              '--add_response_header=k2=v2',
              ],
             ['bin/configmanager',  '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--add_response_headers', 'k1=v1;k2=v2',
              '--service_control_enable_api_key_uid_reporting',
              '--service_json_path', '/tmp/service_config.json',
              ]),
            # Single header flag: --append_response_header
            (['--rollout_strategy=fixed',
              '--service_json_path=/tmp/service_config.json',
              '--append_response_header=k1=v1',
              ],
             ['bin/configmanager',  '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--append_response_headers', 'k1=v1',
              '--service_control_enable_api_key_uid_reporting',
              '--service_json_path', '/tmp/service_config.json',
              ]),
            # Double header flags: --append_response_header
            (['--rollout_strategy=fixed',
              '--service_json_path=/tmp/service_config.json',
              '--append_response_header=k1=v1',
              '--append_response_header=k2=v2',
              ],
             ['bin/configmanager',  '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--append_response_headers', 'k1=v1;k2=v2',
              '--service_control_enable_api_key_uid_reporting',
              '--service_json_path', '/tmp/service_config.json',
              ]),
            # Path security options.
            (['--rollout_strategy=fixed',
              '--service_json_path=/tmp/service_config.json',
              '--disable_normalize_path',
              '--disable_merge_slashes_in_path',
              ],
             ['bin/configmanager',  '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--service_control_enable_api_key_uid_reporting',
              '--service_json_path', '/tmp/service_config.json',
              '--normalize_path=false',
              '--merge_slashes_in_path=false',
              ]),
            (['--rollout_strategy=fixed',
              '--service_json_path=/tmp/service_config.json',
              '--disallow_escaped_slashes_in_path'
              ],
             ['bin/configmanager',  '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--service_control_enable_api_key_uid_reporting',
              '--service_json_path', '/tmp/service_config.json',
              '--disallow_escaped_slashes_in_path',
              ]),
            # Operation name header.
            (['--rollout_strategy=fixed',
              '--service_json_path=/tmp/service_config.json',
              '--enable_operation_name_header'
              ],
             ['bin/configmanager',  '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--enable_operation_name_header',
              '--service_control_enable_api_key_uid_reporting',
              '--service_json_path', '/tmp/service_config.json',
              ]),
            # response_compression.
            (['--rollout_strategy=fixed',
              '--service_json_path=/tmp/service_config.json',
              '--enable_response_compression'
              ],
             ['bin/configmanager',  '--logtostderr', '--rollout_strategy', 'fixed',
              '--backend_address', 'http://127.0.0.1:8082', '--v', '0',
              '--enable_response_compression',
              '--service_control_enable_api_key_uid_reporting',
              '--service_json_path', '/tmp/service_config.json',
              ]),
            # passing the flag --health_check_grp_backend
            (['--service=test_bookstore.gloud.run',
              '--backend=grpc://127.0.0.1:8000',
              '--health_check_grpc_backend',
              '--version=2019-11-09r0',
              ],
             ['bin/configmanager', '--logtostderr',
              '--rollout_strategy', 'fixed',
              '--backend_address', 'grpc://127.0.0.1:8000',
              '--health_check_grpc_backend',
              '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              ]),
            # passing the flag --ads_named_pipe
            (['--service=test_bookstore.gloud.run',
              '--backend=grpc://127.0.0.1:8000',
              '--ads_named_pipe=@espv2-named-pipe-9',
              '--version=2019-11-09r0',
              ],
             ['bin/configmanager', '--logtostderr',
              '--rollout_strategy', 'fixed',
              '--backend_address', 'grpc://127.0.0.1:8000',
              '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              '--ads_named_pipe', '@espv2-named-pipe-9'
              ]),
            # passing the flags: --health_check_grp_backend, --health_check_grp_backend_interval and --health_check_grp_backend_service
            (['--service=test_bookstore.gloud.run',
              '--backend=grpcs://127.0.0.1:8000',
              '--health_check_grpc_backend',
              '--health_check_grpc_backend_interval=3s',
              '--health_check_grpc_backend_service=/foo.bar',
              '--health_check_grpc_backend_no_traffic_interval=5s',
              '--version=2019-11-09r0',
              ],
             ['bin/configmanager', '--logtostderr',
              '--rollout_strategy', 'fixed',
              '--backend_address', 'grpcs://127.0.0.1:8000',
              '--health_check_grpc_backend',
              '--health_check_grpc_backend_service', '/foo.bar',
              '--health_check_grpc_backend_interval', '3s',
              '--health_check_grpc_backend_no_traffic_interval', '5s',
              '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_config_id', '2019-11-09r0',
              '--service_control_enable_api_key_uid_reporting',
              ]),
        ]

        i = 0
        for flags, wantedArgs in testcases:
            print("==== checking flags [{}]".format(', '.join(flags)))
            # Reset the environment variable as start_proxy.py may set it.
            os.environ.pop(GOOGLE_CREDS_KEY, None)
            gotArgs = gen_proxy_config(self.parser.parse_args(flags))
            self.assertEqual(gotArgs, wantedArgs,
                             msg="Fail for input #{}: \ngot  {} \nwant {}".format(i, gotArgs, wantedArgs))
            i += 1

    def test_gen_proxy_config_error(self):
        testcases = [
            ['--unknown_flag'],
            ['--rollout_strategy=managed','--v=2019-11-09r0'],
            ['--rollout_strategy=fixed'],
            ['--service=test_bookstore.gloud.run',
             '--service_json_path=/tmp/service.json'],
            ['--version=2019-11-09r0',
             '--service_json_path=/tmp/service.json'],
            ['--rollout_strategy=managed',
             '--service_json_path=/tmp/service.json'],
            ['--version=2019-11-09r0',
             '--backend_dns_lookup_family=v4'],
            ['--version=2019-11-09r0',
             '--non_gcp'],
            # Duplicate port flags.
            ['--version=2019-11-09r0', '--http_port=8000', '--http2_port=8000'],
            ['--version=2019-11-09r0', '--http_port=8000', '--listener_port=8000'],
            ['--version=2019-11-09r0', '--listener_port=8000', '--ssl_port=9000'],
            # Privileged ports.
            ['--version=2019-11-09r0', '--listener_port=80'],
            ['--version=2019-11-09r0', '--http_port=80'],
            ['--version=2019-11-09r0', '--http2_port=80'],
            ['--version=2019-11-09r0', '--ssl_port=443'],
            # SSL config.
            ['--version=2019-11-09r0', '--ssl_server_cert_path=/etc/endpoint/ssl', '--ssl_port=9000'],
            ['--version=2019-11-09r0', '--ssl_server_cert_path=/etc/endpoint/ssl', '--generate_self_signed_cert'],
            ['--version=2019-11-09r0', '--ssl_backend_client_cert_path=/etc/endpoint/ssl', '--tls_mutual_auth'],
            ['--version=2019-11-09r0', '--ssl_client_cert_path=/etc/endpoint/ssl', '--tls_mutual_auth'],
            ['--version=2019-11-09r0', '--ssl_protocols=TLSv1.3',  '--ssl_minimum_protocol=TLSv1.1'],
            ['--version=2019-11-09r0', '--ssl_minimum_protocol=TLSv11'],
            ['--version=2019-11-09r0', '--ssl_backend_client_root_certs_file', '--enable_grpc_backend_ssl'],
            ['--version=2019-11-09r0', '--ssl_client_root_certs_file', '--enable_grpc_backend_ssl'],
            ['--version=2019-11-09r0',
             '--transcoding_ignore_query_parameters=foo,bar',
             '--transcoding_ignore_unknown_query_parameters'],
            ['--version=2019-11-09r0', '--access_log_format'],
            ['--version=2019-11-09r0', '--dns=127.0.0.1', '--dns_resolver_address=127.0.0.1'],
            ['--version=2019-11-09r0', '--ssl_client_cert_path=/tmp', '--ssl_backend_client_cert_path=/tmp'],
            # The flag --backend default is using http, but the flag --health_check_grpc_backend requires grpc
            ['--version=2019-11-09r0', '--health_check_grpc_backend'],
            # The flag --backend flag is using http, but the flag --health_check_grpc_backend requires grpc
            ['--version=2019-11-09r0', '--health_check_grpc_backend', '--backend=http://abc.com'],
            # The flag --health_check_grpc_backend_interval requires the flag --health_check_grpc_backend
            ['--version=2019-11-09r0', '--health_check_grpc_backend_interval=3s'],
            # The flag --health_check_grpc_backend_service requires the flag --health_check_grpc_backend
            ['--version=2019-11-09r0', '--health_check_grpc_backend_service=/foo.bar'],
            # The flag --health_check_grpc_backend_no_traffic_interval requires the flag --health_check_grpc_backend
            ['--version=2019-11-09r0', '--health_check_grpc_backend_no_traffic_interval=1s'],
            ['--version=2019-11-09r0', '--ssl_client_root_certs_file=/tmp/server.crt', '--ssl_backend_client_root_certs_file=/tmp/server.crt'],
            # The flag --service_account_key and --enable_application_default_credentials cannot be supplied at the same time.
            ['--non_gcp', '--service_account_key=tmp/service_account_key', '--enable_application_default_credentials'],
            # The flag --non_gcp is set without --service_account_key or --enable_application_default_credentials.
            ['--non_gcp']
          ]

        for flags in testcases:
          with self.assertRaises(SystemExit) as cm:
            gen_proxy_config(self.parser.parse_args(flags))
          print(cm.exception)
          self.assertEqual(cm.exception.code, 1)

    def test_gen_envoy_args(self):
      testcases = [
          # Default
          (
              [],
              ["bin/envoy", "-c", "/tmp/bootstrap.json",
               "--disable-hot-restart",
               "--log-format %L%m%d %T.%e %t %@] [%t][%n]%v",
               "--log-format-escaped"]
          ),
          # Debug mode enabled
          (
              ["--enable_debug"],
              ["bin/envoy", "-c", "/tmp/bootstrap.json",
               "--disable-hot-restart",
               "--log-format %L%m%d %T.%e %t %@] [%t][%n]%v",
               "--log-format-escaped",
               "-l debug",
               "--component-log-level upstream:info,main:info"]
          ),
          # Added config-yaml flag
          (
              ["--envoy_extra_config_yaml=node: {id: 'node1'}"],
              ["bin/envoy", "-c", "/tmp/bootstrap.json",
               "--disable-hot-restart",
               "--log-format %L%m%d %T.%e %t %@] [%t][%n]%v",
               "--log-format-escaped",
               "--config-yaml node: {id: 'node1'}"]
          )
      ]

      for flags, wantedArgs in testcases:
        gotArgs = gen_envoy_args(self.parser.parse_args(flags))
        self.assertEqual(gotArgs, wantedArgs)

    def test_service_account_key_with_env(self):
        testcases = [
            (
                # Input: not environment variable, nor --service_account_key
                # Output: not environment variable, nor --service_account_key
                None,
                None,
                ['--service=test_bookstore.gloud.run',
                 '--backend=http://127.0.0.1',
                 '--version=2019-11-09r0',
                 ],
                ['bin/configmanager', '--logtostderr',
                 '--rollout_strategy', 'fixed',
                 '--backend_address', 'http://127.0.0.1',
                 '--v', '0',
                 '--service', 'test_bookstore.gloud.run',
                 '--service_config_id', '2019-11-09r0',
                 '--service_control_enable_api_key_uid_reporting',
                 ],
            ),
            (
                # Input: environment variable is set, but nor --service_account_key.
                # Output: both environment variable and the flag --service_account_key are set.
                "/tmp/service_account_key",
                "/tmp/service_account_key",
                ['--service=test_bookstore.gloud.run',
                 '--backend=http://127.0.0.1',
                 '--version=2019-11-09r0',
                 ],
                ['bin/configmanager', '--logtostderr',
                 '--rollout_strategy', 'fixed',
                 '--backend_address', 'http://127.0.0.1',
                 '--v', '0',
                 '--service', 'test_bookstore.gloud.run',
                 '--service_config_id', '2019-11-09r0',
                 '--service_control_enable_api_key_uid_reporting',
                 '--service_account_key', '/tmp/service_account_key',
                 ],
            ),
            (
                # Input: environment variable is not set, but the flag --service_account_key is set.
                # Output: both environment variable and the flag --service_account_key are set.
                None,
                "/tmp/service_account_key",
                ['--service=test_bookstore.gloud.run',
                 '--backend=http://127.0.0.1',
                 '--version=2019-11-09r0',
                 '--service_account_key', '/tmp/service_account_key',
                 ],
                ['bin/configmanager', '--logtostderr',
                 '--rollout_strategy', 'fixed',
                 '--backend_address', 'http://127.0.0.1',
                 '--v', '0',
                 '--service', 'test_bookstore.gloud.run',
                 '--service_config_id', '2019-11-09r0',
                 '--service_control_enable_api_key_uid_reporting',
                 '--service_account_key', '/tmp/service_account_key',
                 ],
            ),
            (
                # Input: environment variable and the flag --service_account_key are set, but with different values
                # Output: the environment variable is not changed, and the flag --service_account_key is set.
                "/tmp/service_account_key111",
                "/tmp/service_account_key111",
                ['--service=test_bookstore.gloud.run',
                 '--backend=http://127.0.0.1',
                 '--version=2019-11-09r0',
                 '--service_account_key', '/tmp/service_account_key222',
                 ],
                ['bin/configmanager', '--logtostderr',
                 '--rollout_strategy', 'fixed',
                 '--backend_address', 'http://127.0.0.1',
                 '--v', '0',
                 '--service', 'test_bookstore.gloud.run',
                 '--service_config_id', '2019-11-09r0',
                 '--service_control_enable_api_key_uid_reporting',
                 '--service_account_key', '/tmp/service_account_key222',
                 ],
            ),
        ]
        for oldEnv, wantedEnv, flags, wantedArgs in testcases:
            if oldEnv:
                os.environ[GOOGLE_CREDS_KEY] = oldEnv
            else:
                os.environ.pop(GOOGLE_CREDS_KEY, None)
            gotArgs = gen_proxy_config(self.parser.parse_args(flags))
            self.assertEqual(gotArgs, wantedArgs,
                             msg="Fail with diff: got  {} \nwant {}".format(gotArgs, wantedArgs))
            if wantedEnv:
                self.assertEqual(os.environ[GOOGLE_CREDS_KEY], wantedEnv)
            else:
                self.assertFalse(GOOGLE_CREDS_KEY in os.environ)

    def test_fetch_project_id_from_metadata_success(self):
        with mock.patch("urllib.request.urlopen") as mock_urlopen:
            mock_resp = mock.MagicMock()
            mock_resp.read.return_value = b"my-cloud-project\n"
            mock_resp.__enter__.return_value = mock_resp
            mock_urlopen.return_value = mock_resp

            project_id = fetch_project_id_from_metadata()
            self.assertEqual(project_id, "my-cloud-project")

    def test_fetch_project_id_from_metadata_failure(self):
        with mock.patch("urllib.request.urlopen") as mock_urlopen:
            mock_urlopen.side_effect = Exception("network unreachable")

            project_id = fetch_project_id_from_metadata()
            self.assertIsNone(project_id)

    def test_fetch_project_id_from_service_account_key_valid(self):
        with tempfile.NamedTemporaryFile("w", delete=False) as f:
            json.dump({"project_id": "test-sa-proj-123", "client_email": "sa@example.com"}, f)
            path = f.name
        try:
            self.assertEqual(fetch_project_id_from_service_account_key(path), "test-sa-proj-123")
        finally:
            os.remove(path)

    def test_fetch_project_id_from_service_account_key_missing_file(self):
        self.assertIsNone(fetch_project_id_from_service_account_key("/nonexistent/path/sa.json"))

    def test_fetch_project_id_from_service_account_key_corrupted_json(self):
        with tempfile.NamedTemporaryFile("w", delete=False) as f:
            f.write("not valid json {[[")
            path = f.name
        try:
            self.assertIsNone(fetch_project_id_from_service_account_key(path))
        finally:
            os.remove(path)

    def test_fetch_project_id_from_service_account_key_missing_project_id(self):
        with tempfile.NamedTemporaryFile("w", delete=False) as f:
            json.dump({"client_email": "sa@example.com"}, f)
            path = f.name
        try:
            self.assertIsNone(fetch_project_id_from_service_account_key(path))
        finally:
            os.remove(path)

    def test_fetch_project_id_from_service_account_key_none_or_empty(self):
        self.assertIsNone(fetch_project_id_from_service_account_key(None))
        self.assertIsNone(fetch_project_id_from_service_account_key(""))

    def test_setup_otel_resource_attributes(self):
        with tempfile.NamedTemporaryFile("w", delete=False) as f:
            json.dump({"project_id": "sa-project-789"}, f)
            sa_key_path = f.name

        try:
            testcases = [
                (
                    "Injects project ID from --tracing_project_id when env is empty",
                    ["--service=echo", "--version=v1", "--backend=127.0.0.1:8080",
                     "--tracing_project_id=flag-project-123"],
                    None,  # initial env
                    None,  # metadata return
                    "gcp.project_id=flag-project-123",  # expected env
                ),
                (
                    "Appends project ID to existing OTEL_RESOURCE_ATTRIBUTES",
                    ["--service=echo", "--version=v1", "--backend=127.0.0.1:8080",
                     "--tracing_project_id=flag-project-123"],
                    "service.name=my-svc,service.version=1.0",
                    None,
                    "service.name=my-svc,service.version=1.0,gcp.project_id=flag-project-123",
                ),
                (
                    "Does not duplicate if gcp.project_id already present in env",
                    ["--service=echo", "--version=v1", "--backend=127.0.0.1:8080",
                     "--tracing_project_id=flag-project-123"],
                    "gcp.project_id=existing-project",
                    None,
                    "gcp.project_id=existing-project",
                ),
                (
                    "Does not duplicate if legacy gcp.project.id already present in env",
                    ["--service=echo", "--version=v1", "--backend=127.0.0.1:8080",
                     "--tracing_project_id=flag-project-123"],
                    "gcp.project.id=existing-project,env=prod",
                    None,
                    "gcp.project.id=existing-project,env=prod",
                ),
                (
                    "Auto-detects from metadata server when flag is omitted on GCP",
                    ["--service=echo", "--version=v1", "--backend=127.0.0.1:8080"],
                    None,
                    "metadata-project-456",
                    "gcp.project_id=metadata-project-456",
                ),
                (
                    "Skips metadata fetch when --non_gcp is set",
                    ["--service=echo", "--version=v1", "--backend=127.0.0.1:8080", "--non_gcp"],
                    None,
                    "metadata-project-456",
                    None,  # should remain unset
                ),
                (
                    "Skips when --disable_tracing is set",
                    ["--service=echo", "--version=v1", "--backend=127.0.0.1:8080",
                     "--disable_tracing", "--tracing_project_id=flag-project-123"],
                    None,
                    "metadata-project-456",
                    None,
                ),
                (
                    "Leaves env unset when metadata fetch fails and no flag provided",
                    ["--service=echo", "--version=v1", "--backend=127.0.0.1:8080"],
                    None,
                    None,  # metadata returns None
                    None,
                ),
                (
                    "Resolves project ID from --service_account_key when flag is omitted",
                    ["--service=echo", "--version=v1", "--backend=127.0.0.1:8080",
                     f"--service_account_key={sa_key_path}"],
                    None,
                    None,
                    "gcp.project_id=sa-project-789",
                ),
                (
                    "--tracing_project_id flag takes precedence over --service_account_key",
                    ["--service=echo", "--version=v1", "--backend=127.0.0.1:8080",
                     "--tracing_project_id=flag-project-123",
                     f"--service_account_key={sa_key_path}"],
                    None,
                    None,
                    "gcp.project_id=flag-project-123",
                ),
                (
                    "OTEL_RESOURCE_ATTRIBUTES takes precedence over --service_account_key",
                    ["--service=echo", "--version=v1", "--backend=127.0.0.1:8080",
                     f"--service_account_key={sa_key_path}"],
                    "gcp.project_id=existing-env-project",
                    None,
                    "gcp.project_id=existing-env-project",
                ),
                (
                    "Falls back to metadata server when --service_account_key is invalid on GCP",
                    ["--service=echo", "--version=v1", "--backend=127.0.0.1:8080",
                     "--service_account_key=/invalid/path/key.json"],
                    None,
                    "metadata-project-456",
                    "gcp.project_id=metadata-project-456",
                ),
                (
                    "Resolves project ID from --service_account_key even on --non_gcp",
                    ["--service=echo", "--version=v1", "--backend=127.0.0.1:8080",
                     "--non_gcp", f"--service_account_key={sa_key_path}"],
                    None,
                    None,
                    "gcp.project_id=sa-project-789",
                ),
                (
                    "Leaves env unset when --service_account_key is invalid on --non_gcp",
                    ["--service=echo", "--version=v1", "--backend=127.0.0.1:8080",
                     "--non_gcp", "--service_account_key=/invalid/path/key.json"],
                    None,
                    "metadata-should-not-be-called",
                    None,
                ),
            ]

            for desc, flags, initEnv, metadataReturn, expectedEnv in testcases:
                with self.subTest(desc):
                    orig_env = os.environ.get("OTEL_RESOURCE_ATTRIBUTES")
                    try:
                        if initEnv is not None:
                            os.environ["OTEL_RESOURCE_ATTRIBUTES"] = initEnv
                        elif "OTEL_RESOURCE_ATTRIBUTES" in os.environ:
                            del os.environ["OTEL_RESOURCE_ATTRIBUTES"]

                        args = self.parser.parse_args(flags)
                        with mock.patch("start_proxy.fetch_project_id_from_metadata", return_value=metadataReturn):
                            setup_otel_resource_attributes(args)

                        actualEnv = os.environ.get("OTEL_RESOURCE_ATTRIBUTES")
                        self.assertEqual(actualEnv, expectedEnv,
                                         f"{desc}: expected {expectedEnv}, got {actualEnv}")
                    finally:
                        if orig_env is not None:
                            os.environ["OTEL_RESOURCE_ATTRIBUTES"] = orig_env
                        elif "OTEL_RESOURCE_ATTRIBUTES" in os.environ:
                            del os.environ["OTEL_RESOURCE_ATTRIBUTES"]
        finally:
            os.remove(sa_key_path)

    def test_gen_proxy_config_non_gcp_with_service_account_key_keeps_tracing(self):
        with tempfile.NamedTemporaryFile("w", delete=False) as f:
            json.dump({"project_id": "sa-test-proj-789"}, f)
            sa_path = f.name
        try:
            flags = [
                "--service=test_bookstore.gloud.run",
                "--backend=http://127.0.0.1",
                "--version=2019-11-09r0",
                "--non_gcp",
                f"--service_account_key={sa_path}",
            ]
            args = self.parser.parse_args(flags)
            gotArgs = gen_proxy_config(args)
            self.assertNotIn("--disable_tracing", gotArgs)
            self.assertIn("--tracing_project_id", gotArgs)
            idx = gotArgs.index("--tracing_project_id")
            self.assertEqual(gotArgs[idx + 1], "sa-test-proj-789")
        finally:
            os.remove(sa_path)


if __name__ == '__main__':
    unittest.main()

