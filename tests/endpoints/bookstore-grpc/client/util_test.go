package client

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"

	bspb "cloudesf.googlesource.com/gcpproxy/tests/endpoints/bookstore-grpc/proto"
	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/proto"
)

func readerToString(r io.Reader) string {
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestEncodeGRPCWebRequestBody(t *testing.T) {
	testCases := []struct {
		reqMsg proto.Message
		// base64-encoded request body that will be sent by a gRPC-Web client.
		expectedReqBody string
	}{
		{
			reqMsg:          &types.Empty{},
			expectedReqBody: "AAAAAAA=",
		},

		{
			reqMsg:          &bspb.GetShelfRequest{Shelf: 123},
			expectedReqBody: "AAAAAAIIew==",
		},
		{
			reqMsg: &bspb.CreateBookRequest{
				Shelf: 123,
				Book: &bspb.Book{
					Id:     42,
					Author: "J. D. Salinger",
					Title:  "The Catcher in the Rye",
				}},
			expectedReqBody: "AAAAAC4IexIqCCoSDkouIEQuIFNhbGluZ2VyGhZUaGUgQ2F0Y2hlciBpbiB0aGUgUnll",
		},
	}

	for _, tc := range testCases {
		r := EncodeGRPCWebRequestBody(tc.reqMsg)
		encodedReqBody := readerToString(r)
		if encodedReqBody != tc.expectedReqBody {
			t.Errorf("Actual: %v. Expected: %v", encodedReqBody, tc.expectedReqBody)
		}
	}
}

func TestDecodeGRPCWebResponseBody(t *testing.T) {
	successTrailer := GRPCWebTrailer{"grpc-message": "OK", "grpc-status": "0"}
	testCases := []struct {
		desc string
		// base64-encoded response body received by a gRPC-Web client.
		encodedRespBody string
		actualMsg       proto.Message
		expectedMsg     proto.Message
		expectedTrailer GRPCWebTrailer
	}{
		{
			desc:            "ListShelves",
			encodedRespBody: "AAAAABwKDgh7EgpTaGFrc3BlYXJlCgoIfBIGSGFtbGV0gAAAACBncnBjLXN0YXR1czowDQpncnBjLW1lc3NhZ2U6T0sNCg==",
			actualMsg:       &bspb.ListShelvesResponse{},
			expectedMsg: &bspb.ListShelvesResponse{
				Shelves: []*bspb.Shelf{
					&bspb.Shelf{
						Id:    123,
						Theme: "Shakspeare",
					},
					&bspb.Shelf{
						Id:    124,
						Theme: "Hamlet",
					},
				},
			},
			expectedTrailer: successTrailer,
		},
		{
			desc:            "GetShelf",
			encodedRespBody: "AAAAABEIexINVW5rbm93biBTaGVsZg==gAAAACBncnBjLXN0YXR1czowDQpncnBjLW1lc3NhZ2U6T0sNCg==",
			actualMsg:       &bspb.Shelf{},
			expectedMsg: &bspb.Shelf{
				Id:    123,
				Theme: "Unknown Shelf",
			},
			expectedTrailer: successTrailer,
		},
		{
			desc:            "CreateBook",
			encodedRespBody: "AAAAAAwIexoITmV3IEJvb2s=gAAAACBncnBjLXN0YXR1czowDQpncnBjLW1lc3NhZ2U6T0sNCg==",
			actualMsg:       &bspb.Book{},
			expectedMsg: &bspb.Book{
				Id:    123,
				Title: "New Book",
			},
			expectedTrailer: successTrailer,
		},
	}

	for _, tc := range testCases {
		binaryMsg, trailer, err := DecodeGRPCWebResponseBody(strings.NewReader(tc.encodedRespBody))
		if err != nil {
			t.Errorf("%s failed with error: %v,", tc.desc, err)
			continue
		}

		err = proto.Unmarshal(binaryMsg, tc.actualMsg)
		if err != nil {
			t.Errorf("%s failed with error: %v,", tc.desc, err)
			continue
		}

		if !proto.Equal(tc.actualMsg, tc.expectedMsg) {
			t.Errorf("%s failed:\nActual:%v\nExpected:%v", tc.desc, tc.actualMsg, tc.expectedMsg)
		}

		if !reflect.DeepEqual(trailer, tc.expectedTrailer) {
			t.Errorf("%s failed:\nActual:%v\nExpected:%v", tc.desc, trailer, tc.expectedTrailer)
		}
	}
}
