#!/bin/bash

set -e

[[ -z "${SRCDIR}" ]] && SRCDIR="${PWD}"

echo "Starting coverage/cpp_unit.sh..."
echo "    PWD=$(pwd)"
echo "    SRCDIR=${SRCDIR}"

# This is the target that will be run to generate coverage data. It can be overridden by consumer
# projects that want to run coverage on a different/combined target.
# Command-line arguments take precedence over ${COVERAGE_TARGET}.
if [[ $# -gt 0 ]]; then
  COVERAGE_TARGETS=$*
elif [[ -n "${COVERAGE_TARGET}" ]]; then
  COVERAGE_TARGETS=${COVERAGE_TARGET}
else
  COVERAGE_TARGETS=//src/...
fi

# Make sure //third_party/tools/coverage:coverage_tests is up-to-date.
SCRIPT_DIR="$(realpath "$(dirname "$0")")"
"${SCRIPT_DIR}"/gen_build.sh ${COVERAGE_TARGETS}

TARGET=//third_party/tools/coverage:coverage_tests
TARGET_PATH=${TARGET:2}
TARGET_PATH=${TARGET_PATH//://}
echo ${TARGET_PATH}

. "${SCRIPT_DIR}"/gen_coverage.sh