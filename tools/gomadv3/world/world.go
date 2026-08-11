package world

import (
	"container/heap"
	"fmt"
	"math"
	"regexp"
	"sort"
	"sync"
	"unicode/utf8"
)

type RequestState string

const (
	RequestPending   RequestState = "pending"
	RequestQueued    RequestState = "queued"
	RequestCanceled  RequestState = "canceled"
	RequestDelivered RequestState = "delivered"
)

type EventState string

const (
	EventQueued    EventState = "queued"
	EventCanceled  EventState = "canceled"
	EventDelivered EventState = "delivered"
)

type requestState struct {
	id      RequestID
	request Request
	state   RequestState
	eventID EventID
}

type eventState struct {
	id         EventID
	readiness  Readiness
	request    *requestState
	state      EventState
	heapIndex  int
	choiceRank [32]byte
}

type replayState struct {
	plan   ReplayPlan
	cursor uint64
}

type World struct {
	mu sync.Mutex

	config             Config
	now                LogicalTime
	nextRequestID      RequestID
	nextEventID        EventID
	nextTransition     Sequence
	payloadBytes       uint64
	requests           map[RequestID]*requestState
	events             map[EventID]*eventState
	queue              eventHeap
	history            []Transition
	transcript         Digest
	replay             *replayState
	replayPayloadBytes uint64
	recording          *recordingState
}

var resourceComponentPattern = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,63}$`)

func New(config Config) (*World, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	return newWorld(config), nil
}

func (w *World) Seed() Seed {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.config.Seed
}

func newWorld(config Config) *World {
	return &World{
		config: config, now: InitialTime, nextRequestID: 1, nextEventID: 1, nextTransition: 1,
		requests: make(map[RequestID]*requestState), events: make(map[EventID]*eventState), transcript: emptyDigest(),
	}
}

func (w *World) Register(request Request) (RequestID, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := validateRequest(w.config.Limits, request); err != nil {
		return 0, err
	}
	if w.nextRequestID == 0 || w.nextRequestID == RequestID(math.MaxUint64) {
		return 0, capacity("request-ids", math.MaxUint64, math.MaxUint64, 1)
	}
	if err := checkCapacity("requests", w.config.Limits.MaxRequests, uint64(len(w.requests)), 1); err != nil {
		return 0, err
	}
	payloadDelta := uint64(len(request.Payload)) * 2
	if err := w.checkPayload(payloadDelta); err != nil {
		return 0, err
	}
	if err := w.checkTransition(); err != nil {
		return 0, err
	}
	id := w.nextRequestID
	copied := copyRequest(request)
	transition := w.nextRegisterTransition(copied, id)
	if err := w.checkRecordingTransition(transition); err != nil {
		return 0, err
	}
	if err := w.checkReplay(transition); err != nil {
		return 0, err
	}
	w.requests[id] = &requestState{id: id, request: copied, state: RequestPending}
	if w.nextRequestID != RequestID(math.MaxUint64) {
		w.nextRequestID++
	}
	w.payloadBytes += payloadDelta
	w.commitTransition(transition)
	return id, nil
}

func (w *World) Ready(readiness Readiness) (EventID, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	request, err := w.lookupPending(readiness.RequestID)
	if err != nil {
		return 0, err
	}
	if err := validateReadiness(w.config.Limits, readiness); err != nil {
		return 0, err
	}
	if readiness.At < w.now {
		return 0, fmt.Errorf("%w: readiness.at", ErrTimeRegression)
	}
	if w.nextEventID == 0 || w.nextEventID == EventID(math.MaxUint64) {
		return 0, capacity("event-ids", math.MaxUint64, math.MaxUint64, 1)
	}
	if err := checkCapacity("events", w.config.Limits.MaxEvents, uint64(len(w.events)), 1); err != nil {
		return 0, err
	}
	if err := checkCapacity("queued-events", w.config.Limits.MaxQueuedEvents, uint64(len(w.queue)), 1); err != nil {
		return 0, err
	}
	payloadDelta := uint64(len(readiness.Payload)) * 2
	if err := w.checkPayload(payloadDelta); err != nil {
		return 0, err
	}
	if err := w.checkTransition(); err != nil {
		return 0, err
	}
	id := w.nextEventID
	copied := copyReadiness(readiness)
	event := &eventState{id: id, readiness: copied, request: request, state: EventQueued, heapIndex: -1}
	if copied.EquivalenceClass != "" {
		event.choiceRank = choiceRank(w.config.Seed, copied.EquivalenceClass, id)
	}
	transition := w.nextReadyTransition(copied, id)
	if err := w.checkRecordingTransition(transition); err != nil {
		return 0, err
	}
	if err := w.checkReplay(transition); err != nil {
		return 0, err
	}
	w.events[id] = event
	request.state = RequestQueued
	request.eventID = id
	heap.Push(&w.queue, event)
	if w.nextEventID != EventID(math.MaxUint64) {
		w.nextEventID++
	}
	w.payloadBytes += payloadDelta
	w.commitTransition(transition)
	return id, nil
}

func (w *World) Cancel(requestID RequestID) (Cancellation, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	request, found := w.requests[requestID]
	if requestID == 0 || !found {
		return Cancellation{}, fmt.Errorf("%w: requestId", ErrUnknownRequest)
	}
	if err := w.checkTransition(); err != nil {
		return Cancellation{}, err
	}
	result := Cancellation{RequestID: requestID, EventID: request.eventID}
	switch request.state {
	case RequestPending, RequestQueued:
		result.Status = CancelWon
	case RequestCanceled:
		result.Status = CancelAlreadyCanceled
	case RequestDelivered:
		result.Status = CancelAlreadyDelivered
	default:
		panic("invalid request state")
	}
	transition := w.nextCancelTransition(requestID, result)
	if err := w.checkRecordingTransition(transition); err != nil {
		return Cancellation{}, err
	}
	if err := w.checkReplay(transition); err != nil {
		return Cancellation{}, err
	}
	if result.Status == CancelWon {
		request.state = RequestCanceled
		if request.eventID != 0 {
			event := w.events[request.eventID]
			if event.state != EventQueued || event.heapIndex < 0 {
				panic("queued request has no queued event")
			}
			heap.Remove(&w.queue, event.heapIndex)
			event.state = EventCanceled
		}
	}
	w.commitTransition(transition)
	return result, nil
}

func (w *World) Quiesce() (Quiescence, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.checkTransition(); err != nil {
		return Quiescence{}, err
	}
	result := Quiescence{Before: w.now, After: w.now}
	var batch []*eventState
	if len(w.queue) > 0 {
		at := w.queue[0].readiness.At
		for _, event := range w.queue {
			if event.readiness.At == at {
				batch = append(batch, event)
			}
		}
		sort.Slice(batch, func(i, j int) bool { return lessEvent(batch[i], batch[j]) })
		result.Kind = QuiescenceDelivered
		result.After = at
		result.Deliveries = make([]Delivery, len(batch))
		var payloadDelta uint64
		for index, event := range batch {
			result.Deliveries[index] = deliveryOf(event)
			payloadDelta += uint64(len(event.readiness.Payload))
		}
		if err := w.checkPayload(payloadDelta); err != nil {
			return Quiescence{}, err
		}
	} else {
		for id, request := range w.requests {
			if request.state == RequestPending {
				result.Blocked = append(result.Blocked, id)
			}
		}
		sort.Slice(result.Blocked, func(i, j int) bool { return result.Blocked[i] < result.Blocked[j] })
		if len(result.Blocked) > 0 {
			result.Kind = QuiescenceDeadlock
		} else {
			result.Kind = QuiescenceIdle
		}
	}
	transition := w.nextQuiesceTransition(result)
	if err := w.checkRecordingTransition(transition); err != nil {
		return Quiescence{}, err
	}
	if err := w.checkReplay(transition); err != nil {
		return Quiescence{}, err
	}
	if result.Kind == QuiescenceDelivered {
		for _, event := range batch {
			heap.Remove(&w.queue, event.heapIndex)
			event.state = EventDelivered
			event.request.state = RequestDelivered
		}
		w.now = result.After
		for _, delivery := range result.Deliveries {
			w.payloadBytes += uint64(len(delivery.Payload))
		}
	}
	w.commitTransition(transition)
	return copyQuiescence(result), nil
}

func (w *World) ReplayProgress() ReplayProgress {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.replay == nil {
		return ReplayProgress{}
	}
	return ReplayProgress{Cursor: w.replay.cursor, Expected: uint64(len(w.replay.plan.Transitions))}
}

func validateConfig(config Config) error {
	limits := config.Limits
	if limits.MaxRequests == 0 || limits.MaxEvents == 0 || limits.MaxQueuedEvents == 0 || limits.MaxTransitions == 0 || limits.MaxPayloadBytes == 0 || limits.MaxStringBytes == 0 {
		return fmt.Errorf("%w: every limit must be nonzero", ErrInvalidConfig)
	}
	if limits.MaxQueuedEvents > limits.MaxEvents {
		return fmt.Errorf("%w: maxQueuedEvents exceeds maxEvents", ErrInvalidConfig)
	}
	return nil
}

func validateRequest(limits Limits, request Request) error {
	if err := validateString(limits, "request.kind", request.Kind, false); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}
	if !resourceComponentPattern.MatchString(request.Resource.Adapter) || !resourceComponentPattern.MatchString(request.Resource.Kind) || uint64(len(request.Resource.Adapter)) > uint64(limits.MaxStringBytes) || uint64(len(request.Resource.Kind)) > uint64(limits.MaxStringBytes) {
		return fmt.Errorf("%w: request.resource", ErrInvalidRequest)
	}
	if err := validateString(limits, "request.resource.key", request.Resource.Key, true); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}
	if uint64(len(request.Payload)) > limits.MaxPayloadBytes {
		return capacity("payload-bytes", limits.MaxPayloadBytes, 0, uint64(len(request.Payload)))
	}
	return nil
}

func validateReadiness(limits Limits, readiness Readiness) error {
	if readiness.RequestID == 0 {
		return fmt.Errorf("%w: readiness.requestId", ErrUnknownRequest)
	}
	if err := validateString(limits, "readiness.kind", readiness.Kind, false); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}
	if err := validateString(limits, "readiness.equivalenceClass", readiness.EquivalenceClass, true); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidRequest, err)
	}
	return nil
}

func validateString(limits Limits, field, value string, emptyAllowed bool) error {
	if !utf8.ValidString(value) || !emptyAllowed && value == "" || uint64(len(value)) > uint64(limits.MaxStringBytes) {
		return fmt.Errorf("invalid %s", field)
	}
	return nil
}

func (w *World) lookupPending(id RequestID) (*requestState, error) {
	request, found := w.requests[id]
	if id == 0 || !found {
		return nil, fmt.Errorf("%w: requestId", ErrUnknownRequest)
	}
	if request.state != RequestPending {
		return nil, fmt.Errorf("%w: request %d is %s", ErrRequestState, id, request.state)
	}
	return request, nil
}

func checkCapacity(dimension string, limit, used, delta uint64) error {
	if used > limit || delta > limit-used {
		return capacity(dimension, limit, used, delta)
	}
	return nil
}

func capacity(dimension string, limit, used, delta uint64) error {
	return &CapacityError{Dimension: dimension, Limit: limit, Used: used, Delta: delta}
}

func (w *World) checkPayload(delta uint64) error {
	used := w.payloadBytes + w.replayPayloadBytes
	if w.replay != nil && w.replay.cursor < uint64(len(w.replay.plan.Transitions)) {
		used -= transitionPayloadSize(w.replay.plan.Transitions[w.replay.cursor])
	}
	return checkCapacity("payload-bytes", w.config.Limits.MaxPayloadBytes, used, delta)
}

func (w *World) checkTransition() error {
	if w.nextTransition == 0 || w.nextTransition == Sequence(math.MaxUint64) {
		return capacity("transition-sequences", math.MaxUint64, math.MaxUint64, 1)
	}
	return checkCapacity("transitions", w.config.Limits.MaxTransitions, uint64(len(w.history)), 1)
}

func copyRequest(request Request) Request {
	request.Payload = append([]byte(nil), request.Payload...)
	return request
}

func copyReadiness(readiness Readiness) Readiness {
	readiness.Payload = append([]byte(nil), readiness.Payload...)
	return readiness
}

func deliveryOf(event *eventState) Delivery {
	return Delivery{RequestID: event.readiness.RequestID, EventID: event.id, At: event.readiness.At, Kind: event.readiness.Kind, Payload: append([]byte(nil), event.readiness.Payload...)}
}

func copyQuiescence(input Quiescence) Quiescence {
	result := input
	result.Blocked = append([]RequestID(nil), input.Blocked...)
	result.Deliveries = make([]Delivery, len(input.Deliveries))
	for index, delivery := range input.Deliveries {
		result.Deliveries[index] = delivery
		result.Deliveries[index].Payload = append([]byte(nil), delivery.Payload...)
	}
	return result
}
