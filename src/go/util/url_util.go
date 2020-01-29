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

package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ParseURI parses uri into scheme, hostname, port, path with err(if exist).
// If uri has no scheme, it will be regarded as https.
func ParseURI(uri string) (string, string, uint32, string, error) {
	arr := strings.Split(uri, "://")
	scheme := "https"
	if len(arr) == 1 {
		uri = fmt.Sprintf("%s://%s", scheme, uri)
	} else {
		scheme = arr[0]
	}

	u, err := url.Parse(uri)
	if err != nil {
		return "", "", 0, "", err
	}

	_, port, _ := net.SplitHostPort(u.Host)
	if port == "" {
		port = "443"
		if !strings.HasSuffix(u.Scheme, "s") {
			port = "80"
		}
	}

	portVal, err := strconv.Atoi(port)
	if err != nil {
		return "", "", 0, "", err
	}
	return scheme, u.Hostname(), uint32(portVal), strings.TrimSuffix(u.RequestURI(), "/"), nil
}

// ParseBackendPreotocol parses a protocol string into BackendProtocl and UseTLS bool
func ParseBackendProtocol(protocol string) (BackendProtocol, bool, error) {
	protocol = strings.ToLower(protocol)
	var tls bool
	if strings.HasSuffix(protocol, "s") {
		tls = true
		protocol = strings.TrimSuffix(protocol, "s")
	}

	switch protocol {
	case "http":
		return HTTP1, tls, nil
	case "http1":
		return HTTP1, tls, nil
	case "http2":
		return HTTP2, tls, nil
	case "grpc":
		return GRPC, tls, nil
	default:
		return HTTP1, tls, fmt.Errorf(`unknown backend protocol [%v], should be one of "grpc", "http", "http1" or "http2"`, protocol)
	}
}

// Note: the path of openID discovery may be https
var getRemoteContent = func(path string) ([]byte, error) {
	req, _ := http.NewRequest("GET", path, nil)
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Fetching JwkUri returns not 200 OK: %v", resp.Status)
	}
	return ioutil.ReadAll(resp.Body)
}

func ResolveJwksUriUsingOpenID(uri string) (string, error) {
	if !strings.HasPrefix(uri, "http") {
		uri = fmt.Sprintf("https://%s", uri)
	}
	uri = strings.TrimSuffix(uri, "/")
	uri = fmt.Sprintf("%s%s", uri, OpenIDDiscoveryCfgURLSuffix)

	body, err := getRemoteContent(uri)
	if err != nil {
		return "", fmt.Errorf("Failed to fetch jwks_uri from %s: %v", uri, err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	jwksURI, ok := data["jwks_uri"].(string)
	if !ok {
		return "", fmt.Errorf("Invalid jwks_uri %v in openID discovery configuration", data["jwks_uri"])
	}
	return jwksURI, nil
}

func IamIdentityTokenSuffix(IamServiceAccount string) string {
	return fmt.Sprintf("/v1/projects/-/serviceAccounts/%s:generateIdToken", IamServiceAccount)
}

func IamAccessTokenSuffix(IamServiceAccount string) string {
	return fmt.Sprintf("/v1/projects/-/serviceAccounts/%s:generateAccessToken", IamServiceAccount)
}

func ExtraAddressFromURI(jwksUri string) (string, error) {
	_, hostname, port, _, err := ParseURI(jwksUri)
	if err != nil {
		return "", fmt.Errorf("Fail to parse uri %s with error %v", jwksUri, err)
	}
	return fmt.Sprintf("%s:%v", hostname, port), nil
}
