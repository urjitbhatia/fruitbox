package runner

import (
	"context"
	"strings"
	"sync"
)

// Call records a single invocation made against a Fake runner.
type Call struct {
	Args        []string
	Interactive bool
}

// Fake is an in-memory Runner for tests. It records every invocation and lets
// the test program canned responses or errors keyed by an argument substring.
type Fake struct {
	mu    sync.Mutex
	Calls []Call

	// Responses maps an argument-substring match to a canned Result. The first
	// matching entry (in insertion order) wins.
	responses []fakeResponse
}

type fakeResponse struct {
	match  string
	result Result
	err    error
}

// On registers a canned response for any Run whose joined args contain match.
func (f *Fake) On(match string, result Result, err error) *Fake {
	f.responses = append(f.responses, fakeResponse{match: match, result: result, err: err})
	return f
}

// Run implements Runner, recording the call and returning any canned response.
func (f *Fake) Run(_ context.Context, args ...string) (Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, Call{Args: append([]string(nil), args...)})
	joined := strings.Join(args, " ")
	for _, r := range f.responses {
		if strings.Contains(joined, r.match) {
			return r.result, r.err
		}
	}
	return Result{}, nil
}

// RunInteractive implements Runner, recording the call.
func (f *Fake) RunInteractive(_ context.Context, args ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, Call{Args: append([]string(nil), args...), Interactive: true})
	joined := strings.Join(args, " ")
	for _, r := range f.responses {
		if strings.Contains(joined, r.match) {
			return r.err
		}
	}
	return nil
}

// CommandArgs returns the recorded calls as joined argument strings, for terse
// assertions in tests.
func (f *Fake) CommandArgs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.Calls))
	for i, c := range f.Calls {
		out[i] = strings.Join(c.Args, " ")
	}
	return out
}
