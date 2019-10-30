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

package utils

const (
	OK                  = 0
	CANCELLED           = 1
	UNKNOWN             = 2
	INVALID_ARGUMENT    = 3
	DEADLINE_EXCEEDED   = 4
	NOT_FOUND           = 5
	ALREADY_EXISTS      = 6
	PERMISSION_DENIED   = 7
	UNAUTHENTICATED     = 16
	RESOURCE_EXHAUSTED  = 8
	FAILED_PRECONDITION = 9
	ABORTED             = 10
	OUT_OF_RANGE        = 11
	UNIMPLEMENTED       = 12
	INTERNAL            = 13
	UNAVAILABLE         = 14
	DATA_LOSS           = 15
)

func HttpResponseCodeToStatusCode(code int) int {
	switch {
	case code == 400:
		return INVALID_ARGUMENT
	case code == 401:
		return UNAUTHENTICATED
	case code == 403:
		return PERMISSION_DENIED
	case code == 404:
		return NOT_FOUND
	case code == 409:
		return ABORTED
	case code == 416:
		return OUT_OF_RANGE
	case code == 429:
		return RESOURCE_EXHAUSTED
	case code == 499:
		return CANCELLED
	case code == 501:
		return UNIMPLEMENTED
	case code == 503:
		return UNAVAILABLE
	case code == 504:
		return DEADLINE_EXCEEDED
	case code >= 200 && code < 300:
		return OK
	case code >= 400 && code < 500:
		return FAILED_PRECONDITION
	case code >= 500 && code < 600:
		return INTERNAL
	}
	return UNKNOWN
}
