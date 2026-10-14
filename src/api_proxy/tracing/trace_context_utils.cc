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

#include "src/api_proxy/tracing/trace_context_utils.h"

#include "absl/strings/escaping.h"
#include "absl/strings/match.h"
#include "absl/strings/str_cat.h"
#include "absl/strings/str_format.h"
#include "absl/strings/str_split.h"
#include "absl/strings/numbers.h"

namespace espv2 {
namespace api_proxy {
namespace tracing {

absl::optional<std::string> TraceContextUtils::XCloudTraceContextToTraceParent(
    absl::string_view x_cloud_trace_context) {
    // Format: TRACE_ID/SPAN_ID;o=TRACE_TRUE
    // Official GCP Spec: https://cloud.google.com/trace/docs/setup#force-trace
    // Official W3C Spec: https://www.w3.org/TR/trace-context/#traceparent-header
    std::vector<absl::string_view> parts = absl::StrSplit(x_cloud_trace_context, '/');
    if (parts.size() != 2) return absl::nullopt;

    absl::string_view trace_id = parts[0];
    if (trace_id.length() != 32) return absl::nullopt;

    std::vector<absl::string_view> sub_parts = absl::StrSplit(parts[1], ';');
    if (sub_parts.empty()) return absl::nullopt;

    absl::string_view span_id_str = sub_parts[0];
    uint64_t span_id;
    if (!absl::SimpleAtoi(span_id_str, &span_id)) return absl::nullopt;

    std::string span_id_hex = absl::StrFormat("%016x", span_id);

    std::string trace_flags = "00";
    if (sub_parts.size() > 1) {
        if (absl::StrContains(sub_parts[1], "o=1")) {
            trace_flags = "01";
        }
    }

    // W3C traceparent format: 00-trace_id-span_id-trace_flags
    return absl::StrCat("00-", trace_id, "-", span_id_hex, "-", trace_flags);
}

absl::optional<std::string> TraceContextUtils::TraceParentToXCloudTraceContext(
    absl::string_view traceparent) {
    // Format: 00-TRACE_ID-SPAN_ID-TRACE_FLAGS
    // Official W3C Spec: https://www.w3.org/TR/trace-context/#traceparent-header
    // Official GCP Spec: https://cloud.google.com/trace/docs/setup#force-trace
    std::vector<absl::string_view> parts = absl::StrSplit(traceparent, '-');
    if (parts.size() != 4 || parts[0] != "00") return absl::nullopt;

    absl::string_view trace_id = parts[1];
    if (trace_id.length() != 32) return absl::nullopt;

    absl::string_view span_id_hex = parts[2];
    if (span_id_hex.length() != 16) return absl::nullopt;

    uint64_t span_id;
    if (!absl::SimpleHexAtoi(span_id_hex, &span_id)) return absl::nullopt;

    std::string trace_flags = "o=0";
    uint32_t flags;
    if (absl::SimpleHexAtoi(parts[3], &flags)) {
        if ((flags & 1) != 0) {
            trace_flags = "o=1";
        }
    }

    // x-cloud-trace-context format: TRACE_ID/SPAN_ID;o=TRACE_TRUE
    return absl::StrCat(trace_id, "/", span_id, ";", trace_flags);
}

absl::optional<std::string> TraceContextUtils::GrpcTraceBinToTraceParent(
    absl::string_view grpc_trace_bin) {
    // grpc-trace-bin format: https://github.com/census-instrumentation/opencensus-specs/blob/master/encodings/BinaryEncodingId1.md
    // W3C format: https://www.w3.org/TR/trace-context/#traceparent-header
    std::string decoded;
    if (!absl::Base64Unescape(grpc_trace_bin, &decoded)) {
        return absl::nullopt;
    }

    if (decoded.length() < 29) {
        return absl::nullopt;
    }
    if (decoded[0] != 0 || decoded[1] != 0) {
        return absl::nullopt;
    }
    std::string trace_id = absl::BytesToHexString(decoded.substr(2, 16));
    
    if (decoded[18] != 1) {
        return absl::nullopt;
    }
    std::string span_id = absl::BytesToHexString(decoded.substr(19, 8));

    std::string trace_flags = "00";
    if (decoded[27] == 2) {
        uint8_t flags = decoded[28];
        if ((flags & 1) != 0) {
            trace_flags = "01";
        }
    }

    return absl::StrCat("00-", trace_id, "-", span_id, "-", trace_flags);
}

absl::optional<std::string> TraceContextUtils::TraceParentToGrpcTraceBin(
    absl::string_view traceparent) {
    // W3C format: https://www.w3.org/TR/trace-context/#traceparent-header
    // grpc-trace-bin format: https://github.com/census-instrumentation/opencensus-specs/blob/master/encodings/BinaryEncodingId1.md
    std::vector<absl::string_view> parts = absl::StrSplit(traceparent, '-');
    if (parts.size() != 4 || parts[0] != "00") return absl::nullopt;

    absl::string_view trace_id = parts[1];
    if (trace_id.length() != 32) return absl::nullopt;

    absl::string_view span_id = parts[2];
    if (span_id.length() != 16) return absl::nullopt;

    std::string trace_id_bytes = absl::HexStringToBytes(trace_id);
    std::string span_id_bytes = absl::HexStringToBytes(span_id);
    if (trace_id_bytes.empty() || span_id_bytes.empty()) return absl::nullopt;

    std::string encoded;
    encoded.push_back(0); // version
    encoded.push_back(0); // field 0
    encoded.append(trace_id_bytes);
    encoded.push_back(1); // field 1
    encoded.append(span_id_bytes);
    encoded.push_back(2); // field 2
    uint32_t flags = 0;
    if (!absl::SimpleHexAtoi(parts[3], &flags)) return absl::nullopt;
    
    if ((flags & 1) != 0) {
        encoded.push_back(1); // options: sampled
    } else {
        encoded.push_back(0); // options: not sampled
    }

    std::string base64_encoded;
    absl::Base64Escape(encoded, &base64_encoded);
    return base64_encoded;
}

} // namespace tracing
} // namespace api_proxy
} // namespace espv2
