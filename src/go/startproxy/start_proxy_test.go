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
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestGenBootstrap(t *testing.T) {
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

	for i, tc := range testcases {
		_ = os.Unsetenv(GoogleCredsKey)
		args, err := ParseArgs(tc.flags)
		if err != nil {
			t.Fatalf("case %d (%v): unexpected ParseArgs error: %v", i, tc.flags, err)
		}
		got := GenBootstrapConf(args)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("case %d (%v):\ngot  %v\nwant %v", i, tc.flags, got, tc.want)
		}
	}
}

func TestGenProxyConfig(t *testing.T) {
	tmpCertDir := t.TempDir()
	origCertDir := SelfSignedCertDir
	SelfSignedCertDir = tmpCertDir
	defer func() { SelfSignedCertDir = origCertDir }()

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
			want:  []string{"bin/configmanager", "--logtostderr", "--rollout_strategy", "managed", "--backend_address", "http://127.0.0.1:8082", "--v", "0", "--listener_port", "8080", "--ssl_server_cert_path", "/tmp/ssl/endpoints", "--service_control_enable_api_key_uid_reporting", "--disable_tracing"},
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

	for i, tc := range testcases {
		_ = os.Unsetenv(GoogleCredsKey)
		args, err := ParseArgs(tc.flags)
		if err != nil {
			t.Fatalf("case %d (%v): unexpected ParseArgs error: %v", i, tc.flags, err)
		}
		got, err := GenProxyConfig(args)
		if err != nil {
			t.Fatalf("case %d (%v): unexpected GenProxyConfig error: %v", i, tc.flags, err)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("case %d (%v):\ngot  %v\nwant %v", i, tc.flags, got, tc.want)
		}
	}

	// Verify self-signed cert files were generated in pure Go and are valid X.509 / RSA PEM.
	certPEM, err := os.ReadFile(filepath.Join(tmpCertDir, "server.crt"))
	if err != nil {
		t.Fatalf("expected server.crt to be created: %v", err)
	}
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatalf("invalid PEM block in server.crt")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse generated X.509 certificate: %v", err)
	}
	if cert.Subject.CommonName != "localhost" {
		t.Errorf("expected CN=localhost, got %q", cert.Subject.CommonName)
	}

	keyPEM, err := os.ReadFile(filepath.Join(tmpCertDir, "server.key"))
	if err != nil {
		t.Fatalf("expected server.key to be created: %v", err)
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil || keyBlock.Type != "RSA PRIVATE KEY" {
		t.Fatalf("invalid PEM block in server.key")
	}
	if _, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes); err != nil {
		t.Fatalf("failed to parse generated RSA private key: %v", err)
	}
}

func TestGenProxyConfigError(t *testing.T) {
	testcases := [][]string{
		[]string{"--unknown_flag"},
		[]string{"--rollout_strategy=managed", "--v=2019-11-09r0"},
		[]string{"--rollout_strategy=fixed"},
		[]string{"--service=test_bookstore.gloud.run", "--service_json_path=/tmp/service.json"},
		[]string{"--version=2019-11-09r0", "--service_json_path=/tmp/service.json"},
		[]string{"--rollout_strategy=managed", "--service_json_path=/tmp/service.json"},
		[]string{"--version=2019-11-09r0", "--backend_dns_lookup_family=v4"},
		[]string{"--version=2019-11-09r0", "--non_gcp"},
		[]string{"--version=2019-11-09r0", "--http_port=8000", "--http2_port=8000"},
		[]string{"--version=2019-11-09r0", "--http_port=8000", "--listener_port=8000"},
		[]string{"--version=2019-11-09r0", "--listener_port=8000", "--ssl_port=9000"},
		[]string{"--version=2019-11-09r0", "--listener_port=80"},
		[]string{"--version=2019-11-09r0", "--http_port=80"},
		[]string{"--version=2019-11-09r0", "--http2_port=80"},
		[]string{"--version=2019-11-09r0", "--ssl_port=443"},
		[]string{"--version=2019-11-09r0", "--ssl_server_cert_path=/etc/endpoint/ssl", "--ssl_port=9000"},
		[]string{"--version=2019-11-09r0", "--ssl_server_cert_path=/etc/endpoint/ssl", "--generate_self_signed_cert"},
		[]string{"--version=2019-11-09r0", "--ssl_backend_client_cert_path=/etc/endpoint/ssl", "--tls_mutual_auth"},
		[]string{"--version=2019-11-09r0", "--ssl_client_cert_path=/etc/endpoint/ssl", "--tls_mutual_auth"},
		[]string{"--version=2019-11-09r0", "--ssl_protocols=TLSv1.3", "--ssl_minimum_protocol=TLSv1.1"},
		[]string{"--version=2019-11-09r0", "--ssl_minimum_protocol=TLSv11"},
		[]string{"--version=2019-11-09r0", "--ssl_backend_client_root_certs_file", "--enable_grpc_backend_ssl"},
		[]string{"--version=2019-11-09r0", "--ssl_client_root_certs_file", "--enable_grpc_backend_ssl"},
		[]string{"--version=2019-11-09r0", "--transcoding_ignore_query_parameters=foo,bar", "--transcoding_ignore_unknown_query_parameters"},
		[]string{"--version=2019-11-09r0", "--access_log_format"},
		[]string{"--version=2019-11-09r0", "--dns=127.0.0.1", "--dns_resolver_address=127.0.0.1"},
		[]string{"--version=2019-11-09r0", "--ssl_client_cert_path=/tmp", "--ssl_backend_client_cert_path=/tmp"},
		[]string{"--version=2019-11-09r0", "--health_check_grpc_backend"},
		[]string{"--version=2019-11-09r0", "--health_check_grpc_backend", "--backend=http://abc.com"},
		[]string{"--version=2019-11-09r0", "--health_check_grpc_backend_interval=3s"},
		[]string{"--version=2019-11-09r0", "--health_check_grpc_backend_service=/foo.bar"},
		[]string{"--version=2019-11-09r0", "--health_check_grpc_backend_no_traffic_interval=1s"},
		[]string{"--version=2019-11-09r0", "--ssl_client_root_certs_file=/tmp/server.crt", "--ssl_backend_client_root_certs_file=/tmp/server.crt"},
		[]string{"--non_gcp", "--service_account_key=tmp/service_account_key", "--enable_application_default_credentials"},
		[]string{"--non_gcp"},
	}

	for i, flags := range testcases {
		_ = os.Unsetenv(GoogleCredsKey)
		args, err := ParseArgs(flags)
		if err == nil {
			_, err = GenProxyConfig(args)
		}
		if err == nil {
			t.Errorf("case %d (%v): expected error, got nil", i, flags)
		}
	}
}

func TestGenEnvoyArgs(t *testing.T) {
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

	for i, tc := range testcases {
		_ = os.Unsetenv(GoogleCredsKey)
		args, err := ParseArgs(tc.flags)
		if err != nil {
			t.Fatalf("case %d (%v): unexpected ParseArgs error: %v", i, tc.flags, err)
		}
		got := GenEnvoyArgs(args)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("case %d (%v):\ngot  %v\nwant %v", i, tc.flags, got, tc.want)
		}
	}
}

func TestServiceAccountKeyWithEnv(t *testing.T) {
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

	for i, tc := range testcases {
		if tc.oldEnv != nil {
			_ = os.Setenv(GoogleCredsKey, *tc.oldEnv)
		} else {
			_ = os.Unsetenv(GoogleCredsKey)
		}
		args, err := ParseArgs(tc.flags)
		if err != nil {
			t.Fatalf("case %d: unexpected ParseArgs error: %v", i, err)
		}
		got, err := GenProxyConfig(args)
		if err != nil {
			t.Fatalf("case %d: unexpected GenProxyConfig error: %v", i, err)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("case %d:\ngot  %v\nwant %v", i, got, tc.want)
		}
		envVal, hasEnv := os.LookupEnv(GoogleCredsKey)
		if tc.wantedEnv != nil {
			if !hasEnv || envVal != *tc.wantedEnv {
				t.Errorf("case %d: got env %q (exists=%v), want %q", i, envVal, hasEnv, *tc.wantedEnv)
			}
		} else if hasEnv {
			t.Errorf("case %d: expected %s to be unset, got %q", i, GoogleCredsKey, envVal)
		}
	}
	_ = os.Unsetenv(GoogleCredsKey)
}

func TestHelpFlag(t *testing.T) {
	for _, flag := range []string{"-h", "--help"} {
		_, err := ParseArgs([]string{flag})
		if !errors.Is(err, ErrHelpRequested) {
			t.Errorf("expected ErrHelpRequested for flag %q, got: %v", flag, err)
		}

		if exitCode := Run([]string{flag}); exitCode != 0 {
			t.Errorf("expected Run(%q) to return 0, got %d", flag, exitCode)
		}
	}
}

func TestPrintUsage(t *testing.T) {
	var buf bytes.Buffer
	PrintUsage(&buf)
	out := buf.String()
	if !strings.Contains(out, "usage: startproxy [-h]") {
		t.Errorf("expected usage line, got:\n%s", out)
	}
	if !strings.Contains(out, "--service") {
		t.Errorf("expected --service in usage, got:\n%s", out)
	}
	if !strings.Contains(out, "--help") {
		t.Errorf("expected --help in usage, got:\n%s", out)
	}
}
