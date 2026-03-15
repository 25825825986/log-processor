package main

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

const baseURL = "http://localhost:8080"

func getJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(baseURL + path)
	if err != nil {
		t.Skipf("server not reachable (%v), skip smoke test", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status %d for %s: %s", resp.StatusCode, path, string(body))
	}

	var v map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("invalid json for %s: %v", path, err)
	}
	return v
}

func TestStatusEndpoint(t *testing.T) {
	data := getJSON(t, "/api/status")
	if _, ok := data["processor"]; !ok {
		t.Fatalf("missing processor field in /api/status")
	}
	if _, ok := data["config"]; !ok {
		t.Fatalf("missing config field in /api/status")
	}
}

func TestConfigEndpoint(t *testing.T) {
	data := getJSON(t, "/api/config")
	for _, field := range []string{"server", "parser", "processor", "receiver", "storage"} {
		if _, ok := data[field]; !ok {
			t.Fatalf("missing %s field in /api/config", field)
		}
	}
}

func TestOtherReadonlyEndpoints(t *testing.T) {
	getJSON(t, "/api/storage/info")
	getJSON(t, "/api/export/formats")
	getJSON(t, "/api/logs?limit=1")
}
