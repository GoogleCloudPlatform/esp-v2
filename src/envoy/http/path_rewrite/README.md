# Path Rewrite filter

This filter can be configured to modify request path when sending to upstream.

The path can be modifed in two ways
*  prepend a fixed prefix to the path
*  change into a fixed path. This is for sending request to Google Cloud Function.
   Its HTTP trigger URL is a fixed name.

This filter will be funtional independently.

View the [path_rewrite configuration proto](../../../../api/envoy/v9/http/path_rewrite/config.proto)
for inline documentation.

## Statistics

This filter records statistics.

### Counters

- `path_changed`: Number of API Consumer requests whose paths have been modified.
- `path_not_changed`: Number of API Consumer requests whose paths have not been modified.
- `denied_by_no_path`: Number of API Consumer requests that are denied due to path header not present.
- `denied_by_invalid_path`: Number of API Consumer requests that are denied due to path has fragments.
- `denied_by_no_route`: Number of API Consumer requests that are denied due to not route configurated.
- `denied_by_url_template_mismatch`: Number of API Consumer requests that are denied due to mismatched url_template.
