// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadsim

import (
	"encoding/binary"
	"sync"
)

const maximumProcessModelRequests = 4096

type processModelResult struct {
	frame ModelTransportFrame
	ok    bool
}

var processModels = struct {
	sync.Mutex
	write       sync.Mutex
	initialized bool
	failed      bool
	next        uint64
	request     int
	response    int
	pending     map[uint64]chan processModelResult
	done        chan struct{}
}{}

func ProcessModelExchange(node string, incarnation uint64, payload []byte, limit uint64) ([]byte, string, bool) {
	if ProcessRole() != 2 || node == "" || len(node) > 256 || incarnation == 0 || len(payload) == 0 || len(payload) > maximumProcessFrameBytes || limit == 0 || limit > maximumProcessFrameBytes {
		return nil, "", false
	}
	processModels.Lock()
	if !processModels.initialized {
		request, requestOK := processDescriptor(processModelRequestFDEnvironmentName)
		response, responseOK := processDescriptor(processModelResponseFDEnvironmentName)
		if !requestOK || !responseOK {
			processModels.Unlock()
			return nil, "", false
		}
		processModels.initialized = true
		processModels.request = request
		processModels.response = response
		processModels.pending = make(map[uint64]chan processModelResult)
		processModels.done = make(chan struct{})
		go readProcessModelResponses(response)
	}
	if processModels.failed || len(processModels.pending) >= maximumProcessModelRequests {
		processModels.Unlock()
		return nil, "", false
	}
	processModels.next++
	if processModels.next == 0 {
		failProcessModelsLocked()
		processModels.Unlock()
		return nil, "", false
	}
	requestID := processModels.next
	result := make(chan processModelResult, 1)
	processModels.pending[requestID] = result
	requestDescriptor := processModels.request
	done := processModels.done
	processModels.Unlock()
	runtimeSimulationExternalBegin()
	defer runtimeSimulationExternalEnd()

	request := ModelTransportFrame{Request: requestID, Node: node, Incarnation: incarnation, Arrivals: runtimeSimulationTimeTakeArrivals(), Payload: append([]byte(nil), payload...)}
	encoded, err := EncodeModelTransportFrame(request)
	if err != nil {
		removeProcessModelRequest(requestID)
		return nil, "", false
	}
	processModels.write.Lock()
	written := writeProcessModelFrame(requestDescriptor, encoded)
	processModels.write.Unlock()
	if !written {
		failProcessModels()
		return nil, "", false
	}
	select {
	case response := <-result:
		if !response.ok || !response.frame.Response || response.frame.Node != "" && response.frame.Node != node || response.frame.Incarnation != 0 && response.frame.Incarnation != incarnation || uint64(len(response.frame.Payload)) > limit {
			return nil, "", false
		}
		return append([]byte(nil), response.frame.Payload...), response.frame.Error, true
	case <-done:
		return nil, "", false
	}
}

func readProcessModelResponses(descriptor int) {
	for {
		var header [4]byte
		if !processReadFull(descriptor, header[:]) {
			failProcessModels()
			return
		}
		size := binary.BigEndian.Uint32(header[:])
		if size == 0 || size > maximumProcessFrameBytes {
			failProcessModels()
			return
		}
		encoded := make([]byte, size)
		if !processReadFull(descriptor, encoded) {
			failProcessModels()
			return
		}
		frame, err := DecodeModelTransportFrame(encoded)
		if err != nil || !frame.Response || !runtimeSimulationTimeObserve(frame.Time) {
			failProcessModels()
			return
		}
		runtimeSimulationExternalArrive()
		if !acknowledgeProcessArrivals() {
			failProcessModels()
			return
		}
		processModels.Lock()
		result := processModels.pending[frame.Request]
		if result != nil {
			delete(processModels.pending, frame.Request)
		}
		processModels.Unlock()
		if result == nil {
			failProcessModels()
			return
		}
		result <- processModelResult{frame: frame, ok: true}
	}
}

func writeProcessModelFrame(descriptor int, encoded []byte) bool {
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(encoded)))
	return processWriteFull(descriptor, header[:]) && processWriteFull(descriptor, encoded)
}

func removeProcessModelRequest(request uint64) {
	processModels.Lock()
	delete(processModels.pending, request)
	processModels.Unlock()
}

func failProcessModels() {
	processModels.Lock()
	failProcessModelsLocked()
	processModels.Unlock()
}

func failProcessModelsLocked() {
	if !processModels.failed {
		processModels.failed = true
		close(processModels.done)
	}
}
