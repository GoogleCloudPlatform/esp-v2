var grpc = require('grpc');
var protoLoader = require('@grpc/proto-loader');

var PORT = 8082
const path = require('path');
const PROTO_PATH = path.join(__dirname, '/proto/bookstore.proto');

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
  console.log(call.metadata);
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
  console.log(call.metadata);
  if (call.request.shelf.theme == '') {
    callback(null, {
      id: call.request.shelf.id,
      theme: 'New Shelf'
    });
  } else {
    callback(null, {
      id: call.request.shelf.id,
      theme: call.request.shelf.theme,
    });
  }
}

function getShelf(call, callback) {
  console.log(call.metadata);
  callback(null, {
      id: call.request.shelf,
      theme: 'Unknown Shelf'
  });
}

function deleteShelf(call, callback) {
  console.log(call.metadata);
  callback(null, {});
}

function listBooks(call, callback) {
  console.log(call.metadata);
  callback(null, {});
}

function createBook(call, callback) {
  console.log(call.metadata);
  callback(null, {
      id: call.request.shelf,
      title: 'New Book'
  });
}

function getBook(call, callback) {
  console.log(call.metadata);
  callback(null, {
    id: call.request.book,
    title: 'Unknown Book'
  });
}

function deleteBook(call, callback) {
  console.log(call.metadata);
  callback(null, {});
}

function testDecorator(f) {
  return function(call, callback) {
    var testValues = call.metadata.get('x-grpc-test');
    var firstTestValue = undefined;
    if (testValues != undefined && testValues.length > 0) {
      firstTestValue = testValues[0];
    }
    // Add more gRPC statuses as needed.
    switch (firstTestValue) {
    case 'ABORTED':
      callback({
        code: grpc.status.ABORTED,  // 10
        message: 'ABORTED',
      });
      break;
    case 'INTERNAL':
      callback({
        code: grpc.status.INTERNAL,  // 13
        message: 'INTERNAL',
      });
      break;
    case 'DATA_LOSS':
      callback({
        code: grpc.status.DATA_LOSS, // 15
        message: 'DATA_LOSS',
      });
      break;
    default:
      f(call, callback);
    }
  };
}

/**
 * Starts an RPC server that receives requests for the Greeter service at the
 * sample server port
 */
function main() {
  var server = new grpc.Server();
  server.addService(bookstore_proto.Bookstore.service, {
    ListShelves: testDecorator(listShelves),
    CreateShelf: testDecorator(createShelf),
    GetShelf: testDecorator(getShelf),
    DeleteShelf: testDecorator(deleteShelf),
    ListBooks: testDecorator(listBooks),
    CreateBook: testDecorator(createBook),
    GetBook: testDecorator(getBook),
    DeleteBook: testDecorator(deleteBook),
  });
  console.log(`listening on port ${PORT}`);
  server.bind(`0.0.0.0:${PORT}`, grpc.ServerCredentials.createInsecure());
  server.start();
}

main();
