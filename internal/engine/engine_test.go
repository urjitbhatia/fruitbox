package engine

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/compose"
	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func load(t *testing.T, dir string) *types.Project {
	t.Helper()
	proj, err := compose.Load(context.Background(), compose.LoadOptions{
		ConfigPaths: []string{filepath.Join("testdata", dir, "compose.yaml")},
		ProjectName: dir,
	})
	if err != nil {
		t.Fatalf("load %s: %v", dir, err)
	}
	return proj
}

func TestDependencyOrder(t *testing.T) {
	proj := load(t, "basic")
	order, err := DependencyOrder(proj)
	if err != nil {
		t.Fatal(err)
	}
	// web depends_on db, so db must come first.
	idxDB, idxWeb := indexOf(order, "db"), indexOf(order, "web")
	if idxDB == -1 || idxWeb == -1 {
		t.Fatalf("order missing services: %v", order)
	}
	if idxDB > idxWeb {
		t.Errorf("db should start before web, got order %v", order)
	}
}

func TestUpCreatesResourcesThenServicesInOrder(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)

	if err := e.Up(context.Background(), proj, UpOptions{Detach: true}); err != nil {
		t.Fatalf("Up: %v", err)
	}

	calls := fake.CommandArgs()
	// Expect: volume create, network create, then db run, then web run.
	posNet := firstMatch(calls, "network create")
	posVol := firstMatch(calls, "volume create")
	posDB := firstMatch(calls, "--name basic-db-1")
	posWeb := firstMatch(calls, "--name basic-web-1")

	for label, pos := range map[string]int{"network create": posNet, "volume create": posVol, "db run": posDB, "web run": posWeb} {
		if pos == -1 {
			t.Fatalf("missing expected call: %s\nall calls:\n%s", label, strings.Join(calls, "\n"))
		}
	}
	if posDB > posWeb {
		t.Errorf("db should be started before web: db@%d web@%d", posDB, posWeb)
	}
	if posNet > posDB || posVol > posDB {
		t.Errorf("resources should be created before services: net@%d vol@%d db@%d", posNet, posVol, posDB)
	}
}

func TestDownStopsAndRemovesInReverseOrder(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)

	if err := e.Down(context.Background(), proj, DownOptions{RemoveVolumes: true}); err != nil {
		t.Fatalf("Down: %v", err)
	}
	calls := fake.CommandArgs()

	posStopWeb := firstMatch(calls, "stop basic-web-1")
	posStopDB := firstMatch(calls, "stop basic-db-1")
	posNetDel := firstMatch(calls, "network delete basic_net")
	posVolDel := firstMatch(calls, "volume delete basic_dbdata")

	if posStopWeb == -1 || posStopDB == -1 {
		t.Fatalf("missing stop calls:\n%s", strings.Join(calls, "\n"))
	}
	// Reverse order: web stops before db.
	if posStopWeb > posStopDB {
		t.Errorf("web should stop before db on teardown: web@%d db@%d", posStopWeb, posStopDB)
	}
	if posNetDel == -1 || posVolDel == -1 {
		t.Errorf("network/volume should be deleted:\n%s", strings.Join(calls, "\n"))
	}
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

func firstMatch(calls []string, substr string) int {
	for i, c := range calls {
		if strings.Contains(c, substr) {
			return i
		}
	}
	return -1
}
