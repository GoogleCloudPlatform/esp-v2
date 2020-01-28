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

#include "common/common/logger.h"
#include "common/grpc/status.h"
#include "google/protobuf/struct.pb.h"
#include "google/protobuf/timestamp.pb.h"
#include "google/protobuf/util/json_util.h"

namespace Envoy {
namespace Extensions {
namespace Utils {

// A class to use protobuf Struct to parse simple JSON
// * Use JsonStringToMessage to convert a JSON to Struct
// * Use this class to read top level string or integer value.
class JsonStruct {
 public:
  JsonStruct(const google::protobuf::Struct& pb_struct) : struct_(pb_struct) {}

  ::google::protobuf::util::Status getString(const std::string& key,
                                             std::string* value);

  ::google::protobuf::util::Status getInteger(const std::string& key,
                                              int* value);

  ::google::protobuf::util::Status getTimestamp(
      const std::string& key, ::google::protobuf::Timestamp* value);

 private:
  const ::google::protobuf::Struct& struct_;
};

}  // namespace Utils
}  // namespace Extensions
}  // namespace Envoy
