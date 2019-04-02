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
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	port := flag.Int("port", 8082, "server port")
	isHttps := flag.Bool("enable_https", false, "true for HTTPS, false for HTTP")
	enableRootPathHandler := flag.Bool("enable_root_path_handler", false, "true for adding root path for dynamic routing handler")
	httpsCertPath := flag.String("https_cert_path", "../../../env/testdata/localhost.crt", "path for HTTPS cert path")
	httpsKeyPath := flag.String("https_key_path", "../../../env/testdata/localhost.key", "path for HTTPS key path")
	flag.Parse()

	r := mux.NewRouter()

	r.Path("/echo").Methods("POST").
		HandlerFunc(echoHandler)
	r.Path("/echo/nokey").Methods("POST").
		HandlerFunc(echoHandler)
	r.Path("/anypath/x/y/z").Methods("POST").
		HandlerFunc(echoHandler)
	r.Path("/simpleget").Methods("GET").
		HandlerFunc(simpleGet)
	r.Path("/simplegetcors").Methods("GET", "OPTIONS").
		Handler(corsHandler(simpleGetCors))
	r.Path("/auth/info/googlejwt").Methods("GET").
		HandlerFunc(authInfoHandler)
	r.Path("/auth/info/googleidtoken").Methods("GET").
		HandlerFunc(authInfoHandler)
	r.Path("/auth/info/firebase").Methods("GET", "OPTIONS").
		Handler(corsHandler(authInfoHandler))
	r.Path("/auth/info/auth0").Methods("GET").
		HandlerFunc(authInfoHandler)
	r.PathPrefix("/bearertoken/").Methods("GET").
		HandlerFunc(bearerTokenHandler)
	r.PathPrefix("/dynamicrouting").Methods("GET", "POST").
		HandlerFunc(dynamicRoutingHandler)
	if *enableRootPathHandler {
		r.PathPrefix("/").Methods("GET").
			HandlerFunc(dynamicRoutingHandler)
	}

	http.Handle("/", r)
	if *port < 1024 || *port > 65535 {
		log.Fatalf("port (%v) should be integer between 1024-65535", *port)
	}
	fmt.Printf("Echo server is running on port: %d, is_https: %v\n", *port, *isHttps)
	var err error
	if *isHttps {
		err = http.ListenAndServeTLS(fmt.Sprintf(":%d", *port), *httpsCertPath, *httpsKeyPath, nil)
	} else {
		err = http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	}
	log.Fatal(err)
}

// echoHandler reads a JSON object from the body, and writes it back out.
func echoHandler(w http.ResponseWriter, r *http.Request) {
	var msg interface{}
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		if _, ok := err.(*json.SyntaxError); ok {
			errorf(w, http.StatusBadRequest, "Body was not valid JSON: %v", err)
			return
		}
		errorf(w, http.StatusInternalServerError, "Could not get body: %v", err)
		return
	}

	b, err := json.Marshal(msg)
	if err != nil {
		errorf(w, http.StatusInternalServerError, "Could not marshal JSON: %v", err)
		return
	}
	w.Write(b)
}

// dynamicEoutingHandler reads URL from request header, and writes it back out.
func dynamicRoutingHandler(w http.ResponseWriter, r *http.Request) {
	resp := fmt.Sprintf(`{"RequestURI": "%s"}`, r.URL.RequestURI())
	w.Write([]byte(resp))
}

// corsHandler wraps a HTTP handler and applies the appropriate responses for Cross-Origin Resource Sharing.
type corsHandler http.HandlerFunc

func simpleGet(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("simple get message"))
}

func simpleGetCors(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		getMessage := fmt.Sprintf("simple get message, time: %v\n", time.Now())
		w.Write([]byte(getMessage))
		return
	}
	w.Write([]byte(""))
}

func (h corsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Origin") != "" {
		fmt.Printf("Origin: %s", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization")
			w.Header().Set("Access-Control-Expose-Headers", "Cache-Control,Content-Type,Authorization, X-PINGOTHER")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			return
		}
	}
	h(w, r)
}

// authInfoHandler reads authentication info provided by the Endpoints proxy.
func authInfoHandler(w http.ResponseWriter, r *http.Request) {
	encodedInfo := r.Header.Get("X-Endpoint-API-UserInfo")
	if encodedInfo == "" {
		w.Write([]byte(`{"id": "anonymous"}`))
		return
	}

	b, err := base64.StdEncoding.DecodeString(encodedInfo)
	if err != nil {
		errorf(w, http.StatusInternalServerError, "Could not decode auth info: %v", err)
		return
	}
	w.Write(b)
}

// bearerTokenHandler reads "Authorization" header and request URI.
func bearerTokenHandler(w http.ResponseWriter, r *http.Request) {
	bearerToken := r.Header.Get("Authorization")
	reqURI := r.URL.RequestURI()
	resp := fmt.Sprintf(`{"Authorization": "%s", "RequestURI": "%s"}`, bearerToken, reqURI)
	w.Write([]byte(resp))
}

// errorf writes a swagger-compliant error response.
func errorf(w http.ResponseWriter, code int, format string, a ...interface{}) {
	var out struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	out.Code = code
	out.Message = fmt.Sprintf(format, a...)

	b, err := json.Marshal(out)
	if err != nil {
		http.Error(w, `{"code": 500, "message": "Could not format JSON for original message."}`, 500)
		return
	}

	http.Error(w, string(b), code)
}
