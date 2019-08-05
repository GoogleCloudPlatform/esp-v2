import unittest
import sys

import os, inspect

currentdir = os.path.dirname(
    os.path.abspath(inspect.getfile(inspect.currentframe())))
sys.path.append(currentdir + "/../../docker/generic")
from start_proxy import gen_bootstrap_conf, make_argparser


class TestStartProxy(unittest.TestCase):

    def setUp(self):
        self.parser = make_argparser()

    def test_gen_bootstrap(self):
        testcases = [
            (["--http_request_timeout=1m"],
             ['apiproxy/bootstrap', '--http_request_timeout', '1m',
              '/tmp/bootstrap.json']),

            (["--enable_tracing"], ['apiproxy/bootstrap', '--enable_tracing',
                                    '--tracing_sample_rate', '0.001',
                                    "--http_request_timeout", "5s",
                                    '/tmp/bootstrap.json']),

            (["--enable_tracing", "--tracing_project_id=123",
              "--tracing_sample_rate=1",
              "--tracing_incoming_context=fake-incoming-context",
              "--tracing_outgoing_context=fake-outgoing-context"],
             ['apiproxy/bootstrap', '--enable_tracing', '--tracing_project_id',
              "123",
              '--tracing_sample_rate', '1', "--tracing_incoming_context",
              "fake-incoming-context", "--tracing_outgoing_context",
              "fake-outgoing-context", "--http_request_timeout", "5s",
              '/tmp/bootstrap.json'])
        ]

        for flags, wantedArgs in testcases:
            gotArgs = gen_bootstrap_conf(self.parser.parse_args(flags))
            self.assertEqual(gotArgs, wantedArgs)


if __name__ == '__main__':
    unittest.main()
