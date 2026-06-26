package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

func newVersionCommand() *cobra.Command {
	var (
		format string
		short  bool
	)
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show the fruitbox version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			switch {
			case short:
				fmt.Fprintln(out, translate.Version)
			case format == "json":
				b, _ := json.Marshal(map[string]string{"version": "v" + translate.Version})
				fmt.Fprintln(out, string(b))
			default:
				fmt.Fprintf(out, "fruitbox version v%s\n", translate.Version)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "", "Format the output. Values: [pretty | json]")
	cmd.Flags().BoolVar(&short, "short", false, "Show only the version number")
	return cmd
}
