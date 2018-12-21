package client

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"io"

	"github.com/golang/protobuf/proto"
)

// EncodeGrpcWebRequestBody returns an encoded reader given a request message.
func EncodeGrpcWebRequestBody(msg proto.Message) io.Reader {
	data, _ := proto.Marshal(msg)
	lengthPrefix := []byte{0, 0, 0, 0, 0}
	binary.BigEndian.PutUint32(lengthPrefix[1:], uint32(len(data)))
	var buf bytes.Buffer
	buf.Write(lengthPrefix)
	buf.Write(data)

	b := make([]byte, base64.StdEncoding.EncodedLen(buf.Len()))
	base64.StdEncoding.Encode(b, buf.Bytes())
	return bytes.NewReader(b)
}

// TODO(kyuc): add DecodeGrpcWebResponseBody()
