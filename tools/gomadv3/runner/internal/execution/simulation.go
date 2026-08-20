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
const simulationProtocol = "gomadv3.simulation-process/v2"

type SimulationRole string

const (
	SimulationRoleCoordinator SimulationRole = "coordinator"
	SimulationRoleNode        SimulationRole = "node"
)

type SimulationCapability struct {
	Role      SimulationRole
	Bootstrap []byte
	handler   func(context.Context, simulationFrame) (simulationFrame, error)
	hardCrash <-chan struct{}
}

const simulationRoleEnvironmentName = "GOMADV3_SIMULATION_ROLE"
const simulationRequestFDEnvironmentName = "GOMADV3_SIMULATION_REQUEST_FD"
const simulationResponseFDEnvironmentName = "GOMADV3_SIMULATION_RESPONSE_FD"
const simulationBootstrapFDEnvironmentName = "GOMADV3_SIMULATION_BOOTSTRAP_FD"
const simulationControlFDEnvironmentName = "GOMADV3_SIMULATION_CONTROL_FD"
const simulationModelRequestFDEnvironmentName = "GOMADV3_SIMULATION_MODEL_REQUEST_FD"
const simulationModelResponseFDEnvironmentName = "GOMADV3_SIMULATION_MODEL_RESPONSE_FD"

type simulationFrameKind string

const (
	simulationFrameStart     simulationFrameKind = "start"
	simulationFrameActivate  simulationFrameKind = "activate"
	simulationFrameActivated simulationFrameKind = "activated"
	simulationFrameModel     simulationFrameKind = "model"
	simulationFrameStop      simulationFrameKind = "stop"
	simulationFrameCrash     simulationFrameKind = "crash"
	simulationFrameWait      simulationFrameKind = "wait"
	simulationFrameReady     simulationFrameKind = "ready"
	simulationFrameTerminal  simulationFrameKind = "terminal"
	simulationFrameResponse  simulationFrameKind = "response"
)

type simulationFrame struct {
	Profile     string              `json:"profile"`
	Kind        simulationFrameKind `json:"kind"`
	Request     uint64              `json:"request"`
	Node        string              `json:"node,omitempty"`
	Incarnation uint64              `json:"incarnation,omitempty"`
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
	case simulationFrameStart, simulationFrameActivate, simulationFrameActivated, simulationFrameModel, simulationFrameStop, simulationFrameCrash, simulationFrameWait, simulationFrameReady, simulationFrameTerminal, simulationFrameResponse:
	default:
		return fmt.Errorf("invalid simulation frame kind %q", frame.Kind)
	}
	if frame.Request == 0 {
		return errors.New("simulation frame request identity is required")
	}
	if len(frame.Node) > 256 {
		return errors.New("simulation frame node identity exceeds its bound")
	}
	if len(frame.Payload) > maximumSimulationBootstrapBytes {
		return errors.New("simulation frame payload exceeds its bound")
	}
	if len(frame.Error) > 4096 {
		return errors.New("simulation frame error exceeds its bound")
	}
	return nil
}

func writeSimulationFrame(destination io.Writer, frame simulationFrame) error {
	encoded, err := encodeSimulationFrame(frame)
	if err != nil {
		return err
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
	if size == 0 || size > maximumSimulationFrameBytes {
		return simulationFrame{}, errors.New("simulation frame size is invalid")
	}
	encoded := make([]byte, size)
	if _, err := io.ReadFull(source, encoded); err != nil {
		return simulationFrame{}, fmt.Errorf("read simulation frame payload: %w", err)
	}
	return decodeSimulationFrame(encoded)
}

func serveSimulation(ctx context.Context, source io.Reader, destination io.Writer, handler func(context.Context, simulationFrame) (simulationFrame, error)) error {
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
		response, handleErr := handler(ctx, request)
		response.Profile = simulationProtocol
		response.Kind = simulationFrameResponse
		response.Request = request.Request
		if handleErr != nil {
			response.Error = handleErr.Error()
		}
		if err := writeSimulationFrame(destination, response); err != nil {
			return fmt.Errorf("write simulation response: %w", err)
		}
	}
}
