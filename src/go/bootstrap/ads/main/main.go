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

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"time"

	"cloudesf.googlesource.com/gcpproxy/src/go/bootstrap"
	"cloudesf.googlesource.com/gcpproxy/src/go/bootstrap/ads"
	"cloudesf.googlesource.com/gcpproxy/src/go/metadata"
	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
)

var (
	AdsConnectTimeout = flag.Duration("ads_connect_imeout", 10*time.Second, "ads connect timeout in seconds")
	EnableTracing     = flag.Bool("enable_tracing", false, `enable stackdriver tracing`)
	NonGCP            = flag.Bool("non_gcp", false, `By default, the proxy tries to talk to GCP metadata server to get VM location in the first few requests. Setting this flag
  to true to skip this step`)
	TracingProjectId       = flag.String("tracing_project_id", "", "The Google project id required for Stack driver tracing. If not set, will automatically use fetch it from GCP Metadata server")
	MetadataFetcherTimeout = *flag.Duration("http_request_timeout", 5*time.Second, `Set the timeout in second for all requests made by config manager. Must be > 0 and the default is 5 seconds if not set.`)
)

func main() {
	flag.Parse()
	out_path := flag.Arg(0)
	glog.Infof("Output path: %s", out_path)
	if out_path == "" {
		glog.Exitf("Please specify a path to write bootstrap config file")
	}

	connectTimeoutProto := ptypes.DurationProto(*AdsConnectTimeout)
	bt := ads.CreateBootstrapConfig(connectTimeoutProto)

	if *EnableTracing {

		tracingProjectId, err := getTracingProjectId()
		if err != nil {
			glog.Exitf("failed to get project-id for tracing, error: %v", err)
		}

		if bt.Tracing, err = bootstrap.CreateTracing(tracingProjectId); err != nil {
			glog.Exitf("failed to create tracing config, error: %v", err)
		}
	}

	marshaler := &jsonpb.Marshaler{
		Indent: "  ",
	}
	json_str, _ := marshaler.MarshalToString(bt)
	err := ioutil.WriteFile(out_path, []byte(json_str), 0644)
	if err != nil {
		glog.Exitf("failed to write config to %v, error: %v", out_path, err)
	}
}

func getTracingProjectId() (string, error) {

	// If user specified a project-id, use that
	projectId := *TracingProjectId
	if projectId != "" {
		return projectId, nil
	}

	// Otherwise determine project-id automatically
	glog.Infof("tracing_project_id was not specified, attempting to fetch it from GCP Metadata server")
	if *NonGCP {
		return "", fmt.Errorf("tracing_project_id was not specified and can not be fetched from GCP Metadata server on non-GCP runtime")
	}

	return metadata.NewMetadataFetcher(MetadataFetcherTimeout).FetchProjectId()
}
