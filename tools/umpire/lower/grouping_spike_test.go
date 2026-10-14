package lower

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
)

func groupingFixture(t *testing.T, form string) *umpirespb.Model {
	t.Helper()
	return loaded(t, form)
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
		{"nexus-workflow", "operation", []string{"temporal.features.nexus.handler.reply", "temporal.features.nexus.handler.complete", "temporal.features.nexus.network.fault", "temporal.features.nexus.client.terminate"}, []string{"temporal.features.nexus.workflow.caller.schedule"}},
		{"nexus-standalone", "operation", []string{"temporal.features.nexus.handler.reply", "temporal.features.nexus.handler.complete", "temporal.features.nexus.network.fault", "temporal.features.nexus.client.terminate", "temporal.features.nexus.standalone.client.requestCancel"}, []string{"temporal.features.nexus.standalone.client.start"}},
		{"activity-standalone", "activity", []string{
			"temporal.features.activity.worker.poll",
			"temporal.features.activity.worker.respondCanceled",
			"temporal.features.activity.worker.respondCompleted",
			"temporal.features.activity.worker.respondFailed",
			"temporal.features.activity.standalone.client.pause",
			"temporal.features.activity.standalone.client.requestCancel",
			"temporal.features.activity.standalone.client.terminate",
			"temporal.features.activity.standalone.client.unpause",
		}, []string{"temporal.features.activity.standalone.client.start"}},
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
