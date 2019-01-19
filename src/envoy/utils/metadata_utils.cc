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

#include "src/envoy/utils/metadata_utils.h"

namespace Envoy {
namespace Extensions {
namespace Utils {
namespace {
constexpr char kOperation[] = "operation";
}

void setOperationToMetadata(::envoy::api::v2::core::Metadata& metadata,
    const std::string& operation) {
  ProtobufWkt::Value value;
  value.set_string_value(operation);
  ProtobufWkt::Struct md;
  (*md.mutable_fields())[kOperation] = value;
  (*metadata.mutable_filter_metadata())[kPathMatcherFilterName].MergeFrom(md);
}

const std::string& getOperationFromMetadata(const ::envoy::api::v2::core::Metadata& metadata,
    const std::string& default_value) {
  const auto filter_it = metadata.filter_metadata().find(kPathMatcherFilterName);
  // Failure case for missing namespace.
  if (filter_it == metadata.filter_metadata().end()) {
    return default_value;
  }
  // Failure case for missing key.
  const auto fields_it = filter_it->second.fields().find(kOperation);
  if (fields_it == filter_it->second.fields().end()) {
    return default_value;
  }
  return fields_it->second.string_value();
}

}  // namespace Utils
}  // namespace Extensions
}  // namespace Envoy