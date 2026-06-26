package engine

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/urjitbhatia/fruitbox/internal/runner"
)

func TestBuildPrefix(t *testing.T) {
	// Colored, padded to width 6.
	p := buildPrefix("web", 6, 0, logFormat{})
	if !strings.Contains(p, "web   ") || !strings.Contains(p, "| ") || !strings.Contains(p, "\033[") {
		t.Errorf("colored prefix unexpected: %q", p)
	}
	// No color.
	if p := buildPrefix("web", 3, 0, logFormat{noColor: true}); p != "web | " {
		t.Errorf("no-color prefix = %q, want 'web | '", p)
	}
	// No prefix.
	if p := buildPrefix("web", 3, 0, logFormat{noPrefix: true}); p != "" {
		t.Errorf("no-prefix should be empty, got %q", p)
	}
}

func TestLineWriterPrefixesAndBuffers(t *testing.T) {
	var out strings.Builder
	mu := &sync.Mutex{}
	lw := &lineWriter{mu: mu, w: &out, prefix: "web | ", now: func() time.Time { return time.Unix(0, 0) }}
	// Partial then completing write.
	lw.Write([]byte("hello "))
	lw.Write([]byte("world\nsecond line\npartial"))
	lw.flush()
	got := out.String()
	want := "web | hello world\nweb | second line\nweb | partial\n"
	if got != want {
		t.Errorf("lineWriter output =\n%q\nwant\n%q", got, want)
	}
}

func TestLineWriterTimestamps(t *testing.T) {
	var out strings.Builder
	lw := &lineWriter{
		mu:     &sync.Mutex{},
		w:      &out,
		prefix: "web | ",
		format: logFormat{timestamps: true},
		now:    func() time.Time { return time.Unix(0, 0).UTC() },
	}
	lw.Write([]byte("msg\n"))
	if !strings.Contains(out.String(), "1970-01-01T00:00:00Z msg") {
		t.Errorf("timestamp not rendered: %q", out.String())
	}
}

func TestLogsMultiplexesWithPrefixes(t *testing.T) {
	proj := load(t, "basic") // db + web
	fake := &runner.Fake{}
	fake.On("logs basic-web-1", runner.Result{Stdout: "web log A\nweb log B\n"}, nil)
	fake.On("logs basic-db-1", runner.Result{Stdout: "db log A\n"}, nil)
	var out strings.Builder
	e := New(fake, &out)

	if err := e.Logs(context.Background(), proj, nil, LogOptions{NoColor: true}); err != nil {
		t.Fatalf("Logs: %v", err)
	}
	s := out.String()
	// Service names are padded to a common width for column alignment.
	for _, want := range []string{"web | web log A", "web | web log B", "db  | db log A"} {
		if !strings.Contains(s, want) {
			t.Errorf("multiplexed logs missing %q in:\n%s", want, s)
		}
	}
}

func TestLogsNoPrefix(t *testing.T) {
	proj := load(t, "basic")
	fake := &runner.Fake{}
	fake.On("logs basic-web-1", runner.Result{Stdout: "plain line\n"}, nil)
	var out strings.Builder
	e := New(fake, &out)
	if err := e.Logs(context.Background(), proj, []string{"web"}, LogOptions{NoPrefix: true}); err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if strings.Contains(out.String(), "|") || !strings.Contains(out.String(), "plain line") {
		t.Errorf("--no-log-prefix should drop prefix, got: %q", out.String())
	}
}
