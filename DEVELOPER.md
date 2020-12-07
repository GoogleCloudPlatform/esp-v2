# Developer Documentation

ESPv2 is based on Envoy Proxy and Config Manager. Envoy written in C++ is
built using bazel with clang and Config Manager written in go is built using go build.

See the [architecture overview](doc/architecture.md) before getting started.

## Installing Bazelisk as Bazel

It is recommended to use [Bazelisk](https://github.com/bazelbuild/bazelisk) installed as `bazel`, to avoid Bazel compatibility issues.
On Linux, run the following commands:

```
sudo wget -O /usr/local/bin/bazel https://github.com/bazelbuild/bazelisk/releases/download/v0.0.8/bazelisk-linux-amd64
sudo chmod +x /usr/local/bin/bazel
```

## Install Envoy dependencies

To get started building Envoy locally, following the instructions from [Envoy](https://github.com/envoyproxy/envoy/blob/master/bazel/README.md#quick-start-bazel-build-for-developers).

## Install Clang-10

Add the package sources with the following commands.
If these commands fail, reference [the official clang installation instructions](https://apt.llvm.org/) for your OS.

```
# Commands require sudo. Force password prompt early to cache credentials.
sudo echo "hello"

wget -O- https://apt.llvm.org/llvm-snapshot.gpg.key | sudo apt-key add -
echo "deb http://apt.llvm.org/buster/ llvm-toolchain-buster-10 main" | sudo tee -a /etc/apt/sources.list >/dev/null
```

Install the following packages:

```
sudo apt-get update
sudo apt-get install -y llvm-10-dev libclang-10-dev clang-10 \
    clang-tools-10 clang-format-10 xz-utils lld-10
```

## Install Golang

If these commands fail, reference [the official golang installation instructions](https://golang.org/doc/install) for your OS.

```
sudo apt-get install golang-1.15
```

Add the following setting in .profile, then source it:

```
export GOPATH=$HOME/go
export GOBIN=$GOPATH/bin
export PATH=$GOBIN:$PATH
```

## Install other tools

```
sudo apt-get install jq
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
