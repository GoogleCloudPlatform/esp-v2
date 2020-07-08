# ESPv2 Configurations Testdata

Similar to [ESPv2 Configurations](../README.md), but primarily for:
- Full Config Manager bootstrap tests
- Explaining config translation to developers

## [Path Matcher](path_matcher)

Configuration example for the [Path Matcher filter](../../src/envoy/http/path_matcher/README.md).

Specifically, this tests constant address path translation, where the request has path parameters in snake_case.

**Operation Name (Selector)**:

- In the OpenAPI specification, the `path` and `HTTP method` serve as a unique ID.
See the [OpenAPI docs for more information](https://swagger.io/docs/specification/paths-and-operations/)
- In the generated service configuration, the `api.methods.name` serves as a unique ID (such as `GetShelf`).
All other fields in the proto contain references to the operation name via the `selector` field.

**Variable Bindings**:

- In the OpenAPI specification, [Path Templates](https://swagger.io/docs/specification/paths-and-operations/) can be used.
- The generated service configuration preserves these fields, as seen in `http.rules.get`.
- Under certain scenarios, the path matcher filter must parse these variable bindings.
See [Understanding path translation](https://cloud.google.com/endpoints/docs/openapi/openapi-extensions#understanding_path_translation)
for more information.