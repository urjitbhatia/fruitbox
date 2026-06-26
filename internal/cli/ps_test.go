package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/engine"
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

func TestFilterPs(t *testing.T) {
	in := []engine.ContainerStatus{
		{Name: "p-web-1", Service: "web", Status: "running"},
		{Name: "p-db-1", Service: "db", Status: "exited"},
		{Name: "p-cache-1", Service: "cache", Status: "not created"},
	}
	names := func(ss []engine.ContainerStatus) []string {
		var out []string
		for _, s := range ss {
			out = append(out, s.Name)
		}
		return out
	}

	if got := names(filterPs(in, psFilter{})); len(got) != 1 || got[0] != "p-web-1" {
		t.Errorf("default ps should show only running, got %v", got)
	}
	if got := names(filterPs(in, psFilter{all: true})); len(got) != 2 {
		t.Errorf("--all should show running+exited (not 'not created'), got %v", got)
	}
	if got := names(filterPs(in, psFilter{status: "exited"})); len(got) != 1 || got[0] != "p-db-1" {
		t.Errorf("--status exited should show db, got %v", got)
	}
	if got := names(filterPs(in, psFilter{all: true, name: "web"})); len(got) != 1 || got[0] != "p-web-1" {
		t.Errorf("name filter should match web, got %v", got)
	}
}

func TestParsePsFilters(t *testing.T) {
	pf, err := parsePsFilters(true, "", []string{"status=running", "name=db"})
	if err != nil {
		t.Fatal(err)
	}
	if !pf.all || pf.status != "running" || pf.name != "db" {
		t.Errorf("unexpected parse: %+v", pf)
	}
	if _, err := parsePsFilters(false, "", []string{"bogus"}); err == nil {
		t.Error("expected error for malformed filter")
	}
	if _, err := parsePsFilters(false, "", []string{"weird=x"}); err == nil {
		t.Error("expected error for unsupported filter key")
	}
}
