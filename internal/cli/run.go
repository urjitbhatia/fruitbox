package cli

import (
	"github.com/spf13/cobra"
	"github.com/urjitbhatia/fruitbox/internal/engine"
)

func newRunCommand(opts *globalOptions) *cobra.Command {
	var (
		detach        bool
		rm            bool
		noDeps        bool
		name          string
		interactive   bool
		tty           bool
		noTTY         bool
		env           []string
		entrypoint    string
		user          string
		workdir       string
		labels        []string
		volumes       []string
		publish       []string
		capAdd        []string
		capDrop       []string
		servicePorts  bool
		build         bool
		removeOrphans bool
		pull          string
		envFromFile   []string
		quiet         bool
		quietBuild    bool
		quietPull     bool
	)
	cmd := &cobra.Command{
		Use:   "run [flags] SERVICE [COMMAND] [ARGS...]",
		Short: "Run a one-off command for a service",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			e, proj, release, err := opts.lockedEngine(cmd)
			if err != nil {
				return err
			}
			defer release()
			service := args[0]
			var command []string
			if len(args) > 1 {
				command = args[1:]
			}
			return e.RunOneOff(cmd.Context(), proj, service, engine.RunOneOffOptions{
				Command:       command,
				Detach:        detach,
				Remove:        rm,
				NoDeps:        noDeps,
				Name:          name,
				Interactive:   interactive,
				TTY:           tty && !noTTY && ttyAvailable(),
				Env:           env,
				Entrypoint:    entrypoint,
				EntrypointSet: cmd.Flags().Changed("entrypoint"),
				User:          user,
				WorkDir:       workdir,
				Labels:        labels,
				Volumes:       volumes,
				Publish:       publish,
				CapAdd:        capAdd,
				CapDrop:       capDrop,
				ServicePorts:  servicePorts,
				Build:         build,
				RemoveOrphans: removeOrphans,
				Pull:          pull,
				EnvFromFile:   envFromFile,
				Quiet:         quiet,
				QuietBuild:    quietBuild,
				QuietPull:     quietPull,
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
	f.BoolVarP(&noTTY, "no-TTY", "T", false, "Disable pseudo-TTY allocation")
	f.StringArrayVarP(&env, "env", "e", nil, "Set environment variables")
	f.StringVar(&entrypoint, "entrypoint", "", "Override the entrypoint of the image")
	f.StringVarP(&user, "user", "u", "", "Run as the given user")
	f.StringVarP(&workdir, "workdir", "w", "", "Working directory inside the container")
	f.StringArrayVarP(&labels, "label", "l", nil, "Add or override labels")
	f.StringArrayVarP(&volumes, "volume", "v", nil, "Bind mount a volume")
	f.StringArrayVarP(&publish, "publish", "p", nil, "Publish a container's port(s) to the host")
	f.StringArrayVar(&capAdd, "cap-add", nil, "Add Linux capabilities")
	f.StringArrayVar(&capDrop, "cap-drop", nil, "Drop Linux capabilities")
	f.BoolVar(&servicePorts, "service-ports", false, "Run with the service's ports enabled and mapped to the host")
	f.BoolVar(&build, "build", false, "Build image before starting the container")
	f.BoolVar(&removeOrphans, "remove-orphans", false, "Remove containers for services not defined in the Compose file")
	f.StringVar(&pull, "pull", "policy", `Pull image before running ("always"|"missing"|"never")`)
	f.StringArrayVar(&envFromFile, "env-from-file", nil, "Set environment variables from file")
	f.BoolVarP(&quiet, "quiet", "q", false, "Don't print anything to STDOUT")
	f.BoolVar(&quietBuild, "quiet-build", false, "Suppress the build output")
	f.BoolVar(&quietPull, "quiet-pull", false, "Pull without printing progress information")
	return cmd
}
