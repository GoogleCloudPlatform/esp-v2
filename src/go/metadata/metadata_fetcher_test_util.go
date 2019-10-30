package metadata

import (
	"time"

	"github.com/GoogleCloudPlatform/api-proxy/src/go/options"
)

// Creates a mock metadata fetcher and returns the mock instance
func NewMockMetadataFetcher(baseUrl string, now time.Time) *MetadataFetcher {
	return &MetadataFetcher{
		baseUrl: baseUrl,
		timeNow: func() time.Time {
			return now
		},
	}
}

// Injects the mock constructor into source code. Mock metadata fetcher only created
// when source code calls constructor.
func SetMockMetadataFetcher(baseUrl string, now time.Time) {
	NewMetadataFetcher = func(opts options.CommonOptions) *MetadataFetcher {
		return &MetadataFetcher{
			baseUrl: baseUrl,
			timeNow: func() time.Time {
				return now
			},
		}
	}
}
