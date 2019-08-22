// Copyright 2018 Google Cloud Platform Proxy Authors
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

#include <grpc++/grpc++.h>

#include <chrono>
#include <iostream>
#include <memory>
#include <set>
#include <string>

#include "tests/endpoints/grpc-echo/proto/grpc-test.grpc.pb.h"

using ::google::api::servicecontrol::v1::ReportRequest;
using ::grpc::InsecureServerCredentials;
using ::grpc::Server;
using ::grpc::ServerAsyncReaderWriter;
using ::grpc::ServerAsyncResponseWriter;
using ::grpc::ServerAsyncWriter;
using ::grpc::ServerBuilder;
using ::grpc::ServerCompletionQueue;
using ::grpc::ServerContext;
using ::grpc::Status;
using ::grpc::StatusCode;

namespace test {
namespace grpc {

typedef std::function<void(bool)> Tag;

static void *MakeTag(std::function<void(bool)> continuation) {
  return reinterpret_cast<void *>(new Tag(continuation));
}

// The base class for "corkable" calls -- calls that provide a
// standard mechanism for completing the call once the server is
// uncorked.
class CorkableCall {
 public:
  virtual ~CorkableCall() {}
  virtual void RunToCompletion() = 0;
};

class CorkCall;

class TestServer final : public Test::AsyncService {
 public:
  void Run(const char *addr);

  ServerCompletionQueue *completion_queue() { return cq_.get(); }

  const std::set<CorkableCall *> &pending_calls() { return pending_calls_; }

  void clear_pending_cork() { pending_cork_.reset(); }

  void Uncork();

 private:
  void QueuePendingCall(CorkableCall *call);

  void StartEcho();
  void StartEchoReport();
  void StartEchoStream();
  void StartCork();

  std::unique_ptr<ServerCompletionQueue> cq_;
  std::set<CorkableCall *> pending_calls_;
  std::shared_ptr<CorkCall> pending_cork_;
};

class EchoCall final : public CorkableCall {
 public:
  EchoCall(TestServer *test_server)
      : test_server_(test_server), context_(), responder_(&context_) {}

  void Start(void *tag) {
    test_server_->RequestEcho(&context_, &request_, &responder_,
                              test_server_->completion_queue(),
                              test_server_->completion_queue(), tag);
  }

  void RunToCompletion() override {
    response_.set_text(request_.text());
    auto &received_metadata = *response_.mutable_received_metadata();
    for (const auto &it : context_.client_metadata()) {
      received_metadata[std::string(it.first.data(), it.first.length())] =
          std::string(it.second.data(), it.second.length());
    }
    for (const auto &it : request_.return_initial_metadata()) {
      context_.AddInitialMetadata(it.first, it.second);
    }
    for (const auto &it : request_.return_trailing_metadata()) {
      context_.AddTrailingMetadata(it.first, it.second);
    }
    responder_.Finish(response_,
                      Status(StatusCode(request_.return_status().code()),
                             request_.return_status().details()),
                      MakeTag([this](bool ok) { delete this; }));
  }

 private:
  TestServer *test_server_;
  ServerContext context_;
  EchoRequest request_;
  EchoResponse response_;
  ServerAsyncResponseWriter<EchoResponse> responder_;
};

class EchoReportCall final : public CorkableCall {
 public:
  EchoReportCall(TestServer *test_server)
      : test_server_(test_server), context_(), responder_(&context_) {}

  void Start(void *tag) {
    test_server_->RequestEchoReport(&context_, &request_, &responder_,
                                    test_server_->completion_queue(),
                                    test_server_->completion_queue(), tag);
  }

  void RunToCompletion() override {
    // Echo the data back.
    response_ = request_;
    responder_.Finish(response_, Status::OK,
                      MakeTag([this](bool ok) { delete this; }));
  }

 private:
  TestServer *test_server_;
  ServerContext context_;
  ReportRequest request_;
  ReportRequest response_;
  ServerAsyncResponseWriter<ReportRequest> responder_;
};

class EchoStreamCall final : public CorkableCall {
 public:
  EchoStreamCall(TestServer *test_server)
      : test_server_(test_server), context_(), streamer_(&context_) {}

  void Start(void *tag) {
    test_server_->RequestEchoStream(&context_, &streamer_,
                                    test_server_->completion_queue(),
                                    test_server_->completion_queue(), tag);
  }

  void RunToCompletion() override {
    call_start_time_ = std::chrono::steady_clock::now();
    StartRead();
  }

 private:
  void StartRead() {
    streamer_.Read(&request_, MakeTag([this](bool ok) {
      if (!ok) {
        StartFinish();
      } else if (request_.return_status().code()) {
        StartFinish();
      } else {
        StartWrite();
      }
    }));
  }

  void StartWrite() {
    response_.set_text(request_.text());
    response_.set_elapsed_micros(
        std::chrono::duration_cast<std::chrono::microseconds>(
            std::chrono::steady_clock::now() - call_start_time_)
            .count());
    streamer_.Write(response_, MakeTag([this](bool ok) {
                      if (!ok) {
                        StartFinish();
                      } else {
                        StartRead();
                      }
                    }));
  }

  void StartFinish() {
    streamer_.Finish(Status(StatusCode(request_.return_status().code()),
                            request_.return_status().details()),
                     MakeTag([this](bool ok) { delete this; }));
  }

  TestServer *test_server_;
  ServerContext context_;
  EchoRequest request_;
  EchoResponse response_;
  ServerAsyncReaderWriter<EchoResponse, EchoRequest> streamer_;
  std::chrono::steady_clock::time_point call_start_time_;
};

class CorkCall final : public std::enable_shared_from_this<CorkCall> {
 public:
  CorkCall(TestServer *test_server)
      : test_server_(test_server), context_(), streamer_(&context_) {}

  void Start(void *tag) {
    test_server_->RequestCork(&context_, &streamer_,
                              test_server_->completion_queue(),
                              test_server_->completion_queue(), tag);
  }

  void StartRead() {
    if (finished_) {
      return;
    }
    std::shared_ptr<CorkCall> call = shared_from_this();
    streamer_.Read(&request_, MakeTag([call](bool ok) {
      if (!ok) {
        call->StartFinish();
      } else {
        // Discard the request message.
        call->StartRead();
      }
    }));
  }

  void ReportState() {
    if (finished_) {
      return;
    }
    std::shared_ptr<CorkCall> call = shared_from_this();
    CorkState *state = new CorkState();
    state->set_current_corked_calls(test_server_->pending_calls().size());
    streamer_.Write(*state, MakeTag([call, state](bool ok) {
      delete state;
      if (!ok) {
        call->StartFinish(
            Status(::grpc::INTERNAL, "Failed to send a cork state report"));
      }
    }));
  }

  void StartFinish(Status status = Status::OK) {
    if (finished_) {
      return;
    }
    finished_ = true;
    std::shared_ptr<CorkCall> call = shared_from_this();
    streamer_.Finish(status, MakeTag([call](bool ok) {}));
    test_server_->Uncork();
  }

 private:
  TestServer *test_server_;
  ServerContext context_;
  CorkRequest request_;
  ServerAsyncReaderWriter<CorkState, CorkRequest> streamer_;
  bool finished_ = false;
};

void TestServer::Uncork() {
  if (!pending_cork_) {
    return;
  }
  pending_cork_->StartFinish();
  pending_cork_.reset();
  std::set<CorkableCall *> pending;
  pending.swap(pending_calls_);
  for (CorkableCall *call : pending) {
    call->RunToCompletion();
  }
}

void TestServer::QueuePendingCall(CorkableCall *call) {
  pending_calls_.insert(call);
  pending_cork_->ReportState();
}

void TestServer::StartEcho() {
  auto *call = new EchoCall(this);
  call->Start(MakeTag([this, call](bool ok) {
    StartEcho();
    if (!ok) {
      delete call;
      return;
    }
    if (pending_cork_) {
      QueuePendingCall(call);
    } else {
      call->RunToCompletion();
    }
  }));
}

void TestServer::StartEchoReport() {
  auto *call = new EchoReportCall(this);
  call->Start(MakeTag([this, call](bool ok) {
    StartEchoReport();
    if (!ok) {
      delete call;
      return;
    }
    if (pending_cork_) {
      QueuePendingCall(call);
    } else {
      call->RunToCompletion();
    }
  }));
}

void TestServer::StartEchoStream() {
  auto *call = new EchoStreamCall(this);
  call->Start(MakeTag([this, call](bool ok) {
    if (!ok) {
      delete call;
      return;
    }
    StartEchoStream();
    if (pending_cork_) {
      QueuePendingCall(call);
    } else {
      call->RunToCompletion();
    }
  }));
}

void TestServer::StartCork() {
  std::shared_ptr<CorkCall> call(new CorkCall(this));
  call->Start(MakeTag([this, call](bool ok) {
    if (!ok) {
      return;
    }
    if (pending_cork_) {
      call->StartFinish(
          Status(::grpc::ALREADY_EXISTS, "A cork call is already in progress"));
    } else {
      pending_cork_ = call->shared_from_this();
      call->StartRead();
    }
    StartCork();
  }));
}

void TestServer::Run(const char *addr) {
  ServerBuilder builder;
  builder.AddListeningPort(addr, InsecureServerCredentials());
  builder.RegisterService(this);
  builder.SetMaxReceiveMessageSize(INT_MAX);
  builder.SetMaxSendMessageSize(INT_MAX);
  cq_ = builder.AddCompletionQueue();
  std::unique_ptr<Server> server(builder.BuildAndStart());

  StartEcho();
  StartEcho();
  StartEchoReport();
  StartEchoReport();
  StartEchoStream();
  StartEchoStream();
  StartCork();

  std::cout << "Test server listening at address " << addr << std::endl;

  void *tag;
  bool ok;
  while (cq_->Next(&tag, &ok)) {
    Tag *func = reinterpret_cast<Tag *>(tag);
    (*func)(ok);
    delete func;
  }
}

}  // namespace grpc
}  // namespace test

int main(int argc, char **argv) {
  if (argc < 2) {
    std::cerr << "Usage: grpc-test-server <listening address>" << std::endl;
    return EXIT_FAILURE;
  }

  ::test::grpc::TestServer server;
  server.Run(argv[1]);

  return EXIT_SUCCESS;
}
