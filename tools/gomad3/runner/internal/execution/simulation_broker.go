package execution

import (
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
)

//go:embed simulation_broker.go simulation_model.go simulation_progress.go simulation_time.go simulation_time_wire_generated.go
var brokerImplementationSources embed.FS

func BrokerImplementationSHA256() string {
	data := []byte("gomad3-simulation-clock-implementation/v1\n")
	for _, name := range []string{"simulation_broker.go", "simulation_model.go", "simulation_progress.go", "simulation_time.go", "simulation_time_wire_generated.go"} {
		source, err := brokerImplementationSources.ReadFile(name)
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
	r, err := clock.arbiter.quiesce(ctx, p.participant, simulationTimeRequest(request))
	return BrokerTimeResponse{r.Generation, uint8(r.Kind), r.Time}, err
}

func (clock *BrokerClock) Accepted(p *BrokerParticipant, id uint64, arrivals uint32) error {
	return clock.arbiter.progress.apply(coordinatorRequestAccepted{participant: p.participant, request: id, arrivals: arrivals})
}

func (clock *BrokerClock) Delivered(p *BrokerParticipant, id uint64) error {
	return clock.arbiter.progress.apply(coordinatorResponseDelivered{participant: p.participant, request: id})
}

func (clock *BrokerClock) Consume(p *BrokerParticipant, arrivals uint32) error {
	return clock.arbiter.progress.apply(arrivalCreditsConsumed{participant: p.participant, arrivals: arrivals})
}

func (clock *BrokerClock) Remove(p *BrokerParticipant) error {
	return clock.arbiter.progress.apply(participantRemoved{participant: p.participant})
}
