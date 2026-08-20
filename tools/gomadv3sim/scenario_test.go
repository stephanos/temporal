package gomadv3sim

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScenarioSequenceAndRepeatRecordStableActions(t *testing.T) {
	cluster := newScenarioTestCluster(17)
	var calls []string
	first, err := NewScenarioStep("first", func(context.Context, Cluster) error {
		calls = append(calls, "first")
		return nil
	})
	require.NoError(t, err)
	second, err := NewScenarioStep("second", func(context.Context, Cluster) error {
		calls = append(calls, "second")
		return nil
	})
	require.NoError(t, err)

	scenario := Sequence(first, Repeat(2, second))
	require.NoError(t, scenario(context.Background(), cluster))
	require.Equal(t, []string{"first", "second", "second"}, calls)
	require.Equal(t, []ScenarioDecision{
		scenarioDecisionForTest(0, "first", ScenarioDecisionAction, 1, nil, 0),
		scenarioDecisionForTest(1, "second", ScenarioDecisionAction, 1, nil, 0),
		scenarioDecisionForTest(2, "second", ScenarioDecisionAction, 2, nil, 0),
	}, cluster.decisions)
}

func TestScenarioChooseIsDomainSeparatedAndReplayable(t *testing.T) {
	options := make([]ScenarioStep, 3)
	for index, id := range []string{"alpha", "beta", "gamma"} {
		step, err := NewScenarioStep(id, func(context.Context, Cluster) error { return nil })
		require.NoError(t, err)
		options[index] = step
	}
	first := newScenarioTestCluster(19)
	second := newScenarioTestCluster(19)
	require.NoError(t, Choose("pick-worker", options...)(context.Background(), first))
	require.NoError(t, Choose("pick-worker", options...)(context.Background(), second))
	require.Equal(t, first.decisions, second.decisions)
	require.Len(t, first.decisions, 1)
	require.Equal(t, ScenarioDecisionChoose, first.decisions[0].Kind)
	require.Less(t, first.decisions[0].Selected, uint64(len(options)))

	seen := map[uint64]struct{}{}
	for seed := uint64(0); seed < 64; seed++ {
		seen[selectScenarioAlternative(seed, 0, "pick-worker", uint64(len(options)))] = struct{}{}
	}
	require.Greater(t, len(seen), 1)
}

func TestScenarioBoundedParallelHonorsLimitAndReturnsStableFirstError(t *testing.T) {
	cluster := newScenarioTestCluster(23)
	var active atomic.Int64
	var maximum atomic.Int64
	release := make(chan struct{})
	started := make(chan struct{}, 2)
	steps := make([]ScenarioStep, 4)
	for index := range steps {
		index := index
		step, err := NewScenarioStep("worker-"+string(rune('a'+index)), func(context.Context, Cluster) error {
			current := active.Add(1)
			for {
				observed := maximum.Load()
				if current <= observed || maximum.CompareAndSwap(observed, current) {
					break
				}
			}
			started <- struct{}{}
			<-release
			active.Add(-1)
			if index == 1 {
				return errors.New("worker-b failed")
			}
			return nil
		})
		require.NoError(t, err)
		steps[index] = step
	}

	done := make(chan error, 1)
	go func() { done <- BoundedParallel("workers", 2, steps...)(context.Background(), cluster) }()
	<-started
	<-started
	require.EqualValues(t, 2, active.Load())
	close(release)
	require.EqualError(t, <-done, "scenario step \"worker-b\": worker-b failed")
	require.EqualValues(t, 2, maximum.Load())
	require.Equal(t, ScenarioDecisionParallel, cluster.decisions[0].Kind)
	require.Equal(t, []string{"worker-a", "worker-b", "worker-c", "worker-d"}, cluster.decisions[0].Alternatives)
}

func TestScenarioConstructorsRejectInvalidInputsBeforeExecution(t *testing.T) {
	_, err := NewScenarioStep("", func(context.Context, Cluster) error { return nil })
	require.Error(t, err)
	_, err = NewScenarioStep("valid", nil)
	require.Error(t, err)
	cluster := newScenarioTestCluster(1)
	require.Error(t, Choose("choice")(context.Background(), cluster))
	require.Error(t, BoundedParallel("parallel", 0)(context.Background(), cluster))
	valid, err := NewScenarioStep("valid", func(context.Context, Cluster) error { return nil })
	require.NoError(t, err)
	require.Error(t, Sequence(Repeat(MaximumScenarioActions+1, valid))(context.Background(), cluster))
	require.Empty(t, cluster.decisions)
}

type scenarioTestCluster struct {
	seed        uint64
	decisions   []ScenarioDecision
	occurrences map[string]uint64
	mu          sync.Mutex
}

func newScenarioTestCluster(seed uint64) *scenarioTestCluster {
	return &scenarioTestCluster{seed: seed, occurrences: map[string]uint64{}}
}

func (cluster *scenarioTestCluster) recordScenarioDecision(_ context.Context, decision ScenarioDecision) error {
	cluster.mu.Lock()
	defer cluster.mu.Unlock()
	decision.Ordinal = uint64(len(cluster.decisions))
	cluster.occurrences[decision.ID]++
	decision.Occurrence = cluster.occurrences[decision.ID]
	identity, err := scenarioDecisionIdentity(decision)
	if err != nil {
		return err
	}
	decision.Identity = identity
	cluster.decisions = append(cluster.decisions, decision)
	return nil
}

func (cluster *scenarioTestCluster) chooseScenarioAlternative(_ context.Context, decision ScenarioDecision) (uint64, error) {
	cluster.mu.Lock()
	decision.Ordinal = uint64(len(cluster.decisions))
	cluster.mu.Unlock()
	decision.Selected = selectScenarioAlternative(cluster.seed, decision.Ordinal, decision.ID, uint64(len(decision.Alternatives)))
	return decision.Selected, cluster.recordScenarioDecision(context.Background(), decision)
}

func (cluster *scenarioTestCluster) Start(context.Context, NodeID) (NodeHandle, error) {
	return NodeHandle{}, nil
}
func (cluster *scenarioTestCluster) Wait(context.Context, NodeHandle) (NodeResult, error) {
	return NodeResult{}, nil
}
func (cluster *scenarioTestCluster) Stop(context.Context, NodeHandle) error  { return nil }
func (cluster *scenarioTestCluster) Crash(context.Context, NodeHandle) error { return nil }
func (cluster *scenarioTestCluster) Restart(context.Context, NodeID) (NodeHandle, error) {
	return NodeHandle{}, nil
}
func (cluster *scenarioTestCluster) Partition(context.Context, NodeID, NodeID) error { return nil }
func (cluster *scenarioTestCluster) Heal(context.Context, NodeID, NodeID) error      { return nil }
func (cluster *scenarioTestCluster) SetDelay(context.Context, NodeID, NodeID, uint64) error {
	return nil
}
func (cluster *scenarioTestCluster) ApplyFault(context.Context, FaultAction) (FaultRealization, error) {
	return FaultRealization{}, nil
}
func (cluster *scenarioTestCluster) TriggerFault(context.Context, FaultMatch) (FaultRealization, bool, error) {
	return FaultRealization{}, false, nil
}
func (cluster *scenarioTestCluster) Observe(context.Context, Observation) error { return nil }
func (cluster *scenarioTestCluster) RecordOperation(context.Context, HistoryOperation) error {
	return nil
}
func (cluster *scenarioTestCluster) RecordOracle(context.Context, OracleResult) error { return nil }
func (cluster *scenarioTestCluster) EnumerateCrashStates(context.Context, NodeHandle, VolumeID, VolumeCrashEnumerationLimits, *VolumeCrashFrontier) (VolumeCrashEnumeration, error) {
	return VolumeCrashEnumeration{}, nil
}

func scenarioDecisionForTest(ordinal uint64, id string, kind ScenarioDecisionKind, occurrence uint64, alternatives []string, selected uint64) ScenarioDecision {
	decision := ScenarioDecision{Ordinal: ordinal, ID: id, Kind: kind, Occurrence: occurrence, Alternatives: alternatives, Selected: selected}
	identity, err := scenarioDecisionIdentity(decision)
	if err != nil {
		panic(err)
	}
	decision.Identity = identity
	return decision
}
