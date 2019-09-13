// Copyright 2019 Google Cloud Platform Proxy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package grpc_errors_test

import (
	"strings"
	"testing"

	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/grpc_echo/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
)

func TestGRPCErrors(t *testing.T) {
	serviceName := "grpc-echo-service"
	configID := "test-config-id"
	args := []string{"--service=" + serviceName, "--service_config_id=" + configID,
		"--backend_protocol=grpc", "--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestGRPCErrors, "grpc-echo")
	defer s.TearDown()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testPlans := `
plans {
  echo {
    call_config {
      api_key: "this-is-an-api-key"
    }
    request {
      return_status {
        code: 2
        details: "Error propagation test"
      }
      text: "Hello, world!"
    }
  }
}
plans {
  echo {
    call_config {
      api_key: "this-is-an-api-key"
    }
    request {
      return_status {
        code: 3
        details: "Another propagation test"
      }
      text: "Hello, world!"
    }
  }
}
plans {
  echo {
    call_config {
      api_key: "this-is-an-api-key"
    }
    request {
      return_status {
        code: 4
        details: "A long error message.  Like, really ridiculously detailed, the kind of thing you might expect if someone put a Java stack with nested thrown exceptions into an error message, which does actually happen so it is important to make sure long messages are passed through correctly by the grpc_pass implementation within nginx.  Any string longer than 128 bytes should suffice to give us confidence that the HTTP/2 header length encoding implementation at least tries to do the right thing; this one should do just fine."
      }
      text: "Hello, world!"
    }
  }
}
`
	result, err := client.RunGRPCEchoTest(testPlans, s.Ports().ListenerPort)
	wantResult := `
results {
  status {
    code: 2
    details: "Error propagation test"
  }
}
results {
  status {
    code: 3
    details: "Another propagation test"
  }
}
results {
  status {
    code: 4
    details: "A long error message.  Like, really ridiculously detailed, the kind of thing you might expect if someone put a Java stack with nested thrown exceptions into an error message, which does actually happen so it is important to make sure long messages are passed through correctly by the grpc_pass implementation within nginx.  Any string longer than 128 bytes should suffice to give us confidence that the HTTP/2 header length encoding implementation at least tries to do the right thing; this one should do just fine."
  }
}`
	if err != nil {
		t.Errorf("TestGRPCErrors: error during tests: %v", err)
	}
	if !strings.Contains(result, wantResult) {
		t.Errorf("TestGRPCErrors: the results are different,\nreceived:\n%s,\nwanted:\n%s", result, wantResult)
	}
}
