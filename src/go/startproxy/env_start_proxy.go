// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package startproxy

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// MissingServiceConfigError is the help text served when ENDPOINTS_SERVICE_PATH and ENDPOINTS_SERVICE_NAME are both absent.
const MissingServiceConfigError = `
 Did you forget to build the Endpoints service configuration
 into the ESPv2 image? Please refer to the official serverless
 quickstart tutorials (below) for more information.
 
 https://cloud.google.com/endpoints/docs/openapi/get-started-cloud-run#configure_esp
 https://cloud.google.com/endpoints/docs/openapi/get-started-cloud-functions#configure_esp
 https://cloud.google.com/endpoints/docs/grpc/get-started-cloud-run#configure_esp
 
 If you are following along with these tutorials but have not
 reached the step above yet, this error is expected. Feel free
 to temporarily disregard this error message.
 
 If you wish to skip this step, please specify the name of the
 service in the ENDPOINTS_SERVICE_NAME environment variable.
 Note this deployment mode is **not** officially supported.
 It is recommended that you follow the tutorials linked above.
`

// MalformedESPv2ArgsError is the help text served when ESPv2_ARGS has an invalid custom delimiter.
const MalformedESPv2ArgsError = `
 Malformed ESPv2_ARGS environment variable.
 
 Please refer to the official ESPv2 startup reference
 (below) for information on how to format ESPv2_ARGS.
 
 https://cloud.google.com/endpoints/docs/openapi/specify-esp-v2-startup-options#setting-configuration-flags
`

// BuildServerlessFlags translates serverless environment variables (PORT,
// ENDPOINTS_SERVICE_PATH, ENDPOINTS_SERVICE_NAME, ENDPOINTS_SERVICE_VERSION, ESPv2_ARGS)
// into startproxy command-line flags.
func BuildServerlessFlags() ([]string, string, bool, error) {
	port, hasPort := os.LookupEnv("PORT")
	if !hasPort {
		return nil, "", false, fmt.Errorf("Serverless ESPv2 expects PORT in environment variables.\n")
	}

	flags := []string{
		"--on_serverless",
		fmt.Sprintf("--http_port=%s", port),
	}

	if servicePath, ok := os.LookupEnv("ENDPOINTS_SERVICE_PATH"); ok {
		flags = append(flags,
			"--rollout_strategy=fixed",
			fmt.Sprintf("--service_json_path=%s", servicePath),
		)
	} else {
		serviceName, hasServiceName := os.LookupEnv("ENDPOINTS_SERVICE_NAME")
		if !hasServiceName {
			return nil, port, true, fmt.Errorf("Serverless ESPv2 expects ENDPOINTS_SERVICE_NAME in environment variables.\n%s", MissingServiceConfigError)
		}
		flags = append(flags, fmt.Sprintf("--service=%s", serviceName))

		if serviceVersion, hasVersion := os.LookupEnv("ENDPOINTS_SERVICE_VERSION"); hasVersion {
			flags = append(flags,
				"--rollout_strategy=fixed",
				fmt.Sprintf("--version=%s", serviceVersion),
			)
		} else {
			flags = append(flags, "--rollout_strategy=managed")
		}
	}

	if argValue, ok := os.LookupEnv("ESPv2_ARGS"); ok {
		delim := ","
		if strings.HasPrefix(argValue, "^") && strings.Contains(argValue[1:], "^") {
			parts := strings.SplitN(argValue[1:], "^", 2)
			delim = parts[0]
			argValue = parts[1]
		}
		if delim == "" {
			return nil, port, true, fmt.Errorf("%s", MalformedESPv2ArgsError)
		}
		flags = append(flags, strings.Split(argValue, delim)...)
	}

	return flags, port, false, nil
}

// GenServerlessArgs builds the full command argument slice matching env_start_proxy.py's gen_args(cmd).
func GenServerlessArgs(cmd string) ([]string, error) {
	flags, _, _, err := BuildServerlessFlags()
	if err != nil {
		return nil, err
	}
	res := make([]string, 0, len(flags)+2)
	res = append(res, cmd, "/apiproxy/start_proxy.py")
	res = append(res, flags...)
	return res, nil
}

// MakeErrorHandler returns an http.Handler that responds with 503 Service Unavailable and msg.
func MakeErrorHandler(msg string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(msg))
	})
}

// ServeErrorMsg starts an HTTP server on port responding with 503 Service Unavailable and msg.
func ServeErrorMsg(port string, msg string) error {
	return http.ListenAndServe(":"+port, MakeErrorHandler(msg))
}
