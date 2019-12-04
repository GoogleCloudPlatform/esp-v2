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

import (
	"io/ioutil"
	"strings"

	"github.com/golang/glog"
		"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"

)

var versionVal = ""

func ESPv2Version() string {
	if versionVal == "" {
		file, err := ioutil.ReadFile(platform.GetFilePath(platform.Version))
		if err != nil {
			glog.Errorf("Failed to generate version by VERSION under the root path: %v", err)
		}
		versionVal = strings.TrimSpace(string(file))
	}
	return versionVal
}
