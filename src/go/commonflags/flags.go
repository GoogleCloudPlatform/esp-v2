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

package commonflags

import (
	"flag"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/golang/glog"
)

var (
	// Any flags in this file are used by both the ADS Bootstrapper (startup) and Config Generation via the static bootstrapper or config manager.
	// These flags are kept in sync with options.CommonOptions.
	// When adding or changing default values, update options.DefaultCommonOptions.
	AdminAddress               = flag.String("admin_address", "0.0.0.0", "Address that envoy should serve the admin page on. Supports both ipv4 and ipv6 addresses.")
	DiscoveryPort              = flag.Int("discovery_port", 8790, "Port that envoy should use to contact ADS. Defaults to config manager's port.")
	DisableTracing             = flag.Bool("disable_tracing", false, `Disable stackdriver tracing`)
	AdminPort                  = flag.Int("admin_port", 8001, "Enables envoy's admin interface on this port if it is not 0. Not recommended for production use-cases, as the admin port is unauthenticated.")
	HttpRequestTimeoutS        = flag.Int("http_request_timeout_s", 5, `Set the timeout in second for all requests. Must be > 0 and the default is 5 seconds if not set.`)
	Node                       = flag.String("node", "ESPv2", "envoy node id")
	NonGCP                     = flag.Bool("non_gcp", false, `By default, the proxy tries to talk to GCP metadata server to get VM location in the first few requests. Setting this flag to true to skip this step`)
	TracingProjectId           = flag.String("tracing_project_id", "", "The Google project id required for Stack driver tracing. If not set, will automatically use fetch it from GCP Metadata server")
	TracingStackdriverAddress  = flag.String("tracing_stackdriver_address", "", "By default, the Stackdriver exporter will connect to production Stackdriver. If this is non-empty, it will connect to this address. It must be in the gRPC format.")
	TracingSamplingRate        = flag.Float64("tracing_sample_rate", 0.001, "tracing sampling rate from 0.0 to 1.0")
	TracingIncomingContext     = flag.String("tracing_incoming_context", "", "comma separated incoming trace contexts (traceparent|grpc-trace-bin|x-cloud-trace-context)")
	TracingOutgoingContext     = flag.String("tracing_outgoing_context", "", "comma separated outgoing trace contexts (traceparent|grpc-trace-bin|x-cloud-trace-context)")
	TracingMaxNumAttributes    = flag.Int64("tracing_max_num_attributes", 32, "Sets the maximum number of attributes that each span can contain. Defaults to the maximum allowed by Stackdriver. In practice, the number of attributes published will be much less.")
	TracingMaxNumAnnotations   = flag.Int64("tracing_max_num_annotations", 32, "Sets the maximum number of annotations that each span can contain. Defaults to the maximum allowed by Stackdriver. In practice, the number of annotations published will be much less.")
	TracingMaxNumMessageEvents = flag.Int64("tracing_max_num_message_events", 128, "Sets the maximum number of message events that each span can contain. Defaults to the maximum allowed by Stackdriver. In practice, the number of message events published will be much less.")
	TracingMaxNumLinks         = flag.Int64("tracing_max_num_links", 128, "Sets the maximum number of links that each span can contain. Defaults to the maximum allowed by Stackdriver. In practice, the number of links published will be much less.")

	//Suspected Envoy has listener initialization bug: if a http filter needs to use
	//a cluster with DSN lookup for initialization, e.g. fetching a remote access
	//token, the cluster is not ready so the whole listener is destroyed. ADS will
	//repeatedly send the same listener again until the cluster is ready. Then the
	//listener is marked as ready but the whole Envoy server is not marked as ready
	//(worker did not start) somehow. To work around this problem, use IP for
	//metadata server to fetch access token.
	MetadataURL = flag.String("metadata_url", "http://169.254.169.254/computeMetadata", "url of metadata server")
	IamURL      = flag.String("iam_url", "https://iamcredentials.googleapis.com", "url of iam server")

	ServiceControlIamServiceAccount = flag.String("service_control_iam_service_account", "", "The service account used to fetch access token for the Service Control from Google Cloud IAM")
	ServiceControlIamDelegates      = flag.String("service_control_iam_delegates", "", "The sequence of service accounts in a delegation chain used to fetch access token for the Service Control from Google Cloud IAM. The multiple delegates should be separated by \",\" and the flag only applies when ServiceControlIamServiceAccount is not empty.")

	BackendAuthIamServiceAccount = flag.String("backend_auth_iam_service_account", "", "The service account used to fetch identity token for the Backend Auth from Google Cloud IAM")
	BackendAuthIamDelegates      = flag.String("backend_auth_iam_delegates", "", "The sequence of service accounts in a delegation chain used to fetch identity token for the Backend Auth from Google Cloud IAM. The multiple delegates should be separated by \",\" and the flag only applies when BackendAuthIamServiceAccount is not empty.")
)

func DefaultCommonOptionsFromFlags() options.CommonOptions {
	opts := options.CommonOptions{
		AdminAddress:               *AdminAddress,
		AdminPort:                  *AdminPort,
		DisableTracing:             *DisableTracing,
		DiscoveryPort:              *DiscoveryPort,
		HttpRequestTimeout:         time.Duration(*HttpRequestTimeoutS) * time.Second,
		Node:                       *Node,
		NonGCP:                     *NonGCP,
		TracingProjectId:           *TracingProjectId,
		TracingStackdriverAddress:  *TracingStackdriverAddress,
		TracingSamplingRate:        *TracingSamplingRate,
		TracingIncomingContext:     *TracingIncomingContext,
		TracingOutgoingContext:     *TracingOutgoingContext,
		TracingMaxNumAttributes:    *TracingMaxNumAttributes,
		TracingMaxNumAnnotations:   *TracingMaxNumAnnotations,
		TracingMaxNumMessageEvents: *TracingMaxNumMessageEvents,
		TracingMaxNumLinks:         *TracingMaxNumLinks,
		MetadataURL:                *MetadataURL,
		IamURL:                     *IamURL,
	}
	if *BackendAuthIamServiceAccount != "" {
		opts.BackendAuthCredentials = &options.IAMCredentialsOptions{
			ServiceAccountEmail: *BackendAuthIamServiceAccount,
			TokenKind:           options.IDToken,
		}
		if *BackendAuthIamDelegates != "" {
			opts.BackendAuthCredentials.Delegates = strings.Split(*BackendAuthIamDelegates, ",")
		}
	}

	if *ServiceControlIamServiceAccount != "" {
		opts.ServiceControlCredentials = &options.IAMCredentialsOptions{
			ServiceAccountEmail: *ServiceControlIamServiceAccount,
			TokenKind:           options.AccessToken,
		}
		if *ServiceControlIamDelegates != "" {
			opts.ServiceControlCredentials.Delegates = strings.Split(*ServiceControlIamDelegates, ",")
		}
	}

	glog.Infof("Common options: %+v", opts)
	return opts
}
