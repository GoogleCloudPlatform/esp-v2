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

#include <string>

#include "src/envoy/http/path_matcher/filter.h"

#include "common/common/logger.h"
#include "common/http/utility.h"
#include "common/protobuf/utility.h"
#include "envoy/server/filter_config.h"
#include "src/envoy/http/path_matcher/filter_config.h"
#include "src/envoy/utils/metadata_utils.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace PathMatcher {

using ::google::protobuf::util::Status;
using Http::FilterDataStatus;
using Http::FilterHeadersStatus;
using Http::FilterTrailersStatus;
using Http::HeaderMap;
using Http::LowerCaseString;

void Filter::onDestroy() {}

FilterHeadersStatus Filter::decodeHeaders(HeaderMap& headers, bool) {
  const std::string* operation = config_->FindOperation(
      headers.Method()->value().c_str(), headers.Path()->value().c_str());
  if (operation == nullptr) {
    rejectRequest(Http::Code(404),
                  "Path does not match any requirement uri_template.");
    return Http::FilterHeadersStatus::StopIteration;
  }
  ENVOY_LOG(debug, "matched operation: {}", *operation);
  Utils::setOperationToMetadata(
      decoder_callbacks_->streamInfo().dynamicMetadata(), *operation);
  config_->stats().allowed_.inc();
  return FilterHeadersStatus::Continue;
}

FilterDataStatus Filter::decodeData(Buffer::Instance&, bool) {
  return FilterDataStatus::Continue;
}

FilterTrailersStatus Filter::decodeTrailers(HeaderMap&) {
  return FilterTrailersStatus::Continue;
}

void Filter::setDecoderFilterCallbacks(
    Http::StreamDecoderFilterCallbacks& callbacks) {
  decoder_callbacks_ = &callbacks;
}

void Filter::rejectRequest(Http::Code code, absl::string_view error_msg) {
  config_->stats().denied_.inc();

  decoder_callbacks_->sendLocalReply(code, error_msg, nullptr, absl::nullopt);
  decoder_callbacks_->streamInfo().setResponseFlag(
      StreamInfo::ResponseFlag::UnauthorizedExternalService);
}

}  // namespace PathMatcher
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy