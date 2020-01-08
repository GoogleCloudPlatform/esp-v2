#!/usr/bin/python -u
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
import httplib
import json
import sys
import utils


class C:
  pass


FLAGS = C


class Fuzzer(object):
  """ A fuzzer based on templates and tokens. Each template is a string with
      one or more placeholders ('FUZZ'). The placeholders are replaced with tokens
      from the given token set to generate fuzz inputs.
  """

  def __init__(self, templates, tokens):
    self._templates = templates
    self._tokens = tokens

  def _run(self, template, f):
    if 'FUZZ' in template:
      for token in self._tokens:
        # Replace the first placeholder with the current token and call _run() again
        self._run(template.replace('FUZZ', token, 1), f)
    else:
      # All placeholders are already replaced, call the function f()
      f(template)

  def run(self, f):
    """ Executes the given function f() on all the generated fuzz inputs. """
    for template in self._templates:
      self._run(template, f)


class JsonFuzzer(Fuzzer):
  """ JSON fuzzer that uses static and generated templates """

  def __init__(self, max_object_nest_level, max_list_nest_level):
    STATIC_JSON_TEMPLATES = ['', 'FUZZ', '{FUZZ}', '[FUZZ]', '{FUZZ: FUZZ}',
                             '[FUZZ, FUZZ]']
    JSON_TOKEN_LIST = ['{', '}', '[', ']', '"message"', '"text"', 'true',
                       'false', '1234',
                       '-1234', '1.2345e-10', '-3.1415e+13', 'null']

    templates = STATIC_JSON_TEMPLATES
    for n in range(max_object_nest_level):
      templates += [self._get_object_template(n)]
    for n in range(max_list_nest_level):
      templates += [self._get_list_template(n)]

    Fuzzer.__init__(self, templates, JSON_TOKEN_LIST)

  def _get_object_template(self, nest_level):
    template = ""
    for i in range(nest_level):
      template += '{ "message": FUZZ'
      if i != nest_level - 1:
        template += ', '
    for i in range(nest_level):
      template += '}'
    return template

  def _get_list_template(self, nest_level):
    template = ""
    for i in range(nest_level):
      template += '[FUZZ'
      if i != nest_level - 1:
        template += ', '
    for i in range(nest_level):
      template += ']'
    return template


def url_path_fuzzer():
  """ URL Path fuzzer that uses static templates """
  URLPATH_TEMPLATES = ['/FUZZ', '/FUZZ/FUZZ', '/FUZZ/FUZZ/FUZZ']
  URLPATH_TOKEN_LIST = ['', '/', '@$%^& ', '%20%25', 'echo', 'echostream']
  return Fuzzer(URLPATH_TEMPLATES, URLPATH_TOKEN_LIST)


def query_param_fuzzer():
  """ URL Path fuzzer that uses static templates """
  QUERYPARAM_TEMPLATES = ['', '#', '#FUZZ', 'FUZZ', 'FUZZ=FUZZ', 'FUZZ&FUZZ']
  QUERYPARAM_TOKEN_LIST = ['', '/', '#', '=', '&', '?', '&?#', '@#$%^=',
                           'message',
                           'message=msg', '%2F%25%0A%20']
  return Fuzzer(QUERYPARAM_TEMPLATES, QUERYPARAM_TOKEN_LIST)


class ApiProxyTranscodingFuzzTest(object):
  """ ESPv2 Transcoding Fuzz tests """

  def __init__(self):
    self._conn = utils.http_connection(FLAGS.address, True)
    self._status_conn = utils.http_connection(FLAGS.status_address, True)
    self._initial_status = self._get_status()
    self._total_requests = 0
    self._unexpected_errors = 0

  def _get_status(self):
    self._status_conn.request('GET', '/ready')
    status = utils.Response(self._status_conn.getresponse())
    if status.status_code != 200:
      sys.exit(utils.red(
          'ESPv2 crash'))
    return status.text.strip()

  def _check_for_crash(self):
    status = self._get_status()
    if status != "LIVE":
      print(status)
      sys.exit(utils.red(
          'ESPv2 crash'))
      return
    print utils.green('No crashes detected.')

  def _request(self, url_path, query_params, json, expected_codes,
      json_response):
    # Construct the URL using url_path, query_params and the api key
    url = url_path
    if FLAGS.api_key:
      if query_params:
        url = '%s?key=%s&%s' % (url_path, FLAGS.api_key, query_params)
      else:
        url = '%s?key=%s' % (url_path, FLAGS.api_key)
    elif query_params:
      url = '%s?%s' % (url_path, query_params)

    # Prepare Headers
    headers = {'Content-Type': 'application/json'}
    if FLAGS.auth_token:
      headers['Authorization'] = 'Bearer ' + FLAGS.auth_token

    self._conn.request('POST', url, json, headers)
    response = utils.Response(self._conn.getresponse())

    if not response.status_code in expected_codes:
      print(utils.red(
          "Invalid status code {}:\n\turl={},\n\trequest_header={},\n\trequest_body={},\n\tresponse={}\n\n").format(
          response.status_code, url, headers, json, response))
      self._unexpected_errors += 1

    if json_response and not (
        response.status_code != 200 and response.content_type == "text/plain") \
        and not response.is_json():
      print(utils.red(
          "response is not json {}:\n\turl={},\n\trequest_header={},\n\trequest_body={},\n\tresponse={}\n\n").format(
          response.content_type, url, headers, json, response))
      self._unexpected_errors += 1

    self._total_requests += 1

  def _print_results_so_far(self):
    print 'Fuzz test results so far: total - %d, unexpected errors - %d' % (
        self._total_requests, self._unexpected_errors)

  def _run_json_fuzz_tests(self, url, max_object_nest_level,
      max_list_nest_level):
    fuzzer = JsonFuzzer(max_object_nest_level, max_list_nest_level)
    # For requests not dict-type json payload(like, number, string or list),
    # ESPv2 will return 500
    fuzzer.run(
        lambda json: self._request(url, None, json, [200, 400, 500], True))

  def _run_query_param_fuzzer(self, url_path):
    fuzzer = query_param_fuzzer()
    # For requests not matched with trancoding rules, ESPv2 will return 503
    fuzzer.run(
        lambda query_params: self._request(url_path, query_params, "{}",
                                           [200, 503], True))

  def _run_url_path_fuzzer(self):
    fuzzer = url_path_fuzzer()
    # Url path fuzzer generated requests may return 404 Not Found in
    # addition to 200 OK and 400 Bad Request
    fuzzer.run(lambda url_path: self._request(url_path, None, "{}",
                                              [200, 400, 404], False))

  def _run_fuzz_tests(self):
    print 'Running /echo JSON fuzz tests...'
    self._run_json_fuzz_tests('/echo', 4, 2)
    self._check_for_crash()
    self._print_results_so_far()

    print 'Running /echostream JSON fuzz tests...'
    self._run_json_fuzz_tests('/echostream', 2, 4)
    self._check_for_crash()
    self._print_results_so_far()

    print 'Running /echo query param fuzz tests...'
    self._run_query_param_fuzzer('/echo')
    self._check_for_crash()
    self._print_results_so_far()

    print 'Running URL path fuzz tests...'
    self._run_url_path_fuzzer()
    self._check_for_crash()
    self._print_results_so_far()

  def run_all_tests(self):
    for _ in range(FLAGS.runs):
      self._run_fuzz_tests()
    if self._unexpected_errors > 0:
      sys.exit(utils.red('Fuzz test failed.'))
    else:
      print utils.green('Fuzz test passed.')


if __name__ == '__main__':
  parser = argparse.ArgumentParser()
  parser.add_argument('--address', help='Deployed ApiProxy HTTP/1.1 address.')
  parser.add_argument('--status_address',
                      help='Address for getting ApiProxy status (/endpoints_status)')
  parser.add_argument('--api_key', help='Project api_key to access service.')
  parser.add_argument('--auth_token', help='Auth token.')
  parser.add_argument('--runs', type=int, default=1, help='Number of runs')
  flags = parser.parse_args(namespace=FLAGS)

  api_proxy_test = ApiProxyTranscodingFuzzTest()
  api_proxy_test.run_all_tests()
