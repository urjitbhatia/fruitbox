package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/urjitbhatia/fruitbox/internal/engine"
)

func newVolumesCommand(opts *globalOptions) *cobra.Command {
	var (
		quiet  bool
		format string
	)
	cmd := &cobra.Command{
		Use:   "volumes",
		Short: "List volumes used by the project",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			names := opts.engine(cmd.OutOrStdout()).VolumeNames(proj)
			if format == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(names)
			}
			if !quiet {
				fmt.Fprintln(cmd.OutOrStdout(), "VOLUME NAME")
			}
			for _, n := range names {
				fmt.Fprintln(cmd.OutOrStdout(), n)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Only display volume names")
	cmd.Flags().StringVar(&format, "format", "table", "Format output: table or json")
	return cmd
}

func newStatsCommand(opts *globalOptions) *cobra.Command {
	var (
		noStream bool
		format   string
	)
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Display a live stream of container resource usage statistics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			return opts.engine(cmd.OutOrStdout()).Stats(cmd.Context(), proj, engine.StatsOptions{
				NoStream: noStream,
				Format:   format,
			})
		},
	}
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Disable streaming stats and only pull the first result")
	cmd.Flags().StringVar(&format, "format", "", "Format the output")
	return cmd
}

func newExportCommand(opts *globalOptions) *cobra.Command {
	var (
		output string
		index  int
	)
	cmd := &cobra.Command{
		Use:   "export SERVICE",
		Short: "Export a service container's filesystem as a tar archive",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			return opts.engine(cmd.OutOrStdout()).Export(cmd.Context(), proj, args[0], output, index)
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "Write to a file, instead of STDOUT")
	cmd.Flags().IntVar(&index, "index", 1, "Replica index of the container")
	return cmd
}
