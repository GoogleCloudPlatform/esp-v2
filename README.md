# Google Cloud Platform ESPv2

Google Cloud Platform ESPv2 is a service proxy which enables API
management capabilities for JSON/REST or gRPC API services. The current
implementation uses [Envoy](https://www.envoyproxy.io/) as a service proxy.

ESPv2 provides:

*   **Features**: authentication (auth0), API key validation, JSON to
    Protobuf transcoding, user quota rate limiting, as well as API-level monitoring, tracing and logging.

*   **Easy Adoption**: the API service can be implemented in any coding language
    using any IDLs.

*   **Platform flexibility**: support the deployment on any cloud or on-premise
    environment.

*   **Superb performance and scalability**: low latency and high throughput

## Introduction

ESPv2 is a general-purpose L7 service proxy that integrates with Google hosted
services to provide policy checks and telemetry reports. This proxy can be used by
GCP customers, Google Cloud products, and Google internal projects.

ESPv2 can run on GCP and hybrid cloud environments, either as a sidecar or as an API gateway.
However, initial development was primarily done on GKE for API services using [Open API
Specification](https://openapis.org/specification) so our instructions
and samples are focusing on these platforms. If you make it work on other
infrastructure and IDLs, please let us know and contribute instructions/code.

ESPv2 includes two components:

- ConfigManager: Control plane to configure the Envoy proxy
- Envoy: Data plane to process API requests/responses

ConfigManager configures the data plane's Envoy filters dynamically, using [Google API
Service Configuration](https://github.com/googleapis/googleapis/blob/master/google/api/service.proto)
and flags specified by the API producer.

Envoy (with our custom filters) handles API calls using [Service Infrastructure]
(https://cloud.google.com/service-infrastructure/docs/overview), Google's foundational
platform for creating, managing, and consuming APIs and services.

* [Architecture](/doc/architecture.png)
* [ESPv2 Filters](doc/filters.png)
* [API Producer specified flags](docker/generic/start_proxy.py)

## ESPv2 Releases

ESPv2 is released as a docker image.

Currently we only support ESPv2 on Cloud Run:

- [gcr.io/endpoints-release/endpoints-runtime:2](https://gcr.io/endpoints-release/endpoints-runtime:2)

## Repository Structure

* [api](/api): Envoy Filter Configurations developed in ESPv2
* [doc](/doc): Documentation
* [docker](/docker): Scripts for packaging ESPv2 in a Docker image for releases
* [prow](/prow): Prow based test automation scripts
* [scripts](/scripts): Scripts used for build and release ESPv2
* [src](/src): ESPv2 source code, including Envoy Filters and Config Manager
* [tests](/tests): Integration and end-to-end tests for ESPv2
* [tools](/third_party/tools): Assorted tooling

## Contributing

Your contributions are welcome. Please follow the contributor [guidelines](CONTRIBUTING.md).

* [Developer Guide](DEVELOPER.md)

## ESPv2 Tutorial

To find out more about building, running, and testing ESPv2:

* [Run ESPv2 on Google Cloud Run](/doc/esp-v2-on-cloudrun.md)

* [Run ESPv2 on Google GKE](/doc/esp-v2-on-k8s.md)

## Disclaimer

ESPv2 is in Beta currently.

Please make sure to join [google-cloud-endpoints](https://groups.google.com/forum/#!forum/google-cloud-endpoints) Google group, to get emails about all announcements on ESPv2.