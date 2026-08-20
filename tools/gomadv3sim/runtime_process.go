package gomadv3sim

import (
	"errors"
	"fmt"
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

func exchangeProcessFrame(kind processFrameKind, handle NodeHandle, payload []byte) (processFrame, error) {
	processRequests.Lock()
	defer processRequests.Unlock()
	processRequests.next++
	if processRequests.next == 0 {
		return processFrame{}, errors.New("process simulation request identity exhausted")
	}
	request := processFrame{Profile: processProtocol, Kind: kind, Request: processRequests.next, Node: string(handle.Node), Incarnation: handle.Incarnation, Arrivals: runtimeProcessTimeArrivals(), Payload: append([]byte(nil), payload...)}
	if err := validateProcessFrame(request); err != nil {
		return processFrame{}, err
	}
	encoded, err := encodeProcessValue(request)
	if err != nil {
		return processFrame{}, err
	}
	responseBytes, err := processBackendExchange(encoded, maximumProcessFrameBytes)
	if err != nil {
		return processFrame{}, err
	}
	var response processFrame
	if err := decodeProcessValue(responseBytes, &response); err != nil {
		return processFrame{}, err
	}
	if err := validateProcessFrame(response); err != nil {
		return processFrame{}, err
	}
	if response.Kind != processFrameResponse || response.Request != request.Request || response.Node != "" && response.Node != request.Node || response.Incarnation != 0 && response.Incarnation != request.Incarnation {
		return processFrame{}, errors.New("process simulation response identity changed")
	}
	if response.Error != "" {
		return processFrame{}, fmt.Errorf("process simulation %s: %s", kind, response.Error)
	}
	return response, nil
}
