var grpc = require('grpc');
var protoLoader = require('@grpc/proto-loader');

// This port is used by the backend settings in envoy.yaml
var PORT = 8082;
const path = require('path');
const PROTO_PATH = path.join(__dirname, '/proto/bookstore.proto');

var packageDefinition = protoLoader.loadSync(PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

var bookstore_proto =
    grpc.loadPackageDefinition(packageDefinition).endpoints.examples.bookstore;

// The bookstore example uses a simple, in-memory database
// for illustrative purposes only.
var bookstoreDatabase = {
  100: {id: 100, theme: 'Kids', books: {1001: {id: 1001, title: 'Alphabet'}}},
  200: {
    id: 200,
    theme: 'Classic',
    books: {2001: {id: 2001, title: 'Hamlet', author: 'Shakspeare'}}
  }
};

function listShelves(call, callback) {
  console.log(call.metadata);
  var found = [];
  for (var key in bookstoreDatabase) {
    found.push(
      {id: bookstoreDatabase[key].id, theme: bookstoreDatabase[key].theme});
  }
  callback(null, {shelves: found});
}

function createShelf(call, callback) {
  console.log(call.metadata);
  var shelf = {id: call.request.shelf.id, theme: call.request.shelf.theme};
  callback(null, shelf);
}

function getShelf(call, callback) {
  console.log(call.metadata);
  var s = bookstoreDatabase[call.request.shelf];
  callback(null, {id: s.id, theme: s.theme});
}

function deleteShelf(call, callback) {
  console.log(call.metadata);
  callback(null, {});
}

function listBooks(call, callback) {
  console.log(call.metadata);
  var found = [];
  if (call.request.shelf in bookstoreDatabase){
    books = bookstoreDatabase[call.request.shelf].books;
    for (var key in books) {
      found.push({
        id:books[key].id, author:books[key].author, title:books[key].title
      });
    }
  }
  callback(null, {books: found});
}

function createBook(call, callback) {
  console.log(call.metadata);
  if (!(call.request.shelf in bookstoreDatabase)) {
    bookstoreDatabase[call.request.shelf] = {id: call.request.shelf, books:{}};
  }
  var book = {id: call.request.book.id, title: call.request.book.title, author:call.request.book.author};
  bookstoreDatabase[call.request.shelf].books[call.request.book.id] = book
  callback(null, book);
}

function getBook(call, callback) {
  console.log(call.metadata);
  if (!(call.request.shelf in bookstoreDatabase) || !(call.request.book in bookstoreDatabase[call.request.shelf].books)) {
    callback({
          code: grpc.status.NOT_FOUND,  // 5
          message: 'NOT_FOUND',
        });
    return;
  }
  callback(
      null, bookstoreDatabase[call.request.shelf].books[call.request.book]);
}

function deleteBook(call, callback) {
  console.log(call.metadata);
  if ((call.request.shelf in bookstoreDatabase) && (call.request.book in bookstoreDatabase[call.request.shelf].books)) {
    delete bookstoreDatabase[call.request.shelf].books[call.request.book]
  }
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
          code: grpc.status.DATA_LOSS,  // 15
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
  if (process.argv.length >= 3) {
    PORT = parseInt(process.argv[2], 10);
    if (isNaN(PORT) || PORT < 1024 || PORT > 65535) {
      console.log(`port ${process.argv[2]} should be integer between 1024-65535`);
      process.exit(1);
    }
  }
  console.log(`listening on port ${PORT}`);
  server.bind(`0.0.0.0:${PORT}`, grpc.ServerCredentials.createInsecure());
  server.start();
}

main();
