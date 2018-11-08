#pragma once

#include <chrono>

#include "envoy/common/time.h"
#include "src/envoy/http/service_control/token_fetcher.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

// Number of seconds to refetch the token before expires
const int kRefetchBeforeExpire = 3;

class TokenCache {
 public:
  using DoneFunc =
      std::function<void(const ::google::protobuf::util::Status& status,
                         const std::string& token)>;

  TokenCache(
      Upstream::ClusterManager& cm, TimeSource& time_source,
      const ::google::api_proxy::envoy::http::service_control::HttpUri& uri)
      : cm_(cm), time_source_(time_source), uri_(uri) {}

  CancelFunc getToken(DoneFunc on_done) {
    if (time_source_.monotonicTime() >= expiration_time_) {
      return TokenFetcher::fetch(
          cm_, uri_,
          [this, on_done](const ::google::protobuf::util::Status& status,
                          const TokenFetcher::Result& result) {
            if (!status.ok()) {
              on_done(status, token_);
              return;
            }
            token_ = result.token;
            expiration_time_ =
                time_source_.monotonicTime() +
                std::chrono::seconds(result.expires_in - kRefetchBeforeExpire);
            on_done(::google::protobuf::util::Status::OK, token_);
          });
    }
    on_done(::google::protobuf::util::Status::OK, token_);
    return nullptr;
  }

 private:
  Upstream::ClusterManager& cm_;
  TimeSource& time_source_;
  const ::google::api_proxy::envoy::http::service_control::HttpUri& uri_;
  std::string token_;
  std::chrono::steady_clock::time_point expiration_time_;
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
