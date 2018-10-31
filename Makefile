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

#-----------------------------------------------------------------------------
# Target: go test
# ----------------------------------------------------------------------------

.PHONY: test
test:
	@echo "--> running unit tests"
	@go test ./src/go/...

#-----------------------------------------------------------------------------
# Target: go dependencies
#-----------------------------------------------------------------------------
.PHONY: depend.update depend.install

depend.update: tools.glide
	@echo "--> updating dependencies from glide.yaml"
	@glide update

depend.install: tools.glide
	@echo "--> installing dependencies from glide.lock "
	@glide install

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

#-----------------------------------------------------------------------------
# Target : docker
# ----------------------------------------------------------------------------

.PHONY: docker-build, docker-push
docker-build:
	docker build --build-arg IMAGE_ARG=$(IMG):$(TAG)-$(K8S) -t $(IMG):$(VERSION)-$(K8S) .

docker-push: docker-build
	docker push $(IMG):$(TAG)-$(K8S)
