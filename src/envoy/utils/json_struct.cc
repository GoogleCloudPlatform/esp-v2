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

#include "json_struct.h"

using ::google::protobuf::Value;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::error::Code;

namespace Envoy {
namespace Extensions {
namespace Utils {

Status JsonStruct::getString(const std::string& key, std::string* value) {
  const auto& fields = struct_.fields();
  const auto it = fields.find(key);
  if (it == fields.end()) {
    return Status(::google::protobuf::util::error::NOT_FOUND,
                  "Field not found");
  }

  if (it->second.kind_case() != Value::kStringValue) {
    return Status(::google::protobuf::util::error::INVALID_ARGUMENT,
                  "Field is not a string");
  }

  *value = it->second.string_value();
  return Status::OK;
}

Status JsonStruct::getInteger(const std::string& key, int* value) {
  const auto& fields = struct_.fields();
  const auto it = fields.find(key);
  if (it == fields.end()) {
    return Status(::google::protobuf::util::error::NOT_FOUND,
                  "Field not found");
  }

  if (it->second.kind_case() != Value::kNumberValue) {
    return Status(::google::protobuf::util::error::INVALID_ARGUMENT,
                  "Field is not a number");
  }

  *value = static_cast<uint32_t>(it->second.number_value());
  return Status::OK;
}

}  // namespace Utils
}  // namespace Extensions
}  // namespace Envoy
