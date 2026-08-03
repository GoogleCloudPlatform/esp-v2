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

package util

import (
	"testing"

	httppb "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
)

func TestCreateUpstreamProtocolOptionsIncludesKeepalive(t *testing.T) {
	options, ok := CreateUpstreamProtocolOptions()[UpstreamProtocolOptions]
	if !ok {
		t.Fatalf("CreateUpstreamProtocolOptions() did not return %q", UpstreamProtocolOptions)
	}

	httpOptions := &httppb.HttpProtocolOptions{}
	if err := options.UnmarshalTo(httpOptions); err != nil {
		t.Fatalf("options.UnmarshalTo(%T) failed: %v", httpOptions, err)
	}

	keepalive := httpOptions.GetExplicitHttpConfig().GetHttp2ProtocolOptions().GetConnectionKeepalive()
	if keepalive == nil {
		t.Fatal("CreateUpstreamProtocolOptions() did not configure connection keepalive")
	}
	if got := keepalive.GetInterval().AsDuration(); got != Http2KeepaliveInterval {
		t.Errorf("keepalive interval = %v, want %v", got, Http2KeepaliveInterval)
	}
	if got := keepalive.GetTimeout().AsDuration(); got != Http2KeepaliveTimeout {
		t.Errorf("keepalive timeout = %v, want %v", got, Http2KeepaliveTimeout)
	}
}
