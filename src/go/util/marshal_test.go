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
	"testing"

	"github.com/golang/protobuf/proto"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	scpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

func TestUnmarshalBytesToPbMessage(t *testing.T) {
	var wantResp, getResp proto.Message
	wantError := ""

	_test := func() {
		bytes, _ := proto.Marshal(wantResp)
		err := UnmarshalBytesToPbMessage(bytes, getResp)
		if err != nil {
			if err.Error() != wantError {
				t.Errorf("fail in UnmarshalBytesToPbMessage on %T, want error: %s, get error: %v", wantResp, wantError, err)
			}
			return
		}
		if !proto.Equal(getResp, wantResp) {
			t.Errorf("fail in UnmarshalBytesToPbMessage on %T, want: %v, ge: %v", wantResp, wantResp, getResp)
		}
	}

	wantResp = &confpb.Service{
		Id: "test-id",
	}
	getResp = &confpb.Service{}
	_test()

	wantResp = &smpb.ListServiceRolloutsResponse{
		NextPageToken: "next-page-token",
	}
	getResp = &smpb.ListServiceRolloutsResponse{}
	_test()

	wantResp = &scpb.ReportResponse{
		ServiceConfigId: "test-id",
	}
	getResp = &scpb.ReportResponse{}
	_test()

	wantResp = &scpb.ReportRequest{
		ServiceConfigId: "test-id",
	}
	getResp = &scpb.ReportRequest{}
	wantError = "not support unmarshalling *servicecontrol.ReportRequest"
	_test()
}
