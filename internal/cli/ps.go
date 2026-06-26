package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/urjitbhatia/fruitbox/internal/engine"
)

func newPsCommand(opts *globalOptions) *cobra.Command {
	var (
		quiet        bool
		servicesOnly bool
		format       string
		noTrunc      bool
	)
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
			_ = noTrunc // fruitbox never truncates names, so --no-trunc is a no-op

			switch {
			case servicesOnly:
				return printPsServices(cmd, statuses)
			case quiet:
				for _, s := range statuses {
					fmt.Fprintln(cmd.OutOrStdout(), s.Name)
				}
				return nil
			case format == "json":
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(statuses)
			default:
				w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
				fmt.Fprintln(w, "NAME\tSERVICE\tSTATUS")
				for _, s := range statuses {
					fmt.Fprintf(w, "%s\t%s\t%s\n", s.Name, s.Service, s.Status)
				}
				return w.Flush()
			}
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&quiet, "quiet", "q", false, "Only display container names")
	f.BoolVar(&servicesOnly, "services", false, "Display services")
	f.StringVar(&format, "format", "table", "Format output: table or json")
	f.BoolVar(&noTrunc, "no-trunc", false, "Don't truncate output")
	return cmd
}

// printPsServices prints the unique service names with containers, sorted.
func printPsServices(cmd *cobra.Command, statuses []engine.ContainerStatus) error {
	seen := map[string]bool{}
	var names []string
	for _, s := range statuses {
		if !seen[s.Service] {
			seen[s.Service] = true
			names = append(names, s.Service)
		}
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Fprintln(cmd.OutOrStdout(), n)
	}
	return nil
}
