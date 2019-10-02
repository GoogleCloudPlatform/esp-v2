# Copyright 2018 Google Cloud Platform Proxy Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#------------------------------------------------------------------------------
# Variables
#------------------------------------------------------------------------------

SHELL 	:= /bin/bash
BINDIR	:= bin
PKG 	:= cloudesf.googleresource.com/gcpproxy

IMG="gcr.io/cloudesf-testing/gcpproxy-prow"
TAG := $(shell date +v%Y%m%d)-$(shell git describe --tags --always)
K8S := master

CPP_PROTO_FILES = $(shell find . -type f \
		-regex "./\(src\|api\)/.*[.]\(h\|cc\|proto\)" \
		-not -path "./vendor/*")
GOFILES	= $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./src/go/proto/*")
GODIRS	= $(shell go list -f '{{.Dir}}' ./... \
					| grep -vFf <(go list -f '{{.Dir}}' ./vendor/...))

#-----------------------------------------------------------------------------
# Target: build
# ----------------------------------------------------------------------------

$(BINDIR):
	@mkdir -p $(BINDIR)

.PHONY: build build-envoy build-grpc-echo build-grpc-interop upload-e2e-client-binaries
build: format
	@echo "--> building"
	@go build ./src/go/...
	@go build ./tests...
	@go build -o bin/configmanager ./src/go/configmanager/main/server.go
	@go build -o bin/bootstrap ./src/go/bootstrap/ads/main/main.go
	@go build -o bin/echo/server ./tests/endpoints/echo/server/app.go

build-envoy:
	@echo "--> building envoy"
	@bazel build //src/envoy:envoy
	@cp -f bazel-bin/src/envoy/envoy bin/

build-grpc-echo:
	@echo "--> building grpc-echo"
	@bazel build --cxxopt='-std=c++14' tests/endpoints/grpc_echo:grpc-test-client --incompatible_no_support_tools_in_action_inputs=false
	@bazel build //tests/endpoints/grpc_echo:grpc-test-server --incompatible_no_support_tools_in_action_inputs=false
	@bazel build tests/endpoints/grpc_echo:grpc-test_descriptor --incompatible_no_support_tools_in_action_inputs=false
	@cp -f bazel-bin/tests/endpoints/grpc_echo/grpc-test-client bin/grpc_echo_client
	@cp -f bazel-bin/tests/endpoints/grpc_echo/grpc-test-server bin/grpc_echo_server
	@cp -f bazel-genfiles/tests/endpoints/grpc_echo/grpc-test.descriptor tests/endpoints/grpc_echo/proto/api_descriptor.pb


build-grpc-interop:
	@echo "--> building the grpc-interop-test client and server"
	@bazel build @com_github_grpc_grpc//test/cpp/interop:interop_client
	@bazel build @com_github_grpc_grpc//test/cpp/interop:metrics_client
	@bazel build @com_github_grpc_grpc//test/cpp/interop:interop_server
	@bazel build @com_github_grpc_grpc//test/cpp/interop:stress_test
	@cp -f bazel-bin/external/com_github_grpc_grpc/test/cpp/interop/interop_client bin/
	@cp -f bazel-bin/external/com_github_grpc_grpc/test/cpp/interop/metrics_client bin/
	@cp -f bazel-bin/external/com_github_grpc_grpc/test/cpp/interop/interop_server bin/
	@cp -f bazel-bin/external/com_github_grpc_grpc/test/cpp/interop/stress_test bin/

# This target is to upload e2e client binaries so once they get updated,
# this target should be executed.
upload-e2e-client-binaries: build-grpc-echo build-grpc-interop
	@mkdir -p /tmp/apiproxy-test-presubmit-binaries/
	@cp -n bin/interop_client /tmp/apiproxy-test-presubmit-binaries/interop_client
	@cp -n bin/metrics_client /tmp/apiproxy-test-presubmit-binaries/metrics_client
	@cp -n bin/stress_test /tmp/apiproxy-test-presubmit-binaries/stress_test
	@cp -n bin/grpc_echo_client /tmp/apiproxy-test-presubmit-binaries/grpc_echo_client
	@cp -n tests/endpoints/grpc_echo/proto/api_descriptor.pb /tmp/apiproxy-test-presubmit-binaries/api_descriptor.pb
	@gsutil cp -Z -r /tmp/apiproxy-test-presubmit-binaries/* "gs://apiproxy-testing-presubmit-binaries/"


#-----------------------------------------------------------------------------
# Target: go test
# ----------------------------------------------------------------------------

.PHONY: test test-debug test-envoy
test: format
	@echo "--> running unit tests"
	@go test ./src/go/...
	@python3 -m unittest tests/start_proxy/start_proxy_test.py

test-debug: format
	@echo "--> running unit tests"
	@go test -v ./src/go/... --logtostderr

test-envoy: format
	@echo "--> running envoy's unit tests"
	@bazel test //src/...


.PHONY: integration-test integration-debug
integration-test: build build-envoy build-grpc-interop build-grpc-echo
	@echo "--> running integration tests"
	# Default timeout for go test is 10 minutes. Our test suite takes a little longer...
	# logtostderr will cause all glogs in the test framework to print to the console (not too much bloat)
	@go test -v -timeout 20m ./tests/env/... --logtostderr
	@go test -v -p 32 -timeout 20m ./tests/integration/... --logtostderr

integration-debug: build build-envoy build-grpc-interop build-grpc-echo
	@echo "--> running integration tests and showing debug logs"
	@go test -v -timeout 20m ./tests/env/... --logtostderr
	# debug-components can be set as "all", "configmanager", or "envoy".
	@go test -v -timeout 20m ./tests/integration/... --debug_components=envoy --logtostderr

#-----------------------------------------------------------------------------
# Target: dependencies
#-----------------------------------------------------------------------------
.PHONY: depend.update depend.install

depend.update: tools.glide
	@echo "--> generating go proto files"
	./api/scripts/go_proto_gen.sh

depend.install: tools.glide tools.beautysh
	@echo "--> generating go proto files"
	./api/scripts/go_proto_gen.sh

depend.install.endpoints:
	@echo "--> updating dependencies from package.json"
	@npm install ./tests/endpoints/bookstore_grpc/ --no-package-lock
	@npm install ./tests/e2e/testdata/bookstore/ --no-package-lock

#----------------------------------------------------------------------------
# Target:  tools
#----------------------------------------------------------------------------
.PHONY: tools tools.glide tools.goimports tools.golint tools.govet \
	tools.buildifier tools.beautysh

tools: tools.glide tools.goimports tools.golint tools.govet tools.buildifier

tools.goimports:
	@command -v goimports >/dev/null ; if [ $$? -ne 0 ]; then \
		echo "--> installing goimports"; \
		go get golang.org/x/tools/cmd/goimports; \
	fi

tools.govet:
	@go tool vet 2>/dev/null ; if [ $$? -eq 3 ]; then \
		echo "--> installing govet"; \
		go get golang.org/x/tools/cmd/vet; \
	fi

tools.golint:
	@command -v golint >/dev/null ; if [ $$? -ne 0 ]; then \
		echo "--> installing golint"; \
		go get -u golang.org/x/lint/golint; \
	fi

tools.glide:
	@command -v glide >/dev/null ; if [ $$? -ne 0 ]; then \
		echo "--> installing glide"; \
		curl https://glide.sh/get | sh; \
	fi

tools.buildifier:
	@command -v buildifier >/dev/null ; if [ $$? -ne 0 ]; then \
		echo "--> installing buildifier"; \
		go get github.com/bazelbuild/buildtools/buildifier; \
	fi

tools.beautysh:
	@command -v beautysh  >/dev/null ; if [ $$? -ne 0 ]; then \
		echo "--> installing beautysh"; \
		pip install --user beautysh; \
	fi


.PHONY: clean
clean:
	@echo "--> cleaning compiled objects and binaries"
	@go clean -tags netgo -i ./...
	@rm -rf $(BINDIR)/*
	@rm -rf bazel-*

# Should always be called before pushing changes.
.PHONY: check
check: format.check vet lint

.PHONY: spelling.check
spelling.check:
	@echo "--> checking spelling"
	@tools/spelling/check_spelling.sh check

.PHONY: spelling.fix
spelling.fix:
	@echo "--> fixing spelling"
	@tools/spelling/check_spelling.sh fix

.PHONY: format
format: tools.goimports tools.buildifier
	@echo "--> formatting code with 'goimports' tool"
	@goimports -local $(PKG) -w -l $(GOFILES)
	@echo "--> formatting BUILD files with 'buildifier' tool"
	@buildifier -r WORKSPACE ./src/ ./api/

	@make spelling.fix

.PHONY: clang-format shell-format
clang-format:
	@echo "--> formatting code with 'clang-format-7' tool"
	@echo $(CPP_PROTO_FILES) | xargs clang-format-7 -i

shell-format: tools.beautysh
	@echo "--> formatting shell scripts with 'beautysh' tool"
	@git ls-files "*.sh" | xargs ${HOME}/.local/bin/beautysh -i 2

.PHONY: format.check
format.check: tools.goimports
	@echo "--> checking code formatting with 'goimports' tool"
	@goimports -local $(PKG) -l $(GOFILES) | sed -e "s/^/\?\t/" | tee >(test -z)
	@make spelling.check



.PHONY: vet
vet: tools.govet
	@echo "--> checking code correctness with 'go vet' tool"
	@go vet ./...

.PHONY: lint
lint: tools.golint
	@echo "--> checking code style with 'golint' tool"
	@echo $(GODIRS) | xargs -n 1 golint

#-----------------------------------------------------------------------------
# Target : docker
# ----------------------------------------------------------------------------

.PHONY: docker.build-prow, docker.push-prow, docker.build-configmanager
docker.build-prow:
	docker build -f docker/Dockerfile-prow-env --build-arg IMAGE_ARG=$(IMG):$(TAG)-$(K8S) -t $(IMG):$(TAG)-$(K8S) .

docker.push-prow: docker.build-prow
	docker push $(IMG):$(TAG)-$(K8S)

# bookstore image used in e2e test. Only push when there is changes.
docker.build-bookstore:
	docker build -f tests/e2e/testdata/bookstore/bookstore.Dockerfile -t gcr.io/cloudesf-testing/app:bookstore .
docker.push-bookstore:
	gcloud docker -- push gcr.io/cloudesf-testing/app:bookstore
