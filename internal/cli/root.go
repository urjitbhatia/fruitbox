// Package cli implements the fruitbox command-line interface, a Docker
// Compose-compatible front end that drives Apple's `container` runtime.
package cli

import (
	"context"
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

func (g *globalOptions) load(ctx context.Context) (*types.Project, error) {
	return compose.Load(ctx, compose.LoadOptions{
		ConfigPaths: g.files,
		WorkingDir:  g.projectDir,
		ProjectName: g.projectName,
		Profiles:    g.profiles,
		EnvFiles:    g.envFiles,
	})
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

	pf := root.PersistentFlags()
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
		newVersionCommand(),
	)
	return root
}

// Execute runs the root command, returning a process exit code.
func Execute() int {
	root := NewRootCommand()
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	if err := root.Execute(); err != nil {
		root.PrintErrln("Error:", err)
		return 1
	}
	return 0
}
