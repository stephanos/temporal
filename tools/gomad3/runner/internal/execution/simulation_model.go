package execution

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
)

const maximumSimulationModelRequests = 4096

type simulationModelResult struct {
	frame simulationFrame
	err   error
}

type simulationModelRead struct {
	frame simulationFrame
	err   error
}

type simulationModelHandled struct {
	request  simulationFrame
	response simulationFrame
	err      error
}

type simulationModelTransport struct {
	request   io.WriteCloser
	response  io.ReadCloser
	delivered func()
	arrived   func(simulationFrame) error
	discarded func(simulationFrame) error
	writeMu   sync.Mutex
	mu        sync.Mutex
	next      uint64
	pending   map[uint64]chan simulationModelResult
	abandoned map[uint64]struct{}
	done      chan struct{}
	err       error
}

func newSimulationModelTransport(request io.WriteCloser, response io.ReadCloser, delivered func(), arrived func(simulationFrame) error, discarded func(simulationFrame) error) *simulationModelTransport {
	transport := &simulationModelTransport{
		request: request, response: response, delivered: delivered, arrived: arrived, discarded: discarded, pending: make(map[uint64]chan simulationModelResult), abandoned: make(map[uint64]struct{}), done: make(chan struct{}),
	}
	go transport.readResponses()
	return transport
}

func serveSimulationModels(ctx context.Context, source io.Reader, destination io.Writer, handler func(context.Context, simulationFrame) (simulationFrame, error), delivering func(simulationFrame), responded func(simulationFrame)) error {
	if handler == nil {
		return errors.New("simulation model handler is unavailable")
	}
	reads := make(chan simulationModelRead)
	go func() {
		for {
			frame, err := readSimulationModelTransportFrame(source)
			select {
			case reads <- simulationModelRead{frame: frame, err: err}:
			case <-ctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()
	handled := make(chan simulationModelHandled, maximumSimulationModelRequests)
	reading := true
	inFlight := 0
	for reading || inFlight != 0 {
		input := reads
		if !reading || inFlight == maximumSimulationModelRequests {
			input = nil
		}
		select {
		case read := <-input:
			if read.err != nil {
				if !errors.Is(read.err, io.EOF) && !errors.Is(read.err, io.ErrClosedPipe) {
					return fmt.Errorf("read simulation model request: %w", read.err)
				}
				reading = false
				continue
			}
			if read.frame.Kind != simulationFrameModel {
				return errors.New("simulation model request kind is invalid")
			}
			inFlight++
			go func(request simulationFrame) {
				response, err := handler(ctx, request)
				handled <- simulationModelHandled{request: request, response: response, err: err}
			}(read.frame)
		case result := <-handled:
			inFlight--
			response := result.response
			response.Profile = simulationProtocol
			response.Kind = simulationFrameResponse
			response.Request = result.request.Request
			if response.Node == "" {
				response.Node = result.request.Node
			}
			if response.Incarnation == 0 {
				response.Incarnation = result.request.Incarnation
			}
			if result.err != nil {
				response.Error = result.err.Error()
			}
			if err := writeSimulationModelTransportFrameDelivering(destination, response, func() {
				if delivering != nil {
					delivering(result.request)
				}
			}); err != nil {
				return fmt.Errorf("write simulation model response: %w", err)
			}
			if responded != nil {
				responded(result.request)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (transport *simulationModelTransport) exchange(ctx context.Context, frame simulationFrame) (simulationFrame, error) {
	transport.mu.Lock()
	if transport.err != nil {
		err := transport.err
		transport.mu.Unlock()
		return simulationFrame{}, err
	}
	if len(transport.pending)+len(transport.abandoned) >= maximumSimulationModelRequests {
		transport.mu.Unlock()
		return simulationFrame{}, errors.New("simulation model request capacity exceeded")
	}
	transport.next++
	if transport.next == 0 {
		transport.mu.Unlock()
		return simulationFrame{}, errors.New("simulation model request identity exhausted")
	}
	frame.Request = transport.next
	result := make(chan simulationModelResult, 1)
	transport.pending[frame.Request] = result
	transport.mu.Unlock()

	transport.writeMu.Lock()
	err := writeSimulationModelTransportFrameDelivering(transport.request, frame, transport.delivered)
	transport.writeMu.Unlock()
	if err != nil {
		transport.discard(frame.Request)
		return simulationFrame{}, err
	}
	select {
	case response := <-result:
		return response.frame, response.err
	case <-ctx.Done():
		if transport.abandon(frame.Request) {
			return simulationFrame{}, ctx.Err()
		}
		select {
		case response := <-result:
			return response.frame, response.err
		case <-transport.done:
			transport.mu.Lock()
			err := transport.err
			transport.mu.Unlock()
			return simulationFrame{}, err
		}
	case <-transport.done:
		transport.mu.Lock()
		err := transport.err
		transport.mu.Unlock()
		return simulationFrame{}, err
	}
}

func (transport *simulationModelTransport) readResponses() {
	for {
		frame, err := readSimulationModelTransportFrame(transport.response)
		if err != nil {
			transport.fail(err)
			return
		}
		if frame.Kind != simulationFrameResponse {
			transport.fail(errors.New("simulation model response kind is invalid"))
			return
		}
		transport.mu.Lock()
		result := transport.pending[frame.Request]
		_, abandoned := transport.abandoned[frame.Request]
		if result != nil {
			delete(transport.pending, frame.Request)
		} else if abandoned {
			delete(transport.abandoned, frame.Request)
		}
		transport.mu.Unlock()
		if result == nil && !abandoned {
			transport.fail(errors.New("simulation model response identity is unknown"))
			return
		}
		arrival := transport.arrived
		if abandoned {
			arrival = transport.discarded
		}
		if arrival != nil {
			if err := arrival(frame); err != nil {
				transport.fail(err)
				return
			}
		}
		if abandoned {
			continue
		}
		result <- simulationModelResult{frame: frame}
	}
}

func (transport *simulationModelTransport) discard(request uint64) {
	transport.mu.Lock()
	delete(transport.pending, request)
	transport.mu.Unlock()
}

func (transport *simulationModelTransport) abandon(request uint64) bool {
	transport.mu.Lock()
	defer transport.mu.Unlock()
	if _, ok := transport.pending[request]; !ok {
		return false
	}
	delete(transport.pending, request)
	transport.abandoned[request] = struct{}{}
	return true
}

func (transport *simulationModelTransport) fail(err error) {
	transport.mu.Lock()
	if transport.err == nil {
		transport.err = err
		close(transport.done)
	}
	transport.mu.Unlock()
}

func (transport *simulationModelTransport) close() error {
	err := errors.Join(transport.request.Close(), transport.response.Close())
	transport.fail(errors.Join(io.ErrClosedPipe, err))
	return err
}

func writeSimulationModelTransportFrame(destination io.Writer, frame simulationFrame) error {
	return writeSimulationModelTransportFrameDelivering(destination, frame, nil)
}

func writeSimulationModelTransportFrameDelivering(destination io.Writer, frame simulationFrame, delivering func()) error {
	encoded, err := EncodeModelTransportFrame(ModelTransportFrame{
		Response: frame.Kind == simulationFrameResponse, Request: frame.Request, Node: frame.Node,
		Incarnation: frame.Incarnation, Arrivals: frame.Arrivals, Time: frame.Time, Payload: frame.Payload, Error: frame.Error,
	})
	if err != nil {
		return err
	}
	if delivering != nil {
		delivering()
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(encoded)))
	if _, err := destination.Write(header[:]); err != nil {
		return fmt.Errorf("write simulation model frame header: %w", err)
	}
	if _, err := destination.Write(encoded); err != nil {
		return fmt.Errorf("write simulation model frame payload: %w", err)
	}
	return nil
}

func readSimulationModelTransportFrame(source io.Reader) (simulationFrame, error) {
	var header [4]byte
	if _, err := io.ReadFull(source, header[:]); err != nil {
		return simulationFrame{}, err
	}
	size := binary.BigEndian.Uint32(header[:])
	if size == 0 || size > maximumSimulationFrameBytes {
		return simulationFrame{}, errors.New("simulation model frame size is invalid")
	}
	encoded := make([]byte, size)
	if _, err := io.ReadFull(source, encoded); err != nil {
		return simulationFrame{}, fmt.Errorf("read simulation model frame payload: %w", err)
	}
	frame, err := DecodeModelTransportFrame(encoded)
	if err != nil {
		return simulationFrame{}, err
	}
	kind := simulationFrameModel
	if frame.Response {
		kind = simulationFrameResponse
	}
	return simulationFrame{
		Profile: simulationProtocol, Kind: kind, Request: frame.Request, Node: frame.Node,
		Incarnation: frame.Incarnation, Arrivals: frame.Arrivals, Time: frame.Time, Payload: frame.Payload, Error: frame.Error,
	}, nil
}
