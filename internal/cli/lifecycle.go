package cli

import (
	"github.com/spf13/cobra"
	"github.com/urjitbhatia/fruitbox/internal/engine"
)

func newBuildCommand(opts *globalOptions) *cobra.Command {
	var (
		buildArgs []string
		noCache   bool
		pull      bool
		quiet     bool
		memory    string
	)
	cmd := &cobra.Command{
		Use:   "build [SERVICE...]",
		Short: "Build or rebuild service images",
		RunE: func(cmd *cobra.Command, services []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			return opts.engine(cmd.OutOrStdout()).Build(cmd.Context(), proj, services, engine.BuildOptions{
				BuildArgs: buildArgs,
				NoCache:   noCache,
				Pull:      pull,
				Quiet:     quiet,
				Memory:    memory,
			})
		},
	}
	f := cmd.Flags()
	f.StringArrayVar(&buildArgs, "build-arg", nil, "Set build-time variables for services")
	f.BoolVar(&noCache, "no-cache", false, "Do not use cache when building the image")
	f.BoolVar(&pull, "pull", false, "Always attempt to pull a newer version of the image")
	f.BoolVarP(&quiet, "quiet", "q", false, "Don't print anything to STDOUT")
	f.StringVarP(&memory, "memory", "m", "", "Set memory limit for the build container")
	return cmd
}

func newStartCommand(opts *globalOptions) *cobra.Command {
	var (
		wait        bool
		waitTimeout int
	)
	cmd := &cobra.Command{
		Use:   "start [SERVICE...]",
		Short: "Start existing containers for services",
		RunE: func(cmd *cobra.Command, services []string) error {
			e, proj, release, err := opts.lockedEngine(cmd)
			if err != nil {
				return err
			}
			defer release()
			return e.Start(cmd.Context(), proj, services, engine.StartOptions{
				Wait:        wait,
				WaitTimeout: waitTimeout,
			})
		},
	}
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for services to be running|healthy")
	cmd.Flags().IntVar(&waitTimeout, "wait-timeout", 0, "Max seconds to wait for the project to be running|healthy")
	return cmd
}

func newStopCommand(opts *globalOptions) *cobra.Command {
	var timeout int
	cmd := &cobra.Command{
		Use:   "stop [SERVICE...]",
		Short: "Stop running containers without removing them",
		RunE: func(cmd *cobra.Command, services []string) error {
			e, proj, release, err := opts.lockedEngine(cmd)
			if err != nil {
				return err
			}
			defer release()
			return e.Stop(cmd.Context(), proj, services, timeoutPtr(cmd))
		},
	}
	cmd.Flags().IntVarP(&timeout, "timeout", "t", 0, "Shutdown timeout in seconds")
	return cmd
}

func newRestartCommand(opts *globalOptions) *cobra.Command {
	var (
		timeout int
		noDeps  bool
	)
	cmd := &cobra.Command{
		Use:   "restart [SERVICE...]",
		Short: "Restart service containers",
		RunE: func(cmd *cobra.Command, services []string) error {
			e, proj, release, err := opts.lockedEngine(cmd)
			if err != nil {
				return err
			}
			defer release()
			// restart only touches the named services already, so --no-deps is
			// the effective default; the flag is accepted for compatibility.
			_ = noDeps
			return e.Restart(cmd.Context(), proj, services, timeoutPtr(cmd))
		},
	}
	cmd.Flags().IntVarP(&timeout, "timeout", "t", 0, "Shutdown timeout in seconds")
	cmd.Flags().BoolVar(&noDeps, "no-deps", false, "Don't restart dependent services")
	return cmd
}

func newKillCommand(opts *globalOptions) *cobra.Command {
	var (
		signal        string
		removeOrphans bool
	)
	cmd := &cobra.Command{
		Use:   "kill [SERVICE...]",
		Short: "Force-stop service containers by sending a signal",
		RunE: func(cmd *cobra.Command, services []string) error {
			e, proj, release, err := opts.lockedEngine(cmd)
			if err != nil {
				return err
			}
			defer release()
			return e.Kill(cmd.Context(), proj, services, signal, removeOrphans)
		},
	}
	cmd.Flags().StringVarP(&signal, "signal", "s", "", "Signal to send (default SIGKILL)")
	cmd.Flags().BoolVar(&removeOrphans, "remove-orphans", false, "Remove containers for services not in the Compose file")
	return cmd
}

// timeoutPtr returns a pointer to the --timeout value only if the flag was set,
// so an unset flag leaves per-service stop_grace_period in effect.
func timeoutPtr(cmd *cobra.Command) *int {
	if !cmd.Flags().Changed("timeout") {
		return nil
	}
	v, _ := cmd.Flags().GetInt("timeout")
	return &v
}

func newPullCommand(opts *globalOptions) *cobra.Command {
	var (
		quiet           bool
		includeDeps     bool
		ignoreFailures  bool
		ignoreBuildable bool
	)
	cmd := &cobra.Command{
		Use:   "pull [SERVICE...]",
		Short: "Pull images for services",
		RunE: func(cmd *cobra.Command, services []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			return opts.engine(cmd.OutOrStdout()).Pull(cmd.Context(), proj, services, engine.PullOptions{
				Quiet:           quiet,
				IncludeDeps:     includeDeps,
				IgnoreFailures:  ignoreFailures,
				IgnoreBuildable: ignoreBuildable,
			})
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&quiet, "quiet", "q", false, "Pull without printing progress information")
	f.BoolVar(&includeDeps, "include-deps", false, "Also pull services declared as dependencies")
	f.BoolVar(&ignoreFailures, "ignore-pull-failures", false, "Pull what it can and ignore images with pull failures")
	f.BoolVar(&ignoreBuildable, "ignore-buildable", false, "Ignore images that can be built")
	return cmd
}

func newExecCommand(opts *globalOptions) *cobra.Command {
	var (
		interactive bool
		tty         bool
		noTTY       bool
		detach      bool
		index       int
		user        string
		workdir     string
		env         []string
	)
	cmd := &cobra.Command{
		Use:   "exec SERVICE COMMAND [ARGS...]",
		Short: "Run a command in a running service container",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			return opts.engine(cmd.OutOrStdout()).Exec(cmd.Context(), proj, args[0], args[1:], engine.ExecOptions{
				Interactive: interactive,
				TTY:         tty && !noTTY && ttyAvailable(),
				Detach:      detach,
				Index:       index,
				User:        user,
				WorkingDir:  workdir,
				Env:         env,
			})
		},
	}
	f := cmd.Flags()
	// Stop parsing flags at the first positional (SERVICE) so flags meant for
	// the in-container command (e.g. `exec web nginx -v`) pass through.
	f.SetInterspersed(false)
	f.BoolVarP(&interactive, "interactive", "i", true, "Keep STDIN open")
	f.BoolVarP(&tty, "tty", "t", true, "Allocate a TTY")
	f.BoolVarP(&noTTY, "no-tty", "T", false, "Disable pseudo-TTY allocation")
	f.BoolVarP(&detach, "detach", "d", false, "Run the command in the background")
	f.IntVar(&index, "index", 0, "Index of the container replica")
	f.StringVarP(&user, "user", "u", "", "Run as the given user")
	f.StringVarP(&workdir, "workdir", "w", "", "Working directory inside the container")
	f.StringArrayVarP(&env, "env", "e", nil, "Set environment variables")
	return cmd
}
