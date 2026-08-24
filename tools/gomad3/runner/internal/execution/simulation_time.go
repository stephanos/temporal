package execution

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"sync"
)

const simulationInitialTime int64 = 946684800000000000
const simulationTimeRequestBytes = 40
const simulationTimeResponseBytes = 32

var simulationTimeRequestMagic = [8]byte{'G', 'O', 'M', 'A', 'D', 'T', 'Q', 1}
var simulationTimeResponseMagic = [8]byte{'G', 'O', 'M', 'A', 'D', 'T', 'R', 1}

type simulationTimeResponseKind uint8

const (
	simulationTimeAdvance simulationTimeResponseKind = iota + 1
	simulationTimeRetry
	simulationTimeDeadlock
	simulationTimeExternal
)

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

func encodeSimulationTimeRequest(request simulationTimeRequest) ([]byte, error) {
	if err := validateSimulationTimeRequest(request); err != nil {
		return nil, err
	}
	encoded := make([]byte, simulationTimeRequestBytes)
	copy(encoded[:8], simulationTimeRequestMagic[:])
	binary.BigEndian.PutUint64(encoded[8:16], request.Generation)
	binary.BigEndian.PutUint64(encoded[16:24], uint64(request.Current))
	binary.BigEndian.PutUint64(encoded[24:32], uint64(request.Deadline))
	binary.BigEndian.PutUint32(encoded[32:36], request.Arrivals)
	return encoded, nil
}

func decodeSimulationTimeRequest(encoded []byte) (simulationTimeRequest, error) {
	if len(encoded) != simulationTimeRequestBytes || !bytes.Equal(encoded[:8], simulationTimeRequestMagic[:]) || !zeroSimulationTime(encoded[36:40]) {
		return simulationTimeRequest{}, errors.New("simulation time request frame is invalid")
	}
	request := simulationTimeRequest{
		Generation: binary.BigEndian.Uint64(encoded[8:16]),
		Current:    int64(binary.BigEndian.Uint64(encoded[16:24])),
		Deadline:   int64(binary.BigEndian.Uint64(encoded[24:32])),
		Arrivals:   binary.BigEndian.Uint32(encoded[32:36]),
	}
	if err := validateSimulationTimeRequest(request); err != nil {
		return simulationTimeRequest{}, err
	}
	return request, nil
}

func validateSimulationTimeRequest(request simulationTimeRequest) error {
	if request.Generation == 0 || request.Current < simulationInitialTime || request.Deadline < request.Current {
		return errors.New("simulation time request is invalid")
	}
	return nil
}

func encodeSimulationTimeResponse(response simulationTimeResponse) ([]byte, error) {
	if err := validateSimulationTimeResponse(response); err != nil {
		return nil, err
	}
	encoded := make([]byte, simulationTimeResponseBytes)
	copy(encoded[:8], simulationTimeResponseMagic[:])
	binary.BigEndian.PutUint64(encoded[8:16], response.Generation)
	binary.BigEndian.PutUint64(encoded[16:24], uint64(response.Time))
	encoded[24] = byte(response.Kind)
	return encoded, nil
}

func decodeSimulationTimeResponse(encoded []byte) (simulationTimeResponse, error) {
	if len(encoded) != simulationTimeResponseBytes || !bytes.Equal(encoded[:8], simulationTimeResponseMagic[:]) || !zeroSimulationTime(encoded[25:32]) {
		return simulationTimeResponse{}, errors.New("simulation time response frame is invalid")
	}
	response := simulationTimeResponse{
		Generation: binary.BigEndian.Uint64(encoded[8:16]),
		Time:       int64(binary.BigEndian.Uint64(encoded[16:24])),
		Kind:       simulationTimeResponseKind(encoded[24]),
	}
	if err := validateSimulationTimeResponse(response); err != nil {
		return simulationTimeResponse{}, err
	}
	return response, nil
}

func validateSimulationTimeResponse(response simulationTimeResponse) error {
	if response.Generation == 0 || response.Time < simulationInitialTime {
		return errors.New("simulation time response is invalid")
	}
	switch response.Kind {
	case simulationTimeAdvance, simulationTimeRetry, simulationTimeDeadlock, simulationTimeExternal:
		return nil
	default:
		return errors.New("simulation time response kind is invalid")
	}
}

func zeroSimulationTime(value []byte) bool {
	for _, current := range value {
		if current != 0 {
			return false
		}
	}
	return true
}

func encodeSimulationActivationTime(current int64) []byte {
	encoded := make([]byte, 8)
	binary.BigEndian.PutUint64(encoded, uint64(current))
	return encoded
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
	participants map[string]*simulationTimeParticipant
}

func newSimulationTimeArbiter() *simulationTimeArbiter {
	return &simulationTimeArbiter{current: simulationInitialTime, participants: make(map[string]*simulationTimeParticipant)}
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
	if request.Generation != participant.generation+1 || request.Deadline < request.Current || request.Current > arbiter.current {
		err := fmt.Errorf("simulation time request does not match the current epoch: participant=%q active=%t generation=%d want=%d current=%d cluster=%d deadline=%d", participant.name, participant.active, request.Generation, participant.generation+1, request.Current, arbiter.current, request.Deadline)
		arbiter.mu.Unlock()
		return simulationTimeResponse{}, err
	}
	if uint64(request.Arrivals) > participant.delivered {
		arbiter.mu.Unlock()
		return simulationTimeResponse{}, fmt.Errorf("simulation time request acknowledged unknown external work: participant=%q arrivals=%d external=%d delivered=%d", participant.name, request.Arrivals, participant.external, participant.delivered)
	}
	participant.generation = request.Generation
	participant.external -= uint64(request.Arrivals)
	participant.delivered -= uint64(request.Arrivals)
	if request.Current < arbiter.current {
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

func (arbiter *simulationTimeArbiter) runnable(participant *simulationTimeParticipant) {
	arbiter.mu.Lock()
	defer arbiter.mu.Unlock()
	arbiter.runnableLocked(participant)
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

func (arbiter *simulationTimeArbiter) beginExternal(participant *simulationTimeParticipant) {
	arbiter.mu.Lock()
	defer arbiter.mu.Unlock()
	if participant == nil || arbiter.participants[participant.name] != participant {
		return
	}
	participant.external++
	arbiter.externalLocked(participant)
	arbiter.settleLocked()
}

func (arbiter *simulationTimeArbiter) beginHandledExternal(participant *simulationTimeParticipant) {
	arbiter.mu.Lock()
	defer arbiter.mu.Unlock()
	if participant == nil || arbiter.participants[participant.name] != participant {
		return
	}
	participant.external++
	participant.handling++
	arbiter.externalLocked(participant)
	arbiter.settleLocked()
}

func (arbiter *simulationTimeArbiter) beginExternalAfterArrivals(participant *simulationTimeParticipant, arrivals uint32) error {
	arbiter.mu.Lock()
	defer arbiter.mu.Unlock()
	if participant == nil || arbiter.participants[participant.name] != participant {
		return errors.New("simulation time participant is inactive")
	}
	if uint64(arrivals) > participant.delivered {
		return fmt.Errorf("simulation time request acknowledged unknown external work: participant=%q arrivals=%d external=%d delivered=%d", participant.name, arrivals, participant.external, participant.delivered)
	}
	participant.external -= uint64(arrivals)
	participant.delivered -= uint64(arrivals)
	participant.external++
	participant.handling++
	arbiter.externalLocked(participant)
	return nil
}

func (arbiter *simulationTimeArbiter) forwardExternalAfterArrivals(source *simulationTimeParticipant, arrivals uint32, destination *simulationTimeParticipant) error {
	arbiter.mu.Lock()
	defer arbiter.mu.Unlock()
	if source == nil || destination == nil || source == destination || arbiter.participants[source.name] != source || arbiter.participants[destination.name] != destination {
		return errors.New("simulation time participant is inactive")
	}
	if uint64(arrivals) > source.delivered {
		return fmt.Errorf("simulation time request acknowledged unknown external work: participant=%q arrivals=%d external=%d delivered=%d", source.name, arrivals, source.external, source.delivered)
	}
	source.external -= uint64(arrivals)
	source.delivered -= uint64(arrivals)
	source.external++
	destination.external++
	destination.handling++
	arbiter.externalLocked(source)
	arbiter.externalLocked(destination)
	arbiter.settleLocked()
	return nil
}

func (arbiter *simulationTimeArbiter) transferExternalArrival(source *simulationTimeParticipant, arrivals uint32, destination *simulationTimeParticipant) error {
	arbiter.mu.Lock()
	defer arbiter.mu.Unlock()
	if source == nil || arbiter.participants[source.name] != source {
		return errors.New("simulation time external arrival source is inactive")
	}
	if destination == nil || source == destination || arbiter.participants[destination.name] != destination {
		return errors.New("simulation time external arrival destination is inactive")
	}
	if uint64(arrivals) > source.delivered {
		return fmt.Errorf("simulation time request acknowledged unknown external work: participant=%q arrivals=%d external=%d delivered=%d", source.name, arrivals, source.external, source.delivered)
	}
	if destination.delivered >= destination.external {
		return errors.New("simulation time external arrival is unexpected")
	}
	source.external -= uint64(arrivals)
	source.delivered -= uint64(arrivals)
	destination.delivered++
	arbiter.runnableLocked(destination)
	arbiter.settleLocked()
	return nil
}

func (arbiter *simulationTimeArbiter) acknowledgeExternal(participant *simulationTimeParticipant, arrivals uint32) error {
	arbiter.mu.Lock()
	defer arbiter.mu.Unlock()
	if participant == nil || arbiter.participants[participant.name] != participant {
		return errors.New("simulation time participant is inactive")
	}
	if uint64(arrivals) > participant.delivered {
		return fmt.Errorf("simulation time request acknowledged unknown external work: participant=%q arrivals=%d external=%d delivered=%d", participant.name, arrivals, participant.external, participant.delivered)
	}
	participant.external -= uint64(arrivals)
	participant.delivered -= uint64(arrivals)
	arbiter.settleLocked()
	return nil
}

func (arbiter *simulationTimeArbiter) deliverExternal(participant *simulationTimeParticipant) {
	arbiter.mu.Lock()
	defer arbiter.mu.Unlock()
	if participant == nil || arbiter.participants[participant.name] != participant || participant.delivered >= participant.external {
		return
	}
	participant.delivered++
	if participant.handling != 0 {
		participant.handling--
	}
	arbiter.runnableLocked(participant)
}

func (arbiter *simulationTimeArbiter) endExternal(participant *simulationTimeParticipant) {
	arbiter.mu.Lock()
	defer arbiter.mu.Unlock()
	if participant == nil || arbiter.participants[participant.name] != participant || participant.external == 0 {
		return
	}
	participant.external--
	if participant.handling != 0 {
		participant.handling--
	}
	arbiter.settleLocked()
}

func (arbiter *simulationTimeArbiter) remove(participant *simulationTimeParticipant) {
	arbiter.mu.Lock()
	defer arbiter.mu.Unlock()
	if participant == nil || arbiter.participants[participant.name] != participant {
		return
	}
	if participant.waiter != nil {
		participant.waiter <- simulationTimeResult{err: errors.New("simulation time participant was removed")}
	}
	delete(arbiter.participants, participant.name)
	arbiter.settleLocked()
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
		arbiter.current = deadline
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
