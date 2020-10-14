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

```
wget -O- https://apt.llvm.org/llvm-snapshot.gpg.key| apt-key add -
echo "deb http://apt.llvm.org/buster/ llvm-toolchain-buster-10 main" >> /etc/apt/sources.list
sudo apt-get update
sudo apt-get install -y llvm-10-dev libclang-10-dev clang-10 \
    clang-tools-10 clang-format-10 xz-utils lld-10
```

## Install Golang

```
sudo apt-get install golang-1.15
```

Add the following setting in .profile, then source it:

```
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
export GOBIN=$GOPATH/bin
```

## Download Source Code

To download the source code, clone the APIProxy repository:

* git clone  "https://github.com/GoogleCloudPlatform/esp-v2"

The following folder is required:

```
- $HOME/go
  - bin
  - src
    -  cloudesf.googlesource.com
       -  gcpproxy
```

## Build Envoy and Config Manager

In order to build Config Manager, need to install its dependent libraries:

```
make depend.install
```

To build Config Manager, run:

```
make build
```

To build Envoy, run:

```
make build-envoy
```

To run integration tests, run:

```
make integration-test
```

## Run Sanitizer Tests

```
make test-envoy-asan/tsan
make integration-test-asan/tsan
```
