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

package gcsrunner

import (
	"bytes"
	"fmt"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/metadata"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"

	scpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/service_control"
	v2pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	listenerpb "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	bootstrappb "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
)

var (
	// Transformers which can be stubbed in unit testing.
	doServiceControlTransform = addGCPAttributes
	doListenerTransform       = replaceListenerPort
)

func addGCPAttributes(cfg *scpb.FilterConfig, opts FetchConfigOptions) error {
	co := options.DefaultCommonOptions()
	co.MetadataURL = opts.MetadataURL
	mf := metadata.NewMetadataFetcher(co)
	attrs, err := mf.FetchGCPAttributes()
	if err != nil {
		return err
	}
	if opts.OverridePlatform != "" {
		attrs.Platform = opts.OverridePlatform
	}
	cfg.GcpAttributes = attrs
	return nil
}

// replaceListenerPort replaces the listener port with opts.WantPort if specified.
func replaceListenerPort(l *v2pb.Listener, opts FetchConfigOptions) error {
	if opts.WantPort == 0 {
		return nil
	}
	if addr := l.GetAddress().GetSocketAddress(); addr != nil {
		portSpecifier := addr.GetPortSpecifier()
		if portValue, ok := portSpecifier.(*corepb.SocketAddress_PortValue); ok {
			if portValue.PortValue != opts.ReplacePort {
				return fmt.Errorf("listener has port value %d but wanted %d", portValue.PortValue, opts.ReplacePort)
			}
			portValue.PortValue = opts.WantPort
			return nil
		}
	}
	return fmt.Errorf("expected a listener with port value %d but got none: %v", opts.ReplacePort, l)
}

func transformConfigBytes(config []byte, opts FetchConfigOptions) ([]byte, error) {
	bootstrap := &bootstrappb.Bootstrap{}
	u := &jsonpb.Unmarshaler{
		AnyResolver: util.Resolver,
	}
	if err := u.Unmarshal(bytes.NewBuffer(config), bootstrap); err != nil {
		return nil, err
	}

	if err := transformEnvoyConfig(bootstrap, opts); err != nil {
		return nil, err
	}

	m := &jsonpb.Marshaler{
		OrigName:    true,
		AnyResolver: util.Resolver,
	}
	buf := &bytes.Buffer{}
	if err := m.Marshal(buf, bootstrap); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func transformEnvoyConfig(bootstrap *bootstrappb.Bootstrap, opts FetchConfigOptions) error {
	listeners := bootstrap.GetStaticResources().GetListeners()
	if len(listeners) != 1 {
		return fmt.Errorf("expected exactly 1 listener, got: %d", len(listeners))
	}
	if err := doListenerTransform(listeners[0], opts); err != nil {
		return err
	}
	httpConMgrTransformed := false
	for _, c := range listeners[0].GetFilterChains() {
		if filters := c.GetFilters(); filters != nil {
			for _, f := range filters {
				if f.GetName() == util.HTTPConnectionManager {
					if err := transformHTTPConnectionManager(f, opts); err != nil {
						return fmt.Errorf("failed to transform HttpConnectionManager: %v", err)
					}
					httpConMgrTransformed = true
				}
			}
		}
	}
	if !httpConMgrTransformed {
		return fmt.Errorf("did not find an http connection manager filter: %v", listeners[0])
	}
	return nil
}

func transformHTTPConnectionManager(f *listenerpb.Filter, opts FetchConfigOptions) error {
	hcmCfg := f.GetTypedConfig()
	httpConMgr := &hcmpb.HttpConnectionManager{}
	if err := ptypes.UnmarshalAny(hcmCfg, httpConMgr); err != nil {
		return err
	}
	transformed := false
	for _, hf := range httpConMgr.GetHttpFilters() {
		if hf.GetName() == util.ServiceControl {
			if err := transformServiceControlFilter(hf, opts); err != nil {
				return fmt.Errorf("failed to transform service control filter: %v", err)
			}
			transformed = true
		}
	}
	if !transformed {
		return fmt.Errorf("http connection manager did not find a service control filter: %v", f)
	}
	filterCfg, err := ptypes.MarshalAny(httpConMgr)
	if err != nil {
		return err
	}
	f.ConfigType = &listenerpb.Filter_TypedConfig{TypedConfig: filterCfg}
	return nil
}

func transformServiceControlFilter(f *hcmpb.HttpFilter, opts FetchConfigOptions) error {
	scCfg := f.GetTypedConfig()
	if scCfg == nil {
		return fmt.Errorf("failed to unmarshal service control filter as a typed config")
	}
	filterConfig := &scpb.FilterConfig{}
	if err := ptypes.UnmarshalAny(scCfg, filterConfig); err != nil {
		return err
	}

	if err := doServiceControlTransform(filterConfig, opts); err != nil {
		return fmt.Errorf("failed to add GCP attributes: %v", err)
	}

	scs, err := ptypes.MarshalAny(filterConfig)
	if err != nil {
		return err
	}
	f.ConfigType = &hcmpb.HttpFilter_TypedConfig{
		TypedConfig: scs,
	}
	return nil
}
