
# Instructions to run service control filter

## start backend http server at port 8080

```bash
$ node tools/backend/http_server.js
```

## start up envoy

```bash
# Update the service_name to yours in src/envoy/http/service_control/envoy.yaml
$ sed 's/REPLACE_SERVICE_NAME/{YOUR_SERVICE_NAME}' src/envoy/http/service_control/envoy.yaml
$ bazel run //src/envoy:envoy -- -c $PWD/src/envoy/http/service_control/envoy.yaml -l debug
```

## send http request with api key

```bash
# Get your api-key.
$ KEY=YOUR-API-KEY
$ curl http://127.0.0.1:9090/test?key=$KEY
$ curl http://127.0.0.1:9090/tea?key=AIzaSyB3xeV9fv4agFXUpGVyPMtZ2xIMScEazrk
```