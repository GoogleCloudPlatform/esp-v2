#!/bin/bash

set -e

# Using GTEST_SHUFFLE here to workaround https://github.com/envoyproxy/envoy/issues/10108
BAZEL_USE_LLVM_NATIVE_COVERAGE=1 GCOV=llvm-profdata CC=clang-8 CXX=clang++-8 \
    bazel coverage ${BAZEL_BUILD_OPTIONS} \
    -c fastbuild --copt=-DNDEBUG --instrumentation_filter="//src/..." \
    --test_timeout=2000 --cxxopt="-DENVOY_CONFIG_COVERAGE=1" --test_output=errors \
    --test_arg="--log-path /dev/null" --test_arg="-l trace" --test_env=HEAPCHECK= \
    --test_env=GTEST_SHUFFLE=1 --flaky_test_attempts=5 ${TARGET}

COVERAGE_DIR="${SRCDIR}"/generated/${TARGET_PATH}
mkdir -p "${COVERAGE_DIR}"

COVERAGE_IGNORE_REGEX="(/external/|pb\.(validate\.)?(h|cc)|/chromium_url/|/test/|/tmp|/source/extensions/quic_listeners/quiche/)"
COVERAGE_BINARY="bazel-bin/${TARGET_PATH}"
COVERAGE_DATA="${COVERAGE_DIR}/coverage.dat"

echo "Merging coverage data..."
llvm-profdata merge -sparse -o ${COVERAGE_DATA} $(find -L bazel-out/k8-fastbuild/testlogs/${TARGET_PATH} -name coverage.dat)

echo "Generating report..."
llvm-cov show "${COVERAGE_BINARY}" -instr-profile="${COVERAGE_DATA}" -Xdemangler=c++filt \
  -ignore-filename-regex="${COVERAGE_IGNORE_REGEX}" -output-dir=${COVERAGE_DIR} -format=html
sed -i -e 's|>proc/self/cwd/|>|g' "${COVERAGE_DIR}/index.html"
sed -i -e 's|>bazel-out/[^/]*/bin/\([^/]*\)/[^<]*/_virtual_includes/[^/]*|>\1|g' "${COVERAGE_DIR}/index.html"

COVERAGE_VALUE=$(llvm-cov export "${COVERAGE_BINARY}" -instr-profile="${COVERAGE_DATA}" \
    -ignore-filename-regex="${COVERAGE_IGNORE_REGEX}" -summary-only | \
    python3 -c "import sys, json; print(json.load(sys.stdin)['data'][0]['totals']['lines']['percent'])")
echo "Covered lines percentage: ${COVERAGE_VALUE}"

echo "HTML coverage report is in ${COVERAGE_DIR}/index.html"