# Developer Documentation

ESPv2 is based on Envoy Proxy and Config Manager. Envoy written in C++ is
built using bazel with clang and Config Manager written in go is built using go build.

See the [architecture overview](doc/architecture.md) before getting started.

## Test Env

[![C++ Unit Test Coverage Report](https://img.shields.io/badge/Test%20Coverage-C%2B%2B%20Unit-blue)](https://storage.googleapis.com/esp-v2-coverage/latest/coverage/index.html)
[![OSS Fuzz Status](https://oss-fuzz-build-logs.storage.googleapis.com/badges/esp-v2.svg)](https://bugs.chromium.org/p/oss-fuzz/issues/list?sort=-opened&can=1&q=proj:esp-v2)

### Prow Jobs

View all Prow job statuses on https://testgrid.k8s.io/:

- [Periodic jobs](https://testgrid.k8s.io/googleoss-esp-v2-periodic) from `master` branch once a day
- [Presubmit jobs](https://testgrid.k8s.io/googleoss-esp-v2-presubmit), also viewable as PR status
- [Postsubmit jobs](https://testgrid.k8s.io/googleoss-esp-v2-postsubmit) for ESPv2 release

Prow job configuration is located in [oss-test-infra](https://github.com/GoogleCloudPlatform/oss-test-infra/blob/master/prow/prowjobs/GoogleCloudPlatform/esp-v2/esp-v2.yaml) repository.

### Remote Caching

Prow `build` and `*-presubmit` jobs use Bazel's RBE [Remote Caching](https://bazel.build/remote/caching).
Remote caching dramatically speeds up job build time for incremental builds.

Sometimes remote caches may get corrupted and jobs are stuck in build phase.
Increment the `silo_uuid` in `try_setup_bazel_remote_cache()` to force a cache refresh. 

### End-to-End Tests

A few prow jobs deploy ESPv2 in GCP for full end-to-end tests.
These jobs use GCP project `cloudesf-testing`.

## Developer Setup

There are two ways to setup the development environment for this project: the Docker mode and
the native mode. Due to the frequent version changes of software dependecies, it is recommended to
setup your dev env in Docker mode.
Later, we will walk through how to use Intellij and CLion IDEs for the project.

### Docker Mode Setup

#### Clone [ESPv2 Repo]((https://github.com/GoogleCloudPlatform/esp-v2))

```
git clone https://github.com/GoogleCloudPlatform/esp-v2.git
```

#### Find Latest Docker Image

Go to [Setup yaml](https://github.com/GoogleCloudPlatform/oss-test-infra/blob/master/prow/prowjobs/GoogleCloudPlatform/esp-v2/esp-v2.yaml#L13), copy the latest working image link.

For example: ***gcr.io/cloudesf-testing/gcpproxy-prow:v20240727-v2.46.0-27-g6c21f955-master***

#### Run Docker Instance

Run the following command by replacing the **latest_image_link** with the link found above.

```
docker run -ti -v /var/run/docker.sock:/var/run/docker.sock -v ~/esp-v2:/esp-v2 ${latest_image_link}  /bin/bash
```

#### Change Directory to `esp-v2`

```
cd esp-v2
```

#### Add the following Git Config

```
git config --global --add safe.directory /esp-v2
```

#### Install Dependencies

```
make depend.install
```

#### Build Binaries

```
make build
make build-envoy
```

#### Trigger Integration Test ([iam_imds_data_path_test](https://github.com/GoogleCloudPlatform/esp-v2/blob/master/tests/integration_test/iam_imds_data_path_test/iam_imds_data_path_test.go))

```
go test -v -timeout 20m ./tests/integration_test/iam_imds_data_path_test/iam_imds_data_path_test.go --debug_components=envoy --logtostderr
```


### Native Mode Setup

#### Install Golang

Reference [the official golang installation instructions](https://golang.org/doc/install) for your OS. The latest go version should work.

Add the following setting in .profile, then source it:

```
export GOPATH=$HOME/go
export GOBIN=$GOPATH/bin
export PATH=$GOBIN:$PATH
```

#### Install Bazelisk as Bazel

It is recommended to use [Bazelisk](https://github.com/bazelbuild/bazelisk) installed as `bazel` to avoid Bazel compatibility issues. Follow the instructions on the linked website to install it.

#### Install Envoy dependencies

To get started building Envoy locally, following the instructions from [Envoy](https://github.com/envoyproxy/envoy/blob/master/bazel/README.md#quick-start-bazel-build-for-developers).

#### Install Clang-14

Add the package sources with the following commands.
If these commands fail, reference [the official clang installation instructions](https://apt.llvm.org/) for your OS.

Install the following packages:

```
sudo apt-get update
sudo apt-get install -y llvm-14-dev libclang-14-dev clang-14 \
    clang-tools-14 clang-format-14 xz-utils lld-14
```

#### Install build tools

```
sudo apt install -y cmake ninja-build protobuf-compiler brotli \
    libicu-dev libbrotli-dev
```

#### Install other tools

```
sudo apt-get install jq nodejs npm python-is-python3
```

#### Download Source Code

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

#### Build Envoy and Config Manager

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
make integration-test
```

#### Enable IDE Integration

For the C++ code, we will use CLion with the Blaze / Bazel plugin.
Import the project as a Bazel project. CLion should automatically pick up the targets and run configurations.

For the golang code, we will use Intellij Ultimate edition.
Import the project as a Git project (do NOT use the Bazel plugin).
Intellij will automatically pick up the run configurations.

For both CLion and Intellij, change the blaze plugin to use `bazelisk` instead of `bazel` for any commands:

1) Search for the `Bazel binary location` item in the Intellij Settings
2) Change it to the output of `which bazelisk`

#### Install e2e Testing Tools (optional)

You will need the following tools to run manual e2e tests:

1) Docker Engine
2) `gcloud`

Alternatively, you can rely on ESPv2 Prow CI/CD to run e2e tests.

### Envoy version update

#### Clone [ESPv2 Repo]((https://github.com/GoogleCloudPlatform/esp-v2))

Follow [above steps](#docker-mode-setup) to clone the repo. 
If you didn't clone the code to your home directory (`~`), be sure to update the enlist mounting folder `~/esp-v2` in `docker run` command.

#### Envoy version lookup

Go to https://github.com/envoyproxy/envoy/releases to see the latest release.
The version can be found under the release tag. Copy the commit id and set it as `ENVOY_SHA1` in `WORKSPACE` file.

```text
ENVOY_SHA1 = "86dxxx..."  # v1.32.0

ENVOY_SHA256 = ""
```

To get the `ENVOY_SHA256`, leave the field as empty and run `make depend.install` command locally or submit a pull request online. 
Look for the `DEBUG` log in the following pattern and update the `ENVOY_SHA256` accordingly.

Alternatively we can download source code zip for ENVOY_SHA1 release from releases page & find its sha256 using any cmdline tool.
Download link pattern: https://github.com/envoyproxy/envoy/archive/${ENVOY_SHA1}.zip.

```text
DEBUG: Rule 'envoy' indicated that a canonical reproducible form can be obtained by modifying arguments sha256 = "e03xxx..."
```
