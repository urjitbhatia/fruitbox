package cli

import (
	"github.com/spf13/cobra"
	"github.com/urjitbhatia/fruitbox/internal/engine"
)

func newUpCommand(opts *globalOptions) *cobra.Command {
	var (
		detach  bool
		noBuild bool
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
			e := opts.engine(cmd.OutOrStdout())
			return e.Up(cmd.Context(), proj, engine.UpOptions{Detach: detach, NoBuild: noBuild})
		},
	}
	cmd.Flags().BoolVarP(&detach, "detach", "d", false, "Run containers in the background")
	cmd.Flags().BoolVar(&noBuild, "no-build", false, "Don't build images, even if they're missing")
	return cmd
}

func newDownCommand(opts *globalOptions) *cobra.Command {
	var removeVolumes bool
	cmd := &cobra.Command{
		Use:   "down",
		Short: "Stop and remove the project's containers, networks and (optionally) volumes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			e := opts.engine(cmd.OutOrStdout())
			return e.Down(cmd.Context(), proj, engine.DownOptions{RemoveVolumes: removeVolumes})
		},
	}
	cmd.Flags().BoolVarP(&removeVolumes, "volumes", "v", false, "Also remove named volumes declared in the volumes section")
	return cmd
}
