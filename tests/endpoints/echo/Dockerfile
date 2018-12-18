# docker build -f tests/endpoints/echo/Dockerfile -t echo .
# docker run --rm -it -p 8082:8082 echo

FROM golang:1.8.5-jessie

# create a working directory
WORKDIR /go/src/server

# add source code
ADD tests/endpoints/echo/server src/server

ADD vendor/github.com/gorilla/mux /go/src/github.com/gorilla/mux

# run main.go
CMD ["go", "run", "src/server/app.go"]