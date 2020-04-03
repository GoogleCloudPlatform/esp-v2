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

package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/gorilla/websocket"
	"golang.org/x/net/http2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jws"

	tpoauth2 "github.com/GoogleCloudPlatform/esp-v2/third_party/golang_internal"
)

// DoGet performs a Get request to a specified url
func DoGet(url string) ([]byte, error) {
	return DoWithHeaders(url, "GET", "", nil)
}

// DoPost performs a POST request to a specified url
func DoPost(url, message string) ([]byte, error) {
	return DoWithHeaders(url, "POST", message, nil)
}

// DoPostWithHeaders performs a POST request to a specified url with given headers and message
func DoPostWithHeaders(url, message string, headers map[string]string) ([]byte, error) {
	return DoWithHeaders(url, "POST", message, headers)
}

// DoWithHeaders performs a GET/POST/PUT/DELETE/PATCH request to a specified url with given headers and message(if provided)
func DoWithHeaders(url, method, message string, headers map[string]string) ([]byte, error) {
	var request *http.Request
	var err error
	if method == "DELETE" || method == "GET" {
		request, err = http.NewRequest(method, url, nil)
	} else {
		msg := map[string]string{
			"message": message,
		}
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(msg); err != nil {
			return nil, err
		}
		request, err = http.NewRequest(method, url, &buf)
	}

	if err != nil {
		return nil, fmt.Errorf("create request error: %v", err)
	}

	if message != "" {
		request.Header.Set("Content-Type", "application/json")
	}

	for k, v := range headers {
		request.Header.Set(k, v)
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}

	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("http %s error: %v", method, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("http got error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http response status is not 200 OK: %s, %s", resp.Status, string(bodyBytes))
	}
	return bodyBytes, err
}

// DoJWT performs an authenticated request using the credentials in the service account file.
func DoJWT(host, method, path, apiKey, serviceAccount, token string) ([]byte, error) {
	if serviceAccount != "" {
		sa, err := ioutil.ReadFile(serviceAccount)
		if err != nil {
			return nil, fmt.Errorf("Could not read service account file: %v", err)
		}
		conf, err := google.JWTConfigFromJSON(sa)
		if err != nil {
			return nil, fmt.Errorf("Could not parse service account JSON: %v", err)
		}
		rsaKey, err := tpoauth2.ParseKey(conf.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("Could not get RSA key: %v", err)
		}

		iat := time.Now()
		exp := iat.Add(time.Hour)

		jwt := &jws.ClaimSet{
			Iss:   testdata.JwtEndpointsIssuer,
			Sub:   "foo!",
			Aud:   "echo.endpoints.sample.google.com",
			Scope: "email",
			Iat:   iat.Unix(),
			Exp:   exp.Unix(),
		}
		jwsHeader := &jws.Header{
			Algorithm: "RS256",
			Typ:       "JWT",
		}

		token, err = jws.Encode(jwsHeader, jwt, rsaKey)
		if err != nil {
			return nil, fmt.Errorf("Could not encode JWT: %v", err)
		}
	}

	req, _ := http.NewRequest(method, host+path+"?key="+apiKey, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http got error: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("http got error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http response status is not 200 OK: %s, %s", resp.Status, string(bodyBytes))
	}
	return bodyBytes, err
}

// DoCorsSimpleRequest sends a simple request with Origin field in request header
func DoCorsSimpleRequest(url, httpMethod, origin, msg string) (http.Header, error) {
	var req *http.Request
	var err error
	if httpMethod == "GET" || httpMethod == "HEAD" {
		req, err = http.NewRequest(httpMethod, url, nil)
	} else if httpMethod == "POST" {
		msg := map[string]string{
			"message": msg,
		}
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(msg); err != nil {
			return nil, err
		}
		req, err = http.NewRequest("POST", url, &buf)
	} else {
		return nil, fmt.Errorf("DoCorsSimpleRequest only supports GET, HEAD and POST: %v", err)
	}
	if err != nil {
		return nil, fmt.Errorf("NewRequest got error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", origin)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http got error: %v", err)
	}
	defer resp.Body.Close()
	return resp.Header, nil
}

func DoCorsPreflightRequest(url, origin, requestMethod, requestHeader, referer string) (http.Header, error) {
	req, err := http.NewRequest("OPTIONS", url, nil)
	if err != nil {
		return nil, fmt.Errorf("NewRequest got error: %v", err)
	}
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", requestMethod)
	if requestHeader != "" {
		req.Header.Set("Access-Control-Request-Headers", requestHeader)
	}
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http got error: %v", err)
	}
	defer resp.Body.Close()
	return resp.Header, nil
}

// DoHttpsGet performs a HTTPS Get request to a specified url
func DoHttpsGet(url string, httpVersion int, certPath string) (http.Header, []byte, error) {
	client := &http.Client{}
	caCert, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	// Create TLS configuration with the certificate of the server
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}

	// Use the proper transport in the client
	switch httpVersion {
	case 1:
		client.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	case 2:
		client.Transport = &http2.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("http response status is not 200 OK: %s, %s", resp.Status, string(body))
	}
	return resp.Header, body, err
}

func DoWS(address, path, query, reqMsg string, messageCount int) ([]byte, error) {
	var resp []byte
	u := url.URL{Scheme: "ws", Host: address, Path: path, RawQuery: query}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	for i := 0; i < messageCount; i++ {
		err = c.WriteMessage(websocket.TextMessage, []byte(reqMsg))
		if err != nil {
			return nil, err
		}
		_, respMsg, err := c.ReadMessage()
		if err != nil {
			return nil, err
		}
		resp = append(resp, respMsg...)
	}

	return resp, nil
}
