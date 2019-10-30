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

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/echo/client"
)

var (
	host   = flag.String("host", "", "The API host. Required.")
	path   = flag.String("path", "", "url path. Required.")
	apiKey = flag.String("api-key", "", "Your API key. Required.")

	echo           = flag.String("echo", "", "Message to echo. Cannot be used with -service-account")
	serviceAccount = flag.String("service-account", "", "Path to service account JSON file. Cannot be used with -echo.")
	token          = flag.String("token", "", "Authentication token.")
)

func main() {
	flag.Parse()

	var resp []byte
	var err error
	if *echo != "" {
		url := fmt.Sprintf("%v%v?key=%v", *host, "/echo", *apiKey)
		resp, err = client.DoPost(url, *echo)
	} else if *serviceAccount != "" {
		resp, err = client.DoJWT(*host, "GET", *path, *apiKey, *serviceAccount, *token)
	}
	if err != nil {
		log.Fatal(err)
	}
	os.Stdout.Write(resp)
}
