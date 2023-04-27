// Copyright 2023 Google LLC
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

// Package clustergen provides individual Cluster Generators to generate an
// xDS cluster config.
package clustergen

import (
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// ClusterGenerator is an object that generates config for Envoy clusters.
type ClusterGenerator interface {

	// GetName returns the debug name of the cluster.
	// May differ from actual xDS cluster name.
	GetName() string

	// GenConfig generates the full xDS cluster config.
	GenConfig() (*clusterpb.Cluster, error)
}

// ClusterGeneratorOPFactory is the factory function to create an ordered slice
// of ClusterGenerator from One Platform config.
//
// The majority of factories will only return 1 ClusterGenerator, but they should
// be encapsulated by a slice for generalization.
type ClusterGeneratorOPFactory func(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) ([]ClusterGenerator, error)
