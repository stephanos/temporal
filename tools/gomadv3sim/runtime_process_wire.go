package gomadv3sim

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const maximumProcessFrameBytes = 128 << 20
const processProtocol = "gomadv3.simulation-process/v3"
const processNodeBootstrapSchema = "gomadv3.simulation-node-bootstrap/v1"
const processNodeTerminalSchema = "gomadv3.simulation-node-terminal/v1"

type processFrameKind string

const (
	processFrameStart             processFrameKind = "start"
	processFrameActivate          processFrameKind = "activate"
	processFrameActivated         processFrameKind = "activated"
	processFrameModel             processFrameKind = "model"
	processFrameStop              processFrameKind = "stop"
	processFrameCrash             processFrameKind = "crash"
	processFrameWait              processFrameKind = "wait"
	processFrameReady             processFrameKind = "ready"
	processFrameTerminal          processFrameKind = "terminal"
	processFrameResponse          processFrameKind = "response"
	processFrameExplorationPlan   processFrameKind = "exploration_plan"
	processFrameExplorationRecord processFrameKind = "exploration_record"
)

type processFrame struct {
	Profile     string           `json:"profile"`
	Kind        processFrameKind `json:"kind"`
	Request     uint64           `json:"request"`
	Node        string           `json:"node,omitempty"`
	Incarnation uint64           `json:"incarnation,omitempty"`
	Arrivals    uint32           `json:"arrivals,omitempty"`
	Payload     []byte           `json:"payload,omitempty"`
	Error       string           `json:"error,omitempty"`
}

type processNodeBootstrap struct {
	Schema     string      `json:"schema"`
	SpecSHA256 string      `json:"spec_sha256"`
	Boot       BootID      `json:"boot"`
	Context    NodeContext `json:"context"`
}

type processNodeTerminal struct {
	Schema  string              `json:"schema"`
	Error   string              `json:"error,omitempty"`
	Outputs []OutputObservation `json:"outputs,omitempty"`
}

func encodeProcessValue(value any) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode process simulation value: %w", err)
	}
	if len(encoded) == 0 || len(encoded) > maximumProcessFrameBytes {
		return nil, errors.New("process simulation value exceeds its bound")
	}
	return encoded, nil
}

func decodeProcessValue(encoded []byte, destination any) error {
	if len(encoded) == 0 || len(encoded) > maximumProcessFrameBytes {
		return errors.New("process simulation value size is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode process simulation value: %w", err)
	}
	if token, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("process simulation value contains trailing data: %v: %w", token, err)
	}
	return nil
}

func validateProcessFrame(frame processFrame) error {
	if frame.Profile != processProtocol {
		return fmt.Errorf("unsupported process simulation frame profile %q", frame.Profile)
	}
	switch frame.Kind {
	case processFrameStart, processFrameActivate, processFrameActivated, processFrameModel, processFrameStop, processFrameCrash, processFrameWait, processFrameReady, processFrameTerminal, processFrameResponse, processFrameExplorationPlan, processFrameExplorationRecord:
	default:
		return fmt.Errorf("invalid process simulation frame kind %q", frame.Kind)
	}
	if frame.Request == 0 {
		return errors.New("process simulation request identity is required")
	}
	if frame.Kind == processFrameResponse && frame.Arrivals != 0 {
		return errors.New("process simulation response cannot acknowledge external work")
	}
	if len(frame.Node) > 256 || len(frame.Payload) > maximumProcessFrameBytes || len(frame.Error) > MaximumTerminalReasonBytes {
		return errors.New("process simulation frame exceeds its bound")
	}
	return nil
}
