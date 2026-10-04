package execution

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"sync"
)

const simulationInitialTime int64 = 946684800000000000

type simulationTimeResponseKind uint8

type simulationTimeRequest struct {
	Generation uint64
	Current    int64
	Deadline   int64
	Arrivals   uint32
}

type simulationTimeResponse struct {
	Generation uint64
	Kind       simulationTimeResponseKind
	Time       int64
}

func encodeSimulationActivationTime(current int64) []byte {
	encoded := make([]byte, 8)
	binary.BigEndian.PutUint64(encoded, uint64(current))
	return encoded
}

func decodeSimulationActivationTime(encoded []byte) (int64, error) {
	if len(encoded) != 8 {
		return 0, errors.New("process simulation activation time is invalid")
	}
	current := int64(binary.BigEndian.Uint64(encoded))
	if current < simulationInitialTime {
		return 0, errors.New("process simulation activation time is invalid")
	}
	return current, nil
}

func serveSimulationTime(ctx context.Context, source io.Reader, destination io.Writer, handler func(context.Context, simulationTimeRequest) (simulationTimeResponse, error)) error {
	if handler == nil {
		return errors.New("simulation time handler is unavailable")
	}
	for {
		encodedRequest := make([]byte, simulationTimeRequestBytes)
		if _, err := io.ReadFull(source, encodedRequest); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("read simulation time request: %w", err)
		}
		request, err := decodeSimulationTimeRequest(encodedRequest)
		if err != nil {
			return err
		}
		response, err := handler(ctx, request)
		if err != nil {
			return err
		}
		encodedResponse, err := encodeSimulationTimeResponse(response)
		if err != nil {
			return err
		}
		written, err := destination.Write(encodedResponse)
		if err != nil {
			return fmt.Errorf("write simulation time response: %w", err)
		}
		if written != len(encodedResponse) {
			return io.ErrShortWrite
		}
	}
}

type simulationTimeResult struct {
	response simulationTimeResponse
	err      error
}

type simulationTimeParticipant struct {
	name       string
	active     bool
	external   uint64
	delivered  uint64
	handling   uint64
	generation uint64
	deadline   int64
	waiter     chan simulationTimeResult
}

type simulationTimeArbiter struct {
	mu           sync.Mutex
	current      int64
	forward      bool
	participants map[string]*simulationTimeParticipant
	progress     *simulationProgress
}

func newSimulationTimeArbiter(forward bool) *simulationTimeArbiter {
	arbiter := &simulationTimeArbiter{current: simulationInitialTime, forward: forward, participants: make(map[string]*simulationTimeParticipant)}
	arbiter.progress = &simulationProgress{arbiter: arbiter, responses: make(map[simulationResponseIdentity]simulationResponsePhase)}
	return arbiter
}

func (arbiter *simulationTimeArbiter) register(name string) (*simulationTimeParticipant, error) {
	if name == "" || len(name) > 512 {
		return nil, errors.New("simulation time participant identity is invalid")
	}
	arbiter.mu.Lock()
	defer arbiter.mu.Unlock()
	if arbiter.participants[name] != nil {
		return nil, fmt.Errorf("simulation time participant %q is duplicated", name)
	}
	participant := &simulationTimeParticipant{name: name}
	arbiter.participants[name] = participant
	return participant, nil
}

func (arbiter *simulationTimeArbiter) activate(participant *simulationTimeParticipant) int64 {
	arbiter.mu.Lock()
	defer arbiter.mu.Unlock()
	if participant != nil && arbiter.participants[participant.name] == participant {
		participant.active = true
	}
	return arbiter.current
}

func (arbiter *simulationTimeArbiter) activateAt(participant *simulationTimeParticipant, current int64) (int64, error) {
	arbiter.mu.Lock()
	defer arbiter.mu.Unlock()
	if participant == nil || arbiter.participants[participant.name] != participant {
		return 0, errors.New("simulation time participant is inactive")
	}
	if current < simulationInitialTime || !arbiter.forward && current > arbiter.current {
		return 0, fmt.Errorf("simulation activation time does not match the current epoch: participant=%q current=%d cluster=%d", participant.name, current, arbiter.current)
	}
	if current > arbiter.current {
		arbiter.current = current
	}
	participant.active = true
	return arbiter.current, nil
}

func (arbiter *simulationTimeArbiter) quiesce(ctx context.Context, participant *simulationTimeParticipant, request simulationTimeRequest) (simulationTimeResponse, error) {
	if ctx == nil {
		return simulationTimeResponse{}, errors.New("simulation time context is nil")
	}
	arbiter.mu.Lock()
	if participant == nil || arbiter.participants[participant.name] != participant {
		arbiter.mu.Unlock()
		return simulationTimeResponse{}, errors.New("simulation time participant is inactive")
	}
	if participant.waiter != nil {
		arbiter.mu.Unlock()
		return simulationTimeResponse{}, errors.New("simulation time participant is already quiescent")
	}
	if request.Generation != participant.generation+1 || request.Deadline < request.Current || !arbiter.forward && request.Current > arbiter.current {
		err := fmt.Errorf("simulation time request does not match the current epoch: participant=%q active=%t generation=%d want=%d current=%d cluster=%d deadline=%d", participant.name, participant.active, request.Generation, participant.generation+1, request.Current, arbiter.current, request.Deadline)
		arbiter.mu.Unlock()
		return simulationTimeResponse{}, err
	}
	if err := arbiter.progress.validateCredits(participant, request.Arrivals); err != nil {
		arbiter.mu.Unlock()
		return simulationTimeResponse{}, err
	}
	if request.Current > arbiter.current {
		arbiter.current = request.Current
	}
	participant.generation = request.Generation
	arbiter.progress.consume(participant, request.Arrivals)
	if request.Current < arbiter.current && !arbiter.forward {
		response := simulationTimeResponse{Generation: request.Generation, Kind: simulationTimeAdvance, Time: arbiter.current}
		arbiter.mu.Unlock()
		return response, nil
	}
	if !participant.active {
		response := simulationTimeResponse{Generation: request.Generation, Kind: simulationTimeAdvance, Time: arbiter.current}
		arbiter.mu.Unlock()
		return response, nil
	}
	if participant.delivered != 0 {
		response := simulationTimeResponse{Generation: request.Generation, Kind: simulationTimeRetry, Time: arbiter.current}
		arbiter.mu.Unlock()
		return response, nil
	}
	if participant.external != 0 {
		response := simulationTimeResponse{Generation: request.Generation, Kind: simulationTimeExternal, Time: arbiter.current}
		arbiter.mu.Unlock()
		return response, nil
	}
	participant.deadline = request.Deadline
	waiter := make(chan simulationTimeResult, 1)
	participant.waiter = waiter
	arbiter.settleLocked()
	arbiter.mu.Unlock()

	select {
	case result := <-waiter:
		return result.response, result.err
	case <-ctx.Done():
		arbiter.mu.Lock()
		if participant.waiter == waiter {
			participant.waiter = nil
		}
		arbiter.mu.Unlock()
		return simulationTimeResponse{}, ctx.Err()
	}
}

func (arbiter *simulationTimeArbiter) runnableLocked(participant *simulationTimeParticipant) {
	if participant == nil || arbiter.participants[participant.name] != participant || participant.waiter == nil {
		return
	}
	waiter := participant.waiter
	participant.waiter = nil
	waiter <- simulationTimeResult{response: simulationTimeResponse{
		Generation: participant.generation,
		Kind:       simulationTimeRetry,
		Time:       arbiter.current,
	}}
}

func (arbiter *simulationTimeArbiter) externalLocked(participant *simulationTimeParticipant) {
	if participant == nil || arbiter.participants[participant.name] != participant || participant.waiter == nil {
		return
	}
	waiter := participant.waiter
	participant.waiter = nil
	waiter <- simulationTimeResult{response: simulationTimeResponse{
		Generation: participant.generation,
		Kind:       simulationTimeExternal,
		Time:       arbiter.current,
	}}
}

func (arbiter *simulationTimeArbiter) settleLocked() {
	deadline := int64(math.MaxInt64)
	active := 0
	for _, participant := range arbiter.participants {
		if !participant.active {
			return
		}
		if participant.handling != 0 {
			return
		}
		if participant.delivered != 0 {
			return
		}
		if participant.external != 0 {
			continue
		}
		active++
		if participant.waiter == nil {
			return
		}
	}
	if active == 0 {
		return
	}
	for _, participant := range arbiter.participants {
		if participant.active && participant.external == 0 && participant.waiter != nil {
			deadline = min(deadline, participant.deadline)
		}
	}
	if deadline != math.MaxInt64 {
		arbiter.current = max(arbiter.current, deadline)
	}
	for _, participant := range arbiter.participants {
		if !participant.active || participant.external != 0 {
			continue
		}
		kind := simulationTimeAdvance
		if deadline == math.MaxInt64 {
			kind = simulationTimeDeadlock
		}
		participant.waiter <- simulationTimeResult{response: simulationTimeResponse{
			Generation: participant.generation,
			Kind:       kind,
			Time:       arbiter.current,
		}}
		participant.waiter = nil
	}
}

func (arbiter *simulationTimeArbiter) currentTime() int64 {
	arbiter.mu.Lock()
	defer arbiter.mu.Unlock()
	return arbiter.current
}

func (arbiter *simulationTimeArbiter) isQuiescent(participant *simulationTimeParticipant) bool {
	arbiter.mu.Lock()
	defer arbiter.mu.Unlock()
	return participant != nil && arbiter.participants[participant.name] == participant && participant.waiter != nil
}
