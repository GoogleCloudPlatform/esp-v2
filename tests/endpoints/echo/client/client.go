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
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jws"
)

// DoGet performs a Get request to a specified url
func DoGet(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("http got error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http response status is not 200 OK: %s", resp.Status)
	}
	return ioutil.ReadAll(resp.Body)
}

// DoPost performs a POST request to a specified url
func DoPost(url, message string) ([]byte, error) {
	return DoPostWithHeaders(url, message, nil)
}

// DoPostWithHeaders performs a POST request to a specified url with the given headers
func DoPostWithHeaders(url, message string, header map[string]string) ([]byte, error) {
	if header == nil {
		header = map[string]string{}
	}
	msg := map[string]string{
		"message": message,
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(msg); err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", url, &buf)

	if err != nil {
		return nil, fmt.Errorf("create request error: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	for k, v := range header {
		request.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("http post error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http response status is not 200 OK: %s", resp.Status)
	}
	return ioutil.ReadAll(resp.Body)
}

// doJWT performs an authenticated request using the credentials in the service account file.
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
		rsaKey, err := parseKey(conf.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("Could not get RSA key: %v", err)
		}

		iat := time.Now()
		exp := iat.Add(time.Hour)

		jwt := &jws.ClaimSet{
			Iss:   "jwt-client.endpoints.sample.google.com",
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
		return nil, fmt.Errorf("DoCorsSimpleRequest only supports GET, HEAD and POST", err)
	}
	if err != nil {
		return nil, fmt.Errorf("NewRequest got error: ", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", origin)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http got error: ", err)
	}
	defer resp.Body.Close()
	return resp.Header, nil
}

func DoCorsPreflightRequest(url, origin, requestMethod, requestHeader, referer string) (http.Header, error) {
	req, err := http.NewRequest("OPTIONS", url, nil)
	if err != nil {
		return nil, fmt.Errorf("NewRequest got error: ", err)
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
		return nil, fmt.Errorf("http got error: ", err)
	}
	defer resp.Body.Close()
	return resp.Header, nil
}

// The following code is copied from golang.org/x/oauth2/internal
// Copyright (c) 2009 The oauth2 Authors. All rights reserved.

// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:

//   * Redistributions of source code must retain the above copyright
//notice, this list of conditions and the following disclaimer.
//   * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//    * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// parseKey converts the binary contents of a private key file
// to an *rsa.PrivateKey. It detects whether the private key is in a
// PEM container or not. If so, it extracts the the private key
// from PEM container before conversion. It only supports PEM
// containers with no passphrase.
func parseKey(key []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(key)
	if block != nil {
		key = block.Bytes
	}
	parsedKey, err := x509.ParsePKCS8PrivateKey(key)
	if err != nil {
		parsedKey, err = x509.ParsePKCS1PrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("private key should be a PEM or plain PKSC1 or PKCS8; parse error: %v", err)
		}
	}
	parsed, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is invalid")
	}
	return parsed, nil
}
