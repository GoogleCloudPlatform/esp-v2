package components

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	sc "github.com/google/go-genproto/googleapis/api/servicecontrol/v1"
)

func TestMockServiceControl(t *testing.T) {
	s := NewMockServiceCtrl("mmm")

	url := s.GetURL() + "/v1/services/mmm:check"

	req := &sc.CheckRequest{
		ServiceName: "mmm",
	}
	req_body, _ := proto.Marshal(req)

	reqq, _ := http.NewRequest("POST", url, bytes.NewReader(req_body))
	resp, err := http.DefaultClient.Do(reqq)
	if err != nil {
		t.Errorf("Failed in request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("Wrong response status: %v", resp.StatusCode)
	}

	rr, err := s.GetRequests(1, 3*time.Second)
	if err != nil {
		t.Errorf("GetRequests failed with: %v", err)
	}
	if len(rr) != 1 {
		t.Errorf("Wrong number: %d", len(rr))
	}
	if rr[0].ReqType != CHECK_REQUEST {
		t.Errorf("Wrong type: %v", rr[0].ReqType)
	}
	req1 := &sc.CheckRequest{}
	err = proto.Unmarshal(rr[0].ReqBody, req1)
	if err != nil {
		t.Errorf("failed to parse body into CheckRequest.")
	}
	if !proto.Equal(req1, req) {
		t.Errorf("Wrong request data")
	}

	// try to read it again
	rr, err = s.GetRequests(1, 1*time.Second)
	if err == nil {
		t.Errorf("Expected timeout error")
	}
}
