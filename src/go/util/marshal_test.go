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
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	scpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

func TestUnmarshalBytesToPbMessage(t *testing.T) {
	_test := func(wantResp, getResp proto.Message, wantError string) {
		bytes, _ := proto.Marshal(wantResp)
		err := UnmarshalBytesToPbMessage(bytes, getResp)
		if err != nil {
			if !strings.Contains(err.Error(), wantError) {
				t.Errorf("fail in UnmarshalBytesToPbMessage on %T, want error: %s, get error: %v", wantResp, wantError, err)
			}
			return
		}
		if !proto.Equal(getResp, wantResp) {
			t.Errorf("fail in UnmarshalBytesToPbMessage on %T, want: %v, ge: %v", wantResp, wantResp, getResp)
		}
	}

	testCases := []struct {
		getResp   proto.Message
		wantResp  proto.Message
		wantError string
	}{
		{
			getResp: &confpb.Service{},
			wantResp: &confpb.Service{
				Id: "test-id",
			},
		},
		{
			getResp: &smpb.ListServiceRolloutsResponse{},
			wantResp: &smpb.ListServiceRolloutsResponse{
				NextPageToken: "next-page-token",
			},
		},
		{
			getResp: &scpb.ReportResponse{},
			wantResp: &scpb.ReportResponse{
				ServiceConfigId: "test-id",
			},
		},
		{
			getResp:   &scpb.ReportRequest{},
			wantError: "not support unmarshalling",
		},
	}

	for _, tc := range testCases {
		_test(tc.getResp, tc.wantResp, tc.wantError)
	}
}
