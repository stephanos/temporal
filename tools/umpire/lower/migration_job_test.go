package lower_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	runtime "go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	"go.temporal.io/server/tools/umpire/internal/golden"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/encoding/protojson"
)

var originalJobActions = []string{"submit", "take", "fail", "settle", "finish", "drop", "check", "close"}
var readerJobActions = []string{"check", "close", "drop", "fail", "finish", "settle", "submit", "take"}
var originalJobRows = []string{"idle-submit", "queued-take", "queued-drop", "running-fail", "waiting-settle", "running-finish", "running-check", "running-close"}
var readerJobRows = []string{"idle-submit", "queued-drop", "queued-take", "running-check", "running-close", "running-fail", "running-finish", "waiting-settle"}
var originalJobReachable = []string{"idle", "queued", "running", "dropped", "waiting", "done"}
var readerJobReachable = []string{"idle", "queued", "dropped", "running", "done", "waiting"}

func decodeJobJSON(data []byte, value any) error {
	if err := json.Unmarshal(data, value); err != nil {
		return err
	}
	encoded, err := golden.JSON(value)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, encoded) {
		return errors.New("job artifact is not losslessly represented")
	}
	return nil
}

func jobRowKeys(rows []umpiremodel.Row) []string {
	keys := make([]string, len(rows))
	for i, row := range rows {
		keys[i] = row.Key
	}
	return keys
}

// Only these frozen synthetic fixtures predate the reader's sorted class catalog.
func projectJobArtifacts(expected, actual map[string][]byte) (map[string][]byte, error) {
	projected := maps.Clone(expected)
	for _, name := range []string{"once", "retried", "closed"} {
		key := "oracles/job/" + name
		if err := projectJobArtifact(projected, actual, key, name); err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
	}
	return projected, nil
}

func projectJobArtifact(expected, actual map[string][]byte, key, name string) error {
	var before, after umpiremodel.Table
	if err := decodeJobJSON(expected[key+"/table.json"], &before); err != nil {
		return err
	}
	if err := decodeJobJSON(actual[key+"/table.json"], &after); err != nil {
		return err
	}
	if !slices.Equal(before.Actions, originalJobActions) || !slices.Equal(after.Actions, readerJobActions) ||
		!slices.Equal(jobRowKeys(before.Rows), originalJobRows) || !slices.Equal(jobRowKeys(after.Rows), readerJobRows) ||
		!slices.Equal(before.Reachable, originalJobReachable) || !slices.Equal(after.Reachable, readerJobReachable) {
		return errors.New("job catalog, row or reachability order changed")
	}
	rows := make(map[string]umpiremodel.Row, len(before.Rows))
	for _, row := range before.Rows {
		rows[row.Key] = row
	}
	for _, row := range after.Rows {
		original := rows[row.Key]
		if row.Key == "waiting-settle" || row.Key == "running-check" {
			if len(original.Results) != 1 || len(row.Results) != 1 || original.Results[0].Facts != nil || row.Results[0].Facts == nil || len(row.Results[0].Facts) != 0 {
				return fmt.Errorf("%s: unexpected empty-fact representation", row.Key)
			}
			original.Results[0].Facts = []string{}
		}
		if !reflect.DeepEqual(original, row) {
			return fmt.Errorf("%s: job row contents changed", row.Key)
		}
		rows[row.Key] = original
	}
	if err := projectJobIdentity(expected, key, name, &before); err != nil {
		return err
	}
	before.Actions = slices.Clone(readerJobActions)
	before.Reachable = slices.Clone(readerJobReachable)
	for i, row := range readerJobRows {
		before.Rows[i] = rows[row]
	}
	encoded, err := golden.JSON(&before)
	if err != nil {
		return err
	}
	expected[key+"/table.json"] = encoded
	return nil
}

func jobCaseIdentity(encoded []byte, c *testpilotspb.Case) ([]byte, error) {
	canonical, err := recordedrun.CaseIdentity(encoded)
	if err != nil {
		return nil, err
	}
	fingerprint, err := runtime.CaseFingerprint(c)
	if err != nil {
		return nil, err
	}
	return golden.JSON(struct{ Canonical, Fingerprint string }{canonical, fingerprint})
}

func projectJobIdentity(expected map[string][]byte, key, name string, table *umpiremodel.Table) error {
	var ids umpiremodel.IDs
	if err := decodeJobJSON(expected[key+"/ids.json"], &ids); err != nil {
		return err
	}
	// Family is excluded from Table JSON; the frozen IDs bind it independently.
	original := *table
	original.Family = "fixture.job"
	if !reflect.DeepEqual(ids, original.IDs()) {
		return errors.New("original job Definition IDs changed")
	}
	var property struct {
		Name   string
		Groups []umpiremodel.Group
	}
	if err := decodeJobJSON(expected[key+"/property.json"], &property); err != nil {
		return err
	}
	var scenario struct {
		Name, Start string
		Actions     []string
		Limits      umpiremodel.Limits
	}
	if err := decodeJobJSON(expected[key+"/scenario.json"], &scenario); err != nil {
		return err
	}
	query := &umpiremodel.Query{Name: name, Form: umpiremodel.FindForm, Property: &umpiremodel.PropertyDecl{Name: property.Name},
		Scenario: &umpiremodel.ScenarioDecl{Name: scenario.Name, Start: scenario.Start, Actions: scenario.Actions}, Limits: scenario.Limits}
	propertyFingerprint := umpiremodel.Fingerprint(original.PropertySemantic(query.Property.PropertyID(&original), property.Groups))
	beforeFingerprint := umpiremodel.Fingerprint(query.QueryCanonicalOf(&original, propertyFingerprint, original.TargetFingerprint()))
	targetFingerprint := original.TargetFingerprint()
	original.Actions = slices.Clone(readerJobActions)
	if original.TargetFingerprint() != targetFingerprint {
		return errors.New("job target fingerprint changed")
	}
	afterFingerprint := umpiremodel.Fingerprint(query.QueryCanonicalOf(&original, propertyFingerprint, original.TargetFingerprint()))
	c := new(testpilotspb.Case)
	if err := protojson.Unmarshal(expected[key+"/case.json"], c); err != nil {
		return err
	}
	encoded, err := golden.Proto(c)
	if err != nil {
		return err
	}
	if !bytes.Equal(encoded, expected[key+"/case.json"]) {
		return errors.New("original job Case is not losslessly represented")
	}
	identity, err := jobCaseIdentity(encoded, c)
	if err != nil {
		return err
	}
	if !bytes.Equal(identity, expected[key+"/identity.json"]) {
		return errors.New("original job Case identity changed")
	}
	found := 0
	for _, definition := range c.GetProvenance().GetDefinitions() {
		if definition.GetDefinitionId() == "fixture.job.query."+name {
			if definition.GetKind() != testpilotspb.DEFINITION_KIND_QUERY || definition.GetBehaviorFingerprint() != beforeFingerprint {
				return errors.New("original job Query fingerprint changed")
			}
			definition.BehaviorFingerprint = afterFingerprint
			found++
		}
	}
	if found != 1 {
		return errors.New("expected exactly one job Query provenance entry")
	}
	encoded, err = golden.Proto(c)
	if err != nil {
		return err
	}
	expected[key+"/case.json"] = encoded
	expected[key+"/identity.json"], err = jobCaseIdentity(encoded, c)
	if err != nil {
		return err
	}
	expected[key+"/ids.json"], err = golden.JSON(original.IDs())
	return err
}

func TestMigrationJobProjectionRejectsOtherChanges(t *testing.T) {
	expected, err := golden.Read("testdata/migration")
	require.NoError(t, err)
	actual := maps.Clone(expected)
	maps.Copy(actual, migrationJobArtifacts(t))
	require.NoError(t, compareMigrationArtifactInventory(expected, actual))
	mutations := map[string]func(*umpiremodel.Table){
		"other row order":    func(v *umpiremodel.Table) { v.Rows[1], v.Rows[2] = v.Rows[2], v.Rows[1] },
		"missing row":        func(v *umpiremodel.Table) { v.Rows = v.Rows[1:] },
		"duplicate row":      func(v *umpiremodel.Table) { v.Rows[1] = v.Rows[0] },
		"unknown row":        func(v *umpiremodel.Table) { v.Rows[0].Key = "unknown" },
		"result changed":     func(v *umpiremodel.Table) { v.Rows[0].Results[0].State = "done" },
		"result order":       func(v *umpiremodel.Table) { slices.Reverse(v.Rows[4].Results) },
		"fact order":         func(v *umpiremodel.Table) { slices.Reverse(v.Rows[2].Results[0].Facts) },
		"nonempty facts":     func(v *umpiremodel.Table) { v.Rows[3].Results[0].Facts = []string{"count"} },
		"nil facts":          func(v *umpiremodel.Table) { v.Rows[7].Results[0].Facts = nil },
		"result cardinality": func(v *umpiremodel.Table) { v.Rows[3].Results = append(v.Rows[3].Results, v.Rows[3].Results[0]) },
		"other nil to empty": func(v *umpiremodel.Table) { v.Assumptions = []umpiremodel.Assumption{} },
		"action order":       func(v *umpiremodel.Table) { slices.Reverse(v.Actions) },
		"reachable order":    func(v *umpiremodel.Table) { slices.Reverse(v.Reachable) },
	}
	for _, name := range []string{"once", "retried", "closed"} {
		key := "oracles/job/" + name
		for label, mutate := range mutations {
			t.Run(name+"/"+label, func(t *testing.T) {
				changed := maps.Clone(actual)
				var table umpiremodel.Table
				require.NoError(t, json.Unmarshal(actual[key+"/table.json"], &table))
				mutate(&table)
				changed[key+"/table.json"], err = golden.JSON(&table)
				require.NoError(t, err)
				require.Error(t, compareMigrationArtifactInventory(expected, changed))
			})
		}
		for label, mutate := range map[string]func(*testpilotspb.Case){
			"Query fingerprint":    func(c *testpilotspb.Case) { c.Provenance.Definitions[2].BehaviorFingerprint = "sha256:bogus" },
			"unrelated provenance": func(c *testpilotspb.Case) { c.Provenance.ProducerVersion += "changed" },
			"source link":          func(c *testpilotspb.Case) { c.Provenance.Sources[0].Path += "changed" },
			"duplicate Query": func(c *testpilotspb.Case) {
				c.Provenance.Definitions = append(c.Provenance.Definitions, c.Provenance.Definitions[2])
			},
		} {
			t.Run(name+"/"+label, func(t *testing.T) {
				changed := maps.Clone(actual)
				c := new(testpilotspb.Case)
				require.NoError(t, protojson.Unmarshal(actual[key+"/case.json"], c))
				mutate(c)
				changed[key+"/case.json"], err = golden.Proto(c)
				require.NoError(t, err)
				changed[key+"/identity.json"], err = jobCaseIdentity(changed[key+"/case.json"], c)
				require.NoError(t, err)
				require.Error(t, compareMigrationArtifactInventory(expected, changed))
			})
		}
		for _, artifact := range []string{"identity.json", "ids.json", "program.json", "contract.json", "scenario.json", "property.json", "table.json"} {
			t.Run(name+"/"+artifact, func(t *testing.T) {
				changed := maps.Clone(actual)
				changed[key+"/"+artifact] = append(slices.Clone(changed[key+"/"+artifact]), ' ')
				require.Error(t, compareMigrationArtifactInventory(expected, changed))
			})
		}
		t.Run(name+"/original canonical inputs", func(t *testing.T) {
			changed := maps.Clone(expected)
			changed[key+"/scenario.json"] = bytes.Replace(expected[key+"/scenario.json"], []byte(`"Start":"idle"`), []byte(`"Start":"queued"`), 1)
			require.ErrorContains(t, compareMigrationArtifactInventory(changed, actual), "original job Query fingerprint changed")
		})
		t.Run(name+"/original identity", func(t *testing.T) {
			changed := maps.Clone(expected)
			changed[key+"/identity.json"] = []byte(`{"Canonical":"bogus","Fingerprint":"bogus"}`)
			require.Error(t, compareMigrationArtifactInventory(changed, actual))
		})
	}
}
