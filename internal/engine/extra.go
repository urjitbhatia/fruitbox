package engine

import (
	"context"
	"sort"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// VolumeNames returns the resolved runtime names of the project's named
// volumes, sorted (for `compose volumes`).
func (e *Engine) VolumeNames(p *types.Project) []string {
	var out []string
	for _, vol := range p.Volumes {
		if vol.Name != "" {
			out = append(out, vol.Name)
		}
	}
	sort.Strings(out)
	return out
}

// StatsOptions controls Stats.
type StatsOptions struct {
	NoStream bool   // --no-stream: print one snapshot and exit
	Format   string // --format: table|json|yaml
}

// Stats streams resource usage for the project's containers by delegating to
// `container stats`.
func (e *Engine) Stats(ctx context.Context, p *types.Project, opts StatsOptions) error {
	refs, err := e.containerNames(p, nil)
	if err != nil {
		return err
	}
	args := []string{"stats"}
	if opts.NoStream {
		args = append(args, "--no-stream")
	}
	if opts.Format != "" {
		args = append(args, "--format", opts.Format)
	}
	for _, r := range refs {
		args = append(args, r.Container)
	}
	return e.Runner.RunInteractive(ctx, args...)
}

// Export writes a service container's filesystem to a tar archive via
// `container export`.
func (e *Engine) Export(ctx context.Context, p *types.Project, service, output string, index int) error {
	svc, err := p.GetService(service)
	if err != nil {
		return err
	}
	if index <= 0 {
		index = 1
	}
	cname := svc.ContainerName
	if cname == "" {
		cname = translate.ContainerName(p.Name, svc.Name, index)
	}
	args := []string{"export"}
	if output != "" {
		args = append(args, "--output", output)
	}
	args = append(args, cname)
	if output != "" {
		_, err := e.Runner.Run(ctx, args...)
		return err
	}
	return e.Runner.RunInteractive(ctx, args...)
}
