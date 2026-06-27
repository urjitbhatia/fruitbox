package cli

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/spf13/cobra"
	"github.com/urjitbhatia/fruitbox/internal/compose"
)

type configFlags struct {
	format        string
	output        string
	quiet         bool
	services      bool
	volumes       bool
	networks      bool
	models        bool
	profiles      bool
	images        bool
	environment   bool
	hash          string
	noInterpolate bool
	noNormalize   bool
	noConsistency bool
	noPathResolve bool
	noEnvResolve  bool
}

func newConfigCommand(opts *globalOptions) *cobra.Command {
	f := &configFlags{}
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Parse, resolve and render the Compose configuration",
		Long:  "Loads the Compose file(s), applies interpolation/merge/normalization, and prints the canonical configuration.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			proj, err := opts.loadWith(cmd.Context(), func(lo *compose.LoadOptions) {
				lo.NoInterpolate = f.noInterpolate
				lo.NoNormalize = f.noNormalize
				lo.NoConsistency = f.noConsistency
				lo.NoResolvePaths = f.noPathResolve
				lo.NoEnvResolution = f.noEnvResolve
			})
			if err != nil {
				return err
			}
			return runConfig(cmd, proj, f)
		},
	}
	fl := cmd.Flags()
	fl.StringVar(&f.format, "format", "yaml", "Output format: yaml or json")
	fl.StringVarP(&f.output, "output", "o", "", "Save to file (default stdout)")
	fl.BoolVarP(&f.quiet, "quiet", "q", false, "Only validate the configuration, don't print anything")
	fl.BoolVar(&f.services, "services", false, "Print the service names, one per line")
	fl.BoolVar(&f.volumes, "volumes", false, "Print the volume names, one per line")
	fl.BoolVar(&f.networks, "networks", false, "Print the network names, one per line")
	fl.BoolVar(&f.models, "models", false, "Print the model names, one per line")
	fl.BoolVar(&f.profiles, "profiles", false, "Print the profile names, one per line")
	fl.BoolVar(&f.images, "images", false, "Print the image names, one per line")
	fl.BoolVar(&f.environment, "environment", false, "Print environment used for interpolation")
	fl.StringVar(&f.hash, "hash", "", "Print the service config hash, one per line (\"*\" for all)")
	fl.BoolVar(&f.noInterpolate, "no-interpolate", false, "Don't interpolate environment variables")
	fl.BoolVar(&f.noNormalize, "no-normalize", false, "Don't normalize the compose model")
	fl.BoolVar(&f.noConsistency, "no-consistency", false, "Don't check model consistency")
	fl.BoolVar(&f.noPathResolve, "no-path-resolution", false, "Don't resolve file paths")
	fl.BoolVar(&f.noEnvResolve, "no-env-resolution", false, "Don't resolve service env files")
	return cmd
}

func runConfig(cmd *cobra.Command, proj *types.Project, f *configFlags) error {
	// List-style outputs short-circuit (matching docker compose).
	switch {
	case f.services:
		return printLines(cmd, proj.ServiceNames())
	case f.volumes:
		return printLines(cmd, keysOf(proj.Volumes))
	case f.networks:
		return printLines(cmd, keysOf(proj.Networks))
	case f.models:
		return printLines(cmd, keysOf(proj.Models))
	case f.profiles:
		return printLines(cmd, profileNames(proj))
	case f.images:
		return printLines(cmd, imageNames(proj))
	case f.environment:
		return printEnvironment(cmd, proj)
	case f.hash != "":
		return printHashes(cmd, proj, f.hash)
	}

	if f.quiet {
		return nil
	}

	var data []byte
	var err error
	switch f.format {
	case "json":
		if data, err = marshalJSON(proj); err != nil {
			return err
		}
	case "yaml", "":
		if data, err = proj.MarshalYAML(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported format %q (want yaml or json)", f.format)
	}

	if f.output != "" {
		return os.WriteFile(f.output, data, 0o644)
	}
	fmt.Fprint(cmd.OutOrStdout(), string(data))
	if len(data) > 0 && data[len(data)-1] != '\n' {
		fmt.Fprintln(cmd.OutOrStdout())
	}
	return nil
}

func marshalJSON(proj *types.Project) ([]byte, error) {
	raw, err := proj.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var pretty interface{}
	if err := json.Unmarshal(raw, &pretty); err != nil {
		return nil, err
	}
	return json.MarshalIndent(pretty, "", "  ")
}

func printLines(cmd *cobra.Command, items []string) error {
	sort.Strings(items)
	for _, s := range items {
		fmt.Fprintln(cmd.OutOrStdout(), s)
	}
	return nil
}

// printHashes prints "service hash" lines. selector is a comma-separated list
// of service names, or "*" for all.
func printHashes(cmd *cobra.Command, proj *types.Project, selector string) error {
	names := proj.ServiceNames()
	if selector != "*" {
		names = splitComma(selector)
	}
	sort.Strings(names)
	for _, name := range names {
		svc, err := proj.GetService(name)
		if err != nil {
			return err
		}
		h, err := serviceHash(svc)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", name, h)
	}
	return nil
}

// serviceHash returns a stable content hash of a service's resolved config.
func serviceHash(svc types.ServiceConfig) (string, error) {
	b, err := json.Marshal(svc)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return fmt.Sprintf("%x", sum), nil
}

func keysOf[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func profileNames(proj *types.Project) []string {
	seen := map[string]bool{}
	for _, svc := range proj.AllServices() {
		for _, p := range svc.Profiles {
			seen[p] = true
		}
	}
	return keysOf(seen)
}

// printEnvironment prints the KEY=VALUE environment used for interpolation,
// sorted, like docker compose config --environment. Falls back to the process
// environment if the project model didn't capture it.
func printEnvironment(cmd *cobra.Command, proj *types.Project) error {
	var lines []string
	if len(proj.Environment) > 0 {
		for k, v := range proj.Environment {
			lines = append(lines, k+"="+v)
		}
	} else {
		lines = append(lines, os.Environ()...)
	}
	sort.Strings(lines)
	for _, l := range lines {
		fmt.Fprintln(cmd.OutOrStdout(), l)
	}
	return nil
}

func imageNames(proj *types.Project) []string {
	seen := map[string]bool{}
	for _, name := range proj.ServiceNames() {
		svc, _ := proj.GetService(name)
		if svc.Image != "" {
			seen[svc.Image] = true
		}
	}
	return keysOf(seen)
}

func splitComma(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
