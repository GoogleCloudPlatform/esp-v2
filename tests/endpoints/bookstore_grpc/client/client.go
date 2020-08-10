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

package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	bsgrpcv1 "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/proto/v1"
	bspbv1 "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/proto/v1"
	bsgrpcv2 "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/proto/v2"
	bspbv2 "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/proto/v2"
)

var grpcWebHeader = http.Header{
	"X-User-Agent": []string{"grpc-web-javascript/0.1"},
	"Content-Type": []string{"application/grpc-web-text"},
	"Accept":       []string{"application/grpc-web-text"},
	"X-Grpc-Web":   []string{"1"},
}

const (
	// APIKeyHeaderKey is used to pass the API Key to the backend
	APIKeyHeaderKey  = "x-api-key"
	bookstoreService = "endpoints.examples.bookstore.Bookstore"
	// TestHeaderKey is used to force backend to return non-OK status.
	// Refer to tests/endpoints/bookstore_grpc/grpc_server.js for detail.
	TestHeaderKey = "x-grpc-test"
)

func addAllHeaders(req *http.Request, header http.Header) {
	for key, valueList := range header {
		for _, value := range valueList {
			(*req).Header.Add(key, value)
		}
	}
}

// MakeCall returns response in JSON.
// For gRPC-web, use MakeGRPCWebCall instead.
//
// For HTTP, add client.TestHeaderKey to header force the backend to return non-OK status.
func MakeCall(clientProtocol, addr, httpMethod, method, token string, header http.Header) (string, error) {
	if strings.EqualFold(clientProtocol, "http") {
		return makeHTTPCall(addr, httpMethod, method, token, header)
	}

	if strings.EqualFold(clientProtocol, "http2") {
		return makeHTTP2Call(addr, httpMethod, method, token, header)
	}
	if strings.EqualFold(clientProtocol, "grpc") {
		return makeGRPCCall(addr, method, token, header)
	}

	if strings.EqualFold(clientProtocol, "grpc-web") {
		return "", errors.New("Use MakeGRPCWebCall instead")
	}

	return "", fmt.Errorf("unsupported client protocol %s", clientProtocol)
}

var makeHTTPCall = func(addr, httpMethod, method, token string, header http.Header) (string, error) {
	var cli http.Client
	req, _ := http.NewRequest(httpMethod, fmt.Sprintf("http://%s%s", addr, method), nil)

	if token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	addAllHeaders(req, header)

	resp, err := cli.Do(req)
	if err != nil {
		return "", fmt.Errorf("http got error: %v", err)
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http response status is not 200 OK: %s, %s", resp.Status, utils.RpcStatusDeterministicJsonFormat(content))
	}

	return string(content), nil
}

//TODO(b/162626126): cleanup duplicate call methods.
func makeHTTP2Call(addr, httpMethod, method, token string, header http.Header) (string, error) {
	cli := http.Client{
		// Skip TLS dial
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}
	req, _ := http.NewRequest(httpMethod, fmt.Sprintf("http://%s%s", addr, method), nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	addAllHeaders(req, header)

	resp, err := cli.Do(req)
	if err != nil {
		return "", fmt.Errorf("http got error: %v", err)
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http response status is not 200 OK: %s, %s", resp.Status, utils.RpcStatusDeterministicJsonFormat(content))
	}

	return string(content), nil
}

var MakeHttpCallWithBody = func(addr, httpMethod, method, token string, bodyBytes []byte) (string, error) {
	var cli http.Client
	var req *http.Request

	if bodyBytes == nil {
		req, _ = http.NewRequest(httpMethod, fmt.Sprintf("http://%s%s", addr, method), nil)
	} else {
		req, _ = http.NewRequest(httpMethod, fmt.Sprintf("http://%s%s", addr, method), bytes.NewBuffer(bodyBytes))
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := cli.Do(req)
	if err != nil {
		return "", fmt.Errorf("http got error: %v", err)
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http response status is not 200 OK: %s, %s", resp.Status, utils.RpcStatusDeterministicJsonFormat(content))
	}

	return string(content), nil
}

var makeGRPCCall = func(addr, method, token string, header http.Header) (string, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to connect to server: %v", err)
	}
	defer conn.Close()

	cli := bsgrpcv1.NewBookstoreClient(conn)
	ctx := context.Background()
	if token != "" {
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("Authorization", fmt.Sprintf("Bearer %s", token)))
	}

	for key, valueList := range header {
		for _, value := range valueList {
			ctx = metadata.AppendToOutgoingContext(ctx, key, value)
		}
	}

	var respMsg proto.Message
	switch method {
	case "ListShelves":
		req := &bspbv1.Empty{}
		respMsg, err = cli.ListShelves(ctx, req)
	case "CreateShelf":
		req := &bspbv1.CreateShelfRequest{
			Shelf: &bspbv1.Shelf{
				Id:    14785,
				Theme: "New Shelf",
			},
		}
		respMsg, err = cli.CreateShelf(ctx, req)
	case "GetShelf":
		req := &bspbv1.GetShelfRequest{
			Shelf: 100,
		}
		respMsg, err = cli.GetShelf(ctx, req)
	case "CreateBook":
		req := &bspbv1.CreateBookRequest{
			Shelf: 200,
			Book: &bspbv1.Book{
				Id:    20050,
				Title: "Harry Potter",
			},
		}
		respMsg, err = cli.CreateBook(ctx, req)
	case "DeleteShelf":
		req := &bspbv1.DeleteShelfRequest{}
		respMsg, err = cli.DeleteShelf(ctx, req)
	default:
		return "", fmt.Errorf("unexpected method called")
	}

	if err != nil {
		statusErr := status.Convert(err)
		return "", fmt.Errorf("%v got unexpected error: code = %v, message = %v", method, statusErr.Code(), utils.RpcStatusDeterministicJsonFormat([]byte(statusErr.Message())))
	}

	var marshaler jsonpb.Marshaler
	return marshaler.MarshalToString(respMsg)
}

var MakeBookstoreV2GrpcCall = func(addr, method string, header http.Header) (string, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to connect to server: %v", err)
	}
	defer conn.Close()

	cli := bsgrpcv2.NewBookstoreClient(conn)
	ctx := context.Background()

	for key, valueList := range header {
		for _, value := range valueList {
			ctx = metadata.AppendToOutgoingContext(ctx, key, value)
		}
	}

	var respMsg proto.Message
	switch method {
	case "GetShelf":
		req := &bspbv2.GetShelfRequest{
			Shelf: 100,
		}
		respMsg, err = cli.GetShelf(ctx, req)
	default:
		return "", fmt.Errorf("unexpected method called")
	}

	if err != nil {
		return "", fmt.Errorf("%v got unexpected error: %v", method, err)
	}

	var marshaler jsonpb.Marshaler
	return marshaler.MarshalToString(respMsg)
}

// MakeGRPCWebCall returns response in JSON and gRPC-Web trailer.
// Add client.TestHeaderKey to header to force the backend to return a non-OK status.
func MakeGRPCWebCall(addr, method, token string, header http.Header) (string, GRPCWebTrailer, error) {
	var reqMsg proto.Message
	var respMsg proto.Message
	serviceName := bookstoreService
	switch method {
	case "ListShelves":
		reqMsg = &bspbv1.Empty{}
		respMsg = &bspbv1.ListShelvesResponse{}
	case "CreateShelf":
		reqMsg = &bspbv1.CreateShelfRequest{
			Shelf: &bspbv1.Shelf{
				Id:    14785,
				Theme: "New Shelf",
			},
		}
		respMsg = &bspbv1.Shelf{}
	case "GetShelf":
		reqMsg = &bspbv1.GetShelfRequest{
			Shelf: 100,
		}
		respMsg = &bspbv1.Shelf{}
	case "CreateBook":
		reqMsg = &bspbv1.CreateBookRequest{
			Book: &bspbv1.Book{
				Id:    20050,
				Title: "Harry Potter",
			},
		}
		respMsg = &bspbv1.Book{}
	case "DeleteShelf":
		reqMsg = &bspbv1.DeleteShelfRequest{}
		respMsg = &bspbv1.Empty{}
	default:
		return "", nil, fmt.Errorf("unexpected method called")
	}
	body := EncodeGRPCWebRequestBody(reqMsg)

	u, err := url.Parse("http://" + addr)
	if err != nil {
		return "", nil, err
	}
	u.Path = path.Join(u.Path, serviceName, method)
	req, err := http.NewRequest("POST", u.String(), body)
	if err != nil {
		return "", nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	addAllHeaders(req, header)
	addAllHeaders(req, grpcWebHeader)

	var client http.Client
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("request got an error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		content, _ := ioutil.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("http response status is not 200 OK: %s, %s", resp.Status, utils.RpcStatusDeterministicJsonFormat(content))
	}

	bytesMsg, trailer, err := DecodeGRPCWebResponseBody(resp.Body)
	// If the error is io.EOF it might be header only response.
	if err == io.EOF {
		trailer = GRPCWebTrailer{}
		if grpcStatus := resp.Header.Get("grpc-status"); grpcStatus != "" {
			trailer["grpc-status"] = grpcStatus
		}
		if grpcMsg := resp.Header.Get("grpc-message"); grpcMsg != "" {
			trailer["grpc-message"] = grpcMsg
		}
		if len(trailer) > 0 {
			return "", trailer, nil
		}
	}

	if err != nil {
		return "", nil, fmt.Errorf("decode response body failed: %v", err)
	}

	err = proto.Unmarshal(bytesMsg, respMsg)
	if err != nil {
		return "", nil, fmt.Errorf("failed to unmarshal response message: %v", err)
	}

	var marshaler jsonpb.Marshaler
	respJSON, err := marshaler.MarshalToString(respMsg)
	if err != nil {
		return "", nil, fmt.Errorf("failed to convert response message to JSON: %v", err)
	}

	return respJSON, trailer, nil
}

func MakeTokenInQueryCall(addr, httpMethod, method, token string) (string, error) {
	var cli http.Client
	req, _ := http.NewRequest(httpMethod, fmt.Sprintf("http://%s%s", addr, method), nil)
	if token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := cli.Do(req)
	if err != nil {
		return "", fmt.Errorf("http got error: %v", err)
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http response status is not 200 OK: %s, %s", resp.Status, utils.RpcStatusDeterministicJsonFormat(content))
	}

	return string(content), nil
}
