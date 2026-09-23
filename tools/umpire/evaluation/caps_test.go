package evaluation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/tools/umpire/recordedrun"
	"google.golang.org/protobuf/proto"
)

// paddingFor is the producer version that makes the control Case exactly size canonical bytes.
func (c control) paddingFor(t *testing.T, size int) string {
	t.Helper()
	source := proto.CloneOf(c.source)
	source.Provenance.ProducerVersion = "x"
	base := len(compactCase(t, source))
	require.Less(t, base, size)
	return strings.Repeat("x", 1+size-base)
}

// Every admission cap admits a subject exactly at it and rejects one a unit over it as oversized:
// the Case's bytes, the recorded Run's bytes and the Run's events.
func TestAdmissionCapsAtNAndNPlusOne(t *testing.T) {
	c := loadControl(t)
	admit := func(caseBytes, recorded []byte) error {
		_, err := Admit(caseBytes, recorded, controlCatalog)
		return err
	}
	oversized := func(t *testing.T, err error) {
		t.Helper()
		rejection, ok := IsRejection(err)
		require.True(t, ok, "not a rejection: %v", err)
		require.Equal(t, ReasonOversized, rejection.Reason, rejection.Detail)
	}

	t.Run("Case bytes", func(t *testing.T) {
		for _, probe := range []struct {
			size     int
			admitted bool
		}{{MaxCaseBytes, true}, {MaxCaseBytes + 1, false}} {
			padding := c.paddingFor(t, probe.size)
			caseBytes, recorded := c.pair(t, func(edited *testpilotspb.Case) {
				edited.Provenance.ProducerVersion = padding
			}, nil)
			require.Len(t, caseBytes, probe.size)
			if probe.admitted {
				require.NoError(t, admit(caseBytes, recorded))
			} else {
				oversized(t, admit(caseBytes, recorded))
			}
		}
	})

	t.Run("recorded Run bytes", func(t *testing.T) {
		caseIdentity, err := recordedrun.CaseIdentity(c.caseBytes)
		require.NoError(t, err)
		record := func(runID string) []byte {
			run := proto.CloneOf(c.decoded.Run)
			run.RunId = runID
			encoded, err := recordedrun.Encode(caseIdentity, c.decoded.Driver, run)
			require.NoError(t, err)
			return encoded
		}
		base := len(record("r"))
		for _, probe := range []struct {
			size     int
			admitted bool
		}{{MaxRunBytes, true}, {MaxRunBytes + 1, false}} {
			recorded := record(strings.Repeat("r", 1+probe.size-base))
			require.Len(t, recorded, probe.size)
			if probe.admitted {
				require.NoError(t, admit(c.caseBytes, recorded))
			} else {
				oversized(t, admit(c.caseBytes, recorded))
			}
		}
	})

	t.Run("Run events", func(t *testing.T) {
		for _, probe := range []struct {
			events   int
			admitted bool
		}{{MaxRunEvents, true}, {MaxRunEvents + 1, false}} {
			caseBytes, recorded := c.pair(t, nil, func(run *testpilotspb.Run) {
				for len(run.Events) < probe.events {
					run.Events = append(run.Events, &testpilotspb.RunEvent{Sequence: int64(len(run.Events) + 1)})
				}
			})
			if probe.admitted {
				require.NoError(t, admit(caseBytes, recorded))
			} else {
				oversized(t, admit(caseBytes, recorded))
			}
		}
	})
}
