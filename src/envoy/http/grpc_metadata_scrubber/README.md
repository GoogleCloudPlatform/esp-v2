# gRPC Metadata Scrubber Filter

## Overview

This filter checks response headers; if content-type is "application/grpc", if the response
header has content-length, remove content-length header.

This is needed in some special deployment cases with followings:
* downstream uses http1 to transport gRPC requests and uses trunk encoding to support response trailers.
* upstream adds content-length
* envoy drops the response trailers if the response headers have content-length.

