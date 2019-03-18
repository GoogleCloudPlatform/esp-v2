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

package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"

	"golang.org/x/oauth2/google"

	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/bookstore-grpc/client"
)

const (
	apiKeyHeader = "x-api-key"
)

var (
	addr           = flag.String("addr", "", "Address of grpc server.")
	apikey         = flag.String("apikey", "", "The API Key")
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

	header := http.Header{}
	if *apikey != "" {
		header.Add(apiKeyHeader, *apikey)
	}

	resp, err := client.MakeCall(*clientProtocol, *addr, "GET", *method, *token, header)
	if err != nil {
		log.Fatalf("Makecall failed: %v", err)
	}
	log.Printf("grpc got response: %v", resp)
}
