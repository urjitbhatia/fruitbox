//go:build integration

// Package integration drives the fruitbox binary against the *real* Apple
// `container` runtime. These tests are excluded from the normal suite and run
// only with:
//
//	go test -tags=integration ./test/integration/...
//
// They require macOS on Apple silicon with `container` installed and the system
// service running; otherwise they skip. Each test uses a unique project name
// and tears the project down afterward.
package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fruitboxBin builds the fruitbox binary once per test run and returns its path.
func fruitboxBin(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "fruitbox")
	// The repo root is two levels up from test/integration.
	build := exec.Command("go", "build", "-o", bin, "../../cmd/fruitbox")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("build fruitbox: %v", err)
	}
	return bin
}

// requireRuntime skips the test unless the container system is running.
func requireRuntime(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("container"); err != nil {
		t.Skip("container CLI not installed; skipping integration test")
	}
	out, err := exec.Command("container", "system", "status").CombinedOutput()
	if err != nil || !strings.Contains(string(out), "running") {
		t.Skip("container system not running; skipping integration test")
	}
}

type cli struct {
	t       *testing.T
	bin     string
	project string
	dir     string
}

// run executes `fruitbox -p <project> <args...>` in the project directory.
func (c *cli) run(args ...string) (string, error) {
	c.t.Helper()
	full := append([]string{"-p", c.project}, args...)
	cmd := exec.Command(c.bin, full...)
	cmd.Dir = c.dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

func (c *cli) mustRun(args ...string) string {
	c.t.Helper()
	out, err := c.run(args...)
	if err != nil {
		c.t.Fatalf("fruitbox %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return out
}

const composeYAML = `services:
  greeter:
    image: docker.io/library/alpine:3.20
    command: ["sh", "-c", "echo GREETER_READY; sleep 3600"]
  web:
    image: docker.io/library/nginx:1.27
    ports:
      - "18099:80"
`

func TestLifecycleAgainstRealRuntime(t *testing.T) {
	requireRuntime(t)
	bin := fruitboxBin(t)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(composeYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	c := &cli{t: t, bin: bin, project: "fbintegration", dir: dir}

	// Always tear down, even on failure.
	t.Cleanup(func() { _, _ = c.run("down", "--volumes") })

	// config validates without the runtime.
	if out := c.mustRun("config", "--services"); !strings.Contains(out, "greeter") || !strings.Contains(out, "web") {
		t.Fatalf("config --services unexpected:\n%s", out)
	}

	// up -d brings up real containers.
	c.mustRun("up", "-d")

	// ps reports both services running (poll briefly for startup).
	if !eventually(t, 30*time.Second, func() bool {
		out, _ := c.run("ps")
		return strings.Count(out, "running") >= 2
	}) {
		out, _ := c.run("ps")
		t.Fatalf("services did not reach running:\n%s", out)
	}

	// logs show the greeter's output.
	if out := c.mustRun("logs", "greeter"); !strings.Contains(out, "GREETER_READY") {
		t.Errorf("greeter logs missing marker:\n%s", out)
	}

	// exec runs a command in the container, with flag passthrough.
	if out := c.mustRun("exec", "greeter", "echo", "hi-from-exec"); !strings.Contains(out, "hi-from-exec") {
		t.Errorf("exec output unexpected:\n%s", out)
	}

	// A second up with unchanged config must reuse, not recreate.
	if out := c.mustRun("up", "-d"); !strings.Contains(out, "up-to-date") {
		t.Errorf("second up should report up-to-date:\n%s", out)
	}

	// down removes everything.
	c.mustRun("down")
	if out, _ := c.run("ps"); strings.Contains(out, "running") {
		t.Errorf("containers still running after down:\n%s", out)
	}
}

func TestRecreateOnConfigChange(t *testing.T) {
	requireRuntime(t)
	bin := fruitboxBin(t)
	dir := t.TempDir()
	base := `services:
  app:
    image: docker.io/library/alpine:3.20
    command: ["sh", "-c", "sleep 3600"]
`
	write := func(content string) {
		if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(base)
	c := &cli{t: t, bin: bin, project: "fbintrecreate", dir: dir}
	t.Cleanup(func() { _, _ = c.run("down") })

	c.mustRun("up", "-d")
	// Change the config (add an env var) -> up must recreate app.
	write(base + "    environment:\n      CHANGED: \"1\"\n")
	out := c.mustRun("up", "-d")
	if !strings.Contains(out, "Recreating") {
		t.Errorf("config change should recreate app:\n%s", out)
	}
	if got := c.mustRun("exec", "app", "printenv", "CHANGED"); !strings.Contains(got, "1") {
		t.Errorf("recreated container missing new env CHANGED=1:\n%s", got)
	}
}

func eventually(t *testing.T, d time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(time.Second)
	}
	return false
}
