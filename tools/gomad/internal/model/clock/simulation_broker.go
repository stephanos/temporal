package clock

import (
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
)

//go:embed simulation_broker.go simulation_progress.go simulation_time.go
var brokerImplementationSources embed.FS

func BrokerImplementationSHA256() string {
	return brokerImplementationSHA256(brokerImplementationSources)
}

func brokerImplementationSHA256(sources fs.ReadFileFS) string {
	data := []byte("gomad-simulation-clock-implementation/v1\n")
	for _, name := range []string{"simulation_broker.go", "simulation_progress.go", "simulation_time.go"} {
		source, err := sources.ReadFile(name)
		if err != nil {
			return ""
		}
		data = fmt.Appendf(data, "%s:%x\n", name, sha256.Sum256(source))
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}

// BrokerClock exposes the existing simulation progress owner to isolated engines.
type BrokerClock struct{ arbiter *simulationTimeArbiter }

type BrokerParticipant struct{ participant *simulationTimeParticipant }

type BrokerTimeRequest struct {
	Generation        uint64
	Current, Deadline int64
	Arrivals          uint32
}

type BrokerTimeResponse struct {
	Generation uint64
	Kind       uint8
	Time       int64
}

func NewBrokerClock() *BrokerClock {
	owner := newSimulationTimeArbiter(false)
	// Cooperative Go reserves zero as an uninitialized monotonic timestamp.
	owner.current++
	return &BrokerClock{owner}
}

func (clock *BrokerClock) Register(name string) (*BrokerParticipant, error) {
	p, err := clock.arbiter.register(name)
	if err != nil {
		return nil, err
	}
	clock.arbiter.activate(p)
	return &BrokerParticipant{p}, nil
}

func (clock *BrokerClock) Current() int64 { return clock.arbiter.currentTime() }

func (clock *BrokerClock) Quiesce(ctx context.Context, p *BrokerParticipant, request BrokerTimeRequest) (BrokerTimeResponse, error) {
	r, err := clock.arbiter.quiesce(ctx, participantFor(p), simulationTimeRequest(request))
	return BrokerTimeResponse{r.Generation, uint8(r.Kind), r.Time}, err
}

func (clock *BrokerClock) Accepted(p *BrokerParticipant, id uint64, arrivals uint32) error {
	return clock.arbiter.progress.apply(coordinatorRequestAccepted{participant: participantFor(p), request: id, arrivals: arrivals})
}

func (clock *BrokerClock) Delivered(p *BrokerParticipant, id uint64) error {
	return clock.arbiter.progress.apply(coordinatorResponseDelivered{participant: participantFor(p), request: id})
}

func (clock *BrokerClock) Consume(p *BrokerParticipant, arrivals uint32) error {
	return clock.arbiter.progress.apply(arrivalCreditsConsumed{participant: participantFor(p), arrivals: arrivals})
}

func (clock *BrokerClock) Remove(p *BrokerParticipant) error {
	return clock.arbiter.progress.apply(participantRemoved{participant: participantFor(p)})
}

func NewProcessClock(forward bool) *BrokerClock {
	return &BrokerClock{newSimulationTimeArbiter(forward)}
}

func (clock *BrokerClock) RegisterInactive(name string) (*BrokerParticipant, error) {
	p, err := clock.arbiter.register(name)
	if err != nil {
		return nil, err
	}
	return &BrokerParticipant{p}, nil
}
func (clock *BrokerClock) Activate(p *BrokerParticipant) int64 {
	return clock.arbiter.activate(participantFor(p))
}
func (clock *BrokerClock) ActivateAt(p *BrokerParticipant, current int64) (int64, error) {
	return clock.arbiter.activateAt(participantFor(p), current)
}
func (clock *BrokerClock) IsQuiescent(p *BrokerParticipant) bool {
	return clock.arbiter.isQuiescent(participantFor(p))
}

func (clock *BrokerClock) ControlForwarded(p, node *BrokerParticipant, id uint64, arrivals uint32) error {
	return clock.arbiter.progress.apply(coordinatorControlForwarded{participant: participantFor(p), node: participantFor(node), request: id, arrivals: arrivals})
}

func (clock *BrokerClock) WaitAccepted(p *BrokerParticipant, id uint64, arrivals uint32) error {
	return clock.arbiter.progress.apply(coordinatorWaitAccepted{participant: participantFor(p), request: id, arrivals: arrivals})
}

func (clock *BrokerClock) WaitSuspended(p *BrokerParticipant, id uint64) error {
	return clock.arbiter.progress.apply(coordinatorWaitSuspended{participant: participantFor(p), request: id})
}

func (clock *BrokerClock) WaitResumed(p *BrokerParticipant, id uint64) error {
	return clock.arbiter.progress.apply(coordinatorWaitResumed{participant: participantFor(p), request: id})
}

func (clock *BrokerClock) CompletionReserved(p *BrokerParticipant, id uint64) error {
	return clock.arbiter.progress.apply(coordinatorCompletionReserved{participant: participantFor(p), request: id})
}

func (clock *BrokerClock) RequestAccepted(p *BrokerParticipant, arrivals uint32) error {
	return clock.arbiter.progress.apply(participantRequestAccepted{participant: participantFor(p), arrivals: arrivals})
}

func (clock *BrokerClock) ResponseDelivered(p *BrokerParticipant) error {
	return clock.arbiter.progress.apply(participantResponseDelivered{participant: participantFor(p)})
}

func (clock *BrokerClock) Forwarded(source, destination *BrokerParticipant, arrivals uint32, current int64) error {
	return clock.arbiter.progress.apply(modelRequestForwarded{source: participantFor(source), destination: participantFor(destination), arrivals: arrivals, current: current})
}

func (clock *BrokerClock) Dispatched(p *BrokerParticipant) error {
	return clock.arbiter.progress.apply(modelRequestDispatched{coordinator: participantFor(p)})
}

func (clock *BrokerClock) DispatchUnavailable(p *BrokerParticipant) error {
	return clock.arbiter.progress.apply(modelDispatchUnavailable{coordinator: participantFor(p)})
}

func (clock *BrokerClock) ResponseArrived(coordinator, node *BrokerParticipant, arrivals uint32) error {
	return clock.arbiter.progress.apply(modelResponseArrived{coordinator: participantFor(coordinator), node: participantFor(node), arrivals: arrivals})
}

func (clock *BrokerClock) AbandonedResponseDiscarded(p *BrokerParticipant, arrivals uint32) error {
	return clock.arbiter.progress.apply(modelAbandonedResponseDiscarded{coordinator: participantFor(p), arrivals: arrivals})
}

func (clock *BrokerClock) Runnable(p *BrokerParticipant) error {
	return clock.arbiter.progress.apply(participantRunnable{participant: participantFor(p)})
}

func participantFor(participant *BrokerParticipant) *simulationTimeParticipant {
	if participant == nil {
		return nil
	}
	return participant.participant
}
