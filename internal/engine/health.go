package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// Default healthcheck timings, matching Docker's defaults.
const (
	defaultHealthInterval = 30 * time.Second
	defaultHealthTimeout  = 30 * time.Second
	defaultHealthRetries  = 3
)

// waitForDependencies blocks until every depends_on condition of svc is met.
func (e *Engine) waitForDependencies(ctx context.Context, p *types.Project, svc types.ServiceConfig) error {
	for _, depName := range sortedDepNames(svc.DependsOn) {
		dep := svc.DependsOn[depName]
		depSvc, err := p.GetService(depName)
		if err != nil {
			// Dependency not part of the active project; skip (compose tolerates
			// this for disabled/profiled services unless required).
			if dep.Required {
				return fmt.Errorf("service %q depends on missing service %q", svc.Name, depName)
			}
			continue
		}
		cname := containerName(p, depSvc, 1)
		switch dep.Condition {
		case types.ServiceConditionHealthy:
			e.logf("Waiting for %s to be healthy", cname)
			if err := e.waitHealthy(ctx, cname, depSvc.HealthCheck); err != nil {
				if dep.Required {
					return fmt.Errorf("dependency %q never became healthy: %w", depName, err)
				}
			}
		case types.ServiceConditionCompletedSuccessfully:
			e.logf("Waiting for %s to complete", cname)
			if err := e.waitCompleted(ctx, cname); err != nil {
				if dep.Required {
					return fmt.Errorf("dependency %q did not complete successfully: %w", depName, err)
				}
			}
		default:
			// service_started (or empty): the dependency was already started in
			// dependency order, so there is nothing to wait for.
		}
	}
	return nil
}

// waitHealthy runs the service's healthcheck via `container exec` until it
// passes, or fails after the configured number of retries. Apple's container
// runtime does not run healthchecks itself, so fruitbox supervises them.
func (e *Engine) waitHealthy(ctx context.Context, name string, hc *types.HealthCheckConfig) error {
	if hc == nil || hc.Disable || isNoneTest(hc) {
		// No healthcheck defined: treat "started" as healthy.
		return nil
	}
	interval := durationOr(hc.Interval, defaultHealthInterval)
	retries := defaultHealthRetries
	if hc.Retries != nil {
		retries = int(*hc.Retries)
	}
	startPeriod := durationOr(hc.StartPeriod, 0)
	execArgs := healthExecArgs(name, hc.Test)

	start := e.now()
	failures := 0
	for {
		res, _ := e.Runner.Run(ctx, execArgs...)
		if res.ExitCode == 0 {
			return nil
		}
		inStartPeriod := e.now().Sub(start) < startPeriod
		if !inStartPeriod {
			failures++
			if failures >= retries {
				return fmt.Errorf("healthcheck failed %d times", failures)
			}
		}
		if err := e.sleep(ctx, interval); err != nil {
			return err
		}
	}
}

// waitCompleted polls until the container has stopped, returning an error if it
// exited with a non-zero status.
func (e *Engine) waitCompleted(ctx context.Context, name string) error {
	for {
		res, err := e.Runner.Run(ctx, "inspect", name)
		if err == nil {
			if state, code, done := inspectExit(res.Stdout); done {
				if code != 0 {
					return fmt.Errorf("container %s exited with code %d (%s)", name, code, state)
				}
				return nil
			}
		}
		if err := e.sleep(ctx, time.Second); err != nil {
			return err
		}
	}
}

// healthExecArgs builds the `container exec` argument vector for a healthcheck
// test. Compose tests are ["CMD", arg...] or ["CMD-SHELL", "script"].
func healthExecArgs(name string, test types.HealthCheckTest) []string {
	args := []string{"exec", name}
	if len(test) == 0 {
		return append(args, "true")
	}
	switch test[0] {
	case "CMD":
		return append(args, test[1:]...)
	case "CMD-SHELL":
		return append(args, "/bin/sh", "-c", joinShell(test[1:]))
	default:
		// Bare list form: treat as an exec-form command.
		return append(args, test...)
	}
}

func joinShell(parts []string) string {
	if len(parts) == 1 {
		return parts[0]
	}
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " "
		}
		out += p
	}
	return out
}

func isNoneTest(hc *types.HealthCheckConfig) bool {
	return len(hc.Test) > 0 && hc.Test[0] == "NONE"
}

func durationOr(d *types.Duration, fallback time.Duration) time.Duration {
	if d == nil {
		return fallback
	}
	return time.Duration(*d)
}

func containerName(p *types.Project, svc types.ServiceConfig, n int) string {
	if svc.ContainerName != "" {
		return svc.ContainerName
	}
	return translate.ContainerName(p.Name, svc.Name, n)
}

func sortedDepNames(deps types.DependsOnConfig) []string {
	names := make([]string, 0, len(deps))
	for k := range deps {
		names = append(names, k)
	}
	// Stable order for deterministic waiting/logging.
	sort.Strings(names)
	return names
}

// inspectExit tolerantly extracts the terminal state of a container from an
// inspect payload. It returns (stateString, exitCode, done) where done is true
// only when the container has stopped/exited. The container CLI's exact JSON
// shape is not assumed; common field names are scanned.
func inspectExit(payload string) (string, int, bool) {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return "", 0, false
	}
	var generic any
	if err := json.Unmarshal([]byte(payload), &generic); err != nil {
		return "", 0, false
	}
	node := generic
	if arr, ok := generic.([]any); ok {
		if len(arr) == 0 {
			return "", 0, false
		}
		node = arr[0]
	}
	m, ok := node.(map[string]any)
	if !ok {
		return "", 0, false
	}

	state := findString(m, "status", "state", "Status", "State")
	stopped := state == "stopped" || state == "exited" || state == "dead"
	code, hasCode := findInt(m, "exit_code", "exitCode", "ExitCode")
	if !stopped && !hasCode {
		return state, 0, false
	}
	return state, code, true
}

// findString returns the first string value found among the given keys,
// descending into nested objects one level for keys like {"state":{...}}.
func findString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok {
				return s
			}
			if nested, ok := v.(map[string]any); ok {
				if s := findString(nested, keys...); s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func findInt(m map[string]any, keys ...string) (int, bool) {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch n := v.(type) {
			case float64:
				return int(n), true
			case int:
				return n, true
			}
		}
		// Look one level down (e.g. {"state":{"exit_code":1}}).
		for _, v := range m {
			if nested, ok := v.(map[string]any); ok {
				if iv, ok := findInt(nested, k); ok {
					return iv, true
				}
			}
		}
	}
	return 0, false
}
