package cli

import (
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// knownFlagGaps records, per command, the docker compose flags fruitbox does
// NOT yet implement. It is a *ratchet*: TestFlagParity asserts the live gap set
// equals this baseline, so closing a gap (or regressing one — silently losing a
// flag) forces an update here and is impossible to miss. Generated/verified
// against docker compose v5.0.2; regenerate with scripts/compat-audit.sh.
var knownFlagGaps = map[string][]string{
	"attach": {"detach-keys", "no-stdin", "sig-proxy"},
	"build":  {"build-arg", "builder", "check", "memory", "no-cache", "print", "provenance", "pull", "push", "quiet", "sbom", "ssh", "with-dependencies"},
	"config": {"environment", "lock-image-digests", "resolve-image-digests", "variables"},
	"cp":     {"all", "archive", "follow-link"},
	"create": {"build", "force-recreate", "no-build", "no-recreate", "pull", "quiet-pull", "remove-orphans", "yes"},
	"events": {"json", "since", "until"},
	"exec":   {"privileged"},
	"images": {"format"},
	"logs":   {"no-color", "no-log-prefix", "since", "timestamps", "until"},
	"ls":     {"all", "filter", "format"},
	"port":   {"index"},
	"ps":     {"all", "filter", "orphans", "status"},
	"pull":   {"ignore-buildable", "ignore-pull-failures", "include-deps", "policy", "quiet"},
	"push":   {"ignore-push-failures", "include-deps", "quiet"},
	"rm":     {"volumes"},
	"run":    {"env-from-file", "pull", "quiet", "quiet-build", "quiet-pull", "use-aliases"},
	"scale":  {"no-deps"},
	"start":  {"wait", "wait-timeout"},
	"up":     {"abort-on-container-exit", "abort-on-container-failure", "always-recreate-deps", "attach", "attach-dependencies", "build", "exit-code-from", "force-recreate", "menu", "no-attach", "no-color", "no-deps", "no-log-prefix", "no-recreate", "quiet-build", "quiet-pull", "renew-anon-volumes", "timeout", "timestamps", "watch", "yes"},
	"wait":   {"down-project"},
	"watch":  {"no-up", "prune", "quiet"},
}

// flagRe matches long flags; uppercase is allowed because docker spells some
// flags with caps (e.g. `--no-TTY` on run).
var flagRe = regexp.MustCompile(`--([A-Za-z][A-Za-z0-9-]*[A-Za-z0-9])`)

// dockerFlags parses the long flags from `docker compose <cmd> --help`.
func dockerFlags(t *testing.T, cmd string) map[string]bool {
	t.Helper()
	out, _ := exec.Command("docker", "compose", cmd, "--help").CombinedOutput()
	set := map[string]bool{}
	for _, m := range flagRe.FindAllStringSubmatch(string(out), -1) {
		set[m[1]] = true
	}
	return set
}

// fruitboxFlags returns the flags fruitbox accepts for a command: the command's
// own flags plus the root-level global flags (parsed before the subcommand).
func fruitboxFlags(root, cmd *cobra.Command) map[string]bool {
	set := map[string]bool{}
	add := func(f *pflag.Flag) { set[f.Name] = true }
	cmd.Flags().VisitAll(add)
	root.Flags().VisitAll(add) // global project-selection flags
	return set
}

// TestFlagParity asserts fruitbox's per-command flag gaps vs docker compose
// exactly equal the recorded baseline. A diff means either a gap was closed
// (shrink the baseline) or a flag regressed/docker changed (investigate).
func TestFlagParity(t *testing.T) {
	if _, err := exec.Command("docker", "compose", "version").Output(); err != nil {
		t.Skip("docker compose not available; skipping flag-parity ratchet")
	}
	root := NewRootCommand()
	byName := map[string]*cobra.Command{}
	for _, c := range root.Commands() {
		byName[c.Name()] = c
	}
	// Globals appear on every command; don't count them as per-command gaps.
	globals := fruitboxFlags(root, root)
	dockerGlobals := dockerFlags(t, "")

	for cmdName := range knownFlagGaps {
		t.Run(cmdName, func(t *testing.T) {
			fbCmd, ok := byName[cmdName]
			if !ok {
				t.Fatalf("fruitbox has no command %q", cmdName)
			}
			dFlags := dockerFlags(t, cmdName)
			fFlags := fruitboxFlags(root, fbCmd)

			var missing []string
			for f := range dFlags {
				if fFlags[f] || globals[f] || dockerGlobals[f] {
					continue
				}
				missing = append(missing, f)
			}
			sort.Strings(missing)

			want := append([]string(nil), knownFlagGaps[cmdName]...)
			sort.Strings(want)

			if strings.Join(missing, ",") != strings.Join(want, ",") {
				t.Errorf("flag gap for %q changed.\n  live baseline: %v\n  recorded:      %v\n"+
					"If you implemented a flag, remove it from knownFlagGaps. "+
					"If a flag regressed or docker changed, investigate.", cmdName, missing, want)
			}
		})
	}
}
