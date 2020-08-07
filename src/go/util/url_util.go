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
	"regexp"
	"strconv"
	"strings"
)

const (
	// Default port for HTTP.
	HTTPDefaultPort = "80"

	// Default port for HTTPS.
	HTTPSDefaultPort = "443"

	// Default port for DNS.
	DNSDefaultPort = "53"
)

var (
	// Various hacky regular expressions to match a subset of the http template syntax.
	// Replace segments with single wildcards: /v1/books/*
	singleWildcardMatcher = regexp.MustCompile(`/\*`)
	// Replace segments with double wildcards: /v1/**
	doubleWildcardMatcher = regexp.MustCompile(`/\*\*`)
	// Replace any path templates: /v1/books/{book_id}
	pathParamMatcher = regexp.MustCompile(`/{[^{}]+}`)
	// Replace path templates with double wildcards: /v1/{name=**}
	pathParamDoubleWildcardMatcher = regexp.MustCompile(`/{[^{}]+=\*\*}`)

	// Common regex forms that emulate http template syntax.
	// Matches 1 or more segments of any character except '/'.
	singleWildcardReplacementRegex = `/[^\/]+`
	// Matches any character or no characters at all.
	// Note this also allows paths without a '/' suffix.
	doubleWildcardReplacementRegex = `.*`
)

// ParseURI parses uri into scheme, hostname, port, path with err(if exist).
// If uri has no scheme, it will be regarded as https.
// If uri has no port, it will use 80 for non-TLS and 443 for TLS.
// Ensures the path has no trailing slash.
// Strips out query parameters from the path.
func ParseURI(uri string) (string, string, uint32, string, error) {
	arr := strings.Split(uri, "://")
	if len(arr) == 1 {
		// Set the default scheme.
		uri = fmt.Sprintf("https://%s", uri)
	}

	u, err := url.Parse(uri)
	if err != nil {
		return "", "", 0, "", err
	}

	_, port, _ := net.SplitHostPort(u.Host)
	if port == "" {
		// Determine the default port.
		port = HTTPSDefaultPort
		if !strings.HasSuffix(u.Scheme, "s") {
			port = HTTPDefaultPort
		}
	}

	portVal, err := strconv.Atoi(port)
	if err != nil {
		return "", "", 0, "", err
	}

	pathNoTrailingSlash := strings.TrimSuffix(u.Path, "/")
	return u.Scheme, u.Hostname(), uint32(portVal), pathNoTrailingSlash, nil
}

// ParseBackendProtocol parses a scheme string and http protocol string into BackendProtocol and UseTLS bool.
func ParseBackendProtocol(scheme string, httpProtocol string) (BackendProtocol, bool, error) {
	scheme = strings.ToLower(scheme)
	httpProtocol = strings.ToLower(httpProtocol)

	// Default tls to false, even if scheme is invalid.
	tls := false
	if strings.HasSuffix(scheme, "s") {
		tls = true
		scheme = strings.TrimSuffix(scheme, "s")
	}

	switch scheme {
	case "http":
		// Default the http protocol to http/1.1.
		switch httpProtocol {
		case "", "http/1.1":
			return HTTP1, tls, nil
		case "h2":
			return HTTP2, tls, nil
		default:
			return UNKNOWN, tls, fmt.Errorf(`unknown backend http protocol [%v], should be one of "http/1.1", "h2", or not set`, httpProtocol)
		}
	case "grpc":
		return GRPC, tls, nil
	default:
		return UNKNOWN, tls, fmt.Errorf(`unknown backend scheme [%v], should be one of "http(s)" or "grpc(s)"`, scheme)
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

// Returns a regex that will match requests to the uri with path parameters or wildcards.
// If there are no path params or wildcards, returns empty string.
//
// Essentially matches a subset of the http template syntax.
// FIXME(nareddyt): Remove this hack completely when envoy route config supports path matching with path templates.
func WildcardMatcherForPath(uri string) string {

	// Ordering matters, start with most specific and work upwards.
	matcher := pathParamDoubleWildcardMatcher.ReplaceAllString(uri, doubleWildcardReplacementRegex)
	matcher = pathParamMatcher.ReplaceAllString(matcher, singleWildcardReplacementRegex)
	matcher = doubleWildcardMatcher.ReplaceAllString(matcher, doubleWildcardReplacementRegex)
	matcher = singleWildcardMatcher.ReplaceAllString(matcher, singleWildcardReplacementRegex)

	if matcher == uri {
		return ""
	}

	// Enforce strict prefix / suffix.
	return "^" + matcher + "$"
}

var (
	FetchRolloutIdURL = func(serviceControlUrl, serviceName string) string {
		return fmt.Sprintf("%v/v1/services/%s:report",
			serviceControlUrl, serviceName)
	}

	FetchRolloutsURL = func(serviceManagementUrl, serviceName string) string {
		return fmt.Sprintf("%s/v1/services/%s/rollouts?filter=status=SUCCESS",
			serviceManagementUrl, serviceName)
	}

	FetchConfigURL = func(serviceManagementUrl, serviceName, configId string) string {
		return fmt.Sprintf("%s/v1/services/%s/configs/%s?view=FULL",
			serviceManagementUrl, serviceName, configId)
	}
)
