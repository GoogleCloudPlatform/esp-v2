package util

import (
	"fmt"
	"strings"
	"testing"
)

// TODO(nareddyt): Consider moving this to a common file and have all tests reuse it.
func errorDiff(t *testing.T, gotErr error, wantErr error) {
	if gotErr == nil {
		if wantErr != nil {
			t.Fatalf("\ngot no err \nwant err: %v", wantErr)
		}
		return
	}

	if wantErr == nil {
		t.Fatalf("\ngot err : %v \nwant no err", gotErr)
	}

	if !strings.Contains(gotErr.Error(), wantErr.Error()) {
		t.Fatalf("\ngot err : %v \nwant err: %v", gotErr, wantErr)
	}
}

func TestHasOnlyUrlChars(t *testing.T) {
	testData := []struct {
		desc string
		url  string
		// Not full error, tests for substring.
		wantError error
	}{
		{
			desc:      "Valid url passes",
			url:       "https://www.google.com/search",
			wantError: nil,
		},
		{
			desc:      "Non-urls pass",
			url:       "foo-bar",
			wantError: nil,
		},
		{
			desc:      "Valid url with query params and fragment passes",
			url:       "https://www.google.com/search?key=param&key2=param2#id",
			wantError: nil,
		},
		{
			desc:      "Scheme not required",
			url:       "www.google.com/search?key=param#id",
			wantError: nil,
		},
		{
			desc:      "Invalid scheme fails",
			url:       "://www.google.com/search?key=param#id",
			wantError: fmt.Errorf("missing protocol scheme"),
		},
		{
			desc:      "Does not accept \\n char",
			url:       "www.google.com/search\ntest?key=param#id",
			wantError: fmt.Errorf("invalid control character in URL"),
		},
		{
			desc:      "Does not accept \\r char",
			url:       "www.google.com/search\rtest?key=param#id",
			wantError: fmt.Errorf("invalid control character in URL"),
		},
		{
			desc:      "Does not accept \\0 char",
			url:       "www.google.com/search\0033test?key=param#id",
			wantError: fmt.Errorf("invalid control character in URL"),
		},
		{
			desc:      "Empty works",
			url:       "",
			wantError: nil,
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			gotError := HasOnlyUrlChars(tc.url)
			errorDiff(t, gotError, tc.wantError)
		})
	}
}

func TestHasOnlyUrlCharsWithNoQuery(t *testing.T) {
	testData := []struct {
		desc string
		url  string
		// Not full error, tests for substring.
		wantError error
	}{
		{
			desc:      "Valid url passes",
			url:       "https://www.google.com/search",
			wantError: nil,
		},
		{
			desc:      "Non-urls pass",
			url:       "foo-bar",
			wantError: nil,
		},
		{
			desc:      "Does not accept ? char",
			url:       "https://www.google.com/search?key=param",
			wantError: fmt.Errorf("contains query parameters or fragements"),
		},
		{
			desc:      "Does not accept & char",
			url:       "https://www.google.com/search&test",
			wantError: fmt.Errorf("contains query parameters or fragements"),
		},
		{
			desc:      "Does not accept # char",
			url:       "https://www.google.com/search#test",
			wantError: fmt.Errorf("contains query parameters or fragements"),
		},
		{
			desc:      "Does not accept \\n char",
			url:       "www.google.com/search\ntest?key=param#id",
			wantError: fmt.Errorf("invalid control character in URL"),
		},
		{
			desc:      "Does not accept \\r char",
			url:       "www.google.com/search\rtest?key=param#id",
			wantError: fmt.Errorf("invalid control character in URL"),
		},
		{
			desc:      "Does not accept \\0 char",
			url:       "www.google.com/search\0033test?key=param#id",
			wantError: fmt.Errorf("invalid control character in URL"),
		},
		{
			desc:      "Empty works",
			url:       "",
			wantError: nil,
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			gotError := HasOnlyUrlCharsWithNoQuery(tc.url)
			errorDiff(t, gotError, tc.wantError)
		})
	}
}
