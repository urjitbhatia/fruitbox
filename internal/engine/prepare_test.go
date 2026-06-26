package engine

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestUpGeneratesHostsAndHostnameMountsAndSysctls(t *testing.T) {
	proj := loadFrom(t, filepath.Join("testdata", "netcfg", "compose.yaml"), "netcfg")
	fake := &runner.Fake{}
	e := New(fake, io.Discard)
	e.StateDir = t.TempDir()

	if err := e.Up(context.Background(), proj, UpOptions{Detach: true}); err != nil {
		t.Fatalf("Up: %v", err)
	}

	calls := fake.CommandArgs()
	runCall := ""
	for _, c := range calls {
		if strings.Contains(c, "--name netcfg-app-1") {
			runCall = c
		}
	}
	if runCall == "" {
		t.Fatalf("no run call for app, calls:\n%s", strings.Join(calls, "\n"))
	}
	// The run command should bind-mount the generated /etc/hosts and /etc/hostname.
	if !strings.Contains(runCall, ":/etc/hosts:ro") {
		t.Errorf("run should mount /etc/hosts:\n%s", runCall)
	}
	if !strings.Contains(runCall, ":/etc/hostname:ro") {
		t.Errorf("run should mount /etc/hostname:\n%s", runCall)
	}
	// The generated files should exist with the right content.
	dir := filepath.Join(e.StateDir, "netcfg", "netcfg-app-1")
	hosts, err := os.ReadFile(filepath.Join(dir, "hosts"))
	if err != nil {
		t.Fatalf("read generated hosts: %v", err)
	}
	if !strings.Contains(string(hosts), "10.0.0.5\tdb.internal") {
		t.Errorf("hosts file missing extra host:\n%s", hosts)
	}
	hostname, err := os.ReadFile(filepath.Join(dir, "hostname"))
	if err != nil || strings.TrimSpace(string(hostname)) != "apphost" {
		t.Errorf("hostname file = %q (err %v), want apphost", hostname, err)
	}
	// Sysctl should be applied post-start via exec.
	if firstMatch(calls, "exec netcfg-app-1 sysctl -w net.core.somaxconn=1024") == -1 {
		t.Errorf("expected sysctl exec, calls:\n%s", strings.Join(calls, "\n"))
	}
}
