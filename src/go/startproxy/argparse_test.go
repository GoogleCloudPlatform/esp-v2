// Copyright 2026 Google LLC
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

package startproxy

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

// testdata/help.txt is start_proxy.py's --help output on Python 3.11 (the
// version in the last Python-based image) with COLUMNS=80 and the program
// named "startproxy".
func TestHelpMatchesStartProxyPy(t *testing.T) {
	want, err := os.ReadFile("testdata/help.txt")
	if err != nil {
		t.Fatal(err)
	}
	p := newArgParser(flagDefs(defaultArgs()))
	if got := p.formatHelp(78); got != string(want) {
		gotLines, wantLines := strings.Split(got, "\n"), strings.Split(string(want), "\n")
		for i := 0; i < len(gotLines) && i < len(wantLines); i++ {
			if gotLines[i] != wantLines[i] {
				t.Fatalf("help differs at line %d:\ngot  %q\nwant %q", i+1, gotLines[i], wantLines[i])
			}
		}
		t.Fatalf("help has %d lines, want %d", len(gotLines), len(wantLines))
	}
}

func TestEveryVisibleFlagHasHelp(t *testing.T) {
	for _, d := range flagDefs(defaultArgs()) {
		if !d.hidden && strings.TrimSpace(d.help) == "" {
			t.Errorf("%s has no help text", strings.Join(d.names, "/"))
		}
	}
}

func TestRunHelp(t *testing.T) {
	for _, argv := range [][]string{{"-h"}, {"--help"}, {"--hel"}, {"--bogus", "--help"}} {
		if code := Run(argv); code != 0 {
			t.Errorf("Run(%q) = %d, want 0", argv, code)
		}
	}
}

func TestRunParseError(t *testing.T) {
	// start_proxy.py's ArgumentParser.error() exits with status 1, not
	// argparse's usual 2.
	for _, argv := range [][]string{{"--bogus"}, {"--version"}, {"--he"}, {"--listener_port=x"}} {
		if code := Run(argv); code != 1 {
			t.Errorf("Run(%q) = %d, want 1", argv, code)
		}
	}
}

// The expectations below were produced by start_proxy.py on Python 3.11.
func TestParseArgsMatchesArgparse(t *testing.T) {
	testcases := []struct {
		argv    []string
		wantErr string
		// check inspects the parsed arguments when wantErr is empty.
		check func(a *proxyArgs) bool
	}{
		// A value option followed by an option-like token has no value, even if
		// the token is not a known option.
		{argv: []string{"--service_control_url", "--typo_flag=1"}, wantErr: "argument --service_control_url: expected one argument"},
		// Values are kept verbatim; tokens are not trimmed.
		{argv: []string{"--version=v1", "--backend_retry_ons=trail "}, check: func(a *proxyArgs) bool { return *a.BackendRetryOns == "trail " }},
		{argv: []string{"--version=v1 "}, check: func(a *proxyArgs) bool { return a.Version == "v1 " }},
		{argv: []string{" --version=v1"}, wantErr: "unrecognized arguments:  --version=v1"},
		// Integers are parsed like Python's int().
		{argv: []string{"--listener_port= 8080 "}, check: func(a *proxyArgs) bool { return *a.ListenerPort == 8080 }},
		{argv: []string{"--listener_port=8_080"}, check: func(a *proxyArgs) bool { return *a.ListenerPort == 8080 }},
		{argv: []string{"--listener_port=+8080"}, check: func(a *proxyArgs) bool { return *a.ListenerPort == 8080 }},
		{argv: []string{"--listener_port=8__080"}, wantErr: "argument --listener_port: invalid int value: '8__080'"},
		{argv: []string{"--listener_port=0x10"}, wantErr: "argument --listener_port: invalid int value: '0x10'"},
		// Abbreviations, including of --help.
		{argv: []string{"--hel"}, wantErr: errHelpRequested.Error()},
		{argv: []string{"--bogus", "--help"}, wantErr: errHelpRequested.Error()},
		{argv: []string{"--he"}, wantErr: "ambiguous option: --he could match --help, --healthz, --health_check_grpc_backend, " +
			"--health_check_grpc_backend_service, --health_check_grpc_backend_interval, --health_check_grpc_backend_no_traffic_interval"},
		{argv: []string{"--service_j=/tmp/x"}, check: func(a *proxyArgs) bool { return *a.ServiceJSONPath == "/tmp/x" }},
		{argv: []string{"--serv=x"}, wantErr: "ambiguous option: --serv=x could match --service, --service_json_path, " +
			"--service_control_network_fail_policy, --service_control_enable_api_key_uid_reporting, --service_control_url, " +
			"--service_control_check_timeout_ms, --service_control_quota_timeout_ms, --service_control_report_timeout_ms, " +
			"--service_control_check_retries, --service_control_quota_retries, --service_control_report_retries, --service_account_key"},
		// Short options: attached values and combined flags.
		{argv: []string{"-zx=y"}, check: func(a *proxyArgs) bool { return *a.Healthz == "x=y" }},
		{argv: []string{"-sfoo=bar"}, check: func(a *proxyArgs) bool { return a.Service == "foo=bar" }},
		{argv: []string{"-tN", "8001"}, check: func(a *proxyArgs) bool { return a.TLSMutualAuth && a.StatusPort == 8001 }},
		{argv: []string{"-tx"}, wantErr: "argument -t/--tls_mutual_auth: ignored explicit argument 'x'"},
		{argv: []string{"--non_gcp=1"}, wantErr: "argument --non_gcp: ignored explicit argument '1'"},
		// Negative numbers are values, not options.
		{argv: []string{"--version", "-1"}, check: func(a *proxyArgs) bool { return a.Version == "-1" }},
		{argv: []string{"--version=v1", "--"}, wantErr: "unrecognized arguments: --"},
		{argv: []string{"--version=v1", "extra", "--non_gcp", "x"}, wantErr: "unrecognized arguments: extra x"},
		{argv: []string{"--no-service_control_enable_api_key_uid_reporting"}, check: func(a *proxyArgs) bool { return !a.ServiceControlEnableAPIKeyUIDReporting }},
		{argv: []string{"--add_request_header=a=1", "--add_request_header", "b=2"}, check: func(a *proxyArgs) bool {
			return reflect.DeepEqual(a.AddRequestHeader, []string{"a=1", "b=2"})
		}},
		{argv: []string{"--rollout_strategy=bogus"}, wantErr: "argument -R/--rollout_strategy: invalid choice: 'bogus' (choose from 'fixed', 'managed')"},
		// Hidden options are accepted.
		{argv: []string{"-g", "x"}, check: func(a *proxyArgs) bool { return *a.Management == "x" }},
		// The serverless mode switch is not an option.
		{argv: []string{"--serverless"}, wantErr: "unrecognized arguments: --serverless"},
	}
	for _, tc := range testcases {
		t.Run(strings.Join(tc.argv, " "), func(t *testing.T) {
			args, err := parseArgs(tc.argv)
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Errorf("got error %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.check(args) {
				t.Errorf("unexpected parse result: %+v", *args)
			}
		})
	}
}

func TestHelpRequestedIsDistinguishable(t *testing.T) {
	if _, err := parseArgs([]string{"--help"}); !errors.Is(err, errHelpRequested) {
		t.Errorf("got %v, want errHelpRequested", err)
	}
}

func TestPyAtoi(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want int
		ok   bool
	}{
		{"8080", 8080, true}, {" 8080\n", 8080, true}, {"8_080", 8080, true}, {"-1", -1, true}, {"+1", 1, true},
		{"010", 10, true}, {"", 0, false}, {"_1", 0, false}, {"1_", 0, false}, {"1__0", 0, false},
		{"+-1", 0, false}, {"- 1", 0, false}, {"0x10", 0, false}, {"1.0", 0, false},
		// int() strips Unicode whitespace, but not U+001C..U+001F, which
		// str.isspace() does count as whitespace.
		{"\u30008080\u00a0", 8080, true}, {"\x1c8080", 0, false}, {"8080\x1f", 0, false},
	} {
		got, err := pyAtoi(tc.in)
		if (err == nil) != tc.ok || got != tc.want {
			t.Errorf("pyAtoi(%q) = %d, %v; want %d, ok=%v", tc.in, got, err, tc.want, tc.ok)
		}
	}
}

func TestPyRepr(t *testing.T) {
	for in, want := range map[string]string{
		"abc":    `'abc'`,
		"it's":   `"it's"`,
		`'"`:     `'\'"'`,
		"a\tb\n": `'a\tb\n'`,
		"\x01":   `'\x01'`,
		"é":      `'é'`,
	} {
		if got := pyRepr(in); got != want {
			t.Errorf("pyRepr(%q) = %s, want %s", in, got, want)
		}
	}
}

// The expectations below were produced by Python 3.11's textwrap.wrap.
func TestWrapText(t *testing.T) {
	for _, tc := range []struct {
		text  string
		width int
		want  []string
	}{
		{"one two three four", 9, []string{"one two", "three", "four"}},
		{"the X-Forwarded-For header", 15, []string{"the", "X-Forwarded-For", "header"}},
		{"well-known non-blocking re-try", 12, []string{"well-known", "non-blocking", "re-try"}},
		// Words longer than the width are cut, after a hyphen if possible.
		{"see https://example.com/a-long/path-name now", 20, []string{"see https://example.", "com/a-long/path-name", "now"}},
		{"--service_control_url is a flag", 10, []string{"--service_", "control_ur", "l is a", "flag"}},
		// An em-dash between words is a break opportunity.
		{"alpha--beta gamma", 8, []string{"alpha--", "beta", "gamma"}},
	} {
		if got := wrapText(tc.text, tc.width); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("wrapText(%q, %d) = %q, want %q", tc.text, tc.width, got, tc.want)
		}
	}
}
