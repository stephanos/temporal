package runner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestWaitForProgressStart(t *testing.T) {
	for _, test := range []struct {
		name      string
		started   bool
		completed bool
		deadline  bool
		wantError string
	}{
		{name: "started", started: true},
		{name: "nil completion", completed: true, wantError: "runner completed before target execution started: <nil>"},
		{name: "deadline", deadline: true, wantError: "target execution did not start before startup deadline"},
	} {
		t.Run(test.name, func(t *testing.T) {
			started := make(chan struct{})
			start := sync.OnceFunc(func() { close(started) })
			t.Cleanup(start)
			completed := make(chan error, 1)
			deadline := make(chan time.Time, 1)
			if test.started {
				start()
			}
			if test.completed {
				completed <- nil
			}
			if test.deadline {
				deadline <- time.Time{}
			}
			watchdog := time.AfterFunc(time.Second, start)
			defer watchdog.Stop()
			err := waitForProgressStart(started, completed, deadline)
			if test.wantError == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || err.Error() != test.wantError {
				t.Fatalf("startup error = %v, want %q", err, test.wantError)
			}
		})
	}
}

func TestWaitForProgressStartObservesPreparationFailure(t *testing.T) {
	executor := &progressGatedExecutor{started: make(chan struct{}), release: make(chan struct{})}
	config, dependencies := testConfig(t, errorPreparer{err: errors.New("progress-start preparation sentinel")}, executor, "1", PolicyAll, 1)
	t.Cleanup(sync.OnceFunc(func() { close(executor.release) }))
	start := sync.OnceFunc(func() { close(executor.started) })
	t.Cleanup(start)
	completed := make(chan error, 1)
	go func() {
		_, err := exploreWith(context.Background(), config, dependencies)
		completed <- err
	}()
	watchdog := time.AfterFunc(time.Second, start)
	defer watchdog.Stop()
	deadline := time.NewTimer(config.OverallTimeout + config.TerminateGrace)
	defer deadline.Stop()
	err := waitForProgressStart(executor.started, completed, deadline.C)
	if err == nil || !strings.Contains(err.Error(), "progress-start preparation sentinel") {
		t.Fatalf("startup error = %v, want preparation sentinel", err)
	}
	select {
	case <-executor.started:
		t.Fatal("executor started after preparation failure")
	default:
	}
}

func waitForProgressStart(started <-chan struct{}, completed <-chan error, deadline <-chan time.Time) error {
	select {
	case <-started:
		return nil
	case err := <-completed:
		return fmt.Errorf("runner completed before target execution started: %v", err)
	case <-deadline:
		return errors.New("target execution did not start before startup deadline")
	}
}
