// Copyright 2019 Google Cloud Platform Proxy Authors
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
	"time"

	"cloudesf.googlesource.com/gcpproxy/src/go/options"
)

var (
	// Any flags in this file are used by both the ADS Bootstrapper (startup) and Config Generation via the static bootstrapper or config manager.
	// These flags are kept in sync with options.CommonOptions.
	// When adding or changing default values, update options.DefaultCommonOptions.

	AdminPort                 = flag.Int("admin_port", 8001, "Port that envoy should serve the admin page on")
	EnableTracing             = flag.Bool("enable_tracing", false, `enable stackdriver tracing`)
	HttpRequestTimeout        = flag.Duration("http_request_timeout", 5*time.Second, `Set the timeout in second for all requests. Must be > 0 and the default is 5 seconds if not set.`)
	NonGCP                    = flag.Bool("non_gcp", false, `By default, the proxy tries to talk to GCP metadata server to get VM location in the first few requests. Setting this flag to true to skip this step`)
	TracingProjectId          = flag.String("tracing_project_id", "", "The Google project id required for Stack driver tracing. If not set, will automatically use fetch it from GCP Metadata server")
	TracingStackdriverAddress = flag.String("tracing_stackdriver_address", "", "By default, the Stackdriver exporter will connect to production Stackdriver. If this is non-empty, it will connect to this address. It must be in the gRPC format.")
	TracingSamplingRate       = flag.Float64("tracing_sample_rate", 0.001, "tracing sampling rate from 0.0 to 1.0")
	TracingIncomingContext    = flag.String("tracing_incoming_context", "", "comma separated incoming trace contexts (traceparent|grpc-trace-bin|x-cloud-trace-context)")
	TracingOutgoingContext    = flag.String("tracing_outgoing_context", "", "comma separated outgoing trace contexts (traceparent|grpc-trace-bin|x-cloud-trace-context)")
)

func DefaultCommonOptionsFromFlags() options.CommonOptions {
	return options.CommonOptions{
		AdminPort:                 *AdminPort,
		EnableTracing:             *EnableTracing,
		HttpRequestTimeout:        *HttpRequestTimeout,
		NonGCP:                    *NonGCP,
		TracingProjectId:          *TracingProjectId,
		TracingStackdriverAddress: *TracingStackdriverAddress,
		TracingSamplingRate:       *TracingSamplingRate,
		TracingIncomingContext:    *TracingIncomingContext,
		TracingOutgoingContext:    *TracingOutgoingContext,
	}
}
