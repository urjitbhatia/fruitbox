package engine

import (
	"context"
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// CreateOptions controls Create.
type CreateOptions struct {
	Scale         map[string]int
	NoBuild       bool   // skip building images for services with a build section
	Pull          string // "always" pulls images before creating
	RemoveOrphans bool   // remove containers for services not in the compose file
}

// Create creates the project's networks, volumes and service containers
// without starting them (compose `create`).
func (e *Engine) Create(ctx context.Context, p *types.Project, opts CreateOptions) error {
	if !opts.NoBuild {
		if err := e.Build(ctx, p, nil, BuildOptions{}); err != nil {
			return err
		}
	}
	if opts.Pull == "always" {
		if err := e.Pull(ctx, p, nil, PullOptions{}); err != nil {
			return err
		}
	}
	if opts.RemoveOrphans {
		if err := e.removeOrphans(ctx, p); err != nil {
			return err
		}
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
		svc, err := p.GetService(name)
		if err != nil {
			return err
		}
		for _, warning := range translate.UnsupportedWarnings(svc) {
			e.logf("WARNING: %s: %s", svc.Name, warning)
		}
		for n := 1; n <= effectiveScale(svc, opts.Scale); n++ {
			mounts, err := e.prepareGeneratedMounts(p, svc, n)
			if err != nil {
				return err
			}
			args, err := translate.BuildRunArgs(p, svc, translate.RunOptions{
				Number:       n,
				Create:       true,
				ExtraVolumes: mounts,
			})
			if err != nil {
				return err
			}
			e.logf("Creating %s", containerName(p, svc, n))
			if _, err := e.Runner.Run(ctx, args...); err != nil {
				return fmt.Errorf("create %s: %w", svc.Name, err)
			}
		}
	}
	return nil
}

// RmOptions controls Rm.
type RmOptions struct {
	Force bool // also remove running containers
	Stop  bool // stop containers before removing
}

// Rm removes stopped service containers (compose `rm`).
func (e *Engine) Rm(ctx context.Context, p *types.Project, names []string, opts RmOptions) error {
	refs, err := e.containerNames(p, names)
	if err != nil {
		return err
	}
	for i := len(refs) - 1; i >= 0; i-- {
		if opts.Stop {
			_, _ = e.Runner.Run(ctx, e.stopArgs(p, refs[i])...)
		}
		args := []string{"delete"}
		if opts.Force {
			args = append(args, "--force")
		}
		args = append(args, refs[i].Container)
		e.logf("Removing %s", refs[i].Container)
		if _, err := e.Runner.Run(ctx, args...); err != nil {
			return err
		}
	}
	return nil
}

// PushOptions controls Push.
type PushOptions struct {
	Quiet          bool // suppress progress logs
	IncludeDeps    bool // also push dependencies of the named services
	IgnoreFailures bool // continue when an individual push fails
}

// Push pushes the images of the named services (or all) to their registries.
func (e *Engine) Push(ctx context.Context, p *types.Project, names []string, opts PushOptions) error {
	if len(names) == 0 {
		names = p.ServiceNames()
	}
	names = e.maybeIncludeDeps(p, names, opts.IncludeDeps)

	seen := map[string]bool{}
	for _, name := range names {
		svc, err := p.GetService(name)
		if err != nil {
			return err
		}
		image := svc.Image
		if image == "" && svc.Build != nil {
			image = translate.BuildImageTag(p.Name, svc)
		}
		if image == "" || seen[image] {
			continue
		}
		seen[image] = true
		if !opts.Quiet {
			e.logf("Pushing %s (%s)", svc.Name, image)
		}
		if _, err := e.Runner.Run(ctx, "image", "push", image); err != nil {
			if opts.IgnoreFailures {
				e.logf("WARNING: push %s failed: %v", image, err)
				continue
			}
			return fmt.Errorf("push %s: %w", image, err)
		}
	}
	return nil
}

// Scale brings the given services to the requested replica counts by starting
// missing replicas and removing surplus ones.
func (e *Engine) Scale(ctx context.Context, p *types.Project, scale map[string]int) error {
	for _, name := range sortedScaleKeys(scale) {
		svc, err := p.GetService(name)
		if err != nil {
			return err
		}
		target := scale[name]
		// Start replicas 1..target.
		for n := 1; n <= target; n++ {
			mounts, err := e.prepareGeneratedMounts(p, svc, n)
			if err != nil {
				return err
			}
			args, err := translate.BuildRunArgs(p, svc, translate.RunOptions{
				Number: n, Detach: true, ExtraVolumes: mounts,
			})
			if err != nil {
				return err
			}
			e.logf("Scaling %s: ensuring replica %d", svc.Name, n)
			_, _ = e.Runner.Run(ctx, args...)
			e.applySysctls(ctx, p, svc, n)
		}
		// Remove surplus replicas above target (best-effort up to a bound).
		for n := target + 1; n <= target+surplusScanWindow; n++ {
			cname := containerName(p, svc, n)
			_, _ = e.Runner.Run(ctx, "stop", cname)
			_, _ = e.Runner.Run(ctx, "delete", cname)
		}
	}
	return nil
}

// surplusScanWindow bounds how far above the target Scale looks for surplus
// replicas to remove.
const surplusScanWindow = 32

// Attach attaches to a running service container's standard streams.
func (e *Engine) Attach(ctx context.Context, p *types.Project, service string, index int) error {
	svc, err := p.GetService(service)
	if err != nil {
		return err
	}
	if index <= 0 {
		index = 1
	}
	return e.Runner.RunInteractive(ctx, "start", "--attach", "--interactive", containerName(p, svc, index))
}

func sortedScaleKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	return keys
}
