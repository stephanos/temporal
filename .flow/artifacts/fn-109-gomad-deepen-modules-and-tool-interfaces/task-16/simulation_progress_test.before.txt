//go:build unix

package execution

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/world"
)

func TestSimulationTimeProgressArrivalAtInstalledQuiescence(t *testing.T) {
	coordinator, node := simulationProgressCoordinator(t)
	results := simulationProgressWait(t, coordinator.time, coordinator.coordinator, simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 20,
	})
	frame := simulationFrame{Kind: simulationFrameExplorationPlan, Request: 1}
	if _, err := coordinator.handle(context.Background(), frame); err != nil {
		t.Fatal(err)
	}
	result := simulationProgressReceive(t, results)
	simulationProgressResponse(t, result.response, result.err, 1, simulationTimeExternal, simulationInitialTime)
	if coordinator.time.isQuiescent(coordinator.coordinator) {
		t.Fatal("arrival left the coordinator quiescent")
	}
	coordinator.handleCoordinatorDelivery(frame)
	response, err := coordinator.handleCoordinatorTime(context.Background(), simulationTimeRequest{
		Generation: 2, Current: simulationInitialTime, Deadline: simulationInitialTime + 20,
	})
	simulationProgressResponse(t, response, err, 2, simulationTimeRetry, simulationInitialTime)
	results = simulationProgressWait(t, coordinator.time, coordinator.coordinator, simulationTimeRequest{
		Generation: 3, Current: simulationInitialTime, Deadline: simulationInitialTime + 20, Arrivals: 1,
	})
	response, err = coordinator.time.quiesce(context.Background(), node.time, simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
	})
	simulationProgressResponse(t, response, err, 1, simulationTimeAdvance, simulationInitialTime+10)
	result = simulationProgressReceive(t, results)
	simulationProgressResponse(t, result.response, result.err, 3, simulationTimeAdvance, simulationInitialTime+10)
}

func TestSimulationModelProgressConcurrentOperationsOnOneParticipant(t *testing.T) {
	for _, order := range [][]int{{0, 1}, {1, 0}} {
		name := "first-first"
		if order[0] == 1 {
			name = "second-first"
		}
		t.Run(name, func(t *testing.T) {
			coordinator, node := simulationProgressCoordinator(t)
			requestRead, responseWrite, _ := simulationProgressTransport(t, coordinator)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var requests [2]simulationFrame
			var results [2]chan simulationModelResult
			for index, payload := range []string{"first", "second"} {
				results[index] = make(chan simulationModelResult, 1)
				go func(index int, payload string) {
					frame, err := coordinator.handleNodeFrame(ctx, node, simulationFrame{
						Kind: simulationFrameModel, Node: "node", Incarnation: 1, Payload: []byte(payload),
					})
					results[index] <- simulationModelResult{frame: frame, err: err}
				}(index, payload)
				var err error
				requests[index], err = simulationProgressReadFrame(requestRead)
				if err != nil {
					t.Fatal(err)
				}
			}
			response, err := coordinator.time.quiesce(ctx, node.time, simulationTimeRequest{
				Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
			})
			simulationProgressResponse(t, response, err, 1, simulationTimeExternal, simulationInitialTime)
			for position, index := range order {
				frame := requests[index]
				frame.Kind, frame.Arrivals = simulationFrameResponse, 1
				if err := simulationProgressWriteFrame(responseWrite, frame); err != nil {
					t.Fatal(err)
				}
				result := simulationProgressReceive(t, results[index])
				if result.err != nil || !reflect.DeepEqual(result.frame.Payload, requests[index].Payload) {
					t.Fatalf("operation %d response = %#v, error = %v", index, result.frame, result.err)
				}
				generation := uint64(2 + position*2)
				response, err = coordinator.time.quiesce(ctx, node.time, simulationTimeRequest{
					Generation: generation, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
				})
				simulationProgressResponse(t, response, err, generation, simulationTimeRetry, simulationInitialTime)
				if position == 0 {
					response, err = coordinator.time.quiesce(ctx, node.time, simulationTimeRequest{
						Generation: 3, Current: simulationInitialTime, Deadline: simulationInitialTime + 10, Arrivals: 1,
					})
					simulationProgressResponse(t, response, err, 3, simulationTimeExternal, simulationInitialTime)
				}
			}
			nodeResult := simulationProgressWait(t, coordinator.time, node.time, simulationTimeRequest{
				Generation: 5, Current: simulationInitialTime, Deadline: simulationInitialTime + 10, Arrivals: 1,
			})
			response, err = coordinator.handleCoordinatorTime(ctx, simulationTimeRequest{
				Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 20,
			})
			simulationProgressResponse(t, response, err, 1, simulationTimeAdvance, simulationInitialTime+10)
			result := simulationProgressReceive(t, nodeResult)
			simulationProgressResponse(t, result.response, result.err, 5, simulationTimeAdvance, simulationInitialTime+10)
		})
	}
}

func TestSimulationTimeProgressUnknownAcknowledgementBeforeAdmission(t *testing.T) {
	coordinator, err := newSimulationCoordinator(Spec{Simulation: &SimulationCapability{
		ExplorationPlan: []byte("plan"), ExplorationRecordLimit: 100, ExplorationRecordCount: 1,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := coordinator.handle(context.Background(), simulationFrame{
		Kind: simulationFrameExplorationRecord, Request: 1, Arrivals: 1, Payload: []byte("record"),
	}); err == nil || !strings.Contains(err.Error(), "acknowledged unknown external work") {
		t.Fatalf("unknown acknowledgement error = %v", err)
	}
	if records := coordinator.retainedExplorationRecords(); len(records) != 0 {
		t.Fatalf("rejected admission mutated records: %q", records)
	}
	response, err := coordinator.handleCoordinatorTime(context.Background(), simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
	})
	simulationProgressResponse(t, response, err, 1, simulationTimeAdvance, simulationInitialTime+10)
}

func TestSimulationTimeProgressPartialAcknowledgementKeepsOtherDeliveryRunnable(t *testing.T) {
	coordinator, err := newSimulationCoordinator(Spec{})
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []uint64{1, 2} {
		frame := simulationFrame{Kind: simulationFrameExplorationPlan, Request: request}
		if _, err := coordinator.handle(context.Background(), frame); err != nil {
			t.Fatal(err)
		}
		coordinator.handleCoordinatorDelivery(frame)
	}
	response, err := coordinator.handleCoordinatorTime(context.Background(), simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10, Arrivals: 1,
	})
	simulationProgressResponse(t, response, err, 1, simulationTimeRetry, simulationInitialTime)
	response, err = coordinator.handleCoordinatorTime(context.Background(), simulationTimeRequest{
		Generation: 2, Current: simulationInitialTime, Deadline: simulationInitialTime + 10, Arrivals: 1,
	})
	simulationProgressResponse(t, response, err, 2, simulationTimeAdvance, simulationInitialTime+10)
}

func TestSimulationTimeProgressUnknownAcknowledgementBeforeForwardAndTransfer(t *testing.T) {
	coordinator, node := simulationProgressCoordinator(t)
	if _, err := coordinator.handleNodeFrame(context.Background(), node, simulationFrame{
		Kind: simulationFrameModel, Node: "node", Incarnation: 1, Arrivals: 1,
	}); err == nil || !strings.Contains(err.Error(), "acknowledged unknown external work") {
		t.Fatalf("unknown forward acknowledgement error = %v", err)
	}
	if err := coordinator.handleModelArrival(simulationFrame{Node: "node", Incarnation: 1, Arrivals: 1}); err == nil || !strings.Contains(err.Error(), "acknowledged unknown external work") {
		t.Fatalf("unknown transfer acknowledgement error = %v", err)
	}
	results := simulationProgressWait(t, coordinator.time, node.time, simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
	})
	response, err := coordinator.handleCoordinatorTime(context.Background(), simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 20,
	})
	simulationProgressResponse(t, response, err, 1, simulationTimeAdvance, simulationInitialTime+10)
	result := simulationProgressReceive(t, results)
	simulationProgressResponse(t, result.response, result.err, 1, simulationTimeAdvance, simulationInitialTime+10)
}

func TestSimulationTimeProgressPreexistingMalformedWaitWakesQuiescence(t *testing.T) {
	coordinator, _ := simulationProgressCoordinator(t)
	results := simulationProgressWait(t, coordinator.time, coordinator.coordinator, simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
	})
	if err := coordinator.handleWaitAcceptance(simulationFrame{Kind: simulationFrameWait, Request: 1, Arrivals: 1}); err == nil || !strings.Contains(err.Error(), "acknowledged unknown external work") {
		t.Fatalf("malformed wait acknowledgement error = %v", err)
	}
	result := simulationProgressReceive(t, results)
	simulationProgressResponse(t, result.response, result.err, 1, simulationTimeRetry, simulationInitialTime)
}

func TestSimulationTimeProgressPreexistingDuplicateAdmissionConsumesArrival(t *testing.T) {
	coordinator, err := newSimulationCoordinator(Spec{})
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []uint64{1, 2} {
		if _, err := coordinator.handle(context.Background(), simulationFrame{Kind: simulationFrameExplorationPlan, Request: request}); err != nil {
			t.Fatal(err)
		}
		if request == 1 {
			coordinator.handleCoordinatorDelivery(simulationFrame{Request: request})
		}
	}
	if _, err := coordinator.handle(context.Background(), simulationFrame{
		Kind: simulationFrameExplorationPlan, Request: 2, Arrivals: 1,
	}); err == nil {
		t.Fatal("duplicate request was accepted")
	}
	response, err := coordinator.handleCoordinatorTime(context.Background(), simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
	})
	simulationProgressResponse(t, response, err, 1, simulationTimeExternal, simulationInitialTime)
}

func TestSimulationTimeProgressDeathAndRestartRejectStaleIncarnation(t *testing.T) {
	coordinator, old := simulationProgressCoordinator(t)
	results := simulationProgressWait(t, coordinator.time, old.time, simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
	})
	coordinator.removeNode(old)
	result := simulationProgressReceive(t, results)
	if result.err == nil {
		t.Fatal("removed participant's waiter succeeded")
	}
	response, err := coordinator.handleCoordinatorTime(context.Background(), simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 17,
	})
	simulationProgressResponse(t, response, err, 1, simulationTimeAdvance, simulationInitialTime+17)
	restarted, err := coordinator.time.register("node/2")
	if err != nil {
		t.Fatal(err)
	}
	if current := coordinator.time.activate(restarted); current != simulationInitialTime+17 {
		t.Fatalf("restart time = %d", current)
	}
	coordinator.nodes["node/2"] = &simulationNodeProcess{node: "node", incarnation: 2, time: restarted, done: make(chan struct{})}
	if err := coordinator.handleModelArrival(simulationFrame{Node: "node", Incarnation: 1, Arrivals: 1}); err == nil {
		t.Fatal("stale model arrival was accepted")
	}
	if _, err := coordinator.handleNodeFrame(context.Background(), old, simulationFrame{
		Kind: simulationFrameTerminal, Node: "node", Incarnation: 1, Payload: []byte("stale"),
	}); err == nil {
		t.Fatal("removed participant mutated terminal state")
	}
	if old.terminal != nil {
		t.Fatalf("stale terminal state = %q", old.terminal)
	}
	results = simulationProgressWait(t, coordinator.time, coordinator.coordinator, simulationTimeRequest{
		Generation: 2, Current: simulationInitialTime + 17, Deadline: simulationInitialTime + 30,
	})
	response, err = coordinator.time.quiesce(context.Background(), restarted, simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime + 17, Deadline: simulationInitialTime + 20,
	})
	simulationProgressResponse(t, response, err, 1, simulationTimeAdvance, simulationInitialTime+20)
	result = simulationProgressReceive(t, results)
	simulationProgressResponse(t, result.response, result.err, 2, simulationTimeAdvance, simulationInitialTime+20)
}

func TestSimulationModelProgressCancellationKeepsCommitAndDiscardsLateArrival(t *testing.T) {
	coordinator, node := simulationProgressCoordinator(t)
	requestRead, responseWrite, discarded := simulationProgressTransport(t, coordinator)
	model, err := world.New(world.Config{Seed: 1, Limits: world.Limits{
		MaxRequests: 10, MaxEvents: 10, MaxQueuedEvents: 10, MaxTransitions: 100, MaxPayloadBytes: 1024, MaxStringBytes: 128,
	}})
	if err != nil {
		t.Fatal(err)
	}
	serverCtx, stopServer := context.WithCancel(context.Background())
	committedRequest := make(chan simulationFrame, 1)
	release := make(chan struct{})
	served := make(chan error, 1)
	handlerExited := make(chan struct{})
	go func() {
		served <- serveSimulationModels(serverCtx, requestRead, responseWrite, func(ctx context.Context, request simulationFrame) (simulationFrame, error) {
			defer close(handlerExited)
			if _, err := model.Register(world.Request{
				Kind: "write", Resource: world.ResourceID{Adapter: "test", Kind: "volume", Key: "file"}, Payload: request.Payload,
			}); err != nil {
				return simulationFrame{}, err
			}
			committedRequest <- request
			select {
			case <-release:
				return simulationFrame{Arrivals: 1, Payload: []byte("committed")}, nil
			case <-ctx.Done():
				return simulationFrame{}, ctx.Err()
			}
		}, nil, nil)
	}()
	t.Cleanup(func() {
		stopServer()
		if err := errors.Join(requestRead.Close(), responseWrite.Close()); err != nil {
			t.Error(err)
		}
		err := simulationProgressReceive(t, served)
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Error(err)
		}
		simulationProgressReceive(t, handlerExited)
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan simulationModelResult, 1)
	go func() {
		frame, err := coordinator.handleNodeFrame(ctx, node, simulationFrame{
			Kind: simulationFrameModel, Node: "node", Incarnation: 1, Payload: []byte("committed"),
		})
		result <- simulationModelResult{frame: frame, err: err}
	}()
	request := simulationProgressReceive(t, committedRequest)
	committed := model.Snapshot()
	if len(committed.Requests) != 1 || string(committed.Requests[0].Request.Payload) != "committed" {
		t.Fatalf("model did not commit the request: %#v", committed)
	}
	cancel()
	if got := simulationProgressReceive(t, result); !errors.Is(got.err, context.Canceled) {
		t.Fatalf("cancelled operation = %#v", got)
	}
	coordinator.removeNode(node)
	response, err := coordinator.handleCoordinatorTime(context.Background(), simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
	})
	simulationProgressResponse(t, response, err, 1, simulationTimeRetry, simulationInitialTime)
	close(release)
	request.Kind, request.Arrivals = simulationFrameResponse, 1
	if got := simulationProgressReceive(t, discarded); !reflect.DeepEqual(got, request) {
		t.Fatalf("discarded response = %#v, want %#v", got, request)
	}
	response, err = coordinator.handleCoordinatorTime(context.Background(), simulationTimeRequest{
		Generation: 2, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
	})
	simulationProgressResponse(t, response, err, 2, simulationTimeAdvance, simulationInitialTime+10)
	if got := model.Snapshot(); !reflect.DeepEqual(got, committed) {
		t.Fatalf("cancellation or late response changed committed model: %#v", got)
	}
}

func TestSimulationModelProgressRejectsUnknownAndDuplicateResponsesBeforeCallbacks(t *testing.T) {
	for _, test := range []struct {
		name      string
		duplicate bool
		abandoned bool
	}{
		{name: "unknown"},
		{name: "duplicate", duplicate: true},
		{name: "abandoned duplicate", duplicate: true, abandoned: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			requestRead, requestWrite := io.Pipe()
			responseRead, responseWrite := io.Pipe()
			callbacks := make(chan string, 2)
			transport := newSimulationModelTransport(requestWrite, responseRead, nil, func(simulationFrame) error {
				callbacks <- "arrived"
				return nil
			}, func(simulationFrame) error {
				callbacks <- "discarded"
				return nil
			})
			t.Cleanup(func() {
				if err := errors.Join(transport.close(), requestRead.Close(), responseWrite.Close()); err != nil {
					t.Error(err)
				}
			})
			frame := simulationFrame{Kind: simulationFrameResponse, Request: 99, Node: "node", Incarnation: 1}
			if test.duplicate {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				result := make(chan simulationModelResult, 1)
				go func() {
					frame, err := transport.exchange(ctx, simulationFrame{Kind: simulationFrameModel, Node: "node", Incarnation: 1})
					result <- simulationModelResult{frame: frame, err: err}
				}()
				var err error
				frame, err = simulationProgressReadFrame(requestRead)
				if err != nil {
					t.Fatal(err)
				}
				frame.Kind = simulationFrameResponse
				if test.abandoned {
					cancel()
					if got := simulationProgressReceive(t, result); !errors.Is(got.err, context.Canceled) {
						t.Fatalf("cancelled operation = %#v", got)
					}
				}
				if err := simulationProgressWriteFrame(responseWrite, frame); err != nil {
					t.Fatal(err)
				}
				if !test.abandoned {
					if got := simulationProgressReceive(t, result); got.err != nil {
						t.Fatal(got.err)
					}
				}
				wantCallback := "arrived"
				if test.abandoned {
					wantCallback = "discarded"
				}
				if callback := simulationProgressReceive(t, callbacks); callback != wantCallback {
					t.Fatalf("valid response callback = %q, want %q", callback, wantCallback)
				}
			}
			if err := simulationProgressWriteFrame(responseWrite, frame); err != nil {
				t.Fatal(err)
			}
			simulationProgressReceive(t, transport.done)
			if _, err := transport.exchange(context.Background(), simulationFrame{Kind: simulationFrameModel}); err == nil || err.Error() != "simulation model response identity is unknown" {
				t.Fatalf("invalid response error = %v", err)
			}
			select {
			case callback := <-callbacks:
				t.Fatalf("invalid response invoked %q", callback)
			default:
			}
		})
	}
}

func TestSimulationModelProgressUnknownDiscardAcknowledgementKeepsDeliveryRunnable(t *testing.T) {
	coordinator, node := simulationProgressCoordinator(t)
	requestRead, responseWrite, discarded := simulationProgressTransport(t, coordinator)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := coordinator.handleNodeFrame(ctx, node, simulationFrame{Kind: simulationFrameModel, Node: "node", Incarnation: 1})
		result <- err
	}()
	request, err := simulationProgressReadFrame(requestRead)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if err := simulationProgressReceive(t, result); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled operation error = %v", err)
	}
	coordinator.removeNode(node)
	request.Kind, request.Arrivals = simulationFrameResponse, 2
	if err := simulationProgressWriteFrame(responseWrite, request); err != nil {
		t.Fatal(err)
	}
	simulationProgressReceive(t, discarded)
	simulationProgressReceive(t, coordinator.model.done)
	if _, err := coordinator.model.exchange(context.Background(), simulationFrame{Kind: simulationFrameModel}); err == nil || !strings.Contains(err.Error(), "acknowledged unknown external work") {
		t.Fatalf("unknown discard acknowledgement error = %v", err)
	}
	response, err := coordinator.handleCoordinatorTime(context.Background(), simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
	})
	simulationProgressResponse(t, response, err, 1, simulationTimeRetry, simulationInitialTime)
}
