// Copyright 2019 Google Cloud Platform Proxy Authors
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
	"testing"
)

func TestParseURL(t *testing.T) {
	testData := []struct {
		desc           string
		url            string
		wantedHostname string
		wantedPort     uint32
		wantURI        string
		wantErr        string
	}{
		{
			desc:           "successful for https url, ends without slash",
			url:            "https://abc.example.org",
			wantedHostname: "abc.example.org",
			wantedPort:     443,
			wantURI:        "",
			wantErr:        "",
		},
		{
			desc:           "successful for https url, ends with slash",
			url:            "https://abcde.google.org/",
			wantedHostname: "abcde.google.org",
			wantedPort:     443,
			wantURI:        "",
			wantErr:        "",
		},
		{
			desc:           "successful for https url, ends with path",
			url:            "https://abcde.youtube.com/api/",
			wantedHostname: "abcde.youtube.com",
			wantedPort:     443,
			wantURI:        "/api",
			wantErr:        "",
		},
		{
			desc:           "successful for https url with custom port",
			url:            "https://abcde.youtube.com:8989/api/",
			wantedHostname: "abcde.youtube.com",
			wantedPort:     8989,
			wantURI:        "/api",
			wantErr:        "",
		},
		{
			desc:           "fail for http url",
			url:            "http://abcde.youtube.com:8989/api/",
			wantedHostname: "",
			wantedPort:     0,
			wantURI:        "",
			wantErr:        "dynamic routing only supports HTTPS",
		},
		{
			desc:           "fail for https url with IP address",
			url:            "https://192.168.0.1/api/",
			wantedHostname: "",
			wantedPort:     0,
			wantURI:        "",
			wantErr:        "dynamic routing only supports domain name, got IP address: 192.168.0.1",
		},
		{
			desc:           "successful for https url, path ends with slash",
			url:            "https://abc.example.org/path/to/",
			wantedHostname: "abc.example.org",
			wantedPort:     443,
			wantURI:        "/path/to",
			wantErr:        "",
		},
		{
			desc:           "successful for https url, path ends without slash",
			url:            "https://abc.example.org/path",
			wantedHostname: "abc.example.org",
			wantedPort:     443,
			wantURI:        "/path",
			wantErr:        "",
		},
	}

	for i, tc := range testData {
		hostname, port, uri, err := ParseURL(tc.url)
		if hostname != tc.wantedHostname {
			t.Errorf("Test Desc(%d): %s, extract backend address got: %v, want: %v", i, tc.desc, hostname, tc.wantedHostname)
		}
		if port != tc.wantedPort {
			t.Errorf("Test Desc(%d): %s, extract backend address got: %v, want: %v", i, tc.desc, port, tc.wantedPort)
		}
		if uri != tc.wantURI {
			t.Errorf("Test Desc(%d): %s, extract backend address got: %v, want: %v", i, tc.desc, uri, tc.wantURI)
		}
		if (err == nil && tc.wantErr != "") || (err != nil && tc.wantErr == "") {
			t.Errorf("Test Desc(%d): %s, extract backend address got: %v, want: %v", i, tc.desc, err, tc.wantErr)
		}
	}
}
