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

package static

import (
	"fmt"

	bt "cloudesf.googlesource.com/gcpproxy/src/go/bootstrap"
	gen "cloudesf.googlesource.com/gcpproxy/src/go/configgenerator"
	sc "cloudesf.googlesource.com/gcpproxy/src/go/configinfo"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	boot "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// ServiceToBoostrapConfig outputs envoy bootstrap config from service config.
// id is the service configuration ID. It is generated when deploying
// service config to ServiceManagement Server, example: 2017-02-13r0.
func ServiceToBoostrapConfig(serviceConfig *conf.Service, id string) (*boot.Bootstrap, error) {
	bootstrap := &boot.Bootstrap{
		Node:  bt.CreateNode(),
		Admin: bt.CreateAdmin(),
	}

	serviceInfo, err := sc.NewServiceInfoFromServiceConfig(serviceConfig, id)
	if err != nil {
		return nil, fmt.Errorf("fail to initialize ServiceInfo, %s", err)
	}

	listener, err := gen.MakeListeners(serviceInfo)
	if err != nil {
		return nil, err
	}
	clusters, err := gen.MakeClusters(serviceInfo)
	if err != nil {
		return nil, err
	}

	bootstrap.StaticResources = &boot.Bootstrap_StaticResources{
		Listeners: []v2.Listener{*listener},
		Clusters:  clusters,
	}
	return bootstrap, nil
}
