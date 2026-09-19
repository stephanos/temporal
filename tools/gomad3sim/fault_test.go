package gomad3sim

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFaultPlanCanonicalRoundTripAndDetachedInputs(t *testing.T) {
	actions := []FaultAction{
		{
			ID: "isolate-history", Kind: FaultPartition,
			Match: FaultMatch{NodeClass: "history", Model: "network", Resource: "rpc", Operation: "write", Occurrence: 2, Phase: "replication", Equivalence: "history-peer"},
			Left:  []NodeID{"history-a"}, Right: []NodeID{"history-b", "history-c"},
		},
		{ID: "crash-history", Kind: FaultHarshCrash, Candidates: []NodeID{"history-a", "history-b"}, Persistence: FaultPersistencePartial},
	}
	plan, err := NewFaultPlan(actions)
	require.NoError(t, err)
	actions[0].Left[0] = "changed"
	actions[1].Candidates[0] = "changed"
	require.Equal(t, []NodeID{"history-a"}, plan.Actions[0].Left)
	require.Equal(t, []NodeID{"history-a", "history-b"}, plan.Actions[1].Candidates)

	encoded, err := EncodeFaultPlan(plan)
	require.NoError(t, err)
	decoded, err := DecodeFaultPlan(encoded)
	require.NoError(t, err)
	require.Equal(t, plan, decoded)
	require.NotContains(t, string(encoded), "\n")

	var projection map[string]any
	require.NoError(t, json.Unmarshal(encoded, &projection))
	require.Equal(t, FaultPlanSchema, projection["schema"])
}

func TestFaultPlanIdentityChangesWithEverySemanticField(t *testing.T) {
	base, err := NewFaultPlan([]FaultAction{{
		ID: "disconnect", Kind: FaultDisconnect, From: "client", To: "server",
		Match: FaultMatch{Node: "client", Incarnation: 1, NodeClass: "client", Model: "network", Resource: "connection-7", Operation: "deliver", Occurrence: 3, Phase: "request", Equivalence: "request-delivery"},
	}})
	require.NoError(t, err)

	mutations := map[string]func(*FaultAction){
		"id":          func(action *FaultAction) { action.ID = "other" },
		"kind":        func(action *FaultAction) { action.Kind = FaultReconnect },
		"node":        func(action *FaultAction) { action.Match.Node = "server" },
		"incarnation": func(action *FaultAction) { action.Match.Incarnation++ },
		"class":       func(action *FaultAction) { action.Match.NodeClass = "worker" },
		"model":       func(action *FaultAction) { action.Match.Model = "volume" },
		"resource":    func(action *FaultAction) { action.Match.Resource = "connection-8" },
		"operation":   func(action *FaultAction) { action.Match.Operation = "dial" },
		"occurrence":  func(action *FaultAction) { action.Match.Occurrence++ },
		"phase":       func(action *FaultAction) { action.Match.Phase = "response" },
		"equivalence": func(action *FaultAction) { action.Match.Equivalence = "response-delivery" },
		"from":        func(action *FaultAction) { action.From = "worker" },
		"to":          func(action *FaultAction) { action.To = "worker" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			action := cloneFaultAction(base.Actions[0])
			mutate(&action)
			changed, changedErr := NewFaultPlan([]FaultAction{action})
			require.NoError(t, changedErr)
			require.NotEqual(t, base.Identity, changed.Identity)
		})
	}
}

func TestFaultPlanIdentityBindsPriorTargetReference(t *testing.T) {
	base, err := NewFaultPlan([]FaultAction{
		{ID: "crash-a", Kind: FaultHarshCrash, Node: "server-a", Persistence: FaultPersistencePersisted},
		{ID: "crash-b", Kind: FaultHarshCrash, Node: "server-b", Persistence: FaultPersistencePersisted},
		{ID: "restart", Kind: FaultRestart, TargetFrom: "crash-a"},
	})
	require.NoError(t, err)
	changedActions := cloneFaultActions(base.Actions)
	changedActions[2].TargetFrom = "crash-b"
	changed, err := NewFaultPlan(changedActions)
	require.NoError(t, err)
	require.NotEqual(t, base.Identity, changed.Identity)
}

func TestFaultMatchingTreatsEmptyFieldsAsWildcards(t *testing.T) {
	pattern := FaultMatch{NodeClass: "history", Model: "network", Operation: "deliver", Occurrence: 2}
	event := FaultMatch{Node: "history-a", Incarnation: 3, NodeClass: "history", Model: "network", Resource: "rpc-7", Operation: "deliver", Occurrence: 2, Phase: "replication"}
	require.True(t, faultMatches(pattern, event))
	event.Operation = "dial"
	require.False(t, faultMatches(pattern, event))
	event.Operation = "deliver"
	event.Occurrence = 1
	require.False(t, faultMatches(pattern, event))
	require.Error(t, validateFaultEvent(FaultMatch{Model: "network", Occurrence: 1}))
}

func TestFaultPlanRejectsInvalidShapesAndNonCanonicalInput(t *testing.T) {
	tests := map[string]FaultAction{
		"unknown kind":        {ID: "bad", Kind: "unknown", Node: "server"},
		"missing target":      {ID: "bad", Kind: FaultHarshCrash, Persistence: FaultPersistencePartial},
		"mixed target modes":  {ID: "bad", Kind: FaultHarshCrash, Node: "server", Candidates: []NodeID{"server"}, Persistence: FaultPersistencePartial},
		"unsorted candidates": {ID: "bad", Kind: FaultHarshCrash, Candidates: []NodeID{"server-b", "server-a"}, Persistence: FaultPersistencePartial},
		"direction self link": {ID: "bad", Kind: FaultDisconnect, From: "server", To: "server"},
		"partition overlap":   {ID: "bad", Kind: FaultPartition, Left: []NodeID{"server"}, Right: []NodeID{"server"}},
		"delay without value": {ID: "bad", Kind: FaultDelay, From: "client", To: "server"},
	}
	for name, action := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewFaultPlan([]FaultAction{action})
			require.Error(t, err)
		})
	}

	plan, err := NewFaultPlan([]FaultAction{{ID: "crash", Kind: FaultHarshCrash, Node: "server", Persistence: FaultPersistencePartial}})
	require.NoError(t, err)
	encoded, err := EncodeFaultPlan(plan)
	require.NoError(t, err)
	_, err = DecodeFaultPlan(append(encoded, '\n'))
	require.Error(t, err)
	unknown := append([]byte(`{"unknown":true,`), encoded[1:]...)
	_, err = DecodeFaultPlan(unknown)
	require.Error(t, err)
}

func TestFaultPlanCapacity(t *testing.T) {
	actions := make([]FaultAction, MaximumFaultActions+1)
	for index := range actions {
		actions[index] = FaultAction{ID: FaultID("fault-" + strconv.FormatUint(uint64(index), 10)), Kind: FaultHarshCrash, Node: "server", Persistence: FaultPersistencePartial}
	}
	_, err := NewFaultPlan(actions)
	var capacityErr *CapacityError
	require.ErrorAs(t, err, &capacityErr)
	require.Equal(t, "fault_actions", capacityErr.Resource)
}
