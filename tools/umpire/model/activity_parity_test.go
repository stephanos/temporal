package model

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// What the product machine disables is what the baseline pins by hand: a canceled answer without a
// cancel request, a worker stop, and a start of a paused activity.
func TestActivityDisabledBehaviorIsTheBaselines(t *testing.T) {
	product := built(t, activityModel(t))["activityProduct"]
	for _, pair := range [][2]string{
		{"started", "attemptResult-canceled"},
		{"paused", "attemptStart"},
		{"scheduled", "workerStop"},
		{"completed", "timeout"},
	} {
		require.True(t, product.Disabled(pair[0], pair[1]), pair)
	}
	require.False(t, product.Disabled("completed", "control-terminate"), "a control of an activity that is over is answered notFound, not disabled")
	require.Equal(t, []Result{{Outcome: "notFound", State: "completed", Facts: []string{}}},
		sideOf(product.Table).Rows[rowIndex(t, product.Table, "completed-control-terminate")].Results)
}

func TestActivityEvidenceIsInCatalogOrder(t *testing.T) {
	machines := built(t, activityModel(t))
	for _, subject := range frozenReaderMeaning(t, "activity").Subjects {
		if subject.Name == "activityProtocol" || subject.Name == "activityProduct" {
			require.Equal(t, subject.Table.Evidence, machines[subject.Name].Table.Evidence)
		}
	}
}

func TestActivityEveryClaimDeclarationIsLifted(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "..", "model", "temporal", "standaloneactivity", "Claims.scala"))
	require.NoError(t, err)
	declared := []string{}
	for _, match := range regexp.MustCompile(`(?:\.|\b)(property|scenario|query)\(\s*"([^"\n]+)"`).FindAllStringSubmatch(string(source), -1) {
		declared = append(declared, match[1]+" "+match[2])
	}
	m := activityModel(t)
	var lifted []string
	for _, p := range m.GetProperties() {
		lifted = append(lifted, "property "+p.GetName())
	}
	for _, s := range m.GetScenarios() {
		lifted = append(lifted, "scenario "+s.GetName())
	}
	for _, q := range m.GetQueries() {
		lifted = append(lifted, "query "+q.GetName())
	}
	require.NotEmpty(t, declared)
	require.ElementsMatch(t, declared, lifted)
}

func TestActivityTablesAccountForEveryPair(t *testing.T) {
	for name, mm := range built(t, activityModel(t)) {
		require.Empty(t, mm.Holes, name)
		require.NoError(t, mm.Rejected, name)
		got := sideOf(mm.Table)
		// The interpreter's own account of a disabled pair agrees with the rows.
		for _, state := range mm.Table.States {
			for _, class := range mm.Classes {
				require.Equal(t, !hasRow(mm, state+"-"+class.Key), mm.Disabled(state, class.Key), "%s-%s", state, class.Key)
			}
		}
		require.Equal(t, got.StatesTimesClass, got.DisabledPairs+len(got.Rows))
	}
}
