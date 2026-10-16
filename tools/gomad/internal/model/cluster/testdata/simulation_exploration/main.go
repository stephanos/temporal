// simulation_exploration is the Simulation target for Combined Exploration
// campaigns: one two-way Scenario choice whose selected route starts the node,
// lets it bind its modeled address, and prints the route as the program's
// whole output. It lives in the root module because the harness bridges are
// only admitted for the main module's own tools/gomad/internal/model/cluster, and it imports
// nothing outside the standard library so capability review of its closure
// stays supported.
package main

import (
	"context"
	"fmt"
	"net"
	"os"

	gomadsim "go.temporal.io/server/tools/gomad/internal/model/cluster"
)

const boot gomadsim.BootID = "simulation-exploration-fixture"

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
	if err := gomadsim.RegisterBoot(boot, func(_ context.Context, node gomadsim.NodeContext) error {
		listener, err := net.Listen("tcp4", net.JoinHostPort(node.Address, "7233"))
		if err != nil {
			return err
		}
		return listener.Close()
	}); err != nil {
		return err
	}
	route := func(name string) (gomadsim.ScenarioStep, error) {
		return gomadsim.NewScenarioStep(name, func(ctx context.Context, cluster gomadsim.Cluster) error {
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
	result, err := gomadsim.Run(ctx, gomadsim.Spec{
		Schema: gomadsim.SpecSchema, Backend: gomadsim.BackendInProcess, Fidelity: gomadsim.FidelitySimulationModel,
		Seed: seed, Limits: gomadsim.DefaultLimits(),
		Nodes: []gomadsim.NodeSpec{{ID: "server", Boot: boot, Address: "10.0.0.1"}},
	}, gomadsim.Choose("route", alpha, beta))
	if err != nil {
		return err
	}
	if result.Outcome != gomadsim.OutcomeCompleted {
		return fmt.Errorf("simulation outcome = %s", result.Outcome)
	}
	return nil
}
