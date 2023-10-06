package filtergen

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/google/go-cmp/cmp"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
	apipb "google.golang.org/genproto/protobuf/api"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	descpb "google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
)

// validDescriptors is a list of fake file descriptors for testing.
var validDescriptors = &descpb.FileDescriptorSet{
	File: []*descpb.FileDescriptorProto{
		{
			Package: proto.String("google.library.v1"),
			Name:    proto.String("google/library/v1/service.proto"),
			Service: []*descpb.ServiceDescriptorProto{
				{
					Name: proto.String("Bookstore"),
					Method: []*descpb.MethodDescriptorProto{
						{
							Name: proto.String("BuyBook"),
						},
					},
				},
				{
					Name: proto.String("Library"),
					Method: []*descpb.MethodDescriptorProto{
						{
							Name: proto.String("ListBooks"),
						},
						{
							Name: proto.String("LoanBook"),
						},
					},
				},
			},
		},
	},
}

func TestIsGRPCSupportRequiredForOPConfig(t *testing.T) {
	testdata := []struct {
		desc            string
		serviceConfigIn *servicepb.Service
		optsIn          options.ConfigGeneratorOptions
		wantIsGRPC      bool
	}{
		{
			desc:            "From backend address",
			serviceConfigIn: &servicepb.Service{},
			optsIn: options.ConfigGeneratorOptions{
				BackendAddress: "grpc://127.0.0.1:8090",
			},
			wantIsGRPC: true,
		},
		{
			desc: "From backend rule",
			serviceConfigIn: &servicepb.Service{
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Address: "http://backend-1.com",
						},
						{
							Address: "grpcs://backend-2.com",
						},
						{
							Address: "https://backend-3.com",
						},
					},
				},
			},
			wantIsGRPC: true,
		},
		{
			desc: "All http",
			serviceConfigIn: &servicepb.Service{
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Address: "http://backend-1.com",
						},
						{
							Address: "http://backend-2.com",
						},
						{
							Address: "https://backend-3.com",
						},
					},
				},
			},
			optsIn: options.ConfigGeneratorOptions{
				BackendAddress: "http://127.0.0.1:8090",
			},
			wantIsGRPC: false,
		},
		{
			desc: "Discovery API is skipped",
			serviceConfigIn: &servicepb.Service{
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Selector: "google.library.Bookstore.GetBooks",
							Address:  "http://backend-1.com",
						},
						{
							Selector: "google.discovery.GetDiscoveryRest",
							Address:  "grpcs://backend-2.com",
						},
						{
							Selector: "google.library.Bookstore.GetShelves",
							Address:  "https://backend-3.com",
						},
					},
				},
			},
			wantIsGRPC: false,
		},
		{
			desc: "Backend rules skipped when backend address override is enabled",
			serviceConfigIn: &servicepb.Service{
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Address: "http://backend-1.com",
						},
						{
							Address: "grpcs://backend-2.com",
						},
						{
							Address: "https://backend-3.com",
						},
					},
				},
			},
			optsIn: options.ConfigGeneratorOptions{
				EnableBackendAddressOverride: true,
			},
			wantIsGRPC: false,
		},
	}

	for _, tc := range testdata {
		t.Run(tc.desc, func(t *testing.T) {
			gotIsGRPC, err := IsGRPCSupportRequiredForOPConfig(tc.serviceConfigIn, tc.optsIn)
			if err != nil {
				t.Fatalf("IsGRPCSupportRequiredForOPConfig() got unexpected error: %v", err)
			}

			if gotIsGRPC != tc.wantIsGRPC {
				t.Errorf("IsGRPCSupportRequiredForOPConfig() got %v, want %v", gotIsGRPC, tc.wantIsGRPC)
			}
		})
	}
}

func TestGetUsageRulesBySelectorFromOPConfig(t *testing.T) {
	testdata := []struct {
		desc            string
		serviceConfigIn *servicepb.Service
		optsIn          options.ConfigGeneratorOptions
		want            map[string]*servicepb.UsageRule
	}{
		{
			desc: "Usage rules parsed, discovery APIs skipped by default",
			serviceConfigIn: &servicepb.Service{
				Usage: &servicepb.Usage{
					Rules: []*servicepb.UsageRule{
						{
							Selector: "google.library.Bookstore.GetBook",
						},
						{
							Selector:           "google.library.Bookstore.ListBooks",
							SkipServiceControl: true,
						},
						{
							Selector:               "google.library.Bookstore.CreateBook",
							AllowUnregisteredCalls: true,
						},
						{
							Selector:           "google.discovery.GetDiscoveryRest",
							SkipServiceControl: true,
						},
					},
				},
			},
			want: map[string]*servicepb.UsageRule{
				"google.library.Bookstore.GetBook": {
					Selector: "google.library.Bookstore.GetBook",
				},
				"google.library.Bookstore.ListBooks": {
					Selector:           "google.library.Bookstore.ListBooks",
					SkipServiceControl: true,
				},
				"google.library.Bookstore.CreateBook": {
					Selector:               "google.library.Bookstore.CreateBook",
					AllowUnregisteredCalls: true,
				},
			},
		},
		{
			desc: "Health check methods are modified to skip service control by default",
			serviceConfigIn: &servicepb.Service{
				Apis: []*apipb.Api{
					{
						Name: "grpc.health.v1.Health",
						Methods: []*apipb.Method{
							{
								Name: "Check",
							},
							{
								Name: "Watch",
							},
						},
					},
				},
			},
			want: map[string]*servicepb.UsageRule{
				"grpc.health.v1.Health.Check": {
					Selector:           "grpc.health.v1.Health.Check",
					SkipServiceControl: true,
				},
				"grpc.health.v1.Health.Watch": {
					Selector:           "grpc.health.v1.Health.Watch",
					SkipServiceControl: true,
				},
			},
		},
		{
			desc: "User overrides hardcoded skip service control with usage rule",
			serviceConfigIn: &servicepb.Service{
				Apis: []*apipb.Api{
					{
						Name: "grpc.health.v1.Health",
						Methods: []*apipb.Method{
							{
								Name: "Check",
							},
						},
					},
				},
				Usage: &servicepb.Usage{
					Rules: []*servicepb.UsageRule{
						{
							Selector:           "grpc.health.v1.Health.Check",
							SkipServiceControl: false,
						},
					},
				},
			},
			want: map[string]*servicepb.UsageRule{
				"grpc.health.v1.Health.Check": {
					Selector:           "grpc.health.v1.Health.Check",
					SkipServiceControl: false,
				},
			},
		},
	}

	for _, tc := range testdata {
		t.Run(tc.desc, func(t *testing.T) {
			got := GetUsageRulesBySelectorFromOPConfig(tc.serviceConfigIn, tc.optsIn)
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("GetUsageRulesBySelectorFromOPConfig() diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetAPIKeySystemParametersBySelectorFromOPConfig(t *testing.T) {
	testdata := []struct {
		desc            string
		serviceConfigIn *servicepb.Service
		optsIn          options.ConfigGeneratorOptions
		want            map[string][]*servicepb.SystemParameter
	}{
		{
			desc: "Url and query API Keys, ignore non-API key parameters",
			serviceConfigIn: &servicepb.Service{
				SystemParameters: &servicepb.SystemParameters{
					Rules: []*servicepb.SystemParameterRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echo",
							Parameters: []*servicepb.SystemParameter{
								{
									Name:       "other",
									HttpHeader: "non_api_key_header",
								},
								{
									Name:       "api_key",
									HttpHeader: "header_name",
								},
								{
									Name:              "other",
									UrlQueryParameter: "non_api_key_query",
								},
								{
									Name:              "api_key",
									UrlQueryParameter: "query_name",
								},
								{
									Name:              "other",
									HttpHeader:        "combined_non_api_key_header",
									UrlQueryParameter: "combined_non_api_key_query",
								},
								{
									Name:              "api_key",
									HttpHeader:        "combined_header_name",
									UrlQueryParameter: "combined_query_header_name",
								},
							},
						},
					},
				},
			},
			want: map[string][]*servicepb.SystemParameter{
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.echo": {
					{
						Name:       "api_key",
						HttpHeader: "header_name",
					},
					{
						Name:              "api_key",
						UrlQueryParameter: "query_name",
					},
					{
						Name:              "api_key",
						HttpHeader:        "combined_header_name",
						UrlQueryParameter: "combined_query_header_name",
					},
				},
			},
		},
		{
			desc: "Multiple APIs with default API Key",
			serviceConfigIn: &servicepb.Service{
				SystemParameters: &servicepb.SystemParameters{
					Rules: []*servicepb.SystemParameterRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.foo",
							Parameters: []*servicepb.SystemParameter{
								{
									Name:              "api_key",
									HttpHeader:        "header_name_1",
									UrlQueryParameter: "query_name_1",
								},
								{
									Name:              "api_key",
									HttpHeader:        "header_name_2",
									UrlQueryParameter: "query_name_2",
								},
							},
						},
						{
							Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.bar",
							Parameters: []*servicepb.SystemParameter{
								{
									Name:              "api_key",
									HttpHeader:        "header_name_1",
									UrlQueryParameter: "query_name_1",
								},
								{
									Name:              "api_key",
									HttpHeader:        "header_name_2",
									UrlQueryParameter: "query_name_2",
								},
							},
						},
					},
				},
			},
			want: map[string][]*servicepb.SystemParameter{
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.foo": {
					{
						Name:              "api_key",
						HttpHeader:        "header_name_1",
						UrlQueryParameter: "query_name_1",
					},
					{
						Name:              "api_key",
						HttpHeader:        "header_name_2",
						UrlQueryParameter: "query_name_2",
					},
				},
				"2.echo_api_endpoints_cloudesf_testing_cloud_goog.bar": {
					{
						Name:              "api_key",
						HttpHeader:        "header_name_1",
						UrlQueryParameter: "query_name_1",
					},
					{
						Name:              "api_key",
						HttpHeader:        "header_name_2",
						UrlQueryParameter: "query_name_2",
					},
				},
			},
		},
	}

	for _, tc := range testdata {
		t.Run(tc.desc, func(t *testing.T) {
			got := GetAPIKeySystemParametersBySelectorFromOPConfig(tc.serviceConfigIn, tc.optsIn)
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("GetAPIKeySystemParametersBySelectorFromOPConfig() diff (-want +got):\n%s", diff)
			}
		})
	}
}

func setupSourceFileContent(rawDescriptor []byte) (*anypb.Any, error) {
	sourceFile := &smpb.ConfigFile{
		FilePath:     "api_descriptor.pb",
		FileContents: rawDescriptor,
		FileType:     smpb.ConfigFile_FILE_DESCRIPTOR_SET_PROTO,
	}
	content, err := anypb.New(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal source file into any: %v", err)
	}
	return content, nil
}

func setupInvalidSourceFileContent(rawDescriptor []byte) *anypb.Any {
	content := &anypb.Any{Value: rawDescriptor}
	return content
}

func TestGetDescriptorBinFromOPConfigSuccessCase(t *testing.T) {
	rawDescriptor, err := proto.Marshal(validDescriptors)
	if err != nil {
		t.Fatalf("Failed to marshal FileDescriptorSet: %v", err)
	}
	content, err := setupSourceFileContent(rawDescriptor)
	if err != nil {
		t.Fatalf("Failed to setup source file content: %v", err)
	}

	invalidRawDescriptor := []byte("invalid_random_descriptor")
	validContentwithInvalidProto, err := setupSourceFileContent(invalidRawDescriptor)
	if err != nil {
		t.Fatalf("Failed to setup source file content: %v", err)
	}

	testdata := []struct {
		desc            string
		serviceConfigIn *servicepb.Service
		descriptorBin   []byte
	}{
		{
			desc: "Successfully extracted the descriptor bytes from the OP service config.",
			serviceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				SourceInfo: &servicepb.SourceInfo{
					SourceFiles: []*anypb.Any{content},
				},
			},
			descriptorBin: rawDescriptor,
		},
		{
			desc: "Success in extracting descriptor bytes from the OP service config. Valid content with invalid proto.",
			serviceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				SourceInfo: &servicepb.SourceInfo{
					SourceFiles: []*anypb.Any{validContentwithInvalidProto},
				},
			},
			descriptorBin: invalidRawDescriptor,
		},
	}
	for _, tc := range testdata {
		t.Run(tc.desc, func(t *testing.T) {
			descriptorBin, err := GetDescriptorBinFromOPConfig(tc.serviceConfigIn)
			if err != nil {
				t.Fatalf("GetDescriptorBinFromOPConfig() got error: %v", err)
			}
			if diff := cmp.Diff(tc.descriptorBin, descriptorBin); diff != "" {
				t.Fatalf("GetDescriptorBinFromOPConfig() got invalid descriptor. Got %v, want %v", descriptorBin, tc.descriptorBin)
			}
		})
	}
}

func TestGetDescriptorBinFromOPConfigFailureCase(t *testing.T) {
	invalidRawDescriptor := []byte("invalid_random_descriptor")
	invalidContentwithInvalidProto := setupInvalidSourceFileContent(invalidRawDescriptor)

	testdata := []struct {
		desc            string
		serviceConfigIn *servicepb.Service
	}{
		{
			desc: "Failure in extracting descriptor bytes from the OP service config. Invalid content with invalid proto.",
			serviceConfigIn: &servicepb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
				SourceInfo: &servicepb.SourceInfo{
					SourceFiles: []*anypb.Any{invalidContentwithInvalidProto},
				},
			},
		},
	}
	for _, tc := range testdata {
		t.Run(tc.desc, func(t *testing.T) {
			_, err := GetDescriptorBinFromOPConfig(tc.serviceConfigIn)
			if err == nil {
				t.Fatalf("GetDescriptorBinFromOPConfig() got error: %v, want nil", err)
			}
		})
	}
}
