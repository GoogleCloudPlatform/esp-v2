// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	bspbv1 "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/proto/v1"
	bspbv2 "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/proto/v2"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type database struct {
	// All shelves, order matters
	shelves []*bspbv1.Shelf
	// All books, order matters
	books []*bspbv1.Book
	// A map of bookId to the shelfId it resides in
	bookLocation map[int64]int64
}

// BookstoreServerV1Impl represents the gRPC Bookstore health-checkable server.
type BookstoreServerV1Impl struct {
	bsV1Server bspbv1.BookstoreServer
	grpcServer *grpc.Server
	db         *database
}

// BookstoreServerV2Impl represents the gRPC Bookstore health-checkable server.
type BookstoreServerV2Impl struct {
	bsV2Server bspbv2.BookstoreServer
	grpcServer *grpc.Server
}

// BookstoreServer represents two different version gRPC Bookstore servers
// sharing same listener.
type BookstoreServer struct {
	bs1 *BookstoreServerV1Impl
	bs2 *BookstoreServerV2Impl
	lis net.Listener
}

func createDB() *database {
	return &database{
		shelves: []*bspbv1.Shelf{
			{
				Id:    100,
				Theme: "Kids",
			},
			{
				Id:    200,
				Theme: "Classic",
			},
		},
		books: []*bspbv1.Book{
			{
				Id:    1001,
				Title: "Alphabet",
			},
			{
				Id:     2001,
				Title:  "Hamlet",
				Author: "Shakspeare",
			},
		},
		bookLocation: map[int64]int64{
			1001: 100,
			2001: 200,
		},
	}
}

// NewBookstoreServer creates a new server but does not start it.
// This sets up the listening address.
func NewBookstoreServer(port uint16, enableTLS, useUnAuthorizedCert bool, rootCertFile string) (*BookstoreServer, error) {

	// Setup health server
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// Setup gRPC server
	var grpcServer *grpc.Server

	if enableTLS {
		cert := platform.ProxyCert
		key := platform.ProxyKey
		if useUnAuthorizedCert {
			cert = platform.ServerCert
			key = platform.ServerKey
		}
		certificate, err := tls.LoadX509KeyPair(platform.GetFilePath(cert), platform.GetFilePath(key))
		if err != nil {
			return nil, err
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{certificate},
		}

		if rootCertFile != "" {
			certPool := x509.NewCertPool()
			bs, err := ioutil.ReadFile(rootCertFile)
			if err != nil {
				return nil, err
			}

			if !certPool.AppendCertsFromPEM(bs) {
				return nil, fmt.Errorf("failed to append client certs")
			}
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
			tlsConfig.ClientCAs = certPool
		}

		serverOption := grpc.Creds(credentials.NewTLS(tlsConfig))
		grpcServer = grpc.NewServer(serverOption)
		glog.Infof("Bookstore gRPCs server is listening on port %d", port)
	} else {
		grpcServer = grpc.NewServer()
		glog.Infof("Bookstore gRPC server is listening on port %d", port)
	}

	// Create a new listener, allowing it to choose the port
	lis, err := net.Listen("tcp", fmt.Sprintf("%v:%v", platform.GetLoopbackAddress(), port))
	if err != nil {
		return nil, fmt.Errorf("GRPC Bookstore server failed to listen: %v", err)
	}

	endpoint := &BookstoreServer{
		bs1: &BookstoreServerV1Impl{
			grpcServer: grpcServer,
			db:         createDB(),
		},
		bs2: &BookstoreServerV2Impl{
			grpcServer: grpcServer,
		},
		lis: lis,
	}

	// Register gRPC health services and bookstore
	healthgrpc.RegisterHealthServer(grpcServer, healthServer)
	bspbv1.RegisterBookstoreServer(grpcServer, endpoint.bs1)
	bspbv2.RegisterBookstoreServer(grpcServer, endpoint.bs2)

	// Return server
	return endpoint, nil
}

func (s *BookstoreServer) StartServer() {

	// Start server
	go func() {
		glog.Infof("GRPC Bookstore server is running at %s .......\n", s.lis.Addr())
		if err := s.bs1.grpcServer.Serve(s.lis); err != nil {
			glog.Errorf("GRPC Bookstore server fail to serve: %v", err)
		}
		if err := s.bs2.grpcServer.Serve(s.lis); err != nil {
			glog.Errorf("GRPC Bookstore server fail to serve: %v", err)
		}
	}()

}

func (s *BookstoreServer) StopServer() {
	s.bs1.grpcServer.Stop()
	s.bs2.grpcServer.Stop()
}

func testDecorator(ctx context.Context) error {
	// Retrieve headers from call
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return fmt.Errorf("error retrieving metadata from context")
	}
	glog.Infof("GRPC bookstore metadata: %v", md)

	// Check for test headers
	values := md.Get("x-grpc-test")
	if len(values) == 0 {
		return nil
	}
	first := values[0]

	// Return proper response code
	switch first {
	case "ABORTED":
		return status.New(codes.Aborted, first).Err()
	case "INTERNAL":
		return status.New(codes.Internal, first).Err()
	case "DATA_LOSS":
		return status.New(codes.DataLoss, first).Err()
	default:
		glog.Warningf("Unknown metadata: %v", first)
		return nil
	}
}

func (s *BookstoreServerV2Impl) GetShelf(ctx context.Context, req *bspbv2.GetShelfRequest) (*bspbv2.Shelf, error) {
	if err := testDecorator(ctx); err != nil {
		return nil, err
	}

	if req.Shelf == 100 {
		return &bspbv2.Shelf{
			Id:    100,
			Theme: "Kids",
		}, nil
	}

	return nil, status.New(codes.NotFound, "Cannot find requested shelf").Err()
}

func (s *BookstoreServerV2Impl) GetShelfAutoBind(ctx context.Context, req *bspbv2.GetShelfRequest) (*bspbv2.Shelf, error) {
	if err := testDecorator(ctx); err != nil {
		return nil, err
	}

	if req.Shelf == 200 {
		return &bspbv2.Shelf{
			Id:    200,
			Theme: "Classic",
		}, nil
	}

	return nil, status.New(codes.NotFound, "Cannot find requested shelf").Err()
}

func (s *BookstoreServerV1Impl) ListShelves(ctx context.Context, req *bspbv1.Empty) (*bspbv1.ListShelvesResponse, error) {
	if err := testDecorator(ctx); err != nil {
		return nil, err
	}

	return &bspbv1.ListShelvesResponse{
		Shelves: s.db.shelves,
	}, nil
}

func (s *BookstoreServerV1Impl) CreateShelf(ctx context.Context, req *bspbv1.CreateShelfRequest) (*bspbv1.Shelf, error) {
	if err := testDecorator(ctx); err != nil {
		return nil, err
	}

	// Unmarshal, deepcopy and marshal, to verify the received binary `any`
	if req.Shelf.Any != nil {
		var book bspbv1.Book
		err := ptypes.UnmarshalAny(req.Shelf.Any, &book)
		if err != nil {
			return nil, status.New(codes.Internal, "cannot unmarshal shelf.any").Err()
		}

		newBook := book

		req.Shelf.Any, err = ptypes.MarshalAny(&newBook)
		if err != nil {
			return nil, status.New(codes.Internal, "cannot marshal shelf.any").Err()
		}
	}

	return req.Shelf, nil
}

func (s *BookstoreServerV1Impl) GetShelf(ctx context.Context, req *bspbv1.GetShelfRequest) (*bspbv1.Shelf, error) {
	if err := testDecorator(ctx); err != nil {
		return nil, err
	}

	for _, shelf := range s.db.shelves {
		if shelf.Id == req.Shelf {
			return shelf, nil
		}
	}

	return nil, status.New(codes.NotFound, "Cannot find requested shelf").Err()
}

func (s *BookstoreServerV1Impl) DeleteShelf(ctx context.Context, req *bspbv1.DeleteShelfRequest) (*bspbv1.Empty, error) {
	if err := testDecorator(ctx); err != nil {
		return nil, err
	}

	return &bspbv1.Empty{}, nil
}

func (s *BookstoreServerV1Impl) ListBooks(ctx context.Context, req *bspbv1.ListBooksRequest) (*bspbv1.ListBooksResponse, error) {
	if err := testDecorator(ctx); err != nil {
		return nil, err
	}

	books := make([]*bspbv1.Book, 0)
	for _, book := range s.db.books {
		if s.db.bookLocation[book.Id] == req.Shelf {
			books = append(books, book)
		}
	}

	return &bspbv1.ListBooksResponse{
		Books: books,
	}, nil
}

func (s *BookstoreServerV1Impl) CreateBook(ctx context.Context, req *bspbv1.CreateBookRequest) (*bspbv1.Book, error) {
	if err := testDecorator(ctx); err != nil {
		return nil, err
	}

	s.db.books = append(s.db.books, req.Book)
	s.db.bookLocation[req.Book.Id] = req.Shelf

	return req.Book, nil
}

func (s *BookstoreServerV1Impl) GetBook(ctx context.Context, req *bspbv1.GetBookRequest) (*bspbv1.Book, error) {
	if err := testDecorator(ctx); err != nil {
		return nil, err
	}

	for _, book := range s.db.books {
		if book.Id == req.Book && s.db.bookLocation[book.Id] == req.Shelf {
			return book, nil
		}
	}

	return nil, status.New(codes.NotFound, "Cannot find requested book on shelf").Err()
}

func (s *BookstoreServerV1Impl) DeleteBook(ctx context.Context, req *bspbv1.DeleteBookRequest) (*bspbv1.Empty, error) {
	if err := testDecorator(ctx); err != nil {
		return nil, err
	}

	for i, book := range s.db.books {
		if book.Id == req.Book && s.db.bookLocation[book.Id] == req.Shelf {
			s.db.books = s.db.books[:i+copy(s.db.books[i:], s.db.books[i+1:])]
			delete(s.db.bookLocation, book.Id)
			return &bspbv1.Empty{}, nil
		}
	}

	// FIXME: this should return an error, but tests assume it will be OK
	return &bspbv1.Empty{}, nil
}
