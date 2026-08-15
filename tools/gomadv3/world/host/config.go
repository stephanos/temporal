// Package host prepares the Runner side of a modeled World session.
package host

import "go.temporal.io/server/tools/gomadv3/world/internal/transport"

type SessionSpec struct {
	TransitionLimit uint64
	Seed            uint64
	ExpectedInitial []byte
	ReplayPlan      []byte
}

func EncodeSessionSpec(spec SessionSpec) ([]byte, error) {
	return transport.Encode(transport.Config{
		TransitionLimit: spec.TransitionLimit,
		Seed:            spec.Seed,
		ExpectedInitial: spec.ExpectedInitial,
		ReplayPlan:      spec.ReplayPlan,
	})
}
