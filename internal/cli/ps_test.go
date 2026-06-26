package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeStubContainer writes a fake `container` binary that prints running
// inspect JSON, so ps can resolve statuses without the real runtime.
func writeStubContainer(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "container")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = inspect ]; then echo '[{\"status\":\"running\"}]'; fi\n" +
		"exit 0\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPsServices(t *testing.T) {
	stub := writeStubContainer(t)
	file := filepath.Join("testdata", "basic", "compose.yaml")
	out, err := runRoot(t, "-f", file, "-p", "basic", "--container-binary", stub, "ps", "--services")
	if err != nil {
		t.Fatalf("ps --services: %v\n%s", err, out)
	}
	got := strings.Fields(out)
	if len(got) != 2 || got[0] != "db" || got[1] != "web" {
		t.Errorf("ps --services = %v, want [db web]", got)
	}
}

func TestPsFormatJSON(t *testing.T) {
	stub := writeStubContainer(t)
	file := filepath.Join("testdata", "basic", "compose.yaml")
	out, err := runRoot(t, "-f", file, "-p", "basic", "--container-binary", stub, "ps", "--format", "json")
	if err != nil {
		t.Fatalf("ps --format json: %v\n%s", err, out)
	}
	if !strings.Contains(out, `"Name"`) || !strings.Contains(out, "basic-web-1") {
		t.Errorf("json output missing expected fields:\n%s", out)
	}
}
