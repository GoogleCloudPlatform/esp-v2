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

#include "api/envoy/http/error_translator/config.pb.h"
#include "common/common/logger.h"
#include "envoy/server/filter_config.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace error_translator {

/**
 * All stats for the backend routing filter. @see stats_macros.h
 */

// clang-format off
#define ALL_ERROR_TRANSLATOR_FILTER_STATS(COUNTER)     \
  COUNTER(append_path_to_address_request)             \
  COUNTER(constant_address_request)                   \
// clang-format on

/**
 * Wrapper struct for backend routing filter stats. @see stats_macros.h
 */
struct FilterStats {
  ALL_ERROR_TRANSLATOR_FILTER_STATS(GENERATE_COUNTER_STRUCT)
};

// The Envoy filter config for ESPv2 error translator filter.
class FilterConfig : public Envoy::Logger::Loggable<Envoy::Logger::Id::filter> {
 public:
  FilterConfig(const ::google::api::envoy::http::error_translator::FilterConfig&
                   proto_config,
               const std::string& stats_prefix,
               Envoy::Server::Configuration::FactoryContext& context)
      : proto_config_(proto_config),
        stats_(generateStats(stats_prefix, context.scope())) {
  }
  
  FilterStats& stats() { return stats_; }

  bool shouldScrubDebugDetails() {
    return proto_config_.client_error_visibility() != ::google::api::envoy::http::error_translator::FilterConfig::ClientErrorVisibility::FilterConfig_ClientErrorVisibility_FULL_DETAILS;
  }

 private:
  FilterStats generateStats(const std::string& prefix, Envoy::Stats::Scope& scope) {
    const std::string final_prefix = prefix + "error_translator.";
    return {ALL_ERROR_TRANSLATOR_FILTER_STATS(
        POOL_COUNTER_PREFIX(scope, final_prefix))};
  }

  // The config proto
  ::google::api::envoy::http::error_translator::FilterConfig proto_config_;
  // The stats
  FilterStats stats_;
};

typedef std::shared_ptr<FilterConfig> FilterConfigSharedPtr;

}  // namespace BackendRouting
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy

