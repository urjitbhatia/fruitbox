package engine

import (
	"context"
	"sync"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/urjitbhatia/fruitbox/internal/translate"
)

// LogOptions controls log retrieval.
type LogOptions struct {
	// Follow streams new output (--follow).
	Follow bool
	// Tail is the number of lines from the end ("all" or "" for everything).
	Tail string
	// Index selects a single replica (1-based); 0 means all replicas.
	Index int
	// NoPrefix omits the per-service line prefix (--no-log-prefix).
	NoPrefix bool
	// NoColor disables ANSI color in the prefix (--no-color).
	NoColor bool
	// Timestamps prepends an RFC3339 timestamp to each line (--timestamps).
	Timestamps bool
}

// logTarget is one container whose logs should be streamed.
type logTarget struct {
	service   string
	container string
}

// Logs streams logs for the named services (or all services when none are
// given), multiplexing every container's output concurrently with a colored,
// per-service prefix (like docker compose).
func (e *Engine) Logs(ctx context.Context, p *types.Project, services []string, opts LogOptions) error {
	names := services
	if len(names) == 0 {
		names = p.ServiceNames()
	}

	var targets []logTarget
	width := 0
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
			label := containerLogLabel(p, svc, n)
			if len(label) > width {
				width = len(label)
			}
			targets = append(targets, logTarget{service: label, container: cname})
		}
	}

	return e.streamLogs(ctx, targets, width, logFormat{
		noPrefix:   opts.NoPrefix,
		noColor:    opts.NoColor,
		timestamps: opts.Timestamps,
	}, opts.Follow, opts.Tail)
}

// streamLogs runs `container logs` for each target concurrently, prefixing each
// line. It returns once all log commands finish (or the context is cancelled).
func (e *Engine) streamLogs(ctx context.Context, targets []logTarget, width int, f logFormat, follow bool, tail string) error {
	mu := &sync.Mutex{}
	out := e.writer()
	var wg sync.WaitGroup
	for i, t := range targets {
		lw := &lineWriter{
			mu:     mu,
			w:      out,
			prefix: buildPrefix(t.service, width, i, f),
			format: f,
			now:    e.now,
		}
		args := []string{"logs"}
		if follow {
			args = append(args, "--follow")
		}
		if tail != "" && tail != "all" {
			args = append(args, "-n", tail)
		}
		args = append(args, t.container)

		wg.Add(1)
		go func(args []string, lw *lineWriter) {
			defer wg.Done()
			_ = e.Runner.RunWithOutput(ctx, lw, lw, args...)
			lw.flush()
		}(args, lw)
	}
	wg.Wait()
	return nil
}

// containerLogLabel is the per-line label for a service replica. Single-replica
// services use the service name; scaled services include the replica number.
func containerLogLabel(p *types.Project, svc types.ServiceConfig, n int) string {
	if scaleOf(svc) > 1 {
		return svc.Name + "-" + itoa(n)
	}
	return svc.Name
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	// rare for replica counts; fall back to fmt-free conversion
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
