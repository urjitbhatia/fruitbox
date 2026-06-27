package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// requireCompatOptIn skips the version-sensitive docker compose differential
// tests unless FRUITBOX_COMPAT is set. They compare against whatever local
// `docker compose` is installed, so they only hold against a pinned version
// (currently v5.0.2). The default `go test ./...`, including CI, skips them and
// stays hermetic; run them with `FRUITBOX_COMPAT=1 go test ./internal/cli/...`.
func requireCompatOptIn(t *testing.T) {
	t.Helper()
	if os.Getenv("FRUITBOX_COMPAT") == "" {
		t.Skip("set FRUITBOX_COMPAT=1 to run version-sensitive docker compose differential tests")
	}
}

// These tests assert that fruitbox produces output identical to the real
// `docker compose` for the runtime-independent `config` surface (parse,
// resolve, interpolate, merge, normalize, render). fruitbox runs in-process;
// docker compose is shelled out as the reference oracle.
//
// They depend on the exact installed docker compose version, so they are opt-in
// (gated on FRUITBOX_COMPAT) and skip in the default `go test ./...` to keep CI
// hermetic. Run them against a pinned docker compose with
// `FRUITBOX_COMPAT=1 go test ./internal/cli/...` or `make test-compat`.

// composeInvocation is the command used to shell out to the reference compose
// implementation. It defaults to `docker compose` (the plugin) but can be
// overridden via FRUITBOX_COMPOSE_BIN to point at a specific standalone binary,
// e.g. `FRUITBOX_COMPOSE_BIN=/path/to/docker-compose-v5.0.2`. This is what lets
// scripts/compat-matrix.sh run the same diff against several pinned versions.
func composeInvocation() []string {
	if v := os.Getenv("FRUITBOX_COMPOSE_BIN"); v != "" {
		return strings.Fields(v)
	}
	return []string{"docker", "compose"}
}

// composeCommand builds an *exec.Cmd for the configured compose binary with the
// given subcommand args appended.
func composeCommand(args ...string) *exec.Cmd {
	inv := composeInvocation()
	full := append(append([]string{}, inv[1:]...), args...)
	return exec.Command(inv[0], full...)
}

// dockerComposeAvailable reports whether the configured compose binary runs.
func dockerComposeAvailable(t *testing.T) bool {
	t.Helper()
	if err := composeCommand("version").Run(); err != nil {
		return false
	}
	return true
}

// dockerComposeConfig runs `<compose> <args>` and returns stdout.
func dockerComposeConfig(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := composeCommand(args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{} // discard the obsolete-version warning etc.
	err := cmd.Run()
	return out.String(), err
}

// normalizeLines sorts the non-empty lines of s, for order-insensitive
// comparison of list-style outputs.
func normalizeLines(s string) string {
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// compatCase is a single differential scenario: the same args go to both tools.
type compatCase struct {
	name string
	args []string // everything after the tool name (incl. -f, -p, config, flags)
	// listOutput true compares order-insensitively (list flags); false compares
	// the rendered document verbatim.
	listOutput bool
}

func TestConfigMatchesDockerCompose(t *testing.T) {
	requireCompatOptIn(t)
	if !dockerComposeAvailable(t) {
		t.Skip("docker compose not available; skipping differential compatibility tests")
	}

	rich := filepath.Join("testdata", "compat", "rich.yaml")
	override := filepath.Join("testdata", "compat", "override.yaml")

	cases := []compatCase{
		{"services", []string{"-f", rich, "-p", "rich", "config", "--services"}, true},
		{"networks", []string{"-f", rich, "-p", "rich", "config", "--networks"}, true},
		{"volumes", []string{"-f", rich, "-p", "rich", "config", "--volumes"}, true},
		{"profiles", []string{"-f", rich, "-p", "rich", "config", "--profiles"}, true},
		{"images", []string{"-f", rich, "-p", "rich", "config", "--images"}, true},
		{"services-with-profile", []string{"-f", rich, "-p", "rich", "--profile", "tools", "config", "--services"}, true},
		{"images-merged", []string{"-f", rich, "-f", override, "-p", "rich", "config", "--images"}, true},
		{"full-yaml", []string{"-f", rich, "-p", "rich", "config"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want, derr := dockerComposeConfig(t, tc.args...)
			got, gerr := runRoot(t, tc.args...)
			if derr != nil {
				t.Fatalf("docker compose errored: %v", derr)
			}
			if gerr != nil {
				t.Fatalf("fruitbox errored: %v\n%s", gerr, got)
			}
			w, g := want, got
			if tc.listOutput {
				w, g = normalizeLines(want), normalizeLines(got)
			} else {
				w, g = strings.TrimRight(want, "\n"), strings.TrimRight(got, "\n")
			}
			if w != g {
				t.Errorf("output mismatch for `%s`:\n--- docker compose ---\n%s\n--- fruitbox ---\n%s", strings.Join(tc.args, " "), w, g)
			}
		})
	}
}
