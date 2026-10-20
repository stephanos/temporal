package sweep

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func sourceIdentityFixture(t *testing.T) (*fixture, *Sweeper) {
	t.Helper()
	f := newFixture(t)
	if err := os.MkdirAll(f.options.Work, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "--quiet"},
		{"add", "--all"},
		{"-c", "user.name=Test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "fixture"},
	} {
		completed := (HostExecutor{}).Exec(t.Context(), Command{Path: "git", Args: append([]string{"-C", f.options.Checkout}, args...), Dir: f.options.Work, Env: f.options.Environ, Timeout: time.Minute})
		if completed.Err != nil || completed.Exit != 0 {
			t.Fatalf("git %v: %v exit=%d stderr=%s", args, completed.Err, completed.Exit, completed.Stderr)
		}
	}
	sweeper := f.sweeper()
	sweeper.Executor = deadlineExecutor(func(ctx context.Context, command Command) Completed {
		if command.Path == "git" {
			return (HostExecutor{}).Exec(ctx, command)
		}
		return f.Exec(ctx, command)
	})
	return f, sweeper
}

func TestSweepRefusesChangedSourceBeforeResumingRows(t *testing.T) {
	t.Parallel()
	for _, change := range []struct {
		name, path, content string
	}{
		{"dirty-A", "tests/testcore/core.go", "package testcore\n// dirty A\n"},
		{"dirty-B", "tests/testcore/core.go", "package testcore\n// dirty B\n"},
		{"untracked-test", "tests/alpha/new_test.go", "package alpha\n"},
	} {
		t.Run(change.name, func(t *testing.T) {
			t.Parallel()
			f, sweeper := sourceIdentityFixture(t)
			if _, err := sweeper.Run(t.Context()); err != nil {
				t.Fatal(err)
			}
			journal, err := os.ReadFile(JournalPath(f.options.Out))
			if err != nil {
				t.Fatal(err)
			}
			before := readSidecar(t, f.options.Out)
			if before.Candidate.GitDirty || before.ByClass[ClassPass] != 3 {
				t.Fatalf("clean baseline not classified: %+v", before)
			}
			explored := len(f.explores)
			if err := os.WriteFile(filepath.Join(f.options.Checkout, filepath.FromSlash(change.path)), []byte(change.content), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := sweeper.Run(t.Context()); !errors.Is(err, ErrRefused) {
				t.Fatalf("changed source reused prior candidate: %v", err)
			}
			after, err := os.ReadFile(JournalPath(f.options.Out))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(journal, after) || len(f.explores) != explored {
				t.Fatal("refused source appended or executed rows under the prior candidate")
			}
		})
	}
}

func TestSweepAllowsOnlyLiteralOutputDirectoryChanges(t *testing.T) {
	t.Parallel()
	f, sweeper := sourceIdentityFixture(t)
	f.options.Out = filepath.Join(f.options.Checkout, "space [abc]*?")
	sweeper.Options = f.options
	for range 2 {
		if _, err := sweeper.Run(t.Context()); err != nil {
			t.Fatalf("output directory refused: %v", err)
		}
	}
	if sidecar := readSidecar(t, f.options.Out); sidecar.Candidate.GitDirty || sidecar.ByClass[ClassPass] != 3 {
		t.Fatalf("output changed candidate admission: %+v", sidecar)
	}
	path := filepath.Join(f.options.Checkout, "space ax", "untracked_test.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := sweeper.Run(t.Context()); !errors.Is(err, ErrRefused) {
		t.Fatalf("output exclusion matched a different directory: %v", err)
	}
}

func TestSweepSourceStatusFailureStopsAdmission(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	sweeper := f.sweeper()
	sweeper.Executor = deadlineExecutor(func(ctx context.Context, command Command) Completed {
		if command.Path == "git" && slices.Contains(command.Args, "status") {
			return Completed{Exit: -1, Err: errors.New("status unavailable")}
		}
		return f.Exec(ctx, command)
	})
	if _, err := sweeper.Run(t.Context()); err == nil {
		t.Fatal("source status failure was accepted")
	}
	if _, err := os.Stat(JournalPath(f.options.Out)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("source status failure published journal: %v", err)
	}
}

func TestSweepOutputDirectoryDoesNotHideSourceChanges(t *testing.T) {
	t.Parallel()
	f, sweeper := sourceIdentityFixture(t)
	f.options.Out = filepath.Join(f.options.Checkout, "tests", "alpha")
	sweeper.Options = f.options
	if _, err := sweeper.Run(t.Context()); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(f.options.Out, "alpha_test.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(source, []byte("\n// changed source\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := sweeper.Run(t.Context()); !errors.Is(err, ErrRefused) {
		t.Fatalf("output directory hid a changed test input: %v", err)
	}
}
