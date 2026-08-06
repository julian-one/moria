package test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestHealth(t *testing.T) {
	for _, path := range []string{"/health"} {
		req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		var body struct {
			Status string `json:"status"`
			Time   string `json:"time"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: status = %d, want %d", path, resp.StatusCode, http.StatusOK)
		}
		if body.Status != "ok" {
			t.Errorf("%s: status = %q, want %q", path, body.Status, "ok")
		}
		if _, err := time.Parse(time.RFC3339, body.Time); err != nil {
			t.Errorf("%s: time = %q, want RFC3339: %v", path, body.Time, err)
		}
	}
}
