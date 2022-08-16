// Copyright 2021 Google LLC
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

package filterconfig

import (
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
	descpb "github.com/golang/protobuf/protoc-gen-go/descriptor"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

func TestUpdateProtoDescriptor(t *testing.T) {
	testData := []struct {
		desc      string
		service   string
		apiNames  []string
		inDesc    string
		wantDesc  string
		wantError string
	}{
		{
			// The input Descriptor is an invalid data, it results in error.
			desc:      "Failed to unmarshal error",
			service:   "",
			inDesc:    "invalid proto descriptor",
			wantError: "failed to unmarshal",
		},
		{
			// ApiNames is a wrong service name, protoDescriptor is not modified.
			desc: "Wrong apiName, not override",
			service: `
http: {
  rules: {
    selector: "package.name.Service.Method"
    post: "/v2/{name=*}"
  }
}`,
			apiNames: []string{"package.name.WrongService"},
			inDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
      options: {
        [google.api.http]: {
	        post: "/v1/{name=*}"
	      }
      }
    }
  }
}`,
			wantDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
      options: {
        [google.api.http]: {
	        post: "/v1/{name=*}"
	      }
      }
    }
  }
}`,
		},
		{
			// ProtoDescriptor doesn't have MethodOptions, the http rule is copied with default rule added in its additional bindings
			desc: "Not method options",
			service: `
http: {
  rules: {
    selector: "package.name.Service.Method"
    post: "/v2/{name=*}"
  }
}`,
			apiNames: []string{"package.name.Service"},
			inDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
    }
  }
}`,
			wantDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
      options: {
        [google.api.http]: {
          selector: "package.name.Service.Method"
	        post: "/v2/{name=*}"
          additional_bindings: {
            put: "/package.name.Service/Method"
            body: "*"
          }
	      }
      }
    }
  }
}`,
		},
		{
			// ProtoDescriptor has an empty MethodOptions, the http rule is copied with default rule added in its additional bindings
			desc: "Empty method options",
			service: `
http: {
  rules: {
    selector: "package.name.Service.Method"
    post: "/v2/{name=*}"
  }
}`,
			apiNames: []string{"package.name.Service"},
			inDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
      options: {
      }
    }
  }
}`,
			wantDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
      options: {
        [google.api.http]: {
          selector: "package.name.Service.Method"
	        post: "/v2/{name=*}"
          additional_bindings: {
            put: "/package.name.Service/Method"
            body: "*"
          }
	      }
      }
    }
  }
}`,
		},
		{
			// ProtoDescriptor has a different annotation, the http rule is copied with default rule added in its additional bindings
			desc: "Basic overwritten case",
			service: `
http: {
  rules: {
    selector: "package.name.Service.Method"
    post: "/v2/{name=*}"
  }
}`,
			apiNames: []string{"package.name.Service"},
			inDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
      options: {
        [google.api.http]: {
	        post: "/v1/abc/{name=*}"
	      }
      }
    }
  }
}`,
			wantDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
      options: {
        [google.api.http]: {
          selector: "package.name.Service.Method"
	        post: "/v2/{name=*}"
          additional_bindings: {
            put: "/package.name.Service/Method"
            body: "*"
          }
	      }
      }
    }
  }
}`,
		},
		{
			// The http rule has a different service name. It is not copied
			desc: "Empty http rule as it has different service name",
			service: `
http: {
  rules: {
    selector: "package.name.WrongService.Method"
    post: "/v2/{name=*}"
  }
}`,
			apiNames: []string{"package.name.Service"},
			inDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
      options: {
        [google.api.http]: {
	        post: "/v1/abc/{name=*}"
	      }
      }
    }
  }
}`,
			wantDesc: `
file: {
  name: "proto_file_path"
  package: "package.name"
  service: {
    name: "Service"
    method: {
      name: "Method"
      options: {
        [google.api.http]: {
	        post: "/v1/abc/{name=*}"
	      }
      }
    }
  }
}`,
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			serviceConfig := &confpb.Service{}
			if err := prototext.Unmarshal([]byte(tc.service), serviceConfig); err != nil {
				t.Fatal("failed to unmarshal service config: ", err)
			}

			var byteDesc []byte
			fds := &descpb.FileDescriptorSet{}
			if err := prototext.Unmarshal([]byte(tc.inDesc), fds); err != nil {
				// Failed case is to use raw test to test failure
				byteDesc = []byte(tc.inDesc)
			} else {
				byteDesc, _ = proto.Marshal(fds)
			}

			gotByteDesc, err := updateProtoDescriptor(serviceConfig, tc.apiNames, byteDesc)
			if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Errorf("failed, expected: %s, got: %v", tc.wantError, err)
			}
			if tc.wantError == "" && err != nil {
				t.Errorf("got unexpected error: %v", err)
			}

			if tc.wantDesc != "" {
				got := &descpb.FileDescriptorSet{}
				// Not need to check error, gotByteDesc is just marshaled from the updateProtoDescriptor()
				proto.Unmarshal(gotByteDesc, got)
				want := &descpb.FileDescriptorSet{}
				if err := prototext.Unmarshal([]byte(tc.wantDesc), want); err != nil {
					t.Fatal("failed to unmarshal wantDesc: ", err)
				}

				if diff := utils.ProtoDiff(want, got); diff != "" {
					t.Errorf("Result is not the same: diff (-want +got):\n%v", diff)
				}
			}
		})
	}
}
