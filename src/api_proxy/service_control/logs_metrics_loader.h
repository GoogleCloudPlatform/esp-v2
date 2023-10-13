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

#include <functional>

#include "absl/status/status.h"
#include "google/api/service.pb.h"
#include "gtest/gtest_prod.h"

namespace espv2 {
namespace api_proxy {
namespace service_control {

class LogsMetricsLoader final {
 public:
  static absl::Status Load(const ::google::api::Service& service,
                           std::set<std::string>* logs,
                           std::set<std::string>* metrics,
                           std::set<std::string>* labels);

 private:
  LogsMetricsLoader(std::function<bool(const ::google::api::LabelDescriptor&)>
                        label_supported,
                    std::function<bool(const ::google::api::MetricDescriptor&)>
                        metric_supported)
      : label_supported_(label_supported),
        metric_supported_(metric_supported) {}

  absl::Status AddLabels(
      const ::google::protobuf::RepeatedPtrField<
          ::google::api::LabelDescriptor>& descriptors,
      std::map<std::string, const ::google::api::LabelDescriptor&>* labels);

  absl::Status AddLogLabels(
      const ::google::protobuf::RepeatedPtrField<::google::api::LogDescriptor>&
          descriptors,
      const std::string& log_name,
      std::map<std::string, const ::google::api::LabelDescriptor&>* labels);

  absl::Status AddMonitoredResourceLabels(
      const ::google::protobuf::RepeatedPtrField<
          ::google::api::MonitoredResourceDescriptor>& descriptors,
      const std::string& monitored_resource_name,
      std::map<std::string, const ::google::api::LabelDescriptor&>* labels);

  absl::Status AddLoggingDestinations(
      const ::google::protobuf::RepeatedPtrField<
          ::google::api::Logging_LoggingDestination>& destinations,
      const ::google::protobuf::RepeatedPtrField<
          ::google::api::MonitoredResourceDescriptor>& monitored_resources,
      const ::google::protobuf::RepeatedPtrField<::google::api::LogDescriptor>&
          log_descriptors,
      std::set<std::string>* logs,
      std::map<std::string, const ::google::api::LabelDescriptor&>* labels);

  absl::Status AddMonitoringDestinations(
      const ::google::protobuf::RepeatedPtrField<
          ::google::api::Monitoring_MonitoringDestination>& destinations,
      const ::google::protobuf::RepeatedPtrField<
          ::google::api::MonitoredResourceDescriptor>& monitored_resources,
      const ::google::protobuf::RepeatedPtrField<
          ::google::api::MetricDescriptor>& metric_descriptors,
      std::map<std::string, const ::google::api::MetricDescriptor&>* metrics,
      std::map<std::string, const ::google::api::LabelDescriptor&>* labels);

  absl::Status LoadLogsMetrics(const ::google::api::Service& service,
                               std::set<std::string>* logs,
                               std::set<std::string>* metrics,
                               std::set<std::string>* labels);

  std::function<bool(const ::google::api::LabelDescriptor&)> label_supported_;
  std::function<bool(const ::google::api::MetricDescriptor&)> metric_supported_;

 private:
  FRIEND_TEST(LogsMetricsLoader, AddDuplicateLabels);
  FRIEND_TEST(LogsMetricsLoader, AddConflictingLabels);
  FRIEND_TEST(LogsMetricsLoader, AddUnsupportedLabels);
  FRIEND_TEST(LogsMetricsLoader, AddRedundantLabels);
  FRIEND_TEST(LogsMetricsLoader, AddNoLabels);
  FRIEND_TEST(LogsMetricsLoader, AddLogLabels);
  FRIEND_TEST(LogsMetricsLoader, AddMonitoredResourceLabels);
  FRIEND_TEST(LogsMetricsLoader, AddLoggingDestinations);
  FRIEND_TEST(LogsMetricsLoader, AddMonitoringDestinations);
  FRIEND_TEST(LogsMetricsLoader, LoadLogsMetrics);
};

}  // namespace service_control
}  // namespace api_proxy
}  // namespace espv2
