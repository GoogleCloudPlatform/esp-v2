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

// EncodeGrpcWebRequestBody returns an encoded reader given a request message.
func EncodeGrpcWebRequestBody(message proto.Message) io.Reader {
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

// DecodeGrpcWebResponseBody returns a decoded message and a trailer given a request body.
func DecodeGrpcWebResponseBody(body io.Reader) ([]byte, GRPCWebTrailer, error) {
	content, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, nil, err
	}

	decodedBytes := base64DecodeBytes(content)
	data := bytes.NewReader(decodedBytes)

	// Reading the response message.
	_, message, err := readPayloadBytes(data)
	if err != nil {
		return nil, nil, err
	}

	// Reading the trailer.
	lengthPrefix, trailerBytes, err := readPayloadBytes(data)
	trailer := make(GRPCWebTrailer)
	if !isTrailer(lengthPrefix[0]) {
		return nil, nil, errors.New("malformed gRPC-Web response")
	}

	// key-value pairs are delimited by \r\n
	for _, keyValue := range strings.Split(string(trailerBytes), "\r\n") {
		kv := strings.Split(strings.TrimSpace(keyValue), ":")
		if len(kv) == 2 {
			trailer[kv[0]] = kv[1]
		}
	}

	return message, trailer, nil
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
func readPayloadBytes(data io.Reader) ([]byte, []byte, error) {
	lengthPrefix := []byte{0, 0, 0, 0, 0}
	readCount, err := data.Read(lengthPrefix)

	if err != nil {
		return nil, nil, err
	}

	if readCount != 5 {
		return nil, nil, errors.New("malformed data: not enough data to read length prefix")
	}

	payloadLength := binary.BigEndian.Uint32(lengthPrefix[1:])
	trailerBytes := make([]byte, payloadLength)
	readCount, err = data.Read(trailerBytes)
	return lengthPrefix, trailerBytes, nil
}
