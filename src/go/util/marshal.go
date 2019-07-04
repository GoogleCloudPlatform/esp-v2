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

package util

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"google.golang.org/genproto/googleapis/api/annotations"

	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
	sm "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

// Helper to convert Json string to protobuf.Any.
type FuncResolver func(url string) (proto.Message, error)

func (fn FuncResolver) Resolve(url string) (proto.Message, error) {
	return fn(url)
}

var Resolver = FuncResolver(func(url string) (proto.Message, error) {
	switch url {
	case "type.googleapis.com/google.api.servicemanagement.v1.ConfigFile":
		return new(sm.ConfigFile), nil
	case "type.googleapis.com/google.api.HttpRule":
		return new(annotations.HttpRule), nil
	case "type.googleapis.com/google.protobuf.BoolValue":
		return new(types.BoolValue), nil
	case "type.googleapis.com/google.api.Service":
		return new(conf.Service), nil
	default:
		return nil, fmt.Errorf("unexpected protobuf.Any with url: %s", url)
	}
})

func MessageToStruct(msg proto.Message) (*types.Struct, error) {
	if msg == nil {
		return nil, errors.New("nil message")
	}

	m := &jsonpb.Marshaler{
		OrigName:    true,
		AnyResolver: Resolver,
	}
	buf := &bytes.Buffer{}
	if err := m.Marshal(buf, msg); err != nil {
		return nil, err
	}

	pbs := &types.Struct{}
	u := &jsonpb.Unmarshaler{
		AnyResolver: Resolver,
	}
	if err := u.Unmarshal(buf, pbs); err != nil {
		return nil, err
	}

	return pbs, nil
}
