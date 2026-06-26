package cli

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// These tests assert that fruitbox produces output identical to the real
// `docker compose` for the runtime-independent `config` surface (parse,
// resolve, interpolate, merge, normalize, render). fruitbox runs in-process;
// docker compose is shelled out as the reference oracle.
//
// They are skipped automatically when `docker compose` is unavailable, so the
// suite stays hermetic on CI without Docker. Run them locally with Docker
// installed to guard real compatibility.

// dockerComposeAvailable reports whether `docker compose version` works.
func dockerComposeAvailable(t *testing.T) bool {
	t.Helper()
	cmd := exec.Command("docker", "compose", "version")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// dockerComposeConfig runs `docker compose <args>` and returns stdout.
func dockerComposeConfig(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("docker", append([]string{"compose"}, args...)...)
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
