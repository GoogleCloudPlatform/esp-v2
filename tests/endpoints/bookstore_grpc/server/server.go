// Copyright 2019 Google Cloud Platform Proxy Authors
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
	"fmt"
	"net"

	"github.com/GoogleCloudPlatform/api-proxy/tests/env/platform"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	bookgrpc "github.com/GoogleCloudPlatform/api-proxy/tests/endpoints/bookstore_grpc/proto"
	bookpb "github.com/GoogleCloudPlatform/api-proxy/tests/endpoints/bookstore_grpc/proto"
	emptypb "github.com/golang/protobuf/ptypes/empty"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type database struct {
	// All shelves, order matters
	shelves []*bookpb.Shelf
	// All books, order matters
	books []*bookpb.Book
	// A map of bookId to the shelfId it resides in
	bookLocation map[int64]int64
}

// BookstoreServer represents the gRPC Bookstore health-checkable server.
// The address the server is listening on can be extracted from this struct.
type BookstoreServer struct {
	bookgrpc.BookstoreServer
	lis        net.Listener
	grpcServer *grpc.Server
	db         *database
}

func createDB() *database {
	return &database{
		shelves: []*bookpb.Shelf{
			{
				Id:    100,
				Theme: "Kids",
			},
			{
				Id:    200,
				Theme: "Classic",
			},
		},
		books: []*bookpb.Book{
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
func NewBookstoreServer(port uint16) (*BookstoreServer, error) {

	// Setup health server
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// Setup gRPC server
	grpcServer := grpc.NewServer()

	// Create a new listener, allowing it to choose the port
	lis, err := net.Listen("tcp", fmt.Sprintf("%v:%v", platform.GetLoopbackAddress(), port))
	if err != nil {
		return nil, fmt.Errorf("GRPC Bookstore server failed to listen: %v", err)
	}

	bss := &BookstoreServer{
		lis:        lis,
		grpcServer: grpcServer,
		db:         createDB(),
	}

	// Register gRPC health services and bookstore
	healthgrpc.RegisterHealthServer(grpcServer, healthServer)
	bookgrpc.RegisterBookstoreServer(grpcServer, bss)

	// Return server
	return bss, nil
}

func (s *BookstoreServer) StartServer() {

	// Start server
	go func() {
		glog.Infof("GRPC Bookstore server is running at %s .......\n", s.lis.Addr())
		if err := s.grpcServer.Serve(s.lis); err != nil {
			glog.Errorf("GRPC Bookstore server fail to serve: %v", err)
		}
	}()

}

func (s *BookstoreServer) StopServer() {
	s.grpcServer.Stop()
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

func (s *BookstoreServer) ListShelves(ctx context.Context, req *emptypb.Empty) (*bookpb.ListShelvesResponse, error) {
	if err := testDecorator(ctx); err != nil {
		return nil, err
	}

	return &bookpb.ListShelvesResponse{
		Shelves: s.db.shelves,
	}, nil
}

func (s *BookstoreServer) CreateShelf(ctx context.Context, req *bookpb.CreateShelfRequest) (*bookpb.Shelf, error) {
	if err := testDecorator(ctx); err != nil {
		return nil, err
	}

	return req.Shelf, nil
}

func (s *BookstoreServer) GetShelf(ctx context.Context, req *bookpb.GetShelfRequest) (*bookpb.Shelf, error) {
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

func (s *BookstoreServer) DeleteShelf(ctx context.Context, req *bookpb.DeleteShelfRequest) (*emptypb.Empty, error) {
	if err := testDecorator(ctx); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *BookstoreServer) ListBooks(ctx context.Context, req *bookpb.ListBooksRequest) (*bookpb.ListBooksResponse, error) {
	if err := testDecorator(ctx); err != nil {
		return nil, err
	}

	books := make([]*bookpb.Book, 0)
	for _, book := range s.db.books {
		if s.db.bookLocation[book.Id] == req.Shelf {
			books = append(books, book)
		}
	}

	return &bookpb.ListBooksResponse{
		Books: books,
	}, nil
}

func (s *BookstoreServer) CreateBook(ctx context.Context, req *bookpb.CreateBookRequest) (*bookpb.Book, error) {
	if err := testDecorator(ctx); err != nil {
		return nil, err
	}

	s.db.books = append(s.db.books, req.Book)
	s.db.bookLocation[req.Book.Id] = req.Shelf

	return req.Book, nil
}

func (s *BookstoreServer) GetBook(ctx context.Context, req *bookpb.GetBookRequest) (*bookpb.Book, error) {
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

func (s *BookstoreServer) DeleteBook(ctx context.Context, req *bookpb.DeleteBookRequest) (*emptypb.Empty, error) {
	if err := testDecorator(ctx); err != nil {
		return nil, err
	}

	for i, book := range s.db.books {
		if book.Id == req.Book && s.db.bookLocation[book.Id] == req.Shelf {
			s.db.books = s.db.books[:i+copy(s.db.books[i:], s.db.books[i+1:])]
			delete(s.db.bookLocation, book.Id)
			return &emptypb.Empty{}, nil
		}
	}

	// FIXME: this should return an error, but tests assume it will be OK
	return &emptypb.Empty{}, nil
}
