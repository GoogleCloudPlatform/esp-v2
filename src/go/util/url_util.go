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
		if u.Scheme == "http" {
			port = "80"
		}
	}

	portVal, err := strconv.Atoi(port)
	if err != nil {
		return "", "", 0, "", err
	}
	return scheme, u.Hostname(), uint32(portVal), strings.TrimSuffix(u.RequestURI(), "/"), nil
}
