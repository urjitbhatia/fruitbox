package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/urjitbhatia/fruitbox/internal/engine"
)

func newUpCommand(opts *globalOptions) *cobra.Command {
	var (
		detach        bool
		noBuild       bool
		removeOrphans bool
		noStart       bool
		wait          bool
		waitTimeout   int
		pull          string
		scaleFlags    []string
	)
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Create and start the project's containers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			scale, err := parseScale(scaleFlags)
			if err != nil {
				return err
			}
			e := opts.engine(cmd.OutOrStdout())
			return e.Up(cmd.Context(), proj, engine.UpOptions{
				Detach:        detach,
				NoBuild:       noBuild,
				RemoveOrphans: removeOrphans,
				Scale:         scale,
				NoStart:       noStart,
				Wait:          wait,
				WaitTimeout:   waitTimeout,
				Pull:          pull,
			})
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&detach, "detach", "d", false, "Run containers in the background")
	f.BoolVar(&noBuild, "no-build", false, "Don't build images, even if they're missing")
	f.BoolVar(&removeOrphans, "remove-orphans", false, "Remove containers for services not defined in the Compose file")
	f.BoolVar(&noStart, "no-start", false, "Don't start the services after creating them")
	f.BoolVar(&wait, "wait", false, "Wait for services to be running|healthy")
	f.IntVar(&waitTimeout, "wait-timeout", 0, "Max seconds to wait for the project to be running|healthy")
	f.StringVar(&pull, "pull", "policy", `Pull images before running ("always"|"missing"|"never")`)
	f.StringArrayVar(&scaleFlags, "scale", nil, "Scale SERVICE to NUM instances (SERVICE=NUM)")
	return cmd
}

// parseScale parses repeated SERVICE=NUM flags into a map.
func parseScale(flags []string) (map[string]int, error) {
	if len(flags) == 0 {
		return nil, nil
	}
	out := map[string]int{}
	for _, f := range flags {
		svc, num, ok := strings.Cut(f, "=")
		if !ok {
			return nil, fmt.Errorf("invalid --scale %q, want SERVICE=NUM", f)
		}
		n, err := strconv.Atoi(num)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("invalid --scale %q: NUM must be a non-negative integer", f)
		}
		out[svc] = n
	}
	return out, nil
}

func newDownCommand(opts *globalOptions) *cobra.Command {
	var (
		removeVolumes bool
		removeOrphans bool
		rmi           string
		timeout       int
	)
	cmd := &cobra.Command{
		Use:   "down",
		Short: "Stop and remove the project's containers, networks and (optionally) volumes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			down := engine.DownOptions{
				RemoveVolumes: removeVolumes,
				RemoveOrphans: removeOrphans,
				RemoveImages:  rmi,
			}
			if cmd.Flags().Changed("timeout") {
				down.Timeout = &timeout
			}
			e := opts.engine(cmd.OutOrStdout())
			return e.Down(cmd.Context(), proj, down)
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&removeVolumes, "volumes", "v", false, "Also remove named volumes declared in the volumes section")
	f.BoolVar(&removeOrphans, "remove-orphans", false, "Remove containers for services not defined in the Compose file")
	f.StringVar(&rmi, "rmi", "", `Remove images used by services ("local" or "all")`)
	f.IntVarP(&timeout, "timeout", "t", 0, "Shutdown timeout in seconds")
	return cmd
}
