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
	arbiter := newSimulationTimeArbiter()
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

func TestSimulationTimeArbiterCancelsAnEpochWhenExternalWorkArrives(t *testing.T) {
	arbiter := newSimulationTimeArbiter()
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
	arbiter.runnable(node)

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
	arbiter := newSimulationTimeArbiter()
	coordinator, err := arbiter.register("coordinator")
	if err != nil {
		t.Fatal(err)
	}
	participant, err := arbiter.register("node/1")
	if err != nil {
		t.Fatal(err)
	}
	arbiter.activate(coordinator)
	arbiter.beginExternal(coordinator)

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
	arbiter.endExternal(coordinator)

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
	arbiter := newSimulationTimeArbiter()
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
	arbiter.beginExternal(coordinator)

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
	arbiter.endExternal(coordinator)
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
	arbiter := newSimulationTimeArbiter()
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
	arbiter.beginExternal(node)

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
	arbiter := newSimulationTimeArbiter()
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
	if err := arbiter.beginExternalAfterArrivals(node, 0); err != nil {
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
	arbiter := newSimulationTimeArbiter()
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
	if err := arbiter.forwardExternalAfterArrivals(node, 0, coordinator); err != nil {
		t.Fatal(err)
	}
	arbiter.deliverExternal(coordinator)

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
	arbiter := newSimulationTimeArbiter()
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
	if err := arbiter.forwardExternalAfterArrivals(node, 0, coordinator); err != nil {
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
	arbiter := newSimulationTimeArbiter()
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
	arbiter.beginExternal(coordinator)
	arbiter.deliverExternal(coordinator)
	arbiter.beginExternal(node)

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
	if err := arbiter.transferExternalArrival(coordinator, 1, node); err != nil {
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
	arbiter := newSimulationTimeArbiter()
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
	arbiter.beginExternal(node)
	arbiter.deliverExternal(node)

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
	arbiter := newSimulationTimeArbiter()
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
	arbiter.beginExternal(participant)
	wantExternal := simulationTimeResponse{Generation: 1, Kind: simulationTimeExternal, Time: simulationInitialTime}
	if response := <-responses; response != wantExternal {
		t.Fatalf("external response = %#v, want %#v", response, wantExternal)
	}
	if err := <-errors; err != nil {
		t.Fatal(err)
	}
	arbiter.deliverExternal(participant)

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
	arbiter := newSimulationTimeArbiter()
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
