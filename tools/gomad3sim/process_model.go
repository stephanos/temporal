package gomad3sim

import (
	"errors"
	"sync"
)

var processModel = struct {
	sync.Mutex
	once      sync.Once
	handler   func(processModelRequest) ([]byte, error)
	serverErr error
}{}

type processModelRequest struct {
	Handle  NodeHandle
	Payload []byte
}

func beginProcessModelBroker(handler func(processModelRequest) ([]byte, error)) (func() error, error) {
	if handler == nil {
		return nil, errors.New("process simulation model handler is nil")
	}
	processModel.Lock()
	defer processModel.Unlock()
	if processModel.handler != nil {
		return nil, errors.New("process simulation model broker is already active")
	}
	processModel.handler = handler
	processModel.once.Do(func() {
		started := make(chan struct{})
		go func() {
			close(started)
			err := processBackendServeModel(dispatchProcessModelFrame)
			processModel.Lock()
			processModel.serverErr = err
			processModel.Unlock()
		}()
		<-started
	})
	return func() error {
		processModel.Lock()
		defer processModel.Unlock()
		processModel.handler = nil
		return processModel.serverErr
	}, nil
}

func dispatchProcessModelFrame(node string, incarnation uint64, payload []byte) ([]byte, string) {
	if node == "" || len(node) > 256 || incarnation == 0 || len(payload) == 0 || len(payload) > maximumProcessFrameBytes {
		return nil, boundedTerminalText("invalid process simulation model request")
	}
	processModel.Lock()
	handler := processModel.handler
	processModel.Unlock()
	if handler == nil {
		return nil, boundedTerminalText("process simulation model is inactive")
	}
	response, err := handler(processModelRequest{Handle: NodeHandle{Node: NodeID(node), Incarnation: incarnation}, Payload: append([]byte(nil), payload...)})
	if err != nil {
		return nil, boundedTerminalText(err.Error())
	}
	return response, ""
}
