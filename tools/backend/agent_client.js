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