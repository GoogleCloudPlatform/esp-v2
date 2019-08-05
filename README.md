# Google Cloud Platform API Proxy

Google Cloud Platform API Proxy, a.k.a. APIProxy is a proxy which enables API
management capabilities for JSON/REST or gRPC API services. The current
implementation is based on an Envoy proxy server.

APIProxy provides:

*   **Features**: authentication (auth0, gitkit), API key validation, JSON to
    gRPC transcoding, as well as API-level monitoring, tracing and logging. More
    features coming in the near future: quota, billing, ACL, etc.

*   **Easy Adoption**: the API service can be implemented in any coding language
    using any IDLs.

*   **Platform flexibility**: support the deployment on any cloud or on-premise
    environment.

*   **Superb performance and scalability**: low latency and high throughput

## Getting APIProxy

To download the source code, clone the APIProxy repository:

* git clone  "https://cloudesf.googlesource.com/gcpproxy"

## APIProxy Tutorial

To find out more about building, running, and testing APIProxy, please review:

* [Run APIProxy on Kubernetes] (/tests/endpoints/bookstore-grpc/README.md)



