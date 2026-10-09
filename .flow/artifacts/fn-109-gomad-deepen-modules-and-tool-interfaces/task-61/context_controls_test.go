package toolchain

import (
	"context"
	"testing"
	"time"
)

func TestBuildLockWaitContextControls(t *testing.T) {
	type key struct{}
	deadline := time.Now().Add(time.Minute)
	parent, cancel := context.WithDeadline(context.WithValue(context.Background(), key{}, "value"), deadline)
	defer cancel()
	for _, test := range []struct {
		name   string
		parent context.Context
	}{
		{name: "background", parent: context.Background()},
		{name: "cancelable", parent: parent},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := &buildLockWaitContext{Context: test.parent, entered: make(chan struct{})}
			calls := make(chan bool, 4)
			for range 4 {
				go func() { calls <- ctx.Done() == test.parent.Done() }()
			}
			timer := time.NewTimer(5 * time.Second)
			defer timer.Stop()
			for range 4 {
				select {
				case same := <-calls:
					if !same {
						t.Fatal("observer replaced the underlying Done channel")
					}
				case <-timer.C:
					t.Fatal("concurrent Done calls did not complete")
				}
			}
			select {
			case <-ctx.entered:
			default:
				t.Fatal("Done calls did not notify the observer")
			}
			if ctx.Err() != nil {
				t.Fatalf("observer changed an active context error: %v", ctx.Err())
			}
			gotDeadline, gotOK := ctx.Deadline()
			if test.name == "background" {
				if gotOK || ctx.Value(key{}) != nil || ctx.Done() != nil {
					t.Fatal("observer changed background context semantics")
				}
			} else {
				if !gotOK || !gotDeadline.Equal(deadline) || ctx.Value(key{}) != "value" {
					t.Fatal("observer changed deadline or value semantics")
				}
				cancel()
				select {
				case <-ctx.Done():
				case <-timer.C:
					t.Fatal("observer hid cancellation")
				}
				if ctx.Err() != context.Canceled {
					t.Fatalf("observer changed cancellation identity: %v", ctx.Err())
				}
			}
		})
	}
}
