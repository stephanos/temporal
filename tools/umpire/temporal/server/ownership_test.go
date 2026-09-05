package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire"
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

func TestCancellationDuringHostSerialization(t *testing.T) {
	for _, operation := range []string{"open", "mint", "bridge", "publish", "consume", "quarantine", "close-session", "close-host", "diagnose"} {
		t.Run(operation, func(t *testing.T) {
			h, source, _ := fixture(t, "127.0.0.1:1")
			s, origin := nexusSession(t, h, source, "run")
			capability, err := s.NewCompletionCapability(t.Context(), origin, CompletionInfo{URL: "http://localhost", OperationToken: "token"})
			require.NoError(t, err)
			if operation == "consume" {
				require.NoError(t, s.Publish(t.Context(), origin, "capability", capability))
			}
			handle := &effect{session: s}
			call := func(ctx context.Context) error {
				switch operation {
				case "open":
					_, err := h.open(ctx, "new", source.Program)
					return err
				case "mint":
					_, err := s.NewCompletionCapability(ctx, origin, CompletionInfo{URL: "http://localhost", OperationToken: "token"})
					return err
				case "bridge":
					_, err := s.Bridge(ctx)
					return err
				case "publish":
					return s.Publish(ctx, origin, "capability", capability)
				case "consume":
					_, err := s.Consume(ctx, "capability")
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
func TestRejectedCompletionRestoresClaimForCleanup(t *testing.T) {
	for _, failure := range []string{"canceled", "capacity"} {
		t.Run(failure, func(t *testing.T) {
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
			defer target.Close()
			h, source, _ := fixture(t, "127.0.0.1:1")
			s, origin := nexusSession(t, h, source, "run")
			capability, err := s.NewCompletionCapability(t.Context(), origin, CompletionInfo{URL: target.URL, OperationToken: "token"})
			require.NoError(t, err)
			require.NoError(t, s.Publish(t.Context(), origin, "capability", capability))
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			original, err := s.Consume(ctx, "capability")
			require.NoError(t, err)
			release := make(chan struct{})
			var blocker *effect
			if failure == "canceled" {
				cancel()
			} else {
				other, err := h.open(t.Context(), "other", source.Program)
				require.NoError(t, err)
				h.profile.ProgramLimits.MaxAttempts = 1
				blocker, err = other.start(t.Context(), coordinate("other", "check"), source.Program.Entrypoints[0].Nodes[0].Bounds, func(context.Context) umpire.EffectResult { <-release; return umpire.EffectResult{} })
				require.NoError(t, err)
			}
			denied, err := s.CompleteNexusOperation(ctx, coordinate("run", "complete"), original, completionValue())
			require.Error(t, err)
			require.Nil(t, denied)
			cleanupCtx, cleanupCancel := context.WithCancel(t.Context())
			defer cleanupCancel()
			replacement, err := s.Consume(cleanupCtx, "capability")
			require.NoError(t, err)
			denied, err = s.CompleteNexusOperation(t.Context(), coordinate("run", "complete"), original, completionValue())
			require.Error(t, err)
			require.Nil(t, denied)
			if blocker != nil {
				close(release)
				require.NoError(t, blocker.Drain(t.Context()))
			}
			accepted, err := s.CompleteNexusOperation(cleanupCtx, coordinate("run", "complete"), replacement, completionValue())
			require.NoError(t, err)
			require.NoError(t, accepted.Drain(t.Context()))
			cleanupCancel()
			_, err = s.Consume(t.Context(), "capability")
			require.Error(t, err)
			denied, err = s.CompleteNexusOperation(t.Context(), coordinate("run", "complete"), replacement, completionValue())
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
