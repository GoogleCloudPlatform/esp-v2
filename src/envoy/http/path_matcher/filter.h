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

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace PathMatcher {

class Filter : public Http::PassThroughDecoderFilter,
               public Logger::Loggable<Logger::Id::filter> {
 public:
  Filter(FilterConfigSharedPtr config) : config_(config) {}

  Http::FilterHeadersStatus decodeHeaders(Http::RequestHeaderMap&,
                                          bool) override;

 private:
  void rejectRequest(Http::Code code, absl::string_view error_msg);

  const FilterConfigSharedPtr config_;
};

}  // namespace PathMatcher
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
