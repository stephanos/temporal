package testpilot_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func requirePreparationError(t testing.TB, err error, category testpilot.PreparationErrorCategory, path, detail, message string) {
	t.Helper()
	require.Error(t, err)
	var diagnostic *testpilot.PreparationError
	require.ErrorAs(t, err, &diagnostic)
	require.Equal(t, testpilot.PreparationError{Category: category, Path: path, Detail: detail}, testpilot.PreparationError{Category: diagnostic.Category, Path: diagnostic.Path, Detail: diagnostic.Detail})
	require.LessOrEqual(t, len(diagnostic.Path), 256)
	require.EqualError(t, err, message)
	var wrapped *testpilot.PreparationError
	require.ErrorAs(t, fmt.Errorf("admission: %w", err), &wrapped)
	require.Same(t, diagnostic, wrapped)
}

func TestPreparationErrorCatalog(t *testing.T) {
	for _, tc := range []struct {
		name         string
		source       *descriptorpb.FileDescriptorSet
		category     testpilot.PreparationErrorCategory
		path, detail string
	}{
		{"nil", nil, testpilot.PreparationMalformed, "catalog", "descriptor set is required"},
		{"graph", &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{Name: proto.String("bad.proto"), Dependency: []string{"missing.proto"}}}}, testpilot.PreparationMalformed, "catalog", "invalid descriptor graph"},
		{"unknown", func() *descriptorpb.FileDescriptorSet {
			s := &descriptorpb.FileDescriptorSet{}
			s.ProtoReflect().SetUnknown([]byte{0x98, 0x06, 1})
			return s
		}(), testpilot.PreparationUnknown, "catalog", "unknown protobuf fields"},
		{"intrinsic conflict", &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{Name: proto.String("conflict.proto"), Package: proto.String("temporal.server.api.testpilot.v1"), EnumType: []*descriptorpb.EnumDescriptorProto{{Name: proto.String("InstructionOutcomeStatus"), Value: []*descriptorpb.EnumValueDescriptorProto{{Name: proto.String("DIFFERENT"), Number: proto.Int32(0)}}}}}}}, testpilot.PreparationTypeMismatch, "catalog", "conflicting intrinsic enum definition"},
		{"collection limit", &descriptorpb.FileDescriptorSet{File: make([]*descriptorpb.FileDescriptorProto, 10001)}, testpilot.PreparationLimitExceeded, "catalog.file", "repeated collection ceiling exceeded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			catalog, err := testpilot.NewCatalog(tc.source)
			require.Nil(t, catalog)
			requirePreparationError(t, err, tc.category, tc.path, tc.detail, fmt.Sprintf("%s at %s: %s", tc.category, tc.path, tc.detail))
		})
	}
}

func TestPreparationErrorCase(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		mutate               func(*testpilotspb.Case, *testpilot.ProfileSpec)
		category             testpilot.PreparationErrorCategory
		path, detail, prefix string
	}{
		{"program missing", func(c *testpilotspb.Case, _ *testpilot.ProfileSpec) { c.Program = nil }, testpilot.PreparationMalformed, "case", "Case identity, Program and Contract are required", ""},
		{"profile identity", func(_ *testpilotspb.Case, p *testpilot.ProfileSpec) { p.Identity = "" }, testpilot.PreparationMalformed, "policy", "Driver or catalog identity mismatch", ""},
		{"program limit", func(c *testpilotspb.Case, _ *testpilot.ProfileSpec) { c.Program.Limits.MaxNodes++ }, testpilot.PreparationLimitExceeded, "max_nodes", "limit is outside the positive Driver ceiling", ""},
		{"unknown declaration", func(c *testpilotspb.Case, _ *testpilot.ProfileSpec) { c.Contract.Rules[0].InitialStateId = "missing" }, testpilot.PreparationUnknown, "contract", "initial state is not declared", "rule safety: "},
		{"type mismatch", func(c *testpilotspb.Case, _ *testpilot.ProfileSpec) {
			c.Contract.Rules[0].Transitions[0].Predicate.GetLiteral().Value = &testpilotspb.Value_Text{Text: "text"}
		}, testpilot.PreparationTypeMismatch, "literal", "literal does not match its declared type", "rule safety: "},
		{"unavailable observation", func(c *testpilotspb.Case, _ *testpilot.ProfileSpec) {
			c.Program.Observations = []*testpilotspb.ObservationDefinition{{ObservationId: "result", Type: &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_BOOLEAN}}}}}}}
			c.Contract.Rules[0].Transitions[0].Predicate = &testpilotspb.ContractExpression{Expression: &testpilotspb.ContractExpression_Observation{Observation: &testpilotspb.ObservationRef{ObservationId: "result"}}}
		}, testpilot.PreparationUnavailable, "expression", "reference or projection requires an explicit presence guard", "rule safety: "},
		{"unsupported version", func(c *testpilotspb.Case, _ *testpilot.ProfileSpec) { c.Version.Major++ }, testpilot.PreparationUnsupported, "version", "unsupported Case version", ""},
		{"unsupported capability bounded path", func(c *testpilotspb.Case, _ *testpilot.ProfileSpec) {
			c.Program.Entrypoints[0].EntrypointId = strings.Repeat("e", 256)
			c.Program.Entrypoints[0].Instructions = []*testpilotspb.InstructionDefinition{{InstructionId: strings.Repeat("i", 256), Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRPC{}}}}}
		}, testpilot.PreparationUnsupported, strings.Repeat("e", 256), "unsupported instruction context or Driver capability", ""},
		{"contract missing rules", func(c *testpilotspb.Case, _ *testpilot.ProfileSpec) { c.Contract.Rules = nil }, testpilot.PreparationMalformed, "contract", "Contract identity and rules are required", ""},
		{"contract limit", func(c *testpilotspb.Case, _ *testpilot.ProfileSpec) { c.Contract.Limits.MaxRules++ }, testpilot.PreparationLimitExceeded, "contract", "limit outside positive Driver ceiling: max_rules", ""},
		{"correlated version", func(c *testpilotspb.Case, _ *testpilot.ProfileSpec) {
			c.Contract.Correlated = &testpilotspb.CorrelatedContract{Version: 2}
		}, testpilot.PreparationUnknown, "contract", "unsupported correlated capability version", ""},
		{"correlated binding", func(c *testpilotspb.Case, _ *testpilot.ProfileSpec) {
			c.Contract.Correlated = &testpilotspb.CorrelatedContract{Version: 1}
		}, testpilot.PreparationMalformed, "contract", "invalid correlated projection binding", ""},
		{"correlated evidence", func(c *testpilotspb.Case, _ *testpilot.ProfileSpec) {
			c.Contract.Correlated = diagnosticCorrelatedContract()
		}, testpilot.PreparationTypeMismatch, "contract", "correlated evidence requires exact declared CorrelatedEvidence Observation", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source, profile := proofFixture(t)
			tc.mutate(source, &profile)
			prepared, err := testpilot.Prepare(source, profile)
			require.Nil(t, prepared)
			requirePreparationError(t, err, tc.category, tc.path, tc.detail, tc.prefix+fmt.Sprintf("%s at %s: %s", tc.category, tc.path, tc.detail))
		})
	}
}

func diagnosticCorrelatedContract() *testpilotspb.CorrelatedContract {
	return &testpilotspb.CorrelatedContract{Version: 1, ProjectionId: "projection", ProjectionFingerprint: "fingerprint", ScopeFields: []string{"run"}, OperationField: "operation", Sources: []string{"source"}, InitialState: &testpilotspb.CorrelatedValue{DefinitionId: "state"}, EvidenceObservationId: "evidence", Limits: &testpilotspb.CorrelatedLimits{}}
}

func TestPreparationErrorCorrelatedLimits(t *testing.T) {
	source, profile := proofFixture(t)
	files := &descriptorpb.FileDescriptorSet{}
	seen := map[string]bool{}
	var collect func(protoreflect.FileDescriptor)
	collect = func(file protoreflect.FileDescriptor) {
		if seen[file.Path()] {
			return
		}
		seen[file.Path()] = true
		for i := 0; i < file.Imports().Len(); i++ {
			collect(file.Imports().Get(i).FileDescriptor)
		}
		files.File = append(files.File, protodesc.ToFileDescriptorProto(file))
	}
	collect((&testpilotspb.CorrelatedEvidence{}).ProtoReflect().Descriptor().ParentFile())
	catalog, err := testpilot.NewCatalog(files)
	require.NoError(t, err)
	profile.Catalog = catalog
	source.Program.Observations = []*testpilotspb.ObservationDefinition{{ObservationId: "evidence", Type: &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.CorrelatedEvidence"}}}}}}}
	source.Contract.Correlated = diagnosticCorrelatedContract()
	prepared, err := testpilot.Prepare(source, profile)
	require.Nil(t, prepared)
	requirePreparationError(t, err, testpilot.PreparationLimitExceeded, "contract", "correlated limits must be positive", "limit_exceeded at contract: correlated limits must be positive")
}

type nilMapProfile map[string]string
type nilSliceProfile []string
type nilChanProfile chan string
type nilFuncProfile func()

func (nilMapProfile) Snapshot() testpilot.ProfileSpec   { panic("nil Profile Snapshot called") }
func (nilSliceProfile) Snapshot() testpilot.ProfileSpec { panic("nil Profile Snapshot called") }
func (nilChanProfile) Snapshot() testpilot.ProfileSpec  { panic("nil Profile Snapshot called") }
func (nilFuncProfile) Snapshot() testpilot.ProfileSpec  { panic("nil Profile Snapshot called") }

func TestPreparationErrorProfilePreconditions(t *testing.T) {
	for name, profile := range map[string]testpilot.Profile{"nil": nil, "pointer": (*testpilot.ProfileSpec)(nil), "map": nilMapProfile(nil), "slice": nilSliceProfile(nil), "channel": nilChanProfile(nil), "function": nilFuncProfile(nil)} {
		t.Run(name, func(t *testing.T) {
			prepared, err := testpilot.Prepare(nil, profile)
			require.Nil(t, prepared)
			requirePreparationError(t, err, testpilot.PreparationMalformed, "profile", "Profile is required", "Profile is required")
		})
	}
	for name, catalog := range map[string]*testpilot.Catalog{"nil catalog": nil, "zero catalog": {}} {
		t.Run(name, func(t *testing.T) {
			source, profile := proofFixture(t)
			profile.Catalog = catalog
			prepared, err := testpilot.Prepare(source, profile)
			require.Nil(t, prepared)
			requirePreparationError(t, err, testpilot.PreparationMalformed, "profile.catalog", "Profile catalog is required", "Profile catalog is required")
		})
	}
	t.Run("nil case", func(t *testing.T) {
		_, profile := proofFixture(t)
		prepared, err := testpilot.Prepare(nil, profile)
		require.Nil(t, prepared)
		requirePreparationError(t, err, testpilot.PreparationMalformed, "$", "message is required", "malformed at $: message is required")
	})
}

func TestPreparationErrorBindings(t *testing.T) {
	for _, tc := range []struct {
		name          string
		mutate        func(*testpilot.ProfileSpec)
		path, message string
	}{
		{"limits", func(p *testpilot.ProfileSpec) { p.ProgramLimits = nil }, "profile.program_limits", "Profile Program limits are required"},
		{"collection", func(p *testpilot.ProfileSpec) { p.EnvironmentBindings = make([]testpilot.EnvironmentBinding, 10001) }, "profile.environment_bindings", "Profile environment binding collection ceiling exceeded"},
		{"identity", func(p *testpilot.ProfileSpec) { p.EnvironmentBindings[0].ID = "bad id" }, "profile.environment_bindings", "Profile environment binding 0 has an invalid identity"},
		{"value", func(p *testpilot.ProfileSpec) { p.EnvironmentBindings[0].Value = "" }, "profile.environment_bindings", "Profile environment binding \"binding\" has an invalid value"},
		{"utf8", func(p *testpilot.ProfileSpec) { p.EnvironmentBindings[0].Value = "\xff" }, "profile.environment_bindings", "Profile environment binding \"binding\" has an invalid value"},
		{"duplicate", func(p *testpilot.ProfileSpec) {
			p.EnvironmentBindings = append(p.EnvironmentBindings, p.EnvironmentBindings[0])
		}, "profile.environment_bindings", "Profile environment binding \"binding\" is duplicated"},
		{"bytes", func(p *testpilot.ProfileSpec) { p.ProgramLimits.MaxRequestBytes = 1 }, "profile.environment_bindings", "Profile environment binding byte ceiling exceeded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source, profile := proofFixture(t)
			profile.EnvironmentBindings = []testpilot.EnvironmentBinding{{ID: "binding", Value: "value"}}
			tc.mutate(&profile)
			fingerprint, err := profile.BindingFingerprint()
			require.Empty(t, fingerprint)
			requirePreparationError(t, err, testpilot.PreparationMalformed, tc.path, tc.message, tc.message)
			prepared, err := testpilot.Prepare(source, profile)
			require.Nil(t, prepared)
			requirePreparationError(t, err, testpilot.PreparationMalformed, tc.path, tc.message, tc.message)
		})
	}
}

func TestPreparationErrorSuccessAndExclusions(t *testing.T) {
	source, profile := proofFixture(t)
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)
	require.True(t, proto.Equal(source, prepared.Snapshot()))
	var diagnostic *testpilot.PreparationError
	require.NotErrorAs(t, err, &diagnostic)
	decodeErr := protojson.Unmarshal([]byte(`{"unknown":true}`), &testpilotspb.Case{})
	require.Error(t, decodeErr)
	require.NotErrorAs(t, decodeErr, &diagnostic)
	driverErr := errors.New("driver unavailable")
	_, _, err = prepared.Run(t.Context(), &proofDriver{identity: prepared.Identity(), openErr: driverErr})
	require.ErrorIs(t, err, driverErr)
	require.NotErrorAs(t, err, &diagnostic)
}
