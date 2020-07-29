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

package static

import (
	"fmt"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/bootstrap"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"

	gen "github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator"
	sc "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	bootstrappb "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// ServiceToBootstrapConfig outputs envoy bootstrap config from service config.
// id is the service configuration ID. It is generated when deploying
// service config to ServiceManagement Server, example: 2017-02-13r0.
func ServiceToBootstrapConfig(serviceConfig *confpb.Service, id string, opts options.ConfigGeneratorOptions) (*bootstrappb.Bootstrap, error) {
	bt := &bootstrappb.Bootstrap{
		Node:  bootstrap.CreateNode(opts.CommonOptions),
		Admin: bootstrap.CreateAdmin(opts.CommonOptions),
	}

	serviceInfo, err := sc.NewServiceInfoFromServiceConfig(serviceConfig, id, opts)
	if err != nil {
		return nil, fmt.Errorf("fail to initialize ServiceInfo, %s", err)
	}

	clusters, err := gen.MakeClusters(serviceInfo)
	if err != nil {
		return nil, err
	}
	listeners, err := gen.MakeListeners(serviceInfo)
	if err != nil {
		return nil, err
	}

	bt.StaticResources = &bootstrappb.Bootstrap_StaticResources{
		Listeners: listeners,
		Clusters:  clusters,
	}
	return bt, nil
}
