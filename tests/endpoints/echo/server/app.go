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
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var (
	port                  = flag.Int("port", 8082, "server port")
	isHttps               = flag.Bool("enable_https", false, "true for HTTPS, false for HTTP")
	disableHttp2          = flag.Bool("disable_http2", false, "Set to true to disable http/2 handler. By default, accepts http/1 and http/2 connections.")
	mtlsCertFile          = flag.String("mtls_cert_file", "", "Enable Mutual authentication with the cert chain file")
	enableRootPathHandler = flag.Bool("enable_root_path_handler", false, "true for adding root path for dynamic routing handler")
	httpsCertPath         = flag.String("https_cert_path", "", "path for HTTPS cert path")
	httpsKeyPath          = flag.String("https_key_path", "", "path for HTTPS key path")
	alwaysRespondRST      = flag.Bool("always_respond_rst", false, "If true, the backend will respond RST all the time")
	rejectRequestNum      = flag.Int("reject_request_num", 0, "The first N requests that the backend will reject")
	rejectRequestStatus   = flag.Int("reject_request_status", 0, `The http status code when the backend uses to reject the first N requests defined by reject_request_num`)
	webSocketUpgrader     = websocket.Upgrader{}
)

func main() {
	flag.Parse()

	if *alwaysRespondRST {
		s, _ := NewRstListener(fmt.Sprintf(":%v", *port))
		for {
			// Keep sending RST to the downstream.
			_, _ = s.Accept()
			_ = s.Close()
		}
	}

	r := mux.NewRouter()

	r.Path("/echo").Methods("POST", "GET").
		HandlerFunc(echoHandler)
	r.Path("/echoHeader").Methods("GET").
		HandlerFunc(echoHeaderHandler)
	r.PathPrefix("/echoMethod").Methods("GET", "POST", "PUT", "DELETE", "PATCH").
		HandlerFunc(echoMethodHandler)
	r.Path("/prefix/echo/{variable}/path").Methods("GET").HandlerFunc(dynamicRoutingHandler)

	r.Path("/websocketecho").HandlerFunc(websocketEchoHandler)
	r.Path("/echo/nokey").Methods("POST").
		HandlerFunc(echoHandler)
	r.Path("/echo/nokey/OverrideAsGet").Methods("POST").
		HandlerFunc(echoHandler)
	r.Path("/anypath/x/y/z").Methods("POST").
		HandlerFunc(echoHandler)
	r.Path("/simpleget").Methods("GET").
		HandlerFunc(simpleGet)
	r.Path("/simpleget/304").Methods("GET").
		HandlerFunc(simpleGetNotModified)
	r.Path("/simpleget/403").Methods("GET").
		HandlerFunc(simpleGetForbidden)
	r.Path("/simpleget/401").Methods("GET").
		HandlerFunc(simpleGetUnauthorized)
	r.Path("/simplegetcors").Methods("GET", "OPTIONS").
		Handler(corsHandler(simpleGetCors))
	r.Path("/bookstore/shelves").Methods("OPTIONS").Handler(corsHandler(simpleGetCors))
	r.Path("/bookstore/shelves/{shelfId}").Methods("OPTIONS").Handler(corsHandler(simpleGetCors))
	r.Path("/auth/info/googlejwt").Methods("GET").
		HandlerFunc(authInfoHandler)
	r.Path("/auth/info/authJwksCacheTestOnly").Methods("GET").
		HandlerFunc(authInfoHandler)
	r.Path("/auth/info/serviceControlCheckErrorOnly").Methods("GET").
		HandlerFunc(authInfoHandler)
	r.Path("/auth/info/googleidtoken").Methods("GET").
		HandlerFunc(authInfoHandler)
	r.Path("/auth/info/firebase").Methods("GET", "OPTIONS").
		Handler(corsHandler(authInfoHandler))
	r.Path("/auth/info/auth0").Methods("GET").
		HandlerFunc(authInfoHandler)
	r.PathPrefix("/bearertoken/").Methods("GET", "OPTIONS").
		HandlerFunc(bearerTokenHandler)
	r.PathPrefix("/dynamicrouting").Methods("OPTIONS", "GET", "POST").
		Handler(corsHandler(dynamicRoutingHandler))
	r.PathPrefix("/sleep").Methods("GET").
		HandlerFunc(sleepHandler)

	if *enableRootPathHandler {
		r.PathPrefix("/").Methods("GET", "POST").
			HandlerFunc(dynamicRoutingHandler)
	}

	http.Handle("/", RejectMiddleWare(r))
	if *port < 1024 || *port > 65535 {
		log.Fatalf("port (%v) should be integer between 1024-65535", *port)
	}
	fmt.Printf("Echo server is running on port: %d, is_https: %v\n", *port, *isHttps)

	server, err := createServer()
	if err != nil {
		log.Fatal(err)
		return
	}

	if *isHttps {
		err = server.ListenAndServeTLS(*httpsCertPath, *httpsKeyPath)
	} else {
		err = server.ListenAndServe()
	}
	log.Fatal(err)
}

// RstListener is a listener that sends an RST when closed instead of a FIN.
type RstListener struct {
	l net.Listener
}

func NewRstListener(addr string) (*RstListener, error) {
	l, err := net.Listen("tcp", addr)
	return &RstListener{l: l}, err
}

// Accept is like a normal Accept except that it ses SO_LINGER time to 0, so that when
// close is called on the socket it will send an RST instead of a FIN.
func (l *RstListener) Accept() (net.Conn, error) {
	conn, err := l.l.Accept()
	if err != nil {
		return nil, err
	}

	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return nil, fmt.Errorf("Expected a TCP connection")
	}

	err = tcpConn.SetLinger(0)
	if err != nil {
		fmt.Printf("get error before sending rst: %v", err)
	}
	return conn, err
}

func (l *RstListener) Close() error {
	return l.l.Close()
}

func RejectMiddleWare(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if *rejectRequestNum > 0 {
			*rejectRequestNum -= 1
			w.WriteHeader(*rejectRequestStatus)
			return
		}
		h.ServeHTTP(w, r)
	})
}

var skipSleepAfter = -1

// sleepHandler sleeps for the given duration, then responds with 200 OK
// Add the duration to sleep as a query param: ?duration=10s
func sleepHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	sleepDurationStr := queryParams.Get("duration")

	sleepDuration, err := time.ParseDuration(sleepDurationStr)
	if err != nil {
		errorf(w, http.StatusBadRequest, "Invalid duration: %v", err)
		return
	}

	// Use `skipafter` query parameter to simulate the case where the backend
	// intermittently has long processing time.
	// Only the first request carrying `skipafter=cnt` will be effective and the backend
	// won't sleep after cnt requests.
	skipSleepAfterStr := queryParams.Get("skipafter")
	skipSleepAfterInt, err := strconv.Atoi(skipSleepAfterStr)
	if err == nil && skipSleepAfterInt > 0 && skipSleepAfter == -1 {
		skipSleepAfter = skipSleepAfterInt
	}

	if skipSleepAfter > 0 {
		skipSleepAfter -= 1
	} else if skipSleepAfter == 0 {
		w.Write([]byte(fmt.Sprintf("Sleep done: %v", sleepDurationStr)))
		return
	}

	glog.Infof("Echo backend sleeping now: %v", sleepDurationStr)
	time.Sleep(sleepDuration)
	glog.Info("Echo backend done sleeping")

	w.Write([]byte(fmt.Sprintf("Sleep done: %v", sleepDurationStr)))
}

// echoMethodHandler reads a method from the header, and writes it back out.
func echoMethodHandler(w http.ResponseWriter, r *http.Request) {
	resp := fmt.Sprintf(`{"RequestMethod": "%s"}`, r.Method)
	w.Write([]byte(resp))
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

	for key, vals := range r.Header {
		if strings.HasPrefix(key, "Fake-Header-Key") {
			for _, val := range vals {
				w.Header().Add(fmt.Sprintf("Echo-%s", key), val)
			}
		}
	}
	w.Write(b)
}

// echoHeader handler will echo back all headers.
// Attaches prefix `Echo-` to each header before echoing it back.
func echoHeaderHandler(w http.ResponseWriter, r *http.Request) {
	for key, vals := range r.Header {
		for _, val := range vals {
			w.Header().Add(fmt.Sprintf("Echo-%s", key), val)
		}
	}

}

// websocketEchoHandler handles echo request through webstocket
func websocketEchoHandler(w http.ResponseWriter, r *http.Request) {
	c, err := webSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		errorf(w, http.StatusInternalServerError, "websocket upgrade failed: %v", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			errorf(w, http.StatusInternalServerError, "websocket read failed: %v", err)
			break
		}
		log.Printf("recv: %s", message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			errorf(w, http.StatusInternalServerError, "websocket write failed: %v", err)
			break
		}
	}
}

// dynamicRoutingHandler reads URL from request header, and writes it back out.
func dynamicRoutingHandler(w http.ResponseWriter, r *http.Request) {
	// Handle sleeps
	if strings.Contains(r.URL.Path, "/sleep") {
		sleepHandler(w, r)
		return
	}

	resp := fmt.Sprintf(`{"RequestURI": "%s"}`, r.URL.RequestURI())
	w.Write([]byte(resp))
}

// corsHandler wraps a HTTP handler and applies the appropriate responses for Cross-Origin Resource Sharing.
type corsHandler http.HandlerFunc

func simpleGet(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("simple get message"))
}

func simpleGetNotModified(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotModified)
}

func simpleGetForbidden(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusForbidden)
}

func simpleGetUnauthorized(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
}

func simpleGetCors(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		getMessage := fmt.Sprintf("simple get message, time: %v\n", time.Now())
		w.Write([]byte(getMessage))
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
			w.Header().Set("Request-Url", r.RequestURI)
			w.WriteHeader(http.StatusNoContent)
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

	b, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(encodedInfo)
	if err != nil {
		errorf(w, http.StatusInternalServerError, "Could not decode auth info: %v", err)
		return
	}
	w.Write(b)
}

// bearerTokenHandler reads "Authorization" header and request URI.
func bearerTokenHandler(w http.ResponseWriter, r *http.Request) {
	bearerToken := r.Header.Get("Authorization")
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Expose-Headers", fmt.Sprintf("X-Token: %s", bearerToken))
	}
	reqURI := r.URL.RequestURI()
	resp := fmt.Sprintf(`{"Authorization": "%s", "RequestURI": "%s"`, bearerToken, reqURI)

	xForwarded := r.Header.Get("X-Forwarded-Authorization")
	if xForwarded != "" {
		resp += fmt.Sprintf(`, "X-Forwarded-Authorization": "%s"`, xForwarded)
	}

	xEndpoint := r.Header.Get("X-Endpoint-API-UserInfo")
	if xEndpoint != "" {
		resp += fmt.Sprintf(`, "X-Endpoint-API-UserInfo": "%s"`, xEndpoint)
	}

	resp += "}"
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

func createServer() (*http.Server, error) {
	addr := fmt.Sprintf("%v:%d", platform.GetLoopbackAddress(), *port)
	server := &http.Server{
		Addr: addr,
	}

	// Disable HTTP/2 support if needed by setting an empty handler.
	if *isHttps && *disableHttp2 {
		server.TLSNextProto = map[string]func(*http.Server, *tls.Conn, http.Handler){}
	}

	if *mtlsCertFile == "" {
		return server, nil
	}

	// Setup mTLS.
	if !*isHttps {
		return nil, fmt.Errorf("server must be HTTPS server when mTLS is required")
	}
	clientCACert, err := ioutil.ReadFile(*mtlsCertFile)
	if err != nil {
		return nil, err
	}

	clientCertPool := x509.NewCertPool()
	clientCertPool.AppendCertsFromPEM(clientCACert)

	tlsConfig := &tls.Config{
		ClientAuth:               tls.RequireAndVerifyClientCert,
		ClientCAs:                clientCertPool,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS12,
	}

	tlsConfig.BuildNameToCertificate()

	server.TLSConfig = tlsConfig
	return server, nil
}
