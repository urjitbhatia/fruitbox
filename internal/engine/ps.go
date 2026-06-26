package engine

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// ContainerStatus is the runtime state of one expected service container.
type ContainerStatus struct {
	Name    string
	Service string
	Status  string
}

// Ps returns the status of every container the project expects, in dependency
// order. Containers that have not been created report status "not created".
func (e *Engine) Ps(ctx context.Context, p *types.Project) ([]ContainerStatus, error) {
	order, err := DependencyOrder(p)
	if err != nil {
		return nil, err
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
			out = append(out, ContainerStatus{
				Name:    cname,
				Service: svc.Name,
				Status:  e.probeStatus(ctx, cname),
			})
		}
	}
	return out, nil
}

// probeStatus inspects a single container and returns a coarse status string.
// It is tolerant of the exact inspect JSON shape emitted by the container CLI.
func (e *Engine) probeStatus(ctx context.Context, name string) string {
	res, err := e.Runner.Run(ctx, "inspect", name)
	if err != nil {
		return "not created"
	}
	if s := extractStatus(res.Stdout); s != "" {
		return s
	}
	return "created"
}

// extractStatus pulls a status/state field out of a container inspect payload,
// scanning common key names without committing to a fixed schema.
func extractStatus(payload string) string {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return ""
	}
	var generic any
	if err := json.Unmarshal([]byte(payload), &generic); err != nil {
		return ""
	}
	// inspect commonly returns an array of objects.
	switch v := generic.(type) {
	case []any:
		if len(v) > 0 {
			return statusFromNode(v[0])
		}
	default:
		return statusFromNode(generic)
	}
	return ""
}

func statusFromNode(node any) string {
	m, ok := node.(map[string]any)
	if !ok {
		return ""
	}
	for _, key := range []string{"status", "state", "Status", "State"} {
		if val, ok := m[key]; ok {
			if s, ok := val.(string); ok && s != "" {
				return s
			}
			// Nested {"status": {"state": "running"}} style.
			if nested, ok := val.(map[string]any); ok {
				if s := statusFromNode(nested); s != "" {
					return s
				}
			}
		}
	}
	return ""
}
