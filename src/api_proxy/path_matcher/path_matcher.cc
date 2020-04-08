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

#include "src/api_proxy/path_matcher/path_matcher.h"

#include "absl/strings/str_split.h"

namespace espv2 {
namespace api_proxy {
namespace path_matcher {

void ExtractBindingsFromPath(const std::vector<HttpTemplate::Variable>& vars,
                             const std::vector<std::string>& parts,
                             std::vector<VariableBinding>* bindings) {
  for (const auto& var : vars) {
    // Determine the subpath bound to the variable based on the
    // [start_segment, end_segment) segment range of the variable.
    //
    // In case of matching "**" - end_segment is negative and is relative to
    // the end such that end_segment = -1 will match all subsequent segments.
    VariableBinding binding;
    binding.field_path = var.field_path;
    // Calculate the absolute index of the ending segment in case it's negative.
    size_t end_segment = (var.end_segment >= 0)
                             ? var.end_segment
                             : parts.size() + var.end_segment + 1;
    // Joins parts with "/"  to form a path string.
    for (size_t i = var.start_segment; i < end_segment; ++i) {
      binding.value += parts[i];
      if (i < end_segment - 1) {
        binding.value += "/";
      }
    }
    bindings->emplace_back(binding);
  }
}

std::vector<std::string> ExtractRequestParts(
    std::string path, const std::set<std::string>& custom_verbs) {
  // Remove query parameters.
  path = path.substr(0, path.find_first_of('?'));

  // Replace last ':' with '/' to handle custom verb.
  // But not for /foo:bar/const.
  std::size_t last_colon_pos = path.find_last_of(':');
  std::size_t last_slash_pos = path.find_last_of('/');
  if (last_colon_pos != std::string::npos && last_colon_pos > last_slash_pos) {
    std::string verb = path.substr(last_colon_pos + 1);
    // only verb in the configured custom verbs, treat it as verb
    // replace ":" with / as a separate segment.
    if (custom_verbs.find(verb) != custom_verbs.end()) {
      path[last_colon_pos] = '/';
    }
  }

  std::vector<std::string> result;
  if (!path.empty()) {
    result = absl::StrSplit(path.substr(1), '/');
  }
  // Removes all trailing empty parts caused by extra "/".
  while (!result.empty() && (*(--result.end())).empty()) {
    result.pop_back();
  }
  return result;
}

PathMatcherLookupResult LookupInPathMatcherNode(
    const PathMatcherNode& root, const std::vector<std::string>& parts,
    const HttpMethod& http_method) {
  PathMatcherLookupResult result;
  root.LookupPath(parts.begin(), parts.end(), http_method, &result);
  return result;
}

PathMatcherNode::PathInfo TransformHttpTemplate(const HttpTemplate& ht) {
  PathMatcherNode::PathInfo::Builder builder;

  for (const std::string& part : ht.segments()) {
    builder.AppendLiteralNode(part);
  }
  if (!ht.verb().empty()) {
    builder.AppendLiteralNode(ht.verb());
  }

  return builder.Build();
}

}  // namespace path_matcher
}  // namespace api_proxy
}  // namespace espv2
