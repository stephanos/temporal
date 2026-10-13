package cluster

import (
	"errors"
	"fmt"
	"go.temporal.io/server/tools/gomad3/internal/model/cluster/internal/wire"
	"sync"
	_ "unsafe"
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

//go:linkname gomadProcessAvailable internal/gomadsim.ProcessAvailable
func gomadProcessAvailable() bool

//go:linkname gomadProcessRole internal/gomadsim.ProcessRole
func gomadProcessRole() uint8

//go:linkname gomadProcessBootstrap internal/gomadsim.ProcessBootstrap
func gomadProcessBootstrap(uint64) ([]byte, bool)

//go:linkname gomadProcessExchange internal/gomadsim.ProcessExchange
func gomadProcessExchange([]byte, uint64) ([]byte, bool)

//go:linkname gomadProcessWaitStop internal/gomadsim.ProcessWaitStop
func gomadProcessWaitStop() bool

//go:linkname gomadProcessServeModel internal/gomadsim.ProcessServeModel
func gomadProcessServeModel(func(string, uint64, []byte) ([]byte, string)) bool

func gomadInterceptProcessBackendAvailable() (bool, bool) {
	return gomadProcessAvailable(), true
}

func gomadInterceptProcessBackendRole() (processRole, bool) {
	return processRole(gomadProcessRole()), true
}

func gomadInterceptProcessBackendBootstrap(limit uint64) ([]byte, error, bool) {
	encoded, ok := gomadProcessBootstrap(limit)
	if !ok {
		return nil, errors.New("read process simulation bootstrap"), true
	}
	return encoded, nil, true
}

func gomadInterceptProcessBackendExchange(request []byte, limit uint64) ([]byte, error, bool) {
	encoded, ok := gomadProcessExchange(request, limit)
	if !ok {
		return nil, errors.New("exchange process simulation frame"), true
	}
	return encoded, nil, true
}

func gomadInterceptProcessBackendWaitStop() (error, bool) {
	if !gomadProcessWaitStop() {
		return errors.New("wait for process simulation stop"), true
	}
	return nil, true
}

func gomadInterceptProcessBackendServeModel(handler func(string, uint64, []byte) ([]byte, string)) (error, bool) {
	if !gomadProcessServeModel(handler) {
		return errors.New("serve process simulation model"), true
	}
	return nil, true
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
