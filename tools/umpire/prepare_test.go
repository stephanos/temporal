package umpire

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/execution"
	"go.temporal.io/server/tools/umpire/verification"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func preparationFixture(t *testing.T) (*umpirespb.Case, ProfileSpec) {
	t.Helper()
	catalog, err := NewCatalog(&descriptorpb.FileDescriptorSet{})
	require.NoError(t, err)
	limits := &umpirespb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 64, MaxAttempts: 32, MaxRunEvents: 256, MaxExpressionDepth: 16, MaxPathFanout: 128, MaxRequestBytes: 4096, MaxResponseBytes: 4096, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000}
	ceiling := &umpirespb.ContractLimits{MaxRules: 16, MaxStates: 32, MaxTransitions: 64, MaxExpressionDepth: 16, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000, MaxCaptures: 8, MaxCaptureBytes: 65536}
	source := &umpirespb.Case{Version: &umpirespb.FormatVersion{Major: 1}, CaseId: "case", Program: &umpirespb.Program{ProgramId: "program", Limits: limits, Entrypoints: []*umpirespb.Entrypoint{{EntrypointId: "controller", Context: umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER, Activation: &umpirespb.ActivationBinding{Binding: &umpirespb.ActivationBinding_Controller{Controller: &umpirespb.ControllerActivation{}}}}}, Cleanup: &umpirespb.CleanupGraph{EntrypointId: "cleanup", Context: umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER}}, Contract: &umpirespb.Contract{ContractId: "contract", Limits: proto.CloneOf(ceiling), Rules: []*umpirespb.ContractRule{{RuleId: "safety", Kind: umpirespb.CONTRACT_RULE_KIND_SAFETY, InitialState: "start", States: []*umpirespb.ContractState{{StateId: "start", Terminal: umpirespb.CONTRACT_TERMINAL_STATE_NONTERMINAL}, {StateId: "good", Terminal: umpirespb.CONTRACT_TERMINAL_STATE_SATISFIED}}, Transitions: []*umpirespb.ContractTransition{{TransitionId: "success", SourceState: "start", TargetState: "good", Predicate: &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Literal{Literal: &umpirespb.Value{Value: &umpirespb.Value_BoolValue{BoolValue: true}}}}, EventKinds: &umpirespb.RunEventKinds{Kinds: []umpirespb.RunEventKind{umpirespb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED}}, Support: umpirespb.CONTRACT_SUPPORT_MATCHING_EVENT}}}}}}
	return source, ProfileSpec{Identity: "host", Catalog: catalog, ProgramLimits: proto.CloneOf(limits), ContractLimits: ceiling}
}

func TestPrepareCaseComposesProductionAdmission(t *testing.T) {
	source, profile := preparationFixture(t)
	prepared, err := PrepareCase(source, profile)
	require.NoError(t, err)
	require.IsType(t, &verification.PreparedContract{}, prepared.factory)
	expected := prepared.program.Snapshot()
	source.Program.ProgramId = "changed"
	source.Contract.Rules[0].InitialState = "missing"
	profile.Identity = "changed"
	profile.ProgramLimits.MaxNodes = 1
	profile.ContractLimits.MaxRules = 1
	require.True(t, proto.Equal(expected, prepared.program.Snapshot()))
	monitor, err := execution.NewMonitor(context.Background(), prepared.factory, prepared.program.View())
	require.NoError(t, err)
	require.IsType(t, &verification.Evaluator{}, monitor)
	require.Equal(t, HostIdentity{Profile: "host", Catalog: profile.Catalog.Identity()}, prepared.Identity())
	_, err = PrepareCase(source, profile)
	require.Error(t, err)
}

type nilProfileMap map[string]int

func (nilProfileMap) Snapshot() ProfileSpec { panic("typed nil called") }

type nilProfileSlice []int

func (nilProfileSlice) Snapshot() ProfileSpec { panic("typed nil called") }

type nilProfileFunc func()

func (nilProfileFunc) Snapshot() ProfileSpec { panic("typed nil called") }

type nilProfileChan chan int

func (nilProfileChan) Snapshot() ProfileSpec { panic("typed nil called") }

type testHost struct {
	identity HostIdentity
	opens    int
	err      error
}

func (h *testHost) Identity(context.Context) (HostIdentity, error) { return h.identity, h.err }
func (h *testHost) Open(context.Context, string, PreparedProgram) (Session, error) {
	h.opens++
	return nil, nil
}

type nilHostMap map[string]int

func (nilHostMap) Identity(context.Context) (HostIdentity, error) { panic("typed nil called") }
func (nilHostMap) Open(context.Context, string, PreparedProgram) (Session, error) {
	panic("typed nil called")
}

type nilHostSlice []int

func (nilHostSlice) Identity(context.Context) (HostIdentity, error) { panic("typed nil called") }
func (nilHostSlice) Open(context.Context, string, PreparedProgram) (Session, error) {
	panic("typed nil called")
}

type nilHostFunc func()

func (nilHostFunc) Identity(context.Context) (HostIdentity, error) { panic("typed nil called") }
func (nilHostFunc) Open(context.Context, string, PreparedProgram) (Session, error) {
	panic("typed nil called")
}

type nilHostChan chan int

func (nilHostChan) Identity(context.Context) (HostIdentity, error) { panic("typed nil called") }
func (nilHostChan) Open(context.Context, string, PreparedProgram) (Session, error) {
	panic("typed nil called")
}

type factoryFunc func(context.Context, execution.ProgramView) (execution.Monitor, error)

func (f factoryFunc) New(ctx context.Context, v execution.ProgramView) (execution.Monitor, error) {
	return f(ctx, v)
}

type factoryMap map[string]int

func (factoryMap) New(context.Context, execution.ProgramView) (execution.Monitor, error) {
	panic("typed nil called")
}

type factorySlice []int

func (factorySlice) New(context.Context, execution.ProgramView) (execution.Monitor, error) {
	panic("typed nil called")
}

type factoryChan chan int

func (factoryChan) New(context.Context, execution.ProgramView) (execution.Monitor, error) {
	panic("typed nil called")
}

func TestPreparationAndPreflightRejectWithoutEffects(t *testing.T) {
	source, profile := preparationFixture(t)
	for _, profile := range []Profile{nil, (*ProfileSpec)(nil), nilProfileMap(nil), nilProfileSlice(nil), nilProfileFunc(nil), nilProfileChan(nil)} {
		prepared, err := PrepareCase(source, profile)
		require.Error(t, err)
		require.Nil(t, prepared)
	}
	prepared, err := PrepareCase(source, profile)
	require.NoError(t, err)
	for _, host := range []Host{nil, (*testHost)(nil), nilHostMap(nil), nilHostSlice(nil), nilHostFunc(nil), nilHostChan(nil), &testHost{}, &testHost{err: context.Canceled}, &testHost{identity: HostIdentity{Profile: "host", Catalog: "changed"}}, &testHost{identity: HostIdentity{Profile: "changed", Catalog: profile.Catalog.Identity()}}} {
		driver, monitor, err := prepared.preflight(t.Context(), host)
		require.Error(t, err)
		require.Nil(t, driver)
		require.Nil(t, monitor)
		if h, ok := host.(*testHost); ok && h != nil {
			require.Zero(t, h.opens)
		}
	}
	host := &testHost{identity: prepared.Identity()}
	for _, factory := range []execution.MonitorFactory{nil, (*verification.PreparedContract)(nil), factoryFunc(nil), factoryMap(nil), factorySlice(nil), factoryChan(nil), factoryFunc(func(context.Context, execution.ProgramView) (execution.Monitor, error) { return nil, context.Canceled }), factoryFunc(func(context.Context, execution.ProgramView) (execution.Monitor, error) {
		return (*verification.Evaluator)(nil), nil
	})} {
		candidate := *prepared
		candidate.factory = factory
		driver, monitor, err := candidate.preflight(t.Context(), host)
		require.Error(t, err)
		require.Nil(t, driver)
		require.Nil(t, monitor)
		require.Zero(t, host.opens)
	}
	driver, monitor, err := prepared.preflight(t.Context(), host)
	require.NoError(t, err)
	require.NotNil(t, driver)
	require.IsType(t, &verification.Evaluator{}, monitor)
	require.Zero(t, host.opens)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, _, err = prepared.preflight(ctx, host)
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, host.opens)
}

func TestIndependentPreparedCasesAndSourceMutation(t *testing.T) {
	source, profile := preparationFixture(t)
	for i := 0; i < 8; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			local := proto.CloneOf(source)
			prepared, err := PrepareCase(local, profile)
			require.NoError(t, err)
			local.Contract.Rules[0].Transitions[0].TargetState = "missing"
			host := &testHost{identity: prepared.Identity()}
			_, monitor, err := prepared.preflight(t.Context(), host)
			require.NoError(t, err)
			events := []*umpirespb.RunEvent{
				{Sequence: 1, SourceId: "open", Kind: umpirespb.RUN_EVENT_KIND_RUN_OPENED},
				{Sequence: 2, SourceId: "complete", Kind: umpirespb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED},
				{Sequence: 3, SourceId: "close", Kind: umpirespb.RUN_EVENT_KIND_RUN_CLOSED},
			}
			for _, event := range events {
				decision, err := monitor.Observe(t.Context(), event)
				require.NoError(t, err)
				require.Equal(t, execution.Continue, decision)
			}
			verdict, err := monitor.Close(t.Context(), &umpirespb.Run{RunId: fmt.Sprint(i), CaseId: "case", ProgramId: "program", Events: events, Disposition: umpirespb.RUN_DISPOSITION_COMPLETED})
			require.NoError(t, err)
			require.Equal(t, umpirespb.VERDICT_KIND_SATISFIED, verdict.Kind)
		})
	}
}

func TestPrepareFreezesCatalogPolicyAndProvenance(t *testing.T) {
	source, profile := preparationFixture(t)
	descriptors := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{Name: proto.String("example.proto"), Package: proto.String("example"), Syntax: proto.String("proto3"), MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Empty")}}, Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("Service"), Method: []*descriptorpb.MethodDescriptorProto{{Name: proto.String("Call"), InputType: proto.String(".example.Empty"), OutputType: proto.String(".example.Empty")}}}}}}}
	catalog, err := NewCatalog(descriptors)
	require.NoError(t, err)
	profile.Catalog = catalog
	profile.Roles = []RolePolicy{{ID: "endpoint", Kind: umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT, Methods: []string{"/example.Service/Call"}}}
	profile.Capabilities = []Capability{InvokeRPC}
	source.Metadata = &umpirespb.CaseMetadata{ProducerId: "producer", KnownGaps: []*umpirespb.CaseKnownGap{{Kind: umpirespb.CASE_KNOWN_GAP_KIND_INPUT, Code: "gap"}}}
	prepared, err := PrepareCase(source, profile)
	require.NoError(t, err)
	expected := proto.CloneOf(source)
	identity := prepared.Identity()
	descriptors.File[0].Service[0].Method[0].Name = proto.String("Changed")
	source.CaseId = "changed"
	source.Version.Minor = 1
	source.Metadata.ProducerId = "changed"
	source.Metadata.KnownGaps[0].Code = "changed"
	profile.Roles[0].Methods[0] = "/example.Service/Missing"
	profile.Capabilities[0] = Finish
	profile.ProgramLimits.MaxNodes = 1
	profile.ContractLimits.MaxRules = 1
	require.True(t, proto.Equal(expected, prepared.Snapshot()))
	snapshot := prepared.Snapshot()
	snapshot.Metadata.KnownGaps[0].Code = "changed-again"
	require.True(t, proto.Equal(expected, prepared.Snapshot()))
	require.Equal(t, identity, prepared.Identity())
	_, _, err = prepared.preflight(t.Context(), &testHost{identity: identity})
	require.NoError(t, err)
	_, err = PrepareCase(expected, profile)
	require.Error(t, err)
}

type openingHost struct {
	identity HostIdentity
	open     func(context.Context, string, PreparedProgram) (Session, error)
}

func (h openingHost) Identity(context.Context) (HostIdentity, error) { return h.identity, nil }
func (h openingHost) Open(ctx context.Context, id string, program PreparedProgram) (Session, error) {
	return h.open(ctx, id, program)
}

type sessionStub struct{ Session }

func TestDriverTranslatesOnlyAdmittedProgram(t *testing.T) {
	source, profile := preparationFixture(t)
	prepared, err := PrepareCase(source, profile)
	require.NoError(t, err)
	owned := &sessionStub{}
	host := openingHost{identity: prepared.Identity(), open: func(ctx context.Context, id string, program PreparedProgram) (Session, error) {
		require.Same(t, t.Context(), ctx)
		require.Equal(t, "run", id)
		snapshot := program.Snapshot()
		snapshot.ProgramId = "changed"
		entries := program.Entrypoints()
		entries[0].Activation().Binding = nil
		require.Equal(t, "program", program.Snapshot().ProgramId)
		require.NotNil(t, program.Entrypoints()[0].Activation().Binding)
		return owned, nil
	}}
	driver, _, err := prepared.preflight(t.Context(), host)
	require.NoError(t, err)
	identity, err := driver.Identity(t.Context())
	require.NoError(t, err)
	require.Equal(t, prepared.Identity(), identity)
	session, err := driver.Open(t.Context(), "run", prepared.program)
	require.NoError(t, err)
	require.Same(t, owned, session)
	host.open = func(context.Context, string, PreparedProgram) (Session, error) { return (*sessionStub)(nil), nil }
	driver, _, err = prepared.preflight(t.Context(), host)
	require.NoError(t, err)
	_, err = driver.Open(t.Context(), "run", prepared.program)
	require.Error(t, err)
}

func TestPrepareCaseRejectsIncompleteAdmission(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*umpirespb.Case, *ProfileSpec)
	}{
		{"Program authorization", func(c *umpirespb.Case, _ *ProfileSpec) {
			c.Program.Roles = []*umpirespb.ProgramRole{{RoleId: "unauthorized", Kind: umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT}}
		}},
		{"Contract", func(c *umpirespb.Case, _ *ProfileSpec) { c.Contract.Rules[0].InitialState = "missing" }},
		{"missing catalog", func(_ *umpirespb.Case, p *ProfileSpec) { p.Catalog = nil }},
		{"zero catalog", func(_ *umpirespb.Case, p *ProfileSpec) { p.Catalog = &Catalog{} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			source, profile := preparationFixture(t)
			test.mutate(source, &profile)
			prepared, err := PrepareCase(source, profile)
			require.Error(t, err)
			require.Nil(t, prepared)
		})
	}
}
