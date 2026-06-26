package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newPsCommand(opts *globalOptions) *cobra.Command {
	var quiet bool
	cmd := &cobra.Command{
		Use:   "ps",
		Short: "List the project's containers and their status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			e := opts.engine(cmd.OutOrStdout())
			statuses, err := e.Ps(cmd.Context(), proj)
			if err != nil {
				return err
			}
			if quiet {
				for _, s := range statuses {
					fmt.Fprintln(cmd.OutOrStdout(), s.Name)
				}
				return nil
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tSERVICE\tSTATUS")
			for _, s := range statuses {
				fmt.Fprintf(w, "%s\t%s\t%s\n", s.Name, s.Service, s.Status)
			}
			return w.Flush()
		},
	}
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Only display container names")
	return cmd
}
