package gomad3sim

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveFaultLifecycleStates(t *testing.T) {
	kinds := []struct {
		kind        FaultKind
		incarnation uint64
		allowed     []bool
		canonical   string
	}{
		{FaultGracefulStop, 7, []bool{false, true, false, false, false, false}, `{"ordinal":4,"action":{"id":"lifecycle","kind":"graceful_stop","match":{},"node":"server"},"matched":{"operation":"deliver","occurrence":2},"target":{"node":"server","incarnation":7},"identity":""}`},
		{FaultHarshCrash, 7, []bool{false, true, false, false, false, false}, `{"ordinal":4,"action":{"id":"lifecycle","kind":"harsh_crash","match":{},"node":"server"},"matched":{"operation":"deliver","occurrence":2},"target":{"node":"server","incarnation":7},"identity":""}`},
		{FaultRestart, 8, []bool{false, false, true, true, true, true}, `{"ordinal":4,"action":{"id":"lifecycle","kind":"restart","match":{},"node":"server"},"matched":{"operation":"deliver","occurrence":2},"target":{"node":"server","incarnation":8},"identity":""}`},
	}
	states := []NodeState{NodeStateDefined, NodeStateRunning, NodeStateExited, NodeStateStopped, NodeStateCrashed, NodeStateFailed}
	for _, kind := range kinds {
		for index, state := range states {
			for _, busy := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/busy=%t", kind.kind, state, busy), func(t *testing.T) {
					node := &clusterNode{state: state, handle: NodeHandle{Node: "server", Incarnation: 7}}
					if busy {
						node.operation = &nodeOperation{}
					}
					cluster := &inProcessCluster{nodes: map[NodeID]*clusterNode{"server": node}}
					action := FaultAction{ID: "lifecycle", Kind: kind.kind, Node: "server"}
					matched := FaultMatch{Operation: "deliver", Occurrence: 2}
					got, err := cluster.resolveFaultLocked(4, action, matched)
					if !kind.allowed[index] || busy {
						require.Same(t, ErrFaultInapplicable, err)
						require.Equal(t, FaultRealization{}, got)
						return
					}
					require.NoError(t, err)
					digest := sha256.Sum256([]byte("gomad3-fault-realization/v1\x00" + kind.canonical))
					require.Equal(t, FaultRealization{
						Ordinal: 4, Action: action, Matched: matched,
						Target:   NodeHandle{Node: "server", Incarnation: kind.incarnation},
						Identity: fmt.Sprintf("sha256:%x", digest),
					}, got)
					require.Equal(t, state, node.state)
					require.Equal(t, NodeHandle{Node: "server", Incarnation: 7}, node.handle)
				})
			}
		}
	}
}

func TestResolveFaultLifecycleTargets(t *testing.T) {
	tests := []struct {
		name   string
		action FaultAction
		want   NodeHandle
	}{
		{"explicit stop", FaultAction{Kind: FaultGracefulStop, Node: "running"}, NodeHandle{Node: "running", Incarnation: 7}},
		{"explicit crash match", FaultAction{Kind: FaultHarshCrash, Node: "running", Match: FaultMatch{Node: "running", Incarnation: 7}}, NodeHandle{Node: "running", Incarnation: 7}},
		{"restart matches next incarnation", FaultAction{Kind: FaultRestart, Node: "stopped", Match: FaultMatch{Node: "stopped", Incarnation: 8}}, NodeHandle{Node: "stopped", Incarnation: 8}},
		{"missing node", FaultAction{Kind: FaultGracefulStop, Node: "missing"}, NodeHandle{}},
		{"wrong match node", FaultAction{Kind: FaultHarshCrash, Node: "running", Match: FaultMatch{Node: "stopped"}}, NodeHandle{}},
		{"wrong stop incarnation", FaultAction{Kind: FaultGracefulStop, Node: "running", Match: FaultMatch{Incarnation: 8}}, NodeHandle{}},
		{"restart rejects current incarnation", FaultAction{Kind: FaultRestart, Node: "stopped", Match: FaultMatch{Incarnation: 7}}, NodeHandle{}},
		{"candidate crash", FaultAction{ID: "candidate", Kind: FaultHarshCrash, Candidates: []NodeID{"running", "stopped"}}, NodeHandle{}},
		{"candidate restart", FaultAction{ID: "candidate", Kind: FaultRestart, Candidates: []NodeID{"running", "stopped"}}, NodeHandle{Node: "stopped", Incarnation: 8}},
		{"prior restart", FaultAction{Kind: FaultRestart, TargetFrom: "prior"}, NodeHandle{Node: "stopped", Incarnation: 8}},
		{"missing prior", FaultAction{Kind: FaultRestart, TargetFrom: "missing", Candidates: []NodeID{"stopped"}}, NodeHandle{}},
		{"prior overrides explicit node", FaultAction{Kind: FaultRestart, Node: "running", TargetFrom: "prior"}, NodeHandle{Node: "stopped", Incarnation: 8}},
		{"unresolved prior retains explicit node", FaultAction{Kind: FaultGracefulStop, Node: "running", TargetFrom: "missing"}, NodeHandle{Node: "running", Incarnation: 7}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cluster := &inProcessCluster{
				seed: 19,
				nodes: map[NodeID]*clusterNode{
					"running": {state: NodeStateRunning, handle: NodeHandle{Node: "running", Incarnation: 7}},
					"stopped": {state: NodeStateStopped, handle: NodeHandle{Node: "stopped", Incarnation: 7}},
				},
				faults: []FaultRealization{
					{Action: FaultAction{ID: "prior"}, Target: NodeHandle{Node: "stopped", Incarnation: 3}},
					{Action: FaultAction{ID: "prior"}, Target: NodeHandle{Node: "running", Incarnation: 4}},
				},
			}
			got, err := cluster.resolveFaultLocked(4, test.action, FaultMatch{})
			if test.want == (NodeHandle{}) {
				require.Same(t, ErrFaultInapplicable, err)
				require.Equal(t, FaultRealization{}, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want, got.Target)
			require.Equal(t, test.action, got.Action)
			if len(test.action.Candidates) != 0 {
				test.action.Candidates[0] = "changed"
				require.Equal(t, []NodeID{"running", "stopped"}, got.Action.Candidates)
			}
		})
	}
}

func TestResolveFaultOuterControls(t *testing.T) {
	actions := []FaultAction{
		{Kind: FaultDisconnect, From: "left", To: "right"},
		{Kind: FaultReconnect, From: "left", To: "right"},
		{Kind: FaultDelay, From: "left", To: "right", DelayNanos: 12},
		{Kind: FaultPartition, Left: []NodeID{"left"}, Right: []NodeID{"right"}},
		{Kind: FaultHeal, Left: []NodeID{"left"}, Right: []NodeID{"right"}},
		{Kind: "unknown", Node: "left"},
	}
	for _, action := range actions {
		for _, missing := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/missing=%t", action.Kind, missing), func(t *testing.T) {
				cluster := &inProcessCluster{nodes: map[NodeID]*clusterNode{
					"left":  {state: NodeStateDefined, operation: &nodeOperation{}},
					"right": {state: NodeStateFailed, operation: &nodeOperation{}},
				}}
				if missing {
					delete(cluster.nodes, "right")
				}
				got, err := cluster.resolveFaultLocked(4, action, FaultMatch{})
				if missing || action.Kind == "unknown" {
					require.Same(t, ErrFaultInapplicable, err)
					require.Equal(t, FaultRealization{}, got)
					return
				}
				require.NoError(t, err)
				require.Equal(t, NodeHandle{}, got.Target)
				require.Equal(t, action, got.Action)
				if len(action.Left) != 0 {
					got.Action.Left[0] = "changed"
					got.Action.Right[0] = "changed"
					require.Equal(t, []NodeID{"left"}, action.Left)
					require.Equal(t, []NodeID{"right"}, action.Right)
				}
			})
		}
	}
}
