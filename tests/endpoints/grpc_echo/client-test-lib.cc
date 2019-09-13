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

#include <chrono>
#include <cmath>
#include <cstdlib>
#include <forward_list>
#include <functional>
#include <mutex>
#include <random>
#include <string>
#include <type_traits>

#include <grpc++/alarm.h>
#include <grpc++/grpc++.h>

#include "google/protobuf/util/message_differencer.h"
#include "test/test_common/utility.h"
#include "tests/endpoints/grpc_echo/client-test-lib.h"
// #include "src/api_manager/utils/marshalling.h"

using ::google::api::servicecontrol::v1::ReportRequest;
using ::google::protobuf::util::MessageDifferencer;
using ::grpc::Alarm;
using ::grpc::Channel;
using ::grpc::ChannelArguments;
using ::grpc::ChannelCredentials;
using ::grpc::ClientAsyncReaderWriter;
using ::grpc::ClientAsyncResponseReader;
using ::grpc::ClientContext;
using ::grpc::ClientReaderWriter;
using ::grpc::CompletionQueue;
using ::grpc::CreateCustomChannel;
using ::grpc::InsecureChannelCredentials;
using ::grpc::SslCredentials;
using ::grpc::SslCredentialsOptions;
using ::grpc::Status;
using ::grpc::StatusCode;

using ::google::protobuf::Message;
using ::google::protobuf::TextFormat;
using ::google::protobuf::io::IstreamInputStream;
using ::google::protobuf::io::OstreamOutputStream;
using ::google::protobuf::util::TypeResolver;
using ::google::protobuf::util::error::Code;
const char kTypeUrlPrefix[] = "type.googleapis.com";
const char kRpcStatusTypeUrl[] = "type.googleapis.com/google.rpc.Status";


namespace test {
namespace grpc {
namespace {
// Creation function used by static lazy init.
TypeResolver* CreateTypeResolver() {
  return ::google::protobuf::util::NewTypeResolverForDescriptorPool(
      kTypeUrlPrefix, ::google::protobuf::DescriptorPool::generated_pool());
}

TypeResolver* GetTypeResolver() {
  static TypeResolver* resolver = CreateTypeResolver();
  return resolver;
}

std::string GetTypeUrl(const Message& message) {
  return std::string(kTypeUrlPrefix) + "/" +
      message.GetDescriptor()->full_name();
}

::google::protobuf::util::Status FromProto(
    const ::google::protobuf::util::Status &proto_status) {
  if (proto_status.ok()) {
    return ::google::protobuf::util::Status::OK;
  }
  return proto_status;
}
}

::google::protobuf::util::Status JsonToProto(const std::string &json,
                                             Message *message) {
  ::google::protobuf::util::JsonParseOptions options;
  options.ignore_unknown_fields = true;
  std::string binary;
  ::google::protobuf::util::Status status =
      ::google::protobuf::util::JsonToBinaryString(
          GetTypeResolver(), GetTypeUrl(*message), json, &binary, options);
  if (!status.ok()) {
    return FromProto(status);
  }
  if (message->ParseFromString(binary)) {
    return ::google::protobuf::util::Status::OK;
  }
  return ::google::protobuf::util::Status(
      Code::INTERNAL,
      "Unable to parse bytes generated from JsonToBinaryString as proto.");
}

::google::protobuf::util::Status ProtoToJson(const Message& message,
                   ::google::protobuf::io::ZeroCopyOutputStream* json,
                   int options) {
  ::google::protobuf::util::JsonPrintOptions json_options;
  if (options & JsonOptions::PRETTY_PRINT) {
    json_options.add_whitespace = true;
  }
  if (options & JsonOptions::OUTPUT_DEFAULTS) {
    json_options.always_print_primitive_fields = true;
  }
  // TODO: Skip going to bytes and use ProtoObjectSource directly.
  std::string binary = message.SerializeAsString();
  ::google::protobuf::io::ArrayInputStream binary_stream(binary.data(),
                                                         binary.size());
  ::google::protobuf::util::Status status =
      ::google::protobuf::util::BinaryToJsonStream(
          GetTypeResolver(), GetTypeUrl(message), &binary_stream, json,
          json_options);
  return FromProto(status);
}


namespace {

typedef std::function<void(bool)> Tag;

template <class T>
std::shared_ptr<ChannelCredentials> GetCreds(const T &unused_desc) {
  return InsecureChannelCredentials();
}

template <>
std::shared_ptr<ChannelCredentials> GetCreds(const CallConfig &call_config) {
  if (call_config.use_ssl()) {
    SslCredentialsOptions opts;
    return ::grpc::SslCredentials(opts);
  }
  return InsecureChannelCredentials();
}

template <>
std::shared_ptr<ChannelCredentials> GetCreds(const EchoTest &desc) {
  return GetCreds(desc.call_config());
}

template <>
std::shared_ptr<ChannelCredentials> GetCreds(const EchoStreamTest &desc) {
  return GetCreds(desc.call_config());
}

template <class T>
static std::unique_ptr<Test::Stub> GetStub(const std::string &addr,
                                           const T &desc) {
  ChannelArguments args;
  args.SetMaxReceiveMessageSize(INT_MAX);
  args.SetMaxSendMessageSize(INT_MAX);
  std::shared_ptr<Channel> channel(
      CreateCustomChannel(addr, GetCreds(desc), args));
  return std::unique_ptr<Test::Stub>(Test::NewStub(channel));
}

void SetCallConfig(const CallConfig &call_config, ClientContext *ctx) {
  if (!call_config.api_key().empty()) {
    ctx->AddMetadata("x-api-key", call_config.api_key());
  }
  if (!call_config.auth_token().empty()) {
    ctx->AddMetadata("authorization", "Bearer " + call_config.auth_token());
  }
  for (const auto &it : call_config.metadata()) {
    ctx->AddMetadata(it.first, it.second);
  }
  if (call_config.compression()) {
    ctx->set_compression_algorithm(
        static_cast<grpc_compression_algorithm>(call_config.compression()));
  }
}

template <class T1, class T2>
int VerifyMetadata(
    const T1 &expected, const T2 &actual,
    google::protobuf::RepeatedPtrField<EchoResult::MetadataResult> *results) {
  int verified = 0;
  for (const auto &it : actual) {
    std::string key = std::string(it.first.data(), it.first.size());
    std::string value = std::string(it.second.data(), it.second.size());
    if (expected.find(key) == expected.end()) continue;
    if (expected.at(key) == value) {
      ++verified;
    } else {
      auto *metadata = results->Add();
      metadata->set_key(key);
      metadata->set_expected(expected.at(key));
      metadata->set_actual(value);
    }
  }
  return verified;
}

std::string RandomString(int max_size) {
  static std::random_device rd;
  static std::mt19937 gen(rd());
  std::uniform_int_distribution<int> size_dist(0, max_size);
  std::uniform_int_distribution<unsigned char> byte_dist;

  int size = size_dist(gen);
  if (size == 0) {
    return std::string();
  }
  unsigned char buf[size];
  for (int i = 0; i < size; i++) {
    buf[i] = byte_dist(gen);
  }
  return std::string(reinterpret_cast<const char *>(buf), size);
}

void CreateEchoRequestText(EchoRequest *request) {
  if (request->random_payload_max_size() > 0) {
    request->set_text(RandomString(request->random_payload_max_size()));
  }
  if (request->space_payload_size() > 0) {
    request->set_text(std::string(request->space_payload_size(), ' '));
  }
}

std::string GetStatusAggregationKey(const CallStatus &status) {
  std::string details = status.details();
  ::test::grpc::GrpcErrorDetail grpc_error;
  // grpc error message has time stamp fields.
  // only keep description and http2_error fields as key.
  if (JsonToProto(details, &grpc_error).ok()) {
    grpc_error.SerializeToString(&details);
  }
  return details + std::to_string(status.code());
}

class Echo {
 public:
  typedef EchoTest TestDesc;

  static void Start(const EchoTest &desc, Test::Stub *server_stub,
                    Test::Stub *direct_stub, CompletionQueue *cq,
                    std::function<void(bool, const TestResult &)> done) {
    std::shared_ptr<Echo> echo(new Echo(desc, done));
    echo->rpc_ = server_stub->AsyncEcho(&echo->ctx_, echo->desc_.request(), cq);
    echo->rpc_->Finish(
        &echo->response_, &echo->status_, new Tag([echo, done](bool ok) {
          TestResult result;
          if (!echo->status_.ok()) {
            result.mutable_status()->set_code(echo->status_.error_code());
            result.mutable_status()->set_details(echo->status_.error_message());
          } else {
            result.mutable_echo()->set_text(echo->response_.text());
          }
          ok = ((echo->desc_.expected_status().code() ==
                 echo->status_.error_code()) &&
                (echo->desc_.expected_status().details() ==
                 echo->status_.error_message()));
          if (echo->status_.ok()) {
            ok &= (echo->desc_.request().text() == echo->response_.text());

            auto *metadata_results =
                result.mutable_echo()->mutable_metadata_result();
            int verified_metadata = 0;
            verified_metadata += VerifyMetadata(
                echo->desc_.call_config().metadata(),
                echo->response_.received_metadata(), metadata_results);
            verified_metadata += VerifyMetadata(
                echo->desc_.request().return_initial_metadata(),
                echo->ctx_.GetServerInitialMetadata(), metadata_results);
            verified_metadata += VerifyMetadata(
                echo->desc_.request().return_trailing_metadata(),
                echo->ctx_.GetServerTrailingMetadata(), metadata_results);
            result.mutable_echo()->set_verified_metadata(verified_metadata);
            ok &= metadata_results->size() == 0;
          }

          for (auto key : echo->desc_.expected_metadata_keys()) {
            auto it = echo->response_.received_metadata().find(key);
            if (it != echo->response_.received_metadata().end()) {
              (*(result.mutable_additional_metadata()))[key] = it->second;
            }
          }
          done(ok, result);
        }));
  }

 private:
  Echo(const EchoTest &desc, std::function<void(bool, const TestResult &)> done)
      : desc_(desc), done_(done) {
    SetCallConfig(desc_.call_config(), &ctx_);
    CreateEchoRequestText(desc_.mutable_request());
  }

  EchoTest desc_;
  std::function<void(bool, const TestResult &)> done_;
  ClientContext ctx_;
  EchoResponse response_;
  Status status_;
  std::unique_ptr<ClientAsyncResponseReader<EchoResponse>> rpc_;
};

class EchoReport {
 public:
  typedef EchoReportTest TestDesc;

  static void Start(const EchoReportTest &desc, Test::Stub *server_stub,
                    Test::Stub *direct_stub, CompletionQueue *cq,
                    std::function<void(bool, const TestResult &)> done) {
    std::shared_ptr<EchoReport> echo(new EchoReport(desc, done));
    echo->rpc_ =
        server_stub->AsyncEchoReport(&echo->ctx_, echo->desc_.request(), cq);
    echo->rpc_->Finish(
        &echo->response_, &echo->status_, new Tag([echo, done](bool ok) {
          TestResult result;
          if (!echo->status_.ok()) {
            result.mutable_status()->set_code(echo->status_.error_code());
            result.mutable_status()->set_details(echo->status_.error_message());
            ok = false;
          } else {
            *result.mutable_echo_report()->mutable_response() = echo->response_;
            ok = true;
          }
          if (echo->status_.ok()) {
            ok &= MessageDifferencer::Equals(echo->desc_.request(),
                                             echo->response_);
          }
          done(ok, result);
        }));
  }

 private:
  EchoReport(const EchoReportTest &desc,
             std::function<void(bool, const TestResult &)> done)
      : desc_(desc), done_(done) {}

  EchoReportTest desc_;
  std::function<void(bool, const TestResult &)> done_;
  ClientContext ctx_;
  ReportRequest response_;
  Status status_;
  std::unique_ptr<ClientAsyncResponseReader<ReportRequest>> rpc_;
};

class EchoStream {
 public:
  typedef EchoStreamTest TestDesc;

  static void Start(const EchoStreamTest &desc, Test::Stub *server_stub,
                    Test::Stub *direct_stub, CompletionQueue *cq,
                    std::function<void(bool, const TestResult &)> done) {
    std::shared_ptr<EchoStream> es(new EchoStream(desc, server_stub, cq, done));
    es->rpc_ =
        server_stub->AsyncEchoStream(&es->ctx_, cq, new Tag([es](bool ok) {
          if (ok) {
            StartWrite(es);
            StartRead(es);
          } else {
            StartFinish(es);
          }
        }));
  }

  EchoStream(const EchoStreamTest &desc, Test::Stub *server_stub,
             CompletionQueue *cq,
             std::function<void(bool, const TestResult &)> done)
      : desc_(desc),
        server_stub_(server_stub),
        cq_(cq),
        done_(done),
        read_count_expected_(desc.count()),
        start_time_(time(NULL)) {
    SetCallConfig(desc_.call_config(), &ctx_);
    for (auto request : *desc_.mutable_request()) {
      CreateEchoRequestText(&request);
    }
  }

  static void StartWrite(std::shared_ptr<EchoStream> es,
                         bool write_accounted_for = false) {
    if (es->started_done_ || es->started_finish_) {
      return;
    }
    if (!write_accounted_for) {
      es->write_count_++;
    }
    int request_index = es->write_count_ - 1;
    if (es->desc_.request_size() <= request_index) {
      request_index = es->desc_.request_size() - 1;
    }
    const CallStatus &status = es->desc_.expected_status();
    if (status.code()) {
      es->read_count_expected_ = request_index;
      es->status_expected_ =
          Status(StatusCode(status.code()), status.details());
    }
    es->rpc_->Write(es->desc_.request(request_index), new Tag([es](bool ok) {
                      if (ok) {
                        es->in_flight_++;
                        if (es->in_flight_ > es->max_in_flight_) {
                          es->max_in_flight_ = es->in_flight_;
                        }
                      } else {
                        StartFinish(es);
                      }
                    }));
  }

  static void StartRead(std::shared_ptr<EchoStream> es) {
    auto response = std::make_shared<EchoResponse>();
    es->rpc_->Read(response.get(), new Tag([es, response](bool ok) {
                     if (ok) {
                       StartRead(es);
                       es->read_count_++;
                       es->in_flight_--;
                       int curr_time_sec = time(NULL);

                       if ((es->desc_.count() > 0 &&
                            es->write_count_ < es->desc_.count()) ||
                           (es->desc_.duration_in_sec() > 0 &&
                            (curr_time_sec - es->start_time_) <
                                es->desc_.duration_in_sec())) {
                         StartWrite(es);
                       } else {
                         StartDone(es);
                       }

                     } else {
                       StartFinish(es);
                     }
                   }));
  }

  static void StartDone(std::shared_ptr<EchoStream> es) {
    if (es->started_done_) {
      return;
    }
    es->started_done_ = true;
    es->rpc_->WritesDone(new Tag([es](bool ok) { StartFinish(es); }));
  }

  static void StartFinish(std::shared_ptr<EchoStream> es) {
    if (es->started_finish_) {
      return;
    }
    es->started_finish_ = true;
    es->rpc_->Finish(&es->status_, new Tag([es](bool ok) {
      TestResult result;
      if (ok) {
        if (!es->status_.ok()) {
          result.mutable_status()->set_code(es->status_.error_code());
          result.mutable_status()->set_details(es->status_.error_message());
        }

        result.mutable_echo_stream()->set_count(es->read_count_);

        ok &= es->status_.error_code() == es->status_expected_.error_code();
        ok &= (es->status_.error_message() ==
               es->status_expected_.error_message());
        if (es->status_.ok()) {
          ok &= (es->read_count_expected_ == es->read_count_);
        }
      }
      es->done_(ok, result);
    }));
  }

  EchoStreamTest desc_;
  Test::Stub *server_stub_;
  CompletionQueue *cq_;
  std::function<void(bool, const TestResult &)> done_;
  ClientContext ctx_;
  bool started_done_ = false;
  bool started_finish_ = false;
  int write_count_ = 0;          // => the number of writes we have committed to
  int read_count_ = 0;           // => the number of reads we have received
  int read_count_expected_ = 0;  // => the number of reads we expect
  int in_flight_ = 0;
  int max_in_flight_ = 0;
  Status status_;
  Status status_expected_;
  std::unique_ptr<ClientAsyncReaderWriter<EchoRequest, EchoResponse>> rpc_;
  int start_time_;  // stream start time
};

class Parallel {
 public:
  typedef ParallelTest TestDesc;

  static void Start(const ParallelTest &desc, Test::Stub *server_stub,
                    Test::Stub *direct_stub, CompletionQueue *cq,
                    std::function<void(bool, const TestResult &)> done) {
    std::shared_ptr<Parallel> p(new Parallel(desc, server_stub, cq, done));
    StartTests(p);
  }

 private:
  Parallel(const ParallelTest &desc, Test::Stub *server_stub,
           CompletionQueue *cq,
           std::function<void(bool, const TestResult &)> done)
      : desc_(desc),
        cq_(cq),
        server_stub_(server_stub),
        done_(done),
        weight_limit_(GetWeightSum(desc)),
        test_start_time_(std::chrono::steady_clock::now()),
        subtest_stats_(desc.subtests_size()) {}

  static int GetWeightSum(const ParallelTest &desc) {
    int total_weight = 0;
    for (const auto &st : desc.subtests()) {
      total_weight += st.weight();
    }
    return total_weight;
  }

  static void StartTests(std::shared_ptr<Parallel> p) {
    std::lock_guard<std::mutex> lock(p->mu_);
    StartTestsLocked(p);
  }

  static void StartTestsLocked(std::shared_ptr<Parallel> p) {
    while (p->running_ < p->desc_.parallel_limit() &&
           p->started_ < p->desc_.test_count()) {
      p->running_++;
      p->started_++;
      if (p->started_ % 1000 == 0) {
        std::cerr << "Started tests: " << p->started_ << std::endl;
      }

      int subtest_type_index = p->NextSubtestType();

      auto subtest_start_time = std::chrono::steady_clock::now();

      auto subtest_done = [p, subtest_type_index, subtest_start_time](
                              bool ok, const TestResult &result) {
        auto subtest_end_time = std::chrono::steady_clock::now();

        std::lock_guard<std::mutex> lock(p->mu_);
        p->running_--;
        SubtestStats &stats = p->subtest_stats_[subtest_type_index];
        if (ok) {
          stats.succeeded_count++;
        } else {
          stats.failed_count++;
          stats.AddFailure(result.status());
        }
        stats.latencies.push_front(
            std::chrono::duration_cast<std::chrono::microseconds>(
                subtest_end_time - subtest_start_time));
        StartTestsLocked(p);
        if (!p->running_ && p->started_ == p->desc_.test_count()) {
          p->CalculateStats();
          p->done_(true, p->result_);
        }
      };

      const auto &subtest = p->desc_.subtests(subtest_type_index);
      switch (subtest.plan_case()) {
        case ParallelSubtest::kEcho:
          Echo::Start(subtest.echo(), p->server_stub_, nullptr, p->cq_,
                      subtest_done);
          break;
        case ParallelSubtest::kEchoStream:
          EchoStream::Start(subtest.echo_stream(), p->server_stub_, nullptr,
                            p->cq_, subtest_done);
          break;
        default:
          std::cerr << "Unimplemented test plan" << std::endl;
          break;
      }
    }
  }

  // Calculates the index of the next subtest type to run.
  int NextSubtestType() {
    int weight = weight_current_;

    weight_current_++;
    if (weight_limit_ <= weight_current_) {
      weight_current_ = 0;
    }

    int subtest_type_index = 0;
    for (const auto &st : desc_.subtests()) {
      if (weight < st.weight()) {
        break;
      }
      weight -= st.weight();
      subtest_type_index++;
    }
    return subtest_type_index;
  }

  // Fills in results_ after the subtests are complete.
  void CalculateStats() {
    auto *res = result_.mutable_parallel();
    res->set_total_time_micros(
        std::chrono::duration_cast<std::chrono::microseconds>(
            std::chrono::steady_clock::now() - test_start_time_)
            .count());
    int total_succeeded_count = 0;
    int total_failed_count = 0;
    for (const auto &it : subtest_stats_) {
      ParallelSubtestStats *stats = res->add_stats();
      stats->set_succeeded_count(it.succeeded_count);
      stats->set_failed_count(it.failed_count);
      total_succeeded_count += it.succeeded_count;
      total_failed_count += it.failed_count;
      std::int64_t total_latency = 0;
      size_t num_latencies = 0;
      for (const auto &latency : it.latencies) {
        total_latency += latency.count();
        num_latencies++;
      }
      std::int64_t mean_latency = total_latency / num_latencies;
      std::int64_t sum_diffs_latency = 0;
      for (const auto &latency : it.latencies) {
        std::int64_t delta_latency = mean_latency - latency.count();
        sum_diffs_latency += delta_latency * delta_latency;
      }
      std::int64_t variance_latency = sum_diffs_latency / num_latencies;
      std::int64_t stddev_latency = std::sqrt(variance_latency);

      stats->set_mean_latency_micros(mean_latency);
      stats->set_stddev_latency_micros(stddev_latency);

      for (const auto &f : it.failures) {
        f.second.Fill(stats->add_failures());
      }
    }
    if (total_failed_count >
        static_cast<int>(desc_.test_count() * desc_.allowed_failure_rate())) {
      result_.mutable_status()->set_code(::grpc::UNKNOWN);
      result_.mutable_status()->set_details("Parallel test failed.");
    }
    // These two lines are for the result extraction in script/release-qualify
    std::cerr << "Complete requests " << total_succeeded_count << std::endl
              << "Failed requests " << total_failed_count << std::endl;
  }

  class AggregatedStatus {
   public:
    AggregatedStatus(const CallStatus &status) : count_(1), status_(status) {
      start_time_ = end_time_ = time(nullptr);
    }

    void Inc() {
      count_++;
      end_time_ = time(nullptr);
    }

    void Fill(AggregatedCallStatus *call_status) const {
      call_status->set_count(count_);
      call_status->set_start_time(ctime(&start_time_));
      call_status->set_end_time(ctime(&end_time_));
      *call_status->mutable_status() = status_;
    }

   private:
    int count_ = 0;
    time_t start_time_ = -1;
    time_t end_time_ = -1;
    CallStatus status_;
  };

  struct SubtestStats {
    int succeeded_count = 0;
    int failed_count = 0;
    std::forward_list<std::chrono::microseconds> latencies;
    std::map<std::string, AggregatedStatus> failures;

    void AddFailure(const CallStatus &status) {
      std::string key = GetStatusAggregationKey(status);
      std::map<std::string, AggregatedStatus>::iterator it = failures.find(key);
      if (it == failures.end()) {
        failures.emplace(key, status);
      } else {
        it->second.Inc();
      }
    }
  };

  ParallelTest desc_;
  CompletionQueue *cq_;
  Test::Stub *server_stub_;
  std::function<void(bool, const TestResult &)> done_;
  int weight_current_ = 0;
  int weight_limit_;
  std::chrono::time_point<std::chrono::steady_clock> test_start_time_;
  std::vector<SubtestStats> subtest_stats_;

  std::mutex mu_;
  int started_ = 0;    // The number of started tests; only increases.
  int running_ = 0;    // The number of currently running tests.
  TestResult result_;  // Builds the final TestResult.
};

class ProbeCallLimit {
 public:
  typedef ProbeCallLimitTest TestDesc;

  static void Start(const ProbeCallLimitTest &desc, Test::Stub *server_stub,
                    Test::Stub *direct_stub, CompletionQueue *cq,
                    std::function<void(bool, const TestResult &)> done) {
    std::shared_ptr<ProbeCallLimit> pc(
        new ProbeCallLimit(desc, server_stub, cq, done));
    pc->cork_rpc_ =
        direct_stub->AsyncCork(&pc->cork_ctx_, cq, new Tag([pc](bool ok) {
          if (!ok) {
            TestResult result;
            result.mutable_status()->set_code(::grpc::UNKNOWN);
            result.mutable_status()->set_details(
                "Unable to start the cork RPC");
            pc->done_(false, result);
            return;
          }
          StartReadCorkState(pc);
          StartProbe(pc);
        }));
  }

 private:
  ProbeCallLimit(const ProbeCallLimitTest &desc, Test::Stub *server_stub,
                 CompletionQueue *cq,
                 std::function<void(bool, const TestResult &)> done)
      : desc_(desc), server_stub_(server_stub), cq_(cq), done_(done) {
    CreateEchoRequestText(desc_.mutable_request());
  }

  static void StartReadCorkState(std::shared_ptr<ProbeCallLimit> pc) {
    pc->cork_rpc_->Read(&pc->cork_state_, new Tag([pc](bool ok) {
      if (!ok) {
        // There's nothing to read; typically this means the RPC
        // is shutting down (the server->client half of the
        // connection is closed).  This typically happens because
        // we've already closed the other direction, by calling
        // WritesDone(); the subsequent Finish() will return the
        // RPC status.
        return;
      }
      StartReadCorkState(pc);
      // There should be an alarm; cancel it.  Note that the
      // alarm's callback will still fire; we use the result in
      // the alarm callback to determine whether to set up the
      // next probe.  (We can't delete the alarm itself until its
      // tag is dequeued).
      if (pc->alarm_) {
        pc->alarm_->Cancel();
      }
    }));
  }

  static void StartProbe(std::shared_ptr<ProbeCallLimit> pc) {
    // First, set an alarm.  If this alarm goes off, we're at the call limit.
    pc->alarm_.reset(new Alarm(
        pc->cq_,
        gpr_time_add(
            gpr_now(GPR_CLOCK_MONOTONIC),
            gpr_time_from_millis(pc->desc_.timeout_ms(), GPR_TIMESPAN)),
        new Tag([pc](bool ok) {
          if (ok) {
            // The alarm expired (it wasn't cancelled).  So assume
            // we're at the call limit.
            pc->alarm_.reset();
            OnTimeout(pc);
          } else {
            // The alarm was cancelled before it could expire -- so
            // the probe got through.  So now we need to reset the
            // timer and start another probe.
            StartProbe(pc);
          }
        })));

    // Next, start an echo call.
    EchoResponse *response = new EchoResponse();
    Status *status = new Status();
    ClientContext *ctx = new ClientContext();
    ClientAsyncResponseReader<EchoResponse> *rpc =
        pc->server_stub_->AsyncEcho(ctx, pc->desc_.request(), pc->cq_)
            .release();

    // Queue up the finish.
    rpc->Finish(response, status,
                new Tag([response, status, ctx, rpc, pc](bool ok) {
                  delete rpc;
                  delete ctx;
                  delete status;
                  delete response;
                }));

    // And then wait for either the cork state change, or the alarm
    // timeout.
  }

  static void OnTimeout(std::shared_ptr<ProbeCallLimit> pc) {
    // We're done here.  Whatever the cork state indicates the current
    // number of corked calls is, that's our result.
    int call_limit = pc->cork_state_.current_corked_calls();

    pc->cork_rpc_->WritesDone(new Tag([pc, call_limit](bool ok) {
      Status *status = new Status();
      pc->cork_rpc_->Finish(
          status, new Tag([pc, status, call_limit](bool ok) {
            delete status;
            TestResult result;
            result.mutable_probe_call_limit()->set_call_limit(call_limit);
            pc->done_(true, result);
          }));
    }));
  }

  ProbeCallLimitTest desc_;
  Test::Stub *server_stub_;
  CompletionQueue *cq_;
  std::function<void(bool, const TestResult &)> done_;
  ClientContext cork_ctx_;
  // The latest received state from the Cork() streaming RPC.
  CorkState cork_state_;
  std::unique_ptr<ClientAsyncReaderWriter<CorkRequest, CorkState>> cork_rpc_;
  std::unique_ptr<Alarm> alarm_;
};

// Probes for the presence of flow control from the client to the
// server.  It does so by instructing the server to not process any
// requests ("corking" the server), and then issuing requests to the
// server via ESP, setting a timer at the start of each request.  If
// the timer expires before the request write succeeds, the test
// assumes that this is a sign of flow control pushing back on the
// client.
class ProbeDownstreamMessageLimit {
 public:
  typedef ProbeDownstreamMessageLimitTest TestDesc;

  static void Start(const ProbeDownstreamMessageLimitTest &desc,
                    Test::Stub *server_stub, Test::Stub *direct_stub,
                    CompletionQueue *cq,
                    std::function<void(bool, const TestResult &)> done) {
    std::shared_ptr<ProbeDownstreamMessageLimit> pc(
        new ProbeDownstreamMessageLimit(desc, server_stub, cq, done));
    pc->cork_rpc_ =
        direct_stub->AsyncCork(&pc->cork_ctx_, cq, new Tag([pc](bool ok) {
          if (!ok) {
            TestResult result;
            result.mutable_status()->set_code(::grpc::UNKNOWN);
            result.mutable_status()->set_details(
                "Unable to start the cork RPC");
            pc->done_(false, result);
            return;
          }
          StartReadCorkState(pc);
          pc->stream_rpc_ = pc->server_stub_->AsyncEchoStream(
              &pc->stream_ctx_, pc->cq_, new Tag([pc](bool ok) {
                StartReadStreamMessages(pc);
                StartProbe(pc);
              }));
        }));
  }

 private:
  ProbeDownstreamMessageLimit(
      const ProbeDownstreamMessageLimitTest &desc, Test::Stub *server_stub,
      CompletionQueue *cq, std::function<void(bool, const TestResult &)> done)
      : desc_(desc), server_stub_(server_stub), cq_(cq), done_(done) {
    CreateEchoRequestText(desc_.mutable_request());
  }

  static void StartReadCorkState(
      std::shared_ptr<ProbeDownstreamMessageLimit> pc) {
    pc->cork_rpc_->Read(&pc->cork_state_, new Tag([pc](bool ok) {
      if (!ok) {
        return;
      }
      StartReadCorkState(pc);
    }));
  }

  static void StartReadStreamMessages(
      std::shared_ptr<ProbeDownstreamMessageLimit> pc) {
    pc->stream_rpc_->Read(&pc->stream_response_, new Tag([pc](bool ok) {
      if (!ok) {
        return;
      }
      StartReadStreamMessages(pc);
    }));
  }

  static void StartProbe(std::shared_ptr<ProbeDownstreamMessageLimit> pc) {
    // First, set an alarm.  If this alarm goes off, we're at the message limit.

    // This is a little unusual: we're allocating a shared_ptr on the
    // heap.  The reason why is that we want to be able to smuggle the
    // reference to the alarm into the alarm's callback, so that the
    // alarm always remains valid until the callback completes.
    std::shared_ptr<Alarm> *alarmp = new std::shared_ptr<Alarm>();
    *alarmp = std::make_shared<Alarm>(
        pc->cq_,
        gpr_time_add(
            gpr_now(GPR_CLOCK_MONOTONIC),
            gpr_time_from_millis(pc->desc_.timeout_ms(), GPR_TIMESPAN)),
        new Tag([pc, alarmp](bool ok) {
          if (ok) {
            // The alarm expired (it wasn't cancelled).  So assume
            // we're at the message limit.
            OnTimeout(pc);
          } else {
            // Start the next probe.
            StartProbe(pc);
          }
          delete alarmp;
        }));

    // Next, send a message.
    std::shared_ptr<Alarm> alarm = *alarmp;
    pc->writing_ = true;
    pc->stream_rpc_->Write(pc->desc_.request(), new Tag([pc, alarm](bool ok) {
                             pc->writing_ = false;
                             if (pc->saw_timeout_) {
                               StartWritesDone(pc);
                             } else {
                               pc->downstream_message_limit_++;
                             }
                             // The write succeeded; cancel the alarm.  This
                             // will cause
                             // the alarm's callback to fire with ok==false, if
                             // it hasn't
                             // already fired.
                             alarm->Cancel();
                           }));
  }

  static void OnTimeout(std::shared_ptr<ProbeDownstreamMessageLimit> pc) {
    // We're done here.
    pc->saw_timeout_ = true;

    pc->cork_rpc_->WritesDone(new Tag([pc](bool ok) {
      Status *status = new Status();
      pc->cork_rpc_->Finish(status, new Tag([pc, status](bool ok) {
                              delete status;
                              pc->cork_finished_ = true;
                              TryIssueReport(pc);
                            }));
    }));

    if (!pc->writing_) {
      StartWritesDone(pc);
    }
  }

  static void StartWritesDone(std::shared_ptr<ProbeDownstreamMessageLimit> pc) {
    pc->stream_rpc_->WritesDone(new Tag([pc](bool ok) {
      Status *status = new Status();
      pc->stream_rpc_->Finish(status, new Tag([pc, status](bool ok) {
                                delete status;
                                pc->stream_finished_ = true;
                                TryIssueReport(pc);
                              }));
    }));
  }

  static void TryIssueReport(std::shared_ptr<ProbeDownstreamMessageLimit> pc) {
    if (!(pc->cork_finished_ && pc->stream_finished_)) {
      return;
    }

    TestResult result;
    result.mutable_probe_downstream_message_limit()->set_message_limit(
        pc->downstream_message_limit_);
    pc->done_(true, result);
  }

  ProbeDownstreamMessageLimitTest desc_;
  Test::Stub *server_stub_;
  CompletionQueue *cq_;
  std::function<void(bool, const TestResult &)> done_;
  ClientContext cork_ctx_;
  ClientContext stream_ctx_;
  CorkState cork_state_;
  EchoResponse stream_response_;
  std::unique_ptr<ClientAsyncReaderWriter<CorkRequest, CorkState>> cork_rpc_;
  std::unique_ptr<ClientAsyncReaderWriter<EchoRequest, EchoResponse>>
      stream_rpc_;
  int downstream_message_limit_ = 0;
  bool saw_timeout_ = false;
  bool cork_finished_ = false;
  bool stream_finished_ = false;
  bool writing_ = false;
};

// Probes for the presence of flow control from the server to the
// client.  It does so by sending messages for the server to echo back
// to the client, setting a timer at the start of each message, and
// then not reading the echoed messages, causing the pipeline to fill
// up.  If the timer expires before the request write succeeds, the
// test assumes that this is a sign of flow control pushing back on
// the client; since the server is simply echoing messages as it
// receives them, this implies that flow control is pushing back on
// the server.
class ProbeUpstreamMessageLimit {
 public:
  typedef ProbeUpstreamMessageLimitTest TestDesc;

  static void Start(const ProbeUpstreamMessageLimitTest &desc,
                    Test::Stub *server_stub, Test::Stub *direct_stub,
                    CompletionQueue *cq,
                    std::function<void(bool, const TestResult &)> done) {
    std::shared_ptr<ProbeUpstreamMessageLimit> pc(
        new ProbeUpstreamMessageLimit(desc, server_stub, cq, done));
    pc->stream_rpc_ = pc->server_stub_->AsyncEchoStream(
        &pc->stream_ctx_, pc->cq_, new Tag([pc](bool ok) {
          if (!ok) {
            TestResult result;
            result.mutable_status()->set_code(::grpc::UNKNOWN);
            result.mutable_status()->set_details(
                "Unable to start the EchoStream RPC");
            pc->done_(false, result);
            return;
          }
          StartProbe(pc);
        }));
  }

 private:
  ProbeUpstreamMessageLimit(const ProbeUpstreamMessageLimitTest &desc,
                            Test::Stub *server_stub, CompletionQueue *cq,
                            std::function<void(bool, const TestResult &)> done)
      : desc_(desc), server_stub_(server_stub), cq_(cq), done_(done) {
    CreateEchoRequestText(desc_.mutable_request());
  }

  // Note: We only start StartReadStreamMessages here when the timeout
  // expires, i.e. when we detect backpressure.
  static void StartReadStreamMessages(
      std::shared_ptr<ProbeUpstreamMessageLimit> pc) {
    pc->stream_rpc_->Read(&pc->stream_response_, new Tag([pc](bool ok) {
      if (!ok) {
        return;
      }
      StartReadStreamMessages(pc);
    }));
  }

  static void StartProbe(std::shared_ptr<ProbeUpstreamMessageLimit> pc) {
    // First, set an alarm.  If this alarm goes off, we're at the message limit.

    // This is a little unusual: we're allocating a shared_ptr on the
    // heap.  The reason why is that we want to be able to smuggle the
    // reference to the alarm into the alarm's callback, so that the
    // alarm always remains valid until the callback completes.
    std::shared_ptr<Alarm> *alarmp = new std::shared_ptr<Alarm>();
    *alarmp = std::make_shared<Alarm>(
        pc->cq_,
        gpr_time_add(
            gpr_now(GPR_CLOCK_MONOTONIC),
            gpr_time_from_millis(pc->desc_.timeout_ms(), GPR_TIMESPAN)),
        new Tag([pc, alarmp](bool ok) {
          if (ok) {
            // The alarm expired (it wasn't cancelled).  So assume
            // we're at the message limit.
            OnTimeout(pc);
          } else {
            // Start the next probe.
            StartProbe(pc);
          }
          delete alarmp;
        }));

    // Next, send a message.
    std::shared_ptr<Alarm> alarm = *alarmp;
    pc->writing_ = true;
    pc->stream_rpc_->Write(pc->desc_.request(), new Tag([pc, alarm](bool ok) {
                             pc->writing_ = false;
                             if (pc->saw_timeout_) {
                               StartWritesDone(pc);
                             } else {
                               pc->upstream_message_limit_++;
                             }
                             // The write succeeded; cancel the alarm.  This
                             // will cause the alarm's callback to fire with
                             // ok==false, if it hasn't already fired.
                             alarm->Cancel();
                           }));
  }

  static void OnTimeout(std::shared_ptr<ProbeUpstreamMessageLimit> pc) {
    pc->saw_timeout_ = true;

    // Since the timeout has occurred, start draining the incoming
    // queued-up messages, so that the RPC can actually finish.
    StartReadStreamMessages(pc);

    if (!pc->writing_) {
      StartWritesDone(pc);
    }
  }

  static void StartWritesDone(std::shared_ptr<ProbeUpstreamMessageLimit> pc) {
    pc->stream_rpc_->WritesDone(new Tag([pc](bool ok) {
      Status *status = new Status();
      pc->stream_rpc_->Finish(
          status, new Tag([pc, status](bool ok) {
            delete status;
            TestResult result;
            result.mutable_probe_upstream_message_limit()->set_message_limit(
                pc->upstream_message_limit_);
            pc->done_(true, result);
          }));
    }));
  }

  ProbeUpstreamMessageLimitTest desc_;
  Test::Stub *server_stub_;
  CompletionQueue *cq_;
  std::function<void(bool, const TestResult &)> done_;
  ClientContext stream_ctx_;
  EchoResponse stream_response_;
  std::unique_ptr<ClientAsyncReaderWriter<EchoRequest, EchoResponse>>
      stream_rpc_;
  int upstream_message_limit_ = 0;
  bool saw_timeout_ = false;
  bool writing_ = false;
};

template <class T>
void RunSync(const TestPlans &plans, const typename T::TestDesc &desc,
             TestResult *result) {
  // Allocate test context.
  std::unique_ptr<Test::Stub> server_stub, direct_stub;
  if (plans.server_addr() != "") {
    server_stub = GetStub(plans.server_addr(), desc);
  }
  if (plans.direct_addr() != "") {
    direct_stub = GetStub(plans.direct_addr(), desc);
  }
  CompletionQueue cq;

  auto start_test = [&]() {
    T::Start(desc, server_stub.get(), direct_stub.get(), &cq,
             [&cq, &result](bool success, const TestResult &test_result) {
               *result = test_result;
               cq.Shutdown();
             });
  };

  if (plans.has_warmup()) {
    Echo::Start(plans.warmup(), server_stub.get(), direct_stub.get(), &cq,
                [&start_test](bool success, const TestResult &echo_result) {
                  // Start the actual test.
                  start_test();
                });
  } else {
    // No warmup needed.
    start_test();
  }

  // Run the queue until the test is complete.
  void *tag_pointer;
  bool ok;
  while (cq.Next(&tag_pointer, &ok)) {
    Tag *tag = reinterpret_cast<Tag *>(tag_pointer);
    (*tag)(ok);
    delete tag;
  }
}

}  // namespace

void RunTestPlans(const TestPlans &plans, TestResults *results) {
  for (const TestPlan &plan : plans.plans()) {
    TestResult *result = results->add_results();

    switch (plan.plan_case()) {
      case TestPlan::kEcho:
        RunSync<Echo>(plans, plan.echo(), result);
        break;
      case TestPlan::kEchoStream:
        RunSync<EchoStream>(plans, plan.echo_stream(), result);
        break;
      case TestPlan::kParallel:
        RunSync<Parallel>(plans, plan.parallel(), result);
        break;
      case TestPlan::kProbeCallLimit:
        RunSync<ProbeCallLimit>(plans, plan.probe_call_limit(), result);
        break;
      case TestPlan::kProbeDownstreamMessageLimit:
        RunSync<ProbeDownstreamMessageLimit>(
            plans, plan.probe_downstream_message_limit(), result);
        break;
      case TestPlan::kProbeUpstreamMessageLimit:
        RunSync<ProbeUpstreamMessageLimit>(
            plans, plan.probe_upstream_message_limit(), result);
        break;
      case TestPlan::kEchoReport:
        RunSync<EchoReport>(plans, plan.echo_report(), result);
        break;
      default:
        std::cerr << "Internal error: unimplemented test plan" << std::endl;
        std::abort();
    }
  }
}

}  // namespace grpc
}  // namespace test
