package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// RunOneOffOptions controls a one-off `run` invocation.
type RunOneOffOptions struct {
	// Command overrides the service command for this run.
	Command []string
	// Detach runs the one-off container in the background.
	Detach bool
	// Remove deletes the container after it exits (default true for run).
	Remove bool
	// NoDeps skips starting the service's dependencies.
	NoDeps bool
	// Name overrides the generated one-off container name.
	Name string
	// Interactive/TTY wire the container to the terminal.
	Interactive bool
	TTY         bool
	// Env are additional environment variables (KEY=VALUE) for this run.
	Env []string

	// Override flags mirroring `docker compose run`.
	Entrypoint    string   // --entrypoint
	EntrypointSet bool     // whether --entrypoint was provided (allows clearing)
	User          string   // --user
	WorkDir       string   // --workdir
	Labels        []string // --label KEY=VALUE
	Volumes       []string // --volume specs
	Publish       []string // --publish specs
	CapAdd        []string // --cap-add
	CapDrop       []string // --cap-drop
	ServicePorts  bool     // --service-ports: map the service's declared ports
	Build         bool     // --build: build image before running
	RemoveOrphans bool     // --remove-orphans
}

// RunOneOff starts dependencies (unless NoDeps) and then runs a single one-off
// container for the named service, mirroring `docker compose run`.
func (e *Engine) RunOneOff(ctx context.Context, p *types.Project, service string, opts RunOneOffOptions) error {
	svc, err := p.GetService(service)
	if err != nil {
		return err
	}

	if opts.Build {
		if err := e.buildService(ctx, p, svc); err != nil {
			return err
		}
	}
	if !opts.NoDeps {
		if err := e.startDependencies(ctx, p, svc); err != nil {
			return err
		}
	}
	if opts.RemoveOrphans {
		if err := e.removeOrphans(ctx, p); err != nil {
			return err
		}
	}

	runSvc := applyRunOverrides(svc, opts)

	name := opts.Name
	if name == "" {
		name = fmt.Sprintf("%s-%s-run", sanitizeName(p.Name), sanitizeName(svc.Name))
	}
	args, err := translate.BuildRunArgs(p, runSvc, translate.RunOptions{
		Number:       1,
		Detach:       opts.Detach,
		Remove:       opts.Remove,
		Oneoff:       true,
		NameOverride: name,
		Interactive:  opts.Interactive,
		TTY:          opts.TTY,
		ExtraVolumes: opts.Volumes,
	})
	if err != nil {
		return err
	}

	for _, warning := range translate.UnsupportedWarnings(runSvc) {
		e.logf("WARNING: %s: %s", svc.Name, warning)
	}

	if opts.Detach {
		_, err := e.Runner.Run(ctx, args...)
		return err
	}
	return e.Runner.RunInteractive(ctx, args...)
}

// applyRunOverrides returns a copy of svc with `docker compose run` overrides
// applied: ports are dropped unless --service-ports, the command/entrypoint may
// be replaced, and user/workdir/labels/caps/publish/env are merged.
func applyRunOverrides(svc types.ServiceConfig, opts RunOneOffOptions) types.ServiceConfig {
	out := svc // value copy; we reassign slices/maps below rather than mutating

	// By default `run` does NOT publish the service's declared ports.
	if !opts.ServicePorts {
		out.Ports = nil
	}
	for _, spec := range opts.Publish {
		if ports, err := types.ParsePortConfig(spec); err == nil {
			out.Ports = append(append([]types.ServicePortConfig(nil), out.Ports...), ports...)
		}
	}

	if len(opts.Command) > 0 {
		out.Command = opts.Command
	}
	if opts.EntrypointSet {
		if opts.Entrypoint == "" {
			out.Entrypoint = nil
		} else {
			out.Entrypoint = types.ShellCommand{opts.Entrypoint}
		}
	}
	if opts.User != "" {
		out.User = opts.User
	}
	if opts.WorkDir != "" {
		out.WorkingDir = opts.WorkDir
	}
	if len(opts.CapAdd) > 0 {
		out.CapAdd = append(append([]string(nil), out.CapAdd...), opts.CapAdd...)
	}
	if len(opts.CapDrop) > 0 {
		out.CapDrop = append(append([]string(nil), out.CapDrop...), opts.CapDrop...)
	}
	if len(opts.Labels) > 0 {
		merged := types.Labels{}
		for k, v := range out.Labels {
			merged[k] = v
		}
		for _, l := range opts.Labels {
			k, v, _ := strings.Cut(l, "=")
			merged[k] = v
		}
		out.Labels = merged
	}
	if len(opts.Env) > 0 {
		merged := types.MappingWithEquals{}
		for k, v := range out.Environment {
			merged[k] = v
		}
		for _, e := range opts.Env {
			k, v, ok := strings.Cut(e, "=")
			if ok {
				vv := v
				merged[k] = &vv
			} else {
				merged[k] = nil
			}
		}
		out.Environment = merged
	}
	return out
}

// startDependencies brings up (and waits on) all of a service's dependencies in
// dependency order, without starting the service itself.
func (e *Engine) startDependencies(ctx context.Context, p *types.Project, svc types.ServiceConfig) error {
	deps := transitiveDeps(p, svc)
	if len(deps) == 0 {
		return nil
	}
	if err := e.ensureNetworks(ctx, p); err != nil {
		return err
	}
	if err := e.ensureVolumes(ctx, p); err != nil {
		return err
	}
	order, err := DependencyOrder(p)
	if err != nil {
		return err
	}
	for _, name := range order {
		if !deps[name] {
			continue
		}
		depSvc, err := p.GetService(name)
		if err != nil {
			return err
		}
		if err := e.waitForDependencies(ctx, p, depSvc); err != nil {
			return err
		}
		if err := e.startService(ctx, p, depSvc, UpOptions{Detach: true}); err != nil {
			return err
		}
	}
	return nil
}

// transitiveDeps returns the set of services svc depends on, transitively.
func transitiveDeps(p *types.Project, svc types.ServiceConfig) map[string]bool {
	seen := map[string]bool{}
	var visit func(s types.ServiceConfig)
	visit = func(s types.ServiceConfig) {
		for dep := range s.DependsOn {
			if seen[dep] {
				continue
			}
			depSvc, err := p.GetService(dep)
			if err != nil {
				continue
			}
			seen[dep] = true
			visit(depSvc)
		}
	}
	visit(svc)
	return seen
}

func sanitizeName(s string) string {
	return translate.ContainerNameSegment(s)
}
