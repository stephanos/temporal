//go:build gomadv3_toolchain

package gomadv3sim

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFaultPlanPartitionRestartEvidenceAndExactReplay(t *testing.T) {
	boot := uniqueBootID("fault-plan-node")
	require.NoError(t, RegisterBoot(boot, func(context.Context, NodeContext) error { return nil }))
	plan, err := NewFaultPlan([]FaultAction{
		{ID: "isolate-client", Kind: FaultPartition, Left: []NodeID{"client"}, Right: []NodeID{"server-a", "server-b"}},
		{ID: "heal-client", Kind: FaultHeal, Left: []NodeID{"client"}, Right: []NodeID{"server-a", "server-b"}},
		{ID: "crash-server", Kind: FaultHarshCrash, Candidates: []NodeID{"server-a", "server-b"}, Persistence: FaultPersistencePartial},
		{ID: "restart-server", Kind: FaultRestart, TargetFrom: "crash-server"},
	})
	require.NoError(t, err)
	spec := threeNodeFaultSpec(boot, plan)
	scenario := Scenario(func(ctx context.Context, cluster Cluster) error {
		for _, node := range []NodeID{"client", "server-a", "server-b"} {
			if _, err := cluster.Start(ctx, node); err != nil {
				return err
			}
		}
		for _, action := range plan.Actions {
			if _, err := cluster.ApplyFault(ctx, action); err != nil {
				return err
			}
		}
		if err := cluster.Observe(ctx, Observation{ID: "replication-state", Kind: "state", Value: []byte("healed")}); err != nil {
			return err
		}
		if err := cluster.RecordOperation(ctx, HistoryOperation{ID: "replicate-1", Actor: "client", Kind: "replicate", Invocation: 1, Completion: 2, Input: []byte("task"), Output: []byte("ok")}); err != nil {
			return err
		}
		oracle, err := StateInvariant("replication.healed", true, []OracleEvidence{{Label: "state", Value: []byte("healed")}}, spec.Limits.ScenarioEvidenceBytes)
		if err != nil {
			return err
		}
		return cluster.RecordOracle(ctx, oracle)
	})

	recorded, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, recorded.Outcome)
	require.Equal(t, plan, recorded.Record.FaultPlan)
	require.Len(t, recorded.Faults, len(plan.Actions))
	require.Len(t, recorded.Observations, 1)
	require.Len(t, recorded.History, 1)
	require.Len(t, recorded.Oracles, 1)
	require.NotEmpty(t, recorded.Record.Static.TargetSHA256)
	require.NotEmpty(t, recorded.Record.Static.PlatformBundleSHA256)
	require.NotEmpty(t, recorded.Record.Models.FaultSHA256)
	require.Equal(t, NetworkPartition, recorded.Network.Transitions[0].Kind)
	require.EqualValues(t, 2, recorded.Network.Transitions[0].Count)
	require.Equal(t, NetworkPartition, recorded.Network.Transitions[1].Kind)

	replay, err := ReplayPlanFor(recorded.Record)
	require.NoError(t, err)
	replaySpec := spec
	replaySpec.Replay = &replay
	replayed, err := Run(context.Background(), replaySpec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, replayed.Outcome)
	require.Equal(t, recorded.Record.Identity, replayed.Record.Identity)
}

func TestFaultReplayRejectsUnusedExtraReorderedAndChangedActions(t *testing.T) {
	boot := uniqueBootID("fault-divergence-node")
	require.NoError(t, RegisterBoot(boot, func(context.Context, NodeContext) error { return nil }))
	plan, err := NewFaultPlan([]FaultAction{
		{ID: "disconnect", Kind: FaultDisconnect, From: "client", To: "server-a"},
		{ID: "reconnect", Kind: FaultReconnect, From: "client", To: "server-a"},
	})
	require.NoError(t, err)
	spec := threeNodeFaultSpec(boot, plan)
	recordScenario := func(ctx context.Context, cluster Cluster) error {
		for _, action := range plan.Actions {
			if _, err := cluster.ApplyFault(ctx, action); err != nil {
				return err
			}
		}
		return nil
	}
	recorded, err := Run(context.Background(), spec, recordScenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, recorded.Outcome)
	replay, err := ReplayPlanFor(recorded.Record)
	require.NoError(t, err)

	tests := map[string]struct {
		scenario           Scenario
		wantAppliedNetwork uint64
	}{
		"unused": {scenario: func(context.Context, Cluster) error { return nil }},
		"reordered": {scenario: func(ctx context.Context, cluster Cluster) error {
			_, applyErr := cluster.ApplyFault(ctx, plan.Actions[1])
			return applyErr
		}},
		"changed": {scenario: func(ctx context.Context, cluster Cluster) error {
			changed := cloneFaultAction(plan.Actions[0])
			changed.To = "server-b"
			_, applyErr := cluster.ApplyFault(ctx, changed)
			return applyErr
		}},
		"extra": {scenario: func(ctx context.Context, cluster Cluster) error {
			for _, action := range plan.Actions {
				if _, applyErr := cluster.ApplyFault(ctx, action); applyErr != nil {
					return applyErr
				}
			}
			_, applyErr := cluster.ApplyFault(ctx, plan.Actions[0])
			return applyErr
		}, wantAppliedNetwork: 2},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			replaySpec := spec
			cloned := cloneReplayPlan(replay)
			replaySpec.Replay = &cloned
			result, runErr := Run(context.Background(), replaySpec, test.scenario)
			require.NoError(t, runErr)
			require.Equal(t, OutcomeReplayDiverged, result.Outcome)
			require.Equal(t, ReplayDimensionFault, result.Divergence.Dimension)
			require.Len(t, result.Network.Transitions, int(test.wantAppliedNetwork))
		})
	}
}

func TestFaultControllerMatchesStableFieldsAndOwnsOccurrence(t *testing.T) {
	boot := uniqueBootID("fault-match-node")
	require.NoError(t, RegisterBoot(boot, func(context.Context, NodeContext) error { return nil }))
	match := FaultMatch{Model: "network", Resource: "rpc", Operation: "deliver", Occurrence: 2, Phase: "replication", Equivalence: "history-peer"}
	plan, err := NewFaultPlan([]FaultAction{{
		ID: "disconnect-second-delivery", Kind: FaultDisconnect, Match: match, From: "client", To: "server-a",
	}})
	require.NoError(t, err)
	spec := threeNodeFaultSpec(boot, plan)
	event := match
	event.Occurrence = 0
	scenario := func(ctx context.Context, cluster Cluster) error {
		_, applied, err := cluster.TriggerFault(ctx, event)
		if err != nil {
			return err
		}
		if applied {
			return errors.New("fault matched the first occurrence")
		}
		realization, applied, err := cluster.TriggerFault(ctx, event)
		if err != nil {
			return err
		}
		if !applied || realization.Matched.Occurrence != 2 {
			return errors.New("fault did not record its matched occurrence")
		}
		return nil
	}
	recorded, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, recorded.Outcome)
	require.Len(t, recorded.Faults, 1)
	wantMatched := event
	wantMatched.Occurrence = 2
	require.Equal(t, wantMatched, recorded.Faults[0].Matched)
	require.Equal(t, NetworkDisconnect, recorded.Network.Transitions[0].Kind)

	replay, err := ReplayPlanFor(recorded.Record)
	require.NoError(t, err)
	replaySpec := spec
	replaySpec.Replay = &replay
	replayed, err := Run(context.Background(), replaySpec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, replayed.Outcome)
	require.Equal(t, recorded.Record.Identity, replayed.Record.Identity)

	changedScenario := func(ctx context.Context, cluster Cluster) error {
		_, _, triggerErr := cluster.TriggerFault(ctx, event)
		if triggerErr != nil {
			return triggerErr
		}
		changed := event
		changed.Resource = "other-rpc"
		_, _, triggerErr = cluster.TriggerFault(ctx, changed)
		return triggerErr
	}
	diverged, err := Run(context.Background(), replaySpec, changedScenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeReplayDiverged, diverged.Outcome)
	require.Equal(t, ReplayDimensionFault, diverged.Divergence.Dimension)
	require.Empty(t, diverged.Network.Transitions)
}

func TestMatchedFaultCannotBypassControllerMatching(t *testing.T) {
	boot := uniqueBootID("fault-match-bypass-node")
	require.NoError(t, RegisterBoot(boot, func(context.Context, NodeContext) error { return nil }))
	plan, err := NewFaultPlan([]FaultAction{{
		ID: "disconnect", Kind: FaultDisconnect, Match: FaultMatch{Model: "network"}, From: "client", To: "server-a",
	}})
	require.NoError(t, err)
	result, err := Run(context.Background(), threeNodeFaultSpec(boot, plan), func(ctx context.Context, cluster Cluster) error {
		_, applyErr := cluster.ApplyFault(ctx, plan.Actions[0])
		return applyErr
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeScenarioFailed, result.Outcome)
	require.Empty(t, result.Network.Transitions)
}

func TestGroupedPartitionCapacityRejectsBeforeTopologyMutation(t *testing.T) {
	boot := uniqueBootID("fault-group-capacity-node")
	require.NoError(t, RegisterBoot(boot, func(context.Context, NodeContext) error { return nil }))
	plan, err := NewFaultPlan([]FaultAction{{
		ID: "partition", Kind: FaultPartition, Left: []NodeID{"client"}, Right: []NodeID{"server-a", "server-b"},
	}})
	require.NoError(t, err)
	spec := threeNodeFaultSpec(boot, plan)
	spec.Limits.NetworkTransitions = 1
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		_, applyErr := cluster.ApplyFault(ctx, plan.Actions[0])
		return applyErr
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeScenarioFailed, result.Outcome)
	require.Empty(t, result.Network.Transitions)
	for _, link := range result.Network.Snapshot.Links {
		require.True(t, link.Enabled)
	}
}

func TestPartialDiskFaultPlanReproducesSelectedCrashState(t *testing.T) {
	boot := uniqueBootID("partial-disk-node")
	written := make(chan struct{}, 2)
	observed := make(chan []byte, 2)
	require.NoError(t, RegisterBoot(boot, func(_ context.Context, node NodeContext) error {
		if node.Incarnation == 1 {
			if err := os.WriteFile("/data/value", []byte("volatile"), 0o600); err != nil {
				return err
			}
			written <- struct{}{}
			return nil
		}
		value, err := os.ReadFile("/data/value")
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		observed <- append([]byte(nil), value...)
		return nil
	}))
	plan, err := NewFaultPlan([]FaultAction{
		{ID: "partial-crash", Kind: FaultHarshCrash, Node: "server", Persistence: FaultPersistencePartial},
		{ID: "restart", Kind: FaultRestart, TargetFrom: "partial-crash"},
	})
	require.NoError(t, err)
	limits := DefaultLimits()
	spec := Spec{
		Schema: SpecSchema, Backend: BackendInProcess, Fidelity: FidelitySimulationModel, Seed: 31, Limits: limits,
		Nodes:   []NodeSpec{{ID: "server", Boot: boot, Address: "10.0.0.1", Volumes: []VolumeMount{{Volume: "data", Path: "/data"}}}},
		Volumes: []VolumeSpec{{ID: "data", CapacityBytes: 1 << 20}}, Faults: &plan,
	}
	scenario := func(ctx context.Context, cluster Cluster) error {
		if _, err := cluster.Start(ctx, "server"); err != nil {
			return err
		}
		<-written
		if _, err := cluster.ApplyFault(ctx, plan.Actions[0]); err != nil {
			return err
		}
		restarted, err := cluster.ApplyFault(ctx, plan.Actions[1])
		if err != nil {
			return err
		}
		if _, err := cluster.Wait(ctx, restarted.Target); err != nil {
			return err
		}
		return cluster.Observe(ctx, Observation{ID: "restarted-value", Kind: "volume-state", Value: <-observed})
	}
	recorded, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, recorded.Outcome)
	require.Contains(t, volumeTransitionKinds(recorded.Volumes.Transitions), VolumeOperationCrash)
	require.Len(t, recorded.Faults, 2)

	replay, err := ReplayPlanFor(recorded.Record)
	require.NoError(t, err)
	replaySpec := spec
	replaySpec.Replay = &replay
	replayed, err := Run(context.Background(), replaySpec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, replayed.Outcome)
	require.Equal(t, recorded.Record.Identity, replayed.Record.Identity)
}

func TestPersistedOnlyCrashDropsEveryUnsyncedOperation(t *testing.T) {
	boot := uniqueBootID("persisted-only-crash-node")
	ready := make(chan struct{}, 4)
	recoveredMissing := make(chan bool, 2)
	require.NoError(t, RegisterBoot(boot, func(ctx context.Context, node NodeContext) error {
		if node.Incarnation == 1 {
			if err := os.WriteFile("/data/volatile", []byte("unsynced"), 0o600); err != nil {
				return err
			}
			ready <- struct{}{}
			select {}
		}
		_, err := os.Stat("/data/volatile")
		recoveredMissing <- errors.Is(err, os.ErrNotExist)
		ready <- struct{}{}
		<-ctx.Done()
		return ctx.Err()
	}))
	plan, err := NewFaultPlan([]FaultAction{
		{ID: "persisted-crash", Kind: FaultHarshCrash, Node: "server", Persistence: FaultPersistencePersisted},
		{ID: "restart", Kind: FaultRestart, TargetFrom: "persisted-crash"},
	})
	require.NoError(t, err)
	spec := oneNodeVolumeSpec(boot)
	spec.Faults = &plan
	scenario := func(ctx context.Context, cluster Cluster) error {
		if _, err := cluster.Start(ctx, "server"); err != nil {
			return err
		}
		<-ready
		if _, err := cluster.ApplyFault(ctx, plan.Actions[0]); err != nil {
			return err
		}
		restarted, err := cluster.ApplyFault(ctx, plan.Actions[1])
		if err != nil {
			return err
		}
		<-ready
		if !<-recoveredMissing {
			return errors.New("persisted-only crash retained an unsynced file")
		}
		return cluster.Stop(ctx, restarted.Target)
	}
	recorded, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, recorded.Outcome)
	for _, transition := range recorded.Volumes.Transitions {
		if transition.Kind == VolumeOperationCrash {
			require.Empty(t, transition.SelectedOperations)
		}
	}
	replay, err := ReplayPlanFor(recorded.Record)
	require.NoError(t, err)
	replaySpec := spec
	replaySpec.Replay = &replay
	replayed, err := Run(context.Background(), replaySpec, scenario)
	require.NoError(t, err)
	require.Equal(t, recorded.Record.Identity, replayed.Record.Identity)
}

func TestDirectionalFaultsPreserveReverseConnectivity(t *testing.T) {
	type dialCommand struct {
		address string
		timeout time.Duration
		result  chan error
	}
	registerNode := func(name string, commands <-chan dialCommand, ready chan<- struct{}) BootID {
		boot := uniqueBootID(name)
		require.NoError(t, RegisterBoot(boot, func(ctx context.Context, node NodeContext) error {
			listener, err := net.Listen("tcp4", net.JoinHostPort(node.Address, "7233"))
			if err != nil {
				return err
			}
			acceptDone := make(chan struct{})
			go func() {
				defer close(acceptDone)
				for {
					connection, acceptErr := listener.Accept()
					if acceptErr != nil {
						return
					}
					_ = connection.Close()
				}
			}()
			ready <- struct{}{}
			for {
				select {
				case command := <-commands:
					dialCtx := ctx
					cancel := func() {}
					if command.timeout != 0 {
						dialCtx, cancel = context.WithTimeout(ctx, command.timeout)
					}
					connection, dialErr := (&net.Dialer{}).DialContext(dialCtx, "tcp4", command.address)
					cancel()
					if connection != nil {
						dialErr = errors.Join(dialErr, connection.Close())
					}
					command.result <- dialErr
				case <-ctx.Done():
					closeErr := listener.Close()
					<-acceptDone
					return errors.Join(ctx.Err(), closeErr)
				}
			}
		}))
		return boot
	}
	ready := make(chan struct{}, 4)
	clientCommands := make(chan dialCommand)
	serverCommands := make(chan dialCommand)
	clientBoot := registerNode("directional-client", clientCommands, ready)
	serverBoot := registerNode("directional-server", serverCommands, ready)
	plan, err := NewFaultPlan([]FaultAction{
		{ID: "disconnect", Kind: FaultDisconnect, From: "client", To: "server"},
		{ID: "delay-reverse", Kind: FaultDelay, From: "server", To: "client", DelayNanos: uint64(5 * time.Millisecond)},
		{ID: "reconnect", Kind: FaultReconnect, From: "client", To: "server"},
	})
	require.NoError(t, err)
	spec := twoNodeNetworkSpec(serverBoot, clientBoot)
	spec.Faults = &plan
	result, err := Run(context.Background(), spec, func(ctx context.Context, cluster Cluster) error {
		client, err := cluster.Start(ctx, "client")
		if err != nil {
			return err
		}
		server, err := cluster.Start(ctx, "server")
		if err != nil {
			return err
		}
		<-ready
		<-ready
		if _, err := cluster.ApplyFault(ctx, plan.Actions[0]); err != nil {
			return err
		}
		clientResult := make(chan error, 1)
		clientCommands <- dialCommand{address: "10.0.0.1:7233", timeout: time.Millisecond, result: clientResult}
		if err := <-clientResult; !errors.Is(err, context.DeadlineExceeded) {
			return errors.New("directional disconnect did not block the configured direction")
		}
		if _, err := cluster.ApplyFault(ctx, plan.Actions[1]); err != nil {
			return err
		}
		serverResult := make(chan error, 1)
		serverCommands <- dialCommand{address: "10.0.0.2:7233", result: serverResult}
		if err := <-serverResult; err != nil {
			return err
		}
		if _, err := cluster.ApplyFault(ctx, plan.Actions[2]); err != nil {
			return err
		}
		clientCommands <- dialCommand{address: "10.0.0.1:7233", result: clientResult}
		if err := <-clientResult; err != nil {
			return err
		}
		if err := cluster.Stop(ctx, client); err != nil {
			return err
		}
		return cluster.Stop(ctx, server)
	})
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, result.Outcome, result.Reason)
	require.Contains(t, networkTransitionKinds(result.Network.Transitions), NetworkDisconnect)
	require.Contains(t, networkTransitionKinds(result.Network.Transitions), NetworkDirectionalDelay)
	require.Contains(t, networkTransitionKinds(result.Network.Transitions), NetworkReconnect)
	for _, link := range result.Network.Snapshot.Links {
		if link.From == "server" && link.To == "client" {
			require.True(t, link.Enabled)
			require.Equal(t, uint64(5*time.Millisecond), link.DelayNanos)
		}
	}
}

func TestScenarioCompositionSameSeedEqualityAndDifferentSeedDiversity(t *testing.T) {
	boot := uniqueBootID("scenario-composition-node")
	require.NoError(t, RegisterBoot(boot, func(context.Context, NodeContext) error { return nil }))
	alpha, err := NewScenarioStep("alpha", func(context.Context, Cluster) error { return nil })
	require.NoError(t, err)
	beta, err := NewScenarioStep("beta", func(context.Context, Cluster) error { return nil })
	require.NoError(t, err)
	repeated, err := NewScenarioStep("poll", func(context.Context, Cluster) error { return nil })
	require.NoError(t, err)
	left, err := NewScenarioStep("left-actor", func(context.Context, Cluster) error { return nil })
	require.NoError(t, err)
	right, err := NewScenarioStep("right-actor", func(context.Context, Cluster) error { return nil })
	require.NoError(t, err)
	choice, err := NewScenarioStep("choose-worker", Choose("worker", alpha, beta))
	require.NoError(t, err)
	parallel, err := NewScenarioStep("parallel-clients", BoundedParallel("client-actors", 2, left, right))
	require.NoError(t, err)
	scenario := Sequence(choice, Repeat(2, repeated), parallel)

	firstSeed := uint64(0)
	secondSeed := uint64(1)
	for selectScenarioAlternative(firstSeed, 1, "worker", 2) == selectScenarioAlternative(secondSeed, 1, "worker", 2) {
		secondSeed++
	}
	spec := oneNodeScenarioSpec(boot, firstSeed)
	first, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	second, err := Run(context.Background(), spec, scenario)
	require.NoError(t, err)
	require.Equal(t, first.Record.Identity, second.Record.Identity)
	require.Equal(t, first.Scenarios, second.Scenarios)

	diverseSpec := oneNodeScenarioSpec(boot, secondSeed)
	diverse, err := Run(context.Background(), diverseSpec, scenario)
	require.NoError(t, err)
	require.NotEqual(t, selectedScenarioAlternative(first.Scenarios, "worker"), selectedScenarioAlternative(diverse.Scenarios, "worker"))
	require.NotEqual(t, first.Record.Identity, diverse.Record.Identity)

	replay, err := ReplayPlanFor(first.Record)
	require.NoError(t, err)
	replaySpec := spec
	replaySpec.Replay = &replay
	replayed, err := Run(context.Background(), replaySpec, scenario)
	require.NoError(t, err)
	require.Equal(t, first.Record.Identity, replayed.Record.Identity)
}

func TestScenarioReplayRejectsUnusedExtraReorderedAndChangedDecisions(t *testing.T) {
	boot := uniqueBootID("scenario-divergence-node")
	require.NoError(t, RegisterBoot(boot, func(context.Context, NodeContext) error { return nil }))
	alpha, err := NewScenarioStep("alpha", func(context.Context, Cluster) error { return nil })
	require.NoError(t, err)
	beta, err := NewScenarioStep("beta", func(context.Context, Cluster) error { return nil })
	require.NoError(t, err)
	gamma, err := NewScenarioStep("gamma", func(context.Context, Cluster) error { return nil })
	require.NoError(t, err)
	spec := oneNodeScenarioSpec(boot, 47)
	recorded, err := Run(context.Background(), spec, Sequence(alpha, beta))
	require.NoError(t, err)
	require.Equal(t, OutcomeCompleted, recorded.Outcome)
	replay, err := ReplayPlanFor(recorded.Record)
	require.NoError(t, err)

	tests := map[string]struct {
		scenario      Scenario
		wantCommitted int
	}{
		"unused":    {scenario: func(context.Context, Cluster) error { return nil }},
		"reordered": {scenario: Sequence(beta, alpha)},
		"changed":   {scenario: Sequence(gamma, beta)},
		"extra":     {scenario: Sequence(alpha, beta, gamma), wantCommitted: 2},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			replaySpec := spec
			cloned := cloneReplayPlan(replay)
			replaySpec.Replay = &cloned
			result, runErr := Run(context.Background(), replaySpec, test.scenario)
			require.NoError(t, runErr)
			require.Equal(t, OutcomeReplayDiverged, result.Outcome)
			require.Equal(t, ReplayDimensionScenario, result.Divergence.Dimension)
			require.Len(t, result.Scenarios, test.wantCommitted)
		})
	}
}

func TestOracleFailureIdentityIsIndependentOfClusterSeedAndExactlyReplayable(t *testing.T) {
	boot := uniqueBootID("oracle-failure-node")
	require.NoError(t, RegisterBoot(boot, func(context.Context, NodeContext) error { return nil }))
	scenario := func(ctx context.Context, cluster Cluster) error {
		result, err := StateInvariant("matching.no-lost-task", false, []OracleEvidence{{Label: "task", Value: []byte("task-7")}}, MaximumScenarioEvidenceBytes)
		if err != nil {
			return err
		}
		return cluster.RecordOracle(ctx, result)
	}
	first, err := Run(context.Background(), oneNodeScenarioSpec(boot, 41), scenario)
	require.NoError(t, err)
	second, err := Run(context.Background(), oneNodeScenarioSpec(boot, 43), scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeOracleFailed, first.Outcome)
	require.Equal(t, OutcomeOracleFailed, second.Outcome)
	require.Equal(t, first.FailureIdentity, second.FailureIdentity)
	require.NotEqual(t, first.Record.Identity, second.Record.Identity)

	replay, err := ReplayPlanFor(first.Record)
	require.NoError(t, err)
	replaySpec := oneNodeScenarioSpec(boot, 41)
	replaySpec.Replay = &replay
	replayed, err := Run(context.Background(), replaySpec, scenario)
	require.NoError(t, err)
	require.Equal(t, OutcomeOracleFailed, replayed.Outcome)
	require.Equal(t, first.Record.Identity, replayed.Record.Identity)
}

func threeNodeFaultSpec(boot BootID, plan FaultPlan) Spec {
	return Spec{
		Schema: SpecSchema, Backend: BackendInProcess, Fidelity: FidelitySimulationModel, Seed: 29, Limits: DefaultLimits(),
		Nodes: []NodeSpec{
			{ID: "client", Boot: boot, Address: "10.0.0.1"},
			{ID: "server-a", Boot: boot, Address: "10.0.0.2"},
			{ID: "server-b", Boot: boot, Address: "10.0.0.3"},
		},
		Links: []LinkSpec{
			{From: "client", To: "server-a", Enabled: true},
			{From: "client", To: "server-b", Enabled: true},
			{From: "server-a", To: "client", Enabled: true},
			{From: "server-a", To: "server-b", Enabled: true},
			{From: "server-b", To: "client", Enabled: true},
			{From: "server-b", To: "server-a", Enabled: true},
		},
		Faults: &plan,
	}
}

func oneNodeScenarioSpec(boot BootID, seed uint64) Spec {
	return Spec{
		Schema: SpecSchema, Backend: BackendInProcess, Fidelity: FidelitySimulationModel, Seed: seed, Limits: DefaultLimits(),
		Nodes: []NodeSpec{{ID: "node", Boot: boot, Address: "10.0.0.1"}},
	}
}

func selectedScenarioAlternative(decisions []ScenarioDecision, id string) uint64 {
	for _, decision := range decisions {
		if decision.ID == id {
			return decision.Selected
		}
	}
	return ^uint64(0)
}

func volumeTransitionKinds(transitions []VolumeTransition) []VolumeTransitionKind {
	kinds := make([]VolumeTransitionKind, len(transitions))
	for index, transition := range transitions {
		kinds[index] = transition.Kind
	}
	return kinds
}

func networkTransitionKinds(transitions []NetworkTransition) []NetworkTransitionKind {
	kinds := make([]NetworkTransitionKind, len(transitions))
	for index, transition := range transitions {
		kinds[index] = transition.Kind
	}
	return kinds
}
