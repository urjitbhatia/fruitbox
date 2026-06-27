package engine

import (
	"context"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// ContainerStatus is the runtime state of one expected service container.
type ContainerStatus struct {
	Name    string
	Service string
	Status  string
	Image   string
	Ports   string
}

// Ps returns the status of every container the project expects, in dependency
// order, enriched with the live image/ports/status from the runtime. Containers
// that have not been created report status "not created" and their configured
// image.
func (e *Engine) Ps(ctx context.Context, p *types.Project) ([]ContainerStatus, error) {
	order, err := DependencyOrder(p)
	if err != nil {
		return nil, err
	}

	// One `container ls` call enriches every row (image/ports/state).
	live := map[string]listedContainer{}
	if res, err := e.Runner.Run(ctx, "list", "--all", "--format", "json"); err == nil {
		for _, c := range parseContainerList(res.Stdout) {
			live[c.Name] = c
		}
	}

	var out []ContainerStatus
	for _, name := range order {
		svc, err := p.GetService(name)
		if err != nil {
			return nil, err
		}
		for n := 1; n <= scaleOf(svc); n++ {
			cname := svc.ContainerName
			if cname == "" {
				cname = translate.ContainerName(p.Name, svc.Name, n)
			}
			row := ContainerStatus{
				Name:    cname,
				Service: svc.Name,
				Status:  "not created",
				Image:   serviceImage(p, svc),
				Ports:   configuredPorts(svc),
			}
			if c, ok := live[cname]; ok {
				if c.status != "" {
					row.Status = c.status
				} else {
					row.Status = "created"
				}
				if c.image != "" {
					row.Image = c.image
				}
				if c.ports != "" {
					row.Ports = c.ports
				}
			}
			out = append(out, row)
		}
	}
	return out, nil
}

// serviceImage returns a service's effective image (built tag when build-only).
func serviceImage(p *types.Project, svc types.ServiceConfig) string {
	if svc.Image != "" {
		return svc.Image
	}
	if svc.Build != nil {
		return translate.BuildImageTag(p.Name, svc)
	}
	return ""
}

// configuredPorts renders a service's declared ports as "host->container/proto".
func configuredPorts(svc types.ServiceConfig) string {
	var parts []string
	for _, port := range svc.Ports {
		s := ""
		if port.Published != "" {
			s = port.Published + "->"
		}
		s += itoa(int(port.Target))
		proto := port.Protocol
		if proto == "" {
			proto = "tcp"
		}
		s += "/" + proto
		parts = append(parts, s)
	}
	return joinComma(parts)
}

func joinComma(parts []string) string {
	out := ""
	for i, s := range parts {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}
