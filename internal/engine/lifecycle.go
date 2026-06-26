package engine

import (
	"context"
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// containerNames returns the expected container names for the given services
// (or all services when names is empty), in dependency order.
func (e *Engine) containerNames(p *types.Project, names []string) ([]nameRef, error) {
	order, err := DependencyOrder(p)
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
	}
	var refs []nameRef
	for _, svcName := range order {
		if len(names) > 0 && !want[svcName] {
			continue
		}
		svc, err := p.GetService(svcName)
		if err != nil {
			return nil, err
		}
		for n := 1; n <= scaleOf(svc); n++ {
			cname := svc.ContainerName
			if cname == "" {
				cname = translate.ContainerName(p.Name, svc.Name, n)
			}
			refs = append(refs, nameRef{Service: svc.Name, Container: cname})
		}
	}
	return refs, nil
}

type nameRef struct {
	Service   string
	Container string
}

// Start starts existing (stopped) containers for the named services.
func (e *Engine) Start(ctx context.Context, p *types.Project, names []string) error {
	refs, err := e.containerNames(p, names)
	if err != nil {
		return err
	}
	for _, r := range refs {
		e.logf("Starting %s", r.Container)
		if _, err := e.Runner.Run(ctx, "start", r.Container); err != nil {
			return err
		}
	}
	return nil
}

// Stop stops running containers for the named services without removing them.
func (e *Engine) Stop(ctx context.Context, p *types.Project, names []string) error {
	refs, err := e.containerNames(p, names)
	if err != nil {
		return err
	}
	// Reverse order for a graceful shutdown.
	for i := len(refs) - 1; i >= 0; i-- {
		e.logf("Stopping %s", refs[i].Container)
		_, _ = e.Runner.Run(ctx, "stop", refs[i].Container)
	}
	return nil
}

// Restart restarts containers for the named services (stop then start).
func (e *Engine) Restart(ctx context.Context, p *types.Project, names []string) error {
	if err := e.Stop(ctx, p, names); err != nil {
		return err
	}
	return e.Start(ctx, p, names)
}

// Kill sends a signal to the named services' containers (default SIGKILL).
func (e *Engine) Kill(ctx context.Context, p *types.Project, names []string, signal string) error {
	refs, err := e.containerNames(p, names)
	if err != nil {
		return err
	}
	for i := len(refs) - 1; i >= 0; i-- {
		args := []string{"kill"}
		if signal != "" {
			args = append(args, "--signal", signal)
		}
		args = append(args, refs[i].Container)
		e.logf("Killing %s", refs[i].Container)
		_, _ = e.Runner.Run(ctx, args...)
	}
	return nil
}

// Pull pulls the images referenced by the named services (or all services).
func (e *Engine) Pull(ctx context.Context, p *types.Project, names []string) error {
	if len(names) == 0 {
		names = p.ServiceNames()
	}
	seen := map[string]bool{}
	for _, name := range names {
		svc, err := p.GetService(name)
		if err != nil {
			return err
		}
		// Services that build locally and have no explicit image are not pulled.
		if svc.Image == "" {
			continue
		}
		if seen[svc.Image] {
			continue
		}
		seen[svc.Image] = true
		e.logf("Pulling %s (%s)", svc.Name, svc.Image)
		if _, err := e.Runner.Run(ctx, "image", "pull", svc.Image); err != nil {
			return fmt.Errorf("pull %s: %w", svc.Image, err)
		}
	}
	return nil
}

// Exec runs a command in the first replica of a service's container,
// interactively wired to the process stdio.
func (e *Engine) Exec(ctx context.Context, p *types.Project, service string, command []string, opts ExecOptions) error {
	svc, err := p.GetService(service)
	if err != nil {
		return err
	}
	cname := svc.ContainerName
	if cname == "" {
		cname = translate.ContainerName(p.Name, svc.Name, 1)
	}
	args := []string{"exec"}
	if opts.Interactive {
		args = append(args, "--interactive")
	}
	if opts.TTY {
		args = append(args, "--tty")
	}
	if opts.User != "" {
		args = append(args, "--user", opts.User)
	}
	if opts.WorkingDir != "" {
		args = append(args, "--workdir", opts.WorkingDir)
	}
	for _, e := range opts.Env {
		args = append(args, "--env", e)
	}
	args = append(args, cname)
	args = append(args, command...)
	return e.Runner.RunInteractive(ctx, args...)
}

// ExecOptions controls the behavior of Exec.
type ExecOptions struct {
	Interactive bool
	TTY         bool
	User        string
	WorkingDir  string
	Env         []string
}
