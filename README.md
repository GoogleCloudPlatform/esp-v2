# Google Cloud Platform API Proxy

Google Cloud Platform API Proxy is a service proxy which enables API
management capabilities for JSON/REST or gRPC API services. The current
implementation uses [Envoy](https://www.envoyproxy.io/) as a service proxy.

API Proxy provides:

*   **Features**: authentication (auth0, gitkit), API key validation, JSON to
    gRPC transcoding, as well as API-level monitoring, tracing and logging. More
    features coming in the near future: quota, billing, ACL, etc.

*   **Easy Adoption**: the API service can be implemented in any coding language
    using any IDLs.

*   **Platform flexibility**: support the deployment on any cloud or on-premise
    environment.

*   **Superb performance and scalability**: low latency and high throughput

## Introduction

API Proxy is a general-purpose L7 service proxy that integrates with Google hosted
services to provide policy checks and telemetry reports. This proxy can be used by
GCP customers, Google Cloud products, and Google internal projects.

API Proxy can run on GCP and hybrid cloud environments, either as a sidecar or as an API gateway.
However, initial development was primarily done on GKE for API services using [Open API
Specification](https://openapis.org/specification) so our instructions
and samples are focusing on these platforms. If you make it work on other
infrastructure and IDLs, please let us know and contribute instructions/code.

API Proxy includes two components:

- ConfigManager: Control plane to configure the Envoy proxy
- Envoy: Data plane to process API requests/responses

ConfigManager configures the data plane's Envoy filters dynamically, using [Google API
Service Configuration](https://github.com/googleapis/googleapis/blob/master/google/api/service.proto)
and flags specified by the API producer.

Envoy (with our custom filters) handles API calls using [Service Infrastructure]
(https://cloud.google.com/service-infrastructure/docs/overview), Google's foundational
platform for creating, managing, and consuming APIs and services.

* [Architecture](/doc/architecture.png)
* [API Proxy Filters](doc/filters.png)
* [API Producer specified flags](docker/generic/start_proxy.py)

## API Proxy Releases

API Proxy is released as a docker image. The current stable docker images are:

- [gcr.io/apiproxy-release/apiproxy-serverless:latest](https://gcr.io/apiproxy-release/apiproxy-serverless:latest)

More documentation on releases will be coming soon.

## Repository Structure

* [api](/api): Envoy Filter Configurations developed in API Proxy
* [doc](/doc): Documentation (more coming soon)
* [docker](/docker): Scripts for packaging API Proxy in a Docker image for releases
* [prow](/prow): Prow based test automation scripts
* [scripts](/scripts): Scripts used for build and release API Proxy
* [src](/src): API Proxy source code, including Envoy Filters and Config Manager
* [tests](/tests): Integration and end-to-end tests for API Proxy
* [tools](/tools): Assorted tooling

## Contributing

Your contributions are welcome. Please follow the contributor [guidelines](CONTRIBUTING.md).

* [Developer Guide](DEVELOPER.md)

## API Proxy Tutorial

To find out more about building, running, and testing API Proxy:

* [Run API Proxy on Google Cloud Run](/doc/apiproxy-on-cloudrun.md)

* [Run API Proxy on Google GKE](/doc/apiproxy-on-k8s.md)

## Disclaimer

API Proxy is still in Alpha. This is not an officially supported Google product.