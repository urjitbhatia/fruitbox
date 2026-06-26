package engine

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// listedContainer is a tolerant view of one entry from `container ls --format json`.
type listedContainer struct {
	Name    string
	Labels  map[string]string
	status  string
	project string
	service string
}

// removeOrphans deletes containers that belong to this project (by label) but
// whose service is no longer defined in the compose file.
func (e *Engine) removeOrphans(ctx context.Context, p *types.Project) error {
	res, err := e.Runner.Run(ctx, "list", "--all", "--format", "json")
	if err != nil {
		// Listing isn't available; nothing we can safely do.
		return nil
	}
	entries := parseContainerList(res.Stdout)
	active := map[string]bool{}
	for _, s := range p.ServiceNames() {
		active[s] = true
	}
	for _, c := range entries {
		if c.project != p.Name {
			continue
		}
		if c.service == "" || active[c.service] {
			continue
		}
		e.logf("Removing orphan container %s", c.Name)
		_, _ = e.Runner.Run(ctx, "stop", c.Name)
		_, _ = e.Runner.Run(ctx, "delete", c.Name)
	}
	return nil
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
		c := listedContainer{
			Name:   findString(m, "name", "Name", "id", "ID"),
			Labels: extractLabels(m),
			status: findString(m, "status", "Status", "state", "State"),
		}
		c.project = c.Labels[translate.LabelProject]
		c.service = c.Labels[translate.LabelService]
		out = append(out, c)
	}
	return out
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
