package engine

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/runner"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// Engine orchestrates a compose project against the container runtime.
type Engine struct {
	Runner runner.Runner
	// Out receives human-readable progress messages.
	Out io.Writer
}

// New returns an Engine using the given runner and progress writer.
func New(r runner.Runner, out io.Writer) *Engine {
	return &Engine{Runner: r, Out: out}
}

// UpOptions controls the behavior of Up.
type UpOptions struct {
	// Detach starts service containers in the background.
	Detach bool
}

// Up creates the project's networks and volumes, then starts every service
// container in dependency order.
func (e *Engine) Up(ctx context.Context, p *types.Project, opts UpOptions) error {
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
		svc, err := p.GetService(name)
		if err != nil {
			return err
		}
		if err := e.startService(ctx, p, svc, opts); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) ensureNetworks(ctx context.Context, p *types.Project) error {
	for _, key := range sortedKeys(p.Networks) {
		net := p.Networks[key]
		args := translate.BuildNetworkCreateArgs(p.Name, net)
		if args == nil {
			continue // external or nothing to create
		}
		e.logf("Creating network %q", net.Name)
		if _, err := e.Runner.Run(ctx, args...); err != nil {
			return fmt.Errorf("create network %s: %w", net.Name, err)
		}
	}
	return nil
}

func (e *Engine) ensureVolumes(ctx context.Context, p *types.Project) error {
	for _, key := range sortedKeys(p.Volumes) {
		vol := p.Volumes[key]
		args := translate.BuildVolumeCreateArgs(p.Name, vol)
		if args == nil {
			continue
		}
		e.logf("Creating volume %q", vol.Name)
		if _, err := e.Runner.Run(ctx, args...); err != nil {
			return fmt.Errorf("create volume %s: %w", vol.Name, err)
		}
	}
	return nil
}

func (e *Engine) startService(ctx context.Context, p *types.Project, svc types.ServiceConfig, opts UpOptions) error {
	replicas := scaleOf(svc)
	for n := 1; n <= replicas; n++ {
		args, err := translate.BuildRunArgs(p, svc, translate.RunOptions{
			Number: n,
			Detach: opts.Detach,
		})
		if err != nil {
			return fmt.Errorf("build run args for %s: %w", svc.Name, err)
		}
		e.logf("Starting %s", translate.ContainerName(p.Name, svc.Name, n))
		if _, err := e.Runner.Run(ctx, args...); err != nil {
			return fmt.Errorf("start %s: %w", svc.Name, err)
		}
	}
	return nil
}

// DownOptions controls the behavior of Down.
type DownOptions struct {
	// RemoveVolumes also deletes the project's named volumes.
	RemoveVolumes bool
}

// Down stops and removes the project's service containers in reverse
// dependency order, then removes its networks (and optionally volumes).
func (e *Engine) Down(ctx context.Context, p *types.Project, opts DownOptions) error {
	order, err := DependencyOrder(p)
	if err != nil {
		return err
	}
	// Reverse order for teardown.
	for i := len(order) - 1; i >= 0; i-- {
		name := order[i]
		svc, err := p.GetService(name)
		if err != nil {
			return err
		}
		for n := 1; n <= scaleOf(svc); n++ {
			cname := svc.ContainerName
			if cname == "" {
				cname = translate.ContainerName(p.Name, svc.Name, n)
			}
			e.logf("Stopping %s", cname)
			// Best-effort: ignore errors for containers that don't exist.
			_, _ = e.Runner.Run(ctx, "stop", cname)
			e.logf("Removing %s", cname)
			_, _ = e.Runner.Run(ctx, "delete", cname)
		}
	}

	for _, key := range sortedKeys(p.Networks) {
		net := p.Networks[key]
		if net.External {
			continue
		}
		e.logf("Removing network %q", net.Name)
		_, _ = e.Runner.Run(ctx, "network", "delete", net.Name)
	}

	if opts.RemoveVolumes {
		for _, key := range sortedKeys(p.Volumes) {
			vol := p.Volumes[key]
			if vol.External {
				continue
			}
			e.logf("Removing volume %q", vol.Name)
			_, _ = e.Runner.Run(ctx, "volume", "delete", vol.Name)
		}
	}
	return nil
}

func (e *Engine) logf(format string, args ...any) {
	if e.Out == nil {
		return
	}
	fmt.Fprintf(e.Out, format+"\n", args...)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
