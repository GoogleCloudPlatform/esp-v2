// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testdata

var (
	grpcInteropServiceConfigJsonStr = `
  {"name": "endpoints-grpc-interop.cloudendpointsapis.com",
  "title": "gRPC Testing API",
  "producerProjectId": "endpoints-grpc-interop",
  "id": "service-config-id",
  "apis": [{
    "name": "grpc.testing.TestService",
    "methods": [{
      "name": "EmptyCall",
      "requestTypeUrl": "type.googleapis.com/grpc.testing.Empty",
      "responseTypeUrl": "type.googleapis.com/grpc.testing.Empty"
    }, {
      "name": "UnaryCall",
      "requestTypeUrl": "type.googleapis.com/grpc.testing.SimpleRequest",
      "responseTypeUrl": "type.googleapis.com/grpc.testing.SimpleResponse"
    }, {
      "name": "StreamingOutputCall",
      "requestTypeUrl": "type.googleapis.com/grpc.testing.StreamingOutputCallRequest",
      "responseTypeUrl": "type.googleapis.com/grpc.testing.StreamingOutputCallResponse",
      "responseStreaming": true
    }, {
      "name": "StreamingInputCall",
      "requestTypeUrl": "type.googleapis.com/grpc.testing.StreamingInputCallRequest",
      "requestStreaming": true,
      "responseTypeUrl": "type.googleapis.com/grpc.testing.StreamingInputCallResponse"
    }, {
      "name": "FullDuplexCall",
      "requestTypeUrl": "type.googleapis.com/grpc.testing.StreamingOutputCallRequest",
      "requestStreaming": true,
      "responseTypeUrl": "type.googleapis.com/grpc.testing.StreamingOutputCallResponse",
      "responseStreaming": true
    }, {
      "name": "HalfDuplexCall",
      "requestTypeUrl": "type.googleapis.com/grpc.testing.StreamingOutputCallRequest",
      "requestStreaming": true,
      "responseTypeUrl": "type.googleapis.com/grpc.testing.StreamingOutputCallResponse",
      "responseStreaming": true
    }],
    "version": "v1",
    "sourceContext": {
      "fileName": "grpc-interop.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }],
  "types": [{
    "name": "grpc.testing.Empty",
    "sourceContext": {
      "fileName": "grpc-interop.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "grpc.testing.Payload",
    "fields": [{
      "kind": "TYPE_ENUM",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "type",
      "typeUrl": "type.googleapis.com/grpc.testing.PayloadType",
      "jsonName": "type"
    }, {
      "kind": "TYPE_BYTES",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "body",
      "jsonName": "body"
    }],
    "sourceContext": {
      "fileName": "grpc-interop.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "grpc.testing.SimpleRequest",
    "fields": [{
      "kind": "TYPE_ENUM",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "response_type",
      "typeUrl": "type.googleapis.com/grpc.testing.PayloadType",
      "jsonName": "responseType"
    }, {
      "kind": "TYPE_INT32",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "response_size",
      "jsonName": "responseSize"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 3,
      "name": "payload",
      "typeUrl": "type.googleapis.com/grpc.testing.Payload",
      "jsonName": "payload"
    }, {
      "kind": "TYPE_BOOL",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 4,
      "name": "fill_username",
      "jsonName": "fillUsername"
    }, {
      "kind": "TYPE_BOOL",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 5,
      "name": "fill_oauth_scope",
      "jsonName": "fillOauthScope"
    }],
    "sourceContext": {
      "fileName": "grpc-interop.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "grpc.testing.SimpleResponse",
    "fields": [{
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "payload",
      "typeUrl": "type.googleapis.com/grpc.testing.Payload",
      "jsonName": "payload"
    }, {
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "username",
      "jsonName": "username"
    }, {
      "kind": "TYPE_STRING",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 3,
      "name": "oauth_scope",
      "jsonName": "oauthScope"
    }],
    "sourceContext": {
      "fileName": "grpc-interop.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "grpc.testing.StreamingInputCallRequest",
    "fields": [{
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "payload",
      "typeUrl": "type.googleapis.com/grpc.testing.Payload",
      "jsonName": "payload"
    }],
    "sourceContext": {
      "fileName": "grpc-interop.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "grpc.testing.StreamingInputCallResponse",
    "fields": [{
      "kind": "TYPE_INT32",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "aggregated_payload_size",
      "jsonName": "aggregatedPayloadSize"
    }],
    "sourceContext": {
      "fileName": "grpc-interop.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "grpc.testing.ResponseParameters",
    "fields": [{
      "kind": "TYPE_INT32",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "size",
      "jsonName": "size"
    }, {
      "kind": "TYPE_INT32",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 2,
      "name": "interval_us",
      "jsonName": "intervalUs"
    }],
    "sourceContext": {
      "fileName": "grpc-interop.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "grpc.testing.StreamingOutputCallRequest",
    "fields": [{
      "kind": "TYPE_ENUM",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "response_type",
      "typeUrl": "type.googleapis.com/grpc.testing.PayloadType",
      "jsonName": "responseType"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_REPEATED",
      "number": 2,
      "name": "response_parameters",
      "typeUrl": "type.googleapis.com/grpc.testing.ResponseParameters",
      "jsonName": "responseParameters"
    }, {
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 3,
      "name": "payload",
      "typeUrl": "type.googleapis.com/grpc.testing.Payload",
      "jsonName": "payload"
    }],
    "sourceContext": {
      "fileName": "grpc-interop.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }, {
    "name": "grpc.testing.StreamingOutputCallResponse",
    "fields": [{
      "kind": "TYPE_MESSAGE",
      "cardinality": "CARDINALITY_OPTIONAL",
      "number": 1,
      "name": "payload",
      "typeUrl": "type.googleapis.com/grpc.testing.Payload",
      "jsonName": "payload"
    }],
    "sourceContext": {
      "fileName": "grpc-interop.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }],
  "enums": [{
    "name": "grpc.testing.PayloadType",
    "enumvalue": [{
      "name": "COMPRESSABLE"
    }, {
      "name": "UNCOMPRESSABLE",
      "number": 1
    }, {
      "name": "RANDOM",
      "number": 2
    }],
    "sourceContext": {
      "fileName": "grpc-interop.proto"
    },
    "syntax": "SYNTAX_PROTO3"
  }],
  "documentation": {
    "rules": [{
      "selector": "grpc.testing",
      "description": "An integration test service that covers all the method signature permutations\nof unary/streaming requests/responses."
    }, {
      "selector": "grpc.testing.Payload",
      "description": "A block of data, to simply increase gRPC message size."
    }, {
      "selector": "grpc.testing.Payload.type",
      "description": "The type of data in body."
    }, {
      "selector": "grpc.testing.Payload.body",
      "description": "Primary contents of payload."
    }, {
      "selector": "grpc.testing.SimpleRequest",
      "description": "Unary request."
    }, {
      "selector": "grpc.testing.SimpleRequest.response_type",
      "description": "Desired payload type in the response from the server.\nIf response_type is RANDOM, server randomly chooses one from other formats."
    }, {
      "selector": "grpc.testing.SimpleRequest.response_size",
      "description": "Desired payload size in the response from the server.\nIf response_type is COMPRESSABLE, this denotes the size before compression."
    }, {
      "selector": "grpc.testing.SimpleRequest.payload",
      "description": "Optional input payload sent along with the request."
    }, {
      "selector": "grpc.testing.SimpleRequest.fill_username",
      "description": "Whether SimpleResponse should include username."
    }, {
      "selector": "grpc.testing.SimpleRequest.fill_oauth_scope",
      "description": "Whether SimpleResponse should include OAuth scope."
    }, {
      "selector": "grpc.testing.SimpleResponse",
      "description": "Unary response, as configured by the request."
    }, {
      "selector": "grpc.testing.SimpleResponse.payload",
      "description": "Payload to increase message size."
    }, {
      "selector": "grpc.testing.SimpleResponse.username",
      "description": "The user the request came from, for verifying authentication was\nsuccessful when the client expected it."
    }, {
      "selector": "grpc.testing.SimpleResponse.oauth_scope",
      "description": "OAuth scope."
    }, {
      "selector": "grpc.testing.StreamingInputCallRequest",
      "description": "Client-streaming request."
    }, {
      "selector": "grpc.testing.StreamingInputCallRequest.payload",
      "description": "Optional input payload sent along with the request."
    }, {
      "selector": "grpc.testing.StreamingInputCallResponse",
      "description": "Client-streaming response."
    }, {
      "selector": "grpc.testing.StreamingInputCallResponse.aggregated_payload_size",
      "description": "Aggregated size of payloads received from the client."
    }, {
      "selector": "grpc.testing.ResponseParameters",
      "description": "Configuration for a particular response."
    }, {
      "selector": "grpc.testing.ResponseParameters.size",
      "description": "Desired payload sizes in responses from the server.\nIf response_type is COMPRESSABLE, this denotes the size before compression."
    }, {
      "selector": "grpc.testing.ResponseParameters.interval_us",
      "description": "Desired interval between consecutive responses in the response stream in\nmicroseconds."
    }, {
      "selector": "grpc.testing.StreamingOutputCallRequest",
      "description": "Server-streaming request."
    }, {
      "selector": "grpc.testing.StreamingOutputCallRequest.response_type",
      "description": "Desired payload type in the response from the server.\nIf response_type is RANDOM, the payload from each response in the stream\nmight be of different types. This is to simulate a mixed type of payload\nstream."
    }, {
      "selector": "grpc.testing.StreamingOutputCallRequest.response_parameters",
      "description": "Configuration for each expected response message."
    }, {
      "selector": "grpc.testing.StreamingOutputCallRequest.payload",
      "description": "Optional input payload sent along with the request."
    }, {
      "selector": "grpc.testing.StreamingOutputCallResponse",
      "description": "Server-streaming response, as configured by the request and parameters."
    }, {
      "selector": "grpc.testing.StreamingOutputCallResponse.payload",
      "description": "Payload to increase response size."
    }, {
      "selector": "grpc.testing.PayloadType",
      "description": "The type of payload that should be returned."
    }, {
      "selector": "grpc.testing.PayloadType.COMPRESSABLE",
      "description": "Compressable text format."
    }, {
      "selector": "grpc.testing.PayloadType.UNCOMPRESSABLE",
      "description": "Uncompressable binary format."
    }, {
      "selector": "grpc.testing.PayloadType.RANDOM",
      "description": "Randomly chosen from all other formats defined in this enum."
    }, {
      "selector": "grpc.testing.TestService",
      "description": "A simple service to test the various types of RPCs and experiment with\nperformance with various types of payload."
    }, {
      "selector": "grpc.testing.TestService.EmptyCall",
      "description": "One empty request followed by one empty response."
    }, {
      "selector": "grpc.testing.TestService.UnaryCall",
      "description": "One request followed by one response.\nThe server returns the client payload as-is."
    }, {
      "selector": "grpc.testing.TestService.StreamingOutputCall",
      "description": "One request followed by a sequence of responses (streamed download).\nThe server returns the payload with client desired type and sizes."
    }, {
      "selector": "grpc.testing.TestService.StreamingInputCall",
      "description": "A sequence of requests followed by one response (streamed upload).\nThe server returns the aggregated size of client payload as the result."
    }, {
      "selector": "grpc.testing.TestService.FullDuplexCall",
      "description": "A sequence of requests with each request served by the server immediately.\nAs one request could lead to multiple responses, this interface\ndemonstrates the idea of full duplexing."
    }, {
      "selector": "grpc.testing.TestService.HalfDuplexCall",
      "description": "A sequence of requests followed by a sequence of responses.\nThe server buffers all the client requests and then serves them in order. A\nstream of responses are returned to the client when the server starts with\nfirst request."
    }]
  },
  "http": {
  },
  "quota": {
  },
  "authentication": {
  },
  "usage": {
    "rules": [{
      "selector": "grpc.testing.TestService.EmptyCall",
      "allowUnregisteredCalls": true
    }, {
      "selector": "grpc.testing.TestService.UnaryCall",
      "allowUnregisteredCalls": true
    }, {
      "selector": "grpc.testing.TestService.StreamingOutputCall",
      "allowUnregisteredCalls": true
    }, {
      "selector": "grpc.testing.TestService.StreamingInputCall",
      "allowUnregisteredCalls": true
    }, {
      "selector": "grpc.testing.TestService.FullDuplexCall",
      "allowUnregisteredCalls": true
    }, {
      "selector": "grpc.testing.TestService.HalfDuplexCall",
      "allowUnregisteredCalls": true
    }]
  },
  "endpoints": [{
    "name": "${ENDPOINT_SERVICE}"
  }],
  "configVersion": 3,
  "control": {
    "environment": "servicecontrol.googleapis.com"
  },
  "systemParameters": {
  }
}`
)
