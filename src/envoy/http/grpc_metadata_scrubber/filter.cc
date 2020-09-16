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

#include <string>

#include "src/envoy/http/grpc_metadata_scrubber/filter.h"

#include "common/grpc/common.h"
#include "common/http/headers.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace grpc_metadata_scrubber {

Envoy::Http::FilterHeadersStatus Filter::encodeHeaders(
    Envoy::Http::ResponseHeaderMap& headers, bool) {
  ENVOY_LOG(debug, "Filter::encodeHeaders is called.");
  config_->stats().all_.inc();

  if (Envoy::Grpc::Common::hasGrpcContentType(headers) && headers.ContentLength() != nullptr) {
    ENVOY_LOG(debug, "Content-length header is removed");
    headers.removeContentLength();
    config_->stats().removed_.inc();
  }

  return Envoy::Http::FilterHeadersStatus::Continue;
}

}  // namespace grpc_metadata_scrubber
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
