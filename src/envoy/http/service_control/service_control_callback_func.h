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
#include <functional>
#include "src/api_proxy/service_control/request_info.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace service_control {

// The function to be called when check call is completed.
using CheckDoneFunc = std::function<void(
    const ::google::protobuf::util::Status& status,
    const ::espv2::api_proxy::service_control::CheckResponseInfo&)>;

// The function to be called when allocateQuota call is completed.
using QuotaDoneFunc =
    std::function<void(const ::google::protobuf::util::Status& status)>;

// The function to cancel a on-going request.
using CancelFunc = std::function<void()>;

}  // namespace service_control
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
