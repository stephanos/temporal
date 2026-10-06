package lower

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/ir"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	"google.golang.org/protobuf/encoding/protojson"
)

func groupingFixture(t *testing.T, form string) *umpirespb.Model {
	t.Helper()
	m, err := ir.Load(filepath.Join("..", "check", "testdata", "grouping-"+form+".json"))
	require.NoError(t, err)
	require.NoError(t, ir.Validate(m))
	return m
}

// The source-derived extraction retains the executable programs, not only the shared handler IDs.
// The standalone fact-name ledger changes Definition identity, not RPCs, recorded keys or waits.
func TestGroupingBindingsKeepExecutablePrograms(t *testing.T) {
	forms := []struct {
		name    string
		queries []string
	}{
		{"nexus-workflow", functionalQueries},
		{"nexus-standalone", []string{"nexusSystem.terminateSettles", "nexusSystem.cancelIsRequested"}},
		{"activity-standalone", []string{"completion", "nonRetryableFailure", "pauseResume", "scheduleToStartTimeout", "terminate", "activitySystem.terminateSettles", "activitySystem.cancelIsRequested", "retry"}},
	}
	for _, form := range forms {
		t.Run(form.name, func(t *testing.T) {
			original, err := NewProducer(loaded(t, form.name))
			require.NoError(t, err)
			bound, err := NewProducer(groupingFixture(t, form.name))
			require.NoError(t, err)
			for _, query := range form.queries {
				t.Run(query, func(t *testing.T) {
					identity := cp.IdentityFor("temporal.case", "groupingSpike", query)
					before, err := original.Lower(query, identity)
					require.NoError(t, err)
					require.Equal(t, Lowered, before.Standing)
					after, err := bound.Lower(query, identity)
					require.NoError(t, err)
					require.Equal(t, Lowered, after.Standing, "%v", after.Unsupported)
					preparedAsIs(t, after.Case)
					// The archived fixture predates the committed comment conversion. These exact
					// wait-source coordinates also include the Activity fixture's added import.
					for _, entry := range before.Case.GetProgram().GetEntrypoints() {
						for _, node := range entry.GetInstructions() {
							for _, hint := range node.GetWaitHints() {
								switch hint.GetSource().GetPath() {
								case "model/temporal/realize/Behavior.scala":
									line, ok := map[int32]int32{74: 75, 82: 83, 86: 87, 89: 90, 94: 95, 99: 100}[hint.Source.Line]
									require.True(t, ok, "undeclared shared wait-source coordinate %d", hint.Source.Line)
									hint.Source.Line = line
								case "model/temporal/features/activity/standalone/system/Realization.scala":
									line, ok := map[int32]int32{226: 230, 227: 231}[hint.Source.Line]
									require.True(t, ok, "undeclared Activity wait-source coordinate %d", hint.Source.Line)
									hint.Source.Path = "model/irgen/testdata/grouping/activity/standalone/system/Realization.scala"
									hint.Source.Line = line
								default:
									require.FailNowf(t, "undeclared wait-source path", "%s", hint.GetSource().GetPath())
								}
							}
						}
					}
					want, err := protojson.Marshal(before.Case.GetProgram())
					require.NoError(t, err)
					got, err := protojson.Marshal(after.Case.GetProgram())
					require.NoError(t, err)
					ledger := strings.NewReplacer("temporal.features.", "fixture.features.")
					require.JSONEq(t, ledger.Replace(string(want)), string(got), "the finite source identity ledger leaves the entire executable program unchanged")
					oldContract, newContract := before.Case.GetContract().GetCorrelated(), after.Case.GetContract().GetCorrelated()
					require.Equal(t, oldContract.GetProjectionId(), newContract.GetProjectionId())
					require.Equal(t, oldContract.GetEvidenceObservationId(), newContract.GetEvidenceObservationId())
					require.Equal(t, oldContract.GetScopeFields(), newContract.GetScopeFields())
					require.Equal(t, oldContract.GetOperationField(), newContract.GetOperationField())
					var sourceNames []string
					for _, source := range oldContract.GetSources() {
						sourceNames = append(sourceNames, ledger.Replace(source))
					}
					require.ElementsMatch(t, sourceNames, newContract.GetSources(), "the same named evidence sources under the finite fact ledger; canonical identity sorting may reorder the set")
					require.NotEmpty(t, newContract.GetProjectionFingerprint())
					require.NotEqual(t, oldContract.GetProjectionFingerprint(), newContract.GetProjectionFingerprint(), "declared identity/catalog augmentation derives another projection; no fingerprint is stripped")
					if form.name == "nexus-standalone" {
						fields := map[string]string{}
						for _, field := range newContract.GetInitialStateFields() {
							fields[field.GetDefinitionId()] = field.GetValue()
						}
						require.Equal(t, map[string]string{"phase": "unstarted", "cancelRequested": "false", "nexusProduct": "scheduled"}, fields)
					}
				})
			}
		})
	}
}

// Operation identity is read from each form's actual protocol message, independently of Entity.name.
func TestGroupingRealizationsKeepTypedOperationKeys(t *testing.T) {
	for _, tc := range []struct {
		form, query, evidence, key, wrong, message string
	}{
		{"nexus-workflow", "asyncCompletion", "scheduled", "event_id", "operation_id", "temporal.api.history.v1.HistoryEvent"},
		{"nexus-workflow", "asyncCompletion", "completed", "attributes<nexus_operation_completed_event_attributes>.scheduled_event_id", "operation_id", "temporal.api.history.v1.HistoryEvent"},
		{"nexus-standalone", "nexusSystem.terminateSettles", "nexusOperationTerminated", "operation_id", "scheduled_event_id", "temporal.api.nexus.v1.NexusOperationExecutionInfo"},
	} {
		t.Run(tc.form+"/"+tc.evidence, func(t *testing.T) {
			m := groupingFixture(t, tc.form)
			r := m.GetRealizations()[0]
			var evidence *umpirespb.Evidence
			for _, e := range r.GetEvidence() {
				if strings.HasSuffix(e.GetId(), "."+tc.evidence) {
					evidence = e
					break
				}
			}
			require.NotNil(t, evidence)
			require.Equal(t, tc.key, evidence.GetOperation())
			element, err := EvidenceElement(r, evidence)
			require.NoError(t, err)
			require.Equal(t, tc.message, string(element.FullName()))
			positive, err := lowerDerived(t, m, tc.query)
			require.NoError(t, err)
			require.Equal(t, Lowered, positive.Standing)
			evidence.Operation = tc.wrong
			_, err = lowerDerived(t, m, tc.query)
			require.ErrorContains(t, err, "has no field "+tc.wrong)
		})
	}
}

func TestGroupingRetainsAllFormEntityBindings(t *testing.T) {
	for _, tc := range []struct {
		form, entity string
		on, creates  []string
	}{
		{"nexus-workflow", "operation", []string{"fixture.features.nexus.handler.reply", "fixture.features.nexus.handler.complete", "fixture.features.nexus.network.fault", "fixture.features.nexus.client.terminate"}, []string{"fixture.features.nexus.workflow.caller.schedule"}},
		{"nexus-standalone", "operation", []string{"fixture.features.nexus.handler.reply", "fixture.features.nexus.handler.complete", "fixture.features.nexus.network.fault", "fixture.features.nexus.client.terminate", "fixture.features.nexus.standalone.client.requestCancel"}, []string{"fixture.features.nexus.standalone.client.start"}},
		{"activity-standalone", "activity", []string{"fixture.features.activity.worker.poll", "fixture.features.activity.worker.respond", "fixture.features.activity.standalone.client.control"}, []string{"fixture.features.activity.standalone.client.start"}},
	} {
		t.Run(tc.form, func(t *testing.T) {
			m := groupingFixture(t, tc.form)
			actions := map[string]*umpirespb.Action{}
			for _, a := range m.GetActions() {
				actions[a.GetId()] = a
			}
			for _, id := range tc.on {
				require.NotNil(t, actions[id], id)
				require.Equal(t, tc.entity, actions[id].GetOn(), id)
			}
			for _, id := range tc.creates {
				require.NotNil(t, actions[id], id)
				require.Equal(t, tc.entity, actions[id].GetCreates(), id)
			}
		})
	}
}
