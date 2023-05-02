// Copyright 2023 Google LLC
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

package helpers

import (
	"fmt"
	"net/url"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// ParseServiceControlURLFromOPConfig parses the service control URL from
// OP service config + descriptor + ESPv2 options.
func ParseServiceControlURLFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) (url.URL, error) {
	uri := getServiceControlURI(serviceConfig, opts)
	if uri == "" {
		return url.URL{}, nil
	}

	// The assumption about control.environment field. Its format:
	//   [scheme://] +  host + [:port]
	// * It should not have any path part
	// * If scheme is missed, https is the default
	scURL, err := util.ParseURIIntoURL(uri)
	if err != nil {
		return url.URL{}, fmt.Errorf("failed to parse uri %q into scURL: %v", uri, err)
	}
	if scURL.Path != "" {
		return url.URL{}, fmt.Errorf("error parsing service control scURL %+v: should not have path part: %s", scURL, scURL.Path)
	}

	return scURL, nil
}

func getServiceControlURI(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) string {
	// Ignore value from ServiceConfig if flag is set
	if uri := opts.ServiceControlURL; uri != "" {
		return uri
	}

	return serviceConfig.GetControl().GetEnvironment()
}
