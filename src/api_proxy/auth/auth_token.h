/* Copyright 2019 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
#pragma once

#include <stddef.h>

namespace espv2 {
namespace api_proxy {
namespace auth {

// Parse a json secret and generate auth_token
// Returned pointer need to be freed by grpc_free
char* get_auth_token(const char* json_secret, const char* audience);

// Free a buffer allocated by gRPC library.
void grpc_free(char* token);

}  // namespace auth
}  // namespace api_proxy
}  // namespace espv2
