// Package cli implements the fruitbox command-line interface, a Docker
// Compose-compatible front end that drives Apple's `container` runtime.
package cli

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/spf13/cobra"
	"github.com/urjitbhatia/fruitbox/internal/compose"
	"github.com/urjitbhatia/fruitbox/internal/engine"
	"github.com/urjitbhatia/fruitbox/internal/runner"
)

// globalOptions holds the project-selection flags shared by all subcommands,
// mirroring `docker compose` global flags.
type globalOptions struct {
	files       []string
	projectName string
	projectDir  string
	profiles    []string
	envFiles    []string
	binary      string
}

// baseLoadOptions returns the compose load options derived from the global
// flags, before any per-command overrides.
func (g *globalOptions) baseLoadOptions() compose.LoadOptions {
	return compose.LoadOptions{
		ConfigPaths: g.files,
		WorkingDir:  g.projectDir,
		ProjectName: g.projectName,
		Profiles:    g.profiles,
		EnvFiles:    g.envFiles,
	}
}

func (g *globalOptions) load(ctx context.Context) (*types.Project, error) {
	return compose.Load(ctx, g.baseLoadOptions())
}

// loadWith loads the project applying per-command overrides to the load options.
func (g *globalOptions) loadWith(ctx context.Context, mutate func(*compose.LoadOptions)) (*types.Project, error) {
	lo := g.baseLoadOptions()
	if mutate != nil {
		mutate(&lo)
	}
	return compose.Load(ctx, lo)
}

func (g *globalOptions) engine(out io.Writer) *engine.Engine {
	return engine.New(runner.NewExec(g.binary), out)
}

// NewRootCommand builds the top-level fruitbox command tree.
func NewRootCommand() *cobra.Command {
	opts := &globalOptions{}

	root := &cobra.Command{
		Use:           "fruitbox",
		Short:         "Docker Compose-compatible orchestration for Apple's container runtime",
		Long:          "fruitbox runs multi-container applications defined in Compose files on top of Apple's native `container` CLI.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Project-selection flags are parsed BEFORE the subcommand (e.g.
	// `fruitbox -f x.yml -p demo up`), matching docker compose. They are
	// registered as root-local flags (not persistent) with TraverseChildren so
	// they are consumed during traversal and never merged into subcommands.
	// This is what lets subcommands reuse `-f` (logs --follow, rm --force)
	// without a shorthand collision panic against the global `-f/--file`.
	root.TraverseChildren = true
	pf := root.Flags()
	pf.StringArrayVarP(&opts.files, "file", "f", nil, "Compose configuration file(s) (repeatable)")
	pf.StringVarP(&opts.projectName, "project-name", "p", "", "Project name")
	pf.StringVar(&opts.projectDir, "project-directory", "", "Working directory for the project")
	pf.StringArrayVar(&opts.profiles, "profile", nil, "Compose profile(s) to enable")
	pf.StringArrayVar(&opts.envFiles, "env-file", nil, "Environment file(s) to load")
	pf.StringVar(&opts.binary, "container-binary", "container", "Path to the Apple container CLI")

	root.AddCommand(
		newConfigCommand(opts),
		newUpCommand(opts),
		newDownCommand(opts),
		newPsCommand(opts),
		newLogsCommand(opts),
		newBuildCommand(opts),
		newStartCommand(opts),
		newStopCommand(opts),
		newRestartCommand(opts),
		newKillCommand(opts),
		newPullCommand(opts),
		newExecCommand(opts),
		newRunCommand(opts),
		newImagesCommand(opts),
		newPortCommand(opts),
		newCpCommand(opts),
		newLsCommand(opts),
		newWaitCommand(opts),
		newTopCommand(opts),
		newPauseCommand(opts),
		newUnpauseCommand(opts),
		newCreateCommand(opts),
		newRmCommand(opts),
		newPushCommand(opts),
		newScaleCommand(opts),
		newAttachCommand(opts),
		newEventsCommand(opts),
		newWatchCommand(opts),
		newVersionCommand(),
	)
	return root
}

// ttyAvailable reports whether a pseudo-TTY can be allocated — i.e. BOTH stdin
// and stdout are interactive terminals. Like docker compose, fruitbox
// auto-disables TTY when either is redirected (pipe/file/CI), since the
// container runtime rejects a TTY without a real terminal on both ends.
func ttyAvailable() bool {
	return isCharDevice(os.Stdin) && isCharDevice(os.Stdout)
}

func isCharDevice(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// Execute runs the root command, returning a process exit code.
func Execute() int {
	root := NewRootCommand()
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	if err := root.Execute(); err != nil {
		// --exit-code-from / --abort-on-* propagate a container's exit code.
		var exit engine.ExitError
		if errors.As(err, &exit) {
			return exit.Code
		}
		root.PrintErrln("Error:", err)
		return 1
	}
	return 0
}
