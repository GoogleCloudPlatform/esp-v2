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
	"strings"

	bspb "cloudesf.googlesource.com/gcpproxy/tests/endpoints/bookstore-grpc/proto"
	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func MakeCall(clientProtocol, addr, httpMethod, method, token string) (string, error) {
	if strings.EqualFold(clientProtocol, "http") {
		resp, err := makeHttpCall(addr, httpMethod, method, token)
		if err != nil {
			return "", fmt.Errorf("makeHttpCall got unexpected error: %v", err)
		}
		return string(resp), nil
	} else {
		resp, err := makeGrpcCall(addr, method, token)
		if err != nil {
			return "", fmt.Errorf("makeGrpcCall got unexpected error: %v", err)
		}
		return resp, nil
	}
	return "", nil
}

var makeHttpCall = func(addr, httpMethod, method, token string) ([]byte, error) {
	var cli http.Client
	req, _ := http.NewRequest(httpMethod, fmt.Sprintf("http://%s%s", addr, method), nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http got error: ", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http response status is not 200 OK: %s", resp.Status)
	}
	return ioutil.ReadAll(resp.Body)
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

	var marshaler jsonpb.Marshaler
	switch method {
	case "ListShelves":
		req := &types.Empty{}
		resp, err := cli.ListShelves(ctx, req)
		if err != nil {
			return "", fmt.Errorf("ListShelves got unexpected error: %v", err)
		} else {
			respJson, err := marshaler.MarshalToString(resp)
			if err != nil {
				return "", fmt.Errorf("MarshalToString failed: %v", err)
			}
			return respJson, nil
		}

	case "CreateShelf":
		req := &bspb.CreateShelfRequest{
			Shelf: &bspb.Shelf{},
		}
		resp, err := cli.CreateShelf(ctx, req)
		if err != nil {
			return "", fmt.Errorf("CreateShelf got unexpected error: %v", err)
		} else {
			respJson, err := marshaler.MarshalToString(resp)
			if err != nil {
				return "", fmt.Errorf("MarshalToString failed: %v", err)
			}
			return respJson, nil
		}

	case "GetShelf":
		req := &bspb.GetShelfRequest{}
		resp, err := cli.GetShelf(ctx, req)
		if err != nil {
			return "", fmt.Errorf("GetShelf got unexpected error: %v", err)
		} else {
			respJson, err := marshaler.MarshalToString(resp)
			if err != nil {
				return "", fmt.Errorf("MarshalToString failed: %v", err)
			}
			return respJson, nil
		}

	case "CreateBook":
		req := &bspb.CreateBookRequest{}
		resp, err := cli.CreateBook(ctx, req)
		if err != nil {
			return "", fmt.Errorf("CreateBook got unexpected error: %v", err)
		} else {
			respJson, err := marshaler.MarshalToString(resp)
			if err != nil {
				return "", fmt.Errorf("MarshalToString failed: %v", err)
			}
			return respJson, nil
		}

	default:
		return "", fmt.Errorf("unexpected method called")
	}
	return "", nil
}
