package simulation

import (
	"context"
	"errors"

	"go.temporal.io/server/tools/gomad3/internal/backends/target"
	gomad3sim "go.temporal.io/server/tools/gomad3/internal/model/cluster"
)

// ReplayRetained executes a sealed lifecycle script through the existing cluster
// owner. Scheduled network faults and output crash boundaries are retained inputs;
// it never calls an original Go scenario or invokes a source compiler.
func validateRetainedLifecycle(spec gomad3sim.Spec, p *Provider) error {
	plan := spec.Replay
	if len(plan.Faults) != 0 || len(plan.Scenarios) != 0 || len(plan.History) != 0 || len(plan.Observations) != 0 {
		return errors.New("retained WASM lifecycle script contains unsupported host scenario callbacks")
	}
	expectedOracles := 0
	if p.Oracles != nil {
		expectedOracles = len(p.Oracles.Histories)*3 + len(p.Oracles.Convergence) + len(p.Oracles.SQLite)*3
	}
	softAssertions, err := validateRetainedSoftAssertions(spec, p)
	if err != nil {
		return err
	}
	expectedOracles += softAssertions
	if len(plan.Oracles) != expectedOracles {
		return errors.New("retained WASM oracle evidence lacks its sealed observation plan")
	}
	for _, transition := range plan.Transitions {
		if transition.Action != gomad3sim.LifecycleCrash && transition.Action != gomad3sim.LifecycleStop {
			continue
		}
		bound := false
		boundaries := p.OutputCrashes
		if transition.Action == gomad3sim.LifecycleStop {
			boundaries = p.OutputStops
		}
		for _, boundary := range boundaries {
			bound = bound || boundary.Handle == transition.Handle
		}
		if transition.Action == gomad3sim.LifecycleCrash {
			for _, group := range p.OutputCrashGroups {
				for _, handle := range group.Handles {
					bound = bound || handle == transition.Handle
				}
			}
			var boot gomad3sim.BootID
			for _, node := range spec.Nodes {
				if node.ID == transition.Handle.Node {
					boot = node.Boot
				}
			}
			for _, binding := range p.IncarnationBoots {
				if binding.Handle == transition.Handle {
					boot = binding.Boot
				}
			}
			bound = bound || storageCrashBound(p.Boots[boot])
		}
		if !bound {
			return errors.New("retained WASM termination lacks a deterministic output or storage boundary")
		}
	}
	return nil
}

func ReplayRetained(ctx context.Context, payload target.BackendPayload, module RetainedModule, directory string, installed InstalledHelper) (gomad3sim.Result, error) {
	result, _, err := ReplayRetainedResults(ctx, payload, module, directory, installed)
	return result, err
}

// ReplayRetainedResults also returns real helper observations for existing cost collectors.
func ReplayRetainedResults(ctx context.Context, payload target.BackendPayload, module RetainedModule, directory string, installed InstalledHelper) (gomad3sim.Result, []IncarnationResult, error) {
	spec, p, err := ReopenProvider(payload, module, directory, installed)
	if err != nil {
		return gomad3sim.Result{}, nil, err
	}
	result, err := replayRetainedProvider(ctx, spec, p)
	return result, p.CompletedResults(), err
}

func replayRetainedProvider(ctx context.Context, spec gomad3sim.Spec, p *Provider) (gomad3sim.Result, error) {
	plan := spec.Replay
	result, err := gomad3sim.RunWASM(ctx, spec, func(ctx context.Context, c gomad3sim.Cluster) error {
		for _, transition := range plan.Transitions {
			var handle gomad3sim.NodeHandle
			var err error
			switch transition.Action {
			case gomad3sim.LifecycleStart:
				handle, err = c.Start(ctx, transition.Handle.Node)
			case gomad3sim.LifecycleRestart:
				handle, err = c.Restart(ctx, transition.Handle.Node)
			case gomad3sim.LifecycleWait:
				_, err = c.Wait(ctx, transition.Handle)
				handle = transition.Handle
			case gomad3sim.LifecycleStop:
				err = c.Stop(ctx, transition.Handle)
				handle = transition.Handle
			case gomad3sim.LifecycleCrash:
				err = c.Crash(ctx, transition.Handle)
				handle = transition.Handle
			default:
				return errors.New("unknown retained WASM lifecycle action")
			}
			if err != nil {
				return err
			}
			if handle != transition.Handle {
				return gomad3sim.ErrStaleIncarnation
			}
		}
		return p.RecordOracles(ctx, c)
	}, p)
	return result, err
}
