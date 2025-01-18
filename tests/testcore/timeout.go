// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package testcore

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/maruel/panicparse/v2/stack"
)

type testTimeout struct {
	updateCh chan time.Duration
	closeCh  chan struct{}
	resetCh  chan *testing.T
}

func newTestTimeout(t *testing.T, after time.Duration) *testTimeout {
	res := &testTimeout{
		updateCh: make(chan time.Duration),
		closeCh:  make(chan struct{}),
		resetCh:  make(chan *testing.T),
	}
	go func() {
		startedAt := time.Now()
		for {
			tick := time.NewTimer(time.Second)
			select {
			case after = <-res.updateCh:
			case t = <-res.resetCh:
				startedAt = time.Now()
			case <-res.closeCh:
				return
			case <-tick.C:
				if time.Now().After(startedAt.Add(after)) {
					// Collect abridged stacktrace.
					var traces string
					buf := make([]byte, 5<<20) // give it plenty of room
					runtime.Stack(buf, true)
					snap, _, err := stack.ScanSnapshot(bytes.NewReader(buf), io.Discard, stack.DefaultOpts())
					if err != nil && err != io.EOF {
						t.Logf("failed to parse (entire) stack trace: %v", err)
					} else {
						traces = "\nstacktrace:\n\n"
						for _, goroutine := range snap.Goroutines {
							var print bool
							for _, line := range goroutine.Stack.Calls {
								if strings.HasPrefix(line.DirSrc, "tests/") {
									print = true
									break
								}
							}
							if print {
								traces += fmt.Sprintf("goroutine %d [%v]:\n", goroutine.ID, goroutine.State)
								for _, call := range goroutine.Stack.Calls {
									file := call.RelSrcPath
									traces += fmt.Sprintf("\t%s:%d\n", file, call.Line)
								}
								traces += "\n\n"
							}
						}
					}

					// Cannot use t.Fatalf since it will wait until the test completes.
					t.Logf("Test timed out after %v. To extend the time of the test, call s.SetTestTimeout().", after)
					t.Logf(traces)

					// `timeout` exits with code 124 to indicate a timeout - might as well do that here.
					// GitHub Actions will mark this test when it sees this exit code.
					//revive:disable-next-line:deep-exit
					os.Exit(124)
				}
			}
		}
	}()
	return res
}

func (b testTimeout) set(d time.Duration) {
	b.updateCh <- d
}

func (b testTimeout) cancel() {
	b.closeCh <- struct{}{}
}

func (b testTimeout) reset(t *testing.T) {
	b.resetCh <- t
}
