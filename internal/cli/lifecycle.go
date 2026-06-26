package cli

import (
	"github.com/spf13/cobra"
	"github.com/urjitbhatia/fruitbox/internal/engine"
)

func newBuildCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "build [SERVICE...]",
		Short: "Build or rebuild service images",
		RunE: func(cmd *cobra.Command, services []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			return opts.engine(cmd.OutOrStdout()).Build(cmd.Context(), proj, services)
		},
	}
}

func newStartCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "start [SERVICE...]",
		Short: "Start existing containers for services",
		RunE: func(cmd *cobra.Command, services []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			return opts.engine(cmd.OutOrStdout()).Start(cmd.Context(), proj, services)
		},
	}
}

func newStopCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "stop [SERVICE...]",
		Short: "Stop running containers without removing them",
		RunE: func(cmd *cobra.Command, services []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			return opts.engine(cmd.OutOrStdout()).Stop(cmd.Context(), proj, services)
		},
	}
}

func newRestartCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "restart [SERVICE...]",
		Short: "Restart service containers",
		RunE: func(cmd *cobra.Command, services []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			return opts.engine(cmd.OutOrStdout()).Restart(cmd.Context(), proj, services)
		},
	}
}

func newKillCommand(opts *globalOptions) *cobra.Command {
	var signal string
	cmd := &cobra.Command{
		Use:   "kill [SERVICE...]",
		Short: "Force-stop service containers by sending a signal",
		RunE: func(cmd *cobra.Command, services []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			return opts.engine(cmd.OutOrStdout()).Kill(cmd.Context(), proj, services, signal)
		},
	}
	cmd.Flags().StringVarP(&signal, "signal", "s", "", "Signal to send (default SIGKILL)")
	return cmd
}

func newPullCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "pull [SERVICE...]",
		Short: "Pull images for services",
		RunE: func(cmd *cobra.Command, services []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			return opts.engine(cmd.OutOrStdout()).Pull(cmd.Context(), proj, services)
		},
	}
}

func newExecCommand(opts *globalOptions) *cobra.Command {
	var (
		interactive bool
		tty         bool
		user        string
		workdir     string
		env         []string
	)
	cmd := &cobra.Command{
		Use:                "exec SERVICE COMMAND [ARGS...]",
		Short:              "Run a command in a running service container",
		Args:               cobra.MinimumNArgs(2),
		DisableFlagParsing: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			return opts.engine(cmd.OutOrStdout()).Exec(cmd.Context(), proj, args[0], args[1:], engine.ExecOptions{
				Interactive: interactive,
				TTY:         tty,
				User:        user,
				WorkingDir:  workdir,
				Env:         env,
			})
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&interactive, "interactive", "i", true, "Keep STDIN open")
	f.BoolVarP(&tty, "tty", "t", true, "Allocate a TTY")
	f.StringVarP(&user, "user", "u", "", "Run as the given user")
	f.StringVarP(&workdir, "workdir", "w", "", "Working directory inside the container")
	f.StringArrayVarP(&env, "env", "e", nil, "Set environment variables")
	return cmd
}
