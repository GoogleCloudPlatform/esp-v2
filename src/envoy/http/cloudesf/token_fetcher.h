#pragma once

#include "envoy/common/pure.h"
#include "envoy/upstream/cluster_manager.h"
#include "src/envoy/http/cloudesf/config.pb.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace CloudESF {

class TokenFetcher;
typedef std::unique_ptr<TokenFetcher> TokenFetcherPtr;
/**
 * TokenFetcher interface can be used to retrieve remote access_token from
 * GCP metadata server.
 */
class TokenFetcher {
 public:
  class TokenReceiver {
   public:
    enum class Failure {
      /* A network error occurred causing Token retrieval failure. */
      Network,
      /* A failure occurred when trying to parse the retrieved JSON. */
      InvalidToken,
    };

    virtual ~TokenReceiver(){};
    /*
     * Successful retrieval callback.
     * @param token.
     */
    virtual void onTokenSuccess(const std::string& token, int expires_in) PURE;
    /*
     * Retrieval error callback.
     * * @param reason the failure reason.
     */
    virtual void onTokenError(Failure reason) PURE;
  };

  virtual ~TokenFetcher(){};

  /*
   * Cancel any in-flight request.
   */
  virtual void cancel() PURE;

  /*
   * Retrieve a access token from a remote HTTP server.
   * At most one outstanding request may be in-flight,
   * i.e. from the invocation of `fetch()` until either
   * a callback or `cancel()` is invoked, no
   * additional `fetch()` may be issued.
   * @param uri the uri to retrieve the token from.
   * @param receiver the receiver of the fetched token.
   */
  virtual void fetch(
      const ::envoy::config::filter::http::cloudesf::HttpUri& uri,
      TokenReceiver& receiver) PURE;

  /*
   * Factory method for creating a TokenFetcher.
   * @param cm the cluster manager to use during Token retrieval
   * @return a TokenFetcher instance
   */
  static TokenFetcherPtr create(Upstream::ClusterManager& cm);
};

}  // namespace CloudESF
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
