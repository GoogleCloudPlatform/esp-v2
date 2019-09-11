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

#include "src/api_proxy/path_matcher/path_matcher.h"

#include "absl/strings/str_split.h"

namespace google {
namespace api_proxy {
namespace path_matcher {

namespace {

inline bool IsReservedChar(char c) {
  // Reserved characters according to RFC 6570
  switch (c) {
    case '!':
    case '#':
    case '$':
    case '&':
    case '\'':
    case '(':
    case ')':
    case '*':
    case '+':
    case ',':
    case '/':
    case ':':
    case ';':
    case '=':
    case '?':
    case '@':
    case '[':
    case ']':
      return true;
    default:
      return false;
  }
}

// Check if an ASCII character is a hex digit.  We can't use ctype's
// isxdigit() because it is affected by locale. This function is applied
// to the escaped characters in a url, not to natural-language
// strings, so locale should not be taken into account.
inline bool ascii_isxdigit(char c) {
  return ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F') ||
         ('0' <= c && c <= '9');
}

inline int hex_digit_to_int(char c) {
  /* Assume ASCII. */
  int x = static_cast<unsigned char>(c);
  if (x > '9') {
    x += 9;
  }
  return x & 0xf;
}

// This is a helper function for UrlUnescapeString. It takes a string and
// the index of where we are within that string.
//
// The function returns true if the next three characters are of the format:
// "%[0-9A-Fa-f]{2}".
//
// If the next three characters are an escaped character then this function will
// also return what character is escaped.
bool GetEscapedChar(const std::string& src, size_t i,
                    bool unescape_reserved_chars, char* out) {
  if (i + 2 < src.size() && src[i] == '%') {
    if (ascii_isxdigit(src[i + 1]) && ascii_isxdigit(src[i + 2])) {
      char c =
          (hex_digit_to_int(src[i + 1]) << 4) | hex_digit_to_int(src[i + 2]);
      if (!unescape_reserved_chars && IsReservedChar(c)) {
        return false;
      }
      *out = c;
      return true;
    }
  }
  return false;
}

// Unescapes string 'part' and returns the unescaped string. Reserved characters
// (as specified in RFC 6570) are not escaped if unescape_reserved_chars is
// false.
std::string UrlUnescapeString(const std::string& part,
                              bool unescape_reserved_chars) {
  std::string unescaped;
  // Check whether we need to escape at all.
  bool needs_unescaping = false;
  char ch = '\0';
  for (size_t i = 0; i < part.size(); ++i) {
    if (GetEscapedChar(part, i, unescape_reserved_chars, &ch)) {
      needs_unescaping = true;
      break;
    }
  }
  if (!needs_unescaping) {
    unescaped = part;
    return unescaped;
  }

  unescaped.resize(part.size());

  char* begin = &(unescaped)[0];
  char* p = begin;

  for (size_t i = 0; i < part.size();) {
    if (GetEscapedChar(part, i, unescape_reserved_chars, &ch)) {
      *p++ = ch;
      i += 3;
    } else {
      *p++ = part[i];
      i += 1;
    }
  }

  unescaped.resize(p - begin);
  return unescaped;
}

}  // namespace

void ExtractBindingsFromPath(const std::vector<HttpTemplate::Variable>& vars,
                             const std::vector<std::string>& parts,
                             std::vector<VariableBinding>* bindings,
                             bool keep_binding_escaped) {
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
    // It is multi-part match if we have more than one segment. We also make
    // sure that a single URL segment match with ** is also considered a
    // multi-part match by checking if it->second.end_segment is negative.
    bool is_multipart =
        (end_segment - var.start_segment) > 1 || var.end_segment < 0;
    // Joins parts with "/"  to form a path string.
    for (size_t i = var.start_segment; i < end_segment; ++i) {
      // For multipart matches only unescape non-reserved characters.
      if (keep_binding_escaped) {
        binding.value += parts[i];
      } else {
        binding.value += UrlUnescapeString(parts[i], !is_multipart);
      }
      if (i < end_segment - 1) {
        binding.value += "/";
      }
    }
    bindings->emplace_back(binding);
  }
}

void ExtractBindingsFromQueryParameters(
    const std::string& query_params, const std::set<std::string>& system_params,
    std::vector<VariableBinding>* bindings, bool keep_binding_escaped) {
  // The bindings in URL the query parameters have the following form:
  //      <field_path1>=value1&<field_path2>=value2&...&<field_pathN>=valueN
  // Query parameters may also contain system parameters such as `api_key`.
  // We'll need to ignore these. Example:
  //      book.id=123&book.author=Neal%20Stephenson&api_key=AIzaSyAz7fhBkC35D2M
  std::vector<std::string> params = absl::StrSplit(query_params, '&');
  for (const auto& param : params) {
    size_t pos = param.find('=');
    if (pos != 0 && pos != std::string::npos) {
      auto name = param.substr(0, pos);
      // Make sure the query parameter is not a system parameter (e.g.
      // `api_key`) before adding the binding.
      if (system_params.find(name) == std::end(system_params)) {
        // The name of the parameter is a field path, which is a dot-delimited
        // sequence of field names that identify the (potentially deep) field
        // in the request, e.g. `book.author.name`.
        VariableBinding binding;
        binding.field_path = absl::StrSplit(name, '.');
        if (keep_binding_escaped) {
          binding.value = param.substr(pos + 1);
        } else {
          binding.value = UrlUnescapeString(param.substr(pos + 1), true);
        }
        bindings->emplace_back(std::move(binding));
      }
    }
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
}  // namespace google
