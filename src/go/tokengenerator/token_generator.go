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

package tokengenerator

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	_GOOGLE_API_SCOPE = []string{
		// Call servicemanagement to fetch service config.
		"https://www.googleapis.com/auth/service.management.readonly",
		// Call servicecontrol to get latest rollout id.
		"https://www.googleapis.com/auth/servicecontrol",
	}
	tokenCache = &oauth2.Token{}
	tokenMux   = sync.Mutex{}
)

var GenerateAccessTokenFromFile = func(saFilePath string) (string, time.Duration, error) {
	if token, duration := activeAccessToken(); token != "" {
		return token, duration, nil
	}

	data, err := ioutil.ReadFile(saFilePath)
	if err != nil {
		return "", 0, err
	}

	return generateAccessToken(data)
}

// A test-friendly version of `GenerateAccessTokenFromFile`
func generateAccessTokenFromData(saData []byte) (string, time.Duration, error) {
	if token, duration := activeAccessToken(); token != "" {
		return token, duration, nil
	}

	return generateAccessToken(saData)
}

func activeAccessToken() (string, time.Duration) {
	now := time.Now()
	tokenMux.Lock()
	defer tokenMux.Unlock()

	// Follow the similar logic as GCE metadata server, where returned token will be valid for at
	// least 60s.
	if tokenCache.AccessToken == "" || now.After(tokenCache.Expiry.Add(-time.Second*60)) {
		return "", 0

	}

	return tokenCache.AccessToken, tokenCache.Expiry.Sub(now)
}

func generateAccessToken(keyData []byte) (string, time.Duration, error) {
	creds, err := google.CredentialsFromJSON(oauth2.NoContext, keyData, _GOOGLE_API_SCOPE...)
	if err != nil {
		return "", 0, err
	}

	token, err := creds.TokenSource.Token()
	if err != nil {
		return "", 0, err
	}

	tokenMux.Lock()
	defer tokenMux.Unlock()

	tokenCache = token
	return token.AccessToken, token.Expiry.Sub(time.Now()), nil
}

func MakeLatsTokenHandler(serviceAccountKey string) http.Handler {
	r := mux.NewRouter()

	r.PathPrefix(util.AccessTokenSuffix).Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, expire, err := GenerateAccessTokenFromFile(serviceAccountKey)

		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// Access token response is a JSON payload in the format:
		// {
		//   "access_token": "string",
		//   "expires_in": uint
		// }
		_, _ = w.Write([]byte(fmt.Sprintf(`{"access_token": "%s", "expires_in": %v}`, token, int(expire.Seconds()))))
	})

	return r
}
