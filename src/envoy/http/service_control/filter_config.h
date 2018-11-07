#pragma once

#include "api/envoy/http/service_control/config.pb.h"
#include "common/common/logger.h"
#include "envoy/runtime/runtime.h"
#include "src/api_proxy/service_control/request_builder.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

// The Envoy filter config for Cloud ESF service control client.
class FilterConfig : public Logger::Loggable<Logger::Id::filter> {
 public:
  FilterConfig(
      const ::google::api_proxy::envoy::http::service_control::FilterConfig&
          proto_config,
      Upstream::ClusterManager& cm, Runtime::RandomGenerator& random)
      : proto_config_(proto_config),
        cm_(cm),
        random_(random),
        builder_({"endpoints_log"}, proto_config_.service_name(),
                 proto_config_.service_config_id()) {}

  const ::google::api_proxy::envoy::http::service_control::FilterConfig&
  config() const {
    return proto_config_;
  }

  Upstream::ClusterManager& cm() { return cm_; }
  Runtime::RandomGenerator& random() { return random_; }
  ::google::api_proxy::service_control::RequestBuilder& builder() {
    return builder_;
  }

 private:
  // The proto config.
  ::google::api_proxy::envoy::http::service_control::FilterConfig proto_config_;
  Upstream::ClusterManager& cm_;
  Runtime::RandomGenerator& random_;
  ::google::api_proxy::service_control::RequestBuilder builder_;
};

typedef std::shared_ptr<FilterConfig> FilterConfigSharedPtr;

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
