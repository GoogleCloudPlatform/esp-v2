#!/bin/bash

set -e

[[ -z "${SRCDIR}" ]] && SRCDIR="${PWD}"

echo "Starting coverage/cpp_fuzz.sh..."
echo "    PWD=$(pwd)"
echo "    SRCDIR=${SRCDIR}"

echo $(bazel --version)
# This is the fuzz target that will be run to generate coverage data.
if [[ $# -gt 0 ]]; then
  TARGET=$*
else
  TARGET=//src/envoy/utils:json_struct_fuzz_test
fi

TARGET_PATH=${TARGET:2}
TARGET_PATH=${TARGET_PATH//://}
echo ${TARGET_PATH}

# Create a temp directory for the corpus the fuzzer will generate
CORPUS_DIR=$(mktemp -d)

# Get the original corpus directory for the fuzz target.
# Assumes all fuzz targets follow the same directory layouts.
ORIGINAL_CORPUS=$(bazel query "labels(srcs, ${TARGET}_corpus_tar)" | head -1)
ORIGINAL_CORPUS=${ORIGINAL_CORPUS/://}
ORIGINAL_CORPUS="${ORIGINAL_CORPUS%"_corpus"}/"
echo ${ORIGINAL_CORPUS}

echo "RUNNING FUZZER"
# Run the fuzzer for one minute:
bazel run --config=asan-fuzzer ${TARGET}_with_libfuzzer -- -max_total_time=60 $(pwd)${ORIGINAL_CORPUS:1}

SCRIPT_DIR="$(realpath "$(dirname "$0")")"
. "${SCRIPT_DIR}"/gen_coverage.sh