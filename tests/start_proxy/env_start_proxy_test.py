# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import unittest
import sys

import os, inspect

currentdir = os.path.dirname(
    os.path.abspath(inspect.getfile(inspect.currentframe())))
sys.path.append(currentdir + "/../../docker/serverless")

from env_start_proxy import gen_args


class TestStartProxy(unittest.TestCase):

    def test_main(self):
        testcases = [
          { "port": "8080",
            "service": "test_bookstore.goog.cloud",
            "args": "--http_request_timeout_s=1, --disable_tracing ",
            "wantArgs": [
              '/apiproxy/start_proxy.py',
              '/apiproxy/start_proxy.py',
              '--enable_backend_routing',
              "--compute_platform_override=Cloud Run(ESPv2)",
              '--http_port=8080',
              '--service=test_bookstore.goog.cloud',
              '--rollout_strategy=managed',
              '--http_request_timeout_s=1',
              ' --disable_tracing '
            ]
          },
          { "port": "8082",
            "service": "test_bookstore.goog.cloud",
            "version": "2019-02-21r0",
            "args": "^++^--cors_preset=basic,++--cors_allow_origin=*",
            "wantArgs": [
              '/apiproxy/start_proxy.py',
              '/apiproxy/start_proxy.py',
              '--enable_backend_routing',
              "--compute_platform_override=Cloud Run(ESPv2)",
              '--http_port=8082',
              '--service=test_bookstore.goog.cloud',
              '--rollout_strategy=fixed',
              '--version=2019-02-21r0',
              '--cors_preset=basic,',
              '--cors_allow_origin=*'
            ]
          },
          { "port": "8080",
            "servicePath": "/tmp/service_config.json",
            "args": "--disable_tracing",
            "wantArgs": [
              '/apiproxy/start_proxy.py',
              '/apiproxy/start_proxy.py',
              '--enable_backend_routing',
              "--compute_platform_override=Cloud Run(ESPv2)",
              '--http_port=8080',
              '--rollout_strategy=fixed',
              '--service_json_path=/tmp/service_config.json',
              '--disable_tracing'
            ]
          }
        ]

        for testcase in testcases:
            os.environ.clear()
            os.environ["PORT"] = testcase.get("port")
            if testcase.get("service"):
              os.environ["ENDPOINTS_SERVICE_NAME"] = testcase.get("service")
            if testcase.get("version"):
              os.environ["ENDPOINTS_SERVICE_VERSION"] = testcase.get("version")
            if testcase.get("servicePath"):
              os.environ["ENDPOINTS_SERVICE_PATH"] = testcase.get("servicePath")

            os.environ["ESPv2_ARGS"] = testcase.get("args")
            gotArgs = gen_args("/apiproxy/start_proxy.py")
            print(gotArgs)
            self.assertEqual(gotArgs, testcase["wantArgs"])


if __name__ == '__main__':
    unittest.main()
