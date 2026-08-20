// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomadsim

import (
	"encoding/binary"
	"strconv"
	"sync"
	"syscall"
	"unsafe"
)

const maximumProcessFrameBytes = 128 << 20

const processRoleEnvironmentName = "GOMADV3_SIMULATION_ROLE"
const processRequestFDEnvironmentName = "GOMADV3_SIMULATION_REQUEST_FD"
const processResponseFDEnvironmentName = "GOMADV3_SIMULATION_RESPONSE_FD"
const processBootstrapFDEnvironmentName = "GOMADV3_SIMULATION_BOOTSTRAP_FD"
const processControlFDEnvironmentName = "GOMADV3_SIMULATION_CONTROL_FD"
const processModelRequestFDEnvironmentName = "GOMADV3_SIMULATION_MODEL_REQUEST_FD"
const processModelResponseFDEnvironmentName = "GOMADV3_SIMULATION_MODEL_RESPONSE_FD"

var processTransport sync.Mutex
var processBootstrap sync.Mutex
var processBootstrapRead bool

type processModelRead struct {
	payload []byte
	ok      bool
}

//go:linkname runtimeBlockingRead runtime.gomadBlockingRead
func runtimeBlockingRead(int32, unsafe.Pointer, int32) int32

//go:linkname runtimeBlockingWrite runtime.gomadBlockingWrite
func runtimeBlockingWrite(uintptr, unsafe.Pointer, int32) int32

//go:linkname ProcessAvailable
func ProcessAvailable() bool {
	role := ProcessRole()
	if role != 1 && role != 2 {
		return false
	}
	_, requestOK := processDescriptor(processRequestFDEnvironmentName)
	_, responseOK := processDescriptor(processResponseFDEnvironmentName)
	if role == 2 {
		_, bootstrapOK := processDescriptor(processBootstrapFDEnvironmentName)
		_, controlOK := processDescriptor(processControlFDEnvironmentName)
		_, modelRequestOK := processDescriptor(processModelRequestFDEnvironmentName)
		_, modelResponseOK := processDescriptor(processModelResponseFDEnvironmentName)
		return requestOK && responseOK && bootstrapOK && controlOK && modelRequestOK && modelResponseOK
	}
	_, modelRequestOK := processDescriptor(processModelRequestFDEnvironmentName)
	_, modelResponseOK := processDescriptor(processModelResponseFDEnvironmentName)
	return requestOK && responseOK && modelRequestOK && modelResponseOK
}

//go:linkname ProcessServeModel
func ProcessServeModel(handler func(string, uint64, []byte) ([]byte, string)) bool {
	if ProcessRole() != 1 || handler == nil {
		return false
	}
	requestDescriptor, requestOK := processDescriptor(processModelRequestFDEnvironmentName)
	responseDescriptor, responseOK := processDescriptor(processModelResponseFDEnvironmentName)
	if !requestOK || !responseOK {
		return false
	}
	requests := make(chan processModelRead)
	responses := make(chan []byte, 4096)
	go func() {
		for {
			var header [4]byte
			if !processReadFull(requestDescriptor, header[:]) {
				requests <- processModelRead{}
				return
			}
			size := binary.BigEndian.Uint32(header[:])
			if size == 0 || size > maximumProcessFrameBytes {
				requests <- processModelRead{}
				return
			}
			request := make([]byte, size)
			if !processReadFull(requestDescriptor, request) {
				requests <- processModelRead{}
				return
			}
			requests <- processModelRead{payload: request, ok: true}
		}
	}()
	inFlight := 0
	for {
		input := requests
		if inFlight == 4096 {
			input = nil
		}
		select {
		case request := <-input:
			if !request.ok {
				return false
			}
			frame, err := DecodeModelTransportFrame(request.payload)
			if err != nil || frame.Response {
				return false
			}
			inFlight++
			go func(request ModelTransportFrame) {
				payload, errorText := handler(request.Node, request.Incarnation, request.Payload)
				response, encodeErr := EncodeModelTransportFrame(ModelTransportFrame{
					Response: true, Request: request.Request, Node: request.Node, Incarnation: request.Incarnation, Payload: payload, Error: errorText,
				})
				if encodeErr != nil {
					response = nil
				}
				responses <- response
			}(frame)
		case response := <-responses:
			inFlight--
			if len(response) == 0 || len(response) > maximumProcessFrameBytes {
				return false
			}
			var responseHeader [4]byte
			binary.BigEndian.PutUint32(responseHeader[:], uint32(len(response)))
			if !processWriteFull(responseDescriptor, responseHeader[:]) || !processWriteFull(responseDescriptor, response) {
				return false
			}
		}
	}
}

//go:linkname ProcessWaitStop
func ProcessWaitStop() bool {
	if ProcessRole() != 2 {
		return false
	}
	descriptor, ok := processDescriptor(processControlFDEnvironmentName)
	if !ok {
		return false
	}
	var control [1]byte
	return processReadFull(descriptor, control[:]) && control[0] == 1
}

//go:linkname ProcessRole
func ProcessRole() uint8 {
	role, ok := syscall.Getenv(processRoleEnvironmentName)
	if !ok {
		return 0
	}
	switch role {
	case "coordinator":
		return 1
	case "node":
		return 2
	default:
		return 0
	}
}

//go:linkname ProcessBootstrap
func ProcessBootstrap(limit uint64) ([]byte, bool) {
	if limit == 0 || limit > maximumProcessFrameBytes || ProcessRole() != 2 {
		return nil, false
	}
	descriptor, ok := processDescriptor(processBootstrapFDEnvironmentName)
	if !ok {
		return nil, false
	}
	processBootstrap.Lock()
	defer processBootstrap.Unlock()
	if processBootstrapRead {
		return nil, false
	}
	processBootstrapRead = true
	result := make([]byte, 0, min(limit, 4096))
	var buffer [4096]byte
	for {
		count, err := syscall.Read(descriptor, buffer[:])
		if count > 0 {
			if uint64(len(result))+uint64(count) > limit {
				return nil, false
			}
			result = append(result, buffer[:count]...)
		}
		if err != nil {
			return nil, false
		}
		if count == 0 {
			return result, len(result) != 0
		}
	}
}

//go:linkname ProcessExchange
func ProcessExchange(request []byte, limit uint64) ([]byte, bool) {
	if len(request) == 0 || len(request) > maximumProcessFrameBytes || limit == 0 || limit > maximumProcessFrameBytes || !ProcessAvailable() {
		return nil, false
	}
	requestDescriptor, ok := processDescriptor(processRequestFDEnvironmentName)
	if !ok {
		return nil, false
	}
	responseDescriptor, ok := processDescriptor(processResponseFDEnvironmentName)
	if !ok {
		return nil, false
	}
	processTransport.Lock()
	defer processTransport.Unlock()
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(request)))
	if !processWriteFull(requestDescriptor, header[:]) || !processWriteFull(requestDescriptor, request) {
		return nil, false
	}
	if !processReadFull(responseDescriptor, header[:]) {
		return nil, false
	}
	size := binary.BigEndian.Uint32(header[:])
	if size == 0 || uint64(size) > limit {
		return nil, false
	}
	response := make([]byte, size)
	if !processReadFull(responseDescriptor, response) {
		return nil, false
	}
	return response, true
}

func processDescriptor(name string) (int, bool) {
	value, ok := syscall.Getenv(name)
	if !ok {
		return 0, false
	}
	descriptor, err := strconv.Atoi(value)
	return descriptor, err == nil && descriptor >= 3
}

func processWriteFull(descriptor int, source []byte) bool {
	for len(source) != 0 {
		count := int(runtimeBlockingWrite(uintptr(descriptor), unsafe.Pointer(&source[0]), int32(len(source))))
		if count <= 0 || count > len(source) {
			return false
		}
		source = source[count:]
	}
	return true
}

func processReadFull(descriptor int, destination []byte) bool {
	for len(destination) != 0 {
		count := int(runtimeBlockingRead(int32(descriptor), unsafe.Pointer(&destination[0]), int32(len(destination))))
		if count <= 0 || count > len(destination) {
			return false
		}
		destination = destination[count:]
	}
	return true
}
