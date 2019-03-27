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

package util

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// parameter: URL, return value: hostname, port, path, error (not nil if parse URL fails)
func ParseURL(address string) (string, uint32, string, error) {
	backendUrl, err := url.Parse(address)
	if err != nil {
		return "", 0, "", err
	}
	if backendUrl.Scheme != "https" {
		return "", 0, "", fmt.Errorf("dynamic routing only supports HTTPS")
	}
	hostname := backendUrl.Hostname()
	if net.ParseIP(hostname) != nil {
		return "", 0, "", fmt.Errorf("dynamic routing only supports domain name, got IP address: %v", hostname)
	}
	var port uint32 = 443
	if backendUrl.Port() != "" {
		// for cases like "https://example.org:8080"
		var port64 uint64
		var err error
		if port64, err = strconv.ParseUint(backendUrl.Port(), 10, 32); err != nil {
			return "", 0, "", err
		}
		port = uint32(port64)
	}
	// if uri ends with a slash like "/getUser/" or "/", remove last slash
	uri := strings.TrimSuffix(backendUrl.RequestURI(), "/")
	return hostname, port, uri, nil
}
