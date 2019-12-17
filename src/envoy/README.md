# Envoy Filters

This directory contains code which, when built, will be compiled together with
[Envoy][Envoy Home] as [HTTP filters][HTTP Filters]. This
[Envoy filter example][Envoy filter example] may help see how this is done.

The filters are then configured to be used by envoy in one of two ways:

## Bootstrap Configuration

Filters are configured in the `listeners` section of the `envoy.yaml` [bootstrap] file.
See [this example][example envoy] for details.

```yaml
filter_chains:
  - filters:
    - name: envoy.http_connection_manager
      config:
        ... other config options ...
        http_filters:
          ... list of filters...
```

## Config Manager

See the
[Config Manager documentation][Config Manager] for more details.

[bootstrap]: https://www.envoyproxy.io/docs/envoy/latest/api-v2/config/bootstrap/v2/bootstrap.proto#config-bootstrap-v2-bootstrap
[Config Manager]: ../go/README.md
[Envoy filter example]: https://github.com/envoyproxy/envoy-filter-example
[Envoy Home]: https://www.envoyproxy.io/
[example envoy]: http/service_control/testdata/envoy.yaml
[HTTP Filters]: https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/http/http_filters
