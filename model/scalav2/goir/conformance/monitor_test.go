package conformance

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/model/scalav2/goir"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// counted is the store with a monitor over the largest catalog a Model is admitted with, 65,536
// members: it counts steps from initial, saturating at the last member, and is violated at target.
func counted(t testing.TB, initial, target int) *modelirspb.Model {
	t.Helper()
	m := realized(t, lifted(t, "declarations"), store, []kindOf{{"stored", false}})
	for name, body := range map[string]string{
		"test.counter.next": `{"params":[{"name":"m","type":{"int":{}}},{"name":"before","type":{"named":"fixture.declarations.Store"}},{"name":"after","type":{"named":"umpire.Step"}}],
			"body":{"if":{"condition":{"binary":{"op":"OP_LT","left":{"var":"m"},"right":{"literal":{"int":"65535"}}}},
			"then":{"binary":{"op":"OP_ADD","left":{"var":"m"},"right":{"literal":{"int":"1"}}}},"else":{"var":"m"}}}}`,
		"test.counter.violated": fmt.Sprintf(`{"params":[{"name":"m","type":{"int":{}}}],
			"body":{"binary":{"op":"OP_EQ","left":{"var":"m"},"right":{"literal":{"int":"%d"}}}}}`, target),
	} {
		function := &modelirspb.Function{}
		require.NoError(t, protojson.Unmarshal([]byte(body), function))
		function.Name = name
		m.Functions = append(m.Functions, function)
	}
	m.Monitors = append(m.Monitors, &modelirspb.Monitor{Id: "test.counter", Name: "counter",
		State:   &modelirspb.TypeRef{Ref: &modelirspb.TypeRef_IntRange{IntRange: &modelirspb.IntRange{Low: 0, High: 65535}}},
		Initial: &modelirspb.Expr{Kind: &modelirspb.Expr_Literal{Literal: &modelirspb.Value{Kind: &modelirspb.Value_Int{Int: int64(initial)}}}},
		Next:    "test.counter.next", Violated: "test.counter.violated",
		Evaluate: &modelirspb.Monitor_EveryStep{EveryStep: &modelirspb.Empty{}}})
	for _, machine := range m.GetMachines() {
		if machine.GetName() == store {
			machine.Monitors = append(machine.Monitors, "test.counter")
		}
	}
	require.NoError(t, goir.Validate(m))
	return m
}

// A monitor state is told apart from every other and from a state a hole left unknown at every index
// an admitted catalog has: one put takes the counter from each of these states to the next, which
// alone violates it. The indexes straddle the last a signed 16-bit index holds and the last member.
func TestAMonitorStateIsItselfAtEveryIndexOfTheLargestCatalog(t *testing.T) {
	stored := script(store, fact{name: "o0", records: "stored"})[0].evidence
	for _, initial := range []int{0, 32766, 32767, 65534} {
		t.Run(fmt.Sprint(initial), func(t *testing.T) {
			factory, err := Prepare(counted(t, initial, initial+1), goir.ClaimKey{Family: declarationsFamily, Owner: store, Name: "putStores"},
				carrier(store, []string{"stored"}, 1), generous)
			require.NoError(t, err)
			established, outcome := assessDirectly(t, factory, completed, proto.CloneOf(stored))
			violation := testpilot.PropertyAssessment{ID: "counter", Status: testpilot.PropertyViolated, SupportingEventSequences: []int64{2},
				Detail: store + ", " + defaultInstance + ": " + whyViolated}
			require.Equal(t, testpilot.Established{Violations: []testpilot.PropertyAssessment{violation}}, established)
			require.Equal(t, []testpilot.PropertyAssessment{{ID: "putStores", Status: testpilot.PropertySatisfied, SupportingEventSequences: []int64{2}}, violation},
				outcome.Properties)
		})
	}
	// The same step from the same states leaves the counter short of a target one further on.
	factory, err := Prepare(counted(t, 65533, 65535), goir.ClaimKey{Family: declarationsFamily, Owner: store, Name: "putStores"},
		carrier(store, []string{"stored"}, 1), generous)
	require.NoError(t, err)
	established, outcome := assessDirectly(t, factory, completed, proto.CloneOf(stored))
	require.Equal(t, testpilot.Established{}, established)
	require.Equal(t, testpilot.PropertyAssessment{ID: "counter", Status: testpilot.PropertySatisfied, SupportingEventSequences: []int64{2}}, outcome.Properties[1])
}

// Candidates that differ only in a monitor's state are kept apart at every state, the last member of
// the largest catalog and a state a hole left unknown included.
func TestCandidatesAreKeyedByTheWholeMonitorState(t *testing.T) {
	keys := map[string]int32{}
	for _, state := range []int32{lost, 0, 32767, 32768, 65535} {
		key := (&candidate{cells: []cell{{state: state}}}).key()
		other, taken := keys[key]
		require.False(t, taken, "states %d and %d share a key", state, other)
		keys[key] = state
	}
}
