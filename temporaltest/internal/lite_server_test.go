package temporalite

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLiteServerStartContextDoesNotStartAfterCancellation(t *testing.T) {
	t.Parallel()
	started := make(chan struct{})
	server := &controlledServer{startEntered: started}
	liteServer := &LiteServer{internal: server}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := liteServer.StartContext(ctx)

	require.ErrorIs(t, err, context.Canceled)
	select {
	case <-started:
		t.Fatal("server start was called after context cancellation")
	case <-time.After(50 * time.Millisecond):
	}
	require.Zero(t, server.startCalls.Load())
}

func TestLiteServerContextOperationsResumeWithoutDuplicateCalls(t *testing.T) {
	t.Parallel()
	startEntered := make(chan struct{})
	startBlock := make(chan struct{})
	stopEntered := make(chan struct{})
	stopBlock := make(chan struct{})
	server := &controlledServer{
		startEntered: startEntered,
		startBlock:   startBlock,
		stopEntered:  stopEntered,
		stopBlock:    stopBlock,
	}
	liteServer := &LiteServer{internal: server}

	startCtx, cancelStart := context.WithCancel(context.Background())
	startResult := make(chan error, 1)
	go func() { startResult <- liteServer.StartContext(startCtx) }()
	requireClosed(t, startEntered)
	cancelStart()
	require.ErrorIs(t, <-startResult, context.Canceled)
	close(startBlock)
	require.NoError(t, liteServer.StartContext(context.Background()))
	require.Equal(t, int32(1), server.startCalls.Load())

	stopCtx, cancelStop := context.WithCancel(context.Background())
	stopResult := make(chan error, 1)
	go func() { stopResult <- liteServer.StopContext(stopCtx) }()
	requireClosed(t, stopEntered)
	cancelStop()
	require.ErrorIs(t, <-stopResult, context.Canceled)
	close(stopBlock)
	require.NoError(t, liteServer.StopContext(context.Background()))
	require.Equal(t, int32(1), server.stopCalls.Load())
}

func requireClosed(t *testing.T, channel <-chan struct{}) {
	t.Helper()
	select {
	case <-channel:
	case <-time.After(time.Second):
		t.Fatal("operation did not start")
	}
}

type controlledServer struct {
	startCalls   atomic.Int32
	stopCalls    atomic.Int32
	startEntered chan struct{}
	startBlock   chan struct{}
	stopEntered  chan struct{}
	stopBlock    chan struct{}
}

func (s *controlledServer) Start() error {
	s.startCalls.Add(1)
	if s.startEntered != nil {
		close(s.startEntered)
	}
	if s.startBlock != nil {
		<-s.startBlock
	}
	return nil
}

func (s *controlledServer) Stop() error {
	s.stopCalls.Add(1)
	if s.stopEntered != nil {
		close(s.stopEntered)
	}
	if s.stopBlock != nil {
		<-s.stopBlock
	}
	return nil
}
