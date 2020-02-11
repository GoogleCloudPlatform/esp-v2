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

import gflags as flags
import json
import sys

FLAGS = flags.FLAGS

flags.DEFINE_string('server', 'localhost:8001', 'grpc server address')

flags.DEFINE_string('api_key', '', 'api_key for the project')

flags.DEFINE_string('auth_token', '', 'JWT token for auth')

flags.DEFINE_string('payload_string', 'Hollow World!', 'request payload string')

# This flag overrides "--payload_string" flag.
flags.DEFINE_string('payload_file', '',
                    'the file name to read the request payload')
# If larger than 0, randomly generates request payload.
# The payload size is random between 0 and value of this flag.
# This flag overrides all other payload related flags.
flags.DEFINE_integer('random_payload_max_size', 0,
                     'randomly generate request payload up to this size')

flags.DEFINE_integer('request_count', 10000,
                     'total number of requests to send')

flags.DEFINE_integer('concurrent', 10, 'The concurrent requests to send')

flags.DEFINE_float('allowed_failure_rate', 0.001, 'Allowed failure rate.')

flags.DEFINE_integer('requests_per_stream', 100,
                     'The number of requests for each stream')

flags.DEFINE_boolean('use_ssl', False, 'If true, use SSL to connect.')

# kNoApiKeyError = ('Method doesn't allow unregistered callers (callers without'
#                   ' established identity). Please use API Key or other form of'
#                   ' API consumer identity to call this API.')

kNoApiKeyError = (
    'UNAUTHENTICATED:Method doesn\'t allow unregistered callers '
    '(callers without established identity). Please use API Key or'
    ' other form of API consumer identity to call this API.')


def GetRequest():
    if FLAGS.random_payload_max_size:
        return {
            'random_payload_max_size': FLAGS.random_payload_max_size
        }
    elif FLAGS.payload_file:
        return {
            'text': open(FLAGS.payload_file, 'r').read()
        }
    else:
        return {
            'text': FLAGS.payload_string
        }


def SubtestEcho():
    return {
        'weight': 1,
        'echo': {
            'request': GetRequest(),
            'call_config': {
                'use_ssl': FLAGS.use_ssl,
            }
        }
    }


def SubtestEchoStream():
    return {
        'weight': 1,
        'echo_stream': {
            'request': GetRequest(),
            'count': FLAGS.requests_per_stream,
            'call_config': {
                'api_key': FLAGS.api_key,
                'auth_token': FLAGS.auth_token,
                'use_ssl': FLAGS.use_ssl,
            },
        }
    }


def SubtestEchoStreamAuthFail():
    return {
        'weight': 1,
        'echo_stream': {
            'request': GetRequest(),
            'count': FLAGS.requests_per_stream,
            # Requires auth token.
            'expected_status': {
                'code': 16,
                'details': "Jwt is missing",
            },
            'call_config': {
                'use_ssl': FLAGS.use_ssl,
            }
        }
    }


def SubtestEchoStreamNoApiKey():
    return {
        'weight': 1,
        'echo_stream': {
            'request': GetRequest(),
            'count': FLAGS.requests_per_stream,
            # Even auth check passed, it still requires api-key
            'call_config': {
                'auth_token': FLAGS.auth_token,
                'use_ssl': FLAGS.use_ssl,
            },
            'expected_status': {
                'code': 16,
                'details': kNoApiKeyError,
            },
        }
    }


if __name__ == "__main__":
    try:
        argv = FLAGS(sys.argv)  # parse flags
    except flags.FlagsError as e:
        sys.exit('%s\nUsage: %s ARGS\n%s' % (e, sys.argv[0], FLAGS))

    subtests = [
        SubtestEcho(),
    ]

    # TODO: When Cloud Run supports streaming RPCs, disable this check.
    if not FLAGS.use_ssl:
        subtests += [
            SubtestEchoStream(),
            SubtestEchoStreamAuthFail(),
            SubtestEchoStreamNoApiKey(),
        ]

    print json.dumps({
        'server_addr': FLAGS.server,
        'plans': [{
            'parallel': {
                'test_count': FLAGS.request_count,
                'parallel_limit': FLAGS.concurrent,
                'allowed_failure_rate': FLAGS.allowed_failure_rate,
                'subtests': subtests,
            },
        }]
    }, indent=4)
