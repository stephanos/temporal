package lower

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"google.golang.org/protobuf/types/known/emptypb"
)

// A reset settlement lowers to the Program's declaration of the held reservation, the typed reset
// and the fresh reservation after it; a fresh group that is not the held one's next is refused.
func TestActivityResetSettlementLowersTheDeclaredFreshReservation(t *testing.T) {
	declared := &umpirespb.ActivityResetSettlement{
		Carrier: "start", Activity: "reset-attempts", Attempt: 1, Pending: "reset-pending", Held: "await-reset-held",
		ResetRequest: "reset-held-activity", Timer: &umpirespb.ActionClass{Action: "heartbeat"}, FreshAttempt: 2,
		Settlement: "await-reset-completed", Cleanup: &umpirespb.Command{Id: "external-cleanup"},
	}
	a := &adapter{r: &umpirespb.Realization{Cleanup: "cleanup", ResetSettlements: []*umpirespb.ActivityResetSettlement{declared},
		Scripts: []*umpirespb.Script{{Id: "controller", Activation: &umpirespb.Script_Controller{Controller: &emptypb.Empty{}}}}}}
	got, err := a.resetSettlement(declared)
	require.NoError(t, err)
	ref := func(id string) *testpilotspb.InstructionReference {
		return &testpilotspb.InstructionReference{EntrypointId: "controller", InstructionId: id}
	}
	protorequire.ProtoEqual(t, &testpilotspb.ActivityResetSettlement{
		Carrier: ref("start"), ActivityEntrypointId: "reset-attempts", ReservationOrdinal: 0, PendingSlotId: "reset-pending",
		Held: ref("await-reset-held"), ResetRequest: ref("reset-held-activity"), FreshReservationOrdinal: 1,
		Settlement: ref("await-reset-completed"), Cleanup: &testpilotspb.InstructionReference{EntrypointId: "cleanup", InstructionId: "external-cleanup"},
	}, got)
	for _, fresh := range []int64{1, 3} {
		declared.FreshAttempt = fresh
		_, err := a.resetSettlement(declared)
		require.ErrorContains(t, err, "the fresh attempt is the held one's next")
	}
}
