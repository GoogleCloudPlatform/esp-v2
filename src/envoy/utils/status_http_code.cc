// Copyright 2018 Google Cloud Platform Proxy Authors
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

#include "src/envoy/utils/status_http_code.h"
#include "google/protobuf/stubs/status.h"

using StatusCode = ::google::protobuf::util::error::Code;

namespace Envoy {
namespace Extensions {
namespace Utils {

Http::Code statusToHttpCode(int code) {
  // Map Canonical codes to HTTP status codes. This is based on the mapping
  // defined by the protobuf http error space.
  switch (code) {
    case StatusCode::OK:
      return Http::Code::OK;  // 200
    case StatusCode::CANCELLED:
      return Http::Code(499);  // 499  not defined in Envoy
    case StatusCode::UNKNOWN:
      return Http::Code::InternalServerError;  // 500
    case StatusCode::INVALID_ARGUMENT:
      return Http::Code::BadRequest;  // 400
    case StatusCode::DEADLINE_EXCEEDED:
      return Http::Code::GatewayTimeout;  // 504
    case StatusCode::NOT_FOUND:
      return Http::Code::NotFound;  // 404
    case StatusCode::ALREADY_EXISTS:
      return Http::Code::Conflict;  // 409
    case StatusCode::PERMISSION_DENIED:
      return Http::Code::Forbidden;  // 403
    case StatusCode::RESOURCE_EXHAUSTED:
      return Http::Code::TooManyRequests;  // 429
    case StatusCode::FAILED_PRECONDITION:
      return Http::Code::BadRequest;  // 400
    case StatusCode::ABORTED:
      return Http::Code::Conflict;  // 409
    case StatusCode::OUT_OF_RANGE:
      return Http::Code::BadRequest;  // 400
    case StatusCode::UNIMPLEMENTED:
      return Http::Code::NotImplemented;  // 501
    case StatusCode::INTERNAL:
      return Http::Code::InternalServerError;  // 500
    case StatusCode::UNAVAILABLE:
      return Http::Code::ServiceUnavailable;  // 503
    case StatusCode::DATA_LOSS:
      return Http::Code::InternalServerError;  // 500
    case StatusCode::UNAUTHENTICATED:
      return Http::Code::Unauthorized;  // 401
    default:
      return Http::Code::InternalServerError;  // 500
  }
}

}  // namespace Utils
}  // namespace Extensions
}  // namespace Envoy
