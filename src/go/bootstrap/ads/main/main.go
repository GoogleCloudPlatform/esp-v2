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
	"io/ioutil"

	"cloudesf.googlesource.com/gcpproxy/src/go/bootstrap"
	"cloudesf.googlesource.com/gcpproxy/src/go/bootstrap/ads"
	"cloudesf.googlesource.com/gcpproxy/src/go/bootstrap/ads/flags"
	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"

	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
)

func main() {
	flag.Parse()
	outPath := flag.Arg(0)
	glog.Infof("Output path: %s", outPath)
	if outPath == "" {
		glog.Exitf("Please specify a path to write bootstrap config file")
	}

	opts := flags.DefaultBootstrapperOptionsFromFlags()

	// Parse the ADS address
	_, adsHostname, adsPort, _, err := ut.ParseURI(opts.DiscoveryAddress)
	if err != nil {
		glog.Exitf("failed to parse discovery address: %v", err)
	}

	connectTimeoutProto := ptypes.DurationProto(opts.AdsConnectTimeout)
	bt, err := ads.CreateBootstrapConfig(connectTimeoutProto, adsHostname, adsPort, uint32(opts.AdminPort))
	if err != nil {
		glog.Exitf("failed to create bootstrap config, error: %v", err)
	}

	if opts.EnableTracing {
		if bt.Tracing, err = bootstrap.CreateTracing(opts.CommonOptions); err != nil {
			glog.Exitf("failed to create tracing config, error: %v", err)
		}
	}

	marshaler := &jsonpb.Marshaler{
		Indent: "  ",
	}
	jsonStr, _ := marshaler.MarshalToString(bt)
	err = ioutil.WriteFile(outPath, []byte(jsonStr), 0644)
	if err != nil {
		glog.Exitf("failed to write config to %v, error: %v", outPath, err)
	}
}
