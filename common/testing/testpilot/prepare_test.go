package testpilot

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

type nilProfileMap map[string]int

func (nilProfileMap) Snapshot() ProfileSpec { panic("typed nil called") }

type nilProfileSlice []int

func (nilProfileSlice) Snapshot() ProfileSpec { panic("typed nil called") }

type nilProfileFunc func()

func (nilProfileFunc) Snapshot() ProfileSpec { panic("typed nil called") }

type nilProfileChan chan int

func (nilProfileChan) Snapshot() ProfileSpec { panic("typed nil called") }

type nilDriverMap map[string]int

func (nilDriverMap) Identity(context.Context) (DriverIdentity, error) { panic("typed nil called") }
func (nilDriverMap) Open(context.Context, string, PreparedProgram) (Session, error) {
	panic("typed nil called")
}

type nilDriverSlice []int

func (nilDriverSlice) Identity(context.Context) (DriverIdentity, error) { panic("typed nil called") }
func (nilDriverSlice) Open(context.Context, string, PreparedProgram) (Session, error) {
	panic("typed nil called")
}

type nilDriverFunc func()

func (nilDriverFunc) Identity(context.Context) (DriverIdentity, error) { panic("typed nil called") }
func (nilDriverFunc) Open(context.Context, string, PreparedProgram) (Session, error) {
	panic("typed nil called")
}

type nilDriverChan chan int

func (nilDriverChan) Identity(context.Context) (DriverIdentity, error) { panic("typed nil called") }
func (nilDriverChan) Open(context.Context, string, PreparedProgram) (Session, error) {
	panic("typed nil called")
}

type facadeDriver struct {
	identity DriverIdentity
	session  Session
	opens    int
}

func (d *facadeDriver) Identity(context.Context) (DriverIdentity, error) { return d.identity, nil }
func (d *facadeDriver) Open(context.Context, string, PreparedProgram) (Session, error) {
	d.opens++
	return d.session, nil
}

func TestPrepareOwnsMutableInputs(t *testing.T) {
	source, profile := facadeFixture(t)
	expected := proto.CloneOf(source)
	prepared, err := Prepare(source, profile)
	require.NoError(t, err)

	source.CaseId = "changed"
	source.Program.ProgramId = "changed"
	source.Contract.Rules[0].InitialStateId = "missing"
	profile.ProgramLimits.MaxNodes = 1
	profile.ContractLimits.MaxRules = 1
	require.True(t, proto.Equal(expected, prepared.Snapshot()))
	require.Equal(t, DriverIdentity{Profile: "proof", Catalog: profile.Catalog.Identity()}, prepared.Identity())
}

func TestPrepareAndRunRejectTypedNilInterfacesBeforeEffects(t *testing.T) {
	source, profile := facadeFixture(t)
	for _, candidate := range []Profile{nil, (*ProfileSpec)(nil), nilProfileMap(nil), nilProfileSlice(nil), nilProfileFunc(nil), nilProfileChan(nil)} {
		prepared, err := Prepare(source, candidate)
		require.Error(t, err)
		require.Nil(t, prepared)
	}

	prepared, err := Prepare(source, profile)
	require.NoError(t, err)
	for _, candidate := range []Driver{nil, (*facadeDriver)(nil), nilDriverMap(nil), nilDriverSlice(nil), nilDriverFunc(nil), nilDriverChan(nil)} {
		run, verdict, err := prepared.Run(t.Context(), candidate)
		require.Error(t, err)
		require.Nil(t, run)
		require.Nil(t, verdict)
	}

	driver := &facadeDriver{identity: prepared.Identity(), session: (*facadeSession)(nil)}
	run, verdict, err := prepared.Run(t.Context(), driver)
	require.Error(t, err)
	require.Nil(t, run)
	require.Nil(t, verdict)
	require.Equal(t, 1, driver.opens)
}

func TestRunRejectsDriverIdentityMismatchBeforeOpen(t *testing.T) {
	source, profile := facadeFixture(t)
	prepared, err := Prepare(source, profile)
	require.NoError(t, err)

	driver := &facadeDriver{identity: DriverIdentity{Profile: "other", Catalog: prepared.Identity().Catalog}}
	run, verdict, err := prepared.Run(t.Context(), driver)
	require.Error(t, err)
	require.Nil(t, run)
	require.Nil(t, verdict)
	require.Zero(t, driver.opens)
}

type facadeSession struct{ Session }

type reservationEffect struct{ ReservationHandle }

type quarantineSession struct {
	Session
	handle EffectHandle
}

func (s *quarantineSession) Quarantine(_ context.Context, handle EffectHandle) error {
	s.handle = handle
	return nil
}

func TestSessionAdapterReturnsReservationEffectsToTheirDriver(t *testing.T) {
	handle := &reservationEffect{}
	session := &quarantineSession{}
	err := (sessionAdapter{session: session}).Quarantine(t.Context(), reservationAdapter{handle: handle})
	require.NoError(t, err)
	require.Same(t, handle, session.handle)
	var _ execution.EffectHandle = reservationAdapter{}
}

func facadeFixture(t testing.TB) (*testpilotpb.Case, ProfileSpec) {
	t.Helper()
	catalog, err := NewCatalog(&descriptorpb.FileDescriptorSet{})
	require.NoError(t, err)
	programLimits := &testpilotpb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 64, MaxAttempts: 32, MaxRunEvents: 256, MaxExpressionDepth: 16, MaxPathFanout: 128, MaxRequestBytes: 4096, MaxResponseBytes: 4096, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000}
	contractLimits := &testpilotpb.ContractLimits{MaxRules: 16, MaxStates: 32, MaxTransitions: 64, MaxExpressionDepth: 16, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000, MaxCaptures: 8, MaxCaptureBytes: 65536}
	source := &testpilotpb.Case{
		Version:  &testpilotpb.FormatVersion{Major: 1},
		CaseId:   "case",
		Program:  &testpilotpb.Program{ProgramId: "program", Entrypoints: []*testpilotpb.EntrypointDefinition{{EntrypointId: "controller", Activation: &testpilotpb.EntrypointDefinition_Controller{Controller: &testpilotpb.ControllerActivation{}}}}, Cleanup: &testpilotpb.CleanupDefinition{EntrypointId: "cleanup"}, Limits: programLimits},
		Contract: &testpilotpb.Contract{ContractId: "contract", Limits: proto.CloneOf(contractLimits), Rules: []*testpilotpb.ContractRuleDefinition{{RuleId: "safety", Kind: testpilotpb.CONTRACT_RULE_KIND_SAFETY, InitialStateId: "start", States: []*testpilotpb.ContractStateDefinition{{StateId: "start", Status: testpilotpb.CONTRACT_STATE_STATUS_NONTERMINAL}, {StateId: "good", Status: testpilotpb.CONTRACT_STATE_STATUS_SATISFIED}}, Transitions: []*testpilotpb.ContractTransitionDefinition{{TransitionId: "complete", SourceStateId: "start", TargetStateId: "good", EventFilter: &testpilotpb.RunEventFilter{Kinds: []testpilotpb.RunEventKind{testpilotpb.RUN_EVENT_KIND_RUN_CLOSED}}, Predicate: &testpilotpb.ContractExpression{Expression: &testpilotpb.ContractExpression_Literal{Literal: &testpilotpb.Value{Value: &testpilotpb.Value_BoolValue{BoolValue: true}}}}, SupportKind: testpilotpb.CONTRACT_SUPPORT_KIND_MATCHING_EVENT}}}}},
	}
	return source, ProfileSpec{Identity: "proof", Catalog: catalog, ProgramLimits: proto.CloneOf(programLimits), ContractLimits: contractLimits}
}
