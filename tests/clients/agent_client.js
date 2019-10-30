/**
 * Copyright 2019 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

var grpc = require('grpc');
var protoLoader = require('@grpc/proto-loader');

var PROTO_PATH = __dirname + '/../../api/agent/agent_service.proto';
var packageDefinition = protoLoader.loadSync(
  PROTO_PATH, {
    keepCase: true,
    longs: String,
    enums: String,
    defaults: true,
    oneofs: true,
    // includeDirs: [__dirname + '/proto/protobuf', __dirname + '/proto/googleapis'],
  });

var agent_proto = grpc.loadPackageDefinition(packageDefinition).google.api_proxy.agent;

var client = new agent_proto.AgentService('localhost:8790',
  grpc.credentials.createInsecure());


client.GetAccessToken({}, undefined, function(err, res) {
  if (err) {
    console.log("Error: ", err);
  } else {
    console.log("Result: ", res);
  }
});
