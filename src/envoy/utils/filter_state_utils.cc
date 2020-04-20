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

#include "src/envoy/utils/filter_state_utils.h"
#include <memory>

#include "common/common/empty_string.h"
#include "common/router/string_accessor_impl.h"
#include "envoy/stream_info/filter_state.h"

namespace espv2 {
namespace envoy {
namespace utils {

using ::Envoy::Router::StringAccessor;
using ::Envoy::Router::StringAccessorImpl;
using ::Envoy::StreamInfo::FilterState;

// FilterState container needed to store the google.rpc.Status error proto.
struct RpcStatusWrapper : public Envoy::StreamInfo::FilterState::Object {
  google::rpc::Status status_;

  Envoy::ProtobufTypes::MessagePtr serializeAsProto() const override {
    return std::make_unique<google::rpc::Status>(status_);
  }
};

void setStringFilterState(FilterState& filter_state,
                          absl::string_view data_name,
                          absl::string_view value) {
  filter_state.setData(data_name, std::make_unique<StringAccessorImpl>(value),
                       Envoy::StreamInfo::FilterState::StateType::ReadOnly);
}

absl::string_view getStringFilterState(
    const Envoy::StreamInfo::FilterState& filter_state,
    absl::string_view data_name) {
  if (!filter_state.hasData<StringAccessor>(data_name)) {
    return Envoy::EMPTY_STRING;
  }

  return filter_state.getDataReadOnly<StringAccessor>(data_name).asString();
}

void setErrorFilterState(Envoy::StreamInfo::FilterState& filter_state,
                         const google::rpc::Status& status) {
  auto state = std::make_unique<RpcStatusWrapper>();
  state->status_ = status;
  filter_state.setData(kErrorRpcStatus, std::move(state),
                       Envoy::StreamInfo::FilterState::StateType::ReadOnly);
}

bool hasErrorFilterState(Envoy::StreamInfo::FilterState& filter_state) {
  return filter_state.hasData<RpcStatusWrapper>(kErrorRpcStatus);
}

google::rpc::Status getErrorFilterState(
    Envoy::StreamInfo::FilterState& filter_state) {
  auto state = filter_state.getDataReadOnly<RpcStatusWrapper>(kErrorRpcStatus);
  return state.status_;
}

}  // namespace utils
}  // namespace envoy
}  // namespace espv2
