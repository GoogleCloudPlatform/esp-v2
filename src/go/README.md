# Config Manager: the Control Plane for ESP V2

The best way to configure Envoy inside ESP V2 is through Dynamic
configuration.

This Config Manager is acting as the control plane for ESP V2, it utilizes
the [go-control-plane](https://github.com/envoyproxy/go-control-plane), with
extra features specifically for APIs on Google Cloud Platform.

*   **ESP V2 Startup Configuration**: When starting Config Manager, it
    fetches the latest
    [googleapis API service config](https://github.com/googleapis/api-common-protos/blob/master/google/api/service.proto)
    from
    [Google Service Management](https://cloud.google.com/service-infrastructure/docs/service-management/getting-started),
    translates it to envoy xDS configuration and caches inside go-control-plane,
    which feeds envoy with dynamic configurations.

*   **Auto Service Configuration Update**: When '--rollout_strategy' is set as
    'managed', no need to set '--service_config_id'. Instead, Config Manager calls
    [Google Service Management](https://cloud.google.com/service-infrastructure/docs/service-management/getting-started) to get the latest rollout, and retrieves
    the version id with maximum traffic percentage in it. Then, Config Manager
    fetches the corresponding service config and dynamically configures envoy proxy.

    What is more, Config Manager checks with Google Service Management every 60
    seconds, to see whether there is new rollout or not. If yes, it will
    fetches the new deployed service config and updates envoy configurations,
    automatically and silently.
    (Note: currently ESP V2 doesn't support
    [Traffic Percentage Strategy](https://github.com/googleapis/googleapis/blob/master/google/api/servicemanagement/v1/resources.proto#L227))

## Prerequisites:

Since Config Manager utilizes the open source go-control-plane, all
[requirements](https://github.com/envoyproxy/go-control-plane#requirements) for
go-control-plane need to be satisfied.

## Usage:

To start the Config Manager, run:

```shell
go run src/go/configmanager/main/server.go \
  --logtostderr -v 2 \
  --service [YOUR_SERVICE_NAME] \
  --service_config_id [YOUR_CONFIG_ID] \
  --backend_protocol {grpc | http1 | http2}
```

if you want to enable glog, add "-log_dir=./log -v=2".

You should see "config manager server is running at ......" if starting
successfully.

## Quick Test

We have a simple gRPC test client to fetch Listener Discovery Service(LDS)
response from this Config Manager, just run:

```shell
go run tests/clients/lds_grpc_client.go --logtostderr
```

You can see and check the response.
