package execution

import (
	"bytes"
	"context"
	"runtime"
	"testing"
	"time"
)

func TestSimulationTimeWireRoundTripsFixedFrames(t *testing.T) {
	request := simulationTimeRequest{Generation: 7, Current: simulationInitialTime + 11, Deadline: simulationInitialTime + 23, Arrivals: 2}
	encodedRequest, err := encodeSimulationTimeRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if len(encodedRequest) != simulationTimeRequestBytes {
		t.Fatalf("encoded request bytes = %d", len(encodedRequest))
	}
	decodedRequest, err := decodeSimulationTimeRequest(encodedRequest)
	if err != nil {
		t.Fatal(err)
	}
	if decodedRequest != request {
		t.Fatalf("decoded request = %#v, want %#v", decodedRequest, request)
	}

	response := simulationTimeResponse{Generation: 7, Kind: simulationTimeAdvance, Time: simulationInitialTime + 23}
	encodedResponse, err := encodeSimulationTimeResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	if len(encodedResponse) != simulationTimeResponseBytes {
		t.Fatalf("encoded response bytes = %d", len(encodedResponse))
	}
	decodedResponse, err := decodeSimulationTimeResponse(encodedResponse)
	if err != nil {
		t.Fatal(err)
	}
	if decodedResponse != response {
		t.Fatalf("decoded response = %#v, want %#v", decodedResponse, response)
	}

	encodedResponse[0] = 0
	_, err = decodeSimulationTimeResponse(encodedResponse)
	if err == nil {
		t.Fatal("decode accepted a changed response magic")
	}
}

func TestServeSimulationTimeExchangesBoundedFrames(t *testing.T) {
	request := simulationTimeRequest{Generation: 3, Current: simulationInitialTime, Deadline: simulationInitialTime + 5}
	encoded, err := encodeSimulationTimeRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	var destination bytes.Buffer
	err = serveSimulationTime(context.Background(), bytes.NewReader(encoded), &destination, func(_ context.Context, actual simulationTimeRequest) (simulationTimeResponse, error) {
		if actual != request {
			t.Fatalf("request = %#v, want %#v", actual, request)
		}
		return simulationTimeResponse{Generation: actual.Generation, Kind: simulationTimeAdvance, Time: actual.Deadline}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	response, err := decodeSimulationTimeResponse(destination.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	want := simulationTimeResponse{Generation: 3, Kind: simulationTimeAdvance, Time: simulationInitialTime + 5}
	if response != want {
		t.Fatalf("response = %#v, want %#v", response, want)
	}
}

func TestSimulationTimeArbiterAdvancesEveryParticipantToEarliestDeadline(t *testing.T) {
	arbiter := newSimulationTimeArbiter(false)
	coordinator, err := arbiter.register("coordinator")
	if err != nil {
		t.Fatal(err)
	}
	node, err := arbiter.register("node/1")
	if err != nil {
		t.Fatal(err)
	}
	if current := arbiter.activate(coordinator); current != simulationInitialTime {
		t.Fatalf("coordinator activation time = %d", current)
	}
	if current := arbiter.activate(node); current != simulationInitialTime {
		t.Fatalf("node activation time = %d", current)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	responses := make(chan simulationTimeResponse, 2)
	errors := make(chan error, 2)
	go func() {
		response, quiesceErr := arbiter.quiesce(ctx, coordinator, simulationTimeRequest{
			Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 20,
		})
		responses <- response
		errors <- quiesceErr
	}()
	go func() {
		response, quiesceErr := arbiter.quiesce(ctx, node, simulationTimeRequest{
			Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
		})
		responses <- response
		errors <- quiesceErr
	}()

	first := <-responses
	second := <-responses
	if err := <-errors; err != nil {
		t.Fatal(err)
	}
	if err := <-errors; err != nil {
		t.Fatal(err)
	}
	want := simulationTimeResponse{Generation: 1, Kind: simulationTimeAdvance, Time: simulationInitialTime + 10}
	if first != want || second != want {
		t.Fatalf("responses = %#v, %#v, want %#v", first, second, want)
	}
	if current := arbiter.currentTime(); current != simulationInitialTime+10 {
		t.Fatalf("current time = %d", current)
	}
}

func TestSimulationTimeArbiterAcceptsForwardTickEpoch(t *testing.T) {
	arbiter := newSimulationTimeArbiter(true)
	participant, err := arbiter.register("coordinator")
	if err != nil {
		t.Fatal(err)
	}
	arbiter.activate(participant)

	response, err := arbiter.quiesce(context.Background(), participant, simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime + 100, Deadline: simulationInitialTime + 110,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := simulationTimeResponse{Generation: 1, Kind: simulationTimeAdvance, Time: simulationInitialTime + 110}
	if response != want {
		t.Fatalf("response = %#v, want %#v", response, want)
	}
}

func TestSimulationTimeArbiterForwardActivationAdoptsCurrent(t *testing.T) {
	arbiter := newSimulationTimeArbiter(true)
	participant, err := arbiter.register("node/1")
	if err != nil {
		t.Fatal(err)
	}

	current := simulationInitialTime + 100
	activated, err := arbiter.activateAt(participant, current)
	if err != nil {
		t.Fatal(err)
	}
	if activated != current {
		t.Fatalf("activation time = %d, want %d", activated, current)
	}
	if !participant.active {
		t.Fatal("participant was not activated")
	}
	if cluster := arbiter.currentTime(); cluster != current {
		t.Fatalf("cluster time = %d, want %d", cluster, current)
	}
}

func TestSimulationTimeArbiterStrictActivationRejectsFutureCurrent(t *testing.T) {
	arbiter := newSimulationTimeArbiter(false)
	participant, err := arbiter.register("node/1")
	if err != nil {
		t.Fatal(err)
	}

	_, err = arbiter.activateAt(participant, simulationInitialTime+100)
	if err == nil {
		t.Fatal("strict arbiter accepted a future activation epoch")
	}
	if participant.active {
		t.Fatal("participant activated after the request was rejected")
	}
	if current := arbiter.currentTime(); current != simulationInitialTime {
		t.Fatalf("strict arbiter changed time to %d after rejecting activation", current)
	}
}

func TestSimulationTimeArbiterForwardTickEpochIsOrderIndependent(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(map[bool]string{false: "lower-first", true: "higher-first"}[reverse], func(t *testing.T) {
			arbiter := newSimulationTimeArbiter(true)
			lower, err := arbiter.register("node/lower")
			if err != nil {
				t.Fatal(err)
			}
			higher, err := arbiter.register("node/higher")
			if err != nil {
				t.Fatal(err)
			}
			arbiter.activate(lower)
			arbiter.activate(higher)

			type request struct {
				participant *simulationTimeParticipant
				current     int64
				deadline    int64
			}
			requests := []request{
				{participant: lower, current: simulationInitialTime + 11, deadline: simulationInitialTime + 50},
				{participant: higher, current: simulationInitialTime + 17, deadline: simulationInitialTime + 40},
			}
			if reverse {
				requests[0], requests[1] = requests[1], requests[0]
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			responses := make(chan simulationTimeResponse, 2)
			errors := make(chan error, 2)
			quiesce := func(current request) {
				response, quiesceErr := arbiter.quiesce(ctx, current.participant, simulationTimeRequest{
					Generation: 1, Current: current.current, Deadline: current.deadline,
				})
				responses <- response
				errors <- quiesceErr
			}
			go quiesce(requests[0])
			waitForSimulationQuiescence(t, arbiter, requests[0].participant)
			select {
			case response := <-responses:
				t.Fatalf("first participant advanced before the other quiesced: %#v", response)
			default:
			}
			go quiesce(requests[1])

			first := <-responses
			second := <-responses
			if err := <-errors; err != nil {
				t.Fatal(err)
			}
			if err := <-errors; err != nil {
				t.Fatal(err)
			}
			wantTime := simulationInitialTime + 40
			if first.Kind != simulationTimeAdvance || second.Kind != simulationTimeAdvance || first.Time != wantTime || second.Time != wantTime {
				t.Fatalf("responses = %#v, %#v, want advance to %d", first, second, wantTime)
			}
		})
	}
}

func TestSimulationTimeArbiterRejectsForwardTickEpochPastItsDeadline(t *testing.T) {
	arbiter := newSimulationTimeArbiter(true)
	participant, err := arbiter.register("coordinator")
	if err != nil {
		t.Fatal(err)
	}
	arbiter.activate(participant)

	_, err = arbiter.quiesce(context.Background(), participant, simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime + 100, Deadline: simulationInitialTime + 99,
	})
	if err == nil {
		t.Fatal("arbiter accepted a forward-tick epoch past its deadline")
	}
	if current := arbiter.currentTime(); current != simulationInitialTime {
		t.Fatalf("arbiter changed time to %d after rejecting the request", current)
	}
}

func TestSimulationTimeArbiterCancelsAnEpochWhenExternalWorkArrives(t *testing.T) {
	arbiter := newSimulationTimeArbiter(false)
	coordinator, err := arbiter.register("coordinator")
	if err != nil {
		t.Fatal(err)
	}
	node, err := arbiter.register("node/1")
	if err != nil {
		t.Fatal(err)
	}
	arbiter.activate(coordinator)
	arbiter.activate(node)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	responses := make(chan simulationTimeResponse, 1)
	errors := make(chan error, 1)
	go func() {
		response, quiesceErr := arbiter.quiesce(ctx, node, simulationTimeRequest{
			Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
		})
		responses <- response
		errors <- quiesceErr
	}()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for !arbiter.isQuiescent(node) {
		select {
		case <-timer.C:
			t.Fatal("node did not enter the time epoch")
		default:
			runtime.Gosched()
		}
	}
	_ = arbiter.progress.apply(participantRunnable{participant: node})

	want := simulationTimeResponse{Generation: 1, Kind: simulationTimeRetry, Time: simulationInitialTime}
	if response := <-responses; response != want {
		t.Fatalf("response = %#v, want %#v", response, want)
	}
	if err := <-errors; err != nil {
		t.Fatal(err)
	}
	if current := arbiter.currentTime(); current != simulationInitialTime {
		t.Fatalf("current time = %d", current)
	}
}

func TestSimulationTimeArbiterDoesNotAdvancePastExternalWorkOrInactiveParticipants(t *testing.T) {
	arbiter := newSimulationTimeArbiter(false)
	coordinator, err := arbiter.register("coordinator")
	if err != nil {
		t.Fatal(err)
	}
	participant, err := arbiter.register("node/1")
	if err != nil {
		t.Fatal(err)
	}
	arbiter.activate(coordinator)
	simulationPendingExternal(t, arbiter, coordinator)

	response, err := arbiter.quiesce(context.Background(), coordinator, simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	wantExternal := simulationTimeResponse{Generation: 1, Kind: simulationTimeExternal, Time: simulationInitialTime}
	if response != wantExternal {
		t.Fatalf("external response = %#v, want %#v", response, wantExternal)
	}
	simulationFinishExternal(t, arbiter, coordinator)

	responses := make(chan simulationTimeResponse, 2)
	errors := make(chan error, 2)
	go func() {
		response, quiesceErr := arbiter.quiesce(context.Background(), coordinator, simulationTimeRequest{
			Generation: 2, Current: simulationInitialTime, Deadline: simulationInitialTime + 20,
		})
		responses <- response
		errors <- quiesceErr
	}()
	waitForSimulationQuiescence(t, arbiter, coordinator)
	select {
	case response := <-responses:
		t.Fatalf("advanced with an inactive participant: %#v", response)
	default:
	}

	arbiter.activate(participant)
	go func() {
		response, quiesceErr := arbiter.quiesce(context.Background(), participant, simulationTimeRequest{
			Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
		})
		responses <- response
		errors <- quiesceErr
	}()

	first := <-responses
	second := <-responses
	if err := <-errors; err != nil {
		t.Fatal(err)
	}
	if err := <-errors; err != nil {
		t.Fatal(err)
	}
	wantTime := simulationInitialTime + 10
	if first.Kind != simulationTimeAdvance || second.Kind != simulationTimeAdvance || first.Time != wantTime || second.Time != wantTime {
		t.Fatalf("responses = %#v, %#v, want advance to %d", first, second, wantTime)
	}
}

func TestSimulationTimeArbiterExcludesAnExternallyBlockedParticipant(t *testing.T) {
	arbiter := newSimulationTimeArbiter(false)
	coordinator, err := arbiter.register("coordinator")
	if err != nil {
		t.Fatal(err)
	}
	node, err := arbiter.register("node/1")
	if err != nil {
		t.Fatal(err)
	}
	arbiter.activate(coordinator)
	arbiter.activate(node)
	simulationPendingExternal(t, arbiter, coordinator)

	response, err := arbiter.quiesce(context.Background(), node, simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := simulationTimeResponse{Generation: 1, Kind: simulationTimeAdvance, Time: simulationInitialTime + 10}
	if response != want {
		t.Fatalf("response = %#v, want %#v", response, want)
	}
	simulationFinishExternal(t, arbiter, coordinator)
	response, err = arbiter.quiesce(context.Background(), coordinator, simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if response != want {
		t.Fatalf("catch-up response = %#v, want %#v", response, want)
	}
}

func TestSimulationTimeArbiterSettlesWhenLastRunnableParticipantBlocksExternally(t *testing.T) {
	arbiter := newSimulationTimeArbiter(false)
	coordinator, err := arbiter.register("coordinator")
	if err != nil {
		t.Fatal(err)
	}
	node, err := arbiter.register("node/1")
	if err != nil {
		t.Fatal(err)
	}
	arbiter.activate(coordinator)
	arbiter.activate(node)

	responses := make(chan simulationTimeResponse, 1)
	errors := make(chan error, 1)
	go func() {
		response, quiesceErr := arbiter.quiesce(context.Background(), coordinator, simulationTimeRequest{
			Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
		})
		responses <- response
		errors <- quiesceErr
	}()
	waitForSimulationQuiescence(t, arbiter, coordinator)
	simulationPendingExternal(t, arbiter, node)

	if current := arbiter.currentTime(); current != simulationInitialTime+10 {
		t.Fatalf("current time = %d", current)
	}
	want := simulationTimeResponse{Generation: 1, Kind: simulationTimeAdvance, Time: simulationInitialTime + 10}
	if response := <-responses; response != want {
		t.Fatalf("response = %#v, want %#v", response, want)
	}
	if err := <-errors; err != nil {
		t.Fatal(err)
	}
}

func TestSimulationTimeArbiterDoesNotSettleWhileExternalRequestIsHandled(t *testing.T) {
	arbiter := newSimulationTimeArbiter(false)
	coordinator, err := arbiter.register("coordinator")
	if err != nil {
		t.Fatal(err)
	}
	node, err := arbiter.register("node/1")
	if err != nil {
		t.Fatal(err)
	}
	arbiter.activate(coordinator)
	arbiter.activate(node)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if err := arbiter.progress.apply(participantRequestAccepted{participant: node, arrivals: 0}); err != nil {
		t.Fatal(err)
	}
	responses := make(chan simulationTimeResponse, 1)
	errors := make(chan error, 1)
	go func() {
		response, quiesceErr := arbiter.quiesce(ctx, coordinator, simulationTimeRequest{
			Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
		})
		responses <- response
		errors <- quiesceErr
	}()
	waitForSimulationQuiescence(t, arbiter, coordinator)
	if current := arbiter.currentTime(); current != simulationInitialTime {
		t.Fatalf("current time = %d", current)
	}
	cancel()
	<-responses
	if err := <-errors; err == nil {
		t.Fatal("quiescence did not observe cancellation")
	}
}

func TestSimulationTimeArbiterAdvancesAfterForwardedRequestIsDelivered(t *testing.T) {
	arbiter := newSimulationTimeArbiter(false)
	coordinator, err := arbiter.register("coordinator")
	if err != nil {
		t.Fatal(err)
	}
	node, err := arbiter.register("node/1")
	if err != nil {
		t.Fatal(err)
	}
	arbiter.activate(coordinator)
	arbiter.activate(node)
	if err := arbiter.progress.apply(modelRequestForwarded{source: node, arrivals: 0, current: 0, destination: coordinator}); err != nil {
		t.Fatal(err)
	}
	_ = arbiter.progress.apply(participantResponseDelivered{participant: coordinator})

	response, err := arbiter.quiesce(context.Background(), coordinator, simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10, Arrivals: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := simulationTimeResponse{Generation: 1, Kind: simulationTimeAdvance, Time: simulationInitialTime + 10}
	if response != want {
		t.Fatalf("response = %#v, want %#v", response, want)
	}
}

func TestSimulationTimeArbiterForwardsExternalRequestAtomically(t *testing.T) {
	arbiter := newSimulationTimeArbiter(false)
	coordinator, err := arbiter.register("coordinator")
	if err != nil {
		t.Fatal(err)
	}
	node, err := arbiter.register("node/1")
	if err != nil {
		t.Fatal(err)
	}
	arbiter.activate(coordinator)
	arbiter.activate(node)

	responses := make(chan simulationTimeResponse, 1)
	errors := make(chan error, 1)
	go func() {
		response, quiesceErr := arbiter.quiesce(context.Background(), coordinator, simulationTimeRequest{
			Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
		})
		responses <- response
		errors <- quiesceErr
	}()
	waitForSimulationQuiescence(t, arbiter, coordinator)
	if err := arbiter.progress.apply(modelRequestForwarded{source: node, arrivals: 0, current: 0, destination: coordinator}); err != nil {
		t.Fatal(err)
	}
	if current := arbiter.currentTime(); current != simulationInitialTime {
		t.Fatalf("current time = %d", current)
	}
	want := simulationTimeResponse{Generation: 1, Kind: simulationTimeExternal, Time: simulationInitialTime}
	if response := <-responses; response != want {
		t.Fatalf("response = %#v, want %#v", response, want)
	}
	if err := <-errors; err != nil {
		t.Fatal(err)
	}
}

func TestSimulationTimeArbiterTransfersExternalArrivalAtomically(t *testing.T) {
	arbiter := newSimulationTimeArbiter(false)
	coordinator, err := arbiter.register("coordinator")
	if err != nil {
		t.Fatal(err)
	}
	node, err := arbiter.register("node/1")
	if err != nil {
		t.Fatal(err)
	}
	observer, err := arbiter.register("observer/1")
	if err != nil {
		t.Fatal(err)
	}
	arbiter.activate(coordinator)
	arbiter.activate(node)
	arbiter.activate(observer)
	simulationPendingExternal(t, arbiter, coordinator)
	_ = arbiter.progress.apply(participantResponseDelivered{participant: coordinator})
	simulationPendingExternal(t, arbiter, node)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	responses := make(chan simulationTimeResponse, 1)
	errors := make(chan error, 1)
	go func() {
		response, quiesceErr := arbiter.quiesce(ctx, observer, simulationTimeRequest{
			Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10,
		})
		responses <- response
		errors <- quiesceErr
	}()
	waitForSimulationQuiescence(t, arbiter, observer)
	if err := arbiter.progress.apply(modelResponseArrived{coordinator: coordinator, arrivals: 1, node: node}); err != nil {
		t.Fatal(err)
	}
	if current := arbiter.currentTime(); current != simulationInitialTime {
		t.Fatalf("current time = %d", current)
	}
	cancel()
	<-responses
	if err := <-errors; err == nil {
		t.Fatal("quiescence did not observe cancellation")
	}
}

func TestSimulationTimeArbiterWaitsForDeliveredExternalWorkToBeConsumed(t *testing.T) {
	arbiter := newSimulationTimeArbiter(false)
	coordinator, err := arbiter.register("coordinator")
	if err != nil {
		t.Fatal(err)
	}
	node, err := arbiter.register("node/1")
	if err != nil {
		t.Fatal(err)
	}
	arbiter.activate(coordinator)
	arbiter.activate(node)
	simulationPendingExternal(t, arbiter, node)
	_ = arbiter.progress.apply(participantResponseDelivered{participant: node})

	responses := make(chan simulationTimeResponse, 2)
	errors := make(chan error, 2)
	go func() {
		response, quiesceErr := arbiter.quiesce(context.Background(), coordinator, simulationTimeRequest{
			Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 20,
		})
		responses <- response
		errors <- quiesceErr
	}()
	waitForSimulationQuiescence(t, arbiter, coordinator)
	select {
	case response := <-responses:
		t.Fatalf("advanced before delivered work was consumed: %#v", response)
	default:
	}

	go func() {
		response, quiesceErr := arbiter.quiesce(context.Background(), node, simulationTimeRequest{
			Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 10, Arrivals: 1,
		})
		responses <- response
		errors <- quiesceErr
	}()
	first := <-responses
	second := <-responses
	if err := <-errors; err != nil {
		t.Fatal(err)
	}
	if err := <-errors; err != nil {
		t.Fatal(err)
	}
	wantTime := simulationInitialTime + 10
	if first.Kind != simulationTimeAdvance || second.Kind != simulationTimeAdvance || first.Time != wantTime || second.Time != wantTime {
		t.Fatalf("responses = %#v, %#v, want advance to %d", first, second, wantTime)
	}
}

func TestSimulationTimeArbiterRejoinsAfterExternalWorkArrives(t *testing.T) {
	arbiter := newSimulationTimeArbiter(false)
	participant, err := arbiter.register("coordinator")
	if err != nil {
		t.Fatal(err)
	}
	blocker, err := arbiter.register("node/1")
	if err != nil {
		t.Fatal(err)
	}
	arbiter.activate(participant)
	arbiter.activate(blocker)

	responses := make(chan simulationTimeResponse, 2)
	errors := make(chan error, 2)
	go func() {
		response, quiesceErr := arbiter.quiesce(context.Background(), participant, simulationTimeRequest{
			Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 20,
		})
		responses <- response
		errors <- quiesceErr
	}()
	waitForSimulationQuiescence(t, arbiter, participant)
	simulationPendingExternal(t, arbiter, participant)
	wantExternal := simulationTimeResponse{Generation: 1, Kind: simulationTimeExternal, Time: simulationInitialTime}
	if response := <-responses; response != wantExternal {
		t.Fatalf("external response = %#v, want %#v", response, wantExternal)
	}
	if err := <-errors; err != nil {
		t.Fatal(err)
	}
	_ = arbiter.progress.apply(participantResponseDelivered{participant: participant})

	go func() {
		response, quiesceErr := arbiter.quiesce(context.Background(), participant, simulationTimeRequest{
			Generation: 2, Current: simulationInitialTime, Deadline: simulationInitialTime + 10, Arrivals: 1,
		})
		responses <- response
		errors <- quiesceErr
	}()
	waitForSimulationQuiescence(t, arbiter, participant)
	response, err := arbiter.quiesce(context.Background(), blocker, simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 30,
	})
	if err != nil {
		t.Fatal(err)
	}
	other := <-responses
	if err := <-errors; err != nil {
		t.Fatal(err)
	}
	if response.Kind != simulationTimeAdvance || other.Kind != simulationTimeAdvance || response.Time != simulationInitialTime+10 || other.Time != simulationInitialTime+10 {
		t.Fatalf("responses = %#v, %#v", response, other)
	}
}

func TestSimulationTimeArbiterActivatesRestartAtCurrentTime(t *testing.T) {
	arbiter := newSimulationTimeArbiter(false)
	coordinator, err := arbiter.register("coordinator")
	if err != nil {
		t.Fatal(err)
	}
	arbiter.activate(coordinator)

	response, err := arbiter.quiesce(context.Background(), coordinator, simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime, Deadline: simulationInitialTime + 17,
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Time != simulationInitialTime+17 {
		t.Fatalf("advance time = %d", response.Time)
	}

	restarted, err := arbiter.register("node/2")
	if err != nil {
		t.Fatal(err)
	}
	if current := arbiter.activate(restarted); current != simulationInitialTime+17 {
		t.Fatalf("restart activation time = %d", current)
	}
}

func waitForSimulationQuiescence(t *testing.T, arbiter *simulationTimeArbiter, participant *simulationTimeParticipant) {
	t.Helper()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for !arbiter.isQuiescent(participant) {
		select {
		case <-timer.C:
			t.Fatal("participant did not enter the time epoch")
		default:
			runtime.Gosched()
		}
	}
}

func TestSimulationTimeArbiterRejectsForwardTickEpochInStrictMode(t *testing.T) {
	arbiter := newSimulationTimeArbiter(false)
	participant, err := arbiter.register("coordinator")
	if err != nil {
		t.Fatal(err)
	}
	arbiter.activate(participant)

	_, err = arbiter.quiesce(context.Background(), participant, simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime + 100, Deadline: simulationInitialTime + 110,
	})
	if err == nil {
		t.Fatal("strict arbiter accepted a forward-tick epoch")
	}
	if current := arbiter.currentTime(); current != simulationInitialTime {
		t.Fatalf("strict arbiter changed time to %d after rejecting the request", current)
	}
}

func TestSimulationTimeArbiterForwardTickEarlyResponsesDoNotRewind(t *testing.T) {
	for _, test := range []struct {
		name string
		set  func(*simulationTimeArbiter, *simulationTimeParticipant)
		kind simulationTimeResponseKind
	}{
		{name: "inactive", set: func(*simulationTimeArbiter, *simulationTimeParticipant) {}, kind: simulationTimeAdvance},
		{name: "delivered", set: func(arbiter *simulationTimeArbiter, participant *simulationTimeParticipant) {
			arbiter.activate(participant)
			simulationPendingExternal(t, arbiter, participant)
			_ = arbiter.progress.apply(participantResponseDelivered{participant: participant})
		}, kind: simulationTimeRetry},
		{name: "external", set: func(arbiter *simulationTimeArbiter, participant *simulationTimeParticipant) {
			arbiter.activate(participant)
			simulationPendingExternal(t, arbiter, participant)
		}, kind: simulationTimeExternal},
	} {
		t.Run(test.name, func(t *testing.T) {
			arbiter := newSimulationTimeArbiter(true)
			participant, err := arbiter.register("node/1")
			if err != nil {
				t.Fatal(err)
			}
			test.set(arbiter, participant)
			current := simulationInitialTime + 100
			response, err := arbiter.quiesce(context.Background(), participant, simulationTimeRequest{
				Generation: 1, Current: current, Deadline: current + 10,
			})
			if err != nil {
				t.Fatal(err)
			}
			if response.Kind != test.kind || response.Time < current {
				t.Fatalf("response = %#v, want kind %d at or after %d", response, test.kind, current)
			}
			if actual := arbiter.currentTime(); actual != current {
				t.Fatalf("current time = %d, want %d", actual, current)
			}
		})
	}
}

func TestSimulationTimeArbiterMalformedForwardTickRequestDoesNotAdvance(t *testing.T) {
	arbiter := newSimulationTimeArbiter(true)
	participant, err := arbiter.register("node/1")
	if err != nil {
		t.Fatal(err)
	}
	arbiter.activate(participant)

	_, err = arbiter.quiesce(context.Background(), participant, simulationTimeRequest{
		Generation: 1, Current: simulationInitialTime + 100, Deadline: simulationInitialTime + 110, Arrivals: 1,
	})
	if err == nil {
		t.Fatal("arbiter accepted a forward-tick request with an unknown arrival")
	}
	if current := arbiter.currentTime(); current != simulationInitialTime {
		t.Fatalf("arbiter changed time to %d after rejecting the request", current)
	}
}

func TestSimulationTimeArbiterAdoptsForwardedModelCurrent(t *testing.T) {
	arbiter := newSimulationTimeArbiter(true)
	source, err := arbiter.register("node/1")
	if err != nil {
		t.Fatal(err)
	}
	destination, err := arbiter.register("coordinator")
	if err != nil {
		t.Fatal(err)
	}
	arbiter.activate(source)
	arbiter.activate(destination)

	current := simulationInitialTime + 100
	if err := arbiter.progress.apply(modelRequestForwarded{source: source, arrivals: 0, current: current, destination: destination}); err != nil {
		t.Fatal(err)
	}
	if actual := arbiter.currentTime(); actual != current {
		t.Fatalf("current time = %d, want %d", actual, current)
	}
}

func TestSimulationTimeArbiterRejectsInvalidForwardedModelCurrentWithoutMutation(t *testing.T) {
	for _, test := range []struct {
		name     string
		forward  bool
		current  int64
		arrivals uint32
	}{
		{name: "strict future", current: simulationInitialTime + 100},
		{name: "unknown arrival", forward: true, current: simulationInitialTime + 100, arrivals: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			arbiter := newSimulationTimeArbiter(test.forward)
			source, err := arbiter.register("node/1")
			if err != nil {
				t.Fatal(err)
			}
			destination, err := arbiter.register("coordinator")
			if err != nil {
				t.Fatal(err)
			}
			arbiter.activate(source)
			arbiter.activate(destination)

			if err := arbiter.progress.apply(modelRequestForwarded{source: source, arrivals: test.arrivals, current: test.current, destination: destination}); err == nil {
				t.Fatal("arbiter accepted an invalid forwarded model current")
			}
			if current := arbiter.currentTime(); current != simulationInitialTime {
				t.Fatalf("arbiter changed time to %d after rejecting the request", current)
			}
			if source.external != 0 || source.delivered != 0 || destination.external != 0 || destination.handling != 0 {
				t.Fatalf("arbiter changed participants after rejecting the request: source=%#v destination=%#v", source, destination)
			}
		})
	}
}

func simulationPendingExternal(t *testing.T, arbiter *simulationTimeArbiter, participant *simulationTimeParticipant) {
	t.Helper()
	destination, err := arbiter.register("pending/" + participant.name)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []simulationProgressEvent{
		coordinatorControlForwarded{participant: participant, node: destination, request: 1},
		participantResponseDelivered{participant: destination},
		arrivalCreditsConsumed{participant: destination, arrivals: 1},
		participantRemoved{participant: destination},
	} {
		if err := arbiter.progress.apply(event); err != nil {
			t.Fatal(err)
		}
	}
}

func simulationFinishExternal(t *testing.T, arbiter *simulationTimeArbiter, participant *simulationTimeParticipant) {
	t.Helper()
	for _, event := range []simulationProgressEvent{
		coordinatorResponseDelivered{participant: participant, request: 1},
		arrivalCreditsConsumed{participant: participant, arrivals: 1},
	} {
		if err := arbiter.progress.apply(event); err != nil {
			t.Fatal(err)
		}
	}
}
