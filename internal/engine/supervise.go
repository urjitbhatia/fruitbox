package engine

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// restartPolicy is the normalized restart behavior for a service.
type restartPolicy struct {
	mode       string // "no", "always", "on-failure", "unless-stopped"
	maxRetries int    // 0 means unlimited (for on-failure)
}

// wantsRestart reports whether a container that exited with exitCode should be
// restarted, given how many restarts have already happened.
func (p restartPolicy) wantsRestart(exitCode, restarts int) bool {
	switch p.mode {
	case "always", "unless-stopped":
		return true
	case "on-failure":
		if exitCode == 0 {
			return false
		}
		return p.maxRetries == 0 || restarts < p.maxRetries
	default: // "no" or empty
		return false
	}
}

// parseRestartPolicy derives a restart policy from a service's `restart:` field,
// falling back to deploy.restart_policy.condition. The compose short form
// "on-failure:5" encodes max retries.
func parseRestartPolicy(svc types.ServiceConfig) restartPolicy {
	spec := svc.Restart
	if spec == "" && svc.Deploy != nil && svc.Deploy.RestartPolicy != nil {
		// Map the long-form condition to a short-form mode.
		switch svc.Deploy.RestartPolicy.Condition {
		case "any":
			spec = "always"
		case "on-failure":
			spec = "on-failure"
		case "none":
			spec = "no"
		}
		if svc.Deploy.RestartPolicy.MaxAttempts != nil {
			spec += ":" + strconv.FormatUint(*svc.Deploy.RestartPolicy.MaxAttempts, 10)
		}
	}
	mode, rest, _ := strings.Cut(spec, ":")
	p := restartPolicy{mode: mode}
	if n, err := strconv.Atoi(rest); err == nil {
		p.maxRetries = n
	}
	return p
}

// watched tracks one supervised container.
type watched struct {
	svc      types.ServiceConfig
	name     string
	runArgs  []string
	policy   restartPolicy
	restarts int
}

// SuperviseOptions controls foreground supervision (`up` without -d).
type SuperviseOptions struct {
	// AbortOnExit stops all containers when any one exits.
	AbortOnExit bool
	// AbortOnFailure stops all containers when any one exits non-zero.
	AbortOnFailure bool
	// ExitCodeFrom returns the exit code of this service's container (and
	// implies AbortOnExit).
	ExitCodeFrom string
}

// ExitError carries a container exit code so a foreground `up` can propagate it
// as the process exit status (used by --exit-code-from / --abort-on-*).
type ExitError struct{ Code int }

func (e ExitError) Error() string { return fmt.Sprintf("container exited with code %d", e.Code) }

// Supervise watches the containers of the named services and restarts them
// according to their policy when they exit. It returns when every watched
// container has reached a terminal state, the context is cancelled, or (for
// --abort-on-*) the first qualifying exit, after stopping the rest. It is
// intended for a foreground `up`.
func (e *Engine) Supervise(ctx context.Context, p *types.Project, names []string, opts SuperviseOptions) error {
	if len(names) == 0 {
		names = p.ServiceNames()
	}
	abort := opts.AbortOnExit || opts.AbortOnFailure || opts.ExitCodeFrom != ""
	if opts.AbortOnFailure || opts.ExitCodeFrom != "" {
		// Apple's container runtime does not report process exit codes via
		// inspect (only "stopped"), so failure detection and exit-code
		// propagation fall back to 0. Surface this rather than mislead.
		e.logf("WARNING: the container runtime does not report exit codes; " +
			"failure detection / --exit-code-from will report 0")
	}

	var items []*watched
	for _, name := range names {
		svc, err := p.GetService(name)
		if err != nil {
			return err
		}
		policy := parseRestartPolicy(svc)
		// Watch every container so a foreground up blocks until services stop;
		// only those with a policy are actually restarted.
		for n := 1; n <= scaleOf(svc); n++ {
			cname := containerName(p, svc, n)
			args, err := translate.BuildRunArgs(p, svc, translate.RunOptions{Number: n, Detach: true})
			if err != nil {
				return err
			}
			items = append(items, &watched{svc: svc, name: cname, runArgs: args, policy: policy})
		}
	}

	for len(items) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		remaining := items[:0]
		for _, w := range items {
			res, err := e.Runner.Run(ctx, "inspect", w.name)
			if err != nil {
				// Can't inspect (likely gone); keep watching briefly.
				remaining = append(remaining, w)
				continue
			}
			_, code, done := inspectExit(res.Stdout)
			if !done {
				remaining = append(remaining, w) // still running
				continue
			}

			// Abort handling takes precedence over restart policies.
			if abort && shouldAbort(opts, w.svc.Name, code) {
				e.logf("%s exited (code %d); stopping the rest", w.name, code)
				rc := code
				if opts.ExitCodeFrom != "" && w.svc.Name != opts.ExitCodeFrom {
					// Stop everything, then read the chosen service's code.
					rc = e.stopAllAndCode(ctx, items, opts.ExitCodeFrom)
				} else {
					e.stopAll(ctx, items, w.name)
				}
				return ExitError{Code: rc}
			}

			if w.policy.wantsRestart(code, w.restarts) {
				w.restarts++
				e.logf("Restarting %s (exit %d, attempt %d)", w.name, code, w.restarts)
				_, _ = e.Runner.Run(ctx, append([]string{"start"}, w.name)...)
				remaining = append(remaining, w)
			}
			// otherwise: terminal, drop from the watch set.
		}
		items = remaining
		if len(items) == 0 {
			break
		}
		if err := e.sleep(ctx, time.Second); err != nil {
			return err
		}
	}
	return nil
}

// shouldAbort reports whether an exit should trigger an abort given the options.
func shouldAbort(opts SuperviseOptions, service string, code int) bool {
	if opts.ExitCodeFrom != "" && service == opts.ExitCodeFrom {
		return true
	}
	if opts.AbortOnFailure {
		return code != 0
	}
	return opts.AbortOnExit
}

// stopAll stops every watched container except the one already exited.
func (e *Engine) stopAll(ctx context.Context, items []*watched, except string) {
	for _, w := range items {
		if w.name == except {
			continue
		}
		_, _ = e.Runner.Run(ctx, "stop", w.name)
	}
}

// stopAllAndCode stops every watched container and returns the exit code of the
// container belonging to the named service.
func (e *Engine) stopAllAndCode(ctx context.Context, items []*watched, service string) int {
	e.stopAll(ctx, items, "")
	for _, w := range items {
		if w.svc.Name != service {
			continue
		}
		if res, err := e.Runner.Run(ctx, "inspect", w.name); err == nil {
			if _, code, done := inspectExit(res.Stdout); done {
				return code
			}
		}
	}
	return 0
}
