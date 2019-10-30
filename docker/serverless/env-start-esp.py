#!/usr/bin/env python
#
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

from __future__ import print_function
import os


def assert_env_var(name):
    if name not in os.environ:
        raise KeyError(
            "Serverless ESP expects {} in environment variables.".format(name)
        )


def make_error_app(error_msg):
    # error_msg must be a utf-8 or ascii bytestring
    def error_app(environ, start_response):
        start_response("503 Service Unavailable", [("Content-Type", "text/plain")])
        return [error_msg, "\n"]

    return error_app


def serve_error_msg(error_msg):
    print("Serving error handler with '{}'.".format(error_msg))
    import wsgiref.simple_server

    app = make_error_app(error_msg)
    port = int(os.environ["PORT"])
    server = wsgiref.simple_server.make_server("", port, app)
    server.serve_forever()


def main():
    CMD = "/usr/local/bin/python"
    ARGS = [CMD, "/apiproxy/start_proxy.py", "--enable_backend_routing"]

    # TODO(qiwzhang): b/142663789 to use flag --compute_platform_override
    # PLATFORM = "Cloud Run"
    # ARGS = [CMD, "--compute_platform_override='{}'".format(PLATFORM)]

    # Uncaught KeyError; if no port, we can't serve a nice error handler. Crash instead.
    assert_env_var("PORT")
    # TODO(qiwzhang): b/142664239 to use flag --listener_port
    # ARGS.append("--http_port={}".format(os.environ["PORT"]))

    try:
        assert_env_var("ENDPOINTS_SERVICE_NAME")
    except KeyError as error:
        serve_error_msg(str(error))
    ARGS.append("--service={}".format(os.environ["ENDPOINTS_SERVICE_NAME"]))

    if "ENDPOINTS_SERVICE_VERSION" in os.environ:
        ARGS.extend(
            [
                "--rollout_strategy=fixed",
                "--version={}".format(os.environ["ENDPOINTS_SERVICE_VERSION"]),
            ]
        )
    else:
        ARGS.append("--rollout_strategy=managed")

    if "CORS_PRESET" in os.environ:
        ARGS.append("--cors_preset={}".format(os.environ["CORS_PRESET"]))

    if "ESP_ARGS" in os.environ:
        # By default, ESP_ARGS is comma-separated.
        # But if a comma needs to appear within an arg, there is an alternative
        # syntax: Pick a replacement delimiter, specify it at the beginning of the
        # string between two caret (^) symbols, and use it within the arg string.
        # Example:
        # ^++^--cors_allow_methods="GET,POST,PUT,OPTIONS"++--cors_allow_credentials
        arg_value = os.environ["ESP_ARGS"]

        delim = ","
        if arg_value.startswith("^") and "^" in arg_value[1:]:
            delim, arg_value = arg_value[1:].split("^", 1)
        if not delim:
            serve_error_msg("Malformed ESP_ARGS environment variable.")

        ARGS.extend(arg_value.split(delim))

    os.execv(CMD, ARGS)


if __name__ == "__main__":
    main()
