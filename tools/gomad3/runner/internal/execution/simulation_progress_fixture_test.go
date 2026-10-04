//go:build unix

package execution

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"
)

// quiesce evaluates Done after installing its waiter and releasing the arbiter
// mutex. This handshake observes installation without polling private state.
type simulationProgressContext struct {
	context.Context
	entered chan struct{}
	once    sync.Once
}

func (ctx *simulationProgressContext) Done() <-chan struct{} {
	ctx.once.Do(func() { close(ctx.entered) })
	return ctx.Context.Done()
}

func simulationProgressReceive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("simulation progress handshake timed out")
		var zero T
		return zero
	}
}

func simulationProgressReadFrame(source io.ReadCloser) (simulationFrame, error) {
	result := make(chan simulationModelRead, 1)
	exited := make(chan struct{})
	go func() {
		defer close(exited)
		frame, err := readSimulationModelTransportFrame(source)
		result <- simulationModelRead{frame: frame, err: err}
	}()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case read := <-result:
		<-exited
		return read.frame, read.err
	case <-timer.C:
		err := errors.Join(context.DeadlineExceeded, source.Close())
		<-exited
		return simulationFrame{}, fmt.Errorf("simulation progress frame read timed out: %w", err)
	}
}

func simulationProgressWriteFrame(destination io.WriteCloser, frame simulationFrame) error {
	result := make(chan error, 1)
	exited := make(chan struct{})
	go func() {
		defer close(exited)
		result <- writeSimulationModelTransportFrame(destination, frame)
	}()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case err := <-result:
		<-exited
		return err
	case <-timer.C:
		err := errors.Join(context.DeadlineExceeded, destination.Close())
		<-exited
		return fmt.Errorf("simulation progress frame write timed out: %w", err)
	}
}

func simulationProgressWait(t *testing.T, arbiter *simulationTimeArbiter, participant *simulationTimeParticipant, request simulationTimeRequest) <-chan simulationTimeResult {
	t.Helper()
	parent, cancel := context.WithCancel(context.Background())
	ctx := &simulationProgressContext{Context: parent, entered: make(chan struct{})}
	results := make(chan simulationTimeResult, 1)
	exited := make(chan struct{})
	go func() {
		defer close(exited)
		response, err := arbiter.quiesce(ctx, participant, request)
		results <- simulationTimeResult{response: response, err: err}
	}()
	t.Cleanup(func() {
		cancel()
		simulationProgressReceive(t, exited)
	})
	select {
	case <-ctx.entered:
	case result := <-results:
		t.Fatalf("quiescence returned before waiter installation: %#v", result)
	case <-time.After(5 * time.Second):
		t.Fatal("quiescence did not install its waiter")
	}
	if !arbiter.isQuiescent(participant) {
		t.Fatal("participant did not become quiescent")
	}
	return results
}

func simulationProgressResponse(t *testing.T, got simulationTimeResponse, err error, generation uint64, kind simulationTimeResponseKind, at int64) {
	t.Helper()
	want := simulationTimeResponse{Generation: generation, Kind: kind, Time: at}
	if err != nil || got != want {
		t.Fatalf("time response = %#v, error = %v; want %#v", got, err, want)
	}
}

func simulationProgressCoordinator(t *testing.T) (*simulationCoordinator, *simulationNodeProcess) {
	t.Helper()
	coordinator, err := newSimulationCoordinator(Spec{})
	if err != nil {
		t.Fatal(err)
	}
	participant, err := coordinator.time.register("node/1")
	if err != nil {
		t.Fatal(err)
	}
	coordinator.time.activate(participant)
	node := &simulationNodeProcess{node: "node", incarnation: 1, time: participant, done: make(chan struct{})}
	coordinator.nodes["node/1"] = node
	return coordinator, node
}

func simulationProgressTransport(t *testing.T, coordinator *simulationCoordinator) (*io.PipeReader, *io.PipeWriter, <-chan simulationFrame) {
	t.Helper()
	requestRead, requestWrite := io.Pipe()
	responseRead, responseWrite := io.Pipe()
	discarded := make(chan simulationFrame, 1)
	coordinator.model = newSimulationModelTransport(requestWrite, responseRead, func() {
		_ = coordinator.time.progress.apply(modelRequestDispatched{coordinator: coordinator.coordinator})
	}, coordinator.handleModelArrival, func(frame simulationFrame) error {
		err := coordinator.time.progress.apply(modelAbandonedResponseDiscarded{coordinator: coordinator.coordinator, arrivals: frame.Arrivals})
		discarded <- frame
		return err
	})
	t.Cleanup(func() {
		if err := errors.Join(coordinator.model.close(), requestRead.Close(), responseWrite.Close()); err != nil {
			t.Error(err)
		}
	})
	return requestRead, responseWrite, discarded
}
