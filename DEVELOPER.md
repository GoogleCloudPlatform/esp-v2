# Developer Documentation

ESPv2 is based on Envoy Proxy and Config Manager. Envoy written in C++ is
built using bazel with clang and Config Manager written in go is built using go build.

See the [architecture overview](doc/architecture.md) before getting started.

We recommend starting this setup using only the command line at first.
Later, we will walk through how to use Intellij and CLion IDEs for the project.

## Install Golang

Reference [the official golang installation instructions](https://golang.org/doc/install) for your OS. The latest go version should work.

Add the following setting in .profile, then source it:

```
export GOPATH=$HOME/go
export GOBIN=$GOPATH/bin
export PATH=$GOBIN:$PATH
```

## Install Bazelisk as Bazel

It is recommended to use [Bazelisk](https://github.com/bazelbuild/bazelisk) installed as `bazel` to avoid Bazel compatibility issues. Follow the instructions on the linked website to install it.

## Install Envoy dependencies

To get started building Envoy locally, following the instructions from [Envoy](https://github.com/envoyproxy/envoy/blob/master/bazel/README.md#quick-start-bazel-build-for-developers).

## Install Clang-13

Add the package sources with the following commands.
If these commands fail, reference [the official clang installation instructions](https://apt.llvm.org/) for your OS.

Install the following packages:

```
sudo apt-get update
sudo apt-get install -y llvm-13-dev libclang-13-dev clang-13 \
    clang-tools-13 clang-format-13 xz-utils lld-13
```

## Install build tools

```
sudo apt install -y cmake ninja-build protobuf-compiler brotli \
    libicu-dev libbrotli-dev
```

## Install other tools

```
sudo apt-get install jq nodejs npm python-is-python3
```

## Download Source Code

To download the source code, use `go get`:

```
go get github.com/GoogleCloudPlatform/esp-v2
```

This will create the following folder structure:

```
- $HOME/go
  - bin
  - src
    -  github.com
       -  GoogleCloudPlatform
          - esp-v2
```

## Build Envoy and Config Manager

In order to build Config Manager, need to install its dependent libraries:

```
make depend.install
make depend.install.endpoints
```

To build:

```
# Config Manager
make build
# Envoy
make build-envoy
```

To run unit tests:

```
# Config Manager
make test
# Envoy
make test-envoy
```

To run integration tests:

```
make integration-test-run-parallel
```

## Enable IDE Integration

For the C++ code, we will use CLion with the Blaze / Bazel plugin.
Import the project as a Bazel project. CLion should automatically pick up the targets and run configurations.

For the golang code, we will use Intellij Ultimate edition.
Import the project as a Git project (do NOT use the Bazel plugin).
Intellij will automatically pick up the run configurations.

For both CLion and Intellij, change the blaze plugin to use `bazelisk` instead of `bazel` for any commands:

1) Search for the `Bazel binary location` item in the Intellij Settings
2) Change it to the output of `which bazelisk`

## Install e2e Testing Tools (optional)

You will need the following tools to run manual e2e tests:

1) Docker Engine
2) `gcloud`

Alternatively, you can rely on ESPv2 Prow CI/CD to run e2e tests.
