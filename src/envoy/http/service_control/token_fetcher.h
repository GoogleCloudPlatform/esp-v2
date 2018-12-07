#pragma once

#include "envoy/common/time.h"
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
    int64_t expires_in;
  };

  using DoneFunc = std::function<void(
      const ::google::protobuf::util::Status& status, const Result& result)>;

  static CancelFunc fetch(Upstream::ClusterManager& cm, TimeSource& time_source,
                          const std::string& token_cluster, DoneFunc on_done);
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
