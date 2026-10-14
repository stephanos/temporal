package check

// The claims of the Scala framework's own declaration tests that no other test here states, asserted
// over the lifted fixtures: an optional value's catalog, what a channel holds, and what a refinement's
// visible projection reads. .plans/umpire-scala-evaluator-audit.md maps each to the test it replaces.

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/ir"
)

// The disk's flush, a stutter of the store, made to record stored, the fact its put records too.
func flushRecordsStored(m *umpirespb.Model) {
	step := function(m, "Declarations$package$.flushStep").GetBody().GetMatch().GetCases()[0].GetBody().GetList().GetItems()[0]
	facts := step.GetConstruct().GetArgs()[2].GetList()
	facts.Items = append(facts.Items, admLiteral(facts.GetItems()[0], admEnum("fixture.declarations.Fact", "stored")))
}

// A refinement that names no visible facts reads none of a stutter's: the flush that records stored
// stays a stutter, and is none once the refinement names stored visible.
func TestWithoutAVisibleProjectionAStuttersFactsAreNotRead(t *testing.T) {
	unprojected := mutated(t, "declarations", noCrash, flushRecordsStored, func(m *umpirespb.Model) {
		admMachine(m, "disk").GetRefines().Visible = ""
	})
	require.Equal(t, Verified, receiptOf(t, checked(t, unprojected), "refinement disk store").Kind)
	put := "put"
	rows, err := refinementOf(t, unprojected, "disk")
	require.NoError(t, err)
	require.Equal(t, []RefinementRow{{Key: "empty-put", Product: &put}, {Key: "staged-flush"}}, rows)

	projected := receiptOf(t, checked(t, mutated(t, "declarations", noCrash, flushRecordsStored)), "refinement disk store")
	require.Equal(t, []any{RefinementRejected, RefinementVisibleStutter}, []any{projected.Kind, projected.Failure})
}

// A step that carries a step of the refined machine records no visible fact that step does not: the
// disk's put made to record staged too, which the store's put does not, and every fact made visible.
// The flush, a stutter, is made to record nothing, so the put is the one row a visible fact fails.
func TestACarriedStepsVisibleFactsAreFactsTheProductStepRecords(t *testing.T) {
	flushRecordsNothing := func(m *umpirespb.Model) {
		step := function(m, "Declarations$package$.flushStep").GetBody().GetMatch().GetCases()[0].GetBody().GetList().GetItems()[0]
		step.GetConstruct().GetArgs()[2].GetList().Items = nil
	}
	putRecordsStaged := func(m *umpirespb.Model) {
		facts := function(m, "Declarations$package$.putStep").GetBody().GetMatch().GetCases()[0].GetBody().GetList().GetItems()[0].GetConstruct().GetArgs()[2].GetList()
		facts.Items = append(facts.Items, admLiteral(facts.GetItems()[0], admEnum("fixture.declarations.Fact", "staged")))
	}
	seesAll := returning("disk.visible", boolValue(true))

	noisy := receiptOf(t, checked(t, mutated(t, "declarations", noCrash, flushRecordsNothing, putRecordsStaged, seesAll)), "refinement disk store")
	require.Equal(t, []any{RefinementRejected, RefinementUnmatched}, []any{noisy.Kind, noisy.Failure})
	require.Equal(t, []string{"put"}, taken(noisy.Witness))

	// The control: the same put recording only stored is the store's put.
	quiet := receiptOf(t, checked(t, mutated(t, "declarations", noCrash, flushRecordsNothing, seesAll)), "refinement disk store")
	require.Equal(t, Verified, quiet.Kind)
}

// A machine that names the facts, or only the outcomes, a refined machine sees and refines none is
// refused at its Scala line.
func TestNamingWhatARefinedMachineSeesRequiresARefinement(t *testing.T) {
	for name, keep := range map[string]func(r *umpirespb.Refinement){
		"facts only":    func(r *umpirespb.Refinement) { r.VisibleOutcomes = "" },
		"outcomes only": func(r *umpirespb.Refinement) { r.Visible = "" },
	} {
		t.Run(name, func(t *testing.T) {
			m := mutated(t, "declarations", func(m *umpirespb.Model) {
				r := admMachine(m, "disk").GetRefines()
				r.Product = ""
				keep(r)
			})
			require.ErrorContains(t, ir.Validate(m), admDeclaredAt+"93: disk names what a refined machine sees but refines none")
		})
	}
}
