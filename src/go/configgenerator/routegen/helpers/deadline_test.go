package helpers

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
)

func TestNewRouteDeadlineConfigerFromOPConfig(t *testing.T) {
	testdata := []struct {
		desc            string
		opts            options.ConfigGeneratorOptions
		deadline        time.Duration
		isStreaming     bool
		wantDeadline    time.Duration
		wantIdleTimeout time.Duration
	}{
		{
			desc: "Global idle timeout takes priority over small deadline",
			opts: options.ConfigGeneratorOptions{
				StreamIdleTimeout: util.DefaultIdleTimeout,
			},
			deadline:        10*time.Second + 500*time.Millisecond,
			wantDeadline:    10*time.Second + 500*time.Millisecond,
			wantIdleTimeout: util.DefaultIdleTimeout,
		},
		{
			desc: "Deadline takes priority over small global idle timeout",
			opts: options.ConfigGeneratorOptions{
				StreamIdleTimeout: 7 * time.Second,
			},
			deadline:        10*time.Second + 500*time.Millisecond,
			wantDeadline:    10*time.Second + 500*time.Millisecond,
			wantIdleTimeout: 10*time.Second + 500*time.Millisecond + 1*time.Second,
		},
		{
			desc: "Global idle timeout takes priority over missing deadline",
			opts: options.ConfigGeneratorOptions{
				StreamIdleTimeout: 30 * time.Second,
			},
			wantDeadline:    util.DefaultResponseDeadline,
			wantIdleTimeout: 30 * time.Second,
		},
		{
			desc: "Global idle timeout takes priority over negative deadline, deadline modified",
			opts: options.ConfigGeneratorOptions{
				StreamIdleTimeout: util.DefaultIdleTimeout,
			},
			deadline:        -10*time.Second + 500*time.Millisecond,
			wantDeadline:    util.DefaultResponseDeadline,
			wantIdleTimeout: util.DefaultIdleTimeout,
		},
		{
			desc: "Default deadline takes priority over small global idle timeout with missing deadline",
			opts: options.ConfigGeneratorOptions{
				StreamIdleTimeout: 7 * time.Second,
			},
			deadline:        util.DefaultResponseDeadline,
			wantDeadline:    util.DefaultResponseDeadline,
			wantIdleTimeout: util.DefaultResponseDeadline + 1*time.Second,
		},
		{
			desc: "Default deadline takes priority over small global idle timeout and negative deadline",
			opts: options.ConfigGeneratorOptions{
				StreamIdleTimeout: 7 * time.Second,
			},
			deadline:        -10*time.Second + 500*time.Millisecond,
			wantDeadline:    util.DefaultResponseDeadline,
			wantIdleTimeout: util.DefaultResponseDeadline + time.Second,
		},
		{
			desc: "Streaming methods set the idle timeout directly from the deadline, even if the global stream idle timeout is larger.",
			opts: options.ConfigGeneratorOptions{
				StreamIdleTimeout: util.DefaultIdleTimeout,
			},
			deadline:        10*time.Second + 500*time.Millisecond,
			isStreaming:     true,
			wantDeadline:    0,
			wantIdleTimeout: 10*time.Second + 500*time.Millisecond,
		},
		{
			desc: "Streaming methods with NO deadline specified and the global timeout larger than the default deadline, use the global timeout.",
			opts: options.ConfigGeneratorOptions{
				StreamIdleTimeout: 25 * time.Second,
			},
			isStreaming:     true,
			wantDeadline:    0,
			wantIdleTimeout: 25 * time.Second,
		},
		{
			desc: "Streaming methods with NO deadline specified and the global timeout smaller than the default deadline, use the default deadline.",
			opts: options.ConfigGeneratorOptions{
				StreamIdleTimeout: 7 * time.Second,
			},
			isStreaming:     true,
			wantDeadline:    0,
			wantIdleTimeout: util.DefaultResponseDeadline + time.Second,
		},
	}

	for _, tc := range testdata {
		t.Run(tc.desc, func(t *testing.T) {
			c := NewRouteDeadlineConfigerFromOPConfig(tc.opts)
			gotDeadline, gotIdleTimeout := c.CalcIdleTimeout(tc.deadline, tc.isStreaming)

			if gotDeadline != tc.wantDeadline {
				t.Errorf("CalcIdleTimeout(...) returns deadline %q, want deadline %q", gotDeadline.String(), tc.wantDeadline.String())
			}
			if gotIdleTimeout != tc.wantIdleTimeout {
				t.Errorf("CalcIdleTimeout(...) returns idle timeout %q, want idle timeout %q", gotIdleTimeout.String(), tc.wantIdleTimeout.String())
			}
		})
	}
}
