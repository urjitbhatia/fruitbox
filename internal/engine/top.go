package engine

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
)

// Top lists the running processes of each of the project's containers by
// running `ps` inside them via `container exec`. Apple's runtime has no native
// `top`, so fruitbox executes ps in-container, matching `docker compose top`.
func (e *Engine) Top(ctx context.Context, p *types.Project, names []string, psArgs []string) error {
	if len(psArgs) == 0 {
		psArgs = []string{"-ef"}
	}
	refs, err := e.containerNames(p, names)
	if err != nil {
		return err
	}
	for _, r := range refs {
		execArgs := append([]string{"exec", r.Container, "ps"}, psArgs...)
		res, err := e.Runner.Run(ctx, execArgs...)
		if err != nil {
			// Container not running or ps missing; report and continue.
			e.logf("%s: %v", r.Container, err)
			continue
		}
		out := e.writer()
		fmt.Fprintf(out, "%s\n%s", r.Container, res.Stdout)
		if !strings.HasSuffix(res.Stdout, "\n") {
			fmt.Fprintln(out)
		}
		fmt.Fprintln(out)
	}
	return nil
}

// Pause suspends the named services' containers by sending SIGSTOP, since the
// runtime exposes no freezer; Unpause resumes them with SIGCONT.
func (e *Engine) Pause(ctx context.Context, p *types.Project, names []string) error {
	return e.signalAll(ctx, p, names, "SIGSTOP", "Pausing")
}

// Unpause resumes paused containers by sending SIGCONT.
func (e *Engine) Unpause(ctx context.Context, p *types.Project, names []string) error {
	return e.signalAll(ctx, p, names, "SIGCONT", "Unpausing")
}

func (e *Engine) signalAll(ctx context.Context, p *types.Project, names []string, signal, verb string) error {
	refs, err := e.containerNames(p, names)
	if err != nil {
		return err
	}
	for _, r := range refs {
		e.logf("%s %s", verb, r.Container)
		if _, err := e.Runner.Run(ctx, "kill", "--signal", signal, r.Container); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) writer() io.Writer {
	if e.Out != nil {
		return e.Out
	}
	return io.Discard
}
