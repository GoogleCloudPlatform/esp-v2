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
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/google/go-cmp/cmp"
)

const (
	fetchInitialInterval = time.Millisecond
	fetchTimeout         = time.Second
)

type mockReader struct {
	readerReturns          io.Reader
	readerErr              error
	readerCallCount        int
	passAfterMultipleCalls bool
}

func (m *mockReader) Reader(ctx context.Context, _, _ string) (io.Reader, error) {
	if deadline, ok := ctx.Deadline(); ok {
		if time.Now().After(deadline) {
			return nil, context.DeadlineExceeded
		}
	}
	m.readerCallCount++
	if m.passAfterMultipleCalls && m.readerCallCount > 2 {
		return m.readerReturns, nil
	}
	return m.readerReturns, m.readerErr
}

type mockFile struct {
	closeCalled bool
	readErr     error
	writeErr    error
	content     []byte
}

func (m *mockFile) Close() error {
	m.closeCalled = true
	return nil
}

func (m *mockFile) Read(b []byte) (int, error) {
	if m.readErr != nil {
		return 0, m.readErr
	}
	return copy(b, m.content), io.EOF
}

func (m *mockFile) Write(b []byte) (int, error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	m.content = b
	return len(b), nil
}

func readerFactory(r *mockReader, err error, passAfterRetrying bool) func(context.Context, string) (gcsReader, error) {
	callCount := 0
	return func(context.Context, string) (gcsReader, error) {
		callCount++
		if passAfterRetrying && callCount > 2 {
			return r, nil
		}
		if err != nil {
			return nil, err
		}
		return r, err
	}
}

func defaultCredsReaderFactory(r *mockReader, err error, passAfterRetrying bool) func(context.Context) (gcsReader, error) {
	callCount := 0
	return func(context.Context) (gcsReader, error) {
		callCount++
		if passAfterRetrying && callCount > 2 {
			return r, nil
		}
		if err != nil {
			return nil, err
		}
		return r, err
	}
}

func TestReadBytes(t *testing.T) {
	optsNoSA := FetchConfigOptions{
		BucketName:                    "bucket",
		ConfigFileName:                "file",
		WriteFilePath:                 "path/file",
		FetchGCSObjectInitialInterval: fetchInitialInterval,
		FetchGCSObjectTimeout:         fetchTimeout,
	}
	optsSA := FetchConfigOptions{
		ServiceAccount:                "some-service@account",
		BucketName:                    "bucket",
		ConfigFileName:                "file",
		WriteFilePath:                 "path/file",
		FetchGCSObjectInitialInterval: fetchInitialInterval,
		FetchGCSObjectTimeout:         fetchTimeout,
	}
	output := []byte("some output")
	testError := fmt.Errorf("test error")
	testCases := []struct {
		name                               string
		wantErr                            error
		wantTimeoutErr                     bool
		opts                               FetchConfigOptions
		reader                             *mockReader
		readerErr                          error
		defaultCredsReader                 *mockReader
		defaultCredsReaderErr              error
		passAfterRetryingGCSClientCreation bool
	}{
		{
			name: "success",
			opts: optsSA,
			reader: &mockReader{
				readerReturns: &mockFile{content: output},
			},
			defaultCredsReaderErr: fmt.Errorf("expected defaultCredsReader not to be called"),
		},
		{
			name: "success with retries",
			opts: optsSA,
			reader: &mockReader{
				readerReturns:          &mockFile{content: output},
				passAfterMultipleCalls: true,
			},
			defaultCredsReaderErr: fmt.Errorf("expected defaultCredsReader not to be called"),
		},
		{
			name:      "success with defaultCreds",
			opts:      optsNoSA,
			readerErr: fmt.Errorf("expected reader not to be called"),
			defaultCredsReader: &mockReader{
				readerReturns: &mockFile{content: output},
			},
		},
		{
			name: "success retrying newGCS()",
			opts: optsSA,
			reader: &mockReader{
				readerReturns: &mockFile{content: output},
			},
			readerErr:                          testError,
			defaultCredsReaderErr:              fmt.Errorf("expected defaultCredsReader not to be called"),
			passAfterRetryingGCSClientCreation: true,
		},
		{
			name:      "success retrying newDefaultCredsGCS()",
			opts:      optsNoSA,
			readerErr: fmt.Errorf("expected reader not to be called"),
			defaultCredsReader: &mockReader{
				readerReturns: &mockFile{content: output},
			},
			defaultCredsReaderErr:              testError,
			passAfterRetryingGCSClientCreation: true,
		},

		{
			name:                  "timeout retrying newGCS()",
			wantTimeoutErr:        true,
			opts:                  optsSA,
			readerErr:             testError,
			defaultCredsReaderErr: fmt.Errorf("expected defaultCredsReader not to be called"),
		},
		{
			name:                  "timeout retrying newDefaultCredsGCS()",
			wantTimeoutErr:        true,
			opts:                  optsNoSA,
			readerErr:             fmt.Errorf("expected reader not to be called"),
			defaultCredsReaderErr: testError,
		},
		{
			name:           "timeout retrying client.Reader()",
			wantTimeoutErr: true,
			opts:           optsSA,
			reader: &mockReader{
				readerErr: fmt.Errorf("permanent error"),
			},
			defaultCredsReaderErr: fmt.Errorf("expected defaultCredsReader not to be called"),
		},
		{
			name:           "timeout retrying ReadAll()",
			wantTimeoutErr: true,
			opts:           optsSA,
			reader: &mockReader{
				readerReturns: &mockFile{readErr: fmt.Errorf("permmanent error")},
			},
			defaultCredsReaderErr: fmt.Errorf("expected defaultCredsReader not to be called"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			oldNewGCS := newGCS
			oldNewDefaultCredsGCS := newDefaultCredsGCS
			newGCS = readerFactory(tc.reader, tc.readerErr, tc.passAfterRetryingGCSClientCreation)
			newDefaultCredsGCS = defaultCredsReaderFactory(tc.defaultCredsReader, tc.defaultCredsReaderErr, tc.passAfterRetryingGCSClientCreation)
			defer func() {
				newGCS = oldNewGCS
				newDefaultCredsGCS = oldNewDefaultCredsGCS
			}()
			b, err := readBytes(tc.opts)
			if err != tc.wantErr {
				if tc.wantTimeoutErr {
					if _, ok := err.(*backoff.PermanentError); tc.wantTimeoutErr == ok {
						t.Errorf("readBytes() wanted %v=PermanentError to be %v", err, tc.wantTimeoutErr)
					}
				} else {
					t.Errorf("readBytes() = %v, wanted %v", err, tc.wantErr)
				}
			}
			if tc.wantErr == nil && !tc.wantTimeoutErr {
				if diff := cmp.Diff(string(output), string(b)); diff != "" {
					t.Errorf("readBytes() got unexpected value (-want/+got): %s", diff)
				}
			}
		})
	}
}

func TestWriteFile(t *testing.T) {
	opts := FetchConfigOptions{
		BucketName:                    "bucket",
		ConfigFileName:                "file",
		WriteFilePath:                 "path/file",
		FetchGCSObjectInitialInterval: fetchInitialInterval,
		FetchGCSObjectTimeout:         fetchTimeout,
	}
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
			name:          "success",
			input:         []byte(`{"some-key":"some-value"}`),
			wantOutput:    `{"some-key":"some-value"}`,
			createWant:    opts.WriteFilePath,
			createReturns: &mockFile{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			oldOSCreate := osCreate
			osCreate = func(got string) (io.WriteCloser, error) {
				if got != tc.createWant {
					t.Errorf("osCreate called with %s, want %s", got, tc.createWant)
				}
				return tc.createReturns, tc.createErr
			}
			defer func() {
				osCreate = oldOSCreate
			}()
			err := writeFile(tc.input, opts)
			if err != nil != tc.wantErr {
				t.Errorf("writeFile() wanted %v!=nil to be %v", err, tc.wantErr)

			}
			if !tc.createReturns.closeCalled {
				t.Errorf("Created file was not closed")
			}
			if !tc.wantErr {
				if diff := cmp.Diff(string(tc.createReturns.content), tc.wantOutput); diff != "" {
					t.Errorf("WriteString() unexpected output (-got,+want): %s", diff)
				}
			}
		})
	}
}
