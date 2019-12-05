// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metadata

import (
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
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
