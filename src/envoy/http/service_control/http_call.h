#pragma once

#include "api/envoy/http/service_control/config.pb.h"
#include "envoy/common/pure.h"
#include "envoy/upstream/cluster_manager.h"
#include "google/protobuf/stubs/status.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

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
      const ::google::api_proxy::envoy::http::service_control::HttpUri& uri);
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
