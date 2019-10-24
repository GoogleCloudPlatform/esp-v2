#!/bin/bash

# Copyright 2018 Google Cloud Platform Proxy Authors

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
. ${ROOT}/scripts/all-utilities.sh || { echo "Cannot load Bash utilities";
  exit 1; }

function usage() {
  [[ -n "${1}" ]] && echo "${1}"
  echo "usage: ${BASH_SOURCE[0]}
  -g <path_to_gsutil>
  -s <commit sha>
  -t <output dir>"
  exit 1
}

GSUTIL=$(which gsutil) || GSUTIL=~/google-cloud-sdk/bin/gsutil
SHA=""
TMP=~/tmp

while getopts :g:s:t: arg; do
  case ${arg} in
    g) GSUTIL="${OPTARG}" ;;
    s) SHA="${OPTARG}" ;;
    t) TMP="${OPTARG}" ;;
    *) usage "Invalid option: -${OPTARG}" ;;
  esac
done

[[ -n "${SHA}" ]] || usage "Must provide commit sha via '-s' parameter."
[[ -x "${GSUTIL}" ]] || usage "Cannot find gsutil, provide it via '-g' flag."

mkdir -p "${TMP}"
LOGS="$(mktemp -d ${TMP}/qualify-XXXX)"
RESULT="${LOGS}/RESULT.log"
#TODO(taoxuy): add envoy log analysis
ENVOY_LOG_ANALYSIS="${LOGS}/envoy-log-analysis.log"

function check_result() {
  local FILE=${1}
  local COMMIT=${2}

  python - << EOF ${FILE} ${COMMIT}
import sys
import json

log_file = sys.argv[1]
commit_sha = sys.argv[2]

with open(log_file) as log:
  result = json.load(log)

json_status = result.get('scriptStatus', 1)
json_sha = result.get('headCommitHash', '')

status_ok = json_status == 0
sha_ok = json_sha == commit_sha

print "Checking {} SHA={}".format(log_file, commit_sha)
print "  Status == {} ({})".format(json_status, "OK" if status_ok else "FAIL")
print "  SHA == {} ({})".format(json_sha, "OK" if sha_ok else "FAIL")

exit(0 if status_ok and sha_ok else 1)
EOF
}

function count_stress_failures() {
  awk '
    BEGIN {
      failed=0
      complete=0
      non2xx=0
    }

    /Complete requests *[0-9]*/ {
      complete+=$3
    }

    /^Failed requests *[0-9]*/ {
      failed+=$3
    }

    /^Non-2xx responses *[0-9]*/ {
      non2xx+=$3
    }

    END {
      total = complete + failed
      print "Failed requests:   ", failed
      print "Non-2xx responses: ", non2xx
      print "Total requests:    ", total
      if (total > 0) {
        print "Failed/Total:      ", (failed + non2xx) / total
      }
    }' "${@}"
}

( echo "Release qualification of ${SHA}."
echo "It is now: $(date)"

mkdir -p "${LOGS}/${SHA}"

echo "Downloading prow logs to '${LOGS}' directory."
${GSUTIL} -m -q cp -r "gs://apiproxy-continuous-long-run/${SHA}/logs/*" "${LOGS}/${SHA}/" 2>&1  \
 || error_exit "Failed to download logs from endpoints-jenkins.appspot.com."

python "${ROOT}/scripts/release/validate_release.py"  \
 --commit_sha "${SHA}"  \
 --path "${LOGS}/${SHA}"  \
 || error_exit "Release is not qualified."

RQ_TESTS=( )

# This while loop reads from a redirect set up at the "done" clause.
# Because we read user input inside the loop, we set up the input
# coming from the "find" command on file descriptor 3. This is why
# we use "read -u3" here.

while read -u3 LOG_FILE; do
  DIR="$(dirname "${LOG_FILE}")"
  JSON_FILE="${LOG_FILE%.log}.json"
  RUN="${DIR##*/}"

  [[ -f "${JSON_FILE}" ]]  \
   || error_exit "Result of release qualification test ${JSON_FILE} not found."

  echo '*********************************************************************'
  echo "Release qualification run: ${RUN}"

  echo ''
  echo "Checking ${JSON_FILE}"
  echo ''
  check_result "${JSON_FILE}" "${SHA}" || continue

  echo ''
  echo "Checking ${LOG_FILE}"
  echo ''
  count_stress_failures "${LOG_FILE}"

  RQ_TESTS+=(${DIR})

# the ! -path ... excludes the root directory which is otherwise
# included in the result
done 3< <( find "${LOGS}/${SHA}" ! -path "${LOGS}/${SHA}" -type f -name 'long-run-test*.log' )

if [[ ${#RQ_TESTS[@]} -le 0 ]]; then
    echo '*********************************************************************'
    echo '* Release qualification INCOMPLETE.                                 *'
    echo '*                       **********                                  *'
    echo '*                                                                   *'
    echo '* No release qualification tests have been run yet.                 *'
    echo '*********************************************************************'

    ARGS=()
    for arg in "${@}"; do ARGS+=("\"${arg}\""); done
    echo "${BASH_SOURCE[0]}" "${ARGS[@]}"

    exit 0
  fi
#TODO(taoxuy):add envoy log check

  echo ''
  echo '*********************************************************************'
  echo '* Release qualification script completed.                           *'
  echo '*                                                                   *'
  echo '* Additional manual checks may be required.                         *'
  echo '*                                                                   *'
  echo '* Please review the results above and analyze any failed requests.  *'
  echo '* If there are failures, review the ENVOY error logs on the backend *'
  echo '* virtual machines to get more insights into the failures.          *'
  echo '* Update the release bug with any findings, and open bugs for any   *'
  echo '* issues found during the investigation.                            *'
  echo '*********************************************************************'

) | tee ${RESULT}

[[ ${PIPESTATUS[0]} -eq 0 && ${PIPESTATUS[1]} -eq 0 ]] \
  || error_exit "Release qualification failed. Results were saved in ${RESULT}."

echo "Results were saved in ${RESULT}"

printf '\e[31m
***************************************************************************
*      Please paste the script output verbatim into the release bug.      *
***************************************************************************
\e[0m'
