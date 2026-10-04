package execution

import (
	"errors"
	"fmt"
)

type simulationProgressEvent interface{ simulationProgressEvent() }

type simulationProgressTransition struct{}

func (simulationProgressTransition) simulationProgressEvent() {}

type coordinatorRequestAccepted struct {
	simulationProgressTransition
	participant *simulationTimeParticipant
	request     uint64
	arrivals    uint32
}
type coordinatorControlForwarded struct {
	simulationProgressTransition
	participant, node *simulationTimeParticipant
	request           uint64
	arrivals          uint32
}
type coordinatorWaitAccepted struct {
	simulationProgressTransition
	participant *simulationTimeParticipant
	request     uint64
	arrivals    uint32
}
type coordinatorWaitSuspended struct {
	simulationProgressTransition
	participant *simulationTimeParticipant
	request     uint64
}
type coordinatorWaitResumed struct {
	simulationProgressTransition
	participant *simulationTimeParticipant
	request     uint64
}
type coordinatorCompletionReserved struct {
	simulationProgressTransition
	participant *simulationTimeParticipant
	request     uint64
}
type coordinatorResponseDelivered struct {
	simulationProgressTransition
	participant *simulationTimeParticipant
	request     uint64
}
type participantRequestAccepted struct {
	simulationProgressTransition
	participant *simulationTimeParticipant
	arrivals    uint32
}
type participantResponseDelivered struct {
	simulationProgressTransition
	participant *simulationTimeParticipant
}
type modelRequestForwarded struct {
	simulationProgressTransition
	source, destination *simulationTimeParticipant
	arrivals            uint32
	current             int64
}
type modelRequestDispatched struct {
	simulationProgressTransition
	coordinator *simulationTimeParticipant
}
type modelDispatchUnavailable struct {
	simulationProgressTransition
	coordinator *simulationTimeParticipant
}
type modelResponseArrived struct {
	simulationProgressTransition
	coordinator, node *simulationTimeParticipant
	arrivals          uint32
}
type modelAbandonedResponseDiscarded struct {
	simulationProgressTransition
	coordinator *simulationTimeParticipant
	arrivals    uint32
}
type arrivalCreditsConsumed struct {
	simulationProgressTransition
	participant *simulationTimeParticipant
	arrivals    uint32
}
type participantRunnable struct {
	simulationProgressTransition
	participant *simulationTimeParticipant
}
type participantRemoved struct {
	simulationProgressTransition
	participant *simulationTimeParticipant
}

type simulationResponsePhase uint8

const (
	simulationResponseReserved simulationResponsePhase = iota + 1
	simulationWaitReserved
	simulationWaitSuspended
)

type simulationResponseIdentity struct {
	participant *simulationTimeParticipant
	request     uint64
}

// Response identities are private coordinator correlation. Aggregate arrival
// credits cannot identify which model operation or response was consumed.
type simulationProgress struct {
	arbiter   *simulationTimeArbiter
	responses map[simulationResponseIdentity]simulationResponsePhase
}

func (progress *simulationProgress) apply(event simulationProgressEvent) error {
	arbiter := progress.arbiter
	arbiter.mu.Lock()
	defer arbiter.mu.Unlock()
	switch event := event.(type) {
	case coordinatorRequestAccepted:
		if err := progress.validateAdmission(event.participant, event.request, event.arrivals); err != nil {
			return err
		}
		progress.consume(event.participant, event.arrivals)
		progress.reserve(event.participant)
		progress.responses[simulationResponseIdentity{event.participant, event.request}] = simulationResponseReserved
	case coordinatorControlForwarded:
		if err := progress.validateForward(event.participant, event.arrivals, 0, event.node); err != nil {
			return err
		}
		identity := simulationResponseIdentity{event.participant, event.request}
		if progress.responses[identity] != 0 {
			return errors.New("simulation response barrier is duplicated")
		}
		progress.forward(event.participant, event.arrivals, 0, event.node)
		progress.responses[identity] = simulationResponseReserved
	case coordinatorWaitAccepted:
		if err := progress.validateAdmission(event.participant, event.request, event.arrivals); err != nil {
			return err
		}
		progress.consume(event.participant, event.arrivals)
		arbiter.runnableLocked(event.participant)
		progress.reserve(event.participant)
		progress.responses[simulationResponseIdentity{event.participant, event.request}] = simulationWaitReserved
	case coordinatorWaitSuspended:
		if err := progress.validateParticipant(event.participant); err != nil {
			return err
		}
		identity := simulationResponseIdentity{event.participant, event.request}
		if progress.responses[identity] != simulationWaitReserved {
			return errors.New("simulation wait response barrier is unavailable")
		}
		if event.participant.external <= event.participant.delivered || event.participant.handling == 0 {
			return errors.New("simulation wait response reservation is unavailable")
		}
		progress.responses[identity] = simulationWaitSuspended
		event.participant.external--
		event.participant.handling--
	case coordinatorWaitResumed:
		if err := progress.validateParticipant(event.participant); err != nil {
			return err
		}
		identity := simulationResponseIdentity{event.participant, event.request}
		phase := progress.responses[identity]
		if phase == simulationWaitReserved {
			return nil
		}
		if phase != simulationWaitSuspended {
			return errors.New("simulation wait response barrier is unavailable")
		}
		progress.responses[identity] = simulationWaitReserved
		progress.reserve(event.participant)
	case coordinatorCompletionReserved:
		if err := progress.validateParticipant(event.participant); err != nil {
			return err
		}
		identity := simulationResponseIdentity{event.participant, event.request}
		if progress.responses[identity] != 0 {
			return nil
		}
		progress.responses[identity] = simulationResponseReserved
		progress.reserve(event.participant)
	case coordinatorResponseDelivered:
		if err := progress.validateParticipant(event.participant); err != nil {
			return err
		}
		identity := simulationResponseIdentity{event.participant, event.request}
		phase := progress.responses[identity]
		if phase == 0 {
			return errors.New("simulation response identity is unknown")
		}
		if phase != simulationWaitSuspended && event.participant.delivered >= event.participant.external {
			return errors.New("simulation time external arrival is unexpected")
		}
		if phase == simulationWaitSuspended {
			progress.reserve(event.participant)
		}
		delete(progress.responses, identity)
		progress.deliver(event.participant)
	case participantRequestAccepted:
		if err := progress.validateCredits(event.participant, event.arrivals); err != nil {
			return err
		}
		progress.consume(event.participant, event.arrivals)
		progress.reserve(event.participant)
	case modelRequestForwarded:
		if err := progress.validateForward(event.source, event.arrivals, event.current, event.destination); err != nil {
			return err
		}
		progress.forward(event.source, event.arrivals, event.current, event.destination)
	case participantResponseDelivered:
		if err := progress.validateDelivery(event.participant); err != nil {
			return err
		}
		progress.deliver(event.participant)
	case modelRequestDispatched:
		if err := progress.validateDelivery(event.coordinator); err != nil {
			return err
		}
		progress.deliver(event.coordinator)
	case modelDispatchUnavailable:
		if err := progress.validateParticipant(event.coordinator); err != nil {
			return err
		}
		if event.coordinator.external <= event.coordinator.delivered || event.coordinator.handling == 0 {
			return errors.New("simulation model dispatch reservation is unavailable")
		}
		event.coordinator.external--
		event.coordinator.handling--
	case modelResponseArrived:
		if event.coordinator == nil || arbiter.participants[event.coordinator.name] != event.coordinator {
			return errors.New("simulation time external arrival source is inactive")
		}
		if event.node == nil || event.node == event.coordinator || arbiter.participants[event.node.name] != event.node {
			return errors.New("simulation time external arrival destination is inactive")
		}
		if err := progress.validateCredits(event.coordinator, event.arrivals); err != nil {
			return err
		}
		if event.node.delivered >= event.node.external {
			return errors.New("simulation time external arrival is unexpected")
		}
		progress.consume(event.coordinator, event.arrivals)
		event.node.delivered++
		arbiter.runnableLocked(event.node)
	case modelAbandonedResponseDiscarded:
		if err := progress.validateCredits(event.coordinator, event.arrivals); err != nil {
			return err
		}
		progress.consume(event.coordinator, event.arrivals)
	case arrivalCreditsConsumed:
		if err := progress.validateCredits(event.participant, event.arrivals); err != nil {
			return err
		}
		progress.consume(event.participant, event.arrivals)
	case participantRunnable:
		if err := progress.validateParticipant(event.participant); err != nil {
			return err
		}
		arbiter.runnableLocked(event.participant)
	case participantRemoved:
		if err := progress.validateParticipant(event.participant); err != nil {
			return err
		}
		if event.participant.waiter != nil {
			event.participant.waiter <- simulationTimeResult{err: errors.New("simulation time participant was removed")}
			event.participant.waiter = nil
		}
		delete(arbiter.participants, event.participant.name)
		for identity := range progress.responses {
			if identity.participant == event.participant {
				delete(progress.responses, identity)
			}
		}
	default:
		return errors.New("simulation progress event is invalid")
	}
	arbiter.settleLocked()
	return nil
}

func (progress *simulationProgress) validateParticipant(participant *simulationTimeParticipant) error {
	if participant == nil || progress.arbiter.participants[participant.name] != participant {
		return errors.New("simulation time participant is inactive")
	}
	return nil
}

func (progress *simulationProgress) validateCredits(participant *simulationTimeParticipant, arrivals uint32) error {
	if err := progress.validateParticipant(participant); err != nil {
		return err
	}
	if uint64(arrivals) > participant.delivered {
		return fmt.Errorf("simulation time request acknowledged unknown external work: participant=%q arrivals=%d external=%d delivered=%d", participant.name, arrivals, participant.external, participant.delivered)
	}
	return nil
}

func (progress *simulationProgress) validateAdmission(participant *simulationTimeParticipant, request uint64, arrivals uint32) error {
	if err := progress.validateCredits(participant, arrivals); err != nil {
		return err
	}
	if progress.responses[simulationResponseIdentity{participant, request}] != 0 {
		return errors.New("simulation response barrier is duplicated")
	}
	return nil
}

func (progress *simulationProgress) validateForward(source *simulationTimeParticipant, arrivals uint32, current int64, destination *simulationTimeParticipant) error {
	if source == destination {
		return errors.New("simulation time participant is inactive")
	}
	if err := progress.validateParticipant(source); err != nil {
		return err
	}
	if err := progress.validateParticipant(destination); err != nil {
		return err
	}
	if current != 0 && current < simulationInitialTime || !progress.arbiter.forward && current > progress.arbiter.current {
		return fmt.Errorf("simulation forwarded time does not match the current epoch: participant=%q current=%d cluster=%d", source.name, current, progress.arbiter.current)
	}
	return progress.validateCredits(source, arrivals)
}

func (progress *simulationProgress) validateDelivery(participant *simulationTimeParticipant) error {
	if err := progress.validateParticipant(participant); err != nil {
		return err
	}
	if participant.delivered >= participant.external {
		return errors.New("simulation time external arrival is unexpected")
	}
	return nil
}

func (progress *simulationProgress) consume(participant *simulationTimeParticipant, arrivals uint32) {
	participant.external -= uint64(arrivals)
	participant.delivered -= uint64(arrivals)
}

func (progress *simulationProgress) reserve(participant *simulationTimeParticipant) {
	participant.external++
	participant.handling++
	progress.arbiter.externalLocked(participant)
}

func (progress *simulationProgress) deliver(participant *simulationTimeParticipant) {
	participant.delivered++
	if participant.handling != 0 {
		participant.handling--
	}
	progress.arbiter.runnableLocked(participant)
}

func (progress *simulationProgress) forward(source *simulationTimeParticipant, arrivals uint32, current int64, destination *simulationTimeParticipant) {
	progress.arbiter.current = max(progress.arbiter.current, current)
	progress.consume(source, arrivals)
	source.external++
	progress.arbiter.externalLocked(source)
	progress.reserve(destination)
}
