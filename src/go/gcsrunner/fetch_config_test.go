// Copyright 2019 Google LLC
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

package gcsrunner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"google.golang.org/api/option"

	google_oauth "golang.org/x/oauth2/google"
)

const (
	fetchInitialInterval = time.Millisecond
	fetchTimeout         = time.Second
)

var (
	opts = FetchConfigOptions{
		BucketName:                    "bucket",
		ConfigFileName:                "file",
		Port:                          "1234",
		WriteFilePath:                 "path/file",
		FetchGCSObjectInitialInterval: fetchInitialInterval,
		FetchGCSObjectTimeout:         fetchTimeout,
	}
)

type mockObjectReader struct {
	newReaderReturns       io.Reader
	newReaderErr           error
	newReaderCallCount     int
	passAfterMultipleCalls bool
}

func (m *mockObjectReader) readObject(ctx context.Context, _ getObjectRequest) (io.Reader, error) {
	if deadline, ok := ctx.Deadline(); ok {
		if time.Now().After(deadline) {
			return nil, context.DeadlineExceeded
		}
	}
	m.newReaderCallCount++
	if m.passAfterMultipleCalls && m.newReaderCallCount > 2 {
		return m.newReaderReturns, nil
	}
	return m.newReaderReturns, m.newReaderErr
}

type mockFile struct {
	closeCalled bool
	writeErr    error
	gotString   string
}

func (m *mockFile) Close() error {
	m.closeCalled = true
	return nil
}

func (m *mockFile) WriteString(s string) (int, error) {
	m.gotString = s
	return len(s), m.writeErr
}

func TestReadBytes(t *testing.T) {
	output := []byte("some output")
	goodFindDefaultCredentials := func(context.Context, ...string) (*google_oauth.Credentials, error) {
		return nil, nil
	}
	badFindDefaultCredentials := func(context.Context, ...string) (*google_oauth.Credentials, error) {
		return nil, fmt.Errorf("default creds failure")
	}
	testCases := []struct {
		name                    string
		wantErr                 bool
		wantMultipleReaderCalls bool
		passMultipleReaderCalls bool
		wantBytes               []byte
		credsFunc               func(context.Context, ...string) (*google_oauth.Credentials, error)
		newClientErr            error
		newReaderReturns        io.Reader
		newReaderErr            error
	}{
		{
			name:      "error getting default creds",
			wantErr:   true,
			credsFunc: badFindDefaultCredentials,
		},
		{
			name:         "error creating client with creds",
			wantErr:      true,
			newClientErr: fmt.Errorf("client error"),
			credsFunc:    goodFindDefaultCredentials,
		},
		{
			name:                    "timeout retrying NewReader()",
			wantErr:                 true,
			wantMultipleReaderCalls: true,
			newReaderErr:            fmt.Errorf("reader error"),
			credsFunc:               goodFindDefaultCredentials,
		},
		{
			name:             "success",
			wantBytes:        output,
			newReaderReturns: bytes.NewReader(output),
			credsFunc:        goodFindDefaultCredentials,
		},
		{
			name:                    "success with retries",
			wantBytes:               output,
			wantMultipleReaderCalls: true,
			passMultipleReaderCalls: true,
			newReaderReturns:        bytes.NewReader(output),
			newReaderErr:            fmt.Errorf("temporary reader error should resolve"),
			credsFunc:               goodFindDefaultCredentials,
		},
	}

	for _, tc := range testCases {
		findDefaultCredentials = tc.credsFunc
		m := &mockObjectReader{
			newReaderReturns:       tc.newReaderReturns,
			newReaderErr:           tc.newReaderErr,
			passAfterMultipleCalls: tc.passMultipleReaderCalls,
		}
		newStorageObjectReader = func(ctx context.Context, opts ...option.ClientOption) (storageObjectReader, error) {
			if tc.newClientErr != nil {
				return nil, tc.newClientErr
			}
			return m, nil
		}
		b, err := readBytes(opts)
		if err != nil != tc.wantErr {
			t.Errorf("readBytes() wanted %v!=nil to be %v", err, tc.wantErr)
		}
		if m.newReaderCallCount > 1 != tc.wantMultipleReaderCalls {
			t.Errorf("NewReader() wanted %d>1 to be %v", m.newReaderCallCount, tc.wantMultipleReaderCalls)
		}
		if !tc.wantErr {
			if diff := bytes.Compare(b, tc.wantBytes); diff != 0 {
				t.Errorf("readBytes() = %s, _, want %s", string(b), string(tc.wantBytes))
			}
		}
	}
}

func TestWriteFile(t *testing.T) {
	testCases := []struct {
		name          string
		wantErr       bool
		input         []byte
		wantOutput    string
		createWant    string
		createReturns *mockFile
		createErr     error
	}{
		{
			name:          "failure to create file",
			wantErr:       true,
			createWant:    opts.WriteFilePath,
			createErr:     fmt.Errorf("create error"),
			createReturns: &mockFile{},
		},
		{
			name:       "failure to write file",
			wantErr:    true,
			createWant: opts.WriteFilePath,
			createReturns: &mockFile{
				writeErr: fmt.Errorf("write error"),
			},
		},
		{
			name:          "success, no port replacement",
			input:         []byte(`{"some-key":"some-value"}`),
			wantOutput:    `{"some-key":"some-value"}`,
			createWant:    opts.WriteFilePath,
			createReturns: &mockFile{},
		},
		{
			name:          "success, with port replacement",
			input:         []byte(`{"some-key":REPLACE_PORT_NUMBER}`),
			wantOutput:    `{"some-key":1234}`,
			createWant:    opts.WriteFilePath,
			createReturns: &mockFile{},
		},
	}
	for _, tc := range testCases {
		osCreate = func(got string) (file, error) {
			if got != tc.createWant {
				t.Errorf("osCreate called with %s, want %s", got, tc.createWant)
			}
			return tc.createReturns, tc.createErr
		}
		err := writeFile(tc.input, opts)
		if err != nil != tc.wantErr {
			t.Errorf("writeFile() wanted %v!=nil to be %v", err, tc.wantErr)
		}
		if !tc.createReturns.closeCalled {
			t.Errorf("Created file was not closed")
		}
		if !tc.wantErr {
			if tc.createReturns.gotString != tc.wantOutput {
				t.Errorf("WriteString() called with %s, want %s", tc.createReturns.gotString, tc.wantOutput)
			}
		}
	}
}
