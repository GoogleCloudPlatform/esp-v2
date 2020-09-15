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

import argparse
import utils
import httplib
import json
import ssl
import sys
import os
import time
from utils import ApiProxyClientTest

class C:
    pass
FLAGS = C

class ApiProxyBookstoreTest(ApiProxyClientTest):
    """End to end integration test of bookstore application with deployed API
    PROXY at VM. It will call bookstore API according its Swagger spec to check
    1) IP address restriction
    2) iOS application restriction
    3) Android application restriction
    4) http referrer restrictions
    """

    def __init__(self):
        ApiProxyClientTest.__init__(self, FLAGS.host, FLAGS.host_header,
                               FLAGS.allow_unverified_cert,
                               FLAGS.verbose)

    def verify_key_restriction(self):
        # ignore test if required informations are not provided
        if FLAGS.key_restriction_tests == None or \
          FLAGS.key_restriction_keys_file == None:
            return

        # check file exists
        if os.path.exists(FLAGS.key_restriction_tests) == False:
            print ("API keys restriction tests template not exist.")
            sys.exit(1)
        if os.path.exists(FLAGS.key_restriction_keys_file) == False:
            print ("API keys restriction key file not exist. ")
            sys.exit(1)

        # load api keys
        with open(FLAGS.key_restriction_keys_file) as data_file:
            api_keys = json.load(data_file)

        with open(FLAGS.key_restriction_tests) as data_file:
            # Load template and render
            data_text = data_file.read();
            data_text = data_text.replace('${api_key_ip}', api_keys['ip']);
            data_text = data_text.replace('${api_key_ios}', api_keys['ios']);
            data_text = data_text.replace('${api_key_android}',
                                          api_keys['android']);
            data_text = data_text.replace('${api_key_referrers}',
                                          api_keys['referrers']);
            data = json.loads(data_text)

            # run test cases
            for type, testcases in data.iteritems():
                for testcase in testcases:
                    print testcase['description']
                    response = self._call_http(
                        testcase['path'],
                        api_key=testcase['api_key'],
                        userHeaders=testcase['headers'])
                    self.assertEqual(response.status_code,
                        testcase['status_code'])

    def run_all_tests(self):
        self.verify_key_restriction();

        if self._failed_tests:
            sys.exit(utils.red('%d tests passed, %d tests failed.' % (
                self._passed_tests, self._failed_tests)))
        else:
            print utils.green('All %d tests passed' % self._passed_tests)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--verbose', type=bool, help='Turn on/off verbosity.')
    parser.add_argument('--host', help='Deployed application host name.')
    parser.add_argument('--host_header', help='Deployed application host name.')
    parser.add_argument('--allow_unverified_cert', type=bool,
            default=False, help='used for testing self-signed ssl cert.')
    parser.add_argument('--key_restriction_tests',
                        help='Test suites for api key restriction.')
    parser.add_argument('--key_restriction_keys_file',
                        help='File contains API keys with restrictions ')
    flags = parser.parse_args(namespace=FLAGS)

    apiproxy_test = ApiProxyBookstoreTest()
    try:
        apiproxy_test.run_all_tests()
    except KeyError as e:
        sys.exit(utils.red('Test failed.'))