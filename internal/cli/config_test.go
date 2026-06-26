package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func runRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func TestConfigRendersYAML(t *testing.T) {
	file := filepath.Join("testdata", "basic", "compose.yaml")
	out, err := runRoot(t, "-f", file, "-p", "basic", "config")
	if err != nil {
		t.Fatalf("config: %v\n%s", err, out)
	}
	if !strings.Contains(out, "nginx:1.27") {
		t.Errorf("config output missing image:\n%s", out)
	}
	if !strings.Contains(out, "postgres:16") {
		t.Errorf("config output missing db image:\n%s", out)
	}
}

func TestConfigServicesList(t *testing.T) {
	file := filepath.Join("testdata", "basic", "compose.yaml")
	out, err := runRoot(t, "-f", file, "-p", "basic", "config", "--services")
	if err != nil {
		t.Fatalf("config --services: %v\n%s", err, out)
	}
	got := strings.Fields(out)
	if len(got) != 2 || got[0] != "db" || got[1] != "web" {
		t.Errorf("config --services = %q, want [db web]", got)
	}
}

func TestVersion(t *testing.T) {
	out, err := runRoot(t, "version")
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if !strings.Contains(out, "fruitbox version") {
		t.Errorf("unexpected version output: %q", out)
	}
}
