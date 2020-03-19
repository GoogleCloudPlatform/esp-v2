// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#include "src/envoy/http/service_control/http_call.h"

#include <memory>

#include "common/common/empty_string.h"
#include "common/common/enum_to_int.h"
#include "common/http/headers.h"
#include "common/http/message_impl.h"
#include "common/http/utility.h"
#include "common/tracing/http_tracer_impl.h"
#include "envoy/event/deferred_deletable.h"

using ::google::api::envoy::http::common::HttpUri;
using ::google::protobuf::util::Status;
using ::google::protobuf::util::error::Code;
namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {
namespace {

constexpr absl::string_view KApplicationProto = "application/x-protobuf";

class HttpCallImpl : public HttpCall,
                     public Event::DeferredDeletable,
                     public Logger::Loggable<Logger::Id::filter>,
                     public Http::AsyncClient::Callbacks {
 public:
  HttpCallImpl(Upstream::ClusterManager& cm, Event::Dispatcher& dispatcher,
               const HttpUri& uri, const std::string& suffix_url,
               std::function<const std::string&()> token_fn,
               const Protobuf::Message& body, uint32_t timeout_ms,
               uint32_t retries, Envoy::Tracing::Span& parent_span,
               Envoy::TimeSource& time_source,
               const std::string& trace_operation_name)
      : cm_(cm),
        dispatcher_(dispatcher),
        http_uri_(uri),
        retries_(retries),
        request_count_(0),
        timeout_ms_(timeout_ms),
        cancelled(false),
        token_fn_(token_fn),
        parent_span_(parent_span),
        time_source_(time_source),
        trace_operation_name_(trace_operation_name) {
    uri_ = http_uri_.uri() + suffix_url;

    Http::Utility::extractHostPathFromUri(uri_, host_, path_);
    body.SerializeToString(&str_body_);

    ASSERT(!on_done_);
    ENVOY_LOG(trace, "{}", __func__);
  }

  void setDoneFunc(HttpCall::DoneFunc on_done) { on_done_ = on_done; }

  void call() override { makeOneCall(); }

  // HTTP async receive methods
  void onSuccess(Http::ResponseMessagePtr&& response) override {
    ENVOY_LOG(trace, "{}", __func__);

    std::string body;
    try {
      const uint64_t status_code =
          Http::Utility::getResponseStatus(response->headers());

      request_span_->setTag(Tracing::Tags::get().HttpStatusCode,
                            std::to_string(status_code));
      request_span_->finishSpan();

      if (response->body()) {
        const auto len = response->body()->length();
        body = std::string(static_cast<char*>(response->body()->linearize(len)),
                           len);
      }
      if (status_code == enumToInt(Http::Code::OK)) {
        ENVOY_LOG(debug, "http call [uri = {}]: success with body {}", uri_,
                  body);
        on_done_(Status::OK, body);
      } else {
        if (attemptRetry(status_code)) {
          return;
        }

        ENVOY_LOG(debug, "http call response status code: {}, body: {}",
                  status_code, body);
        on_done_(Status(Code::INTERNAL, "Failed to call service control"),
                 body);
      }
    } catch (const EnvoyException& e) {
      ENVOY_LOG(debug, "http call invalid status");
      on_done_(Status(Code::INTERNAL, "Failed to call service control"), body);
    }

    reset();
    deferredDelete();
  }

  void onFailure(Http::AsyncClient::FailureReason reason) override {
    // The status code in reason is always 0.
    ENVOY_LOG(debug, "http call network error");

    switch (reason) {
      case Http::AsyncClient::FailureReason::Reset:
        request_span_->setTag(Tracing::Tags::get().Error,
                              "the stream has been reset");
        break;
      default:
        request_span_->setTag(Tracing::Tags::get().Error,
                              "unknown network error");
        break;
    }
    request_span_->finishSpan();

    if (attemptRetry(0)) {
      return;
    }

    on_done_(Status(Code::INTERNAL, "Failed to call service control"),
             std::string());
    reset();
    deferredDelete();
  }

 private:
  bool attemptRetry(const uint64_t& status_code) {
    // skip if it is the client side problem.
    if (status_code >= 400 && status_code < 500) {
      return false;
    }
    if (retries_ <= 0) {
      return false;
    }
    retries_--;
    ENVOY_LOG(debug,
              "after {} times failures, retrying http call [uri = {}], with "
              "{} remaining chances",
              request_count_, uri_, retries_);

    reset();
    makeOneCall();
    return true;
  }

  void makeOneCall() {
    request_count_++;
    std::string token = token_fn_();
    if (token.empty()) {
      on_done_(Status(Code::INTERNAL,
                      "Missing access token for service control call"),
               EMPTY_STRING);
      deferredDelete();
      return;
    }

    // Trace the request
    auto span_name = request_count_ == 1
                         ? trace_operation_name_
                         : absl::StrCat(trace_operation_name_, " - Retry ",
                                        request_count_ - 1);
    request_span_ =
        parent_span_.spawnChild(Envoy::Tracing::EgressConfig::get(), span_name,
                                time_source_.systemTime());
    request_span_->setTag(Tracing::Tags::get().Component,
                          Tracing::Tags::get().Proxy);
    request_span_->setTag(Tracing::Tags::get().UpstreamCluster,
                          http_uri_.cluster());
    request_span_->setTag(Tracing::Tags::get().HttpUrl, uri_);
    request_span_->setTag(Tracing::Tags::get().HttpMethod, "POST");

    Http::RequestMessagePtr message = prepareHeaders(token);
    ENVOY_LOG(debug, "http call from [uri = {}]: start", uri_);
    request_ = cm_.httpAsyncClientForCluster(http_uri_.cluster())
                   .send(std::move(message), *this,
                         Http::AsyncClient::RequestOptions().setTimeout(
                             std::chrono::milliseconds(timeout_ms_)));
  }

  void cancel() override {
    if (cancelled) {
      return;
    }
    cancelled = true;
    ENVOY_LOG(debug, "Http call [uri = {}]: canceled", uri_);
    if (request_span_) {
      request_span_->setTag(Tracing::Tags::get().Error,
                            Tracing::Tags::get().Canceled);
      request_span_->finishSpan();
    }

    if (request_) {
      request_->cancel();
      ENVOY_LOG(debug, "Http call [uri = {}]: canceled", uri_);
      reset();
    }
    on_done_(Status(Code::CANCELLED, std::string("Request cancelled")),
             EMPTY_STRING);
    deferredDelete();
  }

  void reset() { request_ = nullptr; }

  Http::RequestMessagePtr prepareHeaders(const std::string& token) {
    Http::RequestMessagePtr message(new Http::RequestMessageImpl());
    message->headers().setPath(path_);
    message->headers().setHost(host_);

    message->headers().setReferenceMethod(
        Http::Headers::get().MethodValues.Post);

    message->body() =
        std::make_unique<Buffer::OwnedImpl>(str_body_.data(), str_body_.size());
    message->headers().setContentLength(message->body()->length());

    // assume token is not empty
    std::string token_value = "Bearer " + token;
    message->headers().setAuthorization(token_value);
    message->headers().setContentType(KApplicationProto);
    return message;
  }

  void deferredDelete() {
    dispatcher_.deferredDelete(std::unique_ptr<HttpCallImpl>(this));
  }

 private:
  // The upstream cluster manager
  Upstream::ClusterManager& cm_;
  // The dispatcher for this thread
  Event::Dispatcher& dispatcher_;

  // The request
  Http::AsyncClient::Request* request_{};

  // The callback function when request finished
  HttpCall::DoneFunc on_done_;

  // The serialized request body
  std::string str_body_;

  // The request uri
  std::string uri_;
  // The host of the request uri
  const HttpUri http_uri_;
  // The host of the request uri with buffer owned by uri_
  absl::string_view host_;
  // The path of the request uri with buffer owned by uri_
  absl::string_view path_;

  // The remaining retry times
  uint32_t retries_;
  // The sent request count
  uint32_t request_count_;
  // The timeout
  uint32_t timeout_ms_;
  // whether this call has been cancelled
  bool cancelled;

  // The function for getting token
  std::function<const std::string&()> token_fn_;

  // Tracing data
  Envoy::Tracing::Span& parent_span_;
  Envoy::TimeSource& time_source_;
  Envoy::Tracing::SpanPtr request_span_;
  const std::string trace_operation_name_;
};

}  // namespace

HttpCallFactory::HttpCallFactory(
    Upstream::ClusterManager& cm, Event::Dispatcher& dispatcher,
    const ::google::api::envoy::http::common::HttpUri& uri,
    const std::string& suffix_url, std::function<const std::string&()> token_fn,
    uint32_t timeout_ms, uint32_t retries, Envoy::TimeSource& time_source,
    const std::string& trace_operation_name)
    : cm_(cm),
      dispatcher_(dispatcher),
      uri_(uri),
      suffix_url_(suffix_url),
      token_fn_(token_fn),
      timeout_ms_(timeout_ms),
      retries_(retries),
      destruct_mode_(false),
      time_source_(time_source),
      trace_operation_name_(trace_operation_name){};

HttpCall* HttpCallFactory::createHttpCall(const Protobuf::Message& body,
                                          Envoy::Tracing::Span& parent_span,
                                          HttpCall::DoneFunc on_done) {
  ENVOY_LOG(debug, "{} is created", trace_operation_name_);
  HttpCallImpl* http_call = new HttpCallImpl(
      cm_, dispatcher_, uri_, suffix_url_, token_fn_, body, timeout_ms_,
      retries_, parent_span, time_source_, trace_operation_name_);
  http_call->setDoneFunc([this, on_done, http_call](const Status& status,
                                                    const std::string& body) {
    // When the call is finished, it should be removed from active_calls_ .
    // However, when the factory object is being destructed, all active_calls_
    // will be cancelled in one time so no need to remove them from
    // active_calls_ to avoid removing elements during for-loop iteration.
    if (!destruct_mode_) {
      active_calls_.erase(http_call);
    }
    on_done(status, body);
  });
  active_calls_.insert(http_call);
  return http_call;
}

HttpCallFactory::~HttpCallFactory() {
  destruct_mode_ = true;
  for (auto* httpCall : active_calls_) {
    httpCall->cancel();
  }
}

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
