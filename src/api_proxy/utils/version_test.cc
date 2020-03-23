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
#include "src/api_proxy/utils/version.h"

#include "gtest/gtest.h"

namespace espv2 {
namespace api_proxy {
namespace utils {

TEST(VersionTest, DefaultIsNonEmpty) {
  EXPECT_FALSE(Version::instance().get().empty());
}

TEST(VerstionTest, SetVersionIsReturned) {
  Version::instance().set("test-version");

  EXPECT_EQ(Version::instance().get(), "test-version");
}

}  // namespace utils
}  // namespace api_proxy
}  // namespace espv2
