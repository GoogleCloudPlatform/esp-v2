package routegen

import (
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/google/go-cmp/cmp"
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
