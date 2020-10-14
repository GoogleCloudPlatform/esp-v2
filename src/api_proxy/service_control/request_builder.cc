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
#include "src/api_proxy/service_control/request_builder.h"

#include <time.h>

#include <chrono>
#include <functional>

#include "absl/strings/str_cat.h"
#include "common/common/assert.h"
#include "common/common/base64.h"
#include "google/api/metric.pb.h"
#include "google/protobuf/timestamp.pb.h"
#include "google/protobuf/util/time_util.h"
#include "src/api_proxy/service_control/request_info.h"
#include "src/api_proxy/utils/version.h"
#include "utils/distribution_helper.h"

using ::google::api::servicecontrol::v1::CheckError;
using ::google::api::servicecontrol::v1::CheckRequest;
using ::google::api::servicecontrol::v1::CheckResponse;
using ::google::api::servicecontrol::v1::
    CheckResponse_ConsumerInfo_ConsumerType;
using ::google::api::servicecontrol::v1::Distribution;
using ::google::api::servicecontrol::v1::LogEntry;
using ::google::api::servicecontrol::v1::MetricValue;
using ::google::api::servicecontrol::v1::MetricValueSet;
using ::google::api::servicecontrol::v1::Operation;
using ::google::api::servicecontrol::v1::QuotaError;
using ::google::api::servicecontrol::v1::ReportRequest;
using ::google::protobuf::Map;
using ::google::protobuf::Timestamp;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::error::Code;
using ::google::service_control_client::DistributionHelper;

namespace espv2 {
namespace api_proxy {
namespace service_control {

// Default location. "global" should be used if the location is unknown.
// It is used for the metric label "cloud.googleapis.com/location".
const char kDefaultLocation[] = "global";

struct SupportedMetric {
  const char* name;
  ::google::api::MetricDescriptor_MetricKind metric_kind;
  ::google::api::MetricDescriptor_ValueType value_type;

  enum Mark { PRODUCER = 0, CONSUMER = 1, PRODUCER_BY_CONSUMER = 2 };
  enum Tag { START = 0, INTERMEDIATE = 1, FINAL = 2 };
  Tag tag;
  Mark mark;
  Status (*set)(const SupportedMetric& m, const ReportRequestInfo& info,
                Operation* operation);
};

struct SupportedLabel {
  const char* name;
  ::google::api::LabelDescriptor_ValueType value_type;

  enum Kind { USER = 0, SYSTEM = 1 };
  Kind kind;

  Status (*set)(const SupportedLabel& l, const ReportRequestInfo& info,
                Map<std::string, std::string>* labels);

  bool by_consumer_only;
};

namespace {

// Metric Helpers

MetricValue* AddMetricValue(const char* metric_name, Operation* operation) {
  MetricValueSet* metric_value_set = operation->add_metric_value_sets();
  metric_value_set->set_metric_name(metric_name);
  return metric_value_set->add_metric_values();
}

void AddInt64Metric(const char* metric_name, int64_t value,
                    Operation* operation) {
  MetricValue* metric_value = AddMetricValue(metric_name, operation);
  metric_value->set_int64_value(value);
}

// The parameters to initialize DistributionHelper
struct DistributionHelperOptions {
  int buckets;
  double growth;
  double scale;
};

const DistributionHelperOptions time_distribution = {29, 2.0, 1e-6};
const DistributionHelperOptions size_distribution = {8, 10.0, 1};
const double kMsToSecs = 1e-3;

Status AddDistributionMetric(const DistributionHelperOptions& options,
                             const char* metric_name, double value,
                             Operation* operation) {
  MetricValue* metric_value = AddMetricValue(metric_name, operation);
  Distribution distribution;
  Status status = DistributionHelper::InitExponential(
      options.buckets, options.growth, options.scale, &distribution);
  if (!status.ok()) return status;
  status = DistributionHelper::AddSample(value, &distribution);
  if (!status.ok()) return status;
  *metric_value->mutable_distribution_value() = distribution;
  return Status::OK;
}

// Metrics supported by ESPv2.

Status set_int64_metric_to_constant_1(const SupportedMetric& m,
                                      const ReportRequestInfo&,
                                      Operation* operation) {
  AddInt64Metric(m.name, 1l, operation);
  return Status::OK;
}

Status set_distribution_metric_to_request_size(const SupportedMetric& m,
                                               const ReportRequestInfo& info,
                                               Operation* operation) {
  if (info.request_size >= 0) {
    return AddDistributionMetric(size_distribution, m.name, info.request_size,
                                 operation);
  }
  return Status::OK;
}

Status set_distribution_metric_to_response_size(const SupportedMetric& m,
                                                const ReportRequestInfo& info,
                                                Operation* operation) {
  if (info.response_size >= 0) {
    return AddDistributionMetric(size_distribution, m.name, info.response_size,
                                 operation);
  }
  return Status::OK;
}

// TODO: Consider refactoring following 3 functions to avoid duplicate code
Status set_distribution_metric_to_request_time(const SupportedMetric& m,
                                               const ReportRequestInfo& info,
                                               Operation* operation) {
  if (info.latency.request_time_ms >= 0) {
    double request_time_secs = info.latency.request_time_ms * kMsToSecs;
    return AddDistributionMetric(time_distribution, m.name, request_time_secs,
                                 operation);
  }
  return Status::OK;
}

Status set_distribution_metric_to_backend_time(const SupportedMetric& m,
                                               const ReportRequestInfo& info,
                                               Operation* operation) {
  if (info.latency.backend_time_ms >= 0) {
    double backend_time_secs = info.latency.backend_time_ms * kMsToSecs;
    return AddDistributionMetric(time_distribution, m.name, backend_time_secs,
                                 operation);
  }
  return Status::OK;
}

Status set_distribution_metric_to_overhead_time(const SupportedMetric& m,
                                                const ReportRequestInfo& info,
                                                Operation* operation) {
  if (info.latency.overhead_time_ms >= 0) {
    double overhead_time_secs = info.latency.overhead_time_ms * kMsToSecs;
    return AddDistributionMetric(time_distribution, m.name, overhead_time_secs,
                                 operation);
  }
  return Status::OK;
}

// Currently unsupported metrics:
//
//  "serviceruntime.googleapis.com/api/producer/by_consumer/quota_used_count"
//
const SupportedMetric supported_metrics[] = {
    {
        "serviceruntime.googleapis.com/api/consumer/request_count",
        ::google::api::MetricDescriptor_MetricKind_DELTA,
        ::google::api::MetricDescriptor_ValueType_INT64,
        SupportedMetric::START,
        SupportedMetric::CONSUMER,
        set_int64_metric_to_constant_1,
    },
    {
        "serviceruntime.googleapis.com/api/producer/request_count",
        ::google::api::MetricDescriptor_MetricKind_DELTA,
        ::google::api::MetricDescriptor_ValueType_INT64,
        SupportedMetric::START,
        SupportedMetric::PRODUCER,
        set_int64_metric_to_constant_1,
    },
    {
        "serviceruntime.googleapis.com/api/producer/by_consumer/request_count",
        ::google::api::MetricDescriptor_MetricKind_DELTA,
        ::google::api::MetricDescriptor_ValueType_INT64,
        SupportedMetric::FINAL,
        SupportedMetric::PRODUCER_BY_CONSUMER,
        set_int64_metric_to_constant_1,
    },
    {
        "serviceruntime.googleapis.com/api/consumer/request_sizes",
        ::google::api::MetricDescriptor_MetricKind_DELTA,
        ::google::api::MetricDescriptor_ValueType_DISTRIBUTION,
        SupportedMetric::FINAL,
        SupportedMetric::CONSUMER,
        set_distribution_metric_to_request_size,
    },
    {
        "serviceruntime.googleapis.com/api/producer/request_sizes",
        ::google::api::MetricDescriptor_MetricKind_DELTA,
        ::google::api::MetricDescriptor_ValueType_DISTRIBUTION,
        SupportedMetric::FINAL,
        SupportedMetric::PRODUCER,
        set_distribution_metric_to_request_size,
    },
    {
        "serviceruntime.googleapis.com/api/producer/by_consumer/request_sizes",
        ::google::api::MetricDescriptor_MetricKind_DELTA,
        ::google::api::MetricDescriptor_ValueType_DISTRIBUTION,
        SupportedMetric::FINAL,
        SupportedMetric::PRODUCER_BY_CONSUMER,
        set_distribution_metric_to_request_size,
    },
    {
        "serviceruntime.googleapis.com/api/consumer/response_sizes",
        ::google::api::MetricDescriptor_MetricKind_DELTA,
        ::google::api::MetricDescriptor_ValueType_DISTRIBUTION,
        SupportedMetric::FINAL,
        SupportedMetric::CONSUMER,
        set_distribution_metric_to_response_size,
    },
    {
        "serviceruntime.googleapis.com/api/producer/response_sizes",
        ::google::api::MetricDescriptor_MetricKind_DELTA,
        ::google::api::MetricDescriptor_ValueType_DISTRIBUTION,
        SupportedMetric::FINAL,
        SupportedMetric::PRODUCER,
        set_distribution_metric_to_response_size,
    },
    {
        "serviceruntime.googleapis.com/api/producer/by_consumer/response_sizes",
        ::google::api::MetricDescriptor_MetricKind_DELTA,
        ::google::api::MetricDescriptor_ValueType_DISTRIBUTION,
        SupportedMetric::FINAL,
        SupportedMetric::PRODUCER_BY_CONSUMER,
        set_distribution_metric_to_response_size,
    },
    {
        "serviceruntime.googleapis.com/api/consumer/total_latencies",
        ::google::api::MetricDescriptor_MetricKind_DELTA,
        ::google::api::MetricDescriptor_ValueType_DISTRIBUTION,
        SupportedMetric::FINAL,
        SupportedMetric::CONSUMER,
        set_distribution_metric_to_request_time,
    },
    {
        "serviceruntime.googleapis.com/api/producer/total_latencies",
        ::google::api::MetricDescriptor_MetricKind_DELTA,
        ::google::api::MetricDescriptor_ValueType_DISTRIBUTION,
        SupportedMetric::FINAL,
        SupportedMetric::PRODUCER,
        set_distribution_metric_to_request_time,
    },
    {
        "serviceruntime.googleapis.com/api/producer/by_consumer/"
        "total_latencies",
        ::google::api::MetricDescriptor_MetricKind_DELTA,
        ::google::api::MetricDescriptor_ValueType_DISTRIBUTION,
        SupportedMetric::FINAL,
        SupportedMetric::PRODUCER_BY_CONSUMER,
        set_distribution_metric_to_request_time,
    },
    {
        "serviceruntime.googleapis.com/api/producer/backend_latencies",
        ::google::api::MetricDescriptor_MetricKind_DELTA,
        ::google::api::MetricDescriptor_ValueType_DISTRIBUTION,
        SupportedMetric::FINAL,
        SupportedMetric::PRODUCER,
        set_distribution_metric_to_backend_time,
    },
    {
        "serviceruntime.googleapis.com/api/producer/by_consumer/"
        "backend_latencies",
        ::google::api::MetricDescriptor_MetricKind_DELTA,
        ::google::api::MetricDescriptor_ValueType_DISTRIBUTION,
        SupportedMetric::FINAL,
        SupportedMetric::PRODUCER_BY_CONSUMER,
        set_distribution_metric_to_backend_time,
    },
    {
        "serviceruntime.googleapis.com/api/producer/request_overhead_latencies",
        ::google::api::MetricDescriptor_MetricKind_DELTA,
        ::google::api::MetricDescriptor_ValueType_DISTRIBUTION,
        SupportedMetric::FINAL,
        SupportedMetric::PRODUCER,
        set_distribution_metric_to_overhead_time,
    },
    {
        "serviceruntime.googleapis.com/api/producer/by_consumer/"
        "request_overhead_latencies",
        ::google::api::MetricDescriptor_MetricKind_DELTA,
        ::google::api::MetricDescriptor_ValueType_DISTRIBUTION,
        SupportedMetric::FINAL,
        SupportedMetric::PRODUCER_BY_CONSUMER,
        set_distribution_metric_to_overhead_time,
    },
};

const int supported_metrics_count =
    sizeof(supported_metrics) / sizeof(supported_metrics[0]);

constexpr char kServiceControlCallerIp[] =
    "servicecontrol.googleapis.com/caller_ip";
constexpr char kServiceControlReferer[] =
    "servicecontrol.googleapis.com/referer";
constexpr char kServiceControlServiceAgent[] =
    "servicecontrol.googleapis.com/service_agent";
constexpr char kServiceControlUserAgent[] =
    "servicecontrol.googleapis.com/user_agent";
constexpr char kServiceControlPlatform[] =
    "servicecontrol.googleapis.com/platform";
constexpr char kServiceControlAndroidPackageName[] =
    "servicecontrol.googleapis.com/android_package_name";
constexpr char kServiceControlAndroidCertFingerprint[] =
    "servicecontrol.googleapis.com/android_cert_fingerprint";
constexpr char kServiceControlIosBundleId[] =
    "servicecontrol.googleapis.com/ios_bundle_id";
constexpr char kServiceControlBackendProtocol[] =
    "servicecontrol.googleapis.com/backend_protocol";
constexpr char kServiceControlConsumerProject[] =
    "serviceruntime.googleapis.com/consumer_project";

// User agent label value
// The value for kUserAgent should be configured at service control server.
// Now it is configured as "ESPv2".
constexpr char kUserAgent[] = "ESPv2";

// Service agent label value
constexpr char kServiceAgentPrefix[] = "ESPv2/";

const std::string get_service_agent() {
  return kServiceAgentPrefix + utils::Version::instance().get();
}

// /credential_id
Status set_credential_id(const SupportedLabel& l, const ReportRequestInfo& info,
                         Map<std::string, std::string>* labels) {
  // The rule to set /credential_id is:
  // 1) If api_key is available and valid, set it as apiKey:API-KEY
  // 2) If auth issuer and audience both are available, set it as:
  //    jwtAuth:issuer=base64(issuer)&audience=base64(audience)
  if (info.check_response_info.api_key_state ==
      api_key::ApiKeyState::VERIFIED) {
    ASSERT(!info.api_key.empty(),
           "API Key must be set, otherwise consumer would not be verified.");
    std::string credential_id("apikey:");
    credential_id += info.api_key;
    (*labels)[l.name] = credential_id;
  } else if (!info.auth_issuer.empty()) {
    std::string base64_issuer = Envoy::Base64Url::encode(
        info.auth_issuer.data(), info.auth_issuer.size());
    std::string credential_id = absl::StrCat("jwtauth:issuer=", base64_issuer);
    // auth audience is optional
    if (!info.auth_audience.empty()) {
      std::string base64_audience = Envoy::Base64Url::encode(
          info.auth_audience.data(), info.auth_audience.size());
      absl::StrAppend(&credential_id, "&audience=", base64_audience);
    }
    (*labels)[l.name] = credential_id;
  }
  return Status::OK;
}

constexpr const char* error_types[10] = {"0xx", "1xx", "2xx", "3xx", "4xx",
                                         "5xx", "6xx", "7xx", "8xx", "9xx"};

// /error_type
Status set_error_type(const SupportedLabel& l, const ReportRequestInfo& info,
                      Map<std::string, std::string>* labels) {
  if (info.response_code >= 400) {
    int code = (info.response_code / 100) % 10;
    if (error_types[code]) {
      (*labels)[l.name] = error_types[code];
    }
  }
  return Status::OK;
}

// /protocol
Status set_protocol(const SupportedLabel& l, const ReportRequestInfo& info,
                    Map<std::string, std::string>* labels) {
  (*labels)[l.name] = protocol::ToString(info.frontend_protocol);
  return Status::OK;
}

// /servicecontrol.googleapis.com/backend_protocol
Status set_backend_protocol(const SupportedLabel& l,
                            const ReportRequestInfo& info,
                            Map<std::string, std::string>* labels) {
  // backend_protocol is either GRPC or UNKNOWN.
  if (info.backend_protocol == protocol::GRPC &&
      info.frontend_protocol != info.backend_protocol) {
    (*labels)[l.name] = protocol::ToString(info.backend_protocol);
  }
  return Status::OK;
}

// /servicecontrol.googleapis.com/consumer_project
Status set_consumer_project(const SupportedLabel& l,
                            const ReportRequestInfo& info,
                            Map<std::string, std::string>* labels) {
  (*labels)[l.name] = info.check_response_info.consumer_project_number;
  return Status::OK;
}

// /referer
Status set_referer(const SupportedLabel& l, const ReportRequestInfo& info,
                   Map<std::string, std::string>* labels) {
  if (!info.referer.empty()) {
    (*labels)[l.name] = info.referer;
  }
  return Status::OK;
}

// /response_code
Status set_response_code(const SupportedLabel& l, const ReportRequestInfo& info,
                         Map<std::string, std::string>* labels) {
  char response_code_buf[20];
  snprintf(response_code_buf, sizeof(response_code_buf), "%d",
           info.response_code);
  (*labels)[l.name] = response_code_buf;
  return Status::OK;
}

// /response_code_class
Status set_response_code_class(const SupportedLabel& l,
                               const ReportRequestInfo& info,
                               Map<std::string, std::string>* labels) {
  (*labels)[l.name] = error_types[(info.response_code / 100) % 10];
  return Status::OK;
}

// /status_code
Status set_status_code(const SupportedLabel& l, const ReportRequestInfo& info,
                       Map<std::string, std::string>* labels) {
  char status_code_buf[20];
  snprintf(status_code_buf, sizeof(status_code_buf), "%d",
           info.status.error_code());
  (*labels)[l.name] = status_code_buf;
  return Status::OK;
}

// cloud.googleapis.com/location
Status set_location(const SupportedLabel& l, const ReportRequestInfo& info,
                    Map<std::string, std::string>* labels) {
  if (!info.location.empty()) {
    (*labels)[l.name] = info.location;
  } else {
    // This label SHOULD not be empty, otherwise the server will fail the call.
    (*labels)[l.name] = kDefaultLocation;
  }
  return Status::OK;
}

// serviceruntime.googleapis.com/api_method
Status set_api_method(const SupportedLabel& l, const ReportRequestInfo& info,
                      Map<std::string, std::string>* labels) {
  if (!info.api_method.empty()) {
    (*labels)[l.name] = info.api_method;
  }
  return Status::OK;
}

// serviceruntime.googleapis.com/api_version
Status set_api_version(const SupportedLabel& l, const ReportRequestInfo& info,
                       Map<std::string, std::string>* labels) {
  if (!info.api_version.empty()) {
    (*labels)[l.name] = info.api_version;
  }
  return Status::OK;
}

// servicecontrol.googleapis.com/platform
Status set_platform(const SupportedLabel& l, const ReportRequestInfo& info,
                    Map<std::string, std::string>* labels) {
  (*labels)[l.name] = info.compute_platform;
  return Status::OK;
}

// servicecontrol.googleapis.com/service_agent
Status set_service_agent(const SupportedLabel& l, const ReportRequestInfo&,
                         Map<std::string, std::string>* labels) {
  (*labels)[l.name] = get_service_agent();
  return Status::OK;
}

// serviceruntime.googleapis.com/user_agent
Status set_user_agent(const SupportedLabel& l, const ReportRequestInfo&,
                      Map<std::string, std::string>* labels) {
  (*labels)[l.name] = kUserAgent;
  return Status::OK;
}

const SupportedLabel supported_labels[] = {
    {
        "/credential_id",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::USER,
        set_credential_id,
        false,
    },
    {
        "/end_user",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::USER,
        nullptr,
        false,
    },
    {
        "/end_user_country",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::USER,
        nullptr,
        false,
    },
    {
        "/error_type",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::USER,
        set_error_type,
        false,
    },
    {
        "/protocol",
        ::google::api::LabelDescriptor::STRING,
        SupportedLabel::USER,
        set_protocol,
        false,
    },
    {
        "/referer",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::USER,
        set_referer,
        false,
    },
    {
        "/response_code",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::USER,
        set_response_code,
        false,
    },
    {
        "/response_code_class",
        ::google::api::LabelDescriptor::STRING,
        SupportedLabel::USER,
        set_response_code_class,
        false,
    },
    {
        "/status_code",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::USER,
        set_status_code,
        false,
    },
    {
        "appengine.googleapis.com/clone_id",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::USER,
        nullptr,
        false,
    },
    {
        "appengine.googleapis.com/module_id",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::USER,
        nullptr,
        false,
    },
    {
        "appengine.googleapis.com/replica_index",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::USER,
        nullptr,
        false,
    },
    {
        "appengine.googleapis.com/version_id",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::USER,
        nullptr,
        false,
    },
    {
        "cloud.googleapis.com/location",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::SYSTEM,
        set_location,
        false,
    },
    {
        "cloud.googleapis.com/project",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::SYSTEM,
        nullptr,
        false,
    },
    {
        "cloud.googleapis.com/region",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::SYSTEM,
        nullptr,
        false,
    },
    {
        "cloud.googleapis.com/resource_id",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::USER,
        nullptr,
        false,
    },
    {
        "cloud.googleapis.com/resource_type",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::USER,
        nullptr,
        false,
    },
    {
        "cloud.googleapis.com/service",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::SYSTEM,
        nullptr,
        false,
    },
    {
        "cloud.googleapis.com/zone",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::SYSTEM,
        nullptr,
        false,
    },
    {
        "cloud.googleapis.com/uid",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::SYSTEM,
        nullptr,
        false,
    },
    {
        "serviceruntime.googleapis.com/api_method",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::USER,
        set_api_method,
        false,
    },
    {
        "serviceruntime.googleapis.com/api_version",
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::USER,
        set_api_version,
        false,
    },
    {
        kServiceControlCallerIp,
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::SYSTEM,
        nullptr,
        false,
    },
    {
        kServiceControlReferer,
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::SYSTEM,
        nullptr,
        false,
    },
    {
        kServiceControlServiceAgent,
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::SYSTEM,
        set_service_agent,
        false,
    },
    {
        kServiceControlUserAgent,
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::SYSTEM,
        set_user_agent,
        false,
    },
    {
        kServiceControlPlatform,
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::SYSTEM,
        set_platform,
        false,
    },
    {
        kServiceControlBackendProtocol,
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::SYSTEM,
        set_backend_protocol,
        false,
    },
    {
        kServiceControlConsumerProject,
        ::google::api::LabelDescriptor_ValueType_STRING,
        SupportedLabel::SYSTEM,
        set_consumer_project,
        true,
    },
};

const int supported_labels_count =
    sizeof(supported_labels) / sizeof(supported_labels[0]);

// Supported intrinsic labels:
// "servicecontrol.googleapis.com/operation_name": Operation.operation_name
// "servicecontrol.googleapis.com/consumer_id": Operation.consumer_id

// Unsupported service control labels:
// "servicecontrol.googleapis.com/android_package_name"
// "servicecontrol.googleapis.com/android_cert_fingerprint"
// "servicecontrol.googleapis.com/ios_bundle_id"
// "servicecontrol.googleapis.com/credential_project_number"

// Define Service Control constant strings
constexpr char kConsumerIdApiKey[] = "api_key:";
constexpr char kConsumerIdProject[] = "project:";

// Following names for Log struct_playload field names:
constexpr char kLogFieldNameApiKey[] = "api_key";
constexpr char kLogFieldNameApiMethod[] = "api_method";
constexpr char kLogFieldNameApiName[] = "api_name";
constexpr char kLogFieldNameApiVersion[] = "api_version";
constexpr char kLogFieldNameErrorCause[] = "error_cause";
constexpr char kLogFieldNameJwtPayloads[] = "jwt_payloads";
constexpr char kLogFieldNameLocation[] = "location";
constexpr char kLogFieldNameLogMessage[] = "log_message";
constexpr char kLogFieldNameProducerProjectId[] = "producer_project_id";
constexpr char kLogFieldNameRequestHeaders[] = "request_headers";
constexpr char kLogFieldNameResponseHeaders[] = "response_headers";
constexpr char kLogFieldNameServiceAgent[] = "service_agent";
constexpr char kLogFieldNameConfigId[] = "service_config_id";
constexpr char kLogFieldNameTimestamp[] = "timestamp";
constexpr char kLogFieldNameApiKeyState[] = "api_key_state";
constexpr char kLogFieldNameResponseCodeDetail[] = "response_code_detail";

// Convert time point to proto Timestamp
Timestamp CreateTimestamp(std::chrono::system_clock::time_point tp) {
  long long timestamp_ns = std::chrono::duration_cast<std::chrono::nanoseconds>(
                               tp.time_since_epoch())
                               .count();
  Timestamp timestamp;
  timestamp.set_seconds(timestamp_ns / 1000000000);
  timestamp.set_nanos(timestamp_ns % 1000000000);
  return timestamp;
}

Status VerifyRequiredCheckFields(const OperationInfo& info) {
  if (info.operation_id.empty()) {
    return Status(Code::INVALID_ARGUMENT, "operation_id is required.");
  }
  if (info.operation_name.empty()) {
    return Status(Code::INVALID_ARGUMENT, "operation_name is required.");
  }
  return Status::OK;
}

Status VerifyRequiredReportFields(const OperationInfo&) { return Status::OK; }

void SetOperationCommonFields(const OperationInfo& info,
                              const Timestamp& current_time, Operation* op) {
  if (!info.operation_id.empty()) {
    op->set_operation_id(info.operation_id);
  }
  if (!info.operation_name.empty()) {
    op->set_operation_name(info.operation_name);
  }
  *op->mutable_start_time() = current_time;
  *op->mutable_end_time() = current_time;
}

void FillLogEntry(const ReportRequestInfo& info, const std::string& name,
                  const std::string& config_id, const Timestamp& current_time,
                  LogEntry* log_entry) {
  log_entry->set_name(name);
  *log_entry->mutable_timestamp() = current_time;
  auto severity = (info.response_code >= 400) ? google::logging::type::ERROR
                                              : google::logging::type::INFO;
  log_entry->set_severity(severity);

  // Fill in http request.
  auto* http_request = log_entry->mutable_http_request();
  http_request->set_status(info.response_code);
  http_request->set_protocol(protocol::ToString(info.frontend_protocol));
  if (!info.method.empty()) {
    http_request->set_request_method(info.method);
  }
  if (!info.url.empty()) {
    http_request->set_request_url(info.url);
  }
  if (info.request_size >= 0) {
    http_request->set_request_size(info.request_size);
  }
  if (info.response_size >= 0) {
    http_request->set_response_size(info.response_size);
  }
  if (!info.client_ip.empty()) {
    http_request->set_remote_ip(info.client_ip);
  }
  if (!info.referer.empty()) {
    http_request->set_referer(info.referer);
  }
  if (info.latency.request_time_ms >= 0) {
    const google::protobuf::Duration duration =
        google::protobuf::util::TimeUtil::MillisecondsToDuration(
            info.latency.request_time_ms);
    http_request->mutable_latency()->CopyFrom(duration);
  }

  // Fill in JSON struct.
  // TODO(nareddyt): For backwards compatibility, some of the information from
  // the `http_request` fields is duplicated. Decide if we should remove.
  auto* fields = log_entry->mutable_struct_payload()->mutable_fields();
  (*fields)[kLogFieldNameTimestamp].set_number_value(
      static_cast<double>(current_time.seconds()) +
      static_cast<double>(current_time.nanos()) / 1000000000.0);
  (*fields)[kLogFieldNameConfigId].set_string_value(config_id);
  (*fields)[kLogFieldNameServiceAgent].set_string_value(
      kServiceAgentPrefix + utils::Version::instance().get());

  (*fields)[kLogFieldNameApiKeyState].set_string_value(
      api_key::ToString(info.check_response_info.api_key_state));

  if (!info.response_code_detail.empty()) {
    (*fields)[kLogFieldNameResponseCodeDetail].set_string_value(
        info.response_code_detail);
  }

  if (!info.producer_project_id.empty()) {
    (*fields)[kLogFieldNameProducerProjectId].set_string_value(
        info.producer_project_id);
  }
  if (!info.api_key.empty()) {
    (*fields)[kLogFieldNameApiKey].set_string_value(info.api_key);
  }
  if (!info.api_name.empty()) {
    (*fields)[kLogFieldNameApiName].set_string_value(info.api_name);
  }
  if (!info.api_version.empty()) {
    (*fields)[kLogFieldNameApiVersion].set_string_value(info.api_version);
  }
  if (!info.api_method.empty()) {
    (*fields)[kLogFieldNameApiMethod].set_string_value(info.api_method);
  }
  if (!info.location.empty()) {
    (*fields)[kLogFieldNameLocation].set_string_value(info.location);
  }
  if (!info.log_message.empty()) {
    (*fields)[kLogFieldNameLogMessage].set_string_value(info.log_message);
  }
  if (!info.request_headers.empty()) {
    (*fields)[kLogFieldNameRequestHeaders].set_string_value(
        info.request_headers);
  }
  if (!info.response_headers.empty()) {
    (*fields)[kLogFieldNameResponseHeaders].set_string_value(
        info.response_headers);
  }
  if (!info.jwt_payloads.empty()) {
    (*fields)[kLogFieldNameJwtPayloads].set_string_value(info.jwt_payloads);
  }
  if (info.response_code >= 400 && info.status.error_message().length() > 0) {
    (*fields)[kLogFieldNameErrorCause].set_string_value(
        info.status.error_message().as_string());
  }
}

template <class Element>
std::vector<const Element*> FilterPointers(
    const Element* first, const Element* last,
    std::function<bool(const Element*)> pred) {
  std::vector<const Element*> filtered;
  while (first < last) {
    if (pred(first)) {
      filtered.push_back(first);
    }
    first++;
  }
  return filtered;
}

}  // namespace

RequestBuilder::RequestBuilder(const std::set<std::string>& logs,
                               const std::string& service_name,
                               const std::string& service_config_id)
    : logs_(logs.begin(), logs.end()),
      metrics_(FilterPointers<SupportedMetric>(
          supported_metrics, supported_metrics + supported_metrics_count,
          [](const struct SupportedMetric* m) { return m->set != nullptr; })),
      labels_(FilterPointers<SupportedLabel>(
          supported_labels, supported_labels + supported_labels_count,
          [](const struct SupportedLabel* l) { return l->set != nullptr; })),
      service_name_(service_name),
      service_config_id_(service_config_id) {}

RequestBuilder::RequestBuilder(const std::set<std::string>& logs,
                               const std::set<std::string>& metrics,
                               const std::set<std::string>& labels,
                               const std::string& service_name,
                               const std::string& service_config_id)
    : logs_(logs.begin(), logs.end()),
      metrics_(FilterPointers<SupportedMetric>(
          supported_metrics, supported_metrics + supported_metrics_count,
          [&metrics](const struct SupportedMetric* m) {
            return m->set && metrics.find(m->name) != metrics.end();
          })),
      labels_(FilterPointers<SupportedLabel>(
          supported_labels, supported_labels + supported_labels_count,
          [&labels](const struct SupportedLabel* l) {
            return l->set && (l->kind == SupportedLabel::SYSTEM ||
                              labels.find(l->name) != labels.end());
          })),
      service_name_(service_name),
      service_config_id_(service_config_id) {}

Status RequestBuilder::FillAllocateQuotaRequest(
    const QuotaRequestInfo& info,
    ::google::api::servicecontrol::v1::AllocateQuotaRequest* request) const {
  ::google::api::servicecontrol::v1::QuotaOperation* operation =
      request->mutable_allocate_operation();

  // service_name
  request->set_service_name(service_name_);
  // service_config_id
  request->set_service_config_id(service_config_id_);

  // allocate_operation.operation_id
  if (!info.operation_id.empty()) {
    operation->set_operation_id(info.operation_id);
  }
  // allocate_operation.method_name
  if (!info.method_name.empty()) {
    operation->set_method_name(info.method_name);
  }
  // allocate_operation.consumer_id
  if (!info.api_key.empty()) {
    // For quota request, we send the API key as is.
    operation->set_consumer_id(std::string(kConsumerIdApiKey) +
                               std::string(info.api_key));
  } else if (!info.producer_project_id.empty()) {
    operation->set_consumer_id(std::string(kConsumerIdProject) +
                               std::string(info.producer_project_id));
  }

  // allocate_operation.quota_mode
  operation->set_quota_mode(
      ::google::api::servicecontrol::v1::QuotaOperation_QuotaMode::
          QuotaOperation_QuotaMode_BEST_EFFORT);

  // allocate_operation.labels
  auto* labels = operation->mutable_labels();
  if (!info.client_ip.empty()) {
    (*labels)[kServiceControlCallerIp] = info.client_ip;
  }

  if (!info.referer.empty()) {
    (*labels)[kServiceControlReferer] = info.referer;
  }
  (*labels)[kServiceControlUserAgent] = kUserAgent;
  (*labels)[kServiceControlServiceAgent] = get_service_agent();

  for (auto metric : info.metric_cost_vector) {
    MetricValueSet* value_set = operation->add_quota_metrics();
    value_set->set_metric_name(metric.first);
    MetricValue* value = value_set->add_metric_values();
    const auto& cost = metric.second;
    value->set_int64_value(cost <= 0 ? 1 : cost);
  }

  return Status::OK;
}

Status RequestBuilder::FillCheckRequest(const CheckRequestInfo& info,
                                        CheckRequest* request) const {
  Status status = VerifyRequiredCheckFields(info);
  if (!status.ok()) {
    return status;
  }
  request->set_service_name(service_name_);
  request->set_service_config_id(service_config_id_);

  Timestamp current_time = CreateTimestamp(info.current_time);
  Operation* op = request->mutable_operation();
  SetOperationCommonFields(info, current_time, op);
  if (!info.api_key.empty()) {
    // For check request, we send the API key as is.
    op->set_consumer_id(std::string(kConsumerIdApiKey) +
                        std::string(info.api_key));
  }

  auto* labels = op->mutable_labels();
  if (!info.client_ip.empty()) {
    (*labels)[kServiceControlCallerIp] = info.client_ip;
  }
  if (!info.referer.empty()) {
    (*labels)[kServiceControlReferer] = info.referer;
  }
  (*labels)[kServiceControlUserAgent] = kUserAgent;
  (*labels)[kServiceControlServiceAgent] = get_service_agent();

  if (!info.android_package_name.empty()) {
    (*labels)[kServiceControlAndroidPackageName] = info.android_package_name;
  }
  if (!info.android_cert_fingerprint.empty()) {
    (*labels)[kServiceControlAndroidCertFingerprint] =
        info.android_cert_fingerprint;
  }
  if (!info.ios_bundle_id.empty()) {
    (*labels)[kServiceControlIosBundleId] = info.ios_bundle_id;
  }

  return Status::OK;
}

Status RequestBuilder::FillReportRequest(const ReportRequestInfo& info,
                                         ReportRequest* request) const {
  Status status = VerifyRequiredReportFields(info);
  if (!status.ok()) {
    return status;
  }
  request->set_service_name(service_name_);
  request->set_service_config_id(service_config_id_);

  Timestamp current_time = CreateTimestamp(info.current_time);
  Operation* op = request->add_operations();
  SetOperationCommonFields(info, current_time, op);
  if (info.check_response_info.api_key_state ==
      api_key::ApiKeyState::VERIFIED) {
    ASSERT(!info.api_key.empty(),
           "API Key must be set, otherwise consumer would not be verified.");
    op->set_consumer_id(std::string(kConsumerIdApiKey) +
                        std::string(info.api_key));
  }

  // Only populate metrics if we can associate them with a method/operation.
  if (!info.operation_id.empty() && !info.operation_name.empty()) {
    Map<std::string, std::string>* labels = op->mutable_labels();
    // Set all labels with by_consumer_only is false
    for (auto it = labels_.begin(), end = labels_.end(); it != end; it++) {
      const SupportedLabel* l = *it;
      if (l->set && !l->by_consumer_only) {
        status = (l->set)(*l, info, labels);
        if (!status.ok()) return status;
      }
    }

    // Report will reject consumer metric if it's based on a invalid/unknown api
    // key, or if the service is not activated in the consumer project.
    bool send_consumer_metric = info.check_response_info.api_key_state ==
                                api_key::ApiKeyState::VERIFIED;

    // Populate all metrics.
    for (auto it = metrics_.begin(), end = metrics_.end(); it != end; it++) {
      const SupportedMetric* m = *it;
      if (send_consumer_metric || m->mark != SupportedMetric::CONSUMER) {
        if (m->set && m->mark != SupportedMetric::PRODUCER_BY_CONSUMER) {
          status = (m->set)(*m, info, op);
          if (!status.ok()) return status;
        }
      }
    }
  }

  // Fill log entries.
  for (auto it = logs_.begin(), end = logs_.end(); it != end; it++) {
    FillLogEntry(info, *it, service_config_id_, current_time,
                 op->add_log_entries());
  }

  if (!info.check_response_info.consumer_project_number.empty()) {
    return AppendByConsumerOperations(info, request, current_time);
  }

  return Status::OK;
}

Status RequestBuilder::AppendByConsumerOperations(
    const ReportRequestInfo& info,
    ::google::api::servicecontrol::v1::ReportRequest* request,
    Timestamp current_time) const {
  Operation* op = request->add_operations();
  SetOperationCommonFields(info, current_time, op);
  if (info.check_response_info.api_key_state ==
      api_key::ApiKeyState::VERIFIED) {
    ASSERT(!info.api_key.empty(),
           "API Key must be set, otherwise consumer would not be verified.");
    op->set_consumer_id(std::string(kConsumerIdApiKey) +
                        std::string(info.api_key));
  }
  // issue a new operation id
  op->set_operation_id(op->operation_id() + "1");

  // Only populate metrics if we can associate them with a method/operation.
  if (!info.operation_id.empty() && !info.operation_name.empty()) {
    Map<std::string, std::string>* labels = op->mutable_labels();
    // Set all labels.
    for (auto it = labels_.begin(), end = labels_.end(); it != end; it++) {
      const SupportedLabel* l = *it;
      if (l->set) {
        Status status = (l->set)(*l, info, labels);
        if (!status.ok()) return status;
      }
    }

    // Populate all metrics.
    for (auto it = metrics_.begin(), end = metrics_.end(); it != end; it++) {
      const SupportedMetric* m = *it;
      if (m->set && m->mark == SupportedMetric::PRODUCER_BY_CONSUMER) {
        Status status = (m->set)(*m, info, op);
        if (!status.ok()) return status;
      }
    }
  }

  return Status::OK;
}

bool RequestBuilder::IsMetricSupported(
    const ::google::api::MetricDescriptor& metric) {
  for (int i = 0; i < supported_metrics_count; i++) {
    const SupportedMetric& m = supported_metrics[i];
    if (metric.name() == m.name && metric.metric_kind() == m.metric_kind &&
        metric.value_type() == m.value_type) {
      return true;
    }
  }
  return false;
}

bool RequestBuilder::IsLabelSupported(
    const ::google::api::LabelDescriptor& label) {
  for (int i = 0; i < supported_labels_count; i++) {
    const SupportedLabel& l = supported_labels[i];
    if (label.key() == l.name && label.value_type() == l.value_type) {
      return true;
    }
  }
  return false;
}

}  // namespace service_control
}  // namespace api_proxy
}  // namespace espv2
