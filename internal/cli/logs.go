package cli

import (
	"github.com/spf13/cobra"
)

func newLogsCommand(opts *globalOptions) *cobra.Command {
	var follow bool
	cmd := &cobra.Command{
		Use:   "logs [SERVICE...]",
		Short: "Display logs from the project's containers",
		RunE: func(cmd *cobra.Command, services []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			e := opts.engine(cmd.OutOrStdout())
			return e.Logs(cmd.Context(), proj, services, follow)
		},
	}
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	return cmd
}
