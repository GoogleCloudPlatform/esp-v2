# Config Manager: the Control Plane for ESPv2

The best way to configure Envoy inside ESPv2 is through Dynamic
configuration.

This Config Manager is acting as the control plane for ESPv2, it utilizes
the [go-control-plane](https://github.com/envoyproxy/go-control-plane), with
extra features specifically for APIs on Google Cloud Platform.

*   **ESPv2 Startup Configuration**: When starting Config Manager, it
    fetches the latest
    [googleapis API service config](https://github.com/googleapis/api-common-protos/blob/master/google/api/service.proto)
    from
    [Google Service Management](https://cloud.google.com/service-infrastructure/docs/service-management/getting-started),
    translates it to envoy xDS configuration and caches inside go-control-plane,
    which feeds envoy with dynamic configurations.

*   **Auto Service Configuration Update**: When '--rollout_strategy' is set as
    'managed', no need to set '--service_config_id'. Instead, Config Manager calls
    [Google Service Management](https://cloud.google.com/service-infrastructure/docs/service-management/getting-started) to get the latest rollout, and retrieves
    the version id with maximum traffic percentage in it. Then, Config Manager
    fetches the corresponding service config and dynamically configures envoy proxy.

    Besides, Config Manager checks with Google Service Management every 60
    seconds, to see whether there is new rollout or not. If yes, it will
    fetches the new deployed service config and updates envoy configurations,
    automatically and silently.
    (Note: currently ESPv2 doesn't support
    [Traffic Percentage Strategy](https://github.com/googleapis/googleapis/blob/master/google/api/servicemanagement/v1/resources.proto#L227))

## Prerequisites

Config Manager uses the [go.mod](../../go.mod) file to define all dependencies.

## Running

Config Manager depends on other local and remote services in order to run.
It is recommended you run Config Manager from our docker image or integration tests instead.