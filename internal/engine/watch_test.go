package engine

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestSyncTargetPath(t *testing.T) {
	cases := []struct {
		triggerPath, target, changed, want string
	}{
		{"/host/src", "/app", "/host/src/main.go", "/app/main.go"},
		{"/host/src", "/app", "/host/src/pkg/util.go", "/app/pkg/util.go"},
		{"/host/src", "/app", "/host/src", "/app"},
	}
	for _, c := range cases {
		if got := syncTargetPath(c.triggerPath, c.target, c.changed); got != c.want {
			t.Errorf("syncTargetPath(%q,%q,%q) = %q, want %q", c.triggerPath, c.target, c.changed, got, c.want)
		}
	}
}

func TestWatchIgnored(t *testing.T) {
	if !watchIgnored("/src/node_modules/x.js", []string{"node_modules"}) {
		t.Error("should ignore node_modules segment")
	}
	if !watchIgnored("/src/a.log", []string{"*.log"}) {
		t.Error("should ignore *.log by basename")
	}
	if watchIgnored("/src/main.go", []string{"*.log"}) {
		t.Error("should not ignore main.go")
	}
}

func TestWatchSyncsChangedFile(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(srcDir, "main.go")
	if err := os.WriteFile(file, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	proj := load(t, "basic")
	web, _ := proj.GetService("web")
	web.Develop = &types.DevelopConfig{Watch: []types.Trigger{{
		Path:   srcDir,
		Target: "/app",
		Action: types.WatchActionSync,
	}}}
	proj.Services["web"] = web

	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	// 2 polls: first seeds the snapshot, second detects the (changed-since-seed)
	// file. To force a change, bump the mtime between rounds via a custom sleep.
	round := 0
	e.Sleep = func(_ context.Context, _ time.Duration) error {
		round++
		// After the seeding round, modify the file so the next scan sees it.
		future := time.Unix(900000000, 0)
		_ = os.Chtimes(file, future, future)
		return nil
	}
	e.Now = func() time.Time { return time.Unix(0, 0) }

	if err := e.Watch(context.Background(), proj, 2, WatchOptions{NoUp: true}); err != nil {
		t.Fatalf("Watch: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "cp "+file+" basic-web-1:/app/main.go") == -1 {
		t.Errorf("expected sync cp for changed file, calls: %v", fake.CommandArgs())
	}
}

func TestWatchUpsFirstByDefault(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	// maxPolls=1 (seed snapshot, no firing); default opts -> should up first.
	if err := e.Watch(context.Background(), proj, 1, WatchOptions{}); err != nil {
		t.Fatalf("Watch: %v", err)
	}
	// Up runs containers (detached).
	if firstMatch(fake.CommandArgs(), "run --name basic-db-1 --detach") == -1 {
		t.Errorf("watch should up the project first, calls: %v", fake.CommandArgs())
	}
}

func TestWatchNoUpSkipsUp(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Watch(context.Background(), proj, 1, WatchOptions{NoUp: true}); err != nil {
		t.Fatalf("Watch: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "run --name basic-db-1") != -1 {
		t.Errorf("--no-up should not start services, calls: %v", fake.CommandArgs())
	}
}

func TestWatchPruneRemovesDeleted(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(srcDir, "gone.go")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	proj := load(t, "basic")
	web, _ := proj.GetService("web")
	web.Develop = &types.DevelopConfig{Watch: []types.Trigger{{
		Path: srcDir, Target: "/app", Action: types.WatchActionSync,
	}}}
	proj.Services["web"] = web

	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	round := 0
	e.Sleep = func(_ context.Context, _ time.Duration) error {
		round++
		_ = os.Remove(file) // delete the file after the seeding round
		return nil
	}
	e.Now = func() time.Time { return time.Unix(0, 0) }

	if err := e.Watch(context.Background(), proj, 2, WatchOptions{NoUp: true, Prune: true}); err != nil {
		t.Fatalf("Watch: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "exec basic-web-1 rm -rf /app/gone.go") == -1 {
		t.Errorf("--prune should remove the deleted file in the container, calls: %v", fake.CommandArgs())
	}
}
