package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/urjitbhatia/fruitbox/internal/engine"
)

func newImagesCommand(opts *globalOptions) *cobra.Command {
	var quiet bool
	cmd := &cobra.Command{
		Use:   "images",
		Short: "List images used by the project's services",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			images := opts.engine(cmd.OutOrStdout()).Images(proj)
			if quiet {
				seen := map[string]bool{}
				for _, im := range images {
					if im.Image != "" && !seen[im.Image] {
						seen[im.Image] = true
						fmt.Fprintln(cmd.OutOrStdout(), im.Image)
					}
				}
				return nil
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(w, "SERVICE\tIMAGE")
			for _, im := range images {
				fmt.Fprintf(w, "%s\t%s\n", im.Service, im.Image)
			}
			return w.Flush()
		},
	}
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Only display image names")
	return cmd
}

func newPortCommand(opts *globalOptions) *cobra.Command {
	var protocol string
	cmd := &cobra.Command{
		Use:   "port SERVICE PRIVATE_PORT",
		Short: "Print the public port for a port binding",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			port, proto, err := parsePortArg(args[1], protocol)
			if err != nil {
				return err
			}
			mapped, err := opts.engine(cmd.OutOrStdout()).Port(proj, args[0], port, proto)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), mapped)
			return nil
		},
	}
	cmd.Flags().StringVar(&protocol, "protocol", "tcp", "tcp or udp")
	return cmd
}

func newCpCommand(opts *globalOptions) *cobra.Command {
	var index int
	cmd := &cobra.Command{
		Use:   "cp SRC DEST",
		Short: "Copy files/folders between a container and the local filesystem",
		Long:  "Either SRC or DEST may be SERVICE:PATH.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			return opts.engine(cmd.OutOrStdout()).Copy(cmd.Context(), proj, args[0], args[1], index)
		},
	}
	cmd.Flags().IntVar(&index, "index", 1, "Replica index when a service has multiple containers")
	return cmd
}

// parsePortArg parses the port argument, preferring an explicit /proto suffix
// over the --protocol flag.
func parsePortArg(arg, protoFlag string) (int, string, error) {
	port, proto, err := engine.ParsePort(arg)
	if err != nil {
		return 0, "", err
	}
	if proto == "tcp" && protoFlag != "" {
		proto = protoFlag
	}
	return port, proto, nil
}
