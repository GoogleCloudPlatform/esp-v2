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

## Introduction

APIProxy is a general purpose L7 proxy that integrates with Google hosted
services, to provide policy check and telemetry reporting, for GCP customers,
Google Cloud products, and Google internal projects. APIProxy can be run on GCP
and hybrid cloud environment, either as a sidecar, or as an API gateway.

APIProxy includes two components: the ConfigManager as Control plane and Envoy
as Data plane.

ConfigManager configures its Envoy filters dynamically, using [Google API
Service Configuration](https://github.com/googleapis/googleapis/blob/master/google/api/service.proto) and API producer specified flags.

* [Architecture](/doc/architecture.png)
* [APIProxy Filters](doc/filters.png)
* [API Producer specified flags](docker/generic/start_proxy.py)

## Released APIProxy docker images

TODO(jilinxia)

## Repository Structure

* [api](/api): Envoy Filter Configurations developed in APIProxy
* [doc](/doc): Documentation
* [docker](/docker): Scripts for packaging APIProxy in a Docker image.
* [prow](/prow): Prow based test automation scripts
* [script](/script): Scripts used for build and release APIProxy
* [src](/src): API Proxy source, including Envoy Filters and Config Manager.
* [tests](/test): Applications and Client code used for integration test and end-to-end test.
* [tools](/tools): Assorted tooling.

## Contributing

Your contributions are welcome. Please follow the contributor [guidelines](CONTRIBUTING.md).

* [Developer Guide](DEVELOPER.md)

## APIProxy Tutorial

To find out more about building, running, and testing APIProxy, please review:

* [Run APIProxy on Kubernetes](/doc/apiproxi-on-k8s.md)

