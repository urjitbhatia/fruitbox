package engine

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
)

// Event is a synthesized container lifecycle event.
type Event struct {
	Name   string
	Action string // create, start, die, destroy
}

// containerState maps a container name to a coarse state: "running" or
// "stopped". Absence from the map means the container does not exist.
type containerState map[string]string

// diffEvents computes the lifecycle events implied by a transition from prev to
// cur container state. Events are returned sorted by container name for
// determinism. This is the pure core of the events stream.
func diffEvents(prev, cur containerState) []Event {
	var events []Event
	names := map[string]bool{}
	for n := range prev {
		names[n] = true
	}
	for n := range cur {
		names[n] = true
	}
	ordered := make([]string, 0, len(names))
	for n := range names {
		ordered = append(ordered, n)
	}
	sort.Strings(ordered)

	for _, n := range ordered {
		was, existed := prev[n]
		now, exists := cur[n]
		switch {
		case !existed && exists:
			events = append(events, Event{Name: n, Action: "create"})
			if now == "running" {
				events = append(events, Event{Name: n, Action: "start"})
			}
		case existed && !exists:
			events = append(events, Event{Name: n, Action: "destroy"})
		case existed && exists && was != now:
			if now == "running" {
				events = append(events, Event{Name: n, Action: "start"})
			} else {
				events = append(events, Event{Name: n, Action: "die"})
			}
		}
	}
	return events
}

// projectState reads the current container state for a project from the runtime.
func (e *Engine) projectState(ctx context.Context, projectName string) (containerState, error) {
	res, err := e.Runner.Run(ctx, "list", "--all", "--format", "json")
	if err != nil {
		return nil, err
	}
	state := containerState{}
	for _, c := range parseContainerList(res.Stdout) {
		if c.project != projectName {
			continue
		}
		st := "stopped"
		if c.status == "running" {
			st = "running"
		}
		state[c.Name] = st
	}
	return state, nil
}

// Events streams synthesized container lifecycle events for the project until
// the context is cancelled. maxPolls bounds the number of polls (0 = infinite);
// it exists primarily so tests terminate deterministically.
func (e *Engine) Events(ctx context.Context, p *types.Project, maxPolls int) error {
	var prev containerState
	for i := 0; maxPolls == 0 || i < maxPolls; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		cur, err := e.projectState(ctx, p.Name)
		if err != nil {
			return err
		}
		if prev != nil {
			for _, ev := range diffEvents(prev, cur) {
				fmt.Fprintf(e.writer(), "%s container %s %s\n", e.now().Format(time.RFC3339), ev.Action, ev.Name)
			}
		}
		prev = cur
		if maxPolls != 0 && i == maxPolls-1 {
			break
		}
		if err := e.sleep(ctx, time.Second); err != nil {
			return err
		}
	}
	return nil
}
