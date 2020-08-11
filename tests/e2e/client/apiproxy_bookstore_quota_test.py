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
import sys
import time
from utils import ApiProxyClientTest

class C:
  pass
FLAGS = C

class ApiProxyBookstoreTest(ApiProxyClientTest):
  """End to end integration test of bookstore application with deployed
  ESP at VM.  It will call bookstore API according its Swagger spec
  1) set quota limit to 30
  2) send traffic 60 qpm for 150 seconds and count response code 200
  3) check count between 45 to 135
  """

  def __init__(self):
    ApiProxyClientTest.__init__(self, FLAGS.host, FLAGS.host_header,
                           FLAGS.allow_unverified_cert,
                           FLAGS.verbose)

  def verify_quota_control(self):
    # turn off verbose log
    print("Turn off the verbose log flag...");
    verbose = FLAGS.verbose
    FLAGS.verbose = False
    self.set_verbose(FLAGS.verbose)


    def _exhaust_quota():
      for i in range(100):
        time.sleep(1)
        try:
          response = self._call_http(path='/quota_read',
                                     api_key=FLAGS.api_key)
        except Exception, e:
          print "Exception {0} occurred".format(e)
          continue
        if response.status_code == 429:
          break;
        elif i == 99:
          sys.exit(utils.red("Fail to exhaust quota"))

    # exhaust the quota in the current window.
    print("Exhaust current quota...")
    _exhaust_quota()

    time.sleep(5)

    # waiting for the next quota refill.
    print("Wait for the next quota refill...")
    _exhaust_quota()

    # start counting
    print("Sending requests to count response codes for 150 seconds...");
    code_200 = 0
    code_429 = 0
    code_else = 0

    # run tests for 150 seconds, two quota refills expected
    t_end = time.time() + 60 * 2 + 30
    count = 0;
    while time.time() < t_end:
      try:
        response = self._call_http(path='/quota_read',
                                   api_key=FLAGS.api_key)
      except Exception, e:
          print "Exception {0} occurred".format(e)
          continue

      if response.status_code == 429:
        code_429 += 1
      elif response.status_code == 200:
        code_200 += 1
      else:
        code_else += 1
      count += 1

      print({"lefr_sec": t_end - time.time(), "Code 200": code_200,"Code 429": code_429, "Code else": code_else})
      # delay 1 second after each request
      time.sleep(1);



    # 145 - 150 total requests.
    # code_200 should be between 45 to 135. Allow +- 50% margin.
    # code_else should be 0.
    # The rest is code 429
    print("checking the count of code 200")
    self.assertGE(code_200 , 45);
    self.assertLE(code_200 , 135);
    print("checking the count of code other than 200 and 429")
    self.assertEqual(code_else, 0);

    # restore verbose flag
    FLAGS.verbose = verbose
    self.set_verbose(FLAGS.verbose)

  def run_all_tests(self):

    self.verify_quota_control()

    if self._failed_tests:
      sys.exit(utils.red('%d tests passed, %d tests failed.' % (
          self._passed_tests, self._failed_tests)))
    else:
      print utils.green('All %d tests passed' % self._passed_tests)

if __name__ == '__main__':
  parser = argparse.ArgumentParser()
  parser.add_argument('--verbose', type=bool, help='Turn on/off verbosity.')
  parser.add_argument('--api_key', help='Project api_key to access service.')
  parser.add_argument('--host', help='Deployed application host name.')
  parser.add_argument('--host_header', help='Deployed application host name.')
  parser.add_argument('--auth_token', help='Auth token.')
  parser.add_argument('--allow_unverified_cert', type=bool,
                      default=False, help='used for testing self-signed ssl cert.')
  flags = parser.parse_args(namespace=FLAGS)

  apiproxy_test = ApiProxyBookstoreTest()
  try:
    apiproxy_test.run_all_tests()
  except KeyError as e:
    sys.exit(utils.red('Test failed.'))