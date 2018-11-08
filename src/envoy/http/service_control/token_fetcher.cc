#include "src/envoy/http/service_control/token_fetcher.h"

#include "common/common/enum_to_int.h"
#include "common/http/headers.h"
#include "common/http/message_impl.h"
#include "common/http/utility.h"

using ::google::api_proxy::envoy::http::service_control::HttpUri;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::error::Code;

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {
namespace {

const Http::LowerCaseString kMetadataFlavor{"Metadata-Flavor"};
const std::string kGoogle{"Google"};

// The body is a JSON format as:
// { access_token: token, token_type: json, expires_in }
bool parseJsonToken(const std::string& json_token, std::string* token,
                    int* expires_in) {
  Protobuf::util::JsonParseOptions options;
  ProtobufWkt::Struct struct_pb;
  const auto status =
      Protobuf::util::JsonStringToMessage(json_token, &struct_pb, options);
  if (!status.ok()) {
    return false;
  }

  auto& logger = Logger::Registry::getLog(Logger::Id::config);
  ENVOY_LOG_TO_LOGGER(logger, debug, "struct info: {}",
                      struct_pb.DebugString());

  const auto token_it = struct_pb.fields().find("access_token");
  if (token_it == struct_pb.fields().end() ||
      token_it->second.kind_case() != ProtobufWkt::Value::kStringValue) {
    return false;
  }
  *token = token_it->second.string_value();

  const auto expires_it = struct_pb.fields().find("expires_in");
  if (expires_it == struct_pb.fields().end() ||
      expires_it->second.kind_case() != ProtobufWkt::Value::kNumberValue) {
    return false;
  }
  *expires_in = expires_it->second.number_value();

  return true;
}

Http::MessagePtr prepareHeaders(const HttpUri& http_uri) {
  absl::string_view host, path;
  Http::Utility::extractHostPathFromUri(http_uri.uri(), host, path);

  Http::MessagePtr message(new Http::RequestMessageImpl());
  message->headers().insertPath().value(path.data(), path.size());
  message->headers().insertHost().value(host.data(), host.size());

  message->headers().insertMethod().value().setReference(
      Http::Headers::get().MethodValues.Get);
  message->headers().addReference(kMetadataFlavor, kGoogle);

  return message;
}

class TokenFetcherImpl : public TokenFetcher,
                         public Logger::Loggable<Logger::Id::filter>,
                         public Http::AsyncClient::Callbacks {
 public:
  TokenFetcherImpl(Upstream::ClusterManager& cm, DoneFunc on_done)
      : cm_(cm), on_done_(on_done) {
    ENVOY_LOG(trace, "{}", __func__);
  }

  void cancel() {
    if (request_) {
      request_->cancel();
      ENVOY_LOG(debug, "fetch access_token [uri = {}]: canceled", uri_->uri());
      reset();
    }
    delete this;
  }

  void fetch(const HttpUri& uri) {
    ENVOY_LOG(trace, "{}", __func__);
    uri_ = &uri;
    Http::MessagePtr message = prepareHeaders(uri);
    ENVOY_LOG(debug, "fetch access_token from [uri = {}]: start", uri_->uri());
    request_ =
        cm_.httpAsyncClientForCluster(uri.cluster())
            .send(std::move(message), *this,
                  std::chrono::milliseconds(
                      DurationUtil::durationToMilliseconds(uri.timeout())));
  }

  // HTTP async receive methods
  void onSuccess(Http::MessagePtr&& response) {
    ENVOY_LOG(trace, "{}", __func__);
    const uint64_t status_code =
        Http::Utility::getResponseStatus(response->headers());
    Result result;
    if (status_code == enumToInt(Http::Code::OK)) {
      ENVOY_LOG(debug, "fetch access_token [uri = {}]: success", uri_->uri());
      if (response->body()) {
        const auto len = response->body()->length();
        const auto body = std::string(
            static_cast<char*>(response->body()->linearize(len)), len);

        ENVOY_LOG(debug, "fetch access_token JSON: {} succeeded", body);
        if (parseJsonToken(body, &result.token, &result.expires_in)) {
          ENVOY_LOG(debug, "parsed access_token: {}, expires_in: {}",
                    result.token, result.expires_in);
          on_done_(Status::OK, result);
        } else {
          ENVOY_LOG(debug, "fetch access_token: invalid format");
          on_done_(Status(Code::UNAVAILABLE, "invalid json format"), result);
        }
      } else {
        ENVOY_LOG(debug, "fetch access_token body is empty");
        on_done_(Status(Code::UNAVAILABLE, "empty body"), result);
      }
    } else {
      ENVOY_LOG(debug, "fetch access_token: response status code {}",
                status_code);
      on_done_(Status(Code::UNAVAILABLE, "network failure"), result);
    }
    reset();
    delete this;
  }

  void onFailure(Http::AsyncClient::FailureReason reason) {
    ENVOY_LOG(debug, "fetch access_token: network error {}", enumToInt(reason));
    Result result;
    on_done_(Status(Code::UNAVAILABLE, "network failure"), result);
    reset();
    delete this;
  }

 private:
  Upstream::ClusterManager& cm_;
  DoneFunc on_done_;
  const HttpUri* uri_{};
  Http::AsyncClient::Request* request_{};

  void reset() { request_ = nullptr; }
};

}  // namespace

CancelFunc TokenFetcher::fetch(
    Upstream::ClusterManager& cm,
    const ::google::api_proxy::envoy::http::service_control::HttpUri& uri,
    DoneFunc on_done) {
  auto fetcher = new TokenFetcherImpl(cm, on_done);
  fetcher->fetch(uri);
  return [fetcher]() { fetcher->cancel(); };
}

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
