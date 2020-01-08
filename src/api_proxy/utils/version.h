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

namespace google {
namespace api_proxy {
namespace utils {

// Provides a singleton for getting the ESPv2 version.
class Version final {
 public:
  // Gets the singleton instance.
  static Version& instance();

  // Gets the version, which will be populated from the version file by default.
  const std::string& get() const { return version_; }

  // Sets the version. Only use for tests.
  void set(const std::string& v) { version_ = v; }

 private:
  std::string version_;

  // Constructs with a default version. Private to require singleton use.
  Version(const std::string& v) { version_ = v; }
};

}  // namespace utils
}  // namespace api_proxy
}  // namespace google
