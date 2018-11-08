#pragma once

#include "api/envoy/http/service_control/config.pb.h"
#include "envoy/upstream/cluster_manager.h"
#include "google/protobuf/stubs/status.h"
#include "src/envoy/http/service_control/cancel_func.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

class TokenFetcher {
 public:
  struct Result {
    std::string token;
    int expires_in;
  };

  using DoneFunc = std::function<void(
      const ::google::protobuf::util::Status& status, const Result& result)>;

  static CancelFunc fetch(
      Upstream::ClusterManager& cm,
      const ::google::api_proxy::envoy::http::service_control::HttpUri& uri,
      DoneFunc on_done);
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
