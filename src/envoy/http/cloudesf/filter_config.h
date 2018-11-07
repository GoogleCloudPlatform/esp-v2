#pragma once

#include "common/common/logger.h"
#include "envoy/runtime/runtime.h"
#include "src/envoy/http/cloudesf/config.pb.h"
#include "src/envoy/http/cloudesf/service_control/proto.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace CloudESF {

// The Envoy filter config for Cloud ESF service control client.
class FilterConfig : public Logger::Loggable<Logger::Id::filter> {
 public:
  FilterConfig(
      const ::envoy::config::filter::http::cloudesf::FilterConfig& proto_config,
      Upstream::ClusterManager& cm, Runtime::RandomGenerator& random)
      : proto_config_(proto_config),
        cm_(cm),
        random_(random),
        proto_builder_({"endpoints_log"}, proto_config_.service_name(),
                       proto_config_.service_config_id()) {}

  const ::envoy::config::filter::http::cloudesf::FilterConfig& config() const {
    return proto_config_;
  }

  Upstream::ClusterManager& cm() { return cm_; }
  Runtime::RandomGenerator& random() { return random_; }
  ::google::service_control::Proto& proto_builder() { return proto_builder_; }

 private:
  // The proto config.
  ::envoy::config::filter::http::cloudesf::FilterConfig proto_config_;
  Upstream::ClusterManager& cm_;
  Runtime::RandomGenerator& random_;
  ::google::service_control::Proto proto_builder_;
};

typedef std::shared_ptr<FilterConfig> FilterConfigSharedPtr;

}  // namespace CloudESF
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
