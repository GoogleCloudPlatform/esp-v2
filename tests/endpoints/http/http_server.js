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

const express = require('express')
const app = express()
// This port is used by the backend settings in envoy.yaml
const port = 8082

app.get('/echo', (request, response) => {
  console.log(request.headers)
  response.send('Headers: ' + JSON.stringify(request.headers) + "\n")
})

app.post('/echo', (request, response) => {
  console.log(request.headers)
  response.send('Headers: ' + JSON.stringify(request.headers) + "\n")
})

app.get('/echo2', (request, response) => {
  console.log(request.headers)
  response.send('Headers: ' + JSON.stringify(request.headers) + "\n")
})

app.post('/echo2', (request, response) => {
  console.log(request.headers)
  response.send('Headers: ' + JSON.stringify(request.headers) + "\n")
})

app.listen(port, (err) => {
  if (err) {
    return console.log('something bad happened', err)
  }
  console.log(`server is listening on ${port}`)
})
