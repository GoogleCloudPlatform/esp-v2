package main

import (
	"flag"
	"io/ioutil"
	"log"

	"golang.org/x/oauth2/google"

	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/bookstore-grpc/client"
)

var (
	addr           = flag.String("addr", "", "Address of grpc server.")
	method         = flag.String("method", "", "Method name called.")
	clientProtocol = flag.String("client_protocol", "http", "client protocol, either Http or gRPC.")
	token          = flag.String("token", "", "Authentication token.")
	keyfile        = flag.String("keyfile", "", "Path to a Google service account key file.")
	audience       = flag.String("audience", "", "Audience.")
)

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

	resp, err := client.MakeCall(*clientProtocol, *addr, *method, *token)
	if err != nil {
		log.Fatalf("Makecall failed: %v", err)
	}
	log.Printf("grpc got response: %v", resp)
}
