package engine

import (
	"context"
	"io"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestVolumeNames(t *testing.T) {
	proj := load(t, "basic") // declares volume dbdata -> basic_dbdata
	e := New(&runner.Fake{}, io.Discard)
	names := e.VolumeNames(proj)
	if len(names) != 1 || names[0] != "basic_dbdata" {
		t.Errorf("VolumeNames = %v, want [basic_dbdata]", names)
	}
}

func TestStatsDelegates(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Stats(context.Background(), proj, StatsOptions{NoStream: true, Format: "json"}); err != nil {
		t.Fatalf("Stats: %v", err)
	}
	joined := ""
	for _, c := range fake.CommandArgs() {
		joined = c
	}
	if firstMatch(fake.CommandArgs(), "stats --no-stream --format json basic-db-1 basic-web-1") == -1 {
		t.Errorf("stats args unexpected: %v (%s)", fake.CommandArgs(), joined)
	}
}

func TestExportDelegates(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	if err := e.Export(context.Background(), proj, "web", "/tmp/web.tar", 1); err != nil {
		t.Fatalf("Export: %v", err)
	}
	if firstMatch(fake.CommandArgs(), "export --output /tmp/web.tar basic-web-1") == -1 {
		t.Errorf("export args unexpected: %v", fake.CommandArgs())
	}
}
