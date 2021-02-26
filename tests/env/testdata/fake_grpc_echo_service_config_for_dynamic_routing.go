// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testdata

import (
	"fmt"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
)

var (
	grpcEchoForDynamicRoutingServiceConfigJsonStr = fmt.Sprintf(`{
  "producer_project_id": "producer-project",
  "name": "grpc-echo.endpoints.cloudesf-testing.cloud.goog",
  "id": "test-config-id",
  "title": "GRPC Echo Test",
  "apis": [{
    "name": "test.grpc.Test",
    "methods": [{
      "name": "Echo",
      "requestTypeUrl": "type.googleapis.com/test.grpc.EchoRequest",
      "responseTypeUrl": "type.googleapis.com/test.grpc.EchoResponse"
    }, {
      "name": "EchoStream",
      "requestTypeUrl": "type.googleapis.com/test.grpc.EchoRequest",
      "requestStreaming": true,
      "responseTypeUrl": "type.googleapis.com/test.grpc.EchoResponse",
      "responseStreaming": true
    }, {
      "name": "Cork",
      "requestTypeUrl": "type.googleapis.com/test.grpc.CorkRequest",
      "requestStreaming": true,
      "responseTypeUrl": "type.googleapis.com/test.grpc.CorkState",
      "responseStreaming": true
    }, {
      "name": "EchoReport",
      "requestTypeUrl": "type.googleapis.com/google.api.servicecontrol.v1.ReportRequest",
      "responseTypeUrl": "type.googleapis.com/google.api.servicecontrol.v1.ReportRequest"
    }],
    "version": "v1",
    "sourceContext": {
      "fileName": "grpc-test.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }],
  "types": [{
    "name": "google.protobuf.Any",
    "fields": [{
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "type_url",
      "jsonName": "typeUrl"
    }, {
      "kind": "TYPE_BYTES",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "value",
      "jsonName": "value"
    }],
    "sourceContext": {
      "fileName": "google/protobuf/any.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "google.protobuf.Struct",
    "fields": [{
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_REPEATED",
      "number": 1,
      "name": "fields",
      "typeUrl": "type.googleapis.com/google.protobuf.Struct.FieldsEntry",
      "jsonName": "fields"
    }],
    "sourceContext": {
      "fileName": "google/protobuf/struct.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "google.protobuf.Struct.FieldsEntry",
    "fields": [{
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "key",
      "jsonName": "key"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "value",
      "typeUrl": "type.googleapis.com/google.protobuf.Value",
      "jsonName": "value"
    }],
    "options": [{
      "name": "google.protobuf.MessageOptions.map_entry",
      "value": {
        "@type": "type.googleapis.com/google.protobuf.BoolValue",
        "value": true
      }
    }],
    "sourceContext": {
      "fileName": "google/protobuf/struct.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "google.protobuf.Value",
    "fields": [{
      "kind": "TYPE_ENUM",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "null_value",
      "typeUrl": "type.googleapis.com/google.protobuf.NullValue",
      "oneofIndex": 1,
      "jsonName": "nullValue"
    }, {
      "kind": "TYPE_DOUBLE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "number_value",
      "oneofIndex": 1,
      "jsonName": "numberValue"
    }, {
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 3,
      "name": "string_value",
      "oneofIndex": 1,
      "jsonName": "stringValue"
    }, {
      "kind": "TYPE_BOOL",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 4,
      "name": "bool_value",
      "oneofIndex": 1,
      "jsonName": "boolValue"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 5,
      "name": "struct_value",
      "typeUrl": "type.googleapis.com/google.protobuf.Struct",
      "oneofIndex": 1,
      "jsonName": "structValue"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 6,
      "name": "list_value",
      "typeUrl": "type.googleapis.com/google.protobuf.ListValue",
      "oneofIndex": 1,
      "jsonName": "listValue"
    }],
    "oneofs": ["kind"],
    "sourceContext": {
      "fileName": "google/protobuf/struct.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "google.protobuf.ListValue",
    "fields": [{
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_REPEATED",
      "number": 1,
      "name": "values",
      "typeUrl": "type.googleapis.com/google.protobuf.Value",
      "jsonName": "values"
    }],
    "sourceContext": {
      "fileName": "google/protobuf/struct.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "google.protobuf.Timestamp",
    "fields": [{
      "kind": "TYPE_INT64",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "seconds",
      "jsonName": "seconds"
    }, {
      "kind": "TYPE_INT32",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "nanos",
      "jsonName": "nanos"
    }],
    "sourceContext": {
      "fileName": "google/protobuf/timestamp.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "google.api.servicecontrol.v1.LogEntry",
    "fields": [{
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 10,
      "name": "name",
      "jsonName": "name"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 11,
      "name": "timestamp",
      "typeUrl": "type.googleapis.com/google.protobuf.Timestamp",
      "jsonName": "timestamp"
    }, {
      "kind": "TYPE_ENUM",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 12,
      "name": "severity",
      "typeUrl": "type.googleapis.com/google.logging.type.LogSeverity",
      "jsonName": "severity"
    }, {
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 4,
      "name": "insert_id",
      "jsonName": "insertId"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_REPEATED",
      "number": 13,
      "name": "labels",
      "typeUrl": "type.googleapis.com/google.api.servicecontrol.v1.LogEntry.LabelsEntry",
      "jsonName": "labels"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "proto_payload",
      "typeUrl": "type.googleapis.com/google.protobuf.Any",
      "oneofIndex": 1,
      "jsonName": "protoPayload"
    }, {
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 3,
      "name": "text_payload",
      "oneofIndex": 1,
      "jsonName": "textPayload"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 6,
      "name": "struct_payload",
      "typeUrl": "type.googleapis.com/google.protobuf.Struct",
      "oneofIndex": 1,
      "jsonName": "structPayload"
    }],
    "oneofs": ["payload"],
    "sourceContext": {
      "fileName": "google/api/servicecontrol/v1/log_entry.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "google.api.servicecontrol.v1.LogEntry.LabelsEntry",
    "fields": [{
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "key",
      "jsonName": "key"
    }, {
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "value",
      "jsonName": "value"
    }],
    "options": [{
      "name": "google.protobuf.MessageOptions.map_entry",
      "value": {
        "@type": "type.googleapis.com/google.protobuf.BoolValue",
        "value": true
      }
    }],
    "sourceContext": {
      "fileName": "google/api/servicecontrol/v1/log_entry.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "google.api.servicecontrol.v1.Distribution",
    "fields": [{
      "kind": "TYPE_INT64",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "count",
      "jsonName": "count"
    }, {
      "kind": "TYPE_DOUBLE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "mean",
      "jsonName": "mean"
    }, {
      "kind": "TYPE_DOUBLE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 3,
      "name": "minimum",
      "jsonName": "minimum"
    }, {
      "kind": "TYPE_DOUBLE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 4,
      "name": "maximum",
      "jsonName": "maximum"
    }, {
      "kind": "TYPE_DOUBLE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 5,
      "name": "sum_of_squared_deviation",
      "jsonName": "sumOfSquaredDeviation"
    }, {
      "kind": "TYPE_INT64",
      "cardinality": "CARDINALITY_REPEATED",
      "number": 6,
      "name": "bucket_counts",
      "packed": true,
      "jsonName": "bucketCounts"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 7,
      "name": "linear_buckets",
      "typeUrl": "type.googleapis.com/google.api.servicecontrol.v1.Distribution.LinearBuckets",
      "oneofIndex": 1,
      "jsonName": "linearBuckets"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 8,
      "name": "exponential_buckets",
      "typeUrl": "type.googleapis.com/google.api.servicecontrol.v1.Distribution.ExponentialBuckets",
      "oneofIndex": 1,
      "jsonName": "exponentialBuckets"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 9,
      "name": "explicit_buckets",
      "typeUrl": "type.googleapis.com/google.api.servicecontrol.v1.Distribution.ExplicitBuckets",
      "oneofIndex": 1,
      "jsonName": "explicitBuckets"
    }],
    "oneofs": ["bucket_option"],
    "sourceContext": {
      "fileName": "google/api/servicecontrol/v1/distribution.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "google.api.servicecontrol.v1.Distribution.LinearBuckets",
    "fields": [{
      "kind": "TYPE_INT32",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "num_finite_buckets",
      "jsonName": "numFiniteBuckets"
    }, {
      "kind": "TYPE_DOUBLE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "width",
      "jsonName": "width"
    }, {
      "kind": "TYPE_DOUBLE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 3,
      "name": "offset",
      "jsonName": "offset"
    }],
    "sourceContext": {
      "fileName": "google/api/servicecontrol/v1/distribution.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "google.api.servicecontrol.v1.Distribution.ExponentialBuckets",
    "fields": [{
      "kind": "TYPE_INT32",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "num_finite_buckets",
      "jsonName": "numFiniteBuckets"
    }, {
      "kind": "TYPE_DOUBLE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "growth_factor",
      "jsonName": "growthFactor"
    }, {
      "kind": "TYPE_DOUBLE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 3,
      "name": "scale",
      "jsonName": "scale"
    }],
    "sourceContext": {
      "fileName": "google/api/servicecontrol/v1/distribution.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "google.api.servicecontrol.v1.Distribution.ExplicitBuckets",
    "fields": [{
      "kind": "TYPE_DOUBLE",
      "cardinality": "CARDINALITY_REPEATED",
      "number": 1,
      "name": "bounds",
      "packed": true,
      "jsonName": "bounds"
    }],
    "sourceContext": {
      "fileName": "google/api/servicecontrol/v1/distribution.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "google.api.servicecontrol.v1.MetricValue",
    "fields": [{
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_REPEATED",
      "number": 1,
      "name": "labels",
      "typeUrl": "type.googleapis.com/google.api.servicecontrol.v1.MetricValue.LabelsEntry",
      "jsonName": "labels"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "start_time",
      "typeUrl": "type.googleapis.com/google.protobuf.Timestamp",
      "jsonName": "startTime"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 3,
      "name": "end_time",
      "typeUrl": "type.googleapis.com/google.protobuf.Timestamp",
      "jsonName": "endTime"
    }, {
      "kind": "TYPE_BOOL",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 4,
      "name": "bool_value",
      "oneofIndex": 1,
      "jsonName": "boolValue"
    }, {
      "kind": "TYPE_INT64",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 5,
      "name": "int64_value",
      "oneofIndex": 1,
      "jsonName": "int64Value"
    }, {
      "kind": "TYPE_DOUBLE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 6,
      "name": "double_value",
      "oneofIndex": 1,
      "jsonName": "doubleValue"
    }, {
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 7,
      "name": "string_value",
      "oneofIndex": 1,
      "jsonName": "stringValue"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 8,
      "name": "distribution_value",
      "typeUrl": "type.googleapis.com/google.api.servicecontrol.v1.Distribution",
      "oneofIndex": 1,
      "jsonName": "distributionValue"
    }],
    "oneofs": ["value"],
    "sourceContext": {
      "fileName": "google/api/servicecontrol/v1/metric_value.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "google.api.servicecontrol.v1.MetricValue.LabelsEntry",
    "fields": [{
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "key",
      "jsonName": "key"
    }, {
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "value",
      "jsonName": "value"
    }],
    "options": [{
      "name": "google.protobuf.MessageOptions.map_entry",
      "value": {
        "@type": "type.googleapis.com/google.protobuf.BoolValue",
        "value": true
      }
    }],
    "sourceContext": {
      "fileName": "google/api/servicecontrol/v1/metric_value.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "google.api.servicecontrol.v1.MetricValueSet",
    "fields": [{
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "metric_name",
      "jsonName": "metricName"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_REPEATED",
      "number": 2,
      "name": "metric_values",
      "typeUrl": "type.googleapis.com/google.api.servicecontrol.v1.MetricValue",
      "jsonName": "metricValues"
    }],
    "sourceContext": {
      "fileName": "google/api/servicecontrol/v1/metric_value.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "google.api.servicecontrol.v1.Operation",
    "fields": [{
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "operation_id",
      "jsonName": "operationId"
    }, {
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "operation_name",
      "jsonName": "operationName"
    }, {
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 3,
      "name": "consumer_id",
      "jsonName": "consumerId"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 4,
      "name": "start_time",
      "typeUrl": "type.googleapis.com/google.protobuf.Timestamp",
      "jsonName": "startTime"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 5,
      "name": "end_time",
      "typeUrl": "type.googleapis.com/google.protobuf.Timestamp",
      "jsonName": "endTime"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_REPEATED",
      "number": 6,
      "name": "labels",
      "typeUrl": "type.googleapis.com/google.api.servicecontrol.v1.Operation.LabelsEntry",
      "jsonName": "labels"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_REPEATED",
      "number": 7,
      "name": "metric_value_sets",
      "typeUrl": "type.googleapis.com/google.api.servicecontrol.v1.MetricValueSet",
      "jsonName": "metricValueSets"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_REPEATED",
      "number": 8,
      "name": "log_entries",
      "typeUrl": "type.googleapis.com/google.api.servicecontrol.v1.LogEntry",
      "jsonName": "logEntries"
    }, {
      "kind": "TYPE_ENUM",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 11,
      "name": "importance",
      "typeUrl": "type.googleapis.com/google.api.servicecontrol.v1.Operation.Importance",
      "jsonName": "importance"
    }],
    "sourceContext": {
      "fileName": "google/api/servicecontrol/v1/operation.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "google.api.servicecontrol.v1.Operation.LabelsEntry",
    "fields": [{
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "key",
      "jsonName": "key"
    }, {
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "value",
      "jsonName": "value"
    }],
    "options": [{
      "name": "google.protobuf.MessageOptions.map_entry",
      "value": {
        "@type": "type.googleapis.com/google.protobuf.BoolValue",
        "value": true
      }
    }],
    "sourceContext": {
      "fileName": "google/api/servicecontrol/v1/operation.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "google.api.servicecontrol.v1.ReportRequest",
    "fields": [{
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "service_name",
      "jsonName": "serviceName"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_REPEATED",
      "number": 2,
      "name": "operations",
      "typeUrl": "type.googleapis.com/google.api.servicecontrol.v1.Operation",
      "jsonName": "operations"
    }, {
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 3,
      "name": "service_config_id",
      "jsonName": "serviceConfigId"
    }],
    "sourceContext": {
      "fileName": "google/api/servicecontrol/v1/service_controller.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "test.grpc.EchoRequest",
    "fields": [{
      "kind": "TYPE_BYTES",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "text",
      "jsonName": "text"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "return_status",
      "typeUrl": "type.googleapis.com/test.grpc.CallStatus",
      "jsonName": "returnStatus"
    }, {
      "kind": "TYPE_INT32",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 3,
      "name": "random_payload_max_size",
      "jsonName": "randomPayloadMaxSize"
    }, {
      "kind": "TYPE_INT32",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 6,
      "name": "space_payload_size",
      "jsonName": "spacePayloadSize"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_REPEATED",
      "number": 4,
      "name": "return_initial_metadata",
      "typeUrl": "type.googleapis.com/test.grpc.EchoRequest.ReturnInitialMetadataEntry",
      "jsonName": "returnInitialMetadata"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_REPEATED",
      "number": 5,
      "name": "return_trailing_metadata",
      "typeUrl": "type.googleapis.com/test.grpc.EchoRequest.ReturnTrailingMetadataEntry",
      "jsonName": "returnTrailingMetadata"
    }],
    "sourceContext": {
      "fileName": "grpc-test.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "test.grpc.EchoRequest.ReturnInitialMetadataEntry",
    "fields": [{
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "key",
      "jsonName": "key"
    }, {
      "kind": "TYPE_BYTES",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "value",
      "jsonName": "value"
    }],
    "options": [{
      "name": "google.protobuf.MessageOptions.map_entry",
      "value": {
        "@type": "type.googleapis.com/google.protobuf.BoolValue",
        "value": true
      }
    }],
    "sourceContext": {
      "fileName": "grpc-test.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "test.grpc.EchoRequest.ReturnTrailingMetadataEntry",
    "fields": [{
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "key",
      "jsonName": "key"
    }, {
      "kind": "TYPE_BYTES",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "value",
      "jsonName": "value"
    }],
    "options": [{
      "name": "google.protobuf.MessageOptions.map_entry",
      "value": {
        "@type": "type.googleapis.com/google.protobuf.BoolValue",
        "value": true
      }
    }],
    "sourceContext": {
      "fileName": "grpc-test.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "test.grpc.EchoResponse",
    "fields": [{
      "kind": "TYPE_BYTES",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 101,
      "name": "text",
      "jsonName": "text"
    }, {
      "kind": "TYPE_FIXED64",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 102,
      "name": "elapsed_micros",
      "jsonName": "elapsedMicros"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_REPEATED",
      "number": 103,
      "name": "received_metadata",
      "typeUrl": "type.googleapis.com/test.grpc.EchoResponse.ReceivedMetadataEntry",
      "jsonName": "receivedMetadata"
    }],
    "sourceContext": {
      "fileName": "grpc-test.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "test.grpc.EchoResponse.ReceivedMetadataEntry",
    "fields": [{
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "key",
      "jsonName": "key"
    }, {
      "kind": "TYPE_BYTES",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "value",
      "jsonName": "value"
    }],
    "options": [{
      "name": "google.protobuf.MessageOptions.map_entry",
      "value": {
        "@type": "type.googleapis.com/google.protobuf.BoolValue",
        "value": true
      }
    }],
    "sourceContext": {
      "fileName": "grpc-test.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "test.grpc.CallStatus",
    "fields": [{
      "kind": "TYPE_INT32",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "code",
      "jsonName": "code"
    }, {
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "details",
      "jsonName": "details"
    }],
    "sourceContext": {
      "fileName": "grpc-test.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "test.grpc.CorkRequest",
    "sourceContext": {
      "fileName": "grpc-test.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "test.grpc.CorkState",
    "fields": [{
      "kind": "TYPE_INT64",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "current_corked_calls",
      "jsonName": "currentCorkedCalls"
    }],
    "sourceContext": {
      "fileName": "grpc-test.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }],
  "enums": [{
    "name": "google.logging.type.LogSeverity",
    "enumvalue": [{
      "name": "DEFAULT"
    }, {
      "name": "DEBUG",
      "number": 100
    }, {
      "name": "INFO",
      "number": 200
    }, {
      "name": "NOTICE",
      "number": 300
    }, {
      "name": "WARNING",
      "number": 400
    }, {
      "name": "ERROR",
      "number": 500
    }, {
      "name": "CRITICAL",
      "number": 600
    }, {
      "name": "ALERT",
      "number": 700
    }, {
      "name": "EMERGENCY",
      "number": 800
    }],
    "sourceContext": {
      "fileName": "google/logging/type/log_severity.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "google.protobuf.NullValue",
    "enumvalue": [{
      "name": "NULL_VALUE"
    }],
    "sourceContext": {
      "fileName": "google/protobuf/struct.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "google.api.servicecontrol.v1.Operation.Importance",
    "enumvalue": [{
      "name": "LOW"
    }, {
      "name": "HIGH",
      "number": 1
    }],
    "sourceContext": {
      "fileName": "google/api/servicecontrol/v1/operation.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }],
  "documentation": {
  },
  "http": {
    "rules": [{
      "selector": "test.grpc.Test.Echo",
      "post": "/echo",
      "body": "*"
    }, {
      "selector": "test.grpc.Test.EchoStream",
      "post": "/echostream",
      "body": "*"
    }, {
      "selector": "test.grpc.Test.EchoReport",
      "post": "/echoreport",
      "body": "*"
    }]
  },
  "quota": {
  },
  "endpoints": [{
    "name": "echo.endpoints.cloudesf-testing.cloud.goog"
  }],
  "configVersion": 3,
  "control": {
    "environment": "servicecontrol.googleapis.com"
  },
  "systemParameters": {
  },
	"backend": {
    "rules": [
      {
        "selector": "test.grpc.Test.Echo",
        "address": "grpc://%v:-1/",
        "jwtAudience": "jwt-aud",
        "deadline": 10.0
      },
      {
        "selector": "test.grpc.Test.EchoStream",
        "address": "grpc://%v:-1/",
        "jwtAudience": "jwt-aud"
      },
      {
        "selector": "test.grpc.Test.Cork",
        "address": "grpc://%v:-1/",
        "jwtAudience": "jwt-aud"
      },
      {
        "selector": "test.grpc.Test.EchoReport",
        "address": "grpc://%v:-1/",
        "jwtAudience": "jwt-aud"
      }
    ]
  }
}`, platform.GetLoopbackAddress(), platform.GetLoopbackAddress(), platform.GetLoopbackAddress(), platform.GetLoopbackAddress())
)
