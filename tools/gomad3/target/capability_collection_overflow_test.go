package target

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
	"go.temporal.io/server/tools/gomad3/target/internal/gocommand"
)

func TestCollectCapabilityListingPreservesBoundedOutputError(t *testing.T) {
	for _, test := range []struct {
		name   string
		stdout int
		stderr int
		exit   bool
	}{
		{name: "stdout exact capacity", stdout: maximumCapabilityReviewOutputBytes},
		{name: "stderr exact capacity", stderr: maximumCapabilityReviewOutputBytes},
		{name: "stdout limit plus one", stdout: maximumCapabilityReviewOutputBytes + 1},
		{name: "stderr limit plus one", stderr: maximumCapabilityReviewOutputBytes + 1},
		{name: "both streams overflow", stdout: maximumCapabilityReviewOutputBytes + 1, stderr: maximumCapabilityReviewOutputBytes + 1},
		{name: "stdout overflow before failed process", stdout: maximumCapabilityReviewOutputBytes + 1, exit: true},
		{name: "stderr overflow before failed process", stderr: maximumCapabilityReviewOutputBytes + 1, exit: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := "printf '%s' '{}'\n"
			if test.stdout > 0 {
				body += fmt.Sprintf("head -c %d /dev/zero | tr '\\000' ' '\n", test.stdout-2)
			}
			if test.stderr > 0 {
				body += fmt.Sprintf("head -c %d /dev/zero >&2\n", test.stderr)
			}
			if test.exit {
				body += "exit 7\n"
			}
			command := capabilityListingOverflowCommand(t, body)
			var observed hostexec.Result
			runner := gocommand.New(func(ctx context.Context, request hostexec.Request) (hostexec.Result, error) {
				var err error
				observed, err = hostexec.Run(ctx, request)
				return observed, err
			})
			packages, overlay, err := collectCapabilityListing(t.Context(), command, Spec{}, nil, t.TempDir(), ".", runner)
			if test.stdout <= maximumCapabilityReviewOutputBytes && test.stderr <= maximumCapabilityReviewOutputBytes {
				if err != nil || len(packages) != 1 || overlay != nil || observed.Stdout.TotalBytes != uint64(max(test.stdout, 2)) || observed.Stderr.TotalBytes != uint64(test.stderr) {
					t.Fatalf("exact-capacity listing = %#v, %#v, %v; streams = %#v, %#v", packages, overlay, err, observed.Stdout, observed.Stderr)
				}
				return
			}
			var overflow *gocommand.OverflowError
			cause := errors.Unwrap(err)
			if err == nil || err.Error() != "inspect target capability closure: target capability closure output exceeds 67108864 bytes" || fmt.Sprintf("%T", cause) != "*errors.errorString" || errors.As(err, &overflow) || IsInvalidCapabilityReview(err) || packages != nil || overlay != nil {
				t.Fatalf("bounded listing = %#v, %#v, %T %v; cause = %T %v", packages, overlay, err, err, cause, cause)
			}
			if !observed.GroupGone || observed.Stdout.Truncated != (test.stdout > maximumCapabilityReviewOutputBytes) || observed.Stderr.Truncated != (test.stderr > maximumCapabilityReviewOutputBytes) {
				t.Fatalf("process/stream outcome = %#v", observed)
			}
			if test.exit {
				exit, ok := observed.CommandError.(*exec.ExitError)
				if !ok || exit.ProcessState == nil || exit.ExitCode() != 7 {
					t.Fatalf("failed process outcome = %T %v", observed.CommandError, observed.CommandError)
				}
			}
		})
	}
}

func TestCollectCapabilityListingPreservesInfrastructureContainingOverflow(t *testing.T) {
	infrastructure := errors.New("command cleanup failed")
	for _, operation := range []error{
		infrastructure,
		fmt.Errorf("cleanup: %w", &gocommand.OverflowError{Stream: "stdout", Limit: maximumCapabilityReviewOutputBytes}),
		errors.Join(infrastructure, &gocommand.OverflowError{Stream: "stderr", Limit: maximumCapabilityReviewOutputBytes}),
	} {
		runner := gocommand.New(func(context.Context, hostexec.Request) (hostexec.Result, error) {
			return hostexec.Result{Stdout: hostexec.Output{RawBytes: []byte("{}"), Truncated: true}}, operation
		})
		packages, overlay, err := collectCapabilityListing(t.Context(), "go", Spec{}, nil, t.TempDir(), ".", runner)
		if err == nil || errors.Unwrap(err) != operation || err.Error() != "inspect target capability closure: "+operation.Error() || packages != nil || overlay != nil {
			t.Fatalf("infrastructure listing = %#v, %#v, %T %v", packages, overlay, err, err)
		}
	}
}

func TestCollectCapabilityListingPreservesActualCancellationBeforeOverflow(t *testing.T) {
	ack := filepath.Join(t.TempDir(), "ack")
	command := capabilityListingOverflowCommand(t, fmt.Sprintf("head -c %d /dev/zero\nprintf ready > %q\nexec /bin/sleep 30\n", maximumCapabilityReviewOutputBytes+1, ack))
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				data, err := os.ReadFile(ack)
				if err == nil && string(data) == "ready" {
					cancel()
					return
				}
				if err != nil && !errors.Is(err, os.ErrNotExist) {
					t.Errorf("read command acknowledgement: %v", err)
					cancel()
					return
				}
			}
		}
	}()
	var observed hostexec.Result
	runner := gocommand.New(func(ctx context.Context, request hostexec.Request) (hostexec.Result, error) {
		var err error
		observed, err = hostexec.Run(ctx, request)
		return observed, err
	})
	packages, overlay, err := collectCapabilityListing(ctx, command, Spec{}, nil, t.TempDir(), ".", runner)
	<-finished
	if !errors.Is(err, context.Canceled) || errors.Unwrap(err) != context.Canceled || packages != nil || overlay != nil || !observed.Cancelled || !observed.Stdout.Truncated || !observed.GroupGone {
		t.Fatalf("cancelled overflowing listing = %#v, %#v, %v; process = %#v", packages, overlay, err, observed)
	}
	exit, ok := observed.CommandError.(*exec.ExitError)
	if !ok || exit.ProcessState == nil || exit.ExitCode() != -1 {
		t.Fatalf("cancelled process outcome = %T %v", observed.CommandError, observed.CommandError)
	}
}

func capabilityListingOverflowCommand(t *testing.T, body string) string {
	t.Helper()
	command := filepath.Join(t.TempDir(), "go")
	if err := os.WriteFile(command, []byte("#!/bin/sh\nset -eu\n"+body), 0o700); err != nil {
		t.Fatal(err)
	}
	return command
}
