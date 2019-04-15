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

*   **Auto Service Configuration Update**: When '--rollout_strategy' is set as
    'managed', no need to set '--version'. Instead, Config Manager calls
    [Google Service Management](https://cloud.google.com/service-infrastructure/docs/service-management/getting-started) to get the latest rollout, and retrieves
    the version id with maximum traffic percentage in it. Then, Config Manager
    fetches the corresponding service config and dynamically configures envoy proxy.

    What is more, Config Manager checks with Google Service Management every 60
    seconds, to see whether there is new rollout or not. If yes, it will
    fetches the new deployed service config and updates envoy configurations,
    automatically and silently.
    (Note: currently API Proxy doesn't support
    [Traffic Percentage Strategy](https://github.com/googleapis/googleapis/blob/master/google/api/servicemanagement/v1/resources.proto#L227))

## Prerequisites:

Since Config Manager utilize the open source go-control-plane, all
[requirements](https://github.com/envoyproxy/go-control-plane#requirements) for
go-control-plane need to be satisfied.

## Usage:

To start the Config Manager, run:

```shell
go run server/server.go \
  --logtostderr -v 2 \
  --service [YOUR_SERVICE_NAME] \
  --version [YOUR_CONFIG_ID] \
  --backend_protocol {grpc | http1 | http2}
```

if you want to enable glog, add "-log_dir=./log -v=2".

You should see "config manager server is running ......" if starting
successfully.

## Quick Test

We have a simple gRPC test client to fetch Listener Discovery Service(LDS)
response from this Config Manager, just run:

```shell
go run tests/clients/lds_grpc_client.go
```

You can see and check the response.

## Starting a Backend Server for Testing
### gRPC Bookstore Server
```shell
# Install Node.js if you haven't
# https://github.com/nodesource/distributions/blob/master/README.md#debinstall

# Install the dependencies if you haven't
npm install express
npm install grpc
npm install @grpc/proto-loader

# Run gRPC server
node tests/endpoints/bookstore-grpc/grpc_server.js
```

### HTTP Echo Server
```shell
go run tests/endpoints/echo/server/app.go
```


## Manual Integration with API Proxy

Start Config Manager and a backend first as instructed above, then build and start
Envoy with the dynamic startup configuration:

```shell
bazel build //src/envoy:envoy &&
bazel-bin/src/envoy -l info -c scripts/envoy_bootstrap_v2_startup.yaml
```

## Run API Proxy in Docker

* On the VM instance, create your own container network called apiproxy_net.

```shell
docker network create --driver bridge apiproxy_net
```

* Build and run Bookstore backend server

```shell
docker build -f tests/endpoints/bookstore-grpc/Dockerfile -t bookstore .

docker run --detach --name=bookstore --net=apiproxy_net bookstore
```

* Build and run API Proxy

```shell
docker build -f docker/Dockerfile-proxy -t apiproxy .

docker run --detach --name=apiproxy  --publish=80:8080 --net=apiproxy_net \
apiproxy --service=[YOUR_SERVICE_NAME] --version=[YOUR_CONFIG_ID] \
--backend=grpc://bookstore:8082
```

* Make gRPC calls

```shell
go run tests/endpoints/bookstore-grpc/client_main.go --addr=127.0.0.1:80 \
--method=ListShelves --client_protocol=grpc
```
