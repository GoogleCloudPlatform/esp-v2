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

import httplib
import json
import ssl
import subprocess


def IssueCommand(cmd, force_info_log=False, suppress_warning=False,
    env=None):
    """Tries running the provided command once.
    Args:
      cmd: A list of strings such as is given to the subprocess.Popen()
          constructor.
      env: A dict of key/value strings, such as is given to the subprocess.Popen()
          constructor, that contains environment variables to be injected.
    Returns:
      A tuple of stdout, and retcode from running the provided command.
    """
    print '=== Running: %s' % ' '.join(cmd)
    process = subprocess.Popen(cmd, env=env, stdout=subprocess.PIPE)
    stdout = ''
    while True:
        output = process.stdout.readline()
        if output == '' and process.poll() is not None:
            break
        if output:
            stdout += output
            print '= ' + output.strip()
    rc = process.poll()
    print '=== Finished with code %d' % rc
    return stdout, rc


COLOR_RED = '\033[91m'
COLOR_GREEN = '\033[92m'
COLOR_END = '\033[0m'

HTTPS_PREFIX = 'https://'
HTTP_PREFIX = 'http://'


def green(text):
    return COLOR_GREEN + text + COLOR_END


def red(text):
    return COLOR_RED + text + COLOR_END


def http_connection(host, allow_unverified_cert):
    if host.startswith(HTTPS_PREFIX):
        print('Create HTTPS connection')
        host = host[len(HTTPS_PREFIX):]

        ssl_ctx = ssl.create_default_context()
        if allow_unverified_cert:
            print('Certs NOT verified for this connection')
            ssl_ctx.check_hostname = False
            ssl_ctx.verify_mode = ssl.CERT_NONE
        else:
            print('Certs will be verified for this connection')
            ssl_ctx.check_hostname = True
            ssl_ctx.verify_mode = ssl.CERT_REQUIRED
        return httplib.HTTPSConnection(host, timeout=10, context=ssl_ctx)

    else:
        print('Create HTTP connection')
        if host.startswith(HTTP_PREFIX):
            host = host[len(HTTP_PREFIX):]
        else:
            host = host
        return httplib.HTTPConnection(host)


class Response(object):
    """A class to wrap around httplib.response class."""

    def __init__(self, r):
        self.text = r.read()
        self.status_code = r.status
        self.headers = r.getheaders()
        self.content_type = r.getheader('content-type')
        if self.content_type != None:
            self.content_type = self.content_type.lower()

    def json(self):
        try:
            return json.loads(self.text)
        except ValueError as e:
            print 'Error: failed in JSON decode: %s' % self.text
            return {}

    def is_json(self):
        if self.content_type != 'application/json':
            return False
        try:
            json.loads(self.text)
            return True
        except ValueError as e:
            return False

    def __str__(self):
        return "status_code: {}, text: {}, headers: {}".format(self.status_code,
                                                               self.text,
                                                               self.headers)


class ApiProxyClientTest(object):
    def __init__(self, host, host_header, allow_unverified_cert, verbose=False):
        self._failed_tests = 0
        self._passed_tests = 0
        self._verbose = verbose
        self.conn = http_connection(host, allow_unverified_cert)
        self.host_header = host_header

    def fail(self, msg):
        print '%s: %s' % (red('FAILED'), msg if msg else '')
        self._failed_tests += 1

    def assertEqual(self, got, want):
        msg = 'assertEqual(got=%s, want=%s)' % (str(got), str(want))
        if got == want:
            print '%s: %s' % (green('OK'), msg)
            self._passed_tests += 1
        else:
            self.fail(msg)

    def assertGE(self, a, b):
        msg = 'assertGE(%s, %s)' % (str(a), str(b))
        if a >= b:
            print '%s: %s' % (green('OK'), msg)
            self._passed_tests += 1
        else:
            self.fail(msg)

    def assertLE(self, a, b):
        msg = 'assertLE(%s, %s)' % (str(a), str(b))
        if a <= b:
            print '%s: %s' % (green('OK'), msg)
            self._passed_tests += 1
        else:
            self.fail(msg)

    def _call_http(self, path, api_key=None, auth=None, data=None, method=None,
        userHeaders={}):
        """Makes a http call and returns its response."""
        url = path
        if api_key:
            url += '?key=' + api_key
        headers = {'Content-Type': 'application/json'}
        if auth:
            headers['Authorization'] = 'Bearer ' + auth
        if self.host_header:
            headers["Host"] = self.host_header

        body = json.dumps(data) if data else None
        for key, value in userHeaders.iteritems():
            headers[key] = value
        if not method:
            method = 'POST' if data else 'GET'
        if self._verbose:
            print 'HTTP: %s %s' % (method, url)
            print 'headers: %s' % str(headers)
            print 'body: %s' % body
        self.conn.request(method, url, body, headers)
        response = Response(self.conn.getresponse())
        print response.status_code

        if self._verbose:
            print 'Status: %s, body=%s' % (response.status_code, response.text)
        return response

    def set_verbose(self, verbose):
        self._verbose = verbose
