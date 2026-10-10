// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime_test

import (
	"runtime"
	"testing"
)

func TestGomadVirtualPollSemaphore(t *testing.T) {
	for _, mode := range []int32{'r', 'w'} {
		for _, virtual := range []bool{false, true} {
			for _, ready := range []bool{false, true} {
				for state := 0; state < 4; state++ {
					got, delta, extracted, balanced := runtime.GomadVirtualSemaphoreCase(mode, state, virtual, ready)
					want := uintptr(0)
					if ready || state == 1 {
						want = 1
					}
					wantDelta := int32(0)
					if state == 3 && !virtual {
						wantDelta = -1
					}
					if got != want || delta != wantDelta || extracted != (state == 3) || !balanced {
						t.Fatalf("mode=%c virtual=%t ready=%t state=%d: semaphore=%d delta=%d extracted=%t balanced=%t", mode, virtual, ready, state, got, delta, extracted, balanced)
					}
				}
			}
		}
	}
}

func TestGomadVirtualPollRegistrationLifetime(t *testing.T) {
	if !runtime.GomadVirtualRegistryCase() {
		t.Fatal("registration admitted stale, duplicate, or closing readiness")
	}
}
