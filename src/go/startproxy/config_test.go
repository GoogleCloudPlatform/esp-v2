// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package startproxy

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestGenBootstrap(t *testing.T) {
	l := newLauncher()
	testcases := []struct {
		flags []string
		want  []string
	}{
		{
			flags: []string{"--http_request_timeout_s=1", "--disable_tracing", "--admin_port=8001"},
			want:  []string{"bin/bootstrap", "--logtostderr", "--admin_port", "8001", "--http_request_timeout_s", "1", "/tmp/bootstrap.json"},
		},
		{
			flags: []string{"--ads_named_pipe=@espv2-named-pipe-9", "--disable_tracing", "--admin_port=8001"},
			want:  []string{"bin/bootstrap", "--logtostderr", "--admin_port", "8001", "--ads_named_pipe", "@espv2-named-pipe-9", "/tmp/bootstrap.json"},
		},
		{
			flags: []string{},
			want:  []string{"bin/bootstrap", "--logtostderr", "--admin_port", "0", "/tmp/bootstrap.json"},
		},
	}

	for _, tc := range testcases {
		t.Run(strings.Join(tc.flags, " "), func(t *testing.T) {
			unsetEnvForTest(t, googleCredsEnv)
			args, err := parseArgs(tc.flags)
			if err != nil {
				t.Fatalf("unexpected parseArgs error: %v", err)
			}
			if got := l.genBootstrapConf(args); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got  %q\nwant %q", got, tc.want)
			}
		})
	}
}

func TestGenProxyConfig(t *testing.T) {
	tmpCertDir := t.TempDir()
	l := newLauncher()
	l.certDir = tmpCertDir

	testcases := []struct {
		flags []string
		want  []string
	}{
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--version=2019-11-09r0", "--backend=grpc://127.0.0.1:8000", "--http_request_timeout_s=10", "--log_jwt_payloads=aud,exp", "--disable_tracing", "--healthz=/"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "grpc://127.0.0.1:8000", "--healthz", "/", "--v", "0", "--log_jwt_payloads", "aud,exp", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--http_request_timeout_s", "10", "--service_control_enable_api_key_uid_reporting", "--disable_tracing"},
		},
		{
			flags: []string{"--service=echo.gloud.run", "--backend=http://echo:8080", "--log_request_headers=x-google-x", "--version=2019-11-09r0", "--service_control_check_timeout_ms=100", "-z=hc", "--backend_dns_lookup_family=v4only", "--disable_tracing", "--dns_resolver_addresses=127.0.0.1:53"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://echo:8080", "--healthz", "hc", "--v", "0", "--log_request_headers", "x-google-x", "--service", "echo.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_check_timeout_ms", "100", "--service_control_enable_api_key_uid_reporting", "--disable_tracing", "--backend_dns_lookup_family", "v4only", "--dns_resolver_addresses", "127.0.0.1:53"},
		},
		{
			flags: []string{"--service=echo.gloud.run", "--backend=http://echo:8080", "--log_request_headers=x-google-x", "--version=2019-11-09r0", "--service_control_check_timeout_ms=100", "-z=hc", "--backend_dns_lookup_family=v4only", "--disable_tracing", "--dns=127.0.0.1:53"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://echo:8080", "--healthz", "hc", "--v", "0", "--log_request_headers", "x-google-x", "--service", "echo.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_check_timeout_ms", "100", "--service_control_enable_api_key_uid_reporting", "--disable_tracing", "--backend_dns_lookup_family", "v4only", "--dns_resolver_addresses", "127.0.0.1:53"},
		},
		{
			flags: []string{"-R=managed", "--enable_strict_transport_security", "--http_port=8079", "--service_control_quota_retries=3", "--service_control_report_timeout_ms=300", "--check_metadata", "--disable_tracing", "--underscores_in_headers"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8079", "--enable_strict_transport_security", "--service_control_quota_retries", "3", "--service_control_report_timeout_ms", "300", "--service_control_enable_api_key_uid_reporting", "--check_metadata", "--underscores_in_headers", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--disable_jwks_async_fetch", "--http_port=8079", "--service_control_quota_retries=3", "--service_control_report_timeout_ms=300", "--check_metadata", "--disable_tracing", "--underscores_in_headers"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--disable_jwks_async_fetch", "--listener_port", "8079", "--service_control_quota_retries", "3", "--service_control_report_timeout_ms", "300", "--service_control_enable_api_key_uid_reporting", "--check_metadata", "--underscores_in_headers", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--jwks_async_fetch_fast_listener", "--http_port=8079", "--service_control_quota_retries=3", "--service_control_report_timeout_ms=300", "--check_metadata", "--disable_tracing", "--underscores_in_headers"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--jwks_async_fetch_fast_listener", "--listener_port", "8079", "--service_control_quota_retries", "3", "--service_control_report_timeout_ms", "300", "--service_control_enable_api_key_uid_reporting", "--check_metadata", "--underscores_in_headers", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--jwt_cache_size=300", "--http_port=8079", "--service_control_quota_retries=3", "--service_control_report_timeout_ms=300", "--check_metadata", "--disable_tracing", "--underscores_in_headers"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--jwt_cache_size", "300", "--listener_port", "8079", "--service_control_quota_retries", "3", "--service_control_report_timeout_ms", "300", "--service_control_enable_api_key_uid_reporting", "--check_metadata", "--underscores_in_headers", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--disable_jwt_audience_service_name_check", "--http_port=8079", "--service_control_quota_retries=3", "--service_control_report_timeout_ms=300", "--check_metadata", "--disable_tracing", "--underscores_in_headers"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--disable_jwt_audience_service_name_check", "--listener_port", "8079", "--service_control_quota_retries", "3", "--service_control_report_timeout_ms", "300", "--service_control_enable_api_key_uid_reporting", "--check_metadata", "--underscores_in_headers", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--disable_jwks_async_fetch", "--jwks_fetch_num_retries=10", "--jwks_fetch_retry_back_off_base_interval=100", "--jwks_fetch_retry_back_off_max_interval=32000", "--jwt_pad_forward_payload_header", "--http_port=8079", "--service_control_quota_retries=3", "--service_control_report_timeout_ms=300", "--check_metadata", "--disable_tracing", "--underscores_in_headers"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--disable_jwks_async_fetch", "--jwks_fetch_num_retries", "10", "--jwks_fetch_retry_back_off_base_interval_ms", "100", "--jwks_fetch_retry_back_off_max_interval_ms", "32000", "--jwt_pad_forward_payload_header", "--listener_port", "8079", "--service_control_quota_retries", "3", "--service_control_report_timeout_ms", "300", "--service_control_enable_api_key_uid_reporting", "--check_metadata", "--underscores_in_headers", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--enable_strict_transport_security", "--http_port=8079", "--service_control_quota_retries=3", "--service_control_report_timeout_ms=300", "--service_control_network_fail_policy=open", "--check_metadata", "--disable_tracing", "--underscores_in_headers"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8079", "--enable_strict_transport_security", "--service_control_quota_retries", "3", "--service_control_report_timeout_ms", "300", "--service_control_enable_api_key_uid_reporting", "--check_metadata", "--underscores_in_headers", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--enable_strict_transport_security", "--http_port=8079", "--service_control_quota_retries=3", "--service_control_report_timeout_ms=300", "--service_control_network_fail_policy=open", "--check_metadata", "--no-service_control_enable_api_key_uid_reporting", "--disable_tracing", "--underscores_in_headers"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8079", "--enable_strict_transport_security", "--service_control_quota_retries", "3", "--service_control_report_timeout_ms", "300", "--check_metadata", "--underscores_in_headers", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--listener_port=8080", "--disable_tracing", "--service_control_url=https://non-default-servicecontrol.googleapis.com"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8080", "--service_control_url", "https://non-default-servicecontrol.googleapis.com", "--service_control_enable_api_key_uid_reporting", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--enable_strict_transport_security", "--http_port=8079", "--service_control_quota_retries=3", "--service_control_report_timeout_ms=300", "--service_control_network_fail_policy=close", "--check_metadata", "--disable_tracing", "--underscores_in_headers"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8079", "--enable_strict_transport_security", "--service_control_quota_retries", "3", "--service_control_report_timeout_ms", "300", "--service_control_network_fail_open=false", "--service_control_enable_api_key_uid_reporting", "--check_metadata", "--underscores_in_headers", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--listener_port=8080", "--disable_tracing", "--ssl_server_cert_path=/etc/endpoint/ssl"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8080", "--ssl_server_cert_path", "/etc/endpoint/ssl", "--service_control_enable_api_key_uid_reporting", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--listener_port=8080", "--disable_tracing", "--ssl_server_root_cert_path=/etc/endpoint/ssl/root.cert"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8080", "--ssl_server_root_cert_path", "/etc/endpoint/ssl/root.cert", "--service_control_enable_api_key_uid_reporting", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--ssl_port=9000", "--disable_tracing"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--ssl_server_cert_path", "/etc/nginx/ssl", "--listener_port", "9000", "--service_control_enable_api_key_uid_reporting", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--listener_port=8080", "--disable_tracing", "--ssl_backend_client_cert_path=/etc/endpoint/ssl"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8080", "--ssl_backend_client_cert_path", "/etc/endpoint/ssl", "--service_control_enable_api_key_uid_reporting", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--listener_port=8080", "--disable_tracing", "--ssl_client_cert_path=/etc/endpoint/ssl"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8080", "--ssl_backend_client_cert_path", "/etc/endpoint/ssl", "--service_control_enable_api_key_uid_reporting", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--listener_port=8080", "--disable_tracing", "--ssl_backend_client_root_certs_file=/etc/endpoints/ssl/ca-certificates.crt"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8080", "--ssl_backend_client_root_certs_path", "/etc/endpoints/ssl/ca-certificates.crt", "--service_control_enable_api_key_uid_reporting", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--listener_port=8080", "--disable_tracing", "--ssl_client_root_certs_file=/etc/endpoints/ssl/ca-certificates.crt"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8080", "--ssl_backend_client_root_certs_path", "/etc/endpoints/ssl/ca-certificates.crt", "--service_control_enable_api_key_uid_reporting", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--listener_port=8080", "--disable_tracing", "--enable_grpc_backend_ssl"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8080", "--ssl_backend_client_root_certs_path", "/etc/nginx/trusted-ca-certificates.crt", "--service_control_enable_api_key_uid_reporting", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--listener_port=8080", "--disable_tracing", "--tls_mutual_auth"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8080", "--ssl_backend_client_cert_path", "/etc/nginx/ssl", "--service_control_enable_api_key_uid_reporting", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--listener_port=8080", "--disable_tracing", "--ssl_minimum_protocol=TLSv1.1", "--ssl_maximum_protocol=TLSv1.3"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8080", "--ssl_minimum_protocol", "TLSv1.1", "--ssl_maximum_protocol", "TLSv1.3", "--service_control_enable_api_key_uid_reporting", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--listener_port=8080", "--disable_tracing", "--ssl_server_cipher_suites=AES128-SHA,AES256-GCM-SHA384", "--ssl_backend_client_cipher_suites=AES256-SHA"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8080", "--ssl_server_cipher_suites", "AES128-SHA,AES256-GCM-SHA384", "--ssl_backend_client_cipher_suites", "AES256-SHA", "--service_control_enable_api_key_uid_reporting", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--listener_port=8080", "--disable_tracing", "--ssl_protocols=TLSv1.3", "--ssl_protocols=TLSv1.2"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8080", "--ssl_minimum_protocol", "TLSv1.2", "--ssl_maximum_protocol", "TLSv1.3", "--service_control_enable_api_key_uid_reporting", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--listener_port=8080", "--disable_tracing", "--ssl_protocols=TLSv1.2"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8080", "--ssl_minimum_protocol", "TLSv1.2", "--ssl_maximum_protocol", "TLSv1.2", "--service_control_enable_api_key_uid_reporting", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--listener_port=8080", "--disable_tracing", "--generate_self_signed_cert"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8080", "--ssl_server_cert_path", tmpCertDir, "--service_control_enable_api_key_uid_reporting", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--http2_port=8079", "--service_control_quota_retries=3", "--service_control_report_timeout_ms=300", "--check_metadata", "--disable_tracing"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8079", "--service_control_quota_retries", "3", "--service_control_report_timeout_ms", "300", "--service_control_enable_api_key_uid_reporting", "--check_metadata", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--listener_port=8079", "--service_control_quota_retries=3", "--service_control_report_timeout_ms=300", "--check_metadata", "--disable_tracing"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8079", "--service_control_quota_retries", "3", "--service_control_report_timeout_ms", "300", "--service_control_enable_api_key_uid_reporting", "--check_metadata", "--disable_tracing"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=https://127.0.0.1", "--cors_preset=basic", "--non_gcp", "--version=2019-11-09r0", "--service_account_key", "/tmp/service_accout_key"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "https://127.0.0.1", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--disable_tracing", "--cors_preset", "basic", "--cors_allow_origin", "*", "--cors_allow_origin_regex", "", "--cors_allow_methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS", "--cors_allow_headers", "DNT,User-Agent,X-User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization", "--cors_expose_headers", "Content-Length,Content-Range", "--cors_max_age", "480h", "--service_account_key", "/tmp/service_accout_key", "--non_gcp"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=https://127.0.0.1", "--cors_preset=cors_with_regex", "--cors_allow_origin_regex=^https?://.+.google.com$", "--cors_allow_methods=GET,POST,OPTIONS", "--cors_allow_headers=X-Requested-With,Content-Type,Range,Authorization", "--cors_expose_headers=Content-Length", "--cors_allow_credentials", "--cors_max_age=1200m", "--non_gcp", "--version=2019-11-09r0", "--service_account_key", "/tmp/service_accout_key"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "https://127.0.0.1", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--disable_tracing", "--cors_preset", "cors_with_regex", "--cors_allow_origin", "*", "--cors_allow_origin_regex", "^https?://.+.google.com$", "--cors_allow_methods", "GET,POST,OPTIONS", "--cors_allow_headers", "X-Requested-With,Content-Type,Range,Authorization", "--cors_expose_headers", "Content-Length", "--cors_max_age", "1200m", "--cors_allow_credentials", "--service_account_key", "/tmp/service_accout_key", "--non_gcp"},
		},
		{
			flags: []string{"--backend=https://127.0.0.1:8000", "--enable_backend_routing", "--service_json_path=/tmp/service.json", "--on_serverless", "--disable_tracing"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "https://127.0.0.1:8000", "--v", "0", "--envoy_xff_num_trusted_hops", "0", "--service_control_enable_api_key_uid_reporting", "--service_json_path", "/tmp/service.json", "--disable_tracing", "--compute_platform_override", "Cloud Run(ESPv2)"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--version=2019-11-09r0", "--backend=grpc://127.0.0.1:8000", "--http_request_timeout_s=10", "--log_jwt_payloads=aud,exp", "--disable_tracing"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "grpc://127.0.0.1:8000", "--v", "0", "--log_jwt_payloads", "aud,exp", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--http_request_timeout_s", "10", "--service_control_enable_api_key_uid_reporting", "--disable_tracing"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=grpc://127.0.0.1:8000", "--transcoding_always_print_primitive_fields", "--transcoding_preserve_proto_field_names", "--transcoding_always_print_enums_as_ints", "--disable_tracing", "--version=2019-11-09r0"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "grpc://127.0.0.1:8000", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--disable_tracing", "--transcoding_always_print_primitive_fields", "--transcoding_always_print_enums_as_ints", "--transcoding_preserve_proto_field_names"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=grpc://127.0.0.1:8000", "--transcoding_ignore_query_parameters=foo,bar", "--disable_tracing", "--version=2019-11-09r0"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "grpc://127.0.0.1:8000", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--disable_tracing", "--transcoding_ignore_query_parameters", "foo,bar"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=grpc://127.0.0.1:8000", "--transcoding_ignore_unknown_query_parameters", "--disable_tracing", "--version=2019-11-09r0"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "grpc://127.0.0.1:8000", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--disable_tracing", "--transcoding_ignore_unknown_query_parameters"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=grpc://127.0.0.1:8000", "--transcoding_query_parameters_disable_unescape_plus", "--disable_tracing", "--version=2019-11-09r0"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "grpc://127.0.0.1:8000", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--disable_tracing", "--transcoding_query_parameters_disable_unescape_plus"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=grpc://127.0.0.1:8000", "--transcoding_stream_newline_delimited", "--disable_tracing", "--version=2019-11-09r0"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "grpc://127.0.0.1:8000", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--disable_tracing", "--transcoding_stream_newline_delimited"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=grpc://127.0.0.1:8000", "--transcoding_case_insensitive_enum_parsing", "--disable_tracing", "--version=2019-11-09r0"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "grpc://127.0.0.1:8000", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--disable_tracing", "--transcoding_case_insensitive_enum_parsing"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=grpc://127.0.0.1:8000", "--disallow_colon_in_wildcard_path_segment", "--disable_tracing", "--version=2019-11-09r0"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "grpc://127.0.0.1:8000", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--disable_tracing", "--disallow_colon_in_wildcard_path_segment"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=grpc://127.0.0.1:8000", "--envoy_connection_buffer_limit_bytes=1024", "--disable_tracing", "--version=2019-11-09r0"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "grpc://127.0.0.1:8000", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--disable_tracing", "--connection_buffer_limit_bytes", "1024"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=echo:8000", "--enable_debug", "--disable_tracing", "--version=2019-11-09r0"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://echo:8000", "--v", "1", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--disable_tracing", "--suppress_envoy_headers=false"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=127.0.0.1:8000", "--access_log=/foo/bar", "--access_log_format=%START_TIME%", "--disable_tracing", "--version=2019-11-09r0"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1:8000", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--access_log", "/foo/bar", "--access_log_format", "%START_TIME%", "--disable_tracing"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=http://127.0.0.1", "--version=2019-11-09r0", "--service_account_key", "/tmp/service_accout_key", "--non_gcp"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--disable_tracing", "--service_account_key", "/tmp/service_accout_key", "--non_gcp"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=http://127.0.0.1", "--version=2019-11-09r0", "--enable_application_default_credentials", "--non_gcp", "--disable_tracing"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--disable_tracing", "--enable_application_default_credentials", "--non_gcp"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=http://127.0.0.1", "--version=2019-11-09r0", "--service_account_key", "/tmp/service_accout_key", "--non_gcp", "--tracing_project_id=test_project_1234"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--tracing_project_id", "test_project_1234", "--service_account_key", "/tmp/service_accout_key", "--non_gcp"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=grpc://127.0.0.1:8000", "--tracing_sample_rate=1", "--cloud_trace_url_override=localhost:9990", "--tracing_incoming_context=fake-incoming-context", "--tracing_outgoing_context=fake-outgoing-context", "--version=2019-11-09r0"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "grpc://127.0.0.1:8000", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--tracing_incoming_context", "fake-incoming-context", "--tracing_outgoing_context", "fake-outgoing-context", "--tracing_stackdriver_address", "localhost:9990", "--tracing_sample_rate", "1"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=grpc://127.0.0.1:8000", "--tracing_sample_rate=1", "--version=2019-11-09r0", "--enable_debug"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "grpc://127.0.0.1:8000", "--v", "1", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--tracing_sample_rate", "1", "--suppress_envoy_headers=false"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=grpc://127.0.0.1:8000", "--tracing_sample_rate=1", "--disable_cloud_trace_auto_sampling", "--version=2019-11-09r0"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "grpc://127.0.0.1:8000", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--tracing_sample_rate", "0"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=grpc://127.0.0.1:8000", "--tracing_sample_rate=1", "--disable_tracing", "--version=2019-11-09r0"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "grpc://127.0.0.1:8000", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--http2_port=8079", "--backend_retry_ons=5xx", "--disable_tracing"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8079", "--service_control_enable_api_key_uid_reporting", "--backend_retry_ons", "5xx", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--http2_port=8079", "--backend_retry_num=10", "--disable_tracing"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8079", "--service_control_enable_api_key_uid_reporting", "--backend_retry_num", "10", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--http2_port=8079", "--backend_per_try_timeout=10s", "--disable_tracing"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8079", "--service_control_enable_api_key_uid_reporting", "--backend_per_try_timeout", "10s", "--disable_tracing"},
		},
		{
			flags: []string{"-R=managed", "--http2_port=8079", "--backend_retry_on_status_codes=500,501", "--disable_tracing"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8079", "--service_control_enable_api_key_uid_reporting", "--backend_retry_on_status_codes", "500,501", "--disable_tracing"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=http://127.0.0.1", "--service_account_key", "/tmp/service_account_key", "--disable_tracing", "--version=2019-11-09r0"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--disable_tracing", "--service_account_key", "/tmp/service_account_key"},
		},
		{
			flags: []string{"--on_serverless", "--http_port=8080", "--rollout_strategy=fixed", "--service_json_path=/tmp/service_config.json", "--disable_tracing"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--envoy_xff_num_trusted_hops", "0", "--listener_port", "8080", "--service_control_enable_api_key_uid_reporting", "--service_json_path", "/tmp/service_config.json", "--disable_tracing", "--compute_platform_override", "Cloud Run(ESPv2)"},
		},
		{
			flags: []string{"--on_serverless", "--http_port=8080", "--rollout_strategy=fixed", "--service_json_path=/tmp/service_config.json", "--envoy_xff_num_trusted_hops=3", "--disable_tracing"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--envoy_xff_num_trusted_hops", "3", "--listener_port", "8080", "--service_control_enable_api_key_uid_reporting", "--service_json_path", "/tmp/service_config.json", "--disable_tracing", "--compute_platform_override", "Cloud Run(ESPv2)"},
		},
		{
			flags: []string{"--rollout_strategy=fixed", "--service_json_path=/tmp/service_config.json", "--add_request_header=k1=v1"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--add_request_headers", "k1=v1", "--service_control_enable_api_key_uid_reporting", "--service_json_path", "/tmp/service_config.json"},
		},
		{
			flags: []string{"--rollout_strategy=fixed", "--service_json_path=/tmp/service_config.json", "--add_request_header=k1=v1", "--add_request_header=k2=v2"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--add_request_headers", "k1=v1;k2=v2", "--service_control_enable_api_key_uid_reporting", "--service_json_path", "/tmp/service_config.json"},
		},
		{
			flags: []string{"--rollout_strategy=fixed", "--service_json_path=/tmp/service_config.json", "--append_request_header=k1=v1"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--append_request_headers", "k1=v1", "--service_control_enable_api_key_uid_reporting", "--service_json_path", "/tmp/service_config.json"},
		},
		{
			flags: []string{"--rollout_strategy=fixed", "--service_json_path=/tmp/service_config.json", "--append_request_header=k1=v1", "--append_request_header=k2=v2"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--append_request_headers", "k1=v1;k2=v2", "--service_control_enable_api_key_uid_reporting", "--service_json_path", "/tmp/service_config.json"},
		},
		{
			flags: []string{"--rollout_strategy=fixed", "--service_json_path=/tmp/service_config.json", "--add_response_header=k1=v1"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--add_response_headers", "k1=v1", "--service_control_enable_api_key_uid_reporting", "--service_json_path", "/tmp/service_config.json"},
		},
		{
			flags: []string{"--rollout_strategy=fixed", "--service_json_path=/tmp/service_config.json", "--add_response_header=k1=v1", "--add_response_header=k2=v2"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--add_response_headers", "k1=v1;k2=v2", "--service_control_enable_api_key_uid_reporting", "--service_json_path", "/tmp/service_config.json"},
		},
		{
			flags: []string{"--rollout_strategy=fixed", "--service_json_path=/tmp/service_config.json", "--append_response_header=k1=v1"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--append_response_headers", "k1=v1", "--service_control_enable_api_key_uid_reporting", "--service_json_path", "/tmp/service_config.json"},
		},
		{
			flags: []string{"--rollout_strategy=fixed", "--service_json_path=/tmp/service_config.json", "--append_response_header=k1=v1", "--append_response_header=k2=v2"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--append_response_headers", "k1=v1;k2=v2", "--service_control_enable_api_key_uid_reporting", "--service_json_path", "/tmp/service_config.json"},
		},
		{
			flags: []string{"--rollout_strategy=fixed", "--service_json_path=/tmp/service_config.json", "--disable_normalize_path", "--disable_merge_slashes_in_path"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--service_control_enable_api_key_uid_reporting", "--service_json_path", "/tmp/service_config.json", "--normalize_path=false", "--merge_slashes_in_path=false"},
		},
		{
			flags: []string{"--rollout_strategy=fixed", "--service_json_path=/tmp/service_config.json", "--disallow_escaped_slashes_in_path"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--service_control_enable_api_key_uid_reporting", "--service_json_path", "/tmp/service_config.json", "--disallow_escaped_slashes_in_path"},
		},
		{
			flags: []string{"--rollout_strategy=fixed", "--service_json_path=/tmp/service_config.json", "--enable_operation_name_header"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--enable_operation_name_header", "--service_control_enable_api_key_uid_reporting", "--service_json_path", "/tmp/service_config.json"},
		},
		{
			flags: []string{"--rollout_strategy=fixed", "--service_json_path=/tmp/service_config.json", "--enable_response_compression"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--enable_response_compression", "--service_control_enable_api_key_uid_reporting", "--service_json_path", "/tmp/service_config.json"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=grpc://127.0.0.1:8000", "--health_check_grpc_backend", "--version=2019-11-09r0"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "grpc://127.0.0.1:8000", "--health_check_grpc_backend", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=grpc://127.0.0.1:8000", "--ads_named_pipe=@espv2-named-pipe-9", "--version=2019-11-09r0"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "grpc://127.0.0.1:8000", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--ads_named_pipe", "@espv2-named-pipe-9"},
		},
		{
			flags: []string{"--service=test_bookstore.gloud.run", "--backend=grpcs://127.0.0.1:8000", "--health_check_grpc_backend", "--health_check_grpc_backend_interval=3s", "--health_check_grpc_backend_service=/foo.bar", "--health_check_grpc_backend_no_traffic_interval=5s", "--version=2019-11-09r0"},
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "grpcs://127.0.0.1:8000", "--health_check_grpc_backend", "--health_check_grpc_backend_service", "/foo.bar", "--health_check_grpc_backend_interval", "3s", "--health_check_grpc_backend_no_traffic_interval", "5s", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting"},
		},
	}

	for _, tc := range testcases {
		t.Run(strings.Join(tc.flags, " "), func(t *testing.T) {
			unsetEnvForTest(t, googleCredsEnv)
			unsetEnvForTest(t, otelResourceAttributesEnv)
			args, err := parseArgs(tc.flags)
			if err != nil {
				t.Fatalf("unexpected parseArgs error: %v", err)
			}
			got, err := l.genProxyConfig(args)
			if err != nil {
				t.Fatalf("unexpected genProxyConfig error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got  %q\nwant %q", got, tc.want)
			}
		})
	}
}

func TestGenProxyConfigError(t *testing.T) {
	l := newLauncher()
	// wantErr is start_proxy.py's message (with two typos fixed).
	testcases := []struct {
		flags   []string
		wantErr string
	}{
		{flags: []string{"--unknown_flag"}, wantErr: "unrecognized arguments: --unknown_flag"},
		{flags: []string{"--rollout_strategy=managed", "--v=2019-11-09r0"}, wantErr: "Flag --version cannot be used if --rollout_strategy=managed."},
		{flags: []string{"--rollout_strategy=fixed"}, wantErr: "Flag --version is required if --rollout_strategy=fixed."},
		{flags: []string{"--service=test_bookstore.gloud.run", "--service_json_path=/tmp/service.json"}, wantErr: "Flag --service cannot be used together with --service_json_path."},
		{flags: []string{"--version=2019-11-09r0", "--service_json_path=/tmp/service.json"}, wantErr: "Flag --version cannot be used together with --service_json_path."},
		{flags: []string{"--rollout_strategy=managed", "--service_json_path=/tmp/service.json"}, wantErr: "Flag -R or --rollout_strategy must be fixed with --service_json_path."},
		{flags: []string{"--version=2019-11-09r0", "--backend_dns_lookup_family=v4"}, wantErr: "argument --backend_dns_lookup_family: invalid choice: 'v4' (choose from 'auto', 'v4only', 'v6only', 'v4preferred', 'all')"},
		{flags: []string{"--version=2019-11-09r0", "--non_gcp"}, wantErr: "If --non_gcp is specified, --service_account_key or --enable_application_default_credentials has to be specified, or GOOGLE_APPLICATION_CREDENTIALS has to set in os.environ."},
		{flags: []string{"--version=2019-11-09r0", "--http_port=8000", "--http2_port=8000"}, wantErr: "Multiple port flags --http_port,--http2_port are not allowed, use only the --listener_port flag"},
		{flags: []string{"--version=2019-11-09r0", "--http_port=8000", "--listener_port=8000"}, wantErr: "Multiple port flags --http_port,--listener_port are not allowed, use only the --listener_port flag"},
		{flags: []string{"--version=2019-11-09r0", "--listener_port=8000", "--ssl_port=9000"}, wantErr: "Multiple port flags --listener_port,--ssl_port are not allowed, use only the --listener_port flag"},
		{flags: []string{"--version=2019-11-09r0", "--listener_port=80"}, wantErr: "Port 80 is a privileged port. For security purposes, the ESPv2 container cannot bind to it. Use any port above 1024 instead."},
		{flags: []string{"--version=2019-11-09r0", "--http_port=80"}, wantErr: "Port 80 is a privileged port. For security purposes, the ESPv2 container cannot bind to it. Use any port above 1024 instead."},
		{flags: []string{"--version=2019-11-09r0", "--http2_port=80"}, wantErr: "Port 80 is a privileged port. For security purposes, the ESPv2 container cannot bind to it. Use any port above 1024 instead."},
		{flags: []string{"--version=2019-11-09r0", "--ssl_port=443"}, wantErr: "Port 443 is a privileged port. For security purposes, the ESPv2 container cannot bind to it. Use any port above 1024 instead."},
		{flags: []string{"--version=2019-11-09r0", "--ssl_server_cert_path=/etc/endpoint/ssl", "--ssl_port=9000"}, wantErr: "Flag --ssl_port is going to be deprecated, please use --ssl_server_cert_path only."},
		{flags: []string{"--version=2019-11-09r0", "--ssl_server_cert_path=/etc/endpoint/ssl", "--generate_self_signed_cert"}, wantErr: "Flag --generate_self_signed_cert and --ssl_server_cert_path cannot be used simultaneously."},
		{flags: []string{"--version=2019-11-09r0", "--ssl_backend_client_cert_path=/etc/endpoint/ssl", "--tls_mutual_auth"}, wantErr: "Flag --tls_mutual_auth is going to be deprecated, please use --ssl_backend_client_cert_path only."},
		{flags: []string{"--version=2019-11-09r0", "--ssl_client_cert_path=/etc/endpoint/ssl", "--tls_mutual_auth"}, wantErr: "Flag --tls_mutual_auth is going to be deprecated, please use --ssl_backend_client_cert_path only."},
		{flags: []string{"--version=2019-11-09r0", "--ssl_protocols=TLSv1.3", "--ssl_minimum_protocol=TLSv1.1"}, wantErr: "Flag --ssl_protocols is going to be deprecated, please use --ssl_minimum_protocol and --ssl_maximum_protocol."},
		{flags: []string{"--version=2019-11-09r0", "--ssl_minimum_protocol=TLSv11"}, wantErr: "argument --ssl_minimum_protocol: invalid choice: 'TLSv11' (choose from 'TLSv1.0', 'TLSv1.1', 'TLSv1.2', 'TLSv1.3')"},
		{flags: []string{"--version=2019-11-09r0", "--transcoding_ignore_query_parameters=foo,bar", "--transcoding_ignore_unknown_query_parameters"}, wantErr: "Flag --transcoding_ignore_query_parameters cannot be used together with --transcoding_ignore_unknown_query_parameters."},
		{flags: []string{"--version=2019-11-09r0", "--dns=127.0.0.1", "--dns_resolver_address=127.0.0.1"}, wantErr: "Flag --dns_resolver_addresses cannot be used together with --dns."},
		{flags: []string{"--version=2019-11-09r0", "--ssl_client_cert_path=/tmp", "--ssl_backend_client_cert_path=/tmp"}, wantErr: "Flag --ssl_client_cert_path is renamed to --ssl_backend_client_cert_path, only use the latter flag."},
		{flags: []string{"--version=2019-11-09r0", "--health_check_grpc_backend"}, wantErr: "Flag --health_check_grpc_backend requires the flag --backend to use grpc scheme."},
		{flags: []string{"--version=2019-11-09r0", "--health_check_grpc_backend", "--backend=http://abc.com"}, wantErr: "Flag --health_check_grpc_backend requires the flag --backend to use grpc scheme."},
		{flags: []string{"--version=2019-11-09r0", "--health_check_grpc_backend_interval=3s"}, wantErr: "Flag --health_check_grpc_backend_interval requires the flag --health_check_grpc_backend to be used."},
		{flags: []string{"--version=2019-11-09r0", "--health_check_grpc_backend_service=/foo.bar"}, wantErr: "Flag --health_check_grpc_backend_service requires the flag --health_check_grpc_backend to be used."},
		{flags: []string{"--version=2019-11-09r0", "--health_check_grpc_backend_no_traffic_interval=1s"}, wantErr: "Flag --health_check_grpc_backend_no_traffic_interval requires the flag --health_check_grpc_backend to be used."},
		{flags: []string{"--version=2019-11-09r0", "--ssl_client_root_certs_file=/tmp/server.crt", "--ssl_backend_client_root_certs_file=/tmp/server.crt"}, wantErr: "Flag --ssl_client_root_certs_file is renamed to --ssl_backend_client_root_certs_file, only use the latter flag."},
		{flags: []string{"--version=2019-11-09r0", "--non_gcp", "--service_account_key=tmp/service_account_key", "--enable_application_default_credentials"}, wantErr: "Only one of --service_account_key or --enable_application_default_credentials can be supplied for credentials at once."},
		{flags: []string{"--version=2019-11-09r0", "--ssl_backend_client_root_certs_file=/etc/ssl/certs/ca.crt", "--enable_grpc_backend_ssl"}, wantErr: "Flag --enable_grpc_backend_ssl are going to be deprecated, please use --ssl_backend_client_root_certs_file only."},
		{flags: []string{"--version=2019-11-09r0", "--ssl_client_root_certs_file=/etc/ssl/certs/ca.crt", "--enable_grpc_backend_ssl"}, wantErr: "Flag --enable_grpc_backend_ssl are going to be deprecated, please use --ssl_backend_client_root_certs_file only."},
		{flags: []string{"--version=2019-11-09r0", "--access_log_format=%START_TIME%"}, wantErr: "Flag --access_log_format has to be used together with --access_log."},
	}

	for _, tc := range testcases {
		t.Run(strings.Join(tc.flags, " "), func(t *testing.T) {
			unsetEnvForTest(t, googleCredsEnv)
			args, err := parseArgs(tc.flags)
			if err == nil {
				_, err = l.genProxyConfig(args)
			}
			if err == nil || err.Error() != tc.wantErr {
				t.Errorf("got error %v, want %q", err, tc.wantErr)
			}
		})
	}
}

// start_proxy.py only demanded credentials for --non_gcp when
// --service_account_key was None, so an empty key, or an empty
// GOOGLE_APPLICATION_CREDENTIALS, is accepted.
func TestGenProxyConfigNonGcpEmptyServiceAccountKey(t *testing.T) {
	l := newLauncher()
	want := []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--disable_tracing", "--non_gcp"}
	for _, tc := range []struct {
		desc  string
		flags []string
		env   *string
	}{
		{desc: "empty flag", flags: []string{"--version=2019-11-09r0", "--non_gcp", "--service_account_key="}},
		{desc: "empty env", flags: []string{"--version=2019-11-09r0", "--non_gcp"}, env: new(string)},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			unsetEnvForTest(t, googleCredsEnv)
			unsetEnvForTest(t, otelResourceAttributesEnv)
			if tc.env != nil {
				t.Setenv(googleCredsEnv, *tc.env)
			}
			args, err := parseArgs(tc.flags)
			if err != nil {
				t.Fatalf("unexpected parseArgs error: %v", err)
			}
			got, err := l.genProxyConfig(args)
			if err != nil {
				t.Fatalf("unexpected genProxyConfig error: %v", err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got  %q\nwant %q", got, want)
			}
		})
	}
}

func TestGenEnvoyArgs(t *testing.T) {
	l := newLauncher()
	testcases := []struct {
		flags []string
		want  []string
	}{
		{
			flags: []string{},
			want:  []string{"bin/envoy", "-c", "/tmp/bootstrap.json", "--disable-hot-restart", "--log-format %L%m%d %T.%e %t %@] [%t][%n]%v", "--log-format-escaped"},
		},
		{
			flags: []string{"--enable_debug"},
			want:  []string{"bin/envoy", "-c", "/tmp/bootstrap.json", "--disable-hot-restart", "--log-format %L%m%d %T.%e %t %@] [%t][%n]%v", "--log-format-escaped", "-l debug", "--component-log-level upstream:info,main:info"},
		},
		{
			flags: []string{"--envoy_extra_config_yaml=node: {id: 'node1'}"},
			want:  []string{"bin/envoy", "-c", "/tmp/bootstrap.json", "--disable-hot-restart", "--log-format %L%m%d %T.%e %t %@] [%t][%n]%v", "--log-format-escaped", "--config-yaml node: {id: 'node1'}"},
		},
	}

	for _, tc := range testcases {
		t.Run(strings.Join(tc.flags, " "), func(t *testing.T) {
			args, err := parseArgs(tc.flags)
			if err != nil {
				t.Fatalf("unexpected parseArgs error: %v", err)
			}
			if got := l.genEnvoyArgs(args); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got  %q\nwant %q", got, tc.want)
			}
		})
	}
}

func TestServiceAccountKeyWithEnv(t *testing.T) {
	l := newLauncher()
	strPtr := func(s string) *string { return &s }
	testcases := []struct {
		oldEnv    *string
		wantedEnv *string
		flags     []string
		want      []string
	}{
		{
			oldEnv:    nil,
			wantedEnv: nil,
			flags:     []string{"--service=test_bookstore.gloud.run", "--backend=http://127.0.0.1", "--version=2019-11-09r0"},
			want:      []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting"},
		},
		{
			oldEnv:    strPtr("/tmp/service_account_key"),
			wantedEnv: strPtr("/tmp/service_account_key"),
			flags:     []string{"--service=test_bookstore.gloud.run", "--backend=http://127.0.0.1", "--version=2019-11-09r0"},
			want:      []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--service_account_key", "/tmp/service_account_key"},
		},
		{
			oldEnv:    nil,
			wantedEnv: strPtr("/tmp/service_account_key"),
			flags:     []string{"--service=test_bookstore.gloud.run", "--backend=http://127.0.0.1", "--version=2019-11-09r0", "--service_account_key", "/tmp/service_account_key"},
			want:      []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--service_account_key", "/tmp/service_account_key"},
		},
		{
			oldEnv:    strPtr("/tmp/service_account_key111"),
			wantedEnv: strPtr("/tmp/service_account_key111"),
			flags:     []string{"--service=test_bookstore.gloud.run", "--backend=http://127.0.0.1", "--version=2019-11-09r0", "--service_account_key", "/tmp/service_account_key222"},
			want:      []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "fixed", "--backend_address", "http://127.0.0.1", "--v", "0", "--service", "test_bookstore.gloud.run", "--service_config_id", "2019-11-09r0", "--service_control_enable_api_key_uid_reporting", "--service_account_key", "/tmp/service_account_key222"},
		},
	}

	for _, tc := range testcases {
		t.Run(strings.Join(tc.flags, " "), func(t *testing.T) {
			unsetEnvForTest(t, googleCredsEnv)
			if tc.oldEnv != nil {
				t.Setenv(googleCredsEnv, *tc.oldEnv)
			}
			args, err := parseArgs(tc.flags)
			if err != nil {
				t.Fatalf("unexpected parseArgs error: %v", err)
			}
			got, err := l.genProxyConfig(args)
			if err != nil {
				t.Fatalf("unexpected genProxyConfig error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got  %q\nwant %q", got, tc.want)
			}
			envVal, hasEnv := os.LookupEnv(googleCredsEnv)
			if tc.wantedEnv != nil {
				if !hasEnv || envVal != *tc.wantedEnv {
					t.Errorf("got env %q (exists=%v), want %q", envVal, hasEnv, *tc.wantedEnv)
				}
			} else if hasEnv {
				t.Errorf("expected %s to be unset, got %q", googleCredsEnv, envVal)
			}
		})
	}
}

func TestGenProxyConfigNonGcpWithServiceAccountKeyKeepsTracing(t *testing.T) {
	l := newLauncher()
	unsetEnvForTest(t, googleCredsEnv)
	unsetEnvForTest(t, otelResourceAttributesEnv)
	saPath := writeServiceAccountKey(t, `{"project_id": "sa-test-proj-789"}`)

	args, err := parseArgs([]string{
		"--service=test_bookstore.gloud.run",
		"--backend=http://127.0.0.1",
		"--version=2019-11-09r0",
		"--non_gcp",
		"--service_account_key=" + saPath,
	})
	if err != nil {
		t.Fatalf("unexpected parseArgs error: %v", err)
	}
	got, err := l.genProxyConfig(args)
	if err != nil {
		t.Fatalf("unexpected genProxyConfig error: %v", err)
	}

	tracingProjectID := ""
	for i, arg := range got {
		if arg == "--disable_tracing" {
			t.Errorf("unexpected --disable_tracing in %v", got)
		}
		if arg == "--tracing_project_id" && i+1 < len(got) {
			tracingProjectID = got[i+1]
		}
	}
	if tracingProjectID != "sa-test-proj-789" {
		t.Errorf("--tracing_project_id = %q, want %q; args: %v", tracingProjectID, "sa-test-proj-789", got)
	}
}
