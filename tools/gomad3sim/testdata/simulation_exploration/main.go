// simulation_exploration is the Simulation target for Combined Exploration
// campaigns: one two-way Scenario choice whose selected route starts the node,
// lets it bind its modeled address, and prints the route as the program's
// whole output. It lives in the root module because the harness bridges are
// only admitted for the main module's own tools/gomad3sim, and it imports
// nothing outside the standard library so capability review of its closure
// stays supported.
package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"go.temporal.io/server/tools/gomad3sim"
)

const boot gomad3sim.BootID = "simulation-exploration-fixture"

// The harness requires an exploration plan's base Seed to equal the Spec's,
// and the runtime keeps the Seed out of the target's environment, so campaigns
// over this fixture select exactly this Seed.
const seed = 89

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	if err := gomad3sim.RegisterBoot(boot, func(_ context.Context, node gomad3sim.NodeContext) error {
		listener, err := net.Listen("tcp4", net.JoinHostPort(node.Address, "7233"))
		if err != nil {
			return err
		}
		return listener.Close()
	}); err != nil {
		return err
	}
	route := func(name string) (gomad3sim.ScenarioStep, error) {
		return gomad3sim.NewScenarioStep(name, func(ctx context.Context, cluster gomad3sim.Cluster) error {
			handle, err := cluster.Start(ctx, "server")
			if err != nil {
				return err
			}
			if _, err := cluster.Wait(ctx, handle); err != nil {
				return err
			}
			_, err = fmt.Println("route", name)
			return err
		})
	}
	alpha, err := route("alpha")
	if err != nil {
		return err
	}
	beta, err := route("beta")
	if err != nil {
		return err
	}
	result, err := gomad3sim.Run(ctx, gomad3sim.Spec{
		Schema: gomad3sim.SpecSchema, Backend: gomad3sim.BackendInProcess, Fidelity: gomad3sim.FidelitySimulationModel,
		Seed: seed, Limits: gomad3sim.DefaultLimits(),
		Nodes: []gomad3sim.NodeSpec{{ID: "server", Boot: boot, Address: "10.0.0.1"}},
	}, gomad3sim.Choose("route", alpha, beta))
	if err != nil {
		return err
	}
	if result.Outcome != gomad3sim.OutcomeCompleted {
		return fmt.Errorf("simulation outcome = %s", result.Outcome)
	}
	return nil
}
