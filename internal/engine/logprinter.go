package engine

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"
)

// ansiColors is the palette fruitbox cycles through for per-service log
// prefixes, matching the spirit of docker compose's colored output.
var ansiColors = []string{
	"\033[36m", // cyan
	"\033[33m", // yellow
	"\033[32m", // green
	"\033[35m", // magenta
	"\033[34m", // blue
	"\033[31m", // red
	"\033[37m", // white
}

const ansiReset = "\033[0m"

// logFormat captures the formatting options shared by `logs` and foreground
// `up` log streaming.
type logFormat struct {
	noPrefix   bool
	noColor    bool
	timestamps bool
}

// buildPrefix returns the rendered, padded (and optionally colored) prefix for
// a service's log lines, e.g. "web    | ". Returns "" when prefixes are off.
func buildPrefix(service string, width, colorIdx int, f logFormat) string {
	if f.noPrefix {
		return ""
	}
	name := fmt.Sprintf("%-*s", width, service)
	if !f.noColor {
		name = ansiColors[colorIdx%len(ansiColors)] + name + ansiReset
	}
	return name + " | "
}

// lineWriter buffers bytes and emits complete lines to a shared writer with a
// per-stream prefix, locking so concurrent streams never interleave a line.
type lineWriter struct {
	mu     *sync.Mutex
	w      io.Writer
	prefix string
	format logFormat
	now    func() time.Time
	buf    []byte
}

func (lw *lineWriter) Write(p []byte) (int, error) {
	lw.buf = append(lw.buf, p...)
	for {
		i := bytes.IndexByte(lw.buf, '\n')
		if i < 0 {
			break
		}
		lw.emit(lw.buf[:i])
		lw.buf = lw.buf[i+1:]
	}
	return len(p), nil
}

// emit writes one prefixed (and optionally timestamped) line under the lock.
func (lw *lineWriter) emit(line []byte) {
	lw.mu.Lock()
	defer lw.mu.Unlock()
	var b bytes.Buffer
	b.WriteString(lw.prefix)
	if lw.format.timestamps {
		b.WriteString(lw.now().Format(time.RFC3339))
		b.WriteByte(' ')
	}
	b.Write(line)
	b.WriteByte('\n')
	_, _ = lw.w.Write(b.Bytes())
}

// flush emits any buffered partial line (no trailing newline seen).
func (lw *lineWriter) flush() {
	if len(lw.buf) > 0 {
		lw.emit(lw.buf)
		lw.buf = nil
	}
}
