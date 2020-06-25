
This folder stores filter config proto files for ESPv2 http filters.

## Config versioning:

ESPv2 config is versioned. The current version is stored in `api/VERSION` file.
The folder names under `api/` contain version, e.g. `api/envoy.v7.http/backend_auth`.
The proto package names contain version too, e.g. `espv2.api.envoy.v7.http.common.Pattern`.

## Versioning Rules:
When making changes to the config proto files, make sure:
* No breaking changes, the changes should be backward compatible,
* If a breaking change is required, increase config version.

## Steps to increase config version
If a breaking change is required, use following steps to increase config version.
* Increase `api/VERSION` to a newer version, e.g. from `v6` to `v7`.
* Rename folder name from `api/envoy/v6/http` to `api/envoy/v7/http`.
* Replace package names from `api.envoy.v6.http` to `api.envoy.v7.http` for all proto files under folder `api/`.

Above steps can be achieved by running script `api/script/update_version.sh`.
