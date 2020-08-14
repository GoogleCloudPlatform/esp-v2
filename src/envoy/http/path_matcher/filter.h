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

#include "extensions/filters/http/common/pass_through_filter.h"
#include "src/envoy/http/path_matcher/filter_config.h"

#include <string>

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace path_matcher {

class Filter : public Envoy::Http::PassThroughFilter,
               public Envoy::Logger::Loggable<Envoy::Logger::Id::filter> {
 public:
  Filter(FilterConfigSharedPtr config) : config_(config) {}

  Envoy::Http::FilterHeadersStatus decodeHeaders(Envoy::Http::RequestHeaderMap&,
                                                 bool) override;
  Envoy::Http::FilterHeadersStatus encodeHeaders(
      Envoy::Http::ResponseHeaderMap& headers, bool) override;

 private:
  void rejectRequest(Envoy::Http::Code code, absl::string_view error_msg);

  const FilterConfigSharedPtr config_;
};

}  // namespace path_matcher
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
