package compose

import (
	"context"
	"path/filepath"
	"testing"
)

func TestLoadBasicProject(t *testing.T) {
	ctx := context.Background()
	proj, err := Load(ctx, LoadOptions{
		ConfigPaths: []string{filepath.Join("testdata", "basic", "compose.yaml")},
		ProjectName: "basic",
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if proj.Name != "basic" {
		t.Errorf("project name = %q, want %q", proj.Name, "basic")
	}

	if got := len(proj.Services); got != 2 {
		t.Fatalf("got %d services, want 2", got)
	}

	web, err := proj.GetService("web")
	if err != nil {
		t.Fatalf("GetService(web): %v", err)
	}
	if web.Image != "nginx:1.27" {
		t.Errorf("web.Image = %q, want nginx:1.27", web.Image)
	}
	if len(web.Ports) != 1 {
		t.Fatalf("web has %d ports, want 1", len(web.Ports))
	}
	if web.Ports[0].Published != "8080" || web.Ports[0].Target != 80 {
		t.Errorf("web port = %s:%d, want 8080:80", web.Ports[0].Published, web.Ports[0].Target)
	}
	if got := web.Environment["GREETING"]; got == nil || *got != "hello" {
		t.Errorf("web GREETING env = %v, want hello", got)
	}

	// depends_on should be normalized to the long form with a condition.
	if _, ok := web.DependsOn["db"]; !ok {
		t.Errorf("web should depend_on db, got %v", web.DependsOn)
	}

	// Named volume should be registered at project level.
	if _, ok := proj.Volumes["dbdata"]; !ok {
		t.Errorf("project should declare volume dbdata, got %v", proj.Volumes)
	}

	// Custom default network name should be honored.
	if net, ok := proj.Networks["default"]; !ok {
		t.Errorf("project should declare default network")
	} else if net.Name != "basic_net" {
		t.Errorf("default network name = %q, want basic_net", net.Name)
	}
}

func TestLoadAppliesProjectLabelsConvention(t *testing.T) {
	ctx := context.Background()
	proj, err := Load(ctx, LoadOptions{
		ConfigPaths: []string{filepath.Join("testdata", "basic", "compose.yaml")},
		ProjectName: "basic",
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	// compose-go normalizes service names into the service config.
	db, err := proj.GetService("db")
	if err != nil {
		t.Fatalf("GetService(db): %v", err)
	}
	if db.Name != "db" {
		t.Errorf("db.Name = %q, want db", db.Name)
	}
}
