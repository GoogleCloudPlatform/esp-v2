# Developer documentation

APIProxy is based on Envoy Proxy in C++ and Config Manager in Golang. Envoy is
built using Bazel, and Config Manager is built using Make.

## Install Envoy dependencies

To get started building Envoy locally, following the instructions from [Envoy](https://github.com/envoyproxy/envoy/blob/master/bazel/README.md#quick-start-bazel-build-for-developers).

## Install clang-format-7

```
# NOTE: replace stretch with relevant version if your GCE VM is not Debian stretch
$ sudo apt-get install -y software-properties-common
$ sudo add-apt-repository "deb http://apt.llvm.org/stretch/ llvm-toolchain-stretch-7 main"
$ wget -O - https://apt.llvm.org/llvm-snapshot.gpg.key | sudo apt-key add -
$ sudo apt-get update

$ sudo apt-get install clang-format-7
```

## Install Golang

```
sudo apt-get install golang-1.11
```

Add the following setting in .profile, then source it:

```
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
export GOBIN=$GOPATH/bin
```

## Download source code

To download the source code, clone the APIProxy repository:

* git clone  "https://cloudesf.googlesource.com/gcpproxy"

The following folder is required:

```
- $HOME/go
  - bin
  - src
    -  cloudesf.googlesource.com
       -  gcpproxy
```

## Build Envoy and ConfigManager

In order to build ConfigManager, need to install its dependent libraries:

```
make depend.install
```

To build ConfigManager, run:

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