// Package cli holds what the umpire commands share at their edges: the interruptible context a
// command runs under, the one-line report, and the flattening of a joined error into lines.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"
)

// Interruptible bounds a command's work by the caller's timeout and cancels it on SIGINT.
// Teardown runs on its own context afterwards, so an interrupted command still removes what it
// created.
func Interruptible(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	notified, stopNotify := signal.NotifyContext(parent, os.Interrupt)
	ctx, cancelTimeout := context.WithTimeout(notified, timeout)
	return ctx, func() {
		cancelTimeout()
		stopNotify()
	}
}

// WriteLine reports one line. A report the caller cannot receive is not a failure worth changing
// the exit code for, so the write error is deliberately dropped.
func WriteLine(destination io.Writer, format string, arguments ...any) {
	_, _ = fmt.Fprintf(destination, format+"\n", arguments...)
}

// Flatten renders a joined error as one line per leaf, in order.
func Flatten(err error) []string {
	var joined interface{ Unwrap() []error }
	if errors.As(err, &joined) {
		var lines []string
		for _, nested := range joined.Unwrap() {
			lines = append(lines, Flatten(nested)...)
		}
		return lines
	}
	return []string{err.Error()}
}
