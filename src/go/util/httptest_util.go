package util

import (
	"net/http"
	"net/http/httptest"
	"time"
)

type MockServer struct {
	s             *httptest.Server
	sleepDuration time.Duration
	resp          string
}

func InitMockServer(response string) *MockServer {
	m := &MockServer{
		resp: response,
	}
	m.s = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(m.sleepDuration)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(m.resp))
	}))
	return m
}

func (m *MockServer) SetResp(response string) {
	m.resp = response
}

func (m *MockServer) GetURL() string {
	return m.s.URL
}

func (m *MockServer) Close() {
	m.s.Close()
}

func (m *MockServer) SetSleepTime(sleepDuration time.Duration) {
	m.sleepDuration = sleepDuration
}
func InitMockServerFromPathResp(pathResp map[string]string) *httptest.Server {
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
