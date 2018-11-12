var grpc = require('grpc');
var protoLoader = require('@grpc/proto-loader');

var PORT = 8082
const path = require('path');
const PROTO_PATH = path.join(__dirname, '../proto/bookstore.proto');

var packageDefinition = protoLoader.loadSync(
  PROTO_PATH, {
    keepCase: true,
    longs: String,
    enums: String,
    defaults: true,
    oneofs: true,
  });

var bookstore_proto = grpc.loadPackageDefinition(packageDefinition).endpoints.examples.bookstore;

function listShelves(call, callback) {
  console.log(call.metadata)
  callback(null, {
    shelves: [{
      id: '123',
      theme: 'Shakspeare'
    },{
      id: '124',
      theme: 'Hamlet'
    }]
  });
}

function createShelf(call, callback) {
  callback(null, {});
}

function getShelf(call, callback) {
  console.log(call.metadata)
  callback(null, {
      id: call.request.shelf,
      theme: 'Unknown Book'
  });
}

function deleteShelf(call, callback) {
  callback(null, {});
}

function listBooks(call, callback) {
  callback(null, {});
}

function createBook(call, callback) {
  callback(null, {});
}

function getBook(call, callback) {
  callback(null, {});
}

function deleteBook(call, callback) {
  callback(null, {});
}

/**
 * Starts an RPC server that receives requests for the Greeter service at the
 * sample server port
 */
function main() {
  var server = new grpc.Server();
  server.addService(bookstore_proto.Bookstore.service, {
    ListShelves: listShelves,
    CreateShelf: createShelf,
    GetShelf: getShelf,
    DeleteShelf: deleteShelf,
    ListBooks: listBooks,
    CreateBook: createBook,
    GetBook: getBook,
    DeleteBook: deleteBook,
  });
  console.log(`listening on port ${PORT}`)
  server.bind(`0.0.0.0:${PORT}`, grpc.ServerCredentials.createInsecure());
  server.start();
}

main();
