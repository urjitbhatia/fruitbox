package engine

import (
	"context"
	"sort"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// ProjectSummary aggregates the containers of one compose project found in the
// runtime.
type ProjectSummary struct {
	Name           string
	ContainerCount int
}

// ListProjects scans all containers and groups them by compose project label,
// returning one summary per project, sorted by name.
func (e *Engine) ListProjects(ctx context.Context) ([]ProjectSummary, error) {
	res, err := e.Runner.Run(ctx, "list", "--all", "--format", "json")
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, c := range parseContainerList(res.Stdout) {
		if c.project == "" {
			continue
		}
		counts[c.project]++
	}
	names := make([]string, 0, len(counts))
	for n := range counts {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]ProjectSummary, 0, len(names))
	for _, n := range names {
		out = append(out, ProjectSummary{Name: n, ContainerCount: counts[n]})
	}
	return out, nil
}

// Wait blocks until the named services' containers have stopped, returning the
// exit code of the last container to finish.
func (e *Engine) Wait(ctx context.Context, p *types.Project, names []string) (int, error) {
	if len(names) == 0 {
		names = p.ServiceNames()
	}
	lastCode := 0
	for _, name := range names {
		svc, err := p.GetService(name)
		if err != nil {
			return 0, err
		}
		for n := 1; n <= scaleOf(svc); n++ {
			cname := svc.ContainerName
			if cname == "" {
				cname = translate.ContainerName(p.Name, svc.Name, n)
			}
			code, err := e.waitForExit(ctx, cname)
			if err != nil {
				return 0, err
			}
			lastCode = code
		}
	}
	return lastCode, nil
}

// waitForExit polls a container until it stops, returning its exit code.
func (e *Engine) waitForExit(ctx context.Context, name string) (int, error) {
	for {
		res, err := e.Runner.Run(ctx, "inspect", name)
		if err == nil {
			if _, code, done := inspectExit(res.Stdout); done {
				return code, nil
			}
		}
		if err := e.sleep(ctx, time.Second); err != nil {
			return 0, err
		}
	}
}
