# Instructions to run service control filter

## start backend http server at port 8080

```bash
$ node tools/backend/http_server.js
```

## start up envoy

Update the service_name to yours in `src/envoy/http/service_control/envoy.yaml`

```bash
$ sed 's/REPLACE_SERVICE_NAME/{YOUR_SERVICE_NAME}' src/envoy/http/service_control/envoy.yaml
```

Then start up Envoy by

```bash
$ bazel run //src/envoy:envoy -- -c $PWD/src/envoy/http/service_control/envoy.yaml -l debug
```

`envoy.yaml` defines the Envoy's listener port is `9090` and then Envoy routes the request
to the backend at port `8080`

## send http request with api key

```bash
# Get your api-key.
$ KEY=YOUR-API-KEY

# GET request with API key in the query
$ curl http://127.0.0.1:9090/test?key=$KEY -v

# GET request with API key in the headers
$ curl http://127.0.0.1:9090/test --header "x-goog-apikey:$KEY" -v

# GET request that does not match any pattern, this should return path not 
# matched error
$ curl http://127.0.0.1:9090/tea?key=$KEY -v

# POST request with API key in the query
$ curl -X POST http://127.0.0.1:9090/test?key=$KEY -v

# POST request with API key in the headers
$ curl -X POST http://127.0.0.1:9090/test --header "x-goog-apikey:$KEY" -v

# POST request that does not match any pattern, this should return path not 
# matched error
$ curl -X POST http://127.0.0.1:9090/tea?key=$KEY -v
```
