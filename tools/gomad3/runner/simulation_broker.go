package runner

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"fmt"

	"go.temporal.io/server/tools/gomad3/simulation/clock"
)

//go:embed simulation_broker.go
var simulationBrokerSource []byte

func SimulationClockImplementationSHA256() string {
	data := fmt.Sprintf("gomad3-simulation-clock-facade/v1\n%s\n%x\n", clock.BrokerImplementationSHA256(), sha256.Sum256(simulationBrokerSource))
	return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(data)))
}

type SimulationClock struct{ owner *clock.BrokerClock }
type SimulationParticipant struct{ owner *clock.BrokerParticipant }
type SimulationTimeRequest struct {
	Generation        uint64
	Current, Deadline int64
	Arrivals          uint32
}
type SimulationTimeResponse struct {
	Generation uint64
	Kind       uint8
	Time       int64
}

func NewSimulationClock() *SimulationClock { return &SimulationClock{clock.NewBrokerClock()} }

func (c *SimulationClock) Register(name string) (*SimulationParticipant, error) {
	p, err := c.owner.Register(name)
	if err != nil {
		return nil, err
	}
	return &SimulationParticipant{p}, nil
}
func (c *SimulationClock) Current() int64 { return c.owner.Current() }
func (c *SimulationClock) Quiesce(ctx context.Context, p *SimulationParticipant, r SimulationTimeRequest) (SimulationTimeResponse, error) {
	v, err := c.owner.Quiesce(ctx, p.owner, clock.BrokerTimeRequest{Generation: r.Generation, Current: r.Current, Deadline: r.Deadline, Arrivals: r.Arrivals})
	return SimulationTimeResponse{v.Generation, v.Kind, v.Time}, err
}
func (c *SimulationClock) Accepted(p *SimulationParticipant, id uint64, arrivals uint32) error {
	return c.owner.Accepted(p.owner, id, arrivals)
}
func (c *SimulationClock) Delivered(p *SimulationParticipant, id uint64) error {
	return c.owner.Delivered(p.owner, id)
}
func (c *SimulationClock) Consume(p *SimulationParticipant, arrivals uint32) error {
	return c.owner.Consume(p.owner, arrivals)
}
func (c *SimulationClock) Remove(p *SimulationParticipant) error { return c.owner.Remove(p.owner) }
