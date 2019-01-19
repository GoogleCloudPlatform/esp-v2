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

// TODO(kyuc): Add unit tests and integration tests.

#include <string>

#include "src/envoy/http/backend_auth/filter.h"

#include "src/envoy/utils/metadata_utils.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace BackendAuth {

using ::google::protobuf::util::Status;
using Http::FilterDataStatus;
using Http::FilterHeadersStatus;
using Http::FilterTrailersStatus;
using Http::HeaderMap;
using Http::LowerCaseString;
using Utils::getOperationFromMetadata;

namespace {
constexpr char kBearer[] = "Bearer ";
}  // namespace

FilterHeadersStatus Filter::decodeHeaders(HeaderMap& headers, bool) {
  const std::string& operation = getOperationFromMetadata(
      decoder_callbacks_->streamInfo().dynamicMetadata(), "");
  if (operation == "") {
    rejectRequest(Http::Code(404), "No operation found from DynamicMetadata.");
    return Http::FilterHeadersStatus::Continue;
  }
  ENVOY_LOG(debug, "find operation: {}", operation);
  const TokenSharedPtr jwt_token = config_->cfg_parser().getJwtToken(operation);

  if (!jwt_token) {
    rejectRequest(Http::Code(401), "No JWT token found for operation.");
    return Http::FilterHeadersStatus::StopIteration;
  }
  const auto& authorization = Http::Headers::get().Authorization;
  headers.remove(authorization);
  headers.addCopy(authorization, kBearer + *jwt_token);
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

}  // namespace BackendAuth
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
