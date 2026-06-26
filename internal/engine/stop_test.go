package engine

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/compose"
	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func loadFrom(t *testing.T, path, name string) *types.Project {
	t.Helper()
	proj, err := compose.Load(context.Background(), compose.LoadOptions{
		ConfigPaths: []string{path},
		ProjectName: name,
	})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return proj
}

func TestStopAppliesSignalAndGrace(t *testing.T) {
	// Reuse the translate extras fixture which sets stop_signal/grace.
	path := filepath.Join("..", "translate", "testdata", "extras", "compose.yaml")
	proj := loadFrom(t, path, "extras")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)

	if err := e.Stop(context.Background(), proj, nil); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	calls := fake.CommandArgs()
	if firstMatch(calls, "stop --signal SIGINT --time 25 extras-app-1") == -1 {
		t.Errorf("stop did not apply signal/grace, calls: %v", calls)
	}
}
