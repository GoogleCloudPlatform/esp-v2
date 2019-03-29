// Copyright 2018 Google Cloud Platform Proxy Authors
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
//
////////////////////////////////////////////////////////////////////////////////
//
// A Google Cloud Endpoints example implementation of a simple bookstore API.

'use strict';

var express = require('express');
var bodyParser = require('body-parser');
var swaggerTools = require('swagger-tools');


/**
 * @typedef {Object} InitializationOptions
 * @property {Boolean} log Log incoming requests.
 * @property {Object} swagger Swagger document object.
 */

/**
 * Creates an Express.js application which implements a Bookstore
 * API defined in `swagger.json`.
 *
 * @param {InitializationOptions} options Application initialization options.
 * @return {!express.Application} An initialized Express.js application.
 *
 * If no options are provided, defaults are:
 *     {
 *       log: true,
 *     }
 */
function bookstore(options) {
  options = options || {
    log: true,
  };

  var app = express();
  app.use(bodyParser.json());

  // Serve application version for tests to ensure that
  // bookstore was deployed correctly.
  app.get('/version', function(req, res) {
    res.set('Content-Type', 'application/json');
    res.status(200).send({
      version: '${VERSION}'
    });
  });

  var echoCount = 0;

  // Echo method for stress test.
  function echo(req, res) {
    echoCount += 1;
    res.status(200).json(req.body);
  }

  app.all('/echo', echo);
  app.get('/echo/auth', echo);
  app.post('/echo/auth', echo);

  // Show number of echo requests
  setInterval(function() {
    if (echoCount > 0) {
      var date = new Date();
      var timestamp = date.getHours() + ":" + date.getMinutes() + ":" +
        date.getSeconds() + ":" + date.getMilliseconds();
      console.log(timestamp + ' Echo requests received: ', echoCount);
    }
  }, 1000);

  // Install tracing middleware.
  if (options.log === true) {
    app.use(function(req, res, next) {
      console.log(req.method, req.originalUrl);
      next();
    });
  }


  if (options.swagger) {
    // Initialize the Swagger UI middleware.
    swaggerTools.initializeMiddleware(options.swagger, function(middleware) {
      // Serve the Swagger documents and Swagger UI.
      app.use(middleware.swaggerUi());
    });
  }

  // The bookstore example uses a simple, in-memory database
  // for illustrative purposes only.
  var bookstoreDatabase = {
    shelves: {},
    id: 0
  };

  function createShelf(theme) {
    var id = ++bookstoreDatabase.id;
    var shelf = {
      name: 'shelves/' + id,
      theme: theme,
      books: {}
    };
    bookstoreDatabase.shelves[shelf.name] = shelf;
    return shelf;
  }

  function getShelf(name) {
    return bookstoreDatabase.shelves[name];
  }

  function deleteShelf(name) {
    var shelf = bookstoreDatabase.shelves[name];
    if (shelf === undefined) {
      return undefined;
    }
    delete bookstoreDatabase.shelves[name];
    return shelf;
  }

  function createBook(shelfName, author, title) {
    var shelf = getShelf(shelfName);
    if (shelf === undefined) {
      return undefined;
    }
    var id = ++bookstoreDatabase.id;
    var book = {
      name: shelf.name + '/books/' + id,
      author: author,
      title: title
    };
    shelf.books[book.name] = book;
    return book;
  }

  function getBook(shelfName, bookName) {
    var shelf = getShelf(shelfName);
    if (shelf === undefined) {
      return undefined;
    }
    return shelf.books[bookName];
  }

  function deleteBook(shelfName, bookName) {
    var shelf = getShelf(shelfName);
    if (shelf === undefined) {
      return undefined;
    }
    var book = shelf.books[bookName];
    if (book === undefined) {
      return undefined;
    }
    delete shelf.books[bookName];
    return book;
  }

  /**
   * @typedef {Object} UserInfo
   * @property {String} id An auth provider defined user identity.
   * @property {String} email An authenticated user email address.
   * @property {Object} consumer_id A consumer identifier (currently unused).
   */

  /**
   * Extracts from the request headers user information attached
   * by Endpoints.
   *
   * An example of the result (all properties are optional):
   * {
   *   id: 'xxx',
   *   email: 'xxx',
   *   consumer_id: 'xxx'
   * }
   *
   * @param {!express.Request} req An Express.js Request object.
   * @return {!UserInfo} An authenticated user information.
   */
  function getUserInfo(req) {
    var header = req.get('X-Endpoint-API-UserInfo');
    if (header) {
      return JSON.parse(new Buffer(header, 'base64').toString());
    }
    return {};
  }

  function error(res, status, message) {
    res.status(status).json({
      error: status,
      message: message
    });
  }

  app.get('/', function(req, res) {
    res.status(200).sendFile('index.html', {
      root: __dirname
    });
  });

  app.get('/restricted', function(req, res) {
    res.status(200).json({
      msg: 'restricted'
    });
  });

  app.get('/quota_read', function(req, res) {
    res.status(200).json({
      msg: 'ok'
    });
  });

  app.get('/shelves', function(req, res) {
    var shelves = bookstoreDatabase.shelves;
    var result = [];
    for (var name in shelves) {
      var shelf = shelves[name];
      result.push({
        name: shelf.name,
        theme: shelf.theme
      });
    }
    res.status(200).json({
      shelves: result
    });
  });

  app.post('/shelves', function(req, res) {
    var shelfRequest = req.body;
    if (shelfRequest === undefined) {
      return error(res, 400, 'Missing request body.');
    }
    if (shelfRequest.theme === undefined) {
      return error(res, 400, 'Shelf resource is missing required \'theme\'.');
    }

    var shelf = createShelf(shelfRequest.theme);
    res.status(200).json({
      name: shelf.name,
      theme: shelf.theme
    });
  });

  app.get('/shelves/:shelf', function(req, res) {
    var shelf = getShelf('shelves/' + req.params.shelf);
    if (shelf === undefined) {
      return error(res, 404, 'Cannot find shelf shelves/' + req.params.shelf);
    }

    res.status(200).json({
      name: shelf.name,
      theme: shelf.theme
    });
  });

  app.delete('/shelves/:shelf', function(req, res) {
    var shelf = deleteShelf('shelves/' + req.params.shelf);
    if (shelf === undefined) {
      return error(res, 404, 'Cannot find shelf shelves/' + req.params.shelf);
    }

    res.status(204).end();
  });

  app.get('/shelves/:shelf/books', function(req, res) {
    var shelf = getShelf('shelves/' + req.params.shelf);
    if (shelf === undefined) {
      return error(res, 404, 'Cannot find shelf shelves/' + req.params.shelf);
    }

    var books = shelf.books;
    var result = [];
    for (var name in books) {
      var book = books[name];
      result.push({
        name: book.name,
        author: book.author,
        title: book.title
      });
    }

    res.status(200).json({
      books: result
    });
  });

  app.post('/shelves/:shelf/books/', function(req, res) {
    var bookRequest = req.body;
    if (bookRequest === undefined) {
      return error(res, 400, 'Missing request body.');
    }
    if (bookRequest.author === undefined) {
      return error(res, 400, 'Book resource is missing required \'author\'.');
    }
    if (bookRequest.title === undefined) {
      return error(res, 400, 'Book resource is missing required \'title\'.');
    }
    var shelf = getShelf('shelves/' + req.params.shelf);
    if (shelf === undefined) {
      return error(res, 404, 'Cannot find shelf shelves/' + req.params.shelf);
    }
    var book = createBook('shelves/' + req.params.shelf,
                          bookRequest.author,
                          bookRequest.title);
    res.status(200).json({
      name: book.name,
      author: book.author,
      title: book.title
    });
  });

  app.get('/shelves/:shelf/books/:book', function(req, res) {
    var book = getBook(
      'shelves/' + req.params.shelf,
      'shelves/' + req.params.shelf + '/books/' + req.params.book);
    if (book === undefined) {
      return error(res, 404, 'Cannot find book ' +
        'shelves/' + req.params.shelf + '/books/' + req.params.book);
    }
    res.status(200).json({
      name: book.name,
      author: book.author,
      title: book.title
    });
  });

  app.delete('/shelves/:shelf/books/:book', function(req, res) {
    var book = deleteBook(
        'shelves/' + req.params.shelf,
        'shelves/' + req.params.shelf + '/books/' + req.params.book);
    if (book === undefined) {
      return error(
          res, 404, 'Cannot find book ' +
          'shelves/' + req.params.shelf + '/books/' + req.params.book);
    }
    res.status(204).end();
  });

  // Initialize bookstoreDatabase
  (function() {
    // Initialize Bookstore
    var fiction = createShelf('Fiction');
    var fantasy = createShelf('Fantasy');

    createBook(fiction.name, 'Neal Stephenson', 'REAMDE');
    createBook(fantasy.name, 'George R.R. Martin', 'A Game of Thrones');
  })();

  return app;
}

function loadSwagger(port) {
  var swagger = require('./bookstore_swagger_template.json');

  // When not running on Google App Engine and if the swagger.host
  // has its default value (which is an invalid value), update the
  // host to localhost and change schemes to [ 'http' ] to enable the
  // Swagger UI.

  var gae = 'Google App Engine/';
  var ssw = process.env.SERVER_SOFTWARE;

  if ((!ssw || ssw.substring(0, gae.length) !== gae) &&
    swagger.host === '${MY_PROJECT_ID}.appspot.com') {
    swagger.host = 'localhost:' + port;
    swagger.schemes = [ 'http' ];
  }
  return swagger;
}

// If this file is imported as a module, export the `bookstore` function.
// Otherwise, if `bookstore.js` is executed as a main program, start
// the server and listen on a port.
if (module.parent) {
  module.exports = bookstore;
} else {
  var port = process.env.PORT || '8080';
  var swagger = loadSwagger(port);
  var server = bookstore({ log: false, swagger: swagger }).listen(port, '0.0.0.0',
      function() {
        var host = server.address().address;
        var port = server.address().port;

        console.log('Bookstore listening at http://%s:%s', host, port);
      }
  );
}