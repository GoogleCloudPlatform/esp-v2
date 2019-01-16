#include "src/envoy/utils/token_subscriber.h"
#include "common/grpc/async_client_manager_impl.h"

using ::google::api_proxy::agent::GetAccessTokenRequest;
using ::google::api_proxy::agent::GetAccessTokenResponse;

namespace Envoy {
namespace Extensions {
namespace Utils {
namespace {

// gRPC request timeout
const std::chrono::milliseconds kGrpcRequestTimeoutMs(5000);

// Delay after a failed fetch
const std::chrono::seconds kFailedRequestTimeout(60);

const google::protobuf::MethodDescriptor& descriptor() {
  static const google::protobuf::MethodDescriptor* descriptor =
      ::google::api_proxy::agent::AgentService::descriptor()->FindMethodByName(
          "GetAccessToken");
  ASSERT(descriptor);
  return *descriptor;
}

}  // namespace

Envoy::Grpc::AsyncClientFactoryPtr makeClinetFactory(
    Envoy::Server::Configuration::FactoryContext& context,
    const std::string& token_cluster) {
  ::envoy::api::v2::core::GrpcService grpc_service;
  grpc_service.mutable_envoy_grpc()->set_cluster_name(token_cluster);
  return std::make_unique<Envoy::Grpc::AsyncClientFactoryImpl>(
      context.clusterManager(), grpc_service, true, context.timeSource());
}

TokenSubscriber::TokenSubscriber(Envoy::Server::Configuration::FactoryContext& context,
                                 Envoy::Grpc::AsyncClientFactoryPtr client_factory,
                                 TokenSubscriber::Callback& callback)
    : client_factory_(std::move(client_factory)), token_callback_(callback) {
  refresh_timer_ =
      context.dispatcher().createTimer([this]() -> void { refresh(); });

  context.initManager().registerTarget(*this);
}

TokenSubscriber::~TokenSubscriber() {
  if (active_request_) {
    active_request_->cancel();
  }
}

void TokenSubscriber::initialize(std::function<void()> callback) {
  initialize_callback_ = callback;
  refresh();
}

void TokenSubscriber::refresh() {
  if (active_request_) {
    active_request_->cancel();
  }

  async_client_ = client_factory_->create();
  ENVOY_LOG(debug, "Sending GetAccessToken request");
  active_request_ = async_client_->send(
      descriptor(), GetAccessTokenRequest(), *this,
      Envoy::Tracing::NullSpan::instance(),
      absl::optional<std::chrono::milliseconds>(kGrpcRequestTimeoutMs));
}

void TokenSubscriber::onSuccess(
    std::unique_ptr<GetAccessTokenResponse>&& response, Envoy::Tracing::Span&) {
  active_request_ = nullptr;
  ENVOY_LOG(debug, "GetAccessToken got response: {}", response->DebugString());
  token_callback_.onTokenUpdate(response->access_token());
  runInitializeCallbackIfAny();
  // Update the token 5 seconds before the expiration
  if (response->expires_in().seconds() <= 5) {
    refresh();
  } else {
    refresh_timer_->enableTimer(
        std::chrono::seconds(response->expires_in().seconds() - 5));
  }
}

void TokenSubscriber::onFailure(Envoy::Grpc::Status::GrpcStatus status,
                                const std::string& message, Envoy::Tracing::Span&) {
  active_request_ = nullptr;
  ENVOY_LOG(debug, "GetAccessToken failed with code: {}, {}", status, message);
  runInitializeCallbackIfAny();
  refresh_timer_->enableTimer(kFailedRequestTimeout);
}

void TokenSubscriber::runInitializeCallbackIfAny() {
  if (initialize_callback_) {
    initialize_callback_();
    initialize_callback_ = nullptr;
  }
}

}  // namespace Utils
}  // namespace Extensions
}  // namespace Envoy
