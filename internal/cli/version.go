package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the fruitbox version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "fruitbox version %s\n", translate.Version)
			return nil
		},
	}
}
