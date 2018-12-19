# Config Manager Release Process

To build API Proxy Docker image, run

```shell
docker build -f docker/Dockerfile-proxy -t gcpproxy .
```

To run API Proxy Docker image, run

```shell
docker run --rm -it -p 8080:8080 gcpproxy
--service=bookstore.endpoints.cloudesf-testing.cloud.goog
--version=2018-11-09r0
--backend=grpc://127.0.0.1:8082
```
