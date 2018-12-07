#include "src/envoy/http/service_control/token_fetcher.h"

#include "api/agent/agent_service.pb.h"
#include "common/common/logger.h"
#include "common/grpc/async_client_impl.h"
#include "common/tracing/http_tracer_impl.h"

using ::google::api_proxy::agent::GetAccessTokenRequest;
using ::google::api_proxy::agent::GetAccessTokenResponse;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::error::Code;

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {
namespace {

// gRPC request timeout
const std::chrono::milliseconds kGrpcRequestTimeoutMs(5000);

class GrpcTokenFetcher
    : public Grpc::TypedAsyncRequestCallbacks<GetAccessTokenResponse>,
      public Logger::Loggable<Logger::Id::grpc> {
 public:
  GrpcTokenFetcher(Grpc::AsyncClientPtr async_client,
                   TokenFetcher::DoneFunc on_done)
      : async_client_(std::move(async_client)),
        on_done_(on_done),
        request_(async_client_->send(
            descriptor(), GetAccessTokenRequest(), *this,
            Tracing::NullSpan::instance(),
            absl::optional<std::chrono::milliseconds>(kGrpcRequestTimeoutMs))) {
    ENVOY_LOG(debug, "Sending GetAccessToken request");
  }

  void onCreateInitialMetadata(Http::HeaderMap&) override {}

  void onSuccess(std::unique_ptr<GetAccessTokenResponse>&& response,
                 Tracing::Span&) override {
    ENVOY_LOG(debug, "GetAccessToken got response: {}",
              response->DebugString());
    TokenFetcher::Result result = {response->access_token(),
                                   response->expires_in().seconds()};
    on_done_(Status::OK, result);
    delete this;
  }

  void onFailure(Grpc::Status::GrpcStatus status, const std::string& message,
                 Tracing::Span&) override {
    ENVOY_LOG(debug, "GetAccessToken failed with code: {}, {}", status,
              message);
    on_done_(Status(static_cast<Code>(status), message),
             TokenFetcher::Result());
    delete this;
  }

  void Cancel() {
    ENVOY_LOG(debug, "Cancel gRPC GetAccessToken request");
    request_->cancel();
    delete this;
  }

 private:
  static const google::protobuf::MethodDescriptor& descriptor() {
    static const google::protobuf::MethodDescriptor* descriptor =
        ::google::api_proxy::agent::AgentService::descriptor()
            ->FindMethodByName("GetAccessToken");
    ASSERT(descriptor);
    return *descriptor;
  }

  Grpc::AsyncClientPtr async_client_;
  TokenFetcher::DoneFunc on_done_;
  Grpc::AsyncRequest* request_{};
};

}  // namespace

CancelFunc TokenFetcher::fetch(Upstream::ClusterManager& cm,
                               TimeSource& time_source,
                               const std::string& token_cluster,
                               DoneFunc on_done) {
  envoy::api::v2::core::GrpcService config;
  config.mutable_envoy_grpc()->set_cluster_name(token_cluster);
  auto client =
      std::make_unique<Grpc::AsyncClientImpl>(cm, config, time_source);
  auto fetcher = new GrpcTokenFetcher(std::move(client), on_done);
  return [fetcher]() { fetcher->Cancel(); };
}

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
