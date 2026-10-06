package lint

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var updateCoverage = flag.Bool("update-coverage", false, "rewrite testdata/coverage.golden")

// coverageFixtures is what the coverage golden pins: the lifter's fixture Models and two of the
// checked-in IR files light enough to read in a unit test.
var coverageFixtures = []string{
	"model/ir/activity-standalone.json",
	"model/ir/nexus-workflow-control.json",
	"model/irgen/testdata/lifts/expected/admission.json",
	"model/irgen/testdata/lifts/expected/capabilities.json",
	"model/irgen/testdata/lifts/expected/captured.json",
	"model/irgen/testdata/lifts/expected/channels.json",
	"model/irgen/testdata/lifts/expected/declarations.json",
	"model/irgen/testdata/lifts/expected/presence.json",
	"model/irgen/testdata/lifts/expected/realizations.json",
	"model/irgen/testdata/lifts/expected/taskqueue.json",
}

func fixtureLowering() Lowering {
	l := apiRegistryLowering(nil)
	l.Unrealized = unrealizedByMachine.Unrealized
	return l
}

func lintFixture(t *testing.T, file string) *Result {
	t.Helper()
	m, err := Read(filepath.Join("..", "..", "..", filepath.FromSlash(file)), fixtureLowering(), Options{})
	require.NoError(t, err)
	m.File = file
	r, err := m.Lint()
	require.NoError(t, err)
	return r
}

func TestCoverageSummaryIsPinned(t *testing.T) {
	var out strings.Builder
	for _, file := range coverageFixtures {
		require.NoError(t, WriteCoverage(&out, lintFixture(t, file)))
	}
	golden := filepath.Join("testdata", "coverage.golden")
	if *updateCoverage {
		require.NoError(t, os.WriteFile(golden, []byte(out.String()), 0o644))
	}
	want, err := os.ReadFile(golden)
	require.NoError(t, err)
	require.Equal(t, string(want), out.String(), "go test -tags test_dep ./tools/umpire/lint -run TestCoverageSummaryIsPinned -update-coverage rewrites it")

	// A second reading of one file gives the same bytes.
	var again, first strings.Builder
	require.NoError(t, WriteCoverage(&first, lintFixture(t, coverageFixtures[0])))
	require.NoError(t, WriteCoverage(&again, lintFixture(t, coverageFixtures[0])))
	require.Equal(t, first.String(), again.String())
}

func TestEveryCountIsItsKindsFindings(t *testing.T) {
	counted := map[Kind]bool{}
	for _, file := range coverageFixtures {
		r := lintFixture(t, file)
		findings := map[[2]string]int{}
		for _, f := range r.Findings() {
			findings[[2]string{string(f.Kind), f.Owner}]++
		}
		for _, c := range r.Coverage() {
			counted[c.Kind] = true
			require.Equal(t, findings[[2]string{string(c.Kind), c.Owner}], c.Population-c.Satisfied, "%s %s %s", file, c.Owner, c.Kind)
		}
	}
	// Every count of the summary is printed somewhere over the fixtures.
	for _, k := range kinds() {
		if k.count != nil {
			require.True(t, counted[k.kind], "no fixture prints the count of %s", k.kind)
		}
	}
}

func TestAcceptedFindingsAreJudgedBothWays(t *testing.T) {
	findings := []Finding{
		{Kind: SilentRejection, Owner: "m", Subject: "a in x"},
		{Kind: SilentRejection, Owner: "m", Subject: "b in x"},
	}
	a := &Accepted{Accepted: []Acceptance{{Kind: SilentRejection, Owner: "m", Subjects: []string{"a in x", "c in x"}, Because: "frozen"}}}
	v := a.Judge(findings)
	require.Len(t, v.Accepted, 1)
	require.Equal(t, "frozen", v.Accepted[0].Because)
	require.Equal(t, []Finding{findings[1]}, v.Unaccepted)
	require.Equal(t, []string{`silent-rejection m "c in x" matches no finding`}, v.Stale)
	require.True(t, v.Failed())

	a.Accepted[0].Subjects = []string{"a in x", "b in x"}
	require.False(t, a.Judge(findings).Failed())
}

func TestAcceptedFileRefusesWhatItCannotJudge(t *testing.T) {
	dir := t.TempDir()
	ir := filepath.Join(dir, "m.json")
	for body, problem := range map[string]string{
		`{"accepted": [{"kind": "silent-rejection", "owner": "m", "subjects": ["a"]}]}`:                         "gives no reason",
		`{"accepted": [{"kind": "nothing", "owner": "m", "subjects": ["a"], "because": "x"}]}`:                  "names no kind",
		`{"accepted": [{"kind": "silent-rejection", "owner": "m", "subjects": [], "because": "x"}]}`:            "accepts nothing",
		`{"accepted": [{"kind": "silent-rejection", "owner": "m", "subjects": ["a", "a"], "because": "x"}]}`:    "accepted twice",
		`{"accepted": [{"kind": "silent-rejection", "owner": "m", "subjects": ["a"], "because": "x", "y": 1}]}`: "unknown field",
	} {
		require.NoError(t, os.WriteFile(AcceptedPath(ir), []byte(body), 0o644))
		_, err := ReadAccepted(ir)
		require.ErrorContains(t, err, problem)
	}
	a, err := ReadAccepted(filepath.Join(dir, "none.json"))
	require.NoError(t, err)
	require.Empty(t, a.Accepted)
}
