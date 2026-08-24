package gomad3sim

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

var testBootSequence atomic.Uint64

func uniqueBootID(prefix string) BootID {
	return BootID(fmt.Sprintf("sim0-%s-%d", prefix, testBootSequence.Add(1)))
}

func testNetworkRecord(t *testing.T, spec Spec, incarnations map[NodeID]uint64) NetworkRecord {
	t.Helper()
	record := emptyNetworkRecord()
	for _, node := range spec.Nodes {
		record.Snapshot.Nodes = append(record.Snapshot.Nodes, NetworkNodeSnapshot{
			Node: node.ID, Address: node.Address, LastIncarnation: incarnations[node.ID],
			NextListenerPort: 20000, NextClientPort: 40000,
		})
	}
	for _, link := range spec.Links {
		record.Snapshot.Links = append(record.Snapshot.Links, NetworkLinkSnapshot(link))
	}
	identity, err := networkSnapshotIdentity(record.Snapshot)
	require.NoError(t, err)
	record.Snapshot.Identity = identity
	return record
}

func testReplayPlan(t *testing.T, spec Spec) ReplayPlan {
	t.Helper()
	specSHA256, err := hashSpec(spec)
	require.NoError(t, err)
	models, err := currentClusterModelIdentities()
	require.NoError(t, err)
	static, err := clusterStaticIdentities(spec, models)
	require.NoError(t, err)
	faults, err := NewFaultPlan(nil)
	require.NoError(t, err)
	scenarioChoices, err := NewScenarioChoicePlan(nil)
	require.NoError(t, err)
	return ReplayPlan{
		Schema: ClusterReplaySchema, SpecSHA256: specSHA256, Static: static, Models: models,
		Outcome: OutcomeCompleted, FaultPlan: faults, ScenarioChoices: scenarioChoices, Network: emptyNetworkRecord(), Volumes: emptyVolumeRecord(),
	}
}
