// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package options

import (
	"time"
)

// CommonOptions describes the possible overrides used by both the ADS bootstrapper and the config generator.
// By defining all the common options in one struct, we prevent duplicate flag initialization and reduce repeated code.
type CommonOptions struct {
	// Flags for envoy
	AdminAddress  string
	AdminPort     int
	DiscoveryPort int
	Node          string

	// Flags for tracing
	DisableTracing             bool
	TracingProjectId           string
	TracingStackdriverAddress  string
	TracingSamplingRate        float64
	TracingIncomingContext     string
	TracingOutgoingContext     string
	TracingMaxNumAttributes    int64
	TracingMaxNumAnnotations   int64
	TracingMaxNumMessageEvents int64
	TracingMaxNumLinks         int64

	// Flags for metadata
	NonGCP             bool
	HttpRequestTimeout time.Duration
	MetadataURL        string
	IamURL             string
	// Configures the identity used when making requests to Service Control.
	ServiceControlCredentials *IAMCredentialsOptions
	// Configures the identity used when making requests to backends.
	BackendAuthCredentials *IAMCredentialsOptions
}

// IamTokenKind specifies which type of token to generate using the IAM Credentials API.
type IamTokenKind int

const (
	// AccessToken indicates the access token should be generated.
	AccessToken IamTokenKind = iota
	// IDToken indicates the OpenID Connect ID token should be generated.
	IDToken
)

// IAMCredentialsOptions configures Envoy to authenticate requests using the given service account
// instead of the identity of the machine.
type IAMCredentialsOptions struct {
	// The Service Account to fetch the token for. If left empty, IAM Credentials API will not be used to sign tokens.
	ServiceAccountEmail string
	TokenKind           IamTokenKind
	// Optionally impersonate the ServiceAccountEmail using this chain of delegates. See:
	// https://cloud.google.com/iam/docs/reference/credentials/rest/v1/projects.serviceAccounts/generateIdToken
	Delegates []string
}

// DefaultCommonOptions returns CommonOptions with default values.
//
// The default values are expected to match the default values from the flags.
func DefaultCommonOptions() CommonOptions {
	return CommonOptions{
		AdminAddress:               "0.0.0.0",
		AdminPort:                  8001,
		DiscoveryPort:              8790,
		HttpRequestTimeout:         5 * time.Second,
		Node:                       "ESPv2",
		TracingSamplingRate:        0.001,
		TracingMaxNumAttributes:    32,
		TracingMaxNumAnnotations:   32,
		TracingMaxNumMessageEvents: 128,
		TracingMaxNumLinks:         128,
		MetadataURL:                "http://169.254.169.254/computeMetadata",
		IamURL:                     "https://iamcredentials.googleapis.com",
	}
}
