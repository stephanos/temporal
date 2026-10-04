//go:build unix

package execution

import (
	"context"
	"testing"
)

func TestSimulationTimeLifecycleSuspendedWaitCompletionIsReservedExactlyOnce(t *testing.T) {
	for _, resume := range []bool{false, true} {
		name := "deliver-suspended"
		if resume {
			name = "resume-before-delivery"
		}
		t.Run(name, func(t *testing.T) {
			coordinator, node := simulationProgressCoordinator(t)
			frame := simulationFrame{Kind: simulationFrameWait, Request: 7}
			if err := coordinator.handleWaitAcceptance(frame); err != nil {
				t.Fatal(err)
			}
			if err := coordinator.time.progress.apply(coordinatorWaitSuspended{participant: coordinator.coordinator, request: 7}); err != nil {
				t.Fatal(err)
			}
			coordinator.removeNode(node)
			response, err := coordinator.handleCoordinatorTime(context.Background(), simulationTimeRequest{
				Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
			})
			simulationProgressResponse(t, response, err, 1, simulationTimeAdvance, simulationInitialTime+10)
			for range 2 {
				if err := coordinator.time.progress.apply(coordinatorCompletionReserved{participant: coordinator.coordinator, request: 7}); err != nil {
					t.Fatal(err)
				}
			}
			if resume {
				for range 2 {
					if err := coordinator.time.progress.apply(coordinatorWaitResumed{participant: coordinator.coordinator, request: 7}); err != nil {
						t.Fatal(err)
					}
				}
			}
			coordinator.handleCoordinatorDelivery(frame)
			response, err = coordinator.handleCoordinatorTime(context.Background(), simulationTimeRequest{
				Generation: 2, Current: simulationInitialTime + 10, Deadline: simulationInitialTime + 20,
			})
			simulationProgressResponse(t, response, err, 2, simulationTimeRetry, simulationInitialTime+10)
			response, err = coordinator.handleCoordinatorTime(context.Background(), simulationTimeRequest{
				Generation: 3, Current: simulationInitialTime + 10, Deadline: simulationInitialTime + 20, Arrivals: 1,
			})
			simulationProgressResponse(t, response, err, 3, simulationTimeAdvance, simulationInitialTime+20)
		})
	}
}

func TestSimulationTimeLifecycleCompletionWithoutAdmissionKeepsDeliveredCredit(t *testing.T) {
	coordinator, node := simulationProgressCoordinator(t)
	coordinator.removeNode(node)
	for range 2 {
		if err := coordinator.time.progress.apply(coordinatorCompletionReserved{participant: coordinator.coordinator, request: 11}); err != nil {
			t.Fatal(err)
		}
	}
	if err := coordinator.time.progress.apply(coordinatorResponseDelivered{participant: coordinator.coordinator, request: 12}); err == nil {
		t.Fatal("unknown response was delivered")
	}
	if err := coordinator.time.progress.apply(coordinatorResponseDelivered{participant: coordinator.coordinator, request: 11}); err != nil {
		t.Fatal(err)
	}
	response, err := coordinator.handleCoordinatorTime(context.Background(), simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
	})
	simulationProgressResponse(t, response, err, 1, simulationTimeRetry, simulationInitialTime)
	if err := coordinator.time.progress.apply(coordinatorResponseDelivered{participant: coordinator.coordinator, request: 11}); err == nil {
		t.Fatal("duplicate response was delivered")
	}
	response, err = coordinator.handleCoordinatorTime(context.Background(), simulationTimeRequest{
		Generation: 2, Current: simulationInitialTime, Deadline: simulationInitialTime + 10, Arrivals: 1,
	})
	simulationProgressResponse(t, response, err, 2, simulationTimeAdvance, simulationInitialTime+10)
}

func TestSimulationTimeLifecycleRejectedResponseStepsLeaveWaiterInstalled(t *testing.T) {
	for _, test := range []struct {
		name  string
		event func(*simulationTimeParticipant) simulationProgressEvent
	}{
		{"unknown-suspension", func(p *simulationTimeParticipant) simulationProgressEvent {
			return coordinatorWaitSuspended{participant: p, request: 1}
		}},
		{"unknown-resumption", func(p *simulationTimeParticipant) simulationProgressEvent {
			return coordinatorWaitResumed{participant: p, request: 1}
		}},
		{"unknown-delivery", func(p *simulationTimeParticipant) simulationProgressEvent {
			return coordinatorResponseDelivered{participant: p, request: 1}
		}},
		{"unknown-discard-credit", func(p *simulationTimeParticipant) simulationProgressEvent {
			return modelAbandonedResponseDiscarded{coordinator: p, arrivals: 1}
		}},
		{"unreserved-dispatch", func(p *simulationTimeParticipant) simulationProgressEvent {
			return modelRequestDispatched{coordinator: p}
		}},
		{"unreserved-unavailable-dispatch", func(p *simulationTimeParticipant) simulationProgressEvent {
			return modelDispatchUnavailable{coordinator: p}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			coordinator, node := simulationProgressCoordinator(t)
			results := simulationProgressWait(t, coordinator.time, coordinator.coordinator, simulationTimeRequest{
				Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
			})
			for range 2 {
				if err := coordinator.time.progress.apply(test.event(coordinator.coordinator)); err == nil {
					t.Fatal("invalid response step was accepted")
				}
				if !coordinator.time.isQuiescent(coordinator.coordinator) {
					t.Fatal("rejected response step woke the installed waiter")
				}
			}
			response, err := coordinator.time.quiesce(context.Background(), node.time, simulationTimeRequest{
				Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 20,
			})
			simulationProgressResponse(t, response, err, 1, simulationTimeAdvance, simulationInitialTime+10)
			result := simulationProgressReceive(t, results)
			simulationProgressResponse(t, result.response, result.err, 1, simulationTimeAdvance, simulationInitialTime+10)
		})
	}
}

func TestSimulationTimeLifecycleDuplicateControlForwardPreservesBothParticipants(t *testing.T) {
	coordinator, node := simulationProgressCoordinator(t)
	frame := simulationFrame{Request: 1}
	if _, err := coordinator.handle(context.Background(), simulationFrame{Kind: simulationFrameExplorationPlan, Request: 2}); err != nil {
		t.Fatal(err)
	}
	coordinator.handleCoordinatorDelivery(simulationFrame{Request: 2})
	if err := coordinator.acceptNodeControl(frame, node); err != nil {
		t.Fatal(err)
	}
	frame.Arrivals = 1
	if err := coordinator.acceptNodeControl(frame, node); err == nil {
		t.Fatal("duplicate control was forwarded")
	}
	response, err := coordinator.handleCoordinatorTime(context.Background(), simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 20,
	})
	simulationProgressResponse(t, response, err, 1, simulationTimeRetry, simulationInitialTime)
	if err := coordinator.time.progress.apply(participantResponseDelivered{participant: node.time}); err != nil {
		t.Fatal(err)
	}
	coordinator.handleCoordinatorDelivery(frame)
	results := simulationProgressWait(t, coordinator.time, coordinator.coordinator, simulationTimeRequest{
		Generation: 2, Current: simulationInitialTime, Deadline: simulationInitialTime + 20, Arrivals: 2,
	})
	response, err = coordinator.time.quiesce(context.Background(), node.time, simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10, Arrivals: 1,
	})
	simulationProgressResponse(t, response, err, 1, simulationTimeAdvance, simulationInitialTime+10)
	result := simulationProgressReceive(t, results)
	simulationProgressResponse(t, result.response, result.err, 2, simulationTimeAdvance, simulationInitialTime+10)
}

func TestSimulationModelLifecycleRequestNamespaceIsIndependentOfCoordinatorResponses(t *testing.T) {
	coordinator, node := simulationProgressCoordinator(t)
	control := simulationFrame{Kind: simulationFrameExplorationPlan, Request: 1}
	if _, err := coordinator.handle(context.Background(), control); err != nil {
		t.Fatal(err)
	}
	requestRead, responseWrite, _ := simulationProgressTransport(t, coordinator)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	results := make(chan simulationModelResult, 1)
	go func() {
		response, err := coordinator.handleNodeFrame(ctx, node, simulationFrame{Kind: simulationFrameModel, Node: "node", Incarnation: 1, Payload: []byte("model")})
		results <- simulationModelResult{frame: response, err: err}
	}()
	request, err := simulationProgressReadFrame(requestRead)
	if err != nil {
		t.Fatal(err)
	}
	if request.Request != 1 {
		t.Fatalf("first model request = %d, want independent correlation 1", request.Request)
	}
	request.Kind, request.Arrivals = simulationFrameResponse, 1
	if err := simulationProgressWriteFrame(responseWrite, request); err != nil {
		t.Fatal(err)
	}
	result := simulationProgressReceive(t, results)
	if result.err != nil || string(result.frame.Payload) != "model" {
		t.Fatalf("model result = %#v, error = %v", result.frame, result.err)
	}
	coordinator.handleCoordinatorDelivery(control)
	response, err := coordinator.handleCoordinatorTime(ctx, simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 20,
	})
	simulationProgressResponse(t, response, err, 1, simulationTimeRetry, simulationInitialTime)
	nodeResults := simulationProgressWait(t, coordinator.time, node.time, simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10, Arrivals: 1,
	})
	response, err = coordinator.handleCoordinatorTime(ctx, simulationTimeRequest{
		Generation: 2, Current: simulationInitialTime, Deadline: simulationInitialTime + 20, Arrivals: 1,
	})
	simulationProgressResponse(t, response, err, 2, simulationTimeAdvance, simulationInitialTime+10)
	nodeResult := simulationProgressReceive(t, nodeResults)
	simulationProgressResponse(t, nodeResult.response, nodeResult.err, 1, simulationTimeAdvance, simulationInitialTime+10)
}
