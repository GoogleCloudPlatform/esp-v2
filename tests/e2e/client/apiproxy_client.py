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

"""
An APIPROXY test client to drive HTTP load.
"""

import wrk_runner
import h2load_runner
import gflags as flags
import json
import sys
import time

from string import Template
from collections import Counter

FLAGS = flags.FLAGS

# Test suites are dict of name to list of a test cases,
#
# Each test cases contains five fields:
#   Runner: test execution module
#   n: number of requests
#   c: number of connections to API PROXY
#   t: number of threads
#   d: test duration in seconds
TEST_SUITES = {
        'debug': [
                (wrk_runner, 0, 5, 2, 1)
                ],
        'negative': [
                (wrk_runner, 0, 5, 2, 10)
                ],
        'simple': [
                (wrk_runner, 0, 1, 1, 15),
                (wrk_runner, 0, 2, 1, 15),
                (wrk_runner, 0, 4, 1, 15),
                (wrk_runner, 0, 8, 1, 15)
                ],
        'stress': [
                (wrk_runner, 0, 5, 1, 60),
                (wrk_runner, 0, 10, 1, 60),
                (wrk_runner, 0, 10, 2, 60),
                (wrk_runner, 0, 20, 1, 60),
                ],
        '2m_stress': [
                (wrk_runner, 0, 1, 1, 120),
                (wrk_runner, 0, 5, 1, 120),
                (wrk_runner, 0, 10, 1, 120),
                (wrk_runner, 0, 10, 5, 120),
                (wrk_runner, 0, 20, 1, 120),
                (wrk_runner, 0, 20, 5, 120),
                ],
        'http2': [
                (h2load_runner, 1000, 1, 1, 0)
                ]
        }

flags.DEFINE_enum(
        'test', 'simple', TEST_SUITES.keys(),
        'test suit name')

flags.DEFINE_string('test_env', '',
        'JSON test description')

flags.DEFINE_string('test_data', 'test_data.json.temp',
        'Template for test data')

flags.DEFINE_string('host', 'localhost:8080',
        'Server location')

flags.DEFINE_string('api_key', '',
        'API key')

flags.DEFINE_string('auth_token', '',
        'Authentication token')

flags.DEFINE_string('post_file', '',
        'File for request body content')

def count_failed_requests(out):
    """ Count failed and non-2xx responses """
    failed = 0
    non2xx = 0
    completed = 0
    for metrics, _, _ in out:
        failed += metrics.get('Failed requests', [0])[0]
        non2xx += metrics.get('Non-2xx responses', [0])[0]
        completed += metrics.get('Complete requests', [0])[0]
    return failed, non2xx, completed

if __name__ == "__main__":
    try:
        argv = FLAGS(sys.argv)  # parse flags
    except flags.FlagsError as e:
        sys.exit('%s\nUsage: %s ARGS\n%s' % (e, sys.argv[0], FLAGS))

    test_env = {'test': FLAGS.test}
    if FLAGS.test_env:
        test_env.update(json.load(open(FLAGS.test_env, 'r')))

    if not FLAGS.test_data:
        sys.exit('Error: flag test_data is required')
    with open(FLAGS.test_data) as f:
        test_data = json.loads(Template(f.read()).substitute(
                HOST=FLAGS.host,
                API_KEY=FLAGS.api_key,
                JWT_TOKEN=FLAGS.auth_token,
                POST_FILE=FLAGS.post_file))

        print "=== Test data"
    print json.dumps(test_data)

    results = []
    for run in test_data['test_run']:
        for runner, n, c, t, d in TEST_SUITES[FLAGS.test]:
            ret = runner.test(run, n, c, t, d)
            if ret:
                metrics, errors = ret

                print '=== Metric:'
                # Add prefix for negative metrics to be filtered in result analysis.
                prefix = ''
                if FLAGS.test == 'negative':
                    prefix = '='
                for k in metrics.keys():
                    print '%s%s %s %s' % (prefix, k, metrics[k][0], metrics[k][1])

                print '=== Metadata:'
                metadata = {
                        'runner': runner.__name__,
                        'number': str(n),
                        'concurrent': str(c),
                        'threads': str(t),
                        'duration': str(d) + 's',
                        'time': time.time(),
                        }
                if 'labels' in run:
                    metadata.update(run['labels'])
                print json.dumps(metadata)

                if len(errors) > 0:
                    print '=== Error status responses:'
                    for error, count in Counter(errors).most_common():
                        print '= %06d: %s' % (count, error)

                results.append((metrics, metadata, errors))

    if not results:
        sys.exit('All load tests failed.')
    if FLAGS.test != 'negative':

      failed, non2xx, completed = count_failed_requests(results)
      if failed + non2xx > 0.005 * completed:
        sys.exit(
            ('Load test failed:\n'
             '  {} failed requests,\n'
             '  {} non-2xx responses.\n'
             '  {} completed response.').format(failed, non2xx, completed))

    print "All load tests are successful."