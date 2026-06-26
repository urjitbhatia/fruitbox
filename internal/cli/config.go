package cli

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

func newConfigCommand(opts *globalOptions) *cobra.Command {
	var (
		format       string
		servicesOnly bool
		volumesOnly  bool
		quiet        bool
	)
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Parse, resolve and render the Compose configuration",
		Long:  "Loads the Compose file(s), applies interpolation/merge/normalization, and prints the canonical configuration.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}

			switch {
			case servicesOnly:
				names := proj.ServiceNames()
				sort.Strings(names)
				for _, n := range names {
					fmt.Fprintln(cmd.OutOrStdout(), n)
				}
				return nil
			case volumesOnly:
				names := make([]string, 0, len(proj.Volumes))
				for n := range proj.Volumes {
					names = append(names, n)
				}
				sort.Strings(names)
				for _, n := range names {
					fmt.Fprintln(cmd.OutOrStdout(), n)
				}
				return nil
			}

			if quiet {
				return nil
			}

			switch format {
			case "json":
				data, err := proj.MarshalJSON()
				if err != nil {
					return err
				}
				var pretty interface{}
				if err := json.Unmarshal(data, &pretty); err != nil {
					return err
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(pretty)
			case "yaml", "":
				data, err := proj.MarshalYAML()
				if err != nil {
					return err
				}
				fmt.Fprint(cmd.OutOrStdout(), string(data))
				return nil
			default:
				return fmt.Errorf("unsupported format %q (want yaml or json)", format)
			}
		},
	}
	cmd.Flags().StringVar(&format, "format", "yaml", "Output format: yaml or json")
	cmd.Flags().BoolVar(&servicesOnly, "services", false, "Print the service names, one per line")
	cmd.Flags().BoolVar(&volumesOnly, "volumes", false, "Print the volume names, one per line")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Only validate the configuration, don't print anything")
	return cmd
}
