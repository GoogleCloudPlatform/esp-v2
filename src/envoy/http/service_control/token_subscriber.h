#pragma once

#include "api/agent/agent_service.pb.h"
#include "common/common/logger.h"
#include "common/grpc/async_client_impl.h"
#include "envoy/common/pure.h"
#include "envoy/common/time.h"
#include "envoy/event/dispatcher.h"
#include "envoy/grpc/async_client_manager.h"
#include "envoy/init/init.h"
#include "envoy/server/filter_config.h"
#include "envoy/upstream/cluster_manager.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

// This class fetches a token at the config time in the main thread.
// It also registers a timer to fetch a new token before expiration.
//
// It is using InitManager object. This is how InitManager works:
//
// * If your filter needs to make an async remote call, and needs to
//   wait for the response to continue the data flow, you need to
//   implement a Init::Target and register your Init::Target with InitManager.
//
// * InitManager calls each InitTarget initialize() function at the main thread.
//   Each target starts to make its remote call. This function passes in a
//   callback function which should be called when the remote call response is
//   received.
//
class TokenSubscriber : public Init::Target,
                        public Grpc::TypedAsyncRequestCallbacks<
                            ::google::api_proxy::agent::GetAccessTokenResponse>,
                        public Logger::Loggable<Logger::Id::grpc> {
 public:
  class Callback {
   public:
    virtual ~Callback() {}
    virtual void onTokenUpdate(const std::string& token) PURE;
  };

  TokenSubscriber(Server::Configuration::FactoryContext& context,
                  Grpc::AsyncClientFactoryPtr client_factory,
                  Callback& callback);

  virtual ~TokenSubscriber();

  // Init::Target function
  void initialize(std::function<void()> callback) override;

  // Grpc::TypedAsyncRequestCallbacks functions
  void onCreateInitialMetadata(Http::HeaderMap&) override {}
  void onSuccess(
      std::unique_ptr<::google::api_proxy::agent::GetAccessTokenResponse>&&
          response,
      Tracing::Span&) override;
  void onFailure(Grpc::Status::GrpcStatus status, const std::string& message,
                 Tracing::Span&) override;

 private:
  void runInitializeCallbackIfAny();
  void refresh();

  Grpc::AsyncClientFactoryPtr client_factory_;
  Callback& token_callback_;

  std::function<void()> initialize_callback_;

  Grpc::AsyncClientPtr async_client_;
  Grpc::AsyncRequest* active_request_{};

  Event::TimerPtr refresh_timer_;
};
typedef std::unique_ptr<TokenSubscriber> TokenSubscriberPtr;

// Create Async Client Factory
Grpc::AsyncClientFactoryPtr makeClinetFactory(
    Server::Configuration::FactoryContext& context,
    const std::string& token_cluster);

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
