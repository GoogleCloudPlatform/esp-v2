package util

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func InitMockServer(_ *testing.T, resp string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
	}))
}

func InitMockServerFromPathResp(_ *testing.T, pathResp map[string]string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Root is used to tell if the sever is healthy or not.
		if r.URL.Path == "" || r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if resp, ok := pathResp[r.URL.Path]; ok {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(resp))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}
