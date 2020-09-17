# gRPC Metadata Scrubber Filter

## Overview

This filter checks response headers; if content-type is "application/grpc" and the response
header has content-length, remove content-length header.

This is to retain gRPC trailers in some special deployment cases with followings:
* downstream uses http1 to transport gRPC requests and uses trunk encoding to retain gRPC trailers.
* but upstream somehow adds content-length in the response headers.

Envoy drops the gRPC trailers according to the [RFC](https://tools.ietf.org/html/rfc7230#section-4.1.2)
if the response headers have content-length when sending the response to the
[downstream http1 codec](https://github.com/envoyproxy/envoy/blob/master/source/common/http/http1/codec_impl.cc).
