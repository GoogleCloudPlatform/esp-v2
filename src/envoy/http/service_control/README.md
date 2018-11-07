
# How run service control filter

## Start backend http echo server at port 8080

## run envoy

* change the service_name to your in src/envoy/http/service_control/envoy.yaml
* bazel run //src/envoy:envoy -- -c $PWD/src/envoy/http/service_control/envoy.yaml -l debug

## run http client

* Get your api-key.
* KEY=YOUR-API-KEY
* curl http://127.0.0.1:9090/echo?key=$KEY
