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
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
)

// serverlessModeArg, as the first argument, selects serverless mode: ESPv2 is
// configured by environment variables, like env_start_proxy.py did.
const serverlessModeArg = "--serverless"

// missingServiceConfigMsg is the help text served when ENDPOINTS_SERVICE_PATH and ENDPOINTS_SERVICE_NAME are both absent.
const missingServiceConfigMsg = `
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

// malformedESPv2ArgsMsg is the help text served when ESPv2_ARGS has an invalid custom delimiter.
const malformedESPv2ArgsMsg = `
 Malformed ESPv2_ARGS environment variable.
 
 Please refer to the official ESPv2 startup reference
 (below) for information on how to format ESPv2_ARGS.
 
 https://cloud.google.com/endpoints/docs/openapi/specify-esp-v2-startup-options#setting-configuration-flags
`

// serverlessError reports a misconfigured serverless environment.
type serverlessError struct {
	msg string
	// port, if set, is where msg is served as a 503 page instead of exiting,
	// so callers of the service can see what is wrong.
	port string
	// warning logs msg at WARNING rather than ERROR level.
	warning bool
}

func (e *serverlessError) Error() string { return e.msg }

// serverlessFlags translates the serverless environment variables (PORT,
// ENDPOINTS_SERVICE_PATH, ENDPOINTS_SERVICE_NAME, ENDPOINTS_SERVICE_VERSION and
// ESPv2_ARGS) into startproxy flags, like env_start_proxy.py's gen_args.
func serverlessFlags() ([]string, error) {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		// Without a port there is nowhere to serve an error page.
		return nil, &serverlessError{msg: "Serverless ESPv2 expects PORT in environment variables.\n"}
	}
	flags := []string{"--on_serverless", "--http_port=" + port}

	if servicePath, ok := os.LookupEnv("ENDPOINTS_SERVICE_PATH"); ok {
		flags = append(flags, "--rollout_strategy=fixed", "--service_json_path="+servicePath)
	} else {
		serviceName, ok := os.LookupEnv("ENDPOINTS_SERVICE_NAME")
		if !ok {
			return nil, &serverlessError{
				msg:     "Serverless ESPv2 expects ENDPOINTS_SERVICE_NAME in environment variables.\n" + missingServiceConfigMsg,
				port:    port,
				warning: true,
			}
		}
		flags = append(flags, "--service="+serviceName)
		if version, ok := os.LookupEnv("ENDPOINTS_SERVICE_VERSION"); ok {
			flags = append(flags, "--rollout_strategy=fixed", "--version="+version)
		} else {
			flags = append(flags, "--rollout_strategy=managed")
		}
	}

	if value, ok := os.LookupEnv("ESPv2_ARGS"); ok {
		// ESPv2_ARGS is comma-separated by default. If an argument needs a
		// comma, pick another delimiter and put it between carets first:
		// ^++^--cors_allow_methods="GET,POST,PUT,OPTIONS"++--cors_allow_credentials
		delim := ","
		if strings.HasPrefix(value, "^") && strings.Contains(value[1:], "^") {
			delim, value, _ = strings.Cut(value[1:], "^")
		}
		if delim == "" {
			return nil, &serverlessError{msg: malformedESPv2ArgsMsg, port: port}
		}
		flags = append(flags, strings.Split(value, delim)...)
	}
	return flags, nil
}

// errorPageHandler responds to every request with 503 Service Unavailable and msg.
func errorPageHandler(msg string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(msg))
	})
}

// serveErrorPage serves msg as a 503 page on port until a shutdown signal
// arrives. It returns the process exit code.
func (l *launcher) serveErrorPage(port, msg string) int {
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           errorPageHandler(msg),
		ReadHeaderTimeout: 10 * time.Second,
	}
	sigCh := make(chan os.Signal, 1)
	l.notifySignals(sigCh)
	defer l.stopSignals(sigCh)

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	select {
	case err := <-errCh:
		glog.Errorf("Failed to serve error message: %v", err)
	case sig := <-sigCh:
		glog.Warningf("got signal: %s", signalName(sig))
		_ = srv.Close()
	}
	return 1
}
