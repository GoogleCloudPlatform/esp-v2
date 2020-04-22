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

// This header file is for internal use only since it declares grpc
// internals that auth depends on. A public header file should not
// include any internal grpc header files.

// TODO: Remove this dependency on gRPC internal implementation details,
// or work with gRPC team to support this functionality as a public API
// surface.

#include "src/core/lib/security/credentials/jwt/json_token.h"
