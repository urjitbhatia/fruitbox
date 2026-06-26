package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newLsCommand(opts *globalOptions) *cobra.Command {
	var quiet bool
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List running compose projects",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// `ls` is a runtime-wide query and needs no compose file.
			e := opts.engine(cmd.OutOrStdout())
			projects, err := e.ListProjects(cmd.Context())
			if err != nil {
				return err
			}
			if quiet {
				for _, p := range projects {
					fmt.Fprintln(cmd.OutOrStdout(), p.Name)
				}
				return nil
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tCONTAINERS")
			for _, p := range projects {
				fmt.Fprintf(w, "%s\t%d\n", p.Name, p.ContainerCount)
			}
			return w.Flush()
		},
	}
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Only display project names")
	return cmd
}

func newWaitCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wait [SERVICE...]",
		Short: "Block until the project's containers stop, then print the exit code",
		RunE: func(cmd *cobra.Command, services []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			code, err := opts.engine(cmd.OutOrStdout()).Wait(cmd.Context(), proj, services)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), code)
			return nil
		},
	}
	return cmd
}
