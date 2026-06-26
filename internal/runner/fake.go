package runner

import (
	"context"
	"io"
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

	// responses are matched against the joined args; the first match wins.
	responses []*fakeResponse
}

type fakeResponse struct {
	match   string
	results []Result
	errs    []error
	idx     int
}

// next returns the response for the current invocation, advancing through a
// registered sequence (the final entry repeats once exhausted).
func (r *fakeResponse) next() (Result, error) {
	i := r.idx
	if i >= len(r.results) {
		i = len(r.results) - 1
	}
	r.idx++
	var err error
	if i < len(r.errs) {
		err = r.errs[i]
	}
	return r.results[i], err
}

// On registers a canned response for any call whose joined args contain match.
func (f *Fake) On(match string, result Result, err error) *Fake {
	f.responses = append(f.responses, &fakeResponse{
		match:   match,
		results: []Result{result},
		errs:    []error{err},
	})
	return f
}

// OnSequence registers an ordered sequence of responses for matching calls.
// Successive matches return successive results; the last result repeats
// thereafter. Useful for simulating "unhealthy then healthy" transitions.
func (f *Fake) OnSequence(match string, results ...Result) *Fake {
	f.responses = append(f.responses, &fakeResponse{match: match, results: results})
	return f
}

func (f *Fake) lookup(joined string) (Result, error, bool) {
	for _, r := range f.responses {
		if strings.Contains(joined, r.match) {
			res, err := r.next()
			return res, err, true
		}
	}
	return Result{}, nil, false
}

// Run implements Runner, recording the call and returning any canned response.
func (f *Fake) Run(_ context.Context, args ...string) (Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, Call{Args: append([]string(nil), args...)})
	if res, err, ok := f.lookup(strings.Join(args, " ")); ok {
		return res, err
	}
	return Result{}, nil
}

// RunInteractive implements Runner, recording the call.
func (f *Fake) RunInteractive(_ context.Context, args ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, Call{Args: append([]string(nil), args...), Interactive: true})
	if _, err, ok := f.lookup(strings.Join(args, " ")); ok {
		return err
	}
	return nil
}

// RunWithOutput implements Runner, recording the call and writing any canned
// stdout to the provided writer (so log-multiplexing can be tested).
func (f *Fake) RunWithOutput(_ context.Context, stdout, stderr io.Writer, args ...string) error {
	f.mu.Lock()
	res, err, ok := f.recordAndLookup(args)
	f.mu.Unlock()
	if ok {
		if res.Stdout != "" && stdout != nil {
			_, _ = io.WriteString(stdout, res.Stdout)
		}
		if res.Stderr != "" && stderr != nil {
			_, _ = io.WriteString(stderr, res.Stderr)
		}
		return err
	}
	return nil
}

// recordAndLookup appends a call and returns its canned response (caller holds
// the lock).
func (f *Fake) recordAndLookup(args []string) (Result, error, bool) {
	f.Calls = append(f.Calls, Call{Args: append([]string(nil), args...)})
	return f.lookup(strings.Join(args, " "))
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

// CountMatching returns how many recorded calls contain the given substring.
func (f *Fake) CountMatching(substr string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.Calls {
		if strings.Contains(strings.Join(c.Args, " "), substr) {
			n++
		}
	}
	return n
}
