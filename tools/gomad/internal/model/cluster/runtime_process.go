package cluster

import (
	"errors"
	"fmt"
	"sync"

	"go.temporal.io/server/tools/gomad/internal/model/cluster/internal/wire"
)

type processRole uint8

const (
	processRoleUnavailable processRole = iota
	processRoleCoordinator
	processRoleNode
)

var processRequests = struct {
	sync.Mutex
	next uint64
}{}

func processBackendAvailable() bool {
	return false
}

func processBackendRole() processRole {
	return processRoleUnavailable
}

func processBackendBootstrap(uint64) ([]byte, error) {
	return nil, ErrBackendUnavailable
}

func processBackendExchange([]byte, uint64) ([]byte, error) {
	return nil, ErrBackendUnavailable
}

func processBackendWaitStop() error {
	return ErrBackendUnavailable
}

func processBackendServeModel(func(string, uint64, []byte) ([]byte, string)) error {
	return ErrBackendUnavailable
}

func exchangeProcessFrame(kind wire.ProcessFrameKind, handle NodeHandle, payload []byte) (wire.ProcessFrame, error) {
	processRequests.Lock()
	defer processRequests.Unlock()
	processRequests.next++
	if processRequests.next == 0 {
		return wire.ProcessFrame{}, errors.New("process simulation request identity exhausted")
	}
	request := wire.ProcessFrame{Profile: wire.ProcessProtocol, Kind: kind, Request: processRequests.next, Node: string(handle.Node), Incarnation: handle.Incarnation, Arrivals: runtimeProcessTimeArrivals(), Payload: append([]byte(nil), payload...)}
	if err := wire.ValidateProcessFrame(request); err != nil {
		return wire.ProcessFrame{}, err
	}
	encoded, err := wire.EncodeProcessValue(request)
	if err != nil {
		return wire.ProcessFrame{}, err
	}
	responseBytes, err := processBackendExchange(encoded, wire.MaximumProcessFrameBytes)
	if err != nil {
		return wire.ProcessFrame{}, err
	}
	var response wire.ProcessFrame
	if err := wire.DecodeProcessValue(responseBytes, &response); err != nil {
		return wire.ProcessFrame{}, err
	}
	if err := wire.ValidateProcessFrame(response); err != nil {
		return wire.ProcessFrame{}, err
	}
	if response.Kind != wire.ProcessFrameResponse || response.Request != request.Request || response.Node != "" && response.Node != request.Node || response.Incarnation != 0 && response.Incarnation != request.Incarnation {
		return wire.ProcessFrame{}, errors.New("process simulation response identity changed")
	}
	if response.Error != "" {
		return wire.ProcessFrame{}, fmt.Errorf("process simulation %s: %s", kind, response.Error)
	}
	return response, nil
}

func processExplorationPlan() ([]byte, error) {
	response, err := exchangeProcessFrame(wire.ProcessFrameExplorationPlan, NodeHandle{}, nil)
	if err != nil {
		return nil, err
	}
	if len(response.Payload) > MaximumExplorationPlanBytes {
		return nil, errors.New("process simulation exploration plan exceeds its bound")
	}
	return append([]byte(nil), response.Payload...), nil
}

func publishProcessExplorationRecord(record []byte) error {
	if len(record) == 0 || len(record) > MaximumClusterRecordBytes {
		return errors.New("process simulation exploration record size is invalid")
	}
	_, err := exchangeProcessFrame(wire.ProcessFrameExplorationRecord, NodeHandle{}, record)
	return err
}
