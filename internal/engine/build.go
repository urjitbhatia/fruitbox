package engine

import (
	"context"
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// Build builds images for the named services that declare a build section
// (or all such services when names is empty).
func (e *Engine) Build(ctx context.Context, p *types.Project, names []string) error {
	if len(names) == 0 {
		names = p.ServiceNames()
	}
	for _, name := range names {
		svc, err := p.GetService(name)
		if err != nil {
			return err
		}
		if err := e.buildService(ctx, p, svc); err != nil {
			return err
		}
	}
	return nil
}

// buildService builds a single service's image if it has a build section.
func (e *Engine) buildService(ctx context.Context, p *types.Project, svc types.ServiceConfig) error {
	args := translate.BuildBuildArgs(p.Name, svc)
	if args == nil {
		return nil // nothing to build
	}
	e.logf("Building %s", svc.Name)
	if _, err := e.Runner.Run(ctx, args...); err != nil {
		return fmt.Errorf("build %s: %w", svc.Name, err)
	}
	return nil
}
