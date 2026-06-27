package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
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
		all          bool
		status       string
		filters      []string
		orphans      bool
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

			pf, err := parsePsFilters(all, status, filters)
			if err != nil {
				return err
			}
			statuses = filterPs(statuses, pf)
			if orphans {
				statuses = append(statuses, e.Orphans(cmd.Context(), proj)...)
			}

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
				fmt.Fprintln(w, "NAME\tIMAGE\tSERVICE\tSTATUS\tPORTS")
				for _, s := range statuses {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", s.Name, s.Image, s.Service, s.Status, s.Ports)
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
	f.BoolVarP(&all, "all", "a", false, "Show all stopped containers (including those created by run)")
	f.StringVar(&status, "status", "", "Filter services by status")
	f.StringArrayVar(&filters, "filter", nil, "Filter services by a property (status=, name=)")
	f.BoolVar(&orphans, "orphans", false, "Include orphaned services (not declared by project)")
	return cmd
}

// psFilter holds the resolved ps filtering options.
type psFilter struct {
	all    bool
	status string
	name   string
}

// parsePsFilters merges --all/--status with --filter key=value entries.
func parsePsFilters(all bool, status string, filters []string) (psFilter, error) {
	pf := psFilter{all: all, status: status}
	for _, f := range filters {
		k, v, ok := strings.Cut(f, "=")
		if !ok {
			return pf, fmt.Errorf("invalid --filter %q, want key=value", f)
		}
		switch k {
		case "status":
			pf.status = v
		case "name":
			pf.name = v
		default:
			return pf, fmt.Errorf("unsupported --filter key %q (want status or name)", k)
		}
	}
	return pf, nil
}

// filterPs applies the ps filter. Containers that do not exist ("not created")
// are never listed (matching docker compose). Without --all or an explicit
// status, only running containers are shown.
func filterPs(in []engine.ContainerStatus, pf psFilter) []engine.ContainerStatus {
	var out []engine.ContainerStatus
	for _, s := range in {
		if s.Status == "not created" {
			continue
		}
		if pf.status != "" && s.Status != pf.status {
			continue
		}
		if pf.status == "" && !pf.all && s.Status != "running" {
			continue
		}
		if pf.name != "" && !strings.Contains(s.Name, pf.name) {
			continue
		}
		out = append(out, s)
	}
	return out
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
