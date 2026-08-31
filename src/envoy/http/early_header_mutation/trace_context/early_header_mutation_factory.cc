#include "api/envoy/v12/http/trace_context/config.pb.h"
#include "src/envoy/http/early_header_mutation/trace_context/early_header_mutation.h"
#include "envoy/http/early_header_mutation.h"
#include "envoy/registry/registry.h"
#include "source/common/protobuf/utility.h"

namespace espv2 {
namespace envoy {
namespace http_filters {
namespace trace_context {

class TraceContextEarlyHeaderMutationFactory
    : public Envoy::Http::EarlyHeaderMutationFactory {
 public:
  Envoy::Http::EarlyHeaderMutationPtr createExtension(
      const Protobuf::Message& config,
      Envoy::Server::Configuration::FactoryContext& context) override {
    const auto& typed_config =
        Envoy::MessageUtil::downcastAndValidate<const ::envoy::v12::http::trace_context::TraceContextTranslatorConfig&>(
            config, context.messageValidationVisitor());
    return std::make_unique<TraceContextEarlyHeaderMutation>(
        std::make_shared<::envoy::v12::http::trace_context::TraceContextTranslatorConfig>(
            typed_config));
  }

  Envoy::ProtobufTypes::MessagePtr createEmptyConfigProto() override {
    return std::make_unique<::envoy::v12::http::trace_context::TraceContextTranslatorConfig>();
  }

  std::string name() const override {
    return "com.google.espv2.filters.http.early_header_mutation.trace_context";
  }
};

/**
 * Static registration for the Trace Context early header mutation.
 */
static Envoy::Registry::RegisterFactory<
    TraceContextEarlyHeaderMutationFactory,
    Envoy::Http::EarlyHeaderMutationFactory>
    register_;

}  // namespace trace_context
}  // namespace http_filters
}  // namespace envoy
}  // namespace espv2
