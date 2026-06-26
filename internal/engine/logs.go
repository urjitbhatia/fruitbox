package engine

import (
	"context"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// Logs streams logs for the named services (or all services when none are
// given) by delegating to the container runtime's logs command.
func (e *Engine) Logs(ctx context.Context, p *types.Project, services []string, follow bool) error {
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
			cname := svc.ContainerName
			if cname == "" {
				cname = translate.ContainerName(p.Name, svc.Name, n)
			}
			args := []string{"logs"}
			if follow {
				args = append(args, "--follow")
			}
			args = append(args, cname)
			if err := e.Runner.RunInteractive(ctx, args...); err != nil {
				return err
			}
		}
	}
	return nil
}
