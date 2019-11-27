# Google Cloud Platform ESP V2

Google Cloud Platform ESP V2 is a service proxy which enables API
management capabilities for JSON/REST or gRPC API services. The current
implementation uses [Envoy](https://www.envoyproxy.io/) as a service proxy.

ESP V2 provides:

*   **Features**: authentication (auth0, gitkit), API key validation, JSON to
    gRPC transcoding, as well as API-level monitoring, tracing and logging. More
    features coming in the near future: quota, billing, ACL, etc.

*   **Easy Adoption**: the API service can be implemented in any coding language
    using any IDLs.

*   **Platform flexibility**: support the deployment on any cloud or on-premise
    environment.

*   **Superb performance and scalability**: low latency and high throughput

## Introduction

ESP V2 is a general-purpose L7 service proxy that integrates with Google hosted
services to provide policy checks and telemetry reports. This proxy can be used by
GCP customers, Google Cloud products, and Google internal projects.

ESP V2 can run on GCP and hybrid cloud environments, either as a sidecar or as an API gateway.
However, initial development was primarily done on GKE for API services using [Open API
Specification](https://openapis.org/specification) so our instructions
and samples are focusing on these platforms. If you make it work on other
infrastructure and IDLs, please let us know and contribute instructions/code.

ESP V2 includes two components:

- ConfigManager: Control plane to configure the Envoy proxy
- Envoy: Data plane to process API requests/responses

ConfigManager configures the data plane's Envoy filters dynamically, using [Google API
Service Configuration](https://github.com/googleapis/googleapis/blob/master/google/api/service.proto)
and flags specified by the API producer.

Envoy (with our custom filters) handles API calls using [Service Infrastructure]
(https://cloud.google.com/service-infrastructure/docs/overview), Google's foundational
platform for creating, managing, and consuming APIs and services.

* [Architecture](/doc/architecture.png)
* [ESP V2 Filters](doc/filters.png)
* [API Producer specified flags](docker/generic/start_proxy.py)

## ESP V2 Releases

ESP V2 is released as a docker image. The current stable docker images are:

- [gcr.io/apiproxy-release/apiproxy-serverless:latest](https://gcr.io/apiproxy-release/apiproxy-serverless:latest)

More documentation on releases will be coming soon.

## Repository Structure

* [api](/api): Envoy Filter Configurations developed in ESP V2
* [doc](/doc): Documentation (more coming soon)
* [docker](/docker): Scripts for packaging ESP V2 in a Docker image for releases
* [prow](/prow): Prow based test automation scripts
* [scripts](/scripts): Scripts used for build and release ESP V2
* [src](/src): ESP V2 source code, including Envoy Filters and Config Manager
* [tests](/tests): Integration and end-to-end tests for ESP V2
* [tools](/third_party/tools): Assorted tooling

## Contributing

Your contributions are welcome. Please follow the contributor [guidelines](CONTRIBUTING.md).

* [Developer Guide](DEVELOPER.md)

## ESP V2 Tutorial

To find out more about building, running, and testing ESP V2:

* [Run ESP V2 on Google Cloud Run](/doc/apiproxy-on-cloudrun.md)

* [Run ESP V2 on Google GKE](/doc/apiproxy-on-k8s.md)

## Disclaimer

ESP V2 is still in Alpha. This is not an officially supported Google product.