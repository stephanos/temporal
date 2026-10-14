package interp

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

// A machine derived from another is declared with the same types and steps under its own name, and
// reads as that machine does: the same states, classes, rows and transitions, under its own name.
// What one machine's interpretation holds is its own: changing it changes no other machine's.
func TestAMachineDeclaredAlikeReadsAsTheOneItIsDerivedFrom(t *testing.T) {
	m := proto.CloneOf(readIR(t, activityIR))
	var system *umpirespb.Machine
	for _, mm := range m.GetMachines() {
		if mm.GetName() == "activitySystem" {
			system = mm
		}
	}
	require.NotNil(t, system)
	derived := proto.CloneOf(system)
	derived.Name, derived.Refines = "derivedSystem", nil
	derived.Position = &umpirespb.Position{File: "Derived.scala", Line: 1}
	m.Machines = append(m.Machines, derived)

	shared := NewInterpreter(m).Interpret(m)
	require.Empty(t, shared.Failed)
	alone := proto.CloneOf(m)
	alone.Machines = []*umpirespb.Machine{derived}
	own := NewInterpreter(alone).Interpret(alone)
	require.Empty(t, own.Failed)

	of, as, by := shared.Machines["activitySystem"], shared.Machines["derivedSystem"], own.Machines["derivedSystem"]
	require.Equal(t, "derivedSystem", as.Table.Machine)
	require.Equal(t, "derivedSystem", as.Decl.GetName())
	for _, read := range []*Machine{of, by} {
		require.Equal(t, read.Table.States, as.Table.States)
		require.Equal(t, read.Table.Actions, as.Table.Actions)
		require.Equal(t, read.Table.Rows, as.Table.Rows)
		require.Equal(t, read.Table.Starts, as.Table.Starts)
		require.Equal(t, read.Table.Reachable, as.Table.Reachable)
		require.Equal(t, read.Transitions, as.Transitions)
		require.Equal(t, read.Holes, as.Holes)
		require.Equal(t, read.Work, as.Work)
	}
	require.Equal(t, by.Table.StateFields, as.Table.StateFields)
	require.NotEqual(t, of.Table.StateFields, as.Table.StateFields, "only the refining machine carries its refined state")

	row, transition := of.Table.Rows[0], of.Transitions[0]
	as.Table.Rows[0] = Row{Key: "changed"}
	as.Transitions[0] = Transition{Row: "changed"}
	as.Classes[0] = Class{Key: "changed"}
	require.Equal(t, row, of.Table.Rows[0])
	require.Equal(t, transition, of.Transitions[0])
	require.NotEqual(t, "changed", of.Classes[0].Key)
}
