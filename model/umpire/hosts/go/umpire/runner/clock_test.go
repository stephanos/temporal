package runner

import (
	"context"
	"sync"
	"time"
)

// testClock replaces the Runner's deadlines: none ends on its own, so no test depends on how
// fast the machine is. A test ends a wait with expire, or makes every deadline of a kind end as
// it is made.
type testClock struct {
	mu        sync.Mutex
	changed   *sync.Cond
	immediate map[deadlineKind]bool
	made      map[deadlineKind]int
	active    map[*clockContext]deadlineKind
}

func newTestClock() *testClock {
	c := &testClock{immediate: map[deadlineKind]bool{}, made: map[deadlineKind]int{}, active: map[*clockContext]deadlineKind{}}
	c.changed = sync.NewCond(&c.mu)
	return c
}

func (c *testClock) deadline(parent context.Context, kind deadlineKind, _ time.Duration) (context.Context, context.CancelFunc) {
	ctx := &clockContext{Context: parent, done: make(chan struct{})}
	stop := context.AfterFunc(parent, func() { ctx.end(parent.Err()) })
	c.mu.Lock()
	c.made[kind]++
	immediate := c.immediate[kind]
	if !immediate {
		c.active[ctx] = kind
	}
	c.changed.Broadcast()
	c.mu.Unlock()
	if immediate {
		ctx.end(context.DeadlineExceeded)
	}
	return ctx, func() {
		stop()
		c.mu.Lock()
		delete(c.active, ctx)
		c.mu.Unlock()
		ctx.end(context.Canceled)
	}
}

// count is how many deadlines of a kind the Runner has made so far.
func (c *testClock) count(kind deadlineKind) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.made[kind]
}

// await blocks until the Runner has made n deadlines of a kind.
func (c *testClock) await(kind deadlineKind, n int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for c.made[kind] < n {
		c.changed.Wait()
	}
}

// expire ends every live deadline of a kind.
func (c *testClock) expire(kind deadlineKind) {
	c.mu.Lock()
	var ending []*clockContext
	for ctx, k := range c.active {
		if k == kind {
			ending = append(ending, ctx)
			delete(c.active, ctx)
		}
	}
	c.mu.Unlock()
	for _, ctx := range ending {
		ctx.end(context.DeadlineExceeded)
	}
}

type clockContext struct {
	context.Context
	done chan struct{}
	once sync.Once
	mu   sync.Mutex
	err  error
}

func (c *clockContext) Done() <-chan struct{} { return c.done }

func (c *clockContext) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

func (c *clockContext) end(err error) {
	c.once.Do(func() {
		c.mu.Lock()
		c.err = err
		c.mu.Unlock()
		close(c.done)
	})
}
