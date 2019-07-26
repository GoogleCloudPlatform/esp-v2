package configmanager

import (
	"testing"
	"time"
)

func NewMockMetadataFetcher(_ *testing.T, baseUrl string, now time.Time) *MetadataFetcher {
	return &MetadataFetcher{
		baseUrl: baseUrl,
		timeNow: func() time.Time {
			return now
		},
	}
}
