
This folder stores filter config proto files for ESPv2 http filters.

## Config versioning:

ESPv2 config is versioned. The current version is stored in `api/VERSION` file.
The folder names under `api/` contain version, e.g. `api/envoy/v12/http/backend_auth`.
The proto package names contain version too, e.g. `espv2.api.envoy.v12.http.common.Pattern`.

## Versioning Rules:

Cloud API Gateway may run:

* An older Envoy binary with a newer Config Generator
* A newer Envoy binary with an older Config Generator

When making changes to the config proto files, make sure:

* No breaking changes, the changes should be backward compatible,
* If a breaking change is required, increase config version.

Examples of breaking changes:

* Adding a new filter config to Config Generator: Old Envoys with new config
  will fail to parse the filter config proto type URL.

Examples of potential breaking changes:

* Removing a field: Older configs may last for years in CAG, leading to
  functionality breaking when new Envoys rollout, as they ignore the old field.
* Adding a new CLI flag to GCSRunner: This is baked into the data plane and may
  change functionality when older configs are used.

Examples of non-breaking changes:

* Adding a new field to pre-existing filter: Old Envoys with new config will
  ignore the new field, and the new feature won't be enabled. New Envoys with
  old config will also have the feature disabled.

## Steps to increase config version

If a breaking change is required, use following steps to increase config version.
* Increase `api/VERSION` to a newer version, e.g. from `v6` to `v7`.
* Rename folder name from `api/envoy/v6/http` to `api/envoy/v12/http`.
* Replace package names from `api.envoy.v6.http` to `api.envoy.v12.http` for all proto files under folder `api/`.

Above steps can be automated by running:
* api/scripts/update_version.sh
* api/scripts/go_proto_gen.sh

and use "make test" and "make integration-test" to verify the results.

