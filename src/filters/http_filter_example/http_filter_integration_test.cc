#include "test/integration/http_integration.h"

namespace Envoy {
class HttpFilterSampleIntegrationTest
    : public HttpIntegrationTest,
      public testing::TestWithParam<Network::Address::IpVersion> {
 public:
  HttpFilterSampleIntegrationTest()
      : HttpIntegrationTest(Http::CodecClient::Type::HTTP1, GetParam(), realTime()) {}
  /**
   * Initializer for an individual integration test.
   */
  void SetUp() override { initialize(); }

  void initialize() override {
    config_helper_.addFilter(
        "{ name: sample, config: { key: via, val: sample-filter } }");
    HttpIntegrationTest::initialize();
  }
};

INSTANTIATE_TEST_CASE_P(
    IpVersions, HttpFilterSampleIntegrationTest,
    testing::ValuesIn(TestEnvironment::getIpVersionsForTest()));

TEST_P(HttpFilterSampleIntegrationTest, Test1) {
  Http::TestHeaderMapImpl headers{
      {":method", "GET"}, {":path", "/"}, {":authority", "host"}};

  IntegrationCodecClientPtr codec_client;
  FakeHttpConnectionPtr fake_upstream_connection;
  FakeStreamPtr request_stream;

  codec_client = makeHttpConnection(lookupPort("http"));
  auto response = codec_client->makeHeaderOnlyRequest(headers);
  auto result = fake_upstreams_[0]->waitForHttpConnection(
      *dispatcher_, fake_upstream_connection, std::chrono::milliseconds(5));
  RELEASE_ASSERT(result, result.message());
  result =
      fake_upstream_connection->waitForNewStream(*dispatcher_, request_stream);
  RELEASE_ASSERT(result, result.message());
  result = request_stream->waitForEndStream(*dispatcher_);
  RELEASE_ASSERT(result, result.message());
  response->waitForEndStream();

  EXPECT_STREQ("sample-filter", request_stream->headers()
                                    .get(Http::LowerCaseString("via"))
                                    ->value()
                                    .c_str());

  codec_client->close();
}
}  // namespace Envoy