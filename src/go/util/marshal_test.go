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

package util

import (
	"bytes"
	"strings"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	statspb "github.com/envoyproxy/go-control-plane/envoy/config/metrics/v3"
	accessfilepb "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	accessgrpcpb "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/grpc/v3"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	scpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

func TestResolver(t *testing.T) {
	tests := []struct {
		msg proto.Message
	}{
		{msg: &accessfilepb.FileAccessLog{}},
		{msg: &accessgrpcpb.HttpGrpcAccessLogConfig{}},
		{msg: &accessgrpcpb.TcpGrpcAccessLogConfig{}},
		{msg: &accessgrpcpb.CommonGrpcAccessLogConfig{}},
		{msg: &statspb.StatsSink{}},
		{msg: &statspb.StatsConfig{}},
	}

	marshaler := &jsonpb.Marshaler{
		OrigName:    true,
		AnyResolver: Resolver,
	}

	for _, tc := range tests {
		any, err := ptypes.MarshalAny(tc.msg)
		if err != nil {
			t.Fatalf("MarshalAny(%v) failed: %v", tc.msg, err)
		}
		buf := &bytes.Buffer{}
		if err := marshaler.Marshal(buf, any); err != nil {
			t.Errorf("Marshal(_, %v) failed: %v", any, err)
		}
	}
}

func TestUnmarshalBytesToPbMessage(t *testing.T) {
	testCases := []struct {
		desc          string
		wantResp      proto.Message
		getRespHolder proto.Message
		wantError     string
	}{
		{
			desc: "unmarshal Service",
			wantResp: &confpb.Service{
				Id: "test-id",
			},
			getRespHolder: &confpb.Service{},
		},
		{
			desc: "unmarshal ListServiceRolloutsResponse",
			wantResp: &smpb.ListServiceRolloutsResponse{
				NextPageToken: "next-page-token",
			},
			getRespHolder: &smpb.ListServiceRolloutsResponse{},
		},
		{
			desc: "unmarshal ReportResponse",
			wantResp: &scpb.ReportResponse{
				ServiceConfigId: "test-id",
			},
			getRespHolder: &scpb.ReportResponse{},
		},
		{
			desc:          "unmarshal ReportRequest",
			getRespHolder: &scpb.ReportRequest{},
			wantError:     "not support unmarshalling",
		},
	}

	for _, tc := range testCases {
		bytes, _ := proto.Marshal(tc.wantResp)
		err := UnmarshalBytesToPbMessage(bytes, tc.getRespHolder)
		if err != nil {
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Errorf("Test (%s): fail in UnmarshalBytesToPbMessage on %T, want error: %s, get error: %v", tc.desc, tc.wantResp, tc.wantError, err)
			}
			return
		}
		if !proto.Equal(tc.getRespHolder, tc.wantResp) {
			t.Errorf("Test (%s): fail in UnmarshalBytesToPbMessage on %T, want: %v, ge: %v", tc.desc, tc.wantResp, tc.wantResp, tc.getRespHolder)
		}
	}
}
