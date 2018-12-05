package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"golang.org/x/oauth2/google"

	bspb "cloudesf.googlesource.com/gcpproxy/tests/endpoints/bookstore-grpc/proto"
	"github.com/gogo/protobuf/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	addr           = flag.String("addr", "127.0.0.1:8080", "Address of grpc server.")
	clientProtocol = flag.String("client_protocol", "http", "client protocol, either Http or gRPC.")
	token          = flag.String("token", "", "Authentication token.")
	keyfile        = flag.String("keyfile", "", "Path to a Google service account key file.")
	audience       = flag.String("audience", "", "Audience.")
)

var client http.Client

var makeHttpCall = func() ([]byte, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s/v1/shelves", *addr), nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *token))

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("http got error: ", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatal("http response status is not 200 OK: ", resp.Status)
	}
	return ioutil.ReadAll(resp.Body)
}

var makeGrpcCall = func() error {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(*addr, opts...)
	if err != nil {
		log.Fatalf("failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := bspb.NewBookstoreClient(conn)
	ctx := context.Background()
	if *token != "" {
		log.Printf("Using authentication token: %s", *token)
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("Authorization", fmt.Sprintf("Bearer %s", *token)))
	}

	req := &types.Empty{}
	resp, err := client.ListShelves(ctx, req)
	if err != nil {
		log.Fatalf("ListShelves got unexpected error: %v", err)
	} else {
		log.Printf("grpc got response: %v", resp)
	}
	return nil
}

func main() {
	flag.Parse()

	if *keyfile != "" {
		log.Printf("Authenticating using Google service account key in %s", *keyfile)
		keyBytes, err := ioutil.ReadFile(*keyfile)
		if err != nil {
			log.Fatalf("Unable to read service account key file %s: %v", *keyfile, err)
		}
		tokenSource, err := google.JWTAccessTokenSourceFromJSON(keyBytes, *audience)
		if err != nil {
			log.Fatalf("Error building JWT access token source: %v", err)
		}
		jwt, err := tokenSource.Token()
		if err != nil {
			log.Fatalf("Unable to generate JWT token: %v", err)
		}
		*token = jwt.AccessToken
	}

	if strings.EqualFold(*clientProtocol, "http") {
		resp, err := makeHttpCall()
		if err != nil {
			log.Fatalf("makeHttpCall got unexpected error: %v", err)
		}
		log.Printf("got Http response: %s\n", string(resp))
	} else {
		if err := makeGrpcCall(); err != nil {
			log.Fatalf("makeGrpcCall got unexpected error: %v", err)
		}
	}
}
