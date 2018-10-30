.DEFAULT_GOAL	:= build

#------------------------------------------------------------------------------
# Variables
#------------------------------------------------------------------------------

SHELL 	:= /bin/bash
PKG 		:= cloudesf.googleresource.com/gcpproxy/configmanager

# Pure Go sources (not vendored and not generated)
GOFILES		= $(shell find . -type f -name '*.go' -not -path "./vendor/*")
GODIRS		= $(shell go list -f '{{.Dir}}' ./... \
						| grep -vFf <(go list -f '{{.Dir}}' ./vendor/...))


all: build

build:
	@echo "--> building"
	@go build ./...

test:
	@echo "--> running unit tests"
	@go test ./configmanager/...

