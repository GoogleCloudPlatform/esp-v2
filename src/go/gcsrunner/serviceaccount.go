package gcsrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// staticTokenSource implements oath2.TokenSource
type staticTokenSource struct {
	accessToken string    `json:"access_token"`
	expiry      time.Time `json:"expiry,omitempty"`
}

func (s staticTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken: s.accessToken,
		Expiry:      s.expiry,
	}, nil
}

type iamcredentialsResponse struct {
	AccessToken string `json:accessToken`
	ExpireTime  string `json:expireTime`
}

// TokenSource returns an oauth2.TokenSource which provides a short-lived token
// for impersonating the provided service account.
//
// Because this token should only be used in the start-up script, this token cannot be refreshed.
func TokenSource(ctx context.Context, sa string) (oauth2.TokenSource, error) {
	url := fmt.Sprintf("https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/%s:generateAccessToken", sa)
	bodyBytes, err := json.Marshal(map[string][]string{
		"scope": []string{storage.ScopeReadOnly},
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	c, err := google.FindDefaultCredentials(ctx)
	if err != nil {
		return nil, err
	}
	token, err := c.TokenSource.Token()
	if err != nil {
		return nil, err
	}
	token.SetAuthHeader(req)

	client := http.Client{
		Timeout: time.Second * 5,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed iamcredentials request (could not read body): %v", resp.Status)
		}
		return nil, fmt.Errorf("failed iamcredentials request: %v\n%v", resp.Status, string(respBody))
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var iamCredsResp iamcredentialsResponse
	if err := json.Unmarshal(respBody, &iamCredsResp); err != nil {
		return nil, err
	}
	expiry, err := time.Parse(time.RFC3339, iamCredsResp.ExpireTime)
	if err != nil {
		return nil, fmt.Errorf("could not parse expiry: %v", err)
	}
	return &staticTokenSource{
		accessToken: iamCredsResp.AccessToken,
		expiry:      expiry,
	}, nil
}
