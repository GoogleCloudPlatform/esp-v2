package client

import (
	"bytes"
	"io"
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

func TestEncodeGrpcWebRequestBody(t *testing.T) {
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
		r := EncodeGrpcWebRequestBody(tc.reqMsg)
		encodedReqBody := readerToString(r)
		if encodedReqBody != tc.expectedReqBody {
			t.Errorf("Actual: %v. Expected: %v", encodedReqBody, tc.expectedReqBody)
		}
	}
}
