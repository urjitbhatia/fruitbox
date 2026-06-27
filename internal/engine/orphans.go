package engine

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// listedContainer is a tolerant view of one entry from `container ls --format json`.
type listedContainer struct {
	Name    string
	Labels  map[string]string
	status  string
	image   string
	ports   string
	project string
	service string
}

// projectOrphans returns the containers labelled for this project whose service
// is no longer defined in the compose file.
func (e *Engine) projectOrphans(ctx context.Context, p *types.Project) []listedContainer {
	res, err := e.Runner.Run(ctx, "list", "--all", "--format", "json")
	if err != nil {
		return nil
	}
	active := map[string]bool{}
	for _, s := range p.ServiceNames() {
		active[s] = true
	}
	var orphans []listedContainer
	for _, c := range parseContainerList(res.Stdout) {
		if c.project != p.Name || c.service == "" || active[c.service] {
			continue
		}
		orphans = append(orphans, c)
	}
	return orphans
}

// removeOrphans deletes containers that belong to this project (by label) but
// whose service is no longer defined in the compose file.
func (e *Engine) removeOrphans(ctx context.Context, p *types.Project) error {
	for _, c := range e.projectOrphans(ctx, p) {
		e.logf("Removing orphan container %s", c.Name)
		_, _ = e.Runner.Run(ctx, "stop", c.Name)
		_, _ = e.Runner.Run(ctx, "delete", c.Name)
	}
	return nil
}

// Orphans returns the project's orphan containers as status records, for
// `ps --orphans`.
func (e *Engine) Orphans(ctx context.Context, p *types.Project) []ContainerStatus {
	var out []ContainerStatus
	for _, c := range e.projectOrphans(ctx, p) {
		status := c.status
		if status == "" {
			status = "orphan"
		}
		out = append(out, ContainerStatus{Name: c.Name, Service: c.service, Status: status})
	}
	return out
}

// parseContainerList tolerantly parses container ls JSON into entries, scanning
// common key spellings for the name and label map without assuming a schema.
func parseContainerList(payload string) []listedContainer {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return nil
	}
	var generic any
	if err := json.Unmarshal([]byte(payload), &generic); err != nil {
		return nil
	}
	arr, ok := generic.([]any)
	if !ok {
		if obj, ok := generic.(map[string]any); ok {
			arr = []any{obj}
		} else {
			return nil
		}
	}
	var out []listedContainer
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		cfg, _ := m["configuration"].(map[string]any)
		c := listedContainer{
			Name:   containerName2(m, cfg),
			Labels: extractLabels(m),
			status: findString(m, "status", "Status", "state", "State"),
			image:  extractImage(cfg),
			ports:  extractPorts(cfg),
		}
		c.project = c.Labels[translate.LabelProject]
		c.service = c.Labels[translate.LabelService]
		out = append(out, c)
	}
	return out
}

// containerName2 resolves the container name from the top level or the nested
// configuration object.
func containerName2(m, cfg map[string]any) string {
	if s := findString(m, "name", "Name", "id", "ID"); s != "" {
		return s
	}
	if cfg != nil {
		return findString(cfg, "id", "name")
	}
	return ""
}

// extractImage pulls configuration.image.reference (the image tag) from a
// listed container's configuration.
func extractImage(cfg map[string]any) string {
	if cfg == nil {
		return ""
	}
	if img, ok := cfg["image"].(map[string]any); ok {
		if s, ok := img["reference"].(string); ok {
			return s
		}
	}
	if s, ok := cfg["image"].(string); ok {
		return s
	}
	return ""
}

// extractPorts renders configuration.publishedPorts as a docker-style
// "host:hostPort->containerPort/proto" comma list.
func extractPorts(cfg map[string]any) string {
	if cfg == nil {
		return ""
	}
	list, ok := cfg["publishedPorts"].([]any)
	if !ok {
		return ""
	}
	var parts []string
	for _, p := range list {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		host := numOrEmpty(pm["hostPort"])
		cport := numOrEmpty(pm["containerPort"])
		proto, _ := pm["proto"].(string)
		addr, _ := pm["hostAddress"].(string)
		if proto == "" {
			proto = "tcp"
		}
		if addr == "" {
			addr = "0.0.0.0"
		}
		if host == "" || cport == "" {
			continue
		}
		parts = append(parts, addr+":"+host+"->"+cport+"/"+proto)
	}
	return strings.Join(parts, ", ")
}

func numOrEmpty(v any) string {
	switch n := v.(type) {
	case float64:
		return strconv.Itoa(int(n))
	case int:
		return strconv.Itoa(n)
	case string:
		return n
	}
	return ""
}

// extractLabels finds a labels map under common locations, including a nested
// "configuration" object, and coerces values to strings.
func extractLabels(m map[string]any) map[string]string {
	candidates := []any{m["labels"], m["Labels"]}
	if cfg, ok := m["configuration"].(map[string]any); ok {
		candidates = append(candidates, cfg["labels"], cfg["Labels"])
	}
	for _, cand := range candidates {
		if lm, ok := cand.(map[string]any); ok {
			out := make(map[string]string, len(lm))
			for k, v := range lm {
				if s, ok := v.(string); ok {
					out[k] = s
				}
			}
			return out
		}
	}
	return map[string]string{}
}
