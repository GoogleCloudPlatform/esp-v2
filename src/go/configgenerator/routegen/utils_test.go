package routegen

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	"github.com/google/go-cmp/cmp"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

func TestParseSelectorsFromOPConfig(t *testing.T) {
	testdata := []struct {
		name          string
		serviceConfig *servicepb.Service
		opts          options.ConfigGeneratorOptions
		want          []string
	}{
		{
			name: "happy_path",
			serviceConfig: &servicepb.Service{
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name: "Foo",
							},
							{
								Name: "Bar",
							},
						},
					},
					{
						Name: "endpoints.examples.bookstore.v2.Library",
						Methods: []*apipb.Method{
							{
								Name: "Bar",
							},
							{
								Name: "Baz",
							},
						},
					},
				},
			},
			want: []string{
				// ordering matches OP service config
				"endpoints.examples.bookstore.Bookstore.Foo",
				"endpoints.examples.bookstore.Bookstore.Bar",
				"endpoints.examples.bookstore.v2.Library.Bar",
				"endpoints.examples.bookstore.v2.Library.Baz",
			},
		},
		{
			name: "discovery_api_skipped",
			serviceConfig: &servicepb.Service{
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name: "Foo",
							},
						},
					},
					{
						Name: "google.discovery.Discovery",
						Methods: []*apipb.Method{
							{
								Name: "GetDiscoveryRest",
							},
						},
					},
				},
			},
			want: []string{
				"endpoints.examples.bookstore.Bookstore.Foo",
			},
		},
		{
			name: "discovery_api_allowed_by_option",
			serviceConfig: &servicepb.Service{
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name: "Foo",
							},
						},
					},
					{
						Name: "google.discovery.Discovery",
						Methods: []*apipb.Method{
							{
								Name: "GetDiscoveryRest",
							},
						},
					},
				},
			},
			opts: options.ConfigGeneratorOptions{
				AllowDiscoveryAPIs: true,
			},
			want: []string{
				"endpoints.examples.bookstore.Bookstore.Foo",
				"google.discovery.Discovery.GetDiscoveryRest",
			},
		},
	}

	for _, tc := range testdata {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseSelectorsFromOPConfig(tc.serviceConfig, tc.opts)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ParseSelectorsFromOPConfig(...) diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseBackendClusterBySelectorFromOPConfig(t *testing.T) {
	testdata := []struct {
		name          string
		serviceConfig *servicepb.Service
		opts          options.ConfigGeneratorOptions
		want          map[string]*BackendClusterSpecifier
	}{
		{
			name: "local_clusters_only",
			serviceConfig: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name: "CreateBook",
							},
							{
								Name: "ListBooks",
							},
						},
					},
				},
			},
			want: map[string]*BackendClusterSpecifier{
				"endpoints.examples.bookstore.Bookstore.CreateBook": {
					Name: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
				},
				"endpoints.examples.bookstore.Bookstore.ListBooks": {
					Name: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
				},
			},
		},
		{
			name: "remote_clusters_only",
			serviceConfig: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name: "CreateBook",
							},
							{
								Name: "ListBooks",
							},
						},
					},
				},
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.CreateBook",
							Address:  "https://testapipb.foo.com/baz",
						},
						{
							Selector: "endpoints.examples.bookstore.Bookstore.ListBooks",
							Address:  "https://testapipb.bar.com/yyy",
						},
					},
				},
			},
			want: map[string]*BackendClusterSpecifier{
				"endpoints.examples.bookstore.Bookstore.CreateBook": {
					Name:     "backend-cluster-testapipb.foo.com:443",
					HostName: "testapipb.foo.com",
				},
				"endpoints.examples.bookstore.Bookstore.ListBooks": {
					Name:     "backend-cluster-testapipb.bar.com:443",
					HostName: "testapipb.bar.com",
				},
			},
		},
		{
			name: "mixed_local_and_remote",
			serviceConfig: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name: "CreateBook",
							},
							{
								Name: "ListBooks",
							},
						},
					},
				},
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.CreateBook",
							Address:  "https://testapipb.foo.com/baz",
						},
					},
				},
			},
			want: map[string]*BackendClusterSpecifier{
				"endpoints.examples.bookstore.Bookstore.CreateBook": {
					Name:     "backend-cluster-testapipb.foo.com:443",
					HostName: "testapipb.foo.com",
				},
				"endpoints.examples.bookstore.Bookstore.ListBooks": {
					Name: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
				},
			},
		},
		{
			// Backend rules' remote addresses are ignored when
			// backend address override is enabled.
			name: "backend_address_override",
			serviceConfig: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name: "CreateBook",
							},
							{
								Name: "ListBooks",
							},
						},
					},
				},
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.CreateBook",
							Address:  "https://testapipb.foo.com/baz",
						},
						{
							Selector: "endpoints.examples.bookstore.Bookstore.ListBooks",
							Address:  "https://testapipb.bar.com/yyy",
						},
					},
				},
			},
			opts: options.ConfigGeneratorOptions{
				EnableBackendAddressOverride: true,
			},
			want: map[string]*BackendClusterSpecifier{
				"endpoints.examples.bookstore.Bookstore.CreateBook": {
					Name: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
				},
				"endpoints.examples.bookstore.Bookstore.ListBooks": {
					Name: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
				},
			},
		},
		{
			// Backend rules are ignored when rule has empty address.
			name: "empty_address",
			serviceConfig: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name: "CreateBook",
							},
							{
								Name: "ListBooks",
							},
						},
					},
				},
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.CreateBook",
							Deadline: 180,
						},
						{
							Selector: "endpoints.examples.bookstore.Bookstore.ListBooks",
							Address:  "https://testapipb.bar.com/yyy",
						},
					},
				},
			},
			want: map[string]*BackendClusterSpecifier{
				"endpoints.examples.bookstore.Bookstore.CreateBook": {
					Name: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
				},
				"endpoints.examples.bookstore.Bookstore.ListBooks": {
					Name:     "backend-cluster-testapipb.bar.com:443",
					HostName: "testapipb.bar.com",
				},
			},
		},
		{
			// Backend rules are ignored when selector doesn't map to any existing
			// method.
			// Should never happen in practice, as API compiler guarantees all
			// selectors are valid.
			name: "invalid_selector",
			serviceConfig: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name: "ListBooks",
							},
						},
					},
				},
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.InvalidSelector",
							Deadline: 180,
						},
						{
							Selector: "endpoints.examples.bookstore.Bookstore.ListBooks",
							Address:  "https://testapipb.bar.com/yyy",
						},
					},
				},
			},
			want: map[string]*BackendClusterSpecifier{
				"endpoints.examples.bookstore.Bookstore.ListBooks": {
					Name:     "backend-cluster-testapipb.bar.com:443",
					HostName: "testapipb.bar.com",
				},
			},
		},
	}

	for _, tc := range testdata {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseBackendClusterBySelectorFromOPConfig(tc.serviceConfig, tc.opts)
			if err != nil {
				t.Fatalf("ParseBackendClusterBySelectorFromOPConfig(...) got err %v, want no err", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ParseBackendClusterBySelectorFromOPConfig(...) diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseHTTPPatternsBySelectorFromOPConfig(t *testing.T) {
	testdata := []struct {
		name          string
		serviceConfig *servicepb.Service
		opts          options.ConfigGeneratorOptions
		want          map[string][]*httppattern.Pattern
	}{
		{
			name: "grpc_service_no_http_rules",
			serviceConfig: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name: "ListShelves",
							},
							{
								Name: "CreateShelf",
							},
						},
					},
				},
			},
			opts: options.ConfigGeneratorOptions{
				BackendAddress: "grpc://127.0.0.1:80",
			},
			want: map[string][]*httppattern.Pattern{
				"endpoints.examples.bookstore.Bookstore.ListShelves": {
					{
						HttpMethod:  util.POST,
						UriTemplate: parseUriTemplate(t, "/endpoints.examples.bookstore.Bookstore/ListShelves"),
					},
				},
				"endpoints.examples.bookstore.Bookstore.CreateShelf": {
					{
						HttpMethod:  util.POST,
						UriTemplate: parseUriTemplate(t, "/endpoints.examples.bookstore.Bookstore/CreateShelf"),
					},
				},
			},
		},
		{
			name: "grpc_service_http_rule",
			serviceConfig: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name: "ListShelves",
							},
							{
								Name: "CreateShelf",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/v1/shelves",
							},
						},
						{
							Selector: "endpoints.examples.bookstore.Bookstore.CreateShelf",
							Pattern: &annotationspb.HttpRule_Post{
								Post: "/v2/shelves",
							},
							Body: "shelf",
						},
					},
				},
			},
			opts: options.ConfigGeneratorOptions{
				BackendAddress: "grpc://127.0.0.1:80",
			},
			want: map[string][]*httppattern.Pattern{
				"endpoints.examples.bookstore.Bookstore.ListShelves": {
					{
						HttpMethod:  util.GET,
						UriTemplate: parseUriTemplate(t, "/v1/shelves"),
					},
					{
						HttpMethod:  util.POST,
						UriTemplate: parseUriTemplate(t, "/endpoints.examples.bookstore.Bookstore/ListShelves"),
					},
				},
				"endpoints.examples.bookstore.Bookstore.CreateShelf": {
					{
						HttpMethod:  util.POST,
						UriTemplate: parseUriTemplate(t, "/v2/shelves"),
					},
					{
						HttpMethod:  util.POST,
						UriTemplate: parseUriTemplate(t, "/endpoints.examples.bookstore.Bookstore/CreateShelf"),
					},
				},
			},
		},
		{
			name: "grpc_service_http_rule_additional_bindings",
			serviceConfig: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name: "ListShelves",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/v1/shelves",
							},
							AdditionalBindings: []*annotationspb.HttpRule{
								{
									Pattern: &annotationspb.HttpRule_Get{
										Get: "/v2/stores/{store_id}/shelves",
									},
								},
								{
									Pattern: &annotationspb.HttpRule_Get{
										Get: "/v3/stores/{store_id}/shelves",
									},
								},
							},
						},
					},
				},
			},
			opts: options.ConfigGeneratorOptions{
				BackendAddress: "grpc://127.0.0.1:80",
			},
			want: map[string][]*httppattern.Pattern{
				"endpoints.examples.bookstore.Bookstore.ListShelves": {
					{
						HttpMethod:  util.GET,
						UriTemplate: parseUriTemplate(t, "/v1/shelves"),
					},
					{
						HttpMethod:  util.GET,
						UriTemplate: parseUriTemplate(t, "/v2/stores/{store_id}/shelves"),
					},
					{
						HttpMethod:  util.GET,
						UriTemplate: parseUriTemplate(t, "/v3/stores/{store_id}/shelves"),
					},
					{
						HttpMethod:  util.POST,
						UriTemplate: parseUriTemplate(t, "/endpoints.examples.bookstore.Bookstore/ListShelves"),
					},
				},
			},
		},
		{
			name: "grpc_service_http_rule_custom_method",
			serviceConfig: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name: "ListShelves",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
							Pattern: &annotationspb.HttpRule_Custom{
								Custom: &annotationspb.CustomHttpPattern{
									Kind: "CustomMethod",
									Path: "/v1/shelves",
								},
							},
						},
					},
				},
			},
			opts: options.ConfigGeneratorOptions{
				BackendAddress: "grpc://127.0.0.1:80",
			},
			want: map[string][]*httppattern.Pattern{
				"endpoints.examples.bookstore.Bookstore.ListShelves": {
					{
						HttpMethod:  "CustomMethod",
						UriTemplate: parseUriTemplate(t, "/v1/shelves"),
					},
					{
						HttpMethod:  util.POST,
						UriTemplate: parseUriTemplate(t, "/endpoints.examples.bookstore.Bookstore/ListShelves"),
					},
				},
			},
		},
		{
			name: "grpc_service_discovery_api_skipped",
			serviceConfig: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Apis: []*apipb.Api{
					{
						Name: "google.discovery.Discovery",
						Methods: []*apipb.Method{
							{
								Name: "GetDiscoveryRest",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "google.discovery.Discovery.GetDiscoveryRest",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/$discovery",
							},
						},
					},
				},
			},
			opts: options.ConfigGeneratorOptions{
				BackendAddress: "grpc://127.0.0.1:80",
			},
			want: map[string][]*httppattern.Pattern{},
		},
		{
			name: "grpc_service_discovery_api_allowed_by_option",
			serviceConfig: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Apis: []*apipb.Api{
					{
						Name: "google.discovery.Discovery",
						Methods: []*apipb.Method{
							{
								Name: "GetDiscoveryRest",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "google.discovery.Discovery.GetDiscoveryRest",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/$discovery",
							},
						},
					},
				},
			},
			opts: options.ConfigGeneratorOptions{
				BackendAddress:     "grpc://127.0.0.1:80",
				AllowDiscoveryAPIs: true,
			},
			want: map[string][]*httppattern.Pattern{
				"google.discovery.Discovery.GetDiscoveryRest": {
					{
						HttpMethod:  util.GET,
						UriTemplate: parseUriTemplate(t, "/$discovery"),
					},
					{
						HttpMethod:  util.POST,
						UriTemplate: parseUriTemplate(t, "/google.discovery.Discovery/GetDiscoveryRest"),
					},
				},
			},
		},
		{
			name: "http_service_no_http_rules",
			serviceConfig: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name: "ListShelves",
							},
							{
								Name: "CreateShelf",
							},
						},
					},
				},
			},
			opts: options.ConfigGeneratorOptions{
				BackendAddress: "http://127.0.0.1:80",
			},
			want: map[string][]*httppattern.Pattern{},
		},
		{
			name: "http_service_http_rule",
			serviceConfig: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name: "ListShelves",
							},
							{
								Name: "CreateShelf",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/v1/shelves",
							},
						},
						{
							Selector: "endpoints.examples.bookstore.Bookstore.CreateShelf",
							Pattern: &annotationspb.HttpRule_Post{
								Post: "/v2/shelves",
							},
							Body: "shelf",
						},
					},
				},
			},
			opts: options.ConfigGeneratorOptions{
				BackendAddress: "http://127.0.0.1:80",
			},
			want: map[string][]*httppattern.Pattern{
				"endpoints.examples.bookstore.Bookstore.ListShelves": {
					{
						HttpMethod:  util.GET,
						UriTemplate: parseUriTemplate(t, "/v1/shelves"),
					},
				},
				"endpoints.examples.bookstore.Bookstore.CreateShelf": {
					{
						HttpMethod:  util.POST,
						UriTemplate: parseUriTemplate(t, "/v2/shelves"),
					},
				},
			},
		},
	}

	for _, tc := range testdata {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseHTTPPatternsBySelectorFromOPConfig(tc.serviceConfig, tc.opts)
			if err != nil {
				t.Fatalf("ParseHTTPPatternsBySelectorFromOPConfig(...) got err %v, want no err", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ParseHTTPPatternsBySelectorFromOPConfig(...) diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseDeadlineSelectorFromOPConfig(t *testing.T) {
	testdata := []struct {
		desc          string
		serviceConfig *servicepb.Service
		opts          options.ConfigGeneratorOptions
		want          map[string]*DeadlineSpecifier
	}{
		{
			desc: "Mixed deadlines across multiple backend rules",
			serviceConfig: &servicepb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
					{
						Name: "cnn.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Address:  "grpc://abc.com/api/",
							Selector: "abc.com.api",
							Deadline: 10.5,
							OverridesByRequestProtocol: map[string]*servicepb.BackendRule{
								"http": {
									Address:  "http://http.abc.com/api/",
									Deadline: 20.5,
								},
							},
						},
						{
							Address:  "grpc://cnn.com/api/",
							Selector: "cnn.com.api",
							Deadline: 20,
						},
					},
				},
			},
			want: map[string]*DeadlineSpecifier{
				"abc.com.api": {
					Deadline:            10*time.Second + 500*time.Millisecond,
					HTTPBackendDeadline: 20*time.Second + 500*time.Millisecond,
				},
				"cnn.com.api": {
					Deadline: 20 * time.Second,
				},
			},
		},
		{
			desc: "Deadline with high precision is rounded to milliseconds",
			serviceConfig: &servicepb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Address:  "grpc://abc.com/api/",
							Selector: "abc.com.api",
							Deadline: 30.0009, // 30s 0.9ms
						},
					},
				},
			},
			want: map[string]*DeadlineSpecifier{
				"abc.com.api": {
					Deadline: 30*time.Second + 1*time.Millisecond,
				},
			},
		},
		{
			desc: "Deadline that is non-positive is overridden to default",
			serviceConfig: &servicepb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Address:  "grpc://abc.com/api/",
							Selector: "abc.com.api",
							Deadline: -10.5,
						},
					},
				},
			},
			want: map[string]*DeadlineSpecifier{
				"abc.com.api": {
					Deadline: 0,
				},
			},
		},
		{
			desc: "Missing deadline",
			serviceConfig: &servicepb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
			},
			want: map[string]*DeadlineSpecifier{},
		},
		{
			desc: "Deadlines parsed like normal for streaming methods",
			serviceConfig: &servicepb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name:              "api",
								ResponseStreaming: true,
							},
						},
					},
				},
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Address:  "grpc://abc.com/api/",
							Selector: "abc.com.api",
							Deadline: 10.5,
						},
					},
				},
			},
			want: map[string]*DeadlineSpecifier{
				"abc.com.api": {
					Deadline: 10*time.Second + 500*time.Millisecond,
				},
			},
		},
	}

	for _, tc := range testdata {
		t.Run(tc.desc, func(t *testing.T) {
			got := ParseDeadlineSelectorFromOPConfig(tc.serviceConfig, tc.opts)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ParseDeadlineSelectorFromOPConfig(...) diff (-want +got):\n%s", diff)
			}
		})
	}
}

func parseUriTemplate(t *testing.T, input string) *httppattern.UriTemplate {
	t.Helper()
	u, err := httppattern.ParseUriTemplate(input)
	if err != nil {
		t.Fatalf("fail to parse URI template %q, got err %v, want no err", input, err)
	}
	return u
}
