# Service Control Filter

[Google Service Infrastructure](https://cloud.google.com/service-infrastructure/docs/overview)
is Google's foundational platform for creating, managing, and consuming APIs and services.

This filter uses [Google Service Control's REST API](https://cloud.google.com/service-infrastructure/docs/service-control/reference/rest/)
to check authentication, rate-limit calls, report metrics, and create logs for API requests.

## Prerequisites

This filter will not function unless the following filters appear earlier in the filter chain:

- [Path Matcher](../path_matcher/README.md)

## Statistics

This filter records statistics.

### Counters

- `allowed`: Total number of API consumer requests allowed.
- `allowed_control_plane_fault`: Number of API consumer requests allowed
 due to network fail open policy when Service Control Check was unavailable.
- `denied`: Total number of API consumer requests denied.
- `denied_control_plane_fault`: Number of API consumer requests denied
 due to network fail closed policy when Service Control Check was unavailable.
- `denied_consumer_blocked`: Number of API consumer requests denied due
 to API Key restrictions.
- `denied_consumer_error`: Number of API consumer requests denied due
 to problems with the consumer request.
- `denied_consumer_quota`: Number of API consumer requests denied due
 to exceeding the quota configured by the API Producer.
- `denied_producer_error`: Number of API consumer requests denied due
 to errors in the producer ESPv2 deployment (authentication, roles, etc).

### Histograms

- `request_time` (ms): This is recorded for calls to service control.
 Each operation (Check, AllocateQuota, Report) has its own histogram.
- `backend_time` (ms): Time for the backend to respond.
- `overhead_time` (ms): Overhead introduced by ESPv2.