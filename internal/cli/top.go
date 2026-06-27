package cli

import (
	"github.com/spf13/cobra"
)

func newTopCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "top [SERVICE...]",
		Short: "Display the running processes of the project's containers",
		RunE: func(cmd *cobra.Command, services []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			return opts.engine(cmd.OutOrStdout()).Top(cmd.Context(), proj, services, nil)
		},
	}
	return cmd
}

func newPauseCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "pause [SERVICE...]",
		Short: "Pause the project's containers (SIGSTOP)",
		RunE: func(cmd *cobra.Command, services []string) error {
			e, proj, release, err := opts.lockedEngine(cmd)
			if err != nil {
				return err
			}
			defer release()
			return e.Pause(cmd.Context(), proj, services)
		},
	}
}

func newUnpauseCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "unpause [SERVICE...]",
		Short: "Resume paused containers (SIGCONT)",
		RunE: func(cmd *cobra.Command, services []string) error {
			e, proj, release, err := opts.lockedEngine(cmd)
			if err != nil {
				return err
			}
			defer release()
			return e.Unpause(cmd.Context(), proj, services)
		},
	}
}
