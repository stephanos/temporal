package execution

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestServeSimulationNotifiesAfterWritingResponse(t *testing.T) {
	request := simulationFrame{Profile: simulationProtocol, Kind: simulationFrameStart, Request: 1, Node: "node", Incarnation: 1}
	var source bytes.Buffer
	if err := writeSimulationFrame(&source, request); err != nil {
		t.Fatal(err)
	}
	var destination bytes.Buffer
	responded := false
	delivering := false
	err := serveSimulation(context.Background(), &source, &destination, func(context.Context, simulationFrame) (simulationFrame, error) {
		return simulationFrame{}, nil
	}, nil, func(actual simulationFrame) {
		delivering = true
		if !reflect.DeepEqual(actual, request) {
			t.Fatalf("delivering request = %#v, want %#v", actual, request)
		}
		if destination.Len() != 0 {
			t.Fatal("delivery notification followed the response write")
		}
	}, func(actual simulationFrame) {
		responded = true
		if !delivering {
			t.Fatal("response notification preceded delivery notification")
		}
		if !reflect.DeepEqual(actual, request) {
			t.Fatalf("responded request = %#v, want %#v", actual, request)
		}
		if destination.Len() == 0 {
			t.Fatal("response notification preceded the response write")
		}
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !responded {
		t.Fatal("response notification was omitted")
	}
}

func TestServeSimulationAcknowledgesWaitBeforeHandling(t *testing.T) {
	request := simulationFrame{Profile: simulationProtocol, Kind: simulationFrameWait, Request: 1, Node: "node", Incarnation: 1}
	var source bytes.Buffer
	if err := writeSimulationFrame(&source, request); err != nil {
		t.Fatal(err)
	}
	var destination bytes.Buffer
	accepted := false
	err := serveSimulation(context.Background(), &source, &destination, func(context.Context, simulationFrame) (simulationFrame, error) {
		if !accepted {
			t.Fatal("wait handler ran before acceptance")
		}
		if !reflect.DeepEqual(destination.Bytes(), make([]byte, 4)) {
			t.Fatalf("wait acceptance = %x", destination.Bytes())
		}
		return simulationFrame{}, nil
	}, func(actual simulationFrame) error {
		accepted = true
		if !reflect.DeepEqual(actual, request) {
			t.Fatalf("accepted request = %#v, want %#v", actual, request)
		}
		return nil
	}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !accepted {
		t.Fatal("wait acceptance was omitted")
	}
	response, err := readSimulationFrame(bytes.NewReader(destination.Bytes()[4:]))
	if err != nil {
		t.Fatal(err)
	}
	if response.Kind != simulationFrameResponse || response.Request != request.Request {
		t.Fatalf("wait response = %#v", response)
	}
}

func TestServeSimulationAcceptsArrivalAcknowledgement(t *testing.T) {
	var source bytes.Buffer
	var arrival [8]byte
	binary.BigEndian.PutUint32(arrival[4:], 3)
	if _, err := source.Write(arrival[:]); err != nil {
		t.Fatal(err)
	}
	acknowledged := uint32(0)
	err := serveSimulation(context.Background(), &source, io.Discard, func(context.Context, simulationFrame) (simulationFrame, error) {
		t.Fatal("arrival acknowledgement reached the request handler")
		return simulationFrame{}, nil
	}, nil, nil, nil, func(arrivals uint32) error {
		acknowledged += arrivals
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if acknowledged != 3 {
		t.Fatalf("acknowledged arrivals = %d, want 3", acknowledged)
	}
}

func TestSimulationModelTransportCorrelatesConcurrentResponses(t *testing.T) {
	requestRead, requestWrite := io.Pipe()
	responseRead, responseWrite := io.Pipe()
	transport := newSimulationModelTransport(requestWrite, responseRead, nil, nil, nil)
	defer func() {
		if err := transport.close(); err != nil {
			t.Error(err)
		}
	}()

	served := make(chan error, 1)
	go func() {
		first, err := readSimulationModelTransportFrame(requestRead)
		if err != nil {
			served <- err
			return
		}
		second, err := readSimulationModelTransportFrame(requestRead)
		if err != nil {
			served <- err
			return
		}
		second.Kind, second.Payload = simulationFrameResponse, []byte(second.Node)
		first.Kind, first.Payload = simulationFrameResponse, []byte(first.Node)
		if err := writeSimulationModelTransportFrame(responseWrite, second); err != nil {
			served <- err
			return
		}
		served <- writeSimulationModelTransportFrame(responseWrite, first)
	}()

	results := make(chan string, 2)
	for _, payload := range []string{"first", "second"} {
		go func(payload string) {
			response, err := transport.exchange(context.Background(), simulationFrame{Profile: simulationProtocol, Kind: simulationFrameModel, Request: 1, Node: payload, Incarnation: 1})
			if err != nil {
				results <- err.Error()
				return
			}
			if string(response.Payload) != payload {
				results <- "mismatch"
				return
			}
			results <- payload
		}(payload)
	}
	got := []string{<-results, <-results}
	if !reflect.DeepEqual(map[string]bool{got[0]: true, got[1]: true}, map[string]bool{"first": true, "second": true}) {
		t.Fatalf("responses = %q", got)
	}
	if err := <-served; err != nil {
		t.Fatal(err)
	}
}

func TestSimulationModelTransportCarriesLogicalTime(t *testing.T) {
	var encoded bytes.Buffer
	want := simulationFrame{
		Profile: simulationProtocol, Kind: simulationFrameResponse, Request: 7, Node: "node", Incarnation: 1,
		Arrivals: 2, Time: simulationInitialTime + 11, Payload: []byte("response"),
	}
	if err := writeSimulationModelTransportFrame(&encoded, want); err != nil {
		t.Fatal(err)
	}
	got, err := readSimulationModelTransportFrame(&encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("model response = %#v, want %#v", got, want)
	}
	want.Time = 1
	if err := writeSimulationModelTransportFrame(io.Discard, want); err == nil {
		t.Fatal("model response accepted an invalid logical time")
	}
}

func TestSimulationExplorationFramesUseTheirOwnPayloadBounds(t *testing.T) {
	payload := make([]byte, maximumSimulationBootstrapBytes+1)
	requireValid := func(kind simulationFrameKind) {
		t.Helper()
		if err := validateSimulationFrame(simulationFrame{Profile: simulationProtocol, Kind: kind, Request: 1, Payload: payload}); err != nil {
			t.Fatalf("validate %s frame: %v", kind, err)
		}
	}
	requireValid(simulationFrameExplorationPlan)
	requireValid(simulationFrameExplorationRecord)
	if err := validateSimulationFrame(simulationFrame{Profile: simulationProtocol, Kind: simulationFrameStart, Request: 1, Payload: payload}); err == nil {
		t.Fatal("simulation start frame accepted an oversized bootstrap payload")
	}
}

func TestSimulationModelTransportDiscardsLateCancelledResponse(t *testing.T) {
	requestRead, requestWrite := io.Pipe()
	responseRead, responseWrite := io.Pipe()
	discarded := false
	transport := newSimulationModelTransport(requestWrite, responseRead, nil, func(frame simulationFrame) error {
		if frame.Node == "cancelled" {
			return errors.New("cancelled response reached the active arrival handler")
		}
		return nil
	}, func(frame simulationFrame) error {
		if frame.Node == "cancelled" {
			discarded = true
		}
		return nil
	})
	t.Cleanup(func() {
		if err := errors.Join(transport.close(), requestRead.Close(), responseWrite.Close()); err != nil {
			t.Error(err)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	firstResult := make(chan error, 1)
	go func() {
		_, err := transport.exchange(ctx, simulationFrame{Profile: simulationProtocol, Kind: simulationFrameModel, Node: "cancelled", Incarnation: 1})
		firstResult <- err
	}()
	first, err := readSimulationModelTransportFrame(requestRead)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if err := <-firstResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled exchange error = %v", err)
	}
	first.Kind = simulationFrameResponse
	if err := writeSimulationModelTransportFrame(responseWrite, first); err != nil {
		t.Fatal(err)
	}

	secondResult := make(chan simulationModelResult, 1)
	go func() {
		frame, exchangeErr := transport.exchange(context.Background(), simulationFrame{Profile: simulationProtocol, Kind: simulationFrameModel, Node: "next", Incarnation: 1})
		secondResult <- simulationModelResult{frame: frame, err: exchangeErr}
	}()
	second, err := readSimulationModelTransportFrame(requestRead)
	if err != nil {
		t.Fatal(err)
	}
	second.Kind = simulationFrameResponse
	second.Payload = []byte("ok")
	if err := writeSimulationModelTransportFrame(responseWrite, second); err != nil {
		t.Fatal(err)
	}
	result := <-secondResult
	if result.err != nil {
		t.Fatal(result.err)
	}
	if !reflect.DeepEqual(result.frame.Payload, []byte("ok")) {
		t.Fatalf("second response payload = %q", result.frame.Payload)
	}
	if !discarded {
		t.Fatal("late cancelled response was not discarded")
	}
}

func TestServeSimulationModelsAllowsConcurrentBlockingOperations(t *testing.T) {
	requestRead, requestWrite := io.Pipe()
	responseRead, responseWrite := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	served := make(chan error, 1)
	release := make(chan struct{})
	go func() {
		served <- serveSimulationModels(ctx, requestRead, responseWrite, func(_ context.Context, request simulationFrame) (simulationFrame, error) {
			if request.Node == "blocked" {
				<-release
			} else {
				close(release)
			}
			return simulationFrame{Payload: []byte(request.Node)}, nil
		}, nil, nil)
	}()
	transport := newSimulationModelTransport(requestWrite, responseRead, nil, nil, nil)
	results := make(chan string, 2)
	for _, node := range []string{"blocked", "release"} {
		go func(node string) {
			response, err := transport.exchange(ctx, simulationFrame{Profile: simulationProtocol, Kind: simulationFrameModel, Node: node, Incarnation: 1})
			if err != nil {
				results <- err.Error()
				return
			}
			results <- string(response.Payload)
		}(node)
	}
	got := map[string]bool{<-results: true, <-results: true}
	if !reflect.DeepEqual(got, map[string]bool{"blocked": true, "release": true}) {
		t.Fatalf("responses = %v", got)
	}
	if err := transport.close(); err != nil {
		t.Fatal(err)
	}
	if err := <-served; err != nil {
		t.Fatal(err)
	}
}

func TestSimulationFrameRoundTrip(t *testing.T) {
	request := simulationFrame{
		Profile:     simulationProtocol,
		Kind:        simulationFrameStart,
		Request:     7,
		Node:        "history",
		Incarnation: 2,
		Arrivals:    3,
		Payload:     []byte("bounded bootstrap"),
	}
	encoded, err := encodeSimulationFrame(request)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeSimulationFrame(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, request) {
		t.Fatalf("decoded frame = %+v, want %+v", decoded, request)
	}
}

func TestSimulationActivationFramesAreValid(t *testing.T) {
	for _, kind := range []simulationFrameKind{simulationFrameActivate, simulationFrameActivated} {
		frame := simulationFrame{Profile: simulationProtocol, Kind: kind, Request: 1, Node: "history", Incarnation: 2}
		if _, err := encodeSimulationFrame(frame); err != nil {
			t.Fatalf("encode %q: %v", kind, err)
		}
	}
}

func TestSimulationFrameRejectsUnknownAndOversizedInput(t *testing.T) {
	_, err := decodeSimulationFrame([]byte(`{"profile":"gomadv3.simulation-process/v3","kind":"start","request":1,"unknown":true}`))
	if err == nil {
		t.Fatal("unknown field was accepted")
	}
	_, err = encodeSimulationFrame(simulationFrame{Profile: simulationProtocol, Kind: simulationFrameStart, Request: 1, Payload: make([]byte, maximumSimulationFrameBytes)})
	if err == nil {
		t.Fatal("oversized frame was accepted")
	}
}

func TestSimulationCapabilityValidation(t *testing.T) {
	request := Spec{
		SupervisorCommand: []string{"supervisor"}, BootstrapCommand: []string{"bootstrap"}, Command: "target", Argv0: "target", Dir: t.TempDir(),
		ExecutionTimeout: 1, OutputLimit: 1, World: WorldCapability{RecordLimit: 1, TransitionLimit: 1},
		Simulation: &SimulationCapability{Role: "invalid"},
	}
	if err := validateSpec(request); err == nil {
		t.Fatal("invalid simulation role was accepted")
	}
	request.Simulation.Role = SimulationRoleCoordinator
	if err := validateSpec(request); err != nil {
		t.Fatal(err)
	}
	request.Simulation.Role = SimulationRoleNode
	if err := validateSpec(request); err == nil {
		t.Fatal("node simulation capability without bootstrap was accepted")
	}
}

func TestRunSupervisesSimulationNodeProcess(t *testing.T) {
	result, err := Run(context.Background(), Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADV3_PROCESS_HELPER=simulation-coordinator"}, ExecutionTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 1 << 20,
		World:      WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 7},
		Simulation: &SimulationCapability{Role: SimulationRoleCoordinator},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != TerminationExit || result.ExitCode != 0 || !result.GroupGone {
		t.Fatalf("simulation coordinator result = %#v", result)
	}
}

func TestRunHardCrashesAndReapsSimulationNodeProcess(t *testing.T) {
	result, err := Run(context.Background(), Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestSupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestTargetBootstrapHelper"},
		Command:           os.Args[0], Args: []string{"-test.run=TestTargetHelper"}, Argv0: "gomadv3-target", Dir: t.TempDir(),
		Env: []string{"GOMADV3_PROCESS_HELPER=simulation-coordinator", "GOMADV3_SIMULATION_CASE=crash"}, ExecutionTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 1 << 20,
		World:      WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 7},
		Simulation: &SimulationCapability{Role: SimulationRoleCoordinator},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Termination != TerminationExit || result.ExitCode != 0 || !result.GroupGone {
		t.Fatalf("simulation coordinator result = %#v", result)
	}
}
