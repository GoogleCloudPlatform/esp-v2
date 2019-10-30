// Copyright 2019 Google LLC
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

package static

import (
	"fmt"
	"strings"
	"testing"

	"cloudesf.googlesource.com/gcpproxy/src/go/bootstrap/static/testdata"
	"cloudesf.googlesource.com/gcpproxy/src/go/options"
	"cloudesf.googlesource.com/gcpproxy/src/go/util"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"

	bootstrappb "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
)

func TestServiceToBootstrapConfig(t *testing.T) {
	opts := options.DefaultConfigGeneratorOptions()
	opts.BackendProtocol = "HTTP1"

	// Function under test
	gotBootstrap, err := ServiceToBootstrapConfig(testdata.FakeBookstoreConfig, testdata.FakeConfigID, opts)
	if err != nil {
		t.Fatal(err)
	}

	if err := verifyBootstrapConfig(gotBootstrap, testdata.ExpectedBookstoreEnvoyConfig); err != nil {
		t.Fatalf("Normal ServiceToBootstrapConfig error: %v", err)
	}
}

func verifyBootstrapConfig(got *bootstrappb.Bootstrap, want string) error {
	unmarshaler := &jsonpb.Unmarshaler{
		AnyResolver: util.Resolver,
	}

	// Convert want string to a proto to compare with got
	wantReader := strings.NewReader(want)
	wantBootstrap := &bootstrappb.Bootstrap{}
	err := unmarshaler.Unmarshal(wantReader, wantBootstrap)
	if err != nil {
		return err
	}

	if !proto.Equal(got, wantBootstrap) {
		// Marshal both protos back to json-strings to pretty print them
		marshaler := &jsonpb.Marshaler{
			AnyResolver: util.Resolver,
		}
		gotString, err := marshaler.MarshalToString(got)
		if err != nil {
			return err
		}
		wantString, err := marshaler.MarshalToString(wantBootstrap)
		if err != nil {
			return err
		}
		return fmt.Errorf("\ngot : %v, \nwant: %v", gotString, wantString)
	}

	return nil
}
