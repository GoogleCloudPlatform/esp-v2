package util

import "time"

func MaxDuration(a, b time.Duration) time.Duration {
	if a >= b {
		return a
	}
	return b
}
