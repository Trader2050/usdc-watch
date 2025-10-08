package config

import (
	"os"
	"testing"
)

func TestLoadEndpoints(t *testing.T) {
	content := `# endpoints
[[rpc.endpoints]]
name = "first"
url = "https://a.example"

[[rpc.endpoints]]
url = "https://b.example"
`

	file, err := os.CreateTemp(t.TempDir(), "endpoints-*.toml")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		t.Fatalf("WriteString: %v", err)
	}

	endpoints, err := LoadEndpoints(file.Name())
	if err != nil {
		t.Fatalf("LoadEndpoints error: %v", err)
	}
	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}
	if endpoints[0].Name != "first" {
		t.Fatalf("first endpoint name mismatch: %s", endpoints[0].Name)
	}
	if endpoints[0].URL != "https://a.example" {
		t.Fatalf("first endpoint url mismatch: %s", endpoints[0].URL)
	}
	if endpoints[1].Name != "endpoint-2" {
		t.Fatalf("second endpoint default name mismatch: %s", endpoints[1].Name)
	}
	if endpoints[1].URL != "https://b.example" {
		t.Fatalf("second endpoint url mismatch: %s", endpoints[1].URL)
	}
}

func TestLoadEndpointsErrors(t *testing.T) {
	if _, err := LoadEndpoints("missing-file.toml"); err == nil {
		t.Fatalf("expected error for missing file")
	}
}
