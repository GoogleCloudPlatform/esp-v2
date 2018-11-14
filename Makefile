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
TAG := $(shell date +v%Y%m%d)-$(shell git describe --tags --always --dirty)
K8S := master

GOFILES		= $(shell find . -type f -name '*.go' -not -path "./vendor/*")
GODIRS		= $(shell go list -f '{{.Dir}}' ./... \
						| grep -vFf <(go list -f '{{.Dir}}' ./vendor/...))

#-----------------------------------------------------------------------------
# Target: go build
# ----------------------------------------------------------------------------

.PHONY: build
build:
	@echo "--> building"
	@go build ./src/go/...
	@go build ./tests/configmanager/...

#-----------------------------------------------------------------------------
# Target: go test
# ----------------------------------------------------------------------------

.PHONY: test test-debug
test:
	@echo "--> running unit tests"
	@go test -v ./src/go/...

test-debug:
	@echo "--> running unit tests"
	@go test -v ./src/go/... --logtostderr

#-----------------------------------------------------------------------------
# Target: go dependencies
#-----------------------------------------------------------------------------
.PHONY: depend.update depend.install

depend.update: tools.glide depend.agentproto
	@echo "--> updating dependencies from glide.yaml"
	@glide update

depend.install: tools.glide depend.agentproto
	@echo "--> installing dependencies from glide.lock "
	@glide install

depend.agentproto:
	@echo "--> generating go agent proto files"
	@bazel build //api/agent:agent_service_go_grpc
	@mkdir -p src/go/proto/agent
	@cp -f bazel-bin/api/agent/linux_amd64_stripped/agent_service_go_grpc%/cloudesf.googlesource.com/gcpproxy/src/go/proto/agent/* src/go/proto/agent

vendor:
	@echo "--> installing dependencies from glide.lock "
	@glide install

#----------------------------------------------------------------------------
# Target:  go tools
#----------------------------------------------------------------------------
.PHONY: tools tools.glide tools.goimports tools.golint tools.govet

tools: tools.glide tools.goimports tools.golint tools.govet

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

.PHONY: clean
clean:
	@echo "--> cleaning compiled objects and binaries"
	@go clean -tags netgo -i ./...
	@rm -rf $(BINDIR)/*
	@rm -rf bazel-*

# Should always be called before pushing changes.
.PHONY: check
check: format.check vet lint

.PHONY: format
format: tools.goimports
	@echo "--> formatting code with 'goimports' tool"
	@goimports -local $(PKG) -w -l $(GOFILES)

.PHONY: format.check
format.check: tools.goimports
	@echo "--> checking code formatting with 'goimports' tool"
	@goimports -local $(PKG) -l $(GOFILES) | sed -e "s/^/\?\t/" | tee >(test -z)

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
	docker build -f docker/Dockerfile-prow-env --build-arg IMAGE_ARG=$(IMG):$(TAG)-$(K8S) -t $(IMG):$(VERSION)-$(K8S) .

docker.push-prow: docker.build-prow
	docker push $(IMG):$(TAG)-$(K8S)

docker.build-configmanager:
	docker build -f docker/Dockerfile-configmanager -t configmanager .
