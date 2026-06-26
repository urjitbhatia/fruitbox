package cli

import (
	"github.com/spf13/cobra"
	"github.com/urjitbhatia/fruitbox/internal/engine"
)

func newRunCommand(opts *globalOptions) *cobra.Command {
	var (
		detach      bool
		rm          bool
		noDeps      bool
		name        string
		interactive bool
		tty         bool
		env         []string
	)
	cmd := &cobra.Command{
		Use:   "run [flags] SERVICE [COMMAND] [ARGS...]",
		Short: "Run a one-off command for a service",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := opts.load(cmd.Context())
			if err != nil {
				return err
			}
			service := args[0]
			var command []string
			if len(args) > 1 {
				command = args[1:]
			}
			e := opts.engine(cmd.OutOrStdout())
			return e.RunOneOff(cmd.Context(), proj, service, engine.RunOneOffOptions{
				Command:     command,
				Detach:      detach,
				Remove:      rm,
				NoDeps:      noDeps,
				Name:        name,
				Interactive: interactive,
				TTY:         tty,
				Env:         env,
			})
		},
	}
	f := cmd.Flags()
	f.SetInterspersed(false) // stop flag parsing at the first positional (the command)
	f.BoolVarP(&detach, "detach", "d", false, "Run the container in the background")
	f.BoolVar(&rm, "rm", true, "Remove the container after it exits")
	f.BoolVar(&noDeps, "no-deps", false, "Don't start linked services")
	f.StringVar(&name, "name", "", "Assign a name to the container")
	f.BoolVarP(&interactive, "interactive", "i", true, "Keep STDIN open")
	f.BoolVarP(&tty, "tty", "t", true, "Allocate a TTY")
	f.StringArrayVarP(&env, "env", "e", nil, "Set environment variables")
	return cmd
}
