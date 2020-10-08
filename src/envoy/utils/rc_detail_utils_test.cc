
#include "src/envoy/utils/rc_detail_utils.h"

#include "gtest/gtest.h"

namespace espv2 {
namespace envoy {
namespace utils {
namespace {

TEST(GenerateRcDetailTest, WithDetail) {
  EXPECT_EQ(generateRcDetails("filter_name", "error_type", "DETAIL"),
            "filter_name_error_type{DETAIL}");
}

TEST(GenerateRcDetailTest, WithoutDetail) {
  EXPECT_EQ(generateRcDetails("filter_name", "error_type"),
            "filter_name_error_type");
}

}  // namespace
}  // namespace utils
}  // namespace envoy
}  // namespace espv2
