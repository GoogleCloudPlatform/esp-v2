// Copyright 2019 Google Cloud Platform Proxy Authors
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
#include "src/api_proxy/auth/json_util.h"
#include <stddef.h>
#include <string.h>
#include <sstream>
#include <vector>

extern "C" {
#include "grpc/support/log.h"
}

namespace google {
namespace api_proxy {
namespace auth {
namespace {

// Delimiter of the jwt payloads
const char kJwtPayloadsDelimeter = '.';

bool isNullOrEmpty(const char *str) { return str == nullptr || *str == '\0'; }

}  // namespace

const grpc_json *GetProperty(const grpc_json *json, const char *key) {
  if (json == nullptr || key == nullptr) {
    return nullptr;
  }
  const grpc_json *cur;
  for (cur = json->child; cur != nullptr; cur = cur->next) {
    if (strcmp(cur->key, key) == 0) return cur;
  }
  return nullptr;
}

const char *GetPropertyValue(const grpc_json *json, const char *key,
                             grpc_json_type type) {
  const grpc_json *cur = GetProperty(json, key);
  if (cur != nullptr) {
    if (cur->type != type) {
      gpr_log(GPR_ERROR, "Unexpected type of a %s field [%s]: %d", key,
              cur->value, type);
      return nullptr;
    }
    return cur->value;
  }
  return nullptr;
}
void Split(const std::string &s, char delim, std::vector<std::string> *elems) {
  std::stringstream ss(s);
  std::string item;
  while (std::getline(ss, item, delim)) {
    elems->push_back(item);
  }
}

bool GetPrimitiveFieldValue(const std::string &json_str,
                            const std::string &payload_path,
                            std::string *payload_value) {
  char *json_copy = strdup(json_str.c_str());
  grpc_json *json_root =
      grpc_json_parse_string_with_len(json_copy, strlen(json_copy));
  if (!json_root) {
    gpr_free(json_copy);
    return false;
  }

  const grpc_json *json = json_root;
  std::vector<std::string> path_fields;
  Split(payload_path, kJwtPayloadsDelimeter, &path_fields);
  for (const auto &path_field : path_fields) {
    json = GetProperty(json, path_field.c_str());
  }
  if (!json) {
    grpc_json_destroy(json_root);
    gpr_free(json_copy);
    return false;
  }

  switch (json->type) {
    case GRPC_JSON_STRING:
    case GRPC_JSON_NUMBER:
      *payload_value = json->value;
      break;
    case GRPC_JSON_TRUE:
      *payload_value = "true";
      break;
    case GRPC_JSON_FALSE:
      *payload_value = "false";
      break;
    default:
      grpc_json_destroy(json_root);
      gpr_free(json_copy);
      return false;
  }
  grpc_json_destroy(json_root);
  gpr_free(json_copy);
  return true;
}

const char *GetStringValue(const grpc_json *json, const char *key) {
  return GetPropertyValue(json, key, GRPC_JSON_STRING);
}

const char *GetNumberValue(const grpc_json *json, const char *key) {
  return GetPropertyValue(json, key, GRPC_JSON_NUMBER);
}

void FillChild(grpc_json *child, grpc_json *brother, grpc_json *parent,
               const char *key, const char *value, grpc_json_type type) {
  if (isNullOrEmpty(key) || isNullOrEmpty(value)) {
    return;
  }

  memset(child, 0, sizeof(grpc_json));

  if (brother) brother->next = child;
  if (!parent->child) parent->child = child;

  child->parent = parent;
  child->key = key;
  child->value = value;
  child->type = type;
}

}  // namespace auth
}  // namespace api_proxy
}  // namespace google
