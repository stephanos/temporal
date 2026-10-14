package lower

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
)

// The message a history kind of evidence is read as is what the realization's own reads that lift
// history evidence read: no history event is assumed. Without such a read a history kind is refused
// where it is declared, and every read that lifts history must read the same message.
func TestAHistoryKindIsReadAsWhatItsLiftingReadReads(t *testing.T) {
	m := loaded(t, "nexus-workflow")
	r := m.GetRealizations()[0]
	var history *umpirespb.Evidence
	for _, e := range r.GetEvidence() {
		if e.GetHistory() != "" {
			history = e
			break
		}
	}
	require.NotNil(t, history)
	element, err := EvidenceElement(r, history)
	require.NoError(t, err)
	require.Equal(t, "temporal.api.history.v1.HistoryEvent", string(element.FullName()))

	lifting := 0
	for _, s := range r.GetScripts() {
		for _, item := range s.GetItems() {
			for _, read := range item.GetCommand().GetRpc().GetReads() {
				var kept []*umpirespb.Target
				for _, target := range read.GetTargets() {
					if target.GetLift() == "" {
						kept = append(kept, target)
					} else {
						lifting++
					}
				}
				read.Targets = kept
			}
		}
	}
	require.Positive(t, lifting)
	_, err = EvidenceElement(r, history)
	require.ErrorContains(t, err, "evidence "+history.GetId()+" is read from history, and no read of realization "+r.GetName()+" lifts history evidence")
}
