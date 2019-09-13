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

// Various utility functions for testing gRPC-Web.
//
// Implementations are based on the protocol specificication.
// For details refer to:
// https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-WEB.md#protocol-differences-vs-grpc-over-http2

package client

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"strings"

	"github.com/golang/protobuf/proto"
)

// GRPCWebTrailer represents key-value pairs in gRPC-Web trailer.
type GRPCWebTrailer map[string]string

// EncodeGRPCWebRequestBody returns an encoded reader given a request message.
func EncodeGRPCWebRequestBody(message proto.Message) io.Reader {
	data, _ := proto.Marshal(message)
	lengthPrefix := []byte{0, 0, 0, 0, 0}
	binary.BigEndian.PutUint32(lengthPrefix[1:], uint32(len(data)))
	var buf bytes.Buffer
	buf.Write(lengthPrefix)
	buf.Write(data)

	b := make([]byte, base64.StdEncoding.EncodedLen(buf.Len()))
	base64.StdEncoding.Encode(b, buf.Bytes())
	return bytes.NewReader(b)
}

// DecodeGRPCWebResponseBody returns a decoded message and a trailer given a request body.
func DecodeGRPCWebResponseBody(body io.Reader) ([]byte, GRPCWebTrailer, error) {
	content, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, nil, err
	}

	decodedBytes := base64DecodeBytes(content)
	data := bytes.NewReader(decodedBytes)

	var msg []byte

	// Reading the first message.
	payloadBytes, isTrailer, err := readPayloadBytes(data)
	if err != nil {
		return nil, nil, err
	}

	// Trailer must be the last message based on the protocol spec.
	if isTrailer {
		return nil, makeTrailer(payloadBytes), nil
	}

	msg = payloadBytes

	// Reading the second message.
	payloadBytes, isTrailer, err = readPayloadBytes(data)
	if err != nil {
		return msg, nil, err
	} else {
		return msg, makeTrailer(payloadBytes), nil
	}
}

func makeTrailer(trailerBytes []byte) GRPCWebTrailer {
	trailer := make(GRPCWebTrailer)
	// key-value pairs are delimited by \r\n
	for _, keyValue := range strings.Split(string(trailerBytes), "\r\n") {
		kv := strings.Split(strings.TrimSpace(keyValue), ":")
		if len(kv) == 2 {
			trailer[kv[0]] = kv[1]
		}
	}
	return trailer
}

// base64DecodeBytes decodes base 64 encoded byte slice.
func base64DecodeBytes(encodedBytes []byte) []byte {
	decodedLen := base64.StdEncoding.DecodedLen(len(encodedBytes))
	decodedBytes := make([]byte, decodedLen)
	numBytesRead := 0
	numBytesWritten := 0
	for numBytesRead+3 < len(encodedBytes) {
		n, _ := base64.StdEncoding.Decode(decodedBytes[numBytesWritten:], encodedBytes[numBytesRead:])
		numBytesWritten += n
		numBytesRead += base64.StdEncoding.EncodedLen(n)
	}
	return decodedBytes[:numBytesWritten]
}

// isTrailer returns true if the gRPCFrameByte's most significant bit is 1.
func isTrailer(gRPCFrameByte byte) bool {
	return gRPCFrameByte&(1<<7) == (1 << 7)
}

// readPayloadBytes returns the next payload bytes in the data.
func readPayloadBytes(data io.Reader) ([]byte, bool, error) {
	lengthPrefix := []byte{0, 0, 0, 0, 0}
	readCount, err := data.Read(lengthPrefix)

	if err != nil {
		return nil, false, err
	}

	if readCount != 5 {
		return nil, false, errors.New("malformed data: not enough data to read length prefix")
	}

	payloadLength := binary.BigEndian.Uint32(lengthPrefix[1:])
	payloadBytes := make([]byte, payloadLength)
	readCount, err = data.Read(payloadBytes)
	return payloadBytes, isTrailer(lengthPrefix[0]), nil
}
