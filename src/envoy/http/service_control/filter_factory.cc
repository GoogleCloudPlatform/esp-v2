

#include "api/envoy/http/service_control/config.pb.h"
#include "api/envoy/http/service_control/config.pb.validate.h"
#include "src/envoy/http/service_control/filter.h"

#include "envoy/registry/registry.h"
#include "extensions/filters/http/common/factory_base.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

const std::string FilterName = "envoy.filters.http.service_control";

/**
 * Config registration for cloudESF service control filter.
 */
class FilterFactory
    : public Common::FactoryBase<
          ::google::api_proxy::envoy::http::service_control::FilterConfig> {
 public:
  FilterFactory() : FactoryBase(FilterName) {}

 private:
  Http::FilterFactoryCb createFilterFactoryFromProtoTyped(
      const ::google::api_proxy::envoy::http::service_control::FilterConfig&
          proto_config,
      const std::string&,
      Server::Configuration::FactoryContext& context) override {
    auto filter_config = std::make_shared<FilterConfig>(
        proto_config, context.clusterManager(), context.random());
    return
        [filter_config](Http::FilterChainFactoryCallbacks& callbacks) -> void {
          auto filter = std::make_shared<Filter>(filter_config);
          callbacks.addStreamDecoderFilter(
              Http::StreamDecoderFilterSharedPtr(filter));
          callbacks.addAccessLogHandler(AccessLog::InstanceSharedPtr(filter));
        };
  }
};

/**
 * Static registration for the rate limit filter. @see RegisterFactory.
 */
static Registry::RegisterFactory<
    FilterFactory, Server::Configuration::NamedHttpFilterConfigFactory>
    register_;

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
