package lint

// The law kinds read only the law sidecar, so each is held to sidecars of testdata/laws whose
// vocabulary (Lockable, lockedRefuses) is no kit's: kept.laws.json triggers none of them, and each
// other fixture changes kept in one place to trigger one.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/check"
)

// sidecar is the law sidecar of testdata/laws named name, read as lint reads one beside an IR file.
func sidecar(t *testing.T, name string) *check.LawSidecar {
	t.Helper()
	s, err := check.ReadLawSidecar(filepath.Join("testdata", "laws", name+".json"))
	require.NoError(t, err)
	require.NotNil(t, s)
	return s
}

// withLaws is a Model of the activity IR read with the named sidecar beside it; the law kinds read
// nothing of the IR.
func withLaws(t *testing.T, name string, options Options) *Model {
	t.Helper()
	m := read(t, activityIR)
	m.Laws, m.options = sidecar(t, name), options
	return m
}

// findingsOf is every finding of one kind over the Model, as kind, owner, subject and position.
func findingsOf(t *testing.T, m *Model, kind func(*Model) ([]Tally, error)) [][4]string {
	t.Helper()
	tallies, err := kind(m)
	require.NoError(t, err)
	var out [][4]string
	for _, x := range tallies {
		for _, f := range x.Findings {
			require.NotEmpty(t, f.Message)
			out = append(out, [4]string{string(f.Kind), f.Owner, f.Subject, f.Position})
		}
	}
	return out
}

func TestTheKeptSidecarTriggersNoLawKindButItsWaivers(t *testing.T) {
	m := withLaws(t, "kept", Options{})
	for _, kind := range []func(*Model) ([]Tally, error){lawsWaivedWithoutReason, reasonsNamingNoLaw, parametersWithoutCitation, lawsWithOneInstance} {
		require.Empty(t, findingsOf(t, m, kind))
	}
	// A waiver with a reason, of a law the catalog brings, is a finding the forwarded reason accepts.
	require.Equal(t, [][4]string{
		{"waived-law", "gate", "gate.lockedRefuses", "fixture/Gate.scala:14"},
		{"waived-law", "shed", "shed.lockedStaysLocked", "fixture/Shed.scala:8"},
	}, findingsOf(t, m, waivedLaws))
	verdict := Forward(&Accepted{}, m.Laws).Judge(lawFindings(t, m))
	require.False(t, verdict.Failed(), "%+v", verdict)
	require.Len(t, verdict.Accepted, 2)
	require.Equal(t, "a shed's lock is a latch: server/shed.go Unlatch", verdict.Accepted[1].Because)
}

// lawFindings is every finding of the law kinds over the Model.
func lawFindings(t *testing.T, m *Model) []Finding {
	t.Helper()
	var out []Finding
	for _, k := range kinds() {
		if !slices.Contains([]Kind{WaivedLaw, LawWaivedWithoutReason, ReasonNamesNoLaw, ParameterWithoutCitation, LawWithOneInstance}, k.kind) {
			continue
		}
		tallies, err := k.run(m)
		require.NoError(t, err)
		for _, x := range tallies {
			out = append(out, x.Findings...)
		}
	}
	return out
}

func TestAWaiverWithoutAReasonIsReportedAndNotForwarded(t *testing.T) {
	m := withLaws(t, "unreasoned", Options{})
	require.Equal(t, [][4]string{{"law-waived-without-reason", "shed", "shed.lockedStaysLocked", "fixture/Shed.scala:8"}},
		findingsOf(t, m, lawsWaivedWithoutReason))
	require.Equal(t, [][4]string{{"waived-law", "gate", "gate.lockedRefuses", "fixture/Gate.scala:14"}}, findingsOf(t, m, waivedLaws))
	require.Len(t, Forward(&Accepted{}, m.Laws).Accepted, 1)
}

func TestAReasonNamingNoCatalogLawIsReportedAndNotForwarded(t *testing.T) {
	m := withLaws(t, "nolaw", Options{})
	tallies, err := reasonsNamingNoLaw(m)
	require.NoError(t, err)
	require.Equal(t, [][4]string{{"reason-names-no-law", "shed", "shed.lockedIsSilent", "fixture/Shed.scala:8"}},
		findingsOf(t, m, reasonsNamingNoLaw))
	require.Equal(t, `because "a shed's lock is a latch: server/shed.go Unlatch" names lockedIsSilent, a law the catalog does not bring`,
		tallies[1].Findings[0].Message)
	require.Len(t, Forward(&Accepted{}, m.Laws).Accepted, 1)
}

func TestAParameterBoundWithoutACitationIsReported(t *testing.T) {
	m := withLaws(t, "uncited", Options{})
	tallies, err := parametersWithoutCitation(m)
	require.NoError(t, err)
	require.Equal(t, [][4]string{{"parameter-without-citation", "door", "refused", "fixture/Door.scala:10"}},
		findingsOf(t, m, parametersWithoutCitation))
	require.Equal(t, `refused = fixture.Outcome.refused cites no server code, and lockedRefuses has each instance cite it: `+
		`bind cited(value, "<server file>")`, tallies[0].Findings[0].Message)
	// The gate's override takes the parameter as well, and cites it.
	require.Equal(t, 1, tallies[1].Population)
}

func TestALawWithOneInstantiatingMachineIsReported(t *testing.T) {
	m := withLaws(t, "lonely", Options{})
	tallies, err := lawsWithOneInstance(m)
	require.NoError(t, err)
	require.Equal(t, [][4]string{{"law-with-one-instance", "catalog", "lockedStaysLocked", "fixture/Catalog.scala:5"}},
		findingsOf(t, m, lawsWithOneInstance))
	require.Equal(t, "lockedStaysLocked is instantiated by door with its own state type: a law joins the catalog with two",
		tallies[0].Findings[0].Message)

	// Counted across every sidecar a run reads, a second machine elsewhere makes two; a machine of a
	// state type already counted does not.
	elsewhere := &check.LawSidecar{Catalog: []check.LawEntry{{Law: "lockedStaysLocked", Position: "x:1",
		Instantiating: []check.LawInstance{{Machine: "barn", State: "fixture.BarnState"}}}}}
	require.Empty(t, findingsOf(t, withLaws(t, "lonely", Options{Instances: CountInstances(m.Laws, elsewhere)}), lawsWithOneInstance))
	derived := &check.LawSidecar{Catalog: []check.LawEntry{{Law: "lockedStaysLocked", Position: "x:1",
		Instantiating: []check.LawInstance{{Machine: "backDoor", State: "fixture.DoorState"}}}}}
	require.Len(t, findingsOf(t, withLaws(t, "lonely", Options{Instances: CountInstances(m.Laws, derived)}), lawsWithOneInstance), 1)
}

// A sidecar the reader refuses is its error, and gives no findings: lint reads no IR file whose
// sidecar is malformed.
func TestAMalformedSidecarIsTheReadersErrorWithNoFindings(t *testing.T) {
	for name, says := range map[string]string{
		"unknown-field":  `unknown field "citation"`,
		"unlisted-claim": "claim door.lockedIsSilent is of lockedIsSilent, which the catalog does not list",
		"bare-instance":  "cannot unmarshal string",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := check.ReadLawSidecar(filepath.Join("testdata", "laws", name+".json"))
			require.ErrorContains(t, err, filepath.Join("testdata", "laws", name+".laws.json")+": ")
			require.ErrorContains(t, err, says)

			dir := t.TempDir()
			ir := filepath.Join(dir, "lawful.json")
			encoded, err := os.ReadFile("../../../model/ir/activity.json")
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(ir, encoded, 0o644))
			malformed, err := os.ReadFile(filepath.Join("testdata", "laws", name+".laws.json"))
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(check.LawSidecarPath(ir), malformed, 0o644))
			m, err := Read(ir, unrealizedByMachine, Options{})
			require.ErrorContains(t, err, says)
			require.Nil(t, m)
		})
	}
}

// Forward replaces every forwarded acceptance with the sidecar's waivers and keeps the author's own
// acceptances, in their order, before them.
func TestForwardReplacesTheWaiversAndKeepsTheAuthorsAcceptances(t *testing.T) {
	authored := Acceptance{Kind: UnaskedProperty, Owner: "door", Subjects: []string{"door.opens"}, Because: "kept for a later Query"}
	gone := Acceptance{Kind: WaivedLaw, Owner: "door", Subjects: []string{"door.lockedStaysLocked"}, Because: "no longer waived"}
	forwarded := Forward(&Accepted{Accepted: []Acceptance{gone, authored}}, sidecar(t, "kept"))
	require.Equal(t, []Acceptance{
		authored,
		{Kind: WaivedLaw, Owner: "gate", Subjects: []string{"gate.lockedRefuses"}, Because: "a gate answers a repeated request as the first: server/gate.go Open"},
		{Kind: WaivedLaw, Owner: "shed", Subjects: []string{"shed.lockedStaysLocked"}, Because: "a shed's lock is a latch: server/shed.go Unlatch"},
	}, forwarded.Accepted)
	require.Equal(t, []Acceptance{authored}, Forward(&Accepted{Accepted: []Acceptance{gone, authored}}, nil).Accepted)
}

// A waiver with no reason, or of a law the catalog does not bring, excuses nothing, so the accepted
// findings may not accept one; and a sidecar may waive a law of a machine once.
func TestWaiverFaultsCannotBeAcceptedNorWaivedTwice(t *testing.T) {
	for _, k := range []Kind{LawWaivedWithoutReason, ReasonNamesNoLaw} {
		a := Accepted{Accepted: []Acceptance{{Kind: k, Owner: "shed", Subjects: []string{"shed.lockedStaysLocked"}, Because: "kept"}}}
		require.ErrorContains(t, a.check(), "acceptance 0 accepts "+string(k)+", which is fixed in the declaration, not accepted")
	}
	twice := sidecar(t, "kept")
	twice.Waivers = append(twice.Waivers, twice.Waivers[1])
	encoded, err := json.Marshal(twice)
	require.NoError(t, err)
	ir := filepath.Join(t.TempDir(), "twice.json")
	require.NoError(t, os.WriteFile(check.LawSidecarPath(ir), encoded, 0o644))
	_, err = check.ReadLawSidecar(ir)
	require.ErrorContains(t, err, "shed.lockedStaysLocked is waived twice")
}
