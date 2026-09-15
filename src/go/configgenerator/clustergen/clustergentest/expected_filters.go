// Copyright 2026 Google LLC
package clustergentest

import (
	"google.golang.org/protobuf/types/known/anypb"

	tcpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/trace_context"
	upstreampb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/upstream_codec/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
)

// ExpectedUpstreamFilters returns the HTTP filters that are appended to the Upstream clusters (tracing)
func ExpectedUpstreamFilters() []*hcmpb.HttpFilter {
	return []*hcmpb.HttpFilter{
		{
			Name: "com.google.espv2.filters.http.trace_context",
			ConfigType: &hcmpb.HttpFilter_TypedConfig{
				TypedConfig: MessageToAnyOrDie(&tcpb.TraceContextForwardedConfig{
					OutgoingContexts: []tcpb.TraceContextFormat{tcpb.TraceContextFormat_TRACE_CONTEXT, tcpb.TraceContextFormat_CLOUD_TRACE_CONTEXT},
				}),
			},
		},
		{
			Name: "envoy.extensions.filters.http.upstream_codec.v3.UpstreamCodec",
			ConfigType: &hcmpb.HttpFilter_TypedConfig{
				TypedConfig: MessageToAnyOrDieCodec(&upstreampb.UpstreamCodec{}),
			},
		},
	}
}

func MessageToAnyOrDie(msg *tcpb.TraceContextForwardedConfig) *anypb.Any {
	a, _ := anypb.New(msg)
	return a
}
func MessageToAnyOrDieCodec(msg *upstreampb.UpstreamCodec) *anypb.Any {
	a, _ := anypb.New(msg)
	return a
}
