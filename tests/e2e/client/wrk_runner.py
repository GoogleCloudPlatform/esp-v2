#!/usr/bin/env python

# Copyright 2019 Google LLC

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

import utils
import json
import os
import sys

from string import Template

WRK_PATH = os.environ.get('WRK_PATH', '/usr/local/bin/wrk')
SCRIPT_DIR = os.path.dirname(__file__)


def test(run, n, c, t, d):
    """Run a test and extract its results.
    Args:
        run: is a dict {
           'url': a string
           'headers': [headers]
           'post_file': a string
           }
        n: number of requests (ignored by wrk)
        c: number of connections
        t: number of threads
        d: test duration in seconds
    Returns:
        result: a list of all kinds of result, including Complete requests, failed requests and so on
        metrics: a list of (Latency percentile: (value, unit))
        errors: a list of non-200 responses
    """
    cmd = [WRK_PATH,
           '-t', str(t),
           '--timeout', '2m',
           '-c', str(c),
           '-d', str(d) + 's',
           '-s', os.path.join(SCRIPT_DIR, "wrk_script.lua"),
           '-H', '"Content-Type:application/json"']

    if 'headers' in run:
        for h in run['headers']:
            cmd += ['-H', h]

    if 'post_file' in run:
        wrk_method = "POST"
        wrk_body_file = run['post_file']
    else:
        wrk_method = "GET"
        wrk_body_file = "/dev/null"

    wrk_out = 'wrk_out'
    wrk_err = 'wrk_err'
    with open(os.path.join(SCRIPT_DIR, "wrk_script.lua.temp"), 'r') as f:
        wrk_script = f.read()

    expected_status = run.get('expected_status', '200')
    with open(os.path.join(SCRIPT_DIR, 'wrk_script.lua'), 'w') as f:
        f.write(Template(wrk_script).substitute(
            HTTP_METHOD=wrk_method,
            REQUEST_BODY_FILE=wrk_body_file,
            EXPECTED_STATUS=expected_status,
            OUT=os.path.join(SCRIPT_DIR, wrk_out),
            ERR=os.path.join(SCRIPT_DIR, wrk_err)))

    cmd += [run['url']]

    (_, ret) = utils.IssueCommand(cmd)

    if ret != 0:
        print '==== Failed to run=%s,t=%d,c=%s,ret=%d' % (str(run), t, c, ret)
        sys.exit('Test failed')

    with open(os.path.join(SCRIPT_DIR, wrk_out), 'r') as f:
        records = json.load(f)

    metricOrder = ["Latency percentile: {}".format(i) for i in
                   ["50%", "66%", "75%", "80%", "90%", "95%", "98%", "99%",
                    "100%", "mean"]]
    resultOrder = ["Complete requests", "Failed requests",
                   "Failed requests by read", "Failed requests by write",
                   "Failed requests by timeout",
                   "Non-2xx responses", "Requests per second", "Transfer rate"]

    metrics = [(k, records[k]) for k in metricOrder]
    result = [(k, records[k]) for k in resultOrder]

    errors = []
    for i in range(0, t):
        with open(os.path.join(SCRIPT_DIR, wrk_err + '_' + str(i)), 'r') as f:
            errors.extend(f.readlines())

    return result, metrics, errors
