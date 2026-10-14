package namespacediagnostic

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNamespacePendingStack(t *testing.T) {
	fmt.Println("diagnostic only: one runtime.Stack(all=true) sample scheduled 15s after first Describe entry, maximum 1MiB; timer/goroutine, stop-the-world, allocation and output perturb execution; timer may be delayed; does not reproduce unperturbed trace or establish fairness")
	runNamespaceRPCCause(t, func(ctx context.Context) func() { return namespacePendingSnapshot(t, ctx) })
}

func namespacePendingSnapshot(t *testing.T, ctx context.Context) func() {
	started := time.Now()
	timer := time.NewTimer(15 * time.Second)
	stop, done := make(chan struct{}), make(chan struct{})
	var pending atomic.Bool
	pending.Store(true)
	go func() {
		defer close(done)
		select {
		case <-stop:
			return
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		if !pending.Load() || ctx.Err() != nil {
			return
		}
		buffer := make([]byte, 1<<20)
		pendingBefore := pending.Load()
		if !pendingBefore {
			return
		}
		elapsed := time.Since(started)
		count := runtime.Stack(buffer, true)
		returnedElapsed := time.Since(started)
		pendingAfter := pending.Load()
		fmt.Printf("pending stack snapshot entered elapsed_nanos=%d scheduled_nanos=%d pending_before=%v\n", elapsed.Nanoseconds(), (15 * time.Second).Nanoseconds(), pendingBefore)
		fmt.Printf("pending stack snapshot bytes=%d capacity=%d truncated=%v pending_after=%v returned_elapsed_nanos=%d\n%s\npending stack snapshot returned\n", count, len(buffer), count == len(buffer), pendingAfter, returnedElapsed.Nanoseconds(), buffer[:count])
	}()
	var once sync.Once
	finish := func() {
		once.Do(func() {
			pending.Store(false)
			timer.Stop()
			close(stop)
			<-done
			fmt.Println("pending stack sampler joined")
		})
	}
	t.Cleanup(finish)
	return finish
}

func TestNamespacePendingSnapshotCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	finish := namespacePendingSnapshot(t, ctx)
	cancel()
	finish()
	finish()
}
