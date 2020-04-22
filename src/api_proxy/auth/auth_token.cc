// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
////////////////////////////////////////////////////////////////////////////////
//
#include "src/api_proxy/auth/auth_token.h"

// This header file is for internal. A public header file should not
// include it.
#include "src/api_proxy/auth/grpc_internals.h"

namespace espv2 {
namespace api_proxy {
namespace auth {

namespace {
// Token should expire in 1 hour.
const gpr_timespec TOKEN_LIFETIME = {3600, 0, GPR_TIMESPAN};
}  // namespace

absl::optional<std::string> get_auth_token(const std::string& json_secret,
                                           const std::string& audience) {
  grpc_auth_json_key json_key =
      grpc_auth_json_key_create_from_string(json_secret.c_str());

  if (grpc_auth_json_key_is_valid(&json_key) == 0) {
    return absl::nullopt;
  }

  const char* token = grpc_jwt_encode_and_sign(&json_key, audience.c_str(),
                                         TOKEN_LIFETIME, nullptr);
  grpc_auth_json_key_destruct(&json_key);

  return absl::optional<std::string>{token};
}

}  // namespace auth
}  // namespace api_proxy
}  // namespace espv2
