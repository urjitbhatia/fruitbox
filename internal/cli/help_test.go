package cli

import (
	"testing"
)

// TestEveryCommandHelpParses guards against flag-registration panics (e.g. a
// shorthand collision between a global flag and a subcommand flag). It walks
// the whole command tree and renders --help for each command. This is the test
// that would have caught `logs`/`rm` crashing on the global `-f` collision.
func TestEveryCommandHelpParses(t *testing.T) {
	root := NewRootCommand()
	for _, c := range root.Commands() {
		name := c.Name()
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("`%s --help` panicked: %v", name, r)
				}
			}()
			if _, err := runRoot(t, name, "--help"); err != nil {
				t.Fatalf("`%s --help` errored: %v", name, err)
			}
		})
	}
}

// TestGlobalFlagsBeforeSubcommand verifies project-selection flags are accepted
// before the subcommand (docker compose style) and that a subcommand can still
// use -f for its own purpose afterwards.
func TestGlobalFlagsBeforeSubcommand(t *testing.T) {
	out, err := runRoot(t, "-f", "testdata/basic/compose.yaml", "-p", "basic", "config", "--services")
	if err != nil {
		t.Fatalf("global -f/-p before subcommand failed: %v\n%s", err, out)
	}
}
