package engine

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// recreateDecision is what Up should do with a single service container.
type recreateDecision int

const (
	// decisionCreate: the container does not exist; create and start it fresh.
	decisionCreate recreateDecision = iota
	// decisionRecreate: an existing container must be removed and recreated.
	decisionRecreate
	// decisionStart: an up-to-date container exists; just ensure it is running.
	decisionStart
)

// decideContainer inspects an existing container and decides whether to create,
// recreate, or simply (re)start it, honoring --force-recreate / --no-recreate
// and otherwise comparing the config hash.
func (e *Engine) decideContainer(ctx context.Context, cname, desiredHash string, opts UpOptions) recreateDecision {
	res, err := e.Runner.Run(ctx, "inspect", cname)
	if err != nil || strings.TrimSpace(res.Stdout) == "" {
		// Inspect failed or returned nothing: treat the container as absent.
		return decisionCreate
	}
	if opts.NoRecreate {
		return decisionStart
	}
	if opts.ForceRecreate {
		return decisionRecreate
	}
	if configHashFromInspect(res.Stdout) != desiredHash {
		return decisionRecreate
	}
	return decisionStart
}

// configHashFromInspect extracts the config-hash label from a container inspect
// payload, tolerating the runtime's exact JSON shape.
func configHashFromInspect(payload string) string {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return ""
	}
	var generic any
	if err := json.Unmarshal([]byte(payload), &generic); err != nil {
		return ""
	}
	node := generic
	if arr, ok := generic.([]any); ok {
		if len(arr) == 0 {
			return ""
		}
		node = arr[0]
	}
	m, ok := node.(map[string]any)
	if !ok {
		return ""
	}
	return extractLabels(m)[translate.LabelConfigHash]
}
