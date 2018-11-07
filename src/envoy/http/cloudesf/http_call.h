#pragma once

#include "envoy/common/pure.h"
#include "envoy/upstream/cluster_manager.h"
#include "google/protobuf/stubs/status.h"
#include "src/envoy/http/cloudesf/config.pb.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace CloudESF {

class HttpCall {
 public:
  using DoneFunc =
      std::function<void(const ::google::protobuf::util::Status& status,
                         const std::string& response_body)>;

  virtual ~HttpCall() {}
  /*
   * Cancel any in-flight request.
   */
  virtual void cancel() PURE;

  virtual void call(const std::string& suffix_url, const std::string& token,
                    const Protobuf::Message& body, DoneFunc on_done) PURE;

  /*
   * Factory method for creating a HttpCall.
   * @param cm the cluster manager to use during Token retrieval
   * @return a HttpCall instance
   */
  static HttpCall* create(
      Upstream::ClusterManager& cm,
      const ::envoy::config::filter::http::cloudesf::HttpUri& uri);
};

}  // namespace CloudESF
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
