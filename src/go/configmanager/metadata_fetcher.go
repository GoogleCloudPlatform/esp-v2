// Copyright 2018 Google Cloud Platform Proxy Authors
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

package configmanager

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"cloudesf.googlesource.com/gcpproxy/src/go/flags"
	"cloudesf.googlesource.com/gcpproxy/src/go/util"
	"github.com/golang/glog"

	scpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/service_control"
)

const (
	tokenExpiry = 3599
)

var (
	// metadata updates and stores Metadata from GCE.
	metadata   tokenInfo
	metdataMux sync.Mutex
	// audience -> tokenInfo.
	audToToken       sync.Map
	timeNow          = time.Now
	fetchMetadataURL = func(suffix string) string {
		return *flags.MetadataURL + suffix
	}
)

type tokenInfo struct {
	accessToken  string
	tokenTimeout time.Time
}

type metadataTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

var metadataClient http.Client

var getMetadata = func(path string) ([]byte, error) {
	req, _ := http.NewRequest("GET", path, nil)
	req.Header.Add("Metadata-Flavor", "Google")
	resp, err := metadataClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(`failed fetching metadata: %v, status code %v"`, path, resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}

func fetchAccessToken() (string, time.Duration, error) {
	now := timeNow()
	// Follow the similar logic as GCE metadata server, where returned token will be valid for at
	// least 60s.
	metdataMux.Lock()
	defer metdataMux.Unlock()
	if metadata.accessToken != "" && !now.After(metadata.tokenTimeout.Add(-time.Second*60)) {
		return metadata.accessToken, metadata.tokenTimeout.Sub(now), nil
	}

	tokenBody, err := getMetadata(fetchMetadataURL(util.ServiceAccountTokenSuffix))
	if err != nil {
		return "", 0, err
	}

	var resp metadataTokenResponse
	if err = json.Unmarshal(tokenBody, &resp); err != nil {
		return "", 0, err
	}

	expires := time.Duration(resp.ExpiresIn) * time.Second
	metadata.accessToken = resp.AccessToken
	metadata.tokenTimeout = now.Add(expires)
	return metadata.accessToken, expires, nil
}

// TODO(kyuc): perhaps we need some retry logic and timeout?
func fetchMetadata(key string) (string, error) {
	body, err := getMetadata(fetchMetadataURL(key))
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func fetchServiceName() (string, error) {
	return fetchMetadata(util.ServiceNameSuffix)
}

func fetchConfigId() (string, error) {
	return fetchMetadata(util.ConfigIDSuffix)
}

func fetchRolloutStrategy() (string, error) {
	return fetchMetadata(util.RolloutStrategySuffix)
}

func fetchIdentityJWTToken(audience string) (string, time.Duration, error) {
	now := timeNow()
	// Follow the similar logic as GCE metadata server, where returned token will be valid for at
	// least 60s.
	if ti, ok := audToToken.Load(audience); ok {
		info := ti.(tokenInfo)
		if !now.After(info.tokenTimeout.Add(-time.Second * 60)) {
			return info.accessToken, info.tokenTimeout.Sub(now), nil
		}
	}

	identityTokenURI := util.IdentityTokenSuffix + "?audience=" + audience + "&format=standard"
	token, err := fetchMetadata(identityTokenURI)
	if err != nil {
		return "", 0, err
	}

	expires := time.Duration(tokenExpiry) * time.Second
	audToToken.Store(audience, tokenInfo{
		accessToken:  token,
		tokenTimeout: now.Add(expires),
	},
	)
	return token, expires, nil
}

func fetchGCPAttributes() *scpb.GcpAttributes {
	// Checking if metadata server is reachable.
	if _, err := fetchMetadata(""); err != nil {
		return nil
	}

	attrs := &scpb.GcpAttributes{}
	if projectID, err := fetchMetadata(util.ProjectIDSuffix); err == nil {
		attrs.ProjectId = projectID
	}

	if zone, err := fetchZone(); err == nil {
		attrs.Zone = zone
	}

	attrs.Platform = fetchPlatform()
	return attrs
}

// Do not directly use this function. Use fetchGCPAttributes instead.
func fetchZone() (string, error) {
	zonePath, err := fetchMetadata(util.ZoneSuffix)
	if err != nil {
		return "", err
	}

	// Zone format: projects/PROJECT_ID/ZONE
	// Get the substring after the last '/'.
	index := strings.LastIndex(zonePath, "/")
	if index == -1 || index+1 >= len(zonePath) {
		glog.Warningf("Invalid zone format is fetched: %s", zonePath)
		return "", fmt.Errorf("Invalid zone format: %s", zonePath)
	}
	return zonePath[index+1:], nil
}

// Do not directly use this function. Use fetchGCPAttributes instead.
func fetchPlatform() string {
	if _, err := fetchMetadata(util.GAEServerSoftwareSuffix); err == nil {
		return util.GAEFlex
	}

	if _, err := fetchMetadata(util.KubeEnvSuffix); err == nil {
		return util.GKE
	}

	// TODO(kyuc): what about Cloud Run...?

	return util.GCE
}
