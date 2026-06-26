package engine

import (
	"context"
	"fmt"

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
}

// RunOneOff starts dependencies (unless NoDeps) and then runs a single one-off
// container for the named service, mirroring `docker compose run`.
func (e *Engine) RunOneOff(ctx context.Context, p *types.Project, service string, opts RunOneOffOptions) error {
	svc, err := p.GetService(service)
	if err != nil {
		return err
	}

	if !opts.NoDeps {
		if err := e.startDependencies(ctx, p, svc); err != nil {
			return err
		}
	}

	runOpts := translate.RunOptions{
		Number: 1,
		Detach: opts.Detach,
		Remove: opts.Remove,
		Oneoff: true,
	}
	args, err := translate.BuildRunArgs(p, svc, runOpts)
	if err != nil {
		return err
	}
	args = applyOneOffOverrides(args, svc, p, opts)

	for _, warning := range translate.UnsupportedWarnings(svc) {
		e.logf("WARNING: %s: %s", svc.Name, warning)
	}

	if opts.Detach {
		_, err := e.Runner.Run(ctx, args...)
		return err
	}
	return e.Runner.RunInteractive(ctx, args...)
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

// applyOneOffOverrides adjusts a generated run arg vector for one-off semantics:
// a unique container name, interactive/tty flags, extra env, and a command
// override that replaces the service's default command.
func applyOneOffOverrides(args []string, svc types.ServiceConfig, p *types.Project, opts RunOneOffOptions) []string {
	// Replace the generated --name with a one-off name.
	name := opts.Name
	if name == "" {
		name = fmt.Sprintf("%s-%s-run", sanitizeName(p.Name), sanitizeName(svc.Name))
	}
	args = replaceFlagValue(args, "--name", name)

	// Find the image boundary: everything from the image onward is image+command.
	imageIdx := imageIndex(args, svc, p)

	// Inject interactive/tty/env flags before the image.
	var inject []string
	if opts.Interactive {
		inject = append(inject, "--interactive")
	}
	if opts.TTY {
		inject = append(inject, "--tty")
	}
	for _, env := range opts.Env {
		inject = append(inject, "--env", env)
	}

	head := append([]string{}, args[:imageIdx]...)
	head = append(head, inject...)

	if len(opts.Command) > 0 {
		// Image only, then the override command.
		image := args[imageIdx]
		return append(append(head, image), opts.Command...)
	}
	// Keep the original image+command tail.
	return append(head, args[imageIdx:]...)
}

func imageIndex(args []string, svc types.ServiceConfig, p *types.Project) int {
	image := svc.Image
	if image == "" && svc.Build != nil {
		image = translate.BuildImageTag(p.Name, svc)
	}
	// Search from the end so we don't match a flag value equal to the image.
	for i := len(args) - 1; i >= 0; i-- {
		if args[i] == image {
			return i
		}
	}
	return len(args)
}

func replaceFlagValue(args []string, flag, value string) []string {
	out := append([]string{}, args...)
	for i := 0; i+1 < len(out); i++ {
		if out[i] == flag {
			out[i+1] = value
			return out
		}
	}
	return out
}

func sanitizeName(s string) string {
	return translate.ContainerNameSegment(s)
}
