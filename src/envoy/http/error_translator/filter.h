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
#include "google/rpc/status.pb.h"
#include "src/envoy/http/error_translator/filter_config.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace error_translator {

class Filter : public Envoy::Http::PassThroughEncoderFilter,
               public Envoy::Logger::Loggable<Envoy::Logger::Id::filter> {
 public:
  Filter(FilterConfigSharedPtr config);

  Envoy::Http::FilterHeadersStatus encodeHeaders(
      Envoy::Http::ResponseHeaderMap& headers, bool) override;

  Envoy::Http::FilterDataStatus encodeData(Envoy::Buffer::Instance&,
                                           bool end_stream) override;

  bool isUpstreamResponse();

  bool isEspv2FilterError();

  std::string errorToJson(google::rpc::Status& error);

 private:
  const FilterConfigSharedPtr config_;

  // Current response is gRPC. Filter does not try to create a gRPC body.
  bool is_grpc_response_;

  // Stores the current error that is being translated.
  google::rpc::Status error_;

  // Caches the error converted to JSON.
  std::string error_json_;

  // Save the response headers so they can be modified in `encodeData`.
  Envoy::Http::ResponseHeaderMap* headers_;
};

}  // namespace error_translator
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
