package lower_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/tools/umpire/internal/golden"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestMigrationArtifactProjectionIsClosed(t *testing.T) {
	expected, err := golden.Read(filepath.Join("testdata", "migration"))
	require.NoError(t, err)
	cfg, err := golden.Configuration()
	require.NoError(t, err)
	before, after := map[string][]byte{}, map[string][]byte{}
	for name, encoded := range expected {
		switch {
		case strings.HasPrefix(name, "original/"):
			before[strings.TrimPrefix(name, "original/")] = encoded
		case strings.HasPrefix(name, "mapped/"):
			after[strings.TrimPrefix(name, "mapped/")] = encoded
		default:
			require.True(t, strings.HasPrefix(name, "oracles/"), "unknown baseline entry %s", name)
		}
	}
	require.Len(t, after, len(before))
	for name, original := range before {
		mapped, exists := after[name]
		require.True(t, exists, "missing projected artifact %s", name)
		if strings.HasSuffix(name, "/case.json") {
			oldCase, newCase := new(testpilotspb.Case), new(testpilotspb.Case)
			require.NoError(t, protojson.Unmarshal(original, oldCase))
			require.NoError(t, protojson.Unmarshal(mapped, newCase))
			projected := proto.CloneOf(oldCase)
			migrateCaseSources(t, projected.ProtoReflect(), cfg)
			if strings.Contains(name, "/explorations/") {
				key := strings.TrimSuffix(name, "/case.json")
				oldModel, newModel := new(umpirespb.Model), new(umpirespb.Model)
				require.NoError(t, protojson.Unmarshal(before[key+"/model.json"], oldModel))
				require.NoError(t, protojson.Unmarshal(after[key+"/model.json"], newModel))
				projectedModel, err := cfg.Migrate(oldModel)
				require.NoError(t, err)
				protorequire.ProtoEqual(t, projectedModel, newModel)
				oldWire, err := proto.MarshalOptions{Deterministic: true}.Marshal(oldModel)
				require.NoError(t, err)
				newWire, err := proto.MarshalOptions{Deterministic: true}.Marshal(projectedModel)
				require.NoError(t, err)
				_, setAndRest, ok := strings.Cut(name, "/explorations/")
				require.True(t, ok)
				set, _, _ := strings.Cut(setAndRest, "/")
				oldID := cp.IdentityFor("temporal.case", "scala.explore."+set, golden.Digest(oldWire))
				newID := cp.IdentityFor("temporal.case", "scala.explore."+set, golden.Digest(newWire))
				migrateCaseIdentity(t, projected, oldID, newID)
				require.Equal(t, assessmentIdentity(t, oldModel), assessmentIdentity(t, newModel))
			}
			protorequire.ProtoEqual(t, projected, newCase)
		}
		if strings.HasSuffix(name, "/lowering.json") {
			var a, b any
			require.NoError(t, json.Unmarshal(original, &a))
			require.NoError(t, json.Unmarshal(mapped, &b))
			remapPositions(a, cfg)
			require.Equal(t, a, b, name)
		}
	}
}

func migrateCaseSources(t *testing.T, m protoreflect.Message, cfg golden.Config) {
	t.Helper()
	source := (&testpilotspb.SourceLocation{}).ProtoReflect().Descriptor().FullName()
	if m.Descriptor().FullName() == source {
		field := m.Descriptor().Fields().ByName("path")
		value := m.Get(field).String()
		for _, path := range cfg.Paths {
			if value == path.Old {
				m.Set(field, protoreflect.ValueOfString(path.New))
				return
			}
		}
		require.FailNow(t, "unlisted Case source", value)
	}
	m.Range(func(f protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if f.Message() == nil || f.IsMap() {
			return true
		}
		if f.IsList() {
			for i := range value.List().Len() {
				migrateCaseSources(t, value.List().Get(i).Message(), cfg)
			}
		} else {
			migrateCaseSources(t, value.Message(), cfg)
		}
		return true
	})
}

func migrateCaseIdentity(t *testing.T, c *testpilotspb.Case, oldID, newID cp.Identity) {
	t.Helper()
	require.Equal(t, []string{oldID.CaseID, oldID.ProgramID, oldID.ContractID}, []string{c.CaseId, c.Program.ProgramId, c.Contract.ContractId})
	c.CaseId, c.Program.ProgramId, c.Contract.ContractId = newID.CaseID, newID.ProgramID, newID.ContractID
	oldWorkflow, newWorkflow := "umpire-"+oldID.Fixture+"-workflow", "umpire-"+newID.Fixture+"-workflow"
	for _, entry := range c.Program.Entrypoints {
		if workflow := entry.GetWorkflow(); workflow != nil && workflow.WorkflowType == oldWorkflow {
			workflow.WorkflowType = newWorkflow
		}
		for _, node := range entry.GetInstructions() {
			for _, assignment := range node.GetInstruction().GetInvokeRpc().GetRequestAssignments() {
				literal := assignment.GetValue().GetLiteral()
				if literal.GetTextValue() == oldWorkflow {
					literal.Value = &testpilotspb.Value_TextValue{TextValue: newWorkflow}
				}
			}
		}
	}
	for _, evidence := range c.Program.Evidence {
		for _, scope := range evidence.Scope {
			if scope.GetValue().GetTextValue() == oldID.RunScope {
				scope.Value.Value = &testpilotspb.Value_TextValue{TextValue: newID.RunScope}
			}
		}
	}
}

func remapPositions(value any, cfg golden.Config) {
	switch value := value.(type) {
	case []any:
		for _, item := range value {
			remapPositions(item, cfg)
		}
	case map[string]any:
		for key, child := range value {
			if key == "Position" {
				if position, ok := child.(string); ok {
					for _, path := range cfg.Paths {
						if strings.HasPrefix(position, path.Old+":") {
							value[key] = path.New + strings.TrimPrefix(position, path.Old)
							break
						}
					}
				}
			} else {
				remapPositions(child, cfg)
			}
		}
	default:
	}
}
