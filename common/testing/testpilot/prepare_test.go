package testpilot

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
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
func (nilDriverMap) Validate(context.Context, PreparedProgram) error  { panic("typed nil called") }
func (nilDriverMap) Open(context.Context, string, PreparedProgram) (Session, error) {
	panic("typed nil called")
}

type nilDriverSlice []int

func (nilDriverSlice) Identity(context.Context) (DriverIdentity, error) { panic("typed nil called") }
func (nilDriverSlice) Validate(context.Context, PreparedProgram) error  { panic("typed nil called") }
func (nilDriverSlice) Open(context.Context, string, PreparedProgram) (Session, error) {
	panic("typed nil called")
}

type nilDriverFunc func()

func (nilDriverFunc) Identity(context.Context) (DriverIdentity, error) { panic("typed nil called") }
func (nilDriverFunc) Validate(context.Context, PreparedProgram) error  { panic("typed nil called") }
func (nilDriverFunc) Open(context.Context, string, PreparedProgram) (Session, error) {
	panic("typed nil called")
}

type nilDriverChan chan int

func (nilDriverChan) Identity(context.Context) (DriverIdentity, error) { panic("typed nil called") }
func (nilDriverChan) Validate(context.Context, PreparedProgram) error  { panic("typed nil called") }
func (nilDriverChan) Open(context.Context, string, PreparedProgram) (Session, error) {
	panic("typed nil called")
}

type facadeDriver struct {
	identity    DriverIdentity
	session     Session
	validateErr error
	identities  int
	validates   int
	opens       int
}

func (d *facadeDriver) Identity(context.Context) (DriverIdentity, error) {
	d.identities++
	return d.identity, nil
}
func (d *facadeDriver) Validate(context.Context, PreparedProgram) error {
	d.validates++
	return d.validateErr
}
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

func TestPrepareFingerprintsCanonicalEnvironmentSnapshot(t *testing.T) {
	source, profile := facadeFixture(t)
	source.Program.Environment = []*testpilotspb.EnvironmentDefinition{{BindingId: "alpha"}, {BindingId: "beta"}}
	source.Program.Roles = []*testpilotspb.RoleDefinition{{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingId: "alpha"}, {RoleId: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingId: "alpha", ResourceBindingId: "beta"}}
	profile.Roles = []RolePolicy{{ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER}, {ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE}}
	profile.EnvironmentBindings = []EnvironmentBinding{{ID: "beta", Value: "v|2"}, {ID: "unused", Value: "x\x00y"}, {ID: "alpha", Value: "v:1"}}
	snapshot := profile.Snapshot()
	profile.EnvironmentBindings[0].Value = "mutated"
	require.Equal(t, "v|2", snapshot.EnvironmentBindings[0].Value)
	profile = snapshot

	fingerprint, err := profile.BindingFingerprint()
	require.NoError(t, err)
	require.Equal(t, "d7a2a0aaac12c4a0b44c3d94571a0caa6406228c09251a2da3795e211040c320", fingerprint)
	prepared, err := Prepare(source, profile)
	require.NoError(t, err)
	require.Equal(t, DriverIdentity{Profile: "proof", Catalog: profile.Catalog.Identity(), Bindings: fingerprint}, prepared.Identity())
	require.Equal(t, []PreparedRole{
		{ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingID: "alpha", Namespace: "v:1", ResourceBindingID: "beta", Resource: "v|2"},
		{ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingID: "alpha", Namespace: "v:1"},
	}, PreparedProgram{program: prepared.program}.Roles())
	roles := PreparedProgram{program: prepared.program}.Roles()
	roles[0].Namespace = "changed"
	require.Equal(t, "v:1", PreparedProgram{program: prepared.program}.Roles()[0].Namespace)

	reordered := profile.Snapshot()
	slices.Reverse(reordered.EnvironmentBindings)
	reorderedFingerprint, err := reordered.BindingFingerprint()
	require.NoError(t, err)
	require.Equal(t, fingerprint, reorderedFingerprint)
	require.Len(t, fingerprint, sha256.Size*2)
	require.Equal(t, strings.ToLower(fingerprint), fingerprint)

	left := profile.Snapshot()
	left.EnvironmentBindings = []EnvironmentBinding{{ID: "a", Value: "bc"}, {ID: "d", Value: "e"}}
	right := profile.Snapshot()
	right.EnvironmentBindings = []EnvironmentBinding{{ID: "a", Value: "b"}, {ID: "c", Value: "de"}}
	leftFingerprint, err := left.BindingFingerprint()
	require.NoError(t, err)
	rightFingerprint, err := right.BindingFingerprint()
	require.NoError(t, err)
	require.NotEqual(t, leftFingerprint, rightFingerprint)

	profile.EnvironmentBindings[1].Value = "changed"
	require.Equal(t, "beta", prepared.Snapshot().Program.Environment[1].BindingId)
	changed, err := profile.BindingFingerprint()
	require.NoError(t, err)
	require.NotEqual(t, fingerprint, changed)
}

func TestPrepareRejectsMalformedEnvironmentSnapshots(t *testing.T) {
	for name, bindings := range map[string][]EnvironmentBinding{
		"invalid id":          {{ID: "bad id", Value: "value"}},
		"oversized id":        {{ID: strings.Repeat("a", 257), Value: "value"}},
		"invalid id UTF-8":    {{ID: string([]byte{0xff}), Value: "value"}},
		"invalid value UTF-8": {{ID: "id", Value: string([]byte{0xff})}},
		"empty value":         {{ID: "id"}},
		"duplicate":           {{ID: "id", Value: "one"}, {ID: "id", Value: "two"}},
	} {
		t.Run(name, func(t *testing.T) {
			source, profile := facadeFixture(t)
			profile.EnvironmentBindings = bindings
			_, err := Prepare(source, profile)
			require.Error(t, err)
		})
	}

	source, profile := facadeFixture(t)
	profile.EnvironmentBindings = make([]EnvironmentBinding, 10001)
	_, err := Prepare(source, profile)
	require.Error(t, err)

	source, profile = facadeFixture(t)
	profile.ProgramLimits.MaxRequestBytes = 8
	profile.EnvironmentBindings = []EnvironmentBinding{{ID: "id", Value: strings.Repeat("x", 7)}}
	_, err = Prepare(source, profile)
	require.Error(t, err)
}

func TestConcurrentPreparationsOwnEnvironmentSnapshots(t *testing.T) {
	source, base := facadeFixture(t)
	source.Program.Environment = []*testpilotspb.EnvironmentDefinition{{BindingId: "namespace"}}
	source.Program.Roles = []*testpilotspb.RoleDefinition{{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingId: "namespace"}}
	base.Roles = []RolePolicy{{ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER}}

	for i := 0; i < 8; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			profile := base.Snapshot()
			value := fmt.Sprintf("namespace-%d", i)
			profile.EnvironmentBindings = []EnvironmentBinding{{ID: "namespace", Value: value}}
			prepared, err := Prepare(source, profile)
			require.NoError(t, err)
			fingerprint, err := profile.BindingFingerprint()
			require.NoError(t, err)
			profile.EnvironmentBindings[0].Value = "changed"
			require.Equal(t, "namespace", prepared.Snapshot().Program.Environment[0].BindingId)
			require.NotEmpty(t, fingerprint)
		})
	}
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
	require.Equal(t, 1, driver.validates)
}

func TestRunRejectsDriverIdentityMismatchBeforeValidateAndOpen(t *testing.T) {
	source, profile := facadeFixture(t)
	prepared, err := Prepare(source, profile)
	require.NoError(t, err)

	driver := &facadeDriver{identity: DriverIdentity{Profile: "other", Catalog: prepared.Identity().Catalog}}
	run, verdict, err := prepared.Run(t.Context(), driver)
	require.Error(t, err)
	require.Nil(t, run)
	require.Nil(t, verdict)
	require.Equal(t, 1, driver.identities)
	require.Zero(t, driver.validates)
	require.Zero(t, driver.opens)
}

func TestRunChecksContextBeforeDriverIdentity(t *testing.T) {
	source, profile := facadeFixture(t)
	prepared, err := Prepare(source, profile)
	require.NoError(t, err)
	driver := &facadeDriver{identity: prepared.Identity()}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	run, verdict, err := prepared.Run(ctx, driver)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, run)
	require.Nil(t, verdict)
	require.Zero(t, driver.identities)
	require.Zero(t, driver.validates)
	require.Zero(t, driver.opens)
}

type countingMonitorFactory struct {
	execution.MonitorFactory
	created int
	err     error
}

func (f *countingMonitorFactory) New(ctx context.Context, view execution.ProgramView) (execution.Monitor, error) {
	f.created++
	if f.err != nil {
		return nil, f.err
	}
	return f.MonitorFactory.New(ctx, view)
}

func TestRunRejectsBindingMismatchBeforeValidateAndOpen(t *testing.T) {
	source, profile := facadeFixture(t)
	prepared, err := Prepare(source, profile)
	require.NoError(t, err)
	driver := &facadeDriver{identity: prepared.Identity()}
	driver.identity.Bindings = "changed"

	run, verdict, err := prepared.Run(t.Context(), driver)
	require.Error(t, err)
	require.Nil(t, run)
	require.Nil(t, verdict)
	require.Zero(t, driver.validates)
	require.Zero(t, driver.opens)
}

func TestRunRejectsValidationBeforeMonitorAndOpen(t *testing.T) {
	source, profile := facadeFixture(t)
	prepared, err := Prepare(source, profile)
	require.NoError(t, err)
	factory := &countingMonitorFactory{MonitorFactory: prepared.factory}
	prepared.factory = factory
	validateErr := errors.New("invalid prepared Program")
	driver := &facadeDriver{identity: prepared.Identity(), validateErr: validateErr}

	run, verdict, err := prepared.Run(t.Context(), driver)
	require.ErrorIs(t, err, validateErr)
	require.Nil(t, run)
	require.Nil(t, verdict)
	require.Equal(t, 1, driver.validates)
	require.Zero(t, factory.created)
	require.Zero(t, driver.opens)
}

func TestRunCreatesMonitorBeforeOpen(t *testing.T) {
	source, profile := facadeFixture(t)
	prepared, err := Prepare(source, profile)
	require.NoError(t, err)
	monitorErr := errors.New("monitor unavailable")
	factory := &countingMonitorFactory{MonitorFactory: prepared.factory, err: monitorErr}
	prepared.factory = factory
	driver := &facadeDriver{identity: prepared.Identity()}

	run, verdict, err := prepared.Run(t.Context(), driver)
	require.ErrorIs(t, err, monitorErr)
	require.Nil(t, run)
	require.Nil(t, verdict)
	require.Equal(t, 1, driver.validates)
	require.Equal(t, 1, factory.created)
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

func facadeFixture(t testing.TB) (*testpilotspb.Case, ProfileSpec) {
	t.Helper()
	catalog, err := NewCatalog(&descriptorpb.FileDescriptorSet{})
	require.NoError(t, err)
	programLimits := &testpilotspb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 64, MaxAttempts: 32, MaxRunEvents: 256, MaxExpressionDepth: 16, MaxPathFanout: 128, MaxRequestBytes: 4096, MaxResponseBytes: 4096, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000}
	contractLimits := &testpilotspb.ContractLimits{MaxRules: 16, MaxStates: 32, MaxTransitions: 64, MaxExpressionDepth: 16, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000, MaxCaptures: 8, MaxCaptureBytes: 65536}
	source := &testpilotspb.Case{
		Version:  &testpilotspb.FormatVersion{Major: 1},
		CaseId:   "case",
		Program:  &testpilotspb.Program{ProgramId: "program", Entrypoints: []*testpilotspb.EntrypointDefinition{{EntrypointId: "controller", Activation: &testpilotspb.EntrypointDefinition_Controller{Controller: &testpilotspb.ControllerActivation{}}}}, Cleanup: &testpilotspb.CleanupDefinition{EntrypointId: "cleanup"}, Limits: programLimits},
		Contract: &testpilotspb.Contract{ContractId: "contract", Limits: proto.CloneOf(contractLimits), Rules: []*testpilotspb.ContractRuleDefinition{{RuleId: "safety", Kind: testpilotspb.CONTRACT_RULE_KIND_SAFETY, InitialStateId: "start", States: []*testpilotspb.ContractStateDefinition{{StateId: "start", Status: testpilotspb.CONTRACT_STATE_STATUS_NONTERMINAL}, {StateId: "good", Status: testpilotspb.CONTRACT_STATE_STATUS_SATISFIED}}, Transitions: []*testpilotspb.ContractTransitionDefinition{{TransitionId: "complete", SourceStateId: "start", TargetStateId: "good", EventFilter: &testpilotspb.RunEventFilter{Kinds: []testpilotspb.RunEventKind{testpilotspb.RUN_EVENT_KIND_RUN_CLOSED}}, Predicate: &testpilotspb.ContractExpression{Expression: &testpilotspb.ContractExpression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}}}}, SupportKind: testpilotspb.CONTRACT_SUPPORT_KIND_MATCHING_EVENT}}}}},
	}
	return source, ProfileSpec{Identity: "proof", Catalog: catalog, ProgramLimits: proto.CloneOf(programLimits), ContractLimits: contractLimits}
}
