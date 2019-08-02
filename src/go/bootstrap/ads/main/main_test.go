package main

import (
	"flag"
	"reflect"
	"strings"
	"testing"
	"time"

	"cloudesf.googlesource.com/gcpproxy/src/go/metadata"
	"cloudesf.googlesource.com/gcpproxy/src/go/util"
)

const (
	fakeFlagProjectId     = "fake-flag-project-id"
	fakeMetadataProjectId = "fake-metadata-project-id"
)

func TestDetermineProjectId(t *testing.T) {
	testData := []struct {
		desc       string
		flags      map[string]string
		runServer  bool
		wantError  string
		wantResult string
	}{
		{
			desc: "tracing_project_id not specified, but successfully discovered",
			flags: map[string]string{
				"non_gcp":            "false",
				"tracing_project_id": "",
			},
			runServer:  true,
			wantResult: fakeMetadataProjectId,
		},
		{
			desc: "tracing_project_id not specified, and non GCP runtime",
			flags: map[string]string{
				"non_gcp":            "true",
				"tracing_project_id": "",
			},
			runServer: false,
			wantError: "tracing_project_id was not specified and can not be fetched from GCP Metadata server on non-GCP runtime",
		},
		{
			desc: "tracing_project_id not specified, and error fetching from metadata server",
			flags: map[string]string{
				"non_gcp":            "false",
				"tracing_project_id": "",
			},
			runServer: false,
			wantError: " ", // Allow any error message, depends on underlying http client error
		},
		{
			desc: "tracing_project_id specified, successfully used",
			flags: map[string]string{
				"non_gcp":            "false",
				"tracing_project_id": fakeFlagProjectId,
			},
			wantResult: fakeFlagProjectId,
		},
	}

	for _, tc := range testData {

		runTest(t, tc.runServer, func() {

			for fk, fv := range tc.flags {
				flag.Set(fk, fv)
			}

			got, err := getTracingProjectId()

			if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Errorf("Test (%s): failed, got err: %v, want err: %v", tc.desc, err, tc.wantError)
			}

			if tc.wantError == "" && err != nil {
				t.Errorf("Test (%s): failed, got err: %v, want no err", tc.desc, err)
			}

			if !reflect.DeepEqual(got, tc.wantResult) {
				t.Errorf("Test (%s): failed, got: %v, want: %v", tc.desc, got, tc.wantResult)
			}

		})

	}
}

func runTest(t *testing.T, shouldRunServer bool, f func()) {

	if shouldRunServer {
		// Run a mock server and point injected client to mock server
		mockMetadataServer := util.InitMockServerFromPathResp(map[string]string{
			util.ProjectIDSuffix: fakeMetadataProjectId,
		})
		defer mockMetadataServer.Close()
		metadata.SetMockMetadataFetcher(mockMetadataServer.URL, time.Now())
	} else {
		// Point injected client to non-existent url
		metadata.SetMockMetadataFetcher("non-existent-url-39874983", time.Now())
	}

	f()
}
