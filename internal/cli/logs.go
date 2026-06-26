package cli

import (
	"github.com/spf13/cobra"
	"github.com/urjitbhatia/fruitbox/internal/engine"
)

func newLogsCommand(opts *globalOptions) *cobra.Command {
	var (
		follow     bool
		tail       string
		index      int
		noPrefix   bool
		noColor    bool
		timestamps bool
	)
	cmd := &cobra.Command{
		Use:   "logs [SERVICE...]",
		Short: "Display logs from the project's containers",
		RunE: func(cmd *cobra.Command, services []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			e := opts.engine(cmd.OutOrStdout())
			return e.Logs(cmd.Context(), proj, services, engine.LogOptions{
				Follow:     follow,
				Tail:       tail,
				Index:      index,
				NoPrefix:   noPrefix,
				NoColor:    noColor,
				Timestamps: timestamps,
			})
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&follow, "follow", "f", false, "Follow log output")
	f.StringVar(&tail, "tail", "all", "Number of lines to show from the end of the logs")
	f.IntVar(&index, "index", 0, "Show logs for a single replica index")
	f.BoolVar(&noPrefix, "no-log-prefix", false, "Don't print prefix in logs")
	f.BoolVar(&noColor, "no-color", false, "Produce monochrome output")
	f.BoolVar(&timestamps, "timestamps", false, "Show timestamps")
	return cmd
}
