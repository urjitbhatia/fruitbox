package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/urjitbhatia/fruitbox/internal/engine"
)

func newLsCommand(opts *globalOptions) *cobra.Command {
	var (
		quiet   bool
		format  string
		all     bool
		filters []string
	)
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
			projects, err = filterProjects(projects, all, filters)
			if err != nil {
				return err
			}
			switch {
			case quiet:
				for _, p := range projects {
					fmt.Fprintln(cmd.OutOrStdout(), p.Name)
				}
				return nil
			case format == "json":
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(projects)
			default:
				w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
				fmt.Fprintln(w, "NAME\tCONTAINERS")
				for _, p := range projects {
					fmt.Fprintf(w, "%s\t%d\n", p.Name, p.ContainerCount)
				}
				return w.Flush()
			}
		},
	}
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Only display project names")
	cmd.Flags().StringVar(&format, "format", "table", "Format output: table or json")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Show all stopped Compose projects")
	cmd.Flags().StringArrayVar(&filters, "filter", nil, "Filter output based on conditions provided (name=)")
	return cmd
}

// filterProjects applies --all (default shows only projects with running
// containers) and --filter name= to the project list.
func filterProjects(in []engine.ProjectSummary, all bool, filters []string) ([]engine.ProjectSummary, error) {
	name := ""
	for _, f := range filters {
		k, v, ok := strings.Cut(f, "=")
		if !ok {
			return nil, fmt.Errorf("invalid --filter %q, want key=value", f)
		}
		if k != "name" {
			return nil, fmt.Errorf("unsupported --filter key %q (want name)", k)
		}
		name = v
	}
	var out []engine.ProjectSummary
	for _, p := range in {
		if !all && p.RunningCount == 0 {
			continue
		}
		if name != "" && !strings.Contains(p.Name, name) {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func newWaitCommand(opts *globalOptions) *cobra.Command {
	var downProject bool
	cmd := &cobra.Command{
		Use:   "wait [SERVICE...]",
		Short: "Block until the project's containers stop, then print the exit code",
		RunE: func(cmd *cobra.Command, services []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			e := opts.engine(cmd.OutOrStdout())
			code, err := e.Wait(cmd.Context(), proj, services)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), code)
			if downProject {
				return e.Down(cmd.Context(), proj, engine.DownOptions{})
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&downProject, "down-project", false, "Drop the project after the wait completes")
	return cmd
}
