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
		detach          bool
		noBuild         bool
		removeOrphans   bool
		noStart         bool
		wait            bool
		waitTimeout     int
		pull            string
		forceRecreate   bool
		noRecreate      bool
		noDeps          bool
		timeout         int
		abortOnExit     bool
		abortOnFailure  bool
		exitCodeFrom    string
		attach          []string
		noAttach        []string
		attachDeps      bool
		noLogPrefix     bool
		noColor         bool
		timestamps      bool
		quietBuild      bool
		quietPull       bool
		build           bool
		watch           bool
		yes             bool
		recreateDeps    bool
		renewAnonVolume bool
		scaleFlags      []string
	)
	cmd := &cobra.Command{
		Use:   "up [SERVICE...]",
		Short: "Create and start the project's containers",
		RunE: func(cmd *cobra.Command, services []string) error {
			scale, err := parseScale(scaleFlags)
			if err != nil {
				return err
			}
			_ = yes             // fruitbox never prompts; --yes is satisfied by default
			_ = renewAnonVolume // fruitbox recreate always allocates fresh anon volumes
			e, proj, release, err := opts.lockedEngine(cmd)
			if err != nil {
				return err
			}
			defer release()
			up := engine.UpOptions{
				Detach: detach,
				// --build forces a build and overrides --no-build.
				NoBuild:            noBuild && !build,
				RemoveOrphans:      removeOrphans,
				Scale:              scale,
				NoStart:            noStart,
				Wait:               wait,
				WaitTimeout:        waitTimeout,
				Pull:               pull,
				ForceRecreate:      forceRecreate,
				NoRecreate:         noRecreate,
				Services:           services,
				NoDeps:             noDeps,
				AbortOnExit:        abortOnExit,
				AbortOnFailure:     abortOnFailure,
				ExitCodeFrom:       exitCodeFrom,
				AlwaysRecreateDeps: recreateDeps,
				Attach:             attach,
				NoAttach:           noAttach,
				AttachDependencies: attachDeps,
				NoLogPrefix:        noLogPrefix,
				NoColor:            noColor,
				LogTimestamps:      timestamps,
				QuietBuild:         quietBuild,
				QuietPull:          quietPull,
			}
			if cmd.Flags().Changed("timeout") {
				up.Timeout = &timeout
			}
			// --watch: start the project (detached), then enter watch mode.
			if watch {
				up.Detach = true
				if err := e.Up(cmd.Context(), proj, up); err != nil {
					return err
				}
				return e.Watch(cmd.Context(), proj, 0, engine.WatchOptions{NoUp: true})
			}
			return e.Up(cmd.Context(), proj, up)
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
	f.BoolVar(&forceRecreate, "force-recreate", false, "Recreate containers even if their configuration hasn't changed")
	f.BoolVar(&noRecreate, "no-recreate", false, "If containers already exist, don't recreate them")
	f.BoolVar(&noDeps, "no-deps", false, "Don't start linked services")
	f.IntVarP(&timeout, "timeout", "t", 0, "Shutdown timeout in seconds when recreating")
	f.BoolVar(&abortOnExit, "abort-on-container-exit", false, "Stop all containers if any container stopped")
	f.BoolVar(&abortOnFailure, "abort-on-container-failure", false, "Stop all containers if any container exited with failure")
	f.StringVar(&exitCodeFrom, "exit-code-from", "", "Return the exit code of the selected service's container")
	f.StringArrayVar(&attach, "attach", nil, "Restrict attaching to the specified services")
	f.StringArrayVar(&noAttach, "no-attach", nil, "Do not attach (stream logs) to the specified services")
	f.BoolVar(&attachDeps, "attach-dependencies", false, "Automatically attach to log output of dependent services")
	f.BoolVar(&noLogPrefix, "no-log-prefix", false, "Don't print prefix in logs")
	f.BoolVar(&noColor, "no-color", false, "Produce monochrome output")
	f.BoolVar(&timestamps, "timestamps", false, "Show timestamps")
	f.BoolVar(&quietBuild, "quiet-build", false, "Suppress the build output")
	f.BoolVar(&quietPull, "quiet-pull", false, "Pull without printing progress information")
	f.BoolVar(&build, "build", false, "Build images before starting containers")
	f.BoolVar(&watch, "watch", false, "Watch source code and rebuild/refresh containers on change")
	f.BoolVarP(&yes, "yes", "y", false, "Assume \"yes\" as answer to all prompts")
	f.BoolVar(&recreateDeps, "always-recreate-deps", false, "Recreate dependent containers")
	f.BoolVar(&renewAnonVolume, "renew-anon-volumes", false, "Recreate anonymous volumes instead of retrieving data from the previous containers")
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
			e, proj, release, err := opts.lockedEngine(cmd)
			if err != nil {
				return err
			}
			defer release()
			down := engine.DownOptions{
				RemoveVolumes: removeVolumes,
				RemoveOrphans: removeOrphans,
				RemoveImages:  rmi,
			}
			if cmd.Flags().Changed("timeout") {
				down.Timeout = &timeout
			}
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
