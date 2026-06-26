package cli

import (
	"github.com/spf13/cobra"
	"github.com/urjitbhatia/fruitbox/internal/engine"
)

func newCreateCommand(opts *globalOptions) *cobra.Command {
	var scaleFlags []string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create the project's containers without starting them",
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
			return opts.engine(cmd.OutOrStdout()).Create(cmd.Context(), proj, scale)
		},
	}
	cmd.Flags().StringArrayVar(&scaleFlags, "scale", nil, "Scale SERVICE to NUM instances (SERVICE=NUM)")
	return cmd
}

func newRmCommand(opts *globalOptions) *cobra.Command {
	var (
		force bool
		stop  bool
	)
	cmd := &cobra.Command{
		Use:   "rm [SERVICE...]",
		Short: "Remove stopped service containers",
		RunE: func(cmd *cobra.Command, services []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			return opts.engine(cmd.OutOrStdout()).Rm(cmd.Context(), proj, services, engine.RmOptions{Force: force, Stop: stop})
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Don't ask to confirm removal")
	cmd.Flags().BoolVarP(&stop, "stop", "s", false, "Stop the containers before removing")
	return cmd
}

func newPushCommand(opts *globalOptions) *cobra.Command {
	var (
		quiet          bool
		includeDeps    bool
		ignoreFailures bool
	)
	cmd := &cobra.Command{
		Use:   "push [SERVICE...]",
		Short: "Push service images to their registries",
		RunE: func(cmd *cobra.Command, services []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			return opts.engine(cmd.OutOrStdout()).Push(cmd.Context(), proj, services, engine.PushOptions{
				Quiet:          quiet,
				IncludeDeps:    includeDeps,
				IgnoreFailures: ignoreFailures,
			})
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&quiet, "quiet", "q", false, "Push without printing progress information")
	f.BoolVar(&includeDeps, "include-deps", false, "Also push images of services declared as dependencies")
	f.BoolVar(&ignoreFailures, "ignore-push-failures", false, "Push what it can and ignore images with push failures")
	return cmd
}

func newScaleCommand(opts *globalOptions) *cobra.Command {
	var noDeps bool
	cmd := &cobra.Command{
		Use:   "scale SERVICE=NUM [SERVICE=NUM...]",
		Short: "Scale services to the given number of replicas",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			scale, err := parseScale(args)
			if err != nil {
				return err
			}
			// scale only touches the named services, so --no-deps is the
			// effective default; accepted for compatibility.
			_ = noDeps
			return opts.engine(cmd.OutOrStdout()).Scale(cmd.Context(), proj, scale)
		},
	}
	cmd.Flags().BoolVar(&noDeps, "no-deps", false, "Don't start dependent services")
	return cmd
}

func newWatchCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch source files and sync/restart/rebuild services on change",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			return opts.engine(cmd.OutOrStdout()).Watch(cmd.Context(), proj, 0)
		},
	}
	return cmd
}

func newEventsCommand(opts *globalOptions) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Stream container lifecycle events for the project",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			// 0 = stream until interrupted.
			return opts.engine(cmd.OutOrStdout()).Events(cmd.Context(), proj, 0, jsonOut)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output events as a stream of JSON objects")
	return cmd
}

func newAttachCommand(opts *globalOptions) *cobra.Command {
	var index int
	cmd := &cobra.Command{
		Use:   "attach SERVICE",
		Short: "Attach to a running service container's I/O streams",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			return opts.engine(cmd.OutOrStdout()).Attach(cmd.Context(), proj, args[0], index)
		},
	}
	cmd.Flags().IntVar(&index, "index", 1, "Replica index to attach to")
	return cmd
}
