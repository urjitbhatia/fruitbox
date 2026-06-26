package engine

import (
	"context"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// LogsOptions controls log retrieval.
type LogsOptions struct {
	// Follow streams new output (--follow).
	Follow bool
	// Tail is the number of lines from the end ("all" or "" for everything).
	Tail string
	// Index selects a single replica (1-based); 0 means all replicas.
	Index int
}

// Logs streams logs for the named services (or all services when none are
// given) by delegating to the container runtime's logs command.
func (e *Engine) Logs(ctx context.Context, p *types.Project, services []string, opts LogsOptions) error {
	names := services
	if len(names) == 0 {
		names = p.ServiceNames()
	}
	for _, svcName := range names {
		svc, err := p.GetService(svcName)
		if err != nil {
			return err
		}
		for n := 1; n <= scaleOf(svc); n++ {
			if opts.Index > 0 && n != opts.Index {
				continue
			}
			cname := svc.ContainerName
			if cname == "" {
				cname = translate.ContainerName(p.Name, svc.Name, n)
			}
			args := []string{"logs"}
			if opts.Follow {
				args = append(args, "--follow")
			}
			if opts.Tail != "" && opts.Tail != "all" {
				args = append(args, "-n", opts.Tail)
			}
			args = append(args, cname)
			if err := e.Runner.RunInteractive(ctx, args...); err != nil {
				return err
			}
		}
	}
	return nil
}
