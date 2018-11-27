const express = require('express')
const app = express()
const port = 8080

app.get('/test', (request, response) => {
  console.log(request.headers)
  response.send('Headers: ' + JSON.stringify(request.headers) + "\n")
})

app.listen(port, (err) => {
  if (err) {
    return console.log('something bad happened', err)
  }
  console.log(`server is listening on ${port}`)
})
