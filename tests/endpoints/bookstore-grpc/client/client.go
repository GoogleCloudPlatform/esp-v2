// Copyright 2018 Google Cloud Platform Proxy Authors
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
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"

	bspb "cloudesf.googlesource.com/gcpproxy/tests/endpoints/bookstore-grpc/proto"
	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var grpcWebHeader []header

const (
	bookstoreService = "endpoints.examples.bookstore.Bookstore"
)

type header struct {
	key string
	val string
}

func init() {
	grpcWebHeader = []header{
		{"X-User-Agent", "grpc-web-javascript/0.1"},
		{"Content-Type", "application/grpc-web-text"},
		{"Accept", "application/grpc-web-text"},
		{"X-Grpc-Web", "1"},
	}
}

func MakeCall(clientProtocol, addr, httpMethod, method, token string) (string, error) {
	if strings.EqualFold(clientProtocol, "http") {
		return makeHttpCall(addr, httpMethod, method, token)
	}

	if strings.EqualFold(clientProtocol, "grpc") {
		return makeGrpcCall(addr, method, token)
	}

	if strings.EqualFold(clientProtocol, "grpc-web") {
		return makeGrpcWebCall(addr, method, token)
	}

	return "", fmt.Errorf("unsupported client protocol %s", clientProtocol)
}

var makeHttpCall = func(addr, httpMethod, method, token string) (string, error) {
	var cli http.Client
	req, _ := http.NewRequest(httpMethod, fmt.Sprintf("http://%s%s", addr, method), nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := cli.Do(req)
	if err != nil {
		return "", fmt.Errorf("http got error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http response status is not 200 OK: %s", resp.Status)
	}

	content, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	return string(content), nil
}

var makeGrpcCall = func(addr, method, token string) (string, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to connect to server: %v", err)
	}
	defer conn.Close()

	cli := bspb.NewBookstoreClient(conn)
	ctx := context.Background()
	if token != "" {
		log.Printf("Using authentication token: %s", token)
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("Authorization", fmt.Sprintf("Bearer %s", token)))
	}

	var respMsg proto.Message
	switch method {
	case "ListShelves":
		req := &types.Empty{}
		respMsg, err = cli.ListShelves(ctx, req)
	case "CreateShelf":
		req := &bspb.CreateShelfRequest{
			Shelf: &bspb.Shelf{},
		}
		respMsg, err = cli.CreateShelf(ctx, req)
	case "GetShelf":
		req := &bspb.GetShelfRequest{}
		respMsg, err = cli.GetShelf(ctx, req)
	case "CreateBook":
		req := &bspb.CreateBookRequest{}
		respMsg, err = cli.CreateBook(ctx, req)
	case "DeleteShelf":
		req := &bspb.DeleteShelfRequest{}
		respMsg, err = cli.DeleteShelf(ctx, req)
	default:
		return "", fmt.Errorf("unexpected method called")
	}

	if err != nil {
		return "", fmt.Errorf("%v got unexpected error: %v", method, err)
	}

	var marshaler jsonpb.Marshaler
	return marshaler.MarshalToString(respMsg)
}

var makeGrpcWebCall = func(addr, method, token string) (string, error) {
	var reqMsg proto.Message
	switch method {
	case "ListShelves":
		reqMsg = &types.Empty{}
	case "CreateShelf":
		reqMsg = &bspb.CreateShelfRequest{
			Shelf: &bspb.Shelf{},
		}
	case "GetShelf":
		reqMsg = &bspb.GetShelfRequest{}
	case "CreateBook":
		reqMsg = &bspb.CreateBookRequest{}
	case "DeleteShelf":
		reqMsg = &bspb.DeleteShelfRequest{}
	default:
		return "", fmt.Errorf("unexpected method called")
	}
	body := EncodeGrpcWebRequestBody(reqMsg)

	u, err := url.Parse("http://" + addr)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, bookstoreService, method)
	req, err := http.NewRequest("POST", u.String(), body)
	if err != nil {
		return "", err
	}

	for _, h := range grpcWebHeader {
		req.Header.Add(h.key, h.val)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	var client http.Client
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request got an error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("response status is not 200 OK: %s", resp.Status)
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("cannot read response body: %v", err)
	}
	return string(content), nil
}
