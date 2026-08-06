package test

import (
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"
)

// The request line is a fleet-wide invariant: all three services emit the
// same slog JSON shape, key for key, duration in integer nanoseconds.
// remote_addr is the first X-Forwarded-For entry, trimmed.
func TestRequestLogShape(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/health", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Forwarded-For", "203.0.113.77 , 198.51.100.1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	var line map[string]any
	for deadline := time.Now().Add(2 * time.Second); line == nil; {
		for raw := range strings.SplitSeq(logs.String(), "\n") {
			var m map[string]any
			if json.Unmarshal([]byte(raw), &m) == nil && m["remote_addr"] == "203.0.113.77" {
				line = m
				break
			}
		}
		if line == nil {
			if time.Now().After(deadline) {
				t.Fatal(
					"no request line with remote_addr 203.0.113.77 (first XFF entry, trimmed) was logged",
				)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	keys := []string{"time", "level", "msg", "method", "path", "remote_addr", "status", "duration"}
	for _, k := range keys {
		if _, ok := line[k]; !ok {
			t.Errorf("missing key %q in %v", k, line)
		}
	}
	if len(line) != len(keys) {
		t.Errorf("line has %d keys, want exactly %d: %v", len(line), len(keys), line)
	}
	if line["msg"] != "http request completed" {
		t.Errorf("msg = %q, want %q", line["msg"], "http request completed")
	}
	if line["level"] != "INFO" {
		t.Errorf("level = %q, want %q", line["level"], "INFO")
	}
	if line["method"] != http.MethodGet {
		t.Errorf("method = %q, want %q", line["method"], http.MethodGet)
	}
	if line["path"] != "/health" {
		t.Errorf("path = %q, want %q", line["path"], "/health")
	}
	if line["status"] != float64(http.StatusOK) {
		t.Errorf("status = %v, want %d", line["status"], http.StatusOK)
	}
	if ts, ok := line["time"].(string); !ok {
		t.Errorf("time = %v, want a string", line["time"])
	} else if _, err := time.Parse(time.RFC3339Nano, ts); err != nil {
		t.Errorf("time = %q, want RFC3339: %v", ts, err)
	}
	dur, ok := line["duration"].(float64)
	if !ok || dur <= 0 || dur != math.Trunc(dur) {
		t.Errorf("duration = %v, want positive integer nanoseconds", line["duration"])
	}
}

// The logger wraps the mux itself, so requests that never match a route are
// still logged.
func TestUnmatchedPathIsLogged(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/definitely-not-routed", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	for deadline := time.Now().Add(2 * time.Second); ; {
		for raw := range strings.SplitSeq(logs.String(), "\n") {
			var m map[string]any
			if json.Unmarshal([]byte(raw), &m) == nil &&
				m["path"] == "/definitely-not-routed" &&
				m["status"] == float64(http.StatusNotFound) {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("no request line for the unmatched path was logged")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
