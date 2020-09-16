# gRPC Metadata Scrubber Filter

## Overview

This filter checks response headers. For a gRPC response, content-type is "application/grpc", if the response
header has content-length, remove the content-length header.

This is needed in the special deployment with followings:
* downstream uses http1 to transport gRPC requests and uses trunk encoding to support response trailers.
* upstream adds content-length
* envoy drops response trailers if it sees response headers has content-length.
