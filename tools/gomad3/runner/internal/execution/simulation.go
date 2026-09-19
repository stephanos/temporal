package execution

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const maximumSimulationFrameBytes = 128 << 20
const maximumSimulationBootstrapBytes = 1 << 20
const maximumSimulationExplorationPlanBytes = 16 << 20
const maximumSimulationExplorationRecordBytes = 128 << 20
const maximumSimulationExplorationRecords = 4096
const simulationProtocol = "gomad3.simulation-process/v3"

type SimulationRole string

const (
	SimulationRoleCoordinator SimulationRole = "coordinator"
	SimulationRoleNode        SimulationRole = "node"
)

type SimulationCapability struct {
	Role                   SimulationRole
	Bootstrap              []byte
	ExplorationPlan        []byte
	ExplorationRecordLimit uint64
	ExplorationRecordCount uint64
	handler                func(context.Context, simulationFrame) (simulationFrame, error)
	time                   func(context.Context, simulationTimeRequest) (simulationTimeResponse, error)
	accepting              func(simulationFrame) error
	delivering             func(simulationFrame)
	responded              func(simulationFrame)
	arrived                func(uint32) error
	hardCrash              <-chan struct{}
	reaped                 chan struct{}
}

const simulationRoleEnvironmentName = "GOMAD3_SIMULATION_ROLE"
const simulationRequestFDEnvironmentName = "GOMAD3_SIMULATION_REQUEST_FD"
const simulationResponseFDEnvironmentName = "GOMAD3_SIMULATION_RESPONSE_FD"
const simulationBootstrapFDEnvironmentName = "GOMAD3_SIMULATION_BOOTSTRAP_FD"
const simulationControlFDEnvironmentName = "GOMAD3_SIMULATION_CONTROL_FD"
const simulationModelRequestFDEnvironmentName = "GOMAD3_SIMULATION_MODEL_REQUEST_FD"
const simulationModelResponseFDEnvironmentName = "GOMAD3_SIMULATION_MODEL_RESPONSE_FD"
const simulationTimeRequestFDEnvironmentName = "GOMAD3_SIMULATION_TIME_REQUEST_FD"
const simulationTimeResponseFDEnvironmentName = "GOMAD3_SIMULATION_TIME_RESPONSE_FD"

type simulationFrameKind string

const (
	simulationFrameStart             simulationFrameKind = "start"
	simulationFrameActivate          simulationFrameKind = "activate"
	simulationFrameActivated         simulationFrameKind = "activated"
	simulationFrameModel             simulationFrameKind = "model"
	simulationFrameStop              simulationFrameKind = "stop"
	simulationFrameCrash             simulationFrameKind = "crash"
	simulationFrameWait              simulationFrameKind = "wait"
	simulationFrameReady             simulationFrameKind = "ready"
	simulationFrameTerminal          simulationFrameKind = "terminal"
	simulationFrameResponse          simulationFrameKind = "response"
	simulationFrameArrival           simulationFrameKind = "arrival"
	simulationFrameExplorationPlan   simulationFrameKind = "exploration_plan"
	simulationFrameExplorationRecord simulationFrameKind = "exploration_record"
)

type simulationFrame struct {
	Profile     string              `json:"profile"`
	Kind        simulationFrameKind `json:"kind"`
	Request     uint64              `json:"request"`
	Node        string              `json:"node,omitempty"`
	Incarnation uint64              `json:"incarnation,omitempty"`
	Arrivals    uint32              `json:"arrivals,omitempty"`
	Time        int64               `json:"-"`
	Payload     []byte              `json:"payload,omitempty"`
	Error       string              `json:"error,omitempty"`
}

func encodeSimulationFrame(frame simulationFrame) ([]byte, error) {
	if err := validateSimulationFrame(frame); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(frame)
	if err != nil {
		return nil, fmt.Errorf("encode simulation frame: %w", err)
	}
	if len(encoded) > maximumSimulationFrameBytes {
		return nil, errors.New("simulation frame exceeds its bound")
	}
	return encoded, nil
}

func decodeSimulationFrame(encoded []byte) (simulationFrame, error) {
	if len(encoded) == 0 || len(encoded) > maximumSimulationFrameBytes {
		return simulationFrame{}, errors.New("simulation frame size is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var frame simulationFrame
	if err := decoder.Decode(&frame); err != nil {
		return simulationFrame{}, fmt.Errorf("decode simulation frame: %w", err)
	}
	if token, err := decoder.Token(); err != io.EOF {
		return simulationFrame{}, fmt.Errorf("simulation frame contains trailing data: %v: %w", token, err)
	}
	if err := validateSimulationFrame(frame); err != nil {
		return simulationFrame{}, err
	}
	return frame, nil
}

func validateSimulationFrame(frame simulationFrame) error {
	if frame.Profile != simulationProtocol {
		return fmt.Errorf("unsupported simulation frame profile %q", frame.Profile)
	}
	switch frame.Kind {
	case simulationFrameStart, simulationFrameActivate, simulationFrameActivated, simulationFrameModel, simulationFrameStop, simulationFrameCrash, simulationFrameWait, simulationFrameReady, simulationFrameTerminal, simulationFrameResponse, simulationFrameExplorationPlan, simulationFrameExplorationRecord:
	default:
		return fmt.Errorf("invalid simulation frame kind %q", frame.Kind)
	}
	if frame.Request == 0 {
		return errors.New("simulation frame request identity is required")
	}
	if frame.Kind == simulationFrameResponse && frame.Arrivals != 0 {
		return errors.New("simulation response cannot acknowledge external work")
	}
	if len(frame.Node) > 256 {
		return errors.New("simulation frame node identity exceeds its bound")
	}
	if len(frame.Payload) > simulationFramePayloadLimit(frame.Kind) {
		return errors.New("simulation frame payload exceeds its bound")
	}
	if len(frame.Error) > 4096 {
		return errors.New("simulation frame error exceeds its bound")
	}
	return nil
}

func simulationFramePayloadLimit(kind simulationFrameKind) int {
	switch kind {
	case simulationFrameExplorationPlan:
		return maximumSimulationExplorationPlanBytes
	case simulationFrameExplorationRecord:
		return maximumSimulationExplorationRecordBytes
	default:
		return maximumSimulationBootstrapBytes
	}
}

func writeSimulationFrame(destination io.Writer, frame simulationFrame) error {
	return writeSimulationFrameDelivering(destination, frame, nil)
}

func writeSimulationFrameDelivering(destination io.Writer, frame simulationFrame, delivering func()) error {
	encoded, err := encodeSimulationFrame(frame)
	if err != nil {
		return err
	}
	if delivering != nil {
		delivering()
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(encoded)))
	if _, err := destination.Write(header[:]); err != nil {
		return fmt.Errorf("write simulation frame header: %w", err)
	}
	if _, err := destination.Write(encoded); err != nil {
		return fmt.Errorf("write simulation frame payload: %w", err)
	}
	return nil
}

func readSimulationFrame(source io.Reader) (simulationFrame, error) {
	var header [4]byte
	if _, err := io.ReadFull(source, header[:]); err != nil {
		return simulationFrame{}, err
	}
	size := binary.BigEndian.Uint32(header[:])
	if size == 0 {
		if _, err := io.ReadFull(source, header[:]); err != nil {
			return simulationFrame{}, fmt.Errorf("read simulation arrival frame: %w", err)
		}
		arrivals := binary.BigEndian.Uint32(header[:])
		if arrivals == 0 {
			return simulationFrame{}, errors.New("simulation arrival frame is empty")
		}
		return simulationFrame{Profile: simulationProtocol, Kind: simulationFrameArrival, Arrivals: arrivals}, nil
	}
	if size > maximumSimulationFrameBytes {
		return simulationFrame{}, errors.New("simulation frame size is invalid")
	}
	encoded := make([]byte, size)
	if _, err := io.ReadFull(source, encoded); err != nil {
		return simulationFrame{}, fmt.Errorf("read simulation frame payload: %w", err)
	}
	return decodeSimulationFrame(encoded)
}

func serveSimulation(ctx context.Context, source io.Reader, destination io.Writer, handler func(context.Context, simulationFrame) (simulationFrame, error), accepting func(simulationFrame) error, delivering func(simulationFrame), responded func(simulationFrame), arrived func(uint32) error) error {
	if handler == nil {
		return errors.New("simulation frame handler is unavailable")
	}
	for {
		request, err := readSimulationFrame(source)
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read simulation request: %w", err)
		}
		if request.Kind == simulationFrameArrival {
			if arrived == nil {
				return errors.New("simulation arrival handler is unavailable")
			}
			if err := arrived(request.Arrivals); err != nil {
				return err
			}
			continue
		}
		if err := acceptSimulationWait(destination, accepting, request); err != nil {
			return err
		}
		response, handleErr := handler(ctx, request)
		response.Profile = simulationProtocol
		response.Kind = simulationFrameResponse
		response.Request = request.Request
		if handleErr != nil {
			response.Error = handleErr.Error()
		}
		if err := writeSimulationFrameDelivering(destination, response, func() {
			if delivering != nil {
				delivering(request)
			}
		}); err != nil {
			return fmt.Errorf("write simulation response: %w", err)
		}
		if responded != nil {
			responded(request)
		}
	}
}

func acceptSimulationWait(destination io.Writer, accepting func(simulationFrame) error, request simulationFrame) error {
	if request.Kind != simulationFrameWait {
		return nil
	}
	if accepting == nil {
		return errors.New("simulation wait acceptance handler is unavailable")
	}
	if err := accepting(request); err != nil {
		return err
	}
	var accepted [4]byte
	written, err := destination.Write(accepted[:])
	if err != nil {
		return fmt.Errorf("write simulation wait acceptance: %w", err)
	}
	if written != len(accepted) {
		return io.ErrShortWrite
	}
	return nil
}
