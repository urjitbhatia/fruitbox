package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// BuildOptions overrides build settings from the CLI (`docker compose build`).
type BuildOptions struct {
	BuildArgs []string // extra KEY=VALUE build args (override compose)
	NoCache   bool     // force --no-cache
	Pull      bool     // force --pull
	Quiet     bool     // -q/--quiet
	Memory    string   // -m/--memory
}

// Build builds images for the named services that declare a build section
// (or all such services when names is empty).
func (e *Engine) Build(ctx context.Context, p *types.Project, names []string, opts BuildOptions) error {
	if len(names) == 0 {
		names = p.ServiceNames()
	}
	for _, name := range names {
		svc, err := p.GetService(name)
		if err != nil {
			return err
		}
		if err := e.buildService(ctx, p, applyBuildOverrides(svc, opts), opts); err != nil {
			return err
		}
	}
	return nil
}

// applyBuildOverrides returns a copy of svc with CLI build flags merged into its
// build config.
func applyBuildOverrides(svc types.ServiceConfig, opts BuildOptions) types.ServiceConfig {
	if svc.Build == nil {
		return svc
	}
	b := *svc.Build // copy
	if opts.NoCache {
		b.NoCache = true
	}
	if opts.Pull {
		b.Pull = true
	}
	if len(opts.BuildArgs) > 0 {
		merged := types.MappingWithEquals{}
		for k, v := range b.Args {
			merged[k] = v
		}
		for _, kv := range opts.BuildArgs {
			k, v, ok := strings.Cut(kv, "=")
			if ok {
				vv := v
				merged[k] = &vv
			} else {
				merged[k] = nil
			}
		}
		b.Args = merged
	}
	svc.Build = &b
	return svc
}

// buildService builds a single service's image if it has a build section.
func (e *Engine) buildService(ctx context.Context, p *types.Project, svc types.ServiceConfig, opts BuildOptions) error {
	args := translate.BuildBuildArgs(p.Name, svc, translate.BuildExtra{Quiet: opts.Quiet, Memory: opts.Memory})
	if args == nil {
		return nil // nothing to build
	}
	if !opts.Quiet {
		e.logf("Building %s", svc.Name)
	}
	if _, err := e.Runner.Run(ctx, args...); err != nil {
		return fmt.Errorf("build %s: %w", svc.Name, err)
	}
	return nil
}
