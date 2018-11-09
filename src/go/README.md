# Config Manager: the Control Plane for API Proxy

The best way to configure Envoy inside API Proxy is through Dynamic
configuration.

This Config Manager is acting as the control plane for API Proxy, it utilizes
the [go-control-plane](https://github.com/envoyproxy/go-control-plane), with
extra features specifically for APIs on Google Cloud Platform.

*   **API Proxy Startup Configuration**: When starting Config Manager, it
    fetches the latest
    [googleapis API service config](https://github.com/googleapis/api-common-protos/blob/master/google/api/service.proto)
    from
    [Google Service Management](https://cloud.google.com/service-infrastructure/docs/service-management/getting-started),
    translates it to envoy xDS configuration and caches inside go-control-plane,
    which feeds envoy with dynamic configurations.

*   **Auto Service Configuration Update**: [TBD]

## Prerequisites:

Since Config Manager utilize the open source go-control-plane, all
[requirements](https://github.com/envoyproxy/go-control-plane#requirements) for
go-control-plane need to be satisfied.

## Usage:

To start the Config Manager, run:

```shell
go run server/server.go --logtostderr -v 2 --service_name [YOUR_PROJECT_NAME]  \
--config_id [YOUR_CONFIG_ID]
```

if you want to enable glog, add "-log_dir=./log -v=2".

You should see "config manager server is running ......" if starting
successfully.

## Quick Test

We have a simple gRPC test client to fetch Listener Discovery Service(LDS)
response from this Config Manager, just run:

```shell
go run tests/lds_grpc_client.go
```

You can see and check the response.

## Manually Integration with API Proxy

Start Config Manager first as instructed above, then build and start
cloudesf-envoy with the dynamic startup configuration:

```shell
bazel build :cloudesf-envoy &&
bazel-bin/cloudesf-envoy -l info --v2-config-only -c tools/deploy/envoy_bootstrap_v2_startup.yaml
```
