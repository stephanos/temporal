package server

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/testpilot"
)

type observedContext struct {
	context.Context
	checked chan struct{}
	once    sync.Once
}

func (c *observedContext) Err() error {
	err := c.Context.Err()
	if err == nil {
		c.once.Do(func() { close(c.checked) })
	}
	return err
}

func TestCancellationDuringDriverSerialization(t *testing.T) {
	for _, operation := range []string{"open", "mint", "bridge", "publish", "consume", "quarantine", "close-session", "close-host", "diagnose"} {
		t.Run(operation, func(t *testing.T) {
			h, source, _ := fixture(t, "127.0.0.1:1")
			s, origin := handleSession(t, h, source, "run")
			opaque, err := s.NewHandle(t.Context(), origin, successfulHandleEffect())
			require.NoError(t, err)
			if operation == "consume" {
				require.NoError(t, s.Publish(t.Context(), origin, "handle", opaque))
			}
			handle := &effect{session: s}
			program := prepared(t, h, source)
			call := func(ctx context.Context) error {
				switch operation {
				case "open":
					_, err := h.OpenSession(ctx, "new", program)
					return err
				case "mint":
					_, err := s.NewHandle(ctx, origin, successfulHandleEffect())
					return err
				case "bridge":
					_, err := s.Bridge(ctx)
					return err
				case "publish":
					return s.Publish(ctx, origin, "handle", opaque)
				case "consume":
					_, err := s.Consume(ctx, "handle")
					return err
				case "quarantine":
					return s.Quarantine(ctx, handle)
				case "close-session":
					return s.Close(ctx)
				case "close-host":
					return h.Close(ctx)
				case "diagnose":
					return s.Diagnose(ctx, "run", nil)
				default:
					return errInvalid
				}
			}
			parent, cancel := context.WithCancel(t.Context())
			defer cancel()
			ctx := &observedContext{Context: parent, checked: make(chan struct{})}
			result := make(chan error, 1)
			h.mu.Lock()
			go func() { result <- call(ctx) }()
			<-ctx.checked
			cancel()
			timer := time.NewTimer(100 * time.Millisecond)
			defer timer.Stop()
			var callErr error
			returned := false
			select {
			case callErr = <-result:
				returned = true
			case <-timer.C:
			}
			h.mu.Unlock()
			if !returned {
				callErr = <-result
			}
			require.True(t, returned, "operation did not honor cancellation while serialized")
			require.ErrorIs(t, callErr, context.Canceled)
			require.False(t, s.closed)
			require.False(t, h.closed)
			require.EqualValues(t, 1, s.minted)
			require.Zero(t, s.diagnostics)
		})
	}
}
func TestRejectedHandleInvocationRestoresClaimForCleanup(t *testing.T) {
	for _, failure := range []string{"canceled", "capacity"} {
		t.Run(failure, func(t *testing.T) {
			h, source, _ := fixture(t, "127.0.0.1:1")
			s, origin := handleSession(t, h, source, "run")
			handle, err := s.NewHandle(t.Context(), origin, successfulHandleEffect())
			require.NoError(t, err)
			require.NoError(t, s.Publish(t.Context(), origin, "handle", handle))
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			original, err := s.Consume(ctx, "handle")
			require.NoError(t, err)
			release := make(chan struct{})
			var blocker *effect
			if failure == "canceled" {
				cancel()
			} else {
				other, err := h.OpenSession(t.Context(), "other", prepared(t, h, source))
				require.NoError(t, err)
				h.profile.ProgramLimits.MaxAttempts = 1
				blocker, err = other.start(t.Context(), coordinate("other", "check"), 2000, func(context.Context) testpilot.EffectResult { <-release; return testpilot.EffectResult{} })
				require.NoError(t, err)
			}
			denied, err := s.InvokeHandle(ctx, coordinate("run", "check"), original, handleValue())
			require.Error(t, err)
			require.Nil(t, denied)
			cleanupCtx, cleanupCancel := context.WithCancel(t.Context())
			defer cleanupCancel()
			replacement, err := s.Consume(cleanupCtx, "handle")
			require.NoError(t, err)
			denied, err = s.InvokeHandle(t.Context(), coordinate("run", "check"), original, handleValue())
			require.Error(t, err)
			require.Nil(t, denied)
			if blocker != nil {
				close(release)
				require.NoError(t, blocker.Drain(t.Context()))
			}
			accepted, err := s.InvokeHandle(cleanupCtx, coordinate("run", "check"), replacement, handleValue())
			require.NoError(t, err)
			require.NoError(t, accepted.Drain(t.Context()))
			cleanupCancel()
			_, err = s.Consume(t.Context(), "handle")
			require.Error(t, err)
			denied, err = s.InvokeHandle(t.Context(), coordinate("run", "check"), replacement, handleValue())
			require.Error(t, err)
			require.Nil(t, denied)
		})
	}
}
func TestProfileMethodBoundsBeforeClone(t *testing.T) {
	h, _, _ := fixture(t, "127.0.0.1:1")
	for _, count := range []int{10001, 100000} {
		profile := h.Snapshot()
		profile.Roles[0].Methods = make([]string, count)
		require.False(t, validProfile(profile))
		_, err := New(Options{Profile: profile})
		require.ErrorIs(t, err, errInvalid)
	}
	profile := h.Snapshot()
	role := profile.Roles[0]
	role.Methods = make([]string, 10000)
	for i := 0; i < 10; i++ {
		profile.Roles = append(profile.Roles, role)
	}
	require.False(t, validProfile(profile))
	_, err := New(Options{Profile: profile})
	require.ErrorIs(t, err, errInvalid)
}
