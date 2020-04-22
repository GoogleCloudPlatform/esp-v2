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

import unittest
import sys

import os, inspect

currentdir = os.path.dirname(
    os.path.abspath(inspect.getfile(inspect.currentframe())))
sys.path.insert(0, currentdir + "/../../docker/generic")
from start_proxy import gen_bootstrap_conf, make_argparser, gen_proxy_config, gen_envoy_args


class TestStartProxy(unittest.TestCase):

    def setUp(self):
        self.parser = make_argparser()

    def test_gen_bootstrap(self):
        testcases = [
            (["--http_request_timeout_s=1", "--disable_tracing", "--admin_port=8001"],
             ['bin/bootstrap', '--logtostderr', '--admin_port', '8001',
              '--disable_tracing',
              '--http_request_timeout_s', '1',
              '/tmp/bootstrap.json']),
            ([], ['bin/bootstrap',
                  '--logtostderr', '--admin_port', '0',
                  '--tracing_sample_rate', '0.001',
                  '/tmp/bootstrap.json']),
            (['--service_account_key', '/tmp/service_accout_key', "-N=8001",
              '--tracing_project_id=test_project_1234',
              '--cloud_trace_url_override=localhost:9990'],
             ['bin/bootstrap', '--logtostderr', '--admin_port', '8001',
              '--tracing_project_id', 'test_project_1234',
              '--tracing_stackdriver_address', 'localhost:9990',
              '--tracing_sample_rate', '0.001',
              '/tmp/bootstrap.json']),
            (['--tracing_project_id=123',
              '--tracing_sample_rate=1', "--status_port=8001",
              '--tracing_incoming_context=fake-incoming-context',
              '--tracing_outgoing_context=fake-outgoing-context'],
             ['bin/bootstrap',
              '--logtostderr', '--admin_port', '8001',
              '--tracing_project_id',
              '123', '--tracing_incoming_context',
              'fake-incoming-context', '--tracing_outgoing_context',
              'fake-outgoing-context',
              '--tracing_sample_rate', '1',
              '/tmp/bootstrap.json']),
            # --disable_cloud_trace_auto_sampling overrides --tracing_sample_rate
            (['--disable_cloud_trace_auto_sampling',
              '--tracing_sample_rate=0.5'],
             ['bin/bootstrap',
              '--logtostderr', '--admin_port', '0',
              '--tracing_sample_rate', '0',
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
             ['bin/configmanager', '--logtostderr', '--backend_address', 'grpc://127.0.0.1:8000',
              '--rollout_strategy', 'fixed',  '--healthz', '/',
              '--v', '0', '--log_jwt_payloads', 'aud,exp',
              '--service', 'test_bookstore.gloud.run',
              '--http_request_timeout_s', '10',
              '--service_config_id', '2019-11-09r0',
              '--disable_tracing',
              ]),
            # backend with DNS address, no version.
            (['--service=echo.gloud.run', '--backend=http://echo:8080',
              '--log_request_headers=x-google-x',
              '--service_control_check_timeout_ms=100', '-z=hc',
              '--backend_dns_lookup_family=v4only', '--disable_tracing'],
             ['bin/configmanager', '--logtostderr',
              '--backend_address', 'http://echo:8080',
              '--rollout_strategy', 'fixed', '--healthz', 'hc', '--v', '0',
              '--log_request_headers', 'x-google-x',
              '--service', 'echo.gloud.run',
              '--service_control_check_timeout_ms', '100',
              '--disable_tracing',
              '--backend_dns_lookup_family', 'v4only'
              ]),
            # Default backend
            (['-R=managed','--enable_strict_transport_security',
              '--http_port=8079', '--service_control_quota_retries=3',
              '--service_control_report_timeout_ms=300',
              '--service_control_network_fail_open', '--check_metadata',
              '--disable_tracing', '--underscores_in_headers'],
             ['bin/configmanager', '--logtostderr','--backend_address', 'http://127.0.0.1:8082',
              '--rollout_strategy', 'managed', '--v', '0',
              '--listener_port', '8079', '--enable_strict_transport_security',
              '--service_control_quota_retries', '3',
              '--service_control_report_timeout_ms', '300',
              '--check_metadata', '--underscores_in_headers',
              '--disable_tracing'
              ]),
            # ssl_server_cert_path specified
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--ssl_server_cert_path=/etc/endpoint/ssl'],
             ['bin/configmanager', '--logtostderr',
              '--backend_address', 'http://127.0.0.1:8082',
              '--rollout_strategy', 'managed', '--v', '0',
              '--listener_port', '8080', '--ssl_server_cert_path',
              '/etc/endpoint/ssl', '--disable_tracing'
              ]),
            # legacy ssl_port specified
            (['-R=managed','--ssl_port=443'],
             ['bin/configmanager', '--logtostderr',
              '--backend_address', 'http://127.0.0.1:8082',
              '--rollout_strategy', 'managed', '--v', '0',
              '--ssl_server_cert_path', '/etc/nginx/ssl',
              '--listener_port', '443',
              ]),
            # ssl_client_cert_path specified
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--ssl_client_cert_path=/etc/endpoint/ssl'],
             ['bin/configmanager', '--logtostderr',
              '--backend_address', 'http://127.0.0.1:8082',
              '--rollout_strategy', 'managed', '--v', '0',
              '--listener_port', '8080', '--ssl_client_cert_path',
              '/etc/endpoint/ssl', '--disable_tracing'
              ]),
            # ssl_client_root_certs_file specified
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--ssl_client_root_certs_file=/etc/endpoints/ssl/ca-certificates.crt' ],
             ['bin/configmanager', '--logtostderr',
              '--backend_address', 'http://127.0.0.1:8082',
              '--rollout_strategy', 'managed', '--v', '0',
              '--listener_port', '8080', '--root_certs_path',
              '/etc/endpoints/ssl/ca-certificates.crt', '--disable_tracing'
              ]), 
            # legacy enable_grpc_backend_ssl specified
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--enable_grpc_backend_ssl' ],
             ['bin/configmanager', '--logtostderr',
              '--backend_address', 'http://127.0.0.1:8082',
              '--rollout_strategy', 'managed', '--v', '0',
              '--listener_port', '8080', '--root_certs_path',
              '/etc/nginx/trusted-ca-certificates.crt', '--disable_tracing'
              ]),
            # legacy tls_mutual_auth specified
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--tls_mutual_auth'],
             ['bin/configmanager', '--logtostderr',
              '--backend_address', 'http://127.0.0.1:8082',
              '--rollout_strategy', 'managed', '--v', '0',
              '--listener_port', '8080', '--ssl_client_cert_path',
              '/etc/nginx/ssl', '--disable_tracing'
              ]),
            # ssl_minimum_protocol and ssl_maximum_protocol specified
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--ssl_minimum_protocol=TLSv1.1',
              '--ssl_maximum_protocol=TLSv1.3'],
             ['bin/configmanager', '--logtostderr',
              '--backend_address', 'http://127.0.0.1:8082',
              '--rollout_strategy', 'managed', '--v', '0',
              '--listener_port', '8080', '--ssl_minimum_protocol',
              'TLSv1.1','--ssl_maximum_protocol','TLSv1.3', '--disable_tracing'
              ]),
            # legacy --ssl_protocols specified
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--ssl_protocols=TLSv1.3', '--ssl_protocols=TLSv1.2'],
             ['bin/configmanager', '--logtostderr',
              '--backend_address', 'http://127.0.0.1:8082',
              '--rollout_strategy', 'managed', '--v', '0',
              '--listener_port', '8080', '--ssl_minimum_protocol',
              'TLSv1.2','--ssl_maximum_protocol','TLSv1.3', '--disable_tracing'
              ]),
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--ssl_protocols=TLSv1.2'],
             ['bin/configmanager', '--logtostderr',
              '--backend_address', 'http://127.0.0.1:8082',
              '--rollout_strategy', 'managed', '--v', '0',
              '--listener_port', '8080', '--ssl_minimum_protocol',
              'TLSv1.2','--ssl_maximum_protocol','TLSv1.2', '--disable_tracing'
              ]),
            (['-R=managed','--listener_port=8080',  '--disable_tracing',
              '--generate_self_signed_cert'],
             ['bin/configmanager', '--logtostderr',
              '--backend_address', 'http://127.0.0.1:8082',
              '--rollout_strategy', 'managed', '--v', '0',
              '--listener_port', '8080', '--ssl_server_cert_path',
              '/tmp/ssl/endpoints', '--disable_tracing'
              ]),
            # http2_port specified.
            (['-R=managed',
              '--http2_port=8079', '--service_control_quota_retries=3',
              '--service_control_report_timeout_ms=300',
              '--service_control_network_fail_open', '--check_metadata',
              '--disable_tracing'],
             ['bin/configmanager', '--logtostderr',
              '--backend_address', 'http://127.0.0.1:8082',
              '--rollout_strategy', 'managed', '--v', '0',
              '--listener_port', '8079',
              '--service_control_quota_retries', '3',
              '--service_control_report_timeout_ms', '300',
              '--check_metadata',
              '--disable_tracing'
              ]),
            # listener_port specified.
            (['-R=managed',
              '--listener_port=8079', '--service_control_quota_retries=3',
              '--service_control_report_timeout_ms=300',
              '--service_control_network_fail_open', '--check_metadata',
              '--disable_tracing'],
             ['bin/configmanager', '--logtostderr',
              '--backend_address', 'http://127.0.0.1:8082',
              '--rollout_strategy', 'managed', '--v', '0',
              '--listener_port', '8079',
              '--service_control_quota_retries', '3',
              '--service_control_report_timeout_ms', '300',
              '--check_metadata',
              '--disable_tracing'
              ]),
            # with service account key
            (['--service=test_bookstore.gloud.run',
              '--backend=http://127.0.0.1',
              '--service_account_key', '/tmp/service_accout_key',
              '--tracing_project_id=test_project_1234'],
             ['bin/configmanager', '--logtostderr','--backend_address', 'http://127.0.0.1',
              '--rollout_strategy', 'fixed', '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--service_account_key', '/tmp/service_accout_key', '--non_gcp',
              ]),
            # Cors
            (['--service=test_bookstore.gloud.run',
              '--backend=https://127.0.0.1', '--cors_preset=basic',
              '--cors_allow_headers=X-Requested-With', '--non_gcp',
              '--service_account_key', '/tmp/service_accout_key'],
             ['bin/configmanager', '--logtostderr', '--backend_address', 'https://127.0.0.1',
              '--rollout_strategy', 'fixed', '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--disable_tracing',
              '--cors_preset', 'basic',
              '--cors_allow_origin', '*', '--cors_allow_origin_regex', '',
              '--cors_allow_methods', 'GET, POST, PUT, PATCH, DELETE, OPTIONS',
              '--cors_allow_headers', 'X-Requested-With',
              '--cors_expose_headers', 'Content-Length,Content-Range',
              '--service_account_key', '/tmp/service_accout_key', '--non_gcp',
              ]),
            # backend routing (with deprecated flag)
            (['--backend=https://127.0.0.1:8000', '--enable_backend_routing',
              '--service_json_path=/tmp/service.json',
              '--compute_platform_override', 'Cloud Run(ESPv2)',
              '--disable_tracing'],
             ['bin/configmanager',  '--logtostderr','--backend_address', 'https://127.0.0.1:8000',
              '--rollout_strategy', 'fixed', '--v', '0',
              '--service_json_path', '/tmp/service.json',
              '--disable_tracing',
              '--compute_platform_override', 'Cloud Run(ESPv2)'
              ]),
            # grpc backend with fixed version and tracing
            (['--service=test_bookstore.gloud.run', '--version=2019-11-09r0',
              '--backend=grpc://127.0.0.1:8000', '--http_request_timeout_s=10',
              '--log_jwt_payloads=aud,exp'],
             ['bin/configmanager', '--logtostderr','--backend_address', 'grpc://127.0.0.1:8000',
              '--rollout_strategy', 'fixed', '--v', '0','--log_jwt_payloads', 'aud,exp',
              '--service', 'test_bookstore.gloud.run',
              '--http_request_timeout_s', '10',
              '--service_config_id', '2019-11-09r0',
              ]),
            # json-grpc transcoder json print options
            (['--service=test_bookstore.gloud.run',
              '--backend=grpc://127.0.0.1:8000',
              '--transcoding_always_print_primitive_fields',
              '--transcoding_preserve_proto_field_names',
              '--transcoding_always_print_enums_as_ints'
              ],
             ['bin/configmanager', '--logtostderr','--backend_address', 'grpc://127.0.0.1:8000',
              '--rollout_strategy', 'fixed', '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--transcoding_always_print_primitive_fields',
              '--transcoding_always_print_enums_as_ints',
              '--transcoding_preserve_proto_field_names',
              ]),
            # json-grpc transcoder ignore unknown parameters
            (['--service=test_bookstore.gloud.run',
              '--backend=grpc://127.0.0.1:8000',
              '--transcoding_ignore_query_parameters=foo,bar'
              ],
             ['bin/configmanager', '--logtostderr','--backend_address', 'grpc://127.0.0.1:8000',
              '--rollout_strategy', 'fixed', '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--transcoding_ignore_query_parameters', 'foo,bar'
              ]),
            (['--service=test_bookstore.gloud.run',
              '--backend=grpc://127.0.0.1:8000',
              '--transcoding_ignore_unknown_query_parameters'
              ],
             ['bin/configmanager', '--logtostderr','--backend_address', 'grpc://127.0.0.1:8000',
              '--rollout_strategy', 'fixed', '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--transcoding_ignore_unknown_query_parameters'
              ]),
            # --enable_debug
            (['--service=test_bookstore.gloud.run',
              '--backend=grpc://127.0.0.1:8000',
              '--enable_debug',
              ],
             ['bin/configmanager', '--logtostderr',
              '--backend_address', 'grpc://127.0.0.1:8000',
              '--rollout_strategy', 'fixed',
              '--v', '1',
              '--service', 'test_bookstore.gloud.run',
              '--suppress_envoy_headers=false'
              ]),
            (['--service=test_bookstore.gloud.run',
              '--backend=grpc://127.0.0.1:8000',
              '--access_log=/foo/bar',
              ],
             ['bin/configmanager', '--logtostderr',
              '--backend_address', 'grpc://127.0.0.1:8000',
              '--rollout_strategy', 'fixed',
              '--v', '0',
              '--service', 'test_bookstore.gloud.run',
              '--access_log', "/foo/bar"
              ]),
        ]

        for flags, wantedArgs in testcases:
            gotArgs = gen_proxy_config(self.parser.parse_args(flags))
            self.assertEqual(gotArgs, wantedArgs)

    def test_gen_proxy_config_error(self):
        testcases = [
            ['--unknown_flag'],
            ['--rollout_strategy=mangaed'],
            ['--rollout_strategy=managed','--v=2019-11-09r0'],
            ['--service=test_bookstore.gloud.run',
             '--service_json_path=/tmp/service.json'],
            ['--version=2019-11-09r0',
             '--service_json_path=/tmp/service.json'],
            ['--rollout_strategy=managed',
             '--service_json_path=/tmp/service.json'],
            ['--backend_dns_lookup_family=v4'],
            ['--non_gcp'],
            ['--http_port=80', '--http2_port=80'],
            ['--http_port=80', '--listener_port=80'],
            ['--ssl_server_cert_path=/etc/endpoint/ssl', '--ssl_port=443'],
            ['--ssl_server_cert_path=/etc/endpoint/ssl', '--generate_self_signed_cert'],
            ['--ssl_client_cert_path=/etc/endpoint/ssl', '--tls_mutual_auth'],
            ['--ssl_protocols=TLSv1.3',  '--ssl_minimum_protocol=TLSv1.1'],
            ['--ssl_minimum_protocol=TLSv11'],
            ['--ssl_client_root_certs_file', '--enable_grpc_backend_ssl'],
            ['--transcoding_ignore_query_parameters=foo,bar',
             '--transcoding_ignore_unknown_query_parameters']
          ]

        for flags in testcases:
          with self.assertRaises(SystemExit) as cm:
            gotArgs = gen_proxy_config(self.parser.parse_args(flags))
          print(cm.exception)
          self.assertEqual(cm.exception.code, 1)

    def test_gen_envoy_args(self):
      testcases = [
          # Default
          (
              [],
              ["bin/envoy", "-c", "/tmp/bootstrap.json",
               "--disable-hot-restart",
               "--log-format %L%m%d %T.%e %t envoy] [%t][%n]%v",
               "--log-format-escaped"]
          ),
          # Debug mode enabled
          (
              ["--enable_debug"],
              ["bin/envoy", "-c", "/tmp/bootstrap.json",
               "--disable-hot-restart",
               "--log-format %L%m%d %T.%e %t envoy] [%t][%n]%v",
               "--log-format-escaped",
               "-l debug"]
          )
      ]

      for flags, wantedArgs in testcases:
        gotArgs = gen_envoy_args(self.parser.parse_args(flags))
        self.assertEqual(gotArgs, wantedArgs)

if __name__ == '__main__':
    unittest.main()

