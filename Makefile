# Copyright 2019 Google LLC
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

.PHONY: build build-envoy build-envoy build-envoy-debug build-grpc-echo build-grpc-bookstore build-grpc-interop upload-e2e-client-binaries
build: format
	@echo "--> building"
	@go build ./src/go/...
	@go build ./tests...
	@go build -o bin/configmanager ./src/go/configmanager/main/server.go
	@go build -o bin/bootstrap ./src/go/bootstrap/ads/main/main.go
	@go build -o bin/gcsrunner ./src/go/gcsrunner/main/runner.go
	@go build -o bin/echo/server ./tests/endpoints/echo/server/app.go

build-msan: format
	@echo "--> building"
	@go build -msan ./src/go/...
	@go build -msan ./tests...
	@go build -msan -o bin/configmanager ./src/go/configmanager/main/server.go
	@go build -msan  -o bin/bootstrap ./src/go/bootstrap/ads/main/main.go
	@go build -msan -o bin/gcsrunner ./src/go/gcsrunner/main/runner.go
	@go build -msan -o bin/echo/server ./tests/endpoints/echo/server/app.go

build-race: format
	@echo "--> building"
	@go build -race ./src/go/...
	@go build -race ./tests...
	@go build -race -o bin/configmanager ./src/go/configmanager/main/server.go
	@go build -race  -o bin/bootstrap ./src/go/bootstrap/ads/main/main.go
	@go build -race -o bin/gcsrunner ./src/go/gcsrunner/main/runner.go
	@go build -race -o bin/echo/server ./tests/endpoints/echo/server/app.go



build-envoy-asan:
	@echo "--> building envoy (compilation_mode=fastbuild)"
	@CC=clang-10 CXX=clang++-10 bazel build --config=clang-asan //src/envoy:envoy
	@cp -f bazel-bin/src/envoy/envoy bin/

build-envoy-tsan:
	@echo "--> building envoy (compilation_mode=fastbuild)"
	@CC=clang-10 CXX=clang++-10 bazel build --config=clang-tsan  //src/envoy:envoy
	@cp -f bazel-bin/src/envoy/envoy bin/


build-envoy:
	@echo "--> building envoy (compilation_mode=release)"
	@CC=clang-10 CXX=clang++-10 bazel build --config=clang-release //src/envoy:envoy
	@cp -f bazel-bin/src/envoy/envoy bin/

build-envoy-debug:
	@echo "--> building envoy (compilation_mode=debug)"
	@CC=clang-10 CXX=clang++-10 bazel build --config=debug //src/envoy:envoy
	@cp -f bazel-bin/src/envoy/envoy bin/

build-grpc-echo:
	@echo "--> building grpc-echo"
	@bazel build  //tests/endpoints/grpc_echo:grpc-test-client
	@bazel build  //tests/endpoints/grpc_echo:grpc-test-server
	@bazel build  //tests/endpoints/grpc_echo:grpc-test_descriptor
	@cp -f bazel-bin/tests/endpoints/grpc_echo/grpc-test-client bin/grpc_echo_client
	@cp -f bazel-bin/tests/endpoints/grpc_echo/grpc-test-server bin/grpc_echo_server
	@cp -f bazel-bin/tests/endpoints/grpc_echo/grpc-test.descriptor tests/endpoints/grpc_echo/proto/api_descriptor.pb

build-grpc-bookstore:
	@echo "--> building bookstore-grpc"
	@bazel build tests/endpoints/bookstore_grpc/proto:bookstore_descriptor
	@cp -f bazel-bin/tests/endpoints/bookstore_grpc/proto/bookstore.descriptor tests/endpoints/bookstore_grpc/proto/api_descriptor.pb
	@bazel build @com_google_protobuf//:protoc
	@bazel-bin/external/com_google_protobuf/protoc -I tests/endpoints/bookstore_grpc/proto/v1 -I bazel-esp-v2/external/com_google_protobuf/src -I bazel-esp-v2/external/com_github_googleapis_googleapis \
	  -I tests/endpoints/bookstore_grpc/proto/ tests/endpoints/bookstore_grpc/proto/v1/bookstore.proto --go_out=plugins=grpc:tests/endpoints/bookstore_grpc/proto/v1/
	@bazel-bin/external/com_google_protobuf/protoc -I tests/endpoints/bookstore_grpc/proto/v2 -I bazel-esp-v2/external/com_google_protobuf/src -I bazel-esp-v2/external/com_github_googleapis_googleapis \
	  -I tests/endpoints/bookstore_grpc/proto/ tests/endpoints/bookstore_grpc/proto/v2/bookstore_v2.proto --go_out=plugins=grpc:tests/endpoints/bookstore_grpc/proto/v2/


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

# This target is to upload e2e client binaries so once they get updated, please
# upload the binaries to GCS buckets by this target.
# The execution environment of prow image may be incompatible with your local
# building environment which fails the client binaries so you may have to make
# this target in the prow image, for which you can find the latest version under
# https://github.com/GoogleCloudPlatform/oss-test-infra/blob/master/prow/prowjobs/GoogleCloudPlatform/esp-v2/esp-v2.yaml
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
	# The unit tests under src/go/serviceconfig reads/writes the global variable
	# util.CallGoogleapis(no mutex proctection) plus the function code reads it,
	# so it is easy to get into race condition when multiple test runs
	# and skip them for race detection.
	@go test -race $(shell go list ./src/go/... | grep -v serviceconfig) -count=1
	@go test -msan ./src/go/...
	@python3 -m unittest tests/start_proxy/start_proxy_test.py
	@python3 -m unittest tests/start_proxy/env_start_proxy_test.py

test-debug: format
	@echo "--> running unit tests"
	@go test -v ./src/go/... --logtostderr

test-envoy: clang-format
	@echo "--> running envoy's unit tests"
	@CC=clang-10 CXX=clang++-10  bazel test //src/...

test-envoy-asan: clang-format
	@echo "--> running envoy's unit tests (asan)"
	@CC=clang-10 CXX=clang++-10 ASAN_SYMBOLIZER_PATH=$(which llvm-symbolizer-10) bazel test --config=clang-asan  --test_output=errors //src/...

test-envoy-tsan: clang-format
	@echo "--> running envoy's unit tests (tsan)"
	@CC=clang-10 CXX=clang++-10 ASAN_SYMBOLIZER_PATH=$(which llvm-symbolizer-10) bazel test --config=clang-tsan  --test_output=errors  //src/...

.PHONY: integration-test-run-sequential integration-test-run-parallel integration-test integration-test-asan integration-test-tsan integration-debug
integration-test-run-sequential:
	@echo "--> running integration tests"
	# Default timeout for go test is 10 minutes. Our test suite takes a little longer...
	# logtostderr will cause all glogs in the test framework to print to the console (not too much bloat)
	@go test -timeout 20m ./tests/endpoints/...
	@go test -timeout 20m ./tests/env/... --logtostderr
	@go test -timeout 20m ./tests/utils/... --logtostderr
	@go test -v -timeout 80m -parallel 1 ./tests/integration_test/... --logtostderr

integration-test-run-parallel:
	@echo "--> running integration tests"
	# Default timeout for go test is 10 minutes. Our test suite takes a little longer...
	# logtostderr will cause all glogs in the test framework to print to the console (not too much bloat)
	@go test -timeout 20m ./tests/endpoints/...
	@go test -timeout 20m ./tests/env/... --logtostderr
	@go test -timeout 20m ./tests/utils/... --logtostderr
	@go test -v -timeout 20m ./tests/integration_test/... --logtostderr

integration-test: build  build-envoy build-grpc-interop build-grpc-echo integration-test-run-sequential

integration-debug: build build-envoy-debug build-grpc-interop build-grpc-echo
	@echo "--> running integration tests and showing debug logs"
	@go test -timeout 20m ./tests/endpoints/...
	@go test -v -timeout 20m ./tests/env/... --logtostderr
	@go test -v -timeout 20m ./tests/utils/... --logtostderr
	# debug-components can be set as "all", "configmanager", or "envoy".
	@go test -v -timeout 20m ./tests/integration_test/... --debug_components=envoy --logtostderr

integration-test-asan: build-msan build-envoy-asan build-grpc-interop build-grpc-echo integration-test-run-sequential

# next line is to work around issue: https://github.com/google/sanitizers/issues/953
integration-test-tsan: export TSAN_OPTIONS=report_atomic_races=0
integration-test-tsan: build-race build-envoy-tsan build-grpc-interop build-grpc-echo integration-test-run-sequential


#-----------------------------------------------------------------------------
# Target: dependencies
#-----------------------------------------------------------------------------
.PHONY: depend.update depend.install

depend.update:
	@echo "--> generating go proto files"
	./api/scripts/go_proto_gen.sh

depend.install: tools.beautysh
	@echo "--> generating go proto files"
	./api/scripts/go_proto_gen.sh

depend.install.endpoints:
	@echo "--> updating dependencies from package.json"
	@npm install ./tests/endpoints/bookstore/ --no-package-lock

#----------------------------------------------------------------------------
# Target:  tools
#----------------------------------------------------------------------------
.PHONY: tools tools.goimports tools.golint tools.govet \
	tools.buildifier tools.beautysh

tools: tools.goimports tools.golint tools.govet tools.buildifier

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
	@third_party/tools/spelling/check_spelling.sh check

.PHONY: spelling.fix
spelling.fix:
	@echo "--> fixing spelling"
	@third_party/tools/spelling/check_spelling.sh fix

.PHONY: format
format: tools.goimports tools.buildifier
	@echo "--> formatting code with 'goimports' tool"
	@goimports -w -l $(GOFILES)
	@echo "--> formatting BUILD files with 'buildifier' tool"
	@buildifier -r WORKSPACE ./src/ ./api/
	@echo "--> formatting examples"
	./scripts/format-examples.sh

	@make spelling.fix

.PHONY: clang-format shell-format
clang-format:
	@echo "--> formatting code with 'clang-format-10' tool"
	@echo $(CPP_PROTO_FILES) | xargs clang-format-10 -i

shell-format: tools.beautysh
	@echo "--> formatting shell scripts with 'beautysh' tool"
	@git ls-files "*.sh" | xargs ${HOME}/.local/bin/beautysh -i 2

.PHONY: format.check
format.check: tools.goimports
	@echo "--> checking code formatting with 'goimports' tool"
	@goimports -l $(GOFILES) | sed -e "s/^/\?\t/" | tee >(test -z)
	@make spelling.check

test-ssl-files:
	@openssl genrsa -out tests/env/testdata/server.key 2048
	@openssl req -nodes -new -x509 -sha256 -days 1825 -config tests/env/testdata/cert.conf -extensions 'req_ext' -key tests/env/testdata/server.key -out tests/env/testdata/server.crt
	@cp tests/env/testdata/server.crt tests/env/testdata/client.crt
	@cp tests/env/testdata/server.key tests/env/testdata/client.key
	# TODO(taoxuy@): also add cmd for proxy.crt and proxy.key




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

.PHONY: docker.build-prow docker.push-prow docker.build-configmanager
docker.build-prow:
	docker build -f docker/Dockerfile-prow-env --build-arg IMAGE_ARG=$(IMG):$(TAG)-$(K8S) -t $(IMG):$(TAG)-$(K8S) .

docker.push-prow:
	docker push $(IMG):$(TAG)-$(K8S)

# bookstore image used in e2e test. Only push when there is changes.
docker.build-bookstore:
	docker build -f tests/endpoints/bookstore/bookstore.Dockerfile -t gcr.io/cloudesf-testing/http-bookstore:3 .
docker.push-bookstore:
	gcloud docker -- push gcr.io/cloudesf-testing/http-bookstore:3
