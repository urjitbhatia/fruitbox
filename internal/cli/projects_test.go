package cli

import (
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/engine"
)

func TestFilterProjects(t *testing.T) {
	in := []engine.ProjectSummary{
		{Name: "alpha", ContainerCount: 2, RunningCount: 1},
		{Name: "beta", ContainerCount: 1, RunningCount: 0},
		{Name: "alphabet", ContainerCount: 1, RunningCount: 1},
	}
	names := func(ps []engine.ProjectSummary) []string {
		var out []string
		for _, p := range ps {
			out = append(out, p.Name)
		}
		return out
	}

	// Default: only projects with running containers.
	got, _ := filterProjects(in, false, nil)
	if len(got) != 2 {
		t.Errorf("default should hide stopped projects, got %v", names(got))
	}
	// --all: all projects.
	got, _ = filterProjects(in, true, nil)
	if len(got) != 3 {
		t.Errorf("--all should show all, got %v", names(got))
	}
	// --filter name=alpha (substring) with --all.
	got, _ = filterProjects(in, true, []string{"name=alpha"})
	if len(got) != 2 {
		t.Errorf("name filter should match alpha and alphabet, got %v", names(got))
	}
	// bad filter.
	if _, err := filterProjects(in, true, []string{"status=x"}); err == nil {
		t.Error("expected error for unsupported filter key")
	}
}
