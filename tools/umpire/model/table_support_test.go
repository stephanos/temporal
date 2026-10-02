package model

import (
	"cmp"
	"encoding/json"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/golden"
	"google.golang.org/protobuf/encoding/protojson"
)

const activityIR = "../../../model/ir/activity.json"

var activityBaseline = sync.OnceValues(func() (*umpirespb.Model, error) { return Load(activityIR) })

func activityModel(t *testing.T) *umpirespb.Model {
	t.Helper()
	m, err := activityBaseline()
	require.NoError(t, err)
	return m
}

// tableSide is everything a table says, with a result's typed step left out: Go's is a Go value and
// the IR's an IR value, and each is already spelled by the result's keys. Evidence is its lines as a
// set; TestActivityEvidenceIsInCatalogOrder is about their order.
type tableSide struct {
	Machine, Owner, Entity, Stuck               string
	Family                                      Family
	States, Actions, Outcomes, Facts            []string
	Starts, Ends, Reachable, StateFields        []string
	Rows                                        []Row
	Disabled                                    []string
	Unknown                                     int
	Evidence                                    [][2]string
	IDs                                         IDs
	Fingerprint                                 string
	Assumptions                                 []Assumption
	RowResults, DisabledPairs, StatesTimesClass int
}

// sideOf reads a table as a tableSide. A pair is disabled when the table has no row for it; the rows
// of a table are states-major, so the pairs are listed in that order too.
func sideOf(t *Table) tableSide {
	side := tableSide{Machine: t.Machine, Owner: t.OwnerName(), Entity: t.Entity, Stuck: t.Stuck, Family: t.Family,
		States: t.States, Actions: t.Actions, Outcomes: t.Outcomes, Facts: t.Facts, Starts: t.Starts, Ends: t.Ends,
		Reachable: t.Reachable, StateFields: t.StateFields, Unknown: len(t.Unknown), Evidence: [][2]string{},
		IDs: t.IDs(), Fingerprint: t.TargetFingerprint(), Assumptions: t.Assumptions,
		StatesTimesClass: len(t.States) * len(t.Actions)}
	side.Evidence = append(side.Evidence, t.Evidence...)
	slices.SortFunc(side.Evidence, func(a, b [2]string) int { return cmp.Compare(a[0], b[0]) })
	enabled := map[string]bool{}
	for _, row := range t.Rows {
		enabled[row.Key] = true
		plain := Row{Key: row.Key, Source: row.Source, Action: row.Action}
		for _, res := range row.Results {
			res.Step = nil
			plain.Results = append(plain.Results, res)
			side.RowResults++
		}
		side.Rows = append(side.Rows, plain)
	}
	for _, s := range t.States {
		for _, a := range t.Actions {
			if key := s + "-" + a; !enabled[key] {
				side.Disabled = append(side.Disabled, key)
			}
		}
	}
	side.DisabledPairs = len(side.Disabled)
	return side
}

func rowIndex(t *testing.T, table *Table, key string) int {
	t.Helper()
	for i, row := range table.Rows {
		if row.Key == key {
			return i
		}
	}
	require.Failf(t, "no row", "no row %s in %s", key, table.Machine)
	return -1
}

var readerBaseline = sync.OnceValues(func() (map[string][]byte, error) {
	return golden.Read(filepath.Join("testdata", "migration"))
})

func frozenReaderJSON(t *testing.T, path string, into any) {
	t.Helper()
	files, err := readerBaseline()
	require.NoError(t, err)
	encoded, ok := files[path]
	require.True(t, ok, path)
	require.NoError(t, json.Unmarshal(encoded, into), path)
}

func frozenReaderModel(t *testing.T, name string) *umpirespb.Model {
	t.Helper()
	files, err := readerBaseline()
	require.NoError(t, err)
	m := new(umpirespb.Model)
	require.NoError(t, protojson.Unmarshal(files["inputs/ir/"+name+".json"], m))
	return m
}

func frozenReaderMeaning(t *testing.T, name string) migrationSemantics {
	t.Helper()
	var meaning migrationSemantics
	frozenReaderJSON(t, "semantics/ir/"+name+".json", &meaning)
	return meaning
}
