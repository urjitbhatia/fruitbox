package engine

import (
	"context"
	"fmt"
	"time"

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

// Stop stops running containers for the named services without removing them,
// honoring each service's stop_signal and stop_grace_period. A non-nil timeout
// overrides the grace period (--time).
func (e *Engine) Stop(ctx context.Context, p *types.Project, names []string, timeout *int) error {
	refs, err := e.containerNames(p, names)
	if err != nil {
		return err
	}
	// Reverse order for a graceful shutdown.
	for i := len(refs) - 1; i >= 0; i-- {
		_, _ = e.Runner.Run(ctx, e.stopArgsTimeout(p, refs[i], timeout)...)
		e.logf("Stopping %s", refs[i].Container)
	}
	return nil
}

// stopArgs builds the `container stop` arguments for a container, applying the
// service's stop_signal (--signal) and stop_grace_period (--time).
func (e *Engine) stopArgs(p *types.Project, r nameRef) []string {
	return e.stopArgsTimeout(p, r, nil)
}

// stopArgsTimeout builds `container stop` args. When override is non-nil it
// sets the grace period (--time), overriding the service's stop_grace_period
// (used by `down -t`/`stop -t`).
func (e *Engine) stopArgsTimeout(p *types.Project, r nameRef, override *int) []string {
	args := []string{"stop"}
	svc, err := p.GetService(r.Service)
	if err == nil && svc.StopSignal != "" {
		args = append(args, "--signal", svc.StopSignal)
	}
	switch {
	case override != nil:
		args = append(args, "--time", fmt.Sprintf("%d", *override))
	case err == nil && svc.StopGracePeriod != nil:
		secs := int(time.Duration(*svc.StopGracePeriod).Seconds())
		args = append(args, "--time", fmt.Sprintf("%d", secs))
	}
	return append(args, r.Container)
}

// Restart restarts containers for the named services (stop then start). A
// non-nil timeout overrides the stop grace period.
func (e *Engine) Restart(ctx context.Context, p *types.Project, names []string, timeout *int) error {
	if err := e.Stop(ctx, p, names, timeout); err != nil {
		return err
	}
	return e.Start(ctx, p, names)
}

// Kill sends a signal to the named services' containers (default SIGKILL),
// optionally also removing orphan containers afterward.
func (e *Engine) Kill(ctx context.Context, p *types.Project, names []string, signal string, removeOrphans bool) error {
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
	if removeOrphans {
		return e.removeOrphans(ctx, p)
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
