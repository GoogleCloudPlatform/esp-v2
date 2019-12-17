# Service Control Filter

[Google Service Infrastructure](https://cloud.google.com/service-infrastructure/docs/overview)
is Google's foundational platform for creating, managing, and consuming APIs and services.

This filter uses [Google Service Control's REST API](https://cloud.google.com/service-infrastructure/docs/service-control/reference/rest/)
to check authentication, rate-limit calls, report metrics, and create logs for API requests.

## Prerequisites

This filter will not function unless the following filters appear earlier in the filter chain:

- [Path Matcher](../path_matcher/README.md)