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

#include "google/protobuf/struct.pb.h"
#include "google/protobuf/timestamp.pb.h"
#include "google/protobuf/util/json_util.h"
#include "source/common/common/logger.h"
#include "source/common/grpc/status.h"

namespace espv2 {
namespace envoy {
namespace utils {

// A class to use protobuf Struct to parse simple JSON
// * Use JsonStringToMessage to convert a JSON to Struct
// * Use this class to read top level string or integer value.
class JsonStruct {
 public:
  JsonStruct(const google::protobuf::Struct& pb_struct) : struct_(pb_struct) {}

  absl::Status getString(const std::string& key, std::string* value);

  absl::Status getInteger(const std::string& key, int* value);

  absl::Status getTimestamp(const std::string& key,
                            ::google::protobuf::Timestamp* value);

 private:
  const ::google::protobuf::Struct& struct_;
};

}  // namespace utils
}  // namespace envoy
}  // namespace espv2
