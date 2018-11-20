#pragma once

#include "common/common/logger.h"
#include "envoy/access_log/access_log.h"
#include "envoy/http/filter.h"
#include "envoy/upstream/cluster_manager.h"
#include "src/envoy/http/service_control/config_parser.h"
#include "src/envoy/http/service_control/filter_config.h"
#include "src/envoy/http/service_control/http_call.h"

#include <string>

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

// The Envoy filter for Cloud ESF service control client.
class Filter : public Http::StreamDecoderFilter,
               public AccessLog::Instance,
               public Logger::Loggable<Logger::Id::filter> {
 public:
  Filter(FilterConfigSharedPtr config) : config_(config) {
    config_parser_ =  std::unique_ptr<ServiceControlFilterConfigParser>(
      new ServiceControlFilterConfigParser(config_->config()));
  }

  // Http::StreamFilterBase
  void onDestroy() override;

  // Http::StreamDecoderFilter
  Http::FilterHeadersStatus decodeHeaders(Http::HeaderMap& headers,
                                          bool) override;
  Http::FilterDataStatus decodeData(Buffer::Instance&, bool) override;
  Http::FilterTrailersStatus decodeTrailers(Http::HeaderMap&) override;
  void setDecoderFilterCallbacks(
      Http::StreamDecoderFilterCallbacks& callbacks) override;

  // Called when the request is completed.
  void log(const Http::HeaderMap* request_headers,
           const Http::HeaderMap* response_headers,
           const Http::HeaderMap* response_trailers,
           const StreamInfo::StreamInfo& stream_info) override;

 private:
  void onTokenDone(const ::google::protobuf::util::Status& status,
                   const std::string& token);
  void onCheckResponse(const ::google::protobuf::util::Status& status,
                       const std::string& response_json);
  void rejectRequest(Http::Code code, const std::string& error_msg);

  // The callback funcion.
  Http::StreamDecoderFilterCallbacks* decoder_callbacks_;
  FilterConfigSharedPtr config_;

  // Fills the given requirement if the query matches a pattern in Envoy
  // filter config.
  void ExtractRequestInfo(const Http::HeaderMap&,
    ::google::api::envoy::http::service_control::Requirement* requirement);

  // Helper functions to extract API key.
  void ExtractAPIKeyFromQuery(const Http::HeaderMap& headers,
                              const std::string& query);
  void ExtractAPIKeyFromHeader(const Http::HeaderMap& headers,
                               const std::string& header);
  void ExtractAPIKeyFromCookie(const Http::HeaderMap& headers,
                               const std::string& cookie);

  // The state of the request.
  enum State { Init, Calling, Responded, Complete };
  State state_ = Init;
  // Mark if request has been stopped.
  bool stopped_ = false;

  CancelFunc token_fetcher_;
  std::string token_;
  std::string uuid_;
  std::string operation_name_;
  std::string api_key_;
  std::string api_name_;
  std::string api_version_;
  std::string http_method_;
  std::unique_ptr<ServiceControlFilterConfigParser> config_parser_;

  ::google::api_proxy::service_control::CheckResponseInfo check_response_info_;
  ::google::protobuf::util::Status check_status_;
  HttpCall* check_call_{};
};

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
