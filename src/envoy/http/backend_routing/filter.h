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

#pragma once

#include <string>

#include "common/common/logger.h"
#include "envoy/http/filter.h"
#include "envoy/http/header_map.h"
#include "extensions/filters/http/common/pass_through_filter.h"
#include "src/envoy/http/backend_routing/filter_config.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace backend_routing {

class Filter : public Envoy::Http::PassThroughDecoderFilter,
               public Envoy::Logger::Loggable<Envoy::Logger::Id::filter> {
 public:
  Filter(FilterConfigSharedPtr config);

  // Envoy::Http::StreamDecoderFilter
  Envoy::Http::FilterHeadersStatus decodeHeaders(Envoy::Http::RequestHeaderMap&,
                                                 bool) override;

 private:
  const FilterConfigSharedPtr config_;
};

}  // namespace backend_routing
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
