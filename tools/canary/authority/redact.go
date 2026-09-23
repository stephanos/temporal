package authority

import (
	"bytes"
	"cmp"
	"io"
	"log/slog"
	"net"
	"slices"
	"strings"
	"sync"

	sdklog "go.temporal.io/sdk/log"
)

// Redacted is what every credential and coordinate becomes in written text.
const Redacted = "[redacted]"

const (
	// minLineSecret is the shortest line of a multi-line value redacted on its own.
	minLineSecret = 16
	// maxPending bounds what a Writer holds while it waits for a newline. Past it the held text is
	// written, redacted, so a writer that never ends a line cannot grow without bound.
	maxPending = 64 << 10
)

// Redactor removes every credential and raw coordinate it was built with from text. A multi-line
// value, a PEM block, is also removed line by line, so a value re-encoded with escaped newlines or
// split across lines is still removed; a `host:port` value is also removed by its host.
type Redactor struct {
	replacer *strings.Replacer
}

// NewRedactor redacts each non-empty value, the longest first so no value is left half-removed by
// a shorter one it contains.
func NewRedactor(values ...string) *Redactor {
	seen := map[string]bool{}
	var secrets []string
	add := func(value string) {
		if value != "" && !seen[value] {
			seen[value] = true
			secrets = append(secrets, value)
		}
	}
	for _, value := range values {
		add(value)
		if strings.Contains(value, "\n") {
			for line := range strings.SplitSeq(value, "\n") {
				line = strings.TrimSpace(line)
				// A PEM armour line names the block's type, which is not a secret, and would redact
				// every other PEM block's armour with it; a short last body line alone is no secret
				// and would redact unrelated text.
				if !strings.HasPrefix(line, "-----") && len(line) >= minLineSecret {
					add(line)
				}
			}
		}
		if host, _, err := net.SplitHostPort(value); err == nil {
			add(host)
		}
	}
	slices.SortStableFunc(secrets, func(a, b string) int { return cmp.Compare(len(b), len(a)) })
	pairs := make([]string, 0, 2*len(secrets))
	for _, secret := range secrets {
		pairs = append(pairs, secret, Redacted)
	}
	return &Redactor{replacer: strings.NewReplacer(pairs...)}
}

// Redact returns text with every value removed.
func (r *Redactor) Redact(text string) string {
	return r.replacer.Replace(text)
}

// Writer redacts whole lines before writing them to w, so a value split across two writes is still
// removed. Close writes a last unterminated line.
func (r *Redactor) Writer(w io.Writer) *Writer {
	return &Writer{redactor: r, out: w}
}

// Logger is an SDK logger that writes warnings and errors through the Redactor to w, so an SDK
// client never writes a credential or a coordinate.
func (r *Redactor) Logger(w io.Writer) sdklog.Logger {
	return sdklog.NewStructuredLogger(slog.New(slog.NewTextHandler(r.Writer(w), &slog.HandlerOptions{Level: slog.LevelWarn})))
}

// Writer is a line-buffered redacting writer.
type Writer struct {
	mu       sync.Mutex
	redactor *Redactor
	out      io.Writer
	pending  []byte
}

// Write accepts p whole and writes every line it completes, redacted.
func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pending = append(w.pending, p...)
	end := bytes.LastIndexByte(w.pending, '\n')
	if end < 0 && len(w.pending) <= maxPending {
		return len(p), nil
	}
	if end < 0 {
		end = len(w.pending) - 1
	}
	lines := string(w.pending[:end+1])
	w.pending = append(w.pending[:0], w.pending[end+1:]...)
	if _, err := io.WriteString(w.out, w.redactor.Redact(lines)); err != nil {
		return len(p), err
	}
	return len(p), nil
}

// Close writes the last unterminated line, redacted.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.pending) == 0 {
		return nil
	}
	rest := string(w.pending)
	w.pending = nil
	_, err := io.WriteString(w.out, w.redactor.Redact(rest))
	return err
}
