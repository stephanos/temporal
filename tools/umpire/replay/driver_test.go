package replay

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// corpusRoot holds the Case Runtime's conformance corpus; its `violated` class is one instruction
// whose completion violates a safety rule, which a scripted Driver reproduces offline.
const corpusRoot = "../../../common/testing/testpilot/testdata/case-runtime-conformance"

func loadCorpusCase(t testing.TB, class string) []byte {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join(corpusRoot, class, "case.json"))
	require.NoError(t, err)
	return encoded
}

// preparer prepares a Case under the deployment names the tests bind to, as binding.Prepare does,
// with no deployment behind them.
func preparer(t testing.TB) Preparer {
	t.Helper()
	return preparerIn(t, testpilotdriver.Environment{Namespace: "replay", TaskQueue: "replay-queue", NexusEndpoint: "replay-endpoint"})
}

// preparerIn prepares under the given names; the Profile identity is the preparer's argument.
func preparerIn(t testing.TB, environment testpilotdriver.Environment) Preparer {
	t.Helper()
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	return func(identity string, source *testpilotspb.Case) (*testpilot.PreparedCase, error) {
		environment.Identity = identity
		profile, err := testpilotdriver.DeriveProfile(source, catalog, environment)
		if err != nil {
			return nil, err
		}
		return testpilot.Prepare(source, profile)
	}
}

// scriptedDriver answers every RPC with success and no worker, enough for the corpus Cases.
type scriptedDriver struct{ identity testpilot.DriverIdentity }

func (d *scriptedDriver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return d.identity, nil
}
func (*scriptedDriver) Validate(context.Context, testpilot.PreparedProgram) error { return nil }
func (*scriptedDriver) Open(context.Context, string, testpilot.PreparedProgram) (testpilot.Session, error) {
	return scriptedSession{}, nil
}

type scriptedSession struct{}

func (scriptedSession) Reserve(context.Context, testpilot.ReservationRequest) ([]testpilot.ReservationHandle, error) {
	return nil, errors.New("scripted sessions reserve nothing")
}
func (scriptedSession) InvokeRPC(_ context.Context, _ testpilot.Coordinate, _ string, method protoreflect.MethodDescriptor, _ proto.Message) (testpilot.EffectHandle, error) {
	return scriptedEffect{result: testpilot.EffectResult{
		Outcome:  &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED},
		Response: dynamicpb.NewMessage(method.Output()),
	}}, nil
}
func (s scriptedSession) PollRPC(ctx context.Context, coordinate testpilot.Coordinate, role string, method protoreflect.MethodDescriptor, request proto.Message, _ time.Duration, _ testpilot.PollPredicate) (testpilot.EffectHandle, error) {
	return s.InvokeRPC(ctx, coordinate, role, method, request)
}
func (scriptedSession) InvokeCapability(context.Context, testpilot.Coordinate, testpilot.OpaqueCapability, proto.Message) (testpilot.EffectHandle, error) {
	return nil, errors.New("scripted sessions invoke no capability")
}
func (scriptedSession) InjectFault(context.Context, testpilot.Coordinate, string, testpilotspb.FaultKind) (testpilot.EffectHandle, error) {
	return scriptedEffect{result: testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}}, nil
}
func (scriptedSession) Bridge(context.Context) (testpilot.CapabilityBridge, error) {
	return nil, errors.New("scripted sessions bridge nothing")
}
func (scriptedSession) Quarantine(context.Context, testpilot.EffectHandle) error {
	return errors.New("scripted effects complete synchronously")
}
func (scriptedSession) Close(context.Context) error { return nil }
func (scriptedSession) Diagnose(context.Context, string, *testpilotspb.RunDiagnostic) error {
	return nil
}

type scriptedEffect struct{ result testpilot.EffectResult }

func (e scriptedEffect) Wait(context.Context) (testpilot.EffectResult, error) { return e.result, nil }
func (scriptedEffect) Cancel(context.Context) error                           { return nil }
func (scriptedEffect) Drain(context.Context) error                            { return nil }

// recordedRunOf runs the Case once through the scripted Driver under the given Profile name and
// records what closed, the way umpire-run --record does.
func recordedRunOf(t testing.TB, prepare Preparer, identity string, caseBytes []byte) (testpilot.DriverIdentity, *testpilotspb.Run, []byte) {
	t.Helper()
	source, err := testpilot.DecodeCaseProtoJSON(caseBytes)
	require.NoError(t, err)
	prepared, err := prepare(identity, source)
	require.NoError(t, err)
	run, verdict, err := prepared.Run(t.Context(), &scriptedDriver{identity: prepared.Identity()})
	require.NoError(t, err)
	require.True(t, proto.Equal(verdict, run.GetVerdict()))
	recorded, err := EncodeRecordedRun(prepared.Identity(), run)
	require.NoError(t, err)
	return prepared.Identity(), run, recorded
}
