package conformance

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	celpb "cel.dev/expr"
	"go.temporal.io/server/common/testing/testpilot/cel"
	"go.temporal.io/server/common/testing/testpilot/duration"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/check"
	"go.temporal.io/server/tools/umpire/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
)

// The Runs these tests assess are of three kinds, and no lowered Case has run against a server:
//
//   - recorded from a fake: a carrier Case, written here, reads one piece of correlated evidence per
//     instruction from a fake Driver, so Testpilot's own executor and recorder make the Run, live, and
//     the same Run is then replayed;
//   - played: a lowered Case run live, by Testpilot's own executor and recorder, against a Driver that
//     plays the path of its Query, and then replayed (nexus_test.go, played_test.go);
//   - constructed: a Run of a lowered Case written out event by event, which only a replay reads
//     (nexus_test.go, activity_test.go).

const (
	evidenceObservation = "correlated-evidence"
	sourceRole          = "source"
	sourceMethod        = "/test.conformance.Source/Read"
)

func lifted(t testing.TB, name string) *umpirespb.Model {
	t.Helper()
	m, err := ir.Load(filepath.Join("..", "..", "..", "model", "irgen", "testdata", "lifts", "expected", name+".json"))
	require.NoError(t, err)
	return m
}

// kindOf is one kind of evidence a test realization declares: the fact name it records and what it
// commits to.
type kindOf struct {
	records string
	durable bool
}

// declaring is what a test realization says of one kind beyond that: the fields it keeps, and whether
// its source reports every occurrence.
type declaring struct {
	fields     []*umpirespb.EvidenceField
	exhaustive bool
}

// closingInstruction is the command a test realization closes its exhaustive kinds by, and the instruction of
// the carrier Case that stands for it.
const closingInstruction = "close"

// admissionKinds is the activity specimen's evidence and capability matrix
// (model/specimens/activity.md): the statuses are public, what a caller reads back; the
// dispatch, the admission commit and the rejection are internal, observed where the server commits
// them. The admission fixture declares no realization, so the tests declare this one.
var admissionKinds = []kindOf{
	{"statusStarted", false}, {"statusPaused", false}, {"statusCompleted", false},
	{"dispatchEnqueued", true}, {"attemptAdmitted", true}, {"admissionRejected", true},
}

// publicKinds is what a Run carries with the commit observations removed: what a caller was told.
var publicKinds = []string{"statusStarted", "statusPaused", "statusCompleted"}

func kindID(machine, records string) string   { return "test." + machine + ".evidence." + records }
func sourceID(machine, records string) string { return "test." + machine + ".source." + records }

// realized adds a realization of one machine to a Model: its kinds of evidence, each from a source
// of its own, and the correlation that reads them. It declares no script: these Runs are driven by
// the carrier Case.
func realized(t testing.TB, m *umpirespb.Model, machine string, kinds []kindOf) *umpirespb.Model {
	t.Helper()
	return realizedWith(t, m, machine, kinds, nil)
}

// realizedWith is realized with more declared of the kinds that record the named facts. The
// exhaustive ones are closed by one read of a controller script.
func realizedWith(t testing.TB, m *umpirespb.Model, machine string, kinds []kindOf, more map[string]declaring) *umpirespb.Model {
	t.Helper()
	out := proto.CloneOf(m)
	r := &umpirespb.Realization{Id: "test." + machine + ".realization", Name: machine + "Evidence", Machine: machine, Producer: "test.conformance",
		ProducerVersion: "1",
		Observations:    []*umpirespb.Observed{{Id: evidenceObservation, Message: "temporal.server.api.testpilot.v1.CorrelatedEvidence"}},
		Correlation: &umpirespb.Correlation{Projection: "test." + machine + ".projection", Run: "run", Operation: "operation",
			Observation: evidenceObservation, Events: 64, Buffered: 16, Keys: 8, Support: 128, Work: 1000000, EventSize: 512},
		// Attempts are numbered as the Temporal kit declares: from 1, every one of the activity's one run.
		Behavior: &umpirespb.ApiBehavior{AttemptNumbering: &umpirespb.AttemptNumbering{First: 1, OneRun: true}}}
	for _, k := range kinds {
		commitment := umpirespb.Evidence_COMMITMENT_REPORTED
		if k.durable {
			commitment = umpirespb.Evidence_COMMITMENT_DURABLE
		}
		r.Evidence = append(r.Evidence, &umpirespb.Evidence{Id: kindID(machine, k.records), Records: k.records, Source: sourceID(machine, k.records),
			From:      &umpirespb.Evidence_History{History: "workflow_execution_started_event_attributes"},
			Operation: "event_id", Commitment: commitment, Fields: more[k.records].fields, Exhaustive: more[k.records].exhaustive})
	}
	// The exhaustive kinds are closed by one read of the controller, which lifts them.
	closing := &umpirespb.Command{Id: closingInstruction, Instruction: &umpirespb.Command_Rpc{Rpc: &umpirespb.Rpc{Role: sourceRole, Method: sourceMethod,
		Reads: []*umpirespb.ResponseRead{{Path: "history.events[*]", Cardinality: umpirespb.ResponseRead_CARDINALITY_EACH,
			Targets: []*umpirespb.Target{{Target: &umpirespb.Target_Lift{Lift: evidenceObservation}}}}}}}}
	for _, k := range kinds {
		if more[k.records].exhaustive {
			closing.Closes = append(closing.Closes, kindID(machine, k.records))
		}
	}
	if len(closing.GetCloses()) > 0 {
		r.Roles = []*umpirespb.Role{{Id: sourceRole, Kind: umpirespb.Role_KIND_ENDPOINT}}
		r.Scripts = []*umpirespb.Script{{Id: "controller", Activation: &umpirespb.Script_Controller{Controller: &emptypb.Empty{}},
			Items: []*umpirespb.Item{{Command: closing}}}}
	}
	out.Realizations = append(out.Realizations, r)
	require.NoError(t, ir.Validate(out))
	return out
}

// carrier is a Case whose Program reads one piece of correlated evidence per instruction and whose
// correlated Contract gives every kind it carries no meaning of its own: the Contract checks that the
// evidence is well formed and concludes nothing from it, so the Run closes whatever the model would
// say. The kinds it carries are the ones of machine whose facts are named.
func carrier(machine string, carried []string, reads int) *testpilotspb.Case {
	return carrierWith(machine, carried, reads, nil, false)
}

// retained is the policy of one field a carrier Case's evidence keeps with its value.
func retained(id string, kind testpilotspb.ScalarKind) *testpilotspb.CorrelatedFieldPolicy {
	return &testpilotspb.CorrelatedFieldPolicy{FieldId: id, Type: &testpilotspb.ScalarType{Kind: kind}, Disposition: testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN}
}

// carrierWith is a carrier Case whose kinds carry the named fields, by the fact each kind records, and
// which ends, when closes is set, with the instruction that stands for the realization's closing read:
// a call that reads nothing into the Run and whose outcome the Run records.
func carrierWith(machine string, carried []string, reads int, fields map[string][]*testpilotspb.CorrelatedFieldPolicy, closes bool) *testpilotspb.Case {
	message := func(name string) *testpilotspb.ValueType {
		return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{
			Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: name}}}}}
	}
	step := func(field testpilotspb.CorrelatedStepField, definition string) *testpilotspb.Expression {
		return cel.Compare("_>_", cel.Size(cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_CorrelatedStep{CorrelatedStep: &testpilotspb.CorrelatedStepReference{Field: field, DefinitionId: definition}}})), cel.Literal(&celpb.Value{Kind: &celpb.Value_Int64Value{Int64Value: 0}}))
	}
	idle := &testpilotspb.ModelValue{DefinitionId: "state", Value: "idle"}
	never := &testpilotspb.ModelValue{DefinitionId: "action", Value: "never"}
	quiet := &testpilotspb.ModelValue{DefinitionId: "outcome", Value: "quiet"}
	correlated := &testpilotspb.CorrelatedContract{ProjectionId: "projection", ProjectionFingerprint: "sha256:carrier",
		EvidenceObservationId: evidenceObservation, ScopeFields: []string{"run"}, OperationField: "operation", InitialStateId: "s1",
		States:      []*testpilotspb.CorrelatedState{{StateId: "s1", Atom: idle}},
		Results:     []*testpilotspb.CorrelatedResult{{ResultId: "r1", Action: never, StateId: "s1", Outcome: quiet}},
		Transitions: []*testpilotspb.CorrelatedTransition{{PriorStateId: "s1", ResultId: "r1"}},
		Rules: []*testpilotspb.CorrelatedRule{{RuleId: "carried", Trigger: step(testpilotspb.CORRELATED_STEP_FIELD_ACTION, "action"),
			Response: step(testpilotspb.CORRELATED_STEP_FIELD_OUTCOME, "outcome"),
			Bound:    1, Ending: testpilotspb.TRACE_ENDING_PARTIAL}}}
	for _, records := range carried {
		correlated.Sources = append(correlated.Sources, sourceID(machine, records))
		correlated.ProjectionRules = append(correlated.ProjectionRules, &testpilotspb.CorrelatedProjectionRule{Kind: kindID(machine, records),
			Meaning: testpilotspb.CORRELATED_EVIDENCE_MEANING_IRRELEVANT, Fields: fields[records]})
	}
	controller := &testpilotspb.Entrypoint{EntrypointId: "controller", Activation: &testpilotspb.Entrypoint_Controller{Controller: &emptypb.Empty{}}}
	for i := range reads {
		node := &testpilotspb.InstructionNode{InstructionId: "read." + strconv.Itoa(i), Instruction: &testpilotspb.Instruction{
			Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{EndpointRoleId: sourceRole, Method: sourceMethod,
				ResponseReads: []*testpilotspb.ResponseRead{{
					Targets: []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_ObservationId{ObservationId: evidenceObservation}}}}}}}}}
		if i > 0 {
			// A read runs whatever became of the one before it: a transport failure loses one piece of
			// evidence and not the rest of the Run.
			node.Guard = cel.Literal(&celpb.Value{Kind: &celpb.Value_BoolValue{BoolValue: true}})
		}
		controller.Instructions = append(controller.Instructions, node)
	}
	name := fmt.Sprintf("test.conformance.%s.%s.%d", machine, strings.Join(carried, "-"), reads)
	if closes {
		name += ".closed"
		controller.Instructions = append(controller.Instructions, &testpilotspb.InstructionNode{InstructionId: closingInstruction,
			Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{
				EndpointRoleId: sourceRole, Method: sourceMethod}}},
			Guard: cel.Literal(&celpb.Value{Kind: &celpb.Value_BoolValue{BoolValue: true}})})
	}
	return &testpilotspb.Case{CaseId: name, Version: &testpilotspb.FormatVersion{Major: 4},
		Provenance: &testpilotspb.CaseProvenance{ProducerId: "test.conformance"},
		Program: &testpilotspb.Program{ProgramId: name + ".program", Roles: []*testpilotspb.Role{{RoleId: sourceRole, Kind: testpilotspb.ROLE_KIND_ENDPOINT}},
			Observations: []*testpilotspb.Observation{{ObservationId: evidenceObservation, Type: message("temporal.server.api.testpilot.v1.CorrelatedEvidence")}},
			Entrypoints:  []*testpilotspb.Entrypoint{controller}, Cleanup: &testpilotspb.Cleanup{EntrypointId: "cleanup"}},
		Contract: &testpilotspb.Contract{ContractId: name + ".contract", Correlated: correlated}}
}

func descriptorClosure(root protoreflect.FileDescriptor) *descriptorpb.FileDescriptorSet {
	seen, out := map[string]bool{}, &descriptorpb.FileDescriptorSet{}
	var add func(protoreflect.FileDescriptor)
	add = func(file protoreflect.FileDescriptor) {
		if seen[file.Path()] {
			return
		}
		seen[file.Path()] = true
		for i := range file.Imports().Len() {
			add(file.Imports().Get(i))
		}
		out.File = append(out.File, protodesc.ToFileDescriptorProto(file))
	}
	add(root)
	return out
}

// carrierProfile authorizes the one method the carrier Case reads its evidence through.
func carrierProfile(t testing.TB) testpilot.ProfileSpec {
	t.Helper()
	descriptors := descriptorClosure(testpilotspb.File_temporal_server_api_testpilot_v1_case_proto)
	evidence := ".temporal.server.api.testpilot.v1.CorrelatedEvidence"
	descriptors.File = append(descriptors.File, &descriptorpb.FileDescriptorProto{
		Name: proto.String("test/conformance/source.proto"), Package: proto.String("test.conformance"), Syntax: proto.String("proto3"),
		Dependency: []string{testpilotspb.File_temporal_server_api_testpilot_v1_correlated_proto.Path()},
		Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("Source"), Method: []*descriptorpb.MethodDescriptorProto{{
			Name: proto.String("Read"), InputType: proto.String(evidence), OutputType: proto.String(evidence)}}}},
	})
	catalog, err := testpilot.NewCatalog(descriptors)
	require.NoError(t, err)
	return testpilot.ProfileSpec{
		Identity: "conformance-carrier", Catalog: catalog,
		Roles:   []testpilot.RolePolicy{{ID: sourceRole, Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{sourceMethod}}},
		Opcodes: []testpilot.Opcode{testpilot.InvokeRPC},
		ProgramLimits: &testpilotspb.ProgramLimits{
			MaxEntrypoints: 4, MaxNodes: 256, MaxEdges: 256, MaxActivations: 256, MaxAttempts: 256, MaxRunEvents: 2048,
			MaxExpressionDepth: 8, MaxPathFanout: 256, MaxRequestBytes: 4096, MaxResponseBytes: 4096,
			MaxDuration: duration.FromMilliseconds(10000), CleanupDuration: duration.FromMilliseconds(1000),
			MaxInstructionEmittedEvents: 1, MaxInstructionResponseBytes: 4096,
		},
		ContractLimits: &testpilotspb.ContractLimits{
			MaxRules: 16, MaxStates: 32, MaxTransitions: 64, MaxExpressionDepth: 16,
			MaxWorkPerEvent: 1000000, MaxTotalWork: 1000000000, MaxCaptures: 32, MaxCaptureBytes: 1 << 20,
		},
		CorrelatedLimits: &testpilotspb.CorrelatedLimits{
			MaxEvents: 64, MaxBuffered: 64, MaxKeys: 8, MaxSupport: 4096, MaxProjectionWork: 1000000000,
			MaxEventBytes: 1024, MaxSemanticTransitions: 32, MaxObligations: 16, MaxObligationWork: 1000000000,
		},
		InstructionDefaults: testpilot.InstructionDefaults{TimeoutMilliseconds: 1000, MaxAttempts: 1},
	}
}

// read is what one instruction of a carrier Case's Run gets back: a piece of evidence, a transport
// timeout that loses the response and nothing else, or the loss of the source, which leaves the Run
// incomplete.
type read struct {
	name     string
	evidence *testpilotspb.CorrelatedEvidence
	timeout  bool
	lost     bool
}

// fact is evidence of one fact of machine for an operation of a Run scope, at an ordinal of its
// kind's own source, ordered after the named reads of the same script, carrying these fields.
type fact struct {
	name, records, operation, run string
	ordinal                       int64
	after                         []string
	fields                        []*testpilotspb.NamedValue
}

func textField(id, value string) *testpilotspb.NamedValue {
	return &testpilotspb.NamedValue{FieldId: id, Value: &celpb.Value{Kind: &celpb.Value_StringValue{StringValue: value}}}
}

func numberField(id string, value uint64) *testpilotspb.NamedValue {
	return &testpilotspb.NamedValue{FieldId: id, Value: &celpb.Value{Kind: &celpb.Value_Uint64Value{
		Uint64Value: value}}}
}

// script turns facts and failures into what each read returns, resolving each `after` to the identity
// of the evidence it names.
func script(machine string, items ...any) []read {
	identities := map[string]*testpilotspb.CorrelatedIdentity{}
	var out []read
	for _, item := range items {
		switch item := item.(type) {
		case fact:
			run, operation := item.run, item.operation
			if run == "" {
				run = "run-1"
			}
			if operation == "" {
				operation = "activity-1"
			}
			identity := &testpilotspb.CorrelatedIdentity{EvidenceSource: sourceID(machine, item.records), Ordinal: item.ordinal,
				Scope: []*testpilotspb.NamedValue{{FieldId: "run", Value: &celpb.Value{Kind: &celpb.Value_StringValue{StringValue: run}}}}}
			evidence := &testpilotspb.CorrelatedEvidence{Identity: identity, Operation: operation, Kind: kindID(machine, item.records), Fields: item.fields}
			for _, parent := range item.after {
				evidence.Parents = append(evidence.Parents, proto.CloneOf(identities[parent]))
			}
			identities[item.name] = identity
			out = append(out, read{name: item.name, evidence: evidence})
		case read:
			out = append(out, item)
		default:
		}
	}
	return out
}

var (
	timedOut   = read{name: "timeout", timeout: true}
	sourceLost = read{name: "lost", lost: true}
)

// sourceDriver is a fake Driver written against the public facade alone: it answers instruction
// read.N with the Nth read of its script, and the closing read with success unless it is to fail.
type sourceDriver struct {
	identity testpilot.DriverIdentity
	script   []read
	// closingFails makes the closing read time out, as a read does.
	closingFails bool
}

func (d *sourceDriver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return d.identity, nil
}
func (d *sourceDriver) Validate(context.Context, testpilot.PreparedProgram) error { return nil }
func (d *sourceDriver) Open(context.Context, string, testpilot.PreparedProgram) (testpilot.Session, error) {
	return &sourceSession{script: d.script, closingFails: d.closingFails}, nil
}

type sourceSession struct {
	script       []read
	closingFails bool
}

var errUnscripted = errors.New("the carrier Case makes no such call")

func (s *sourceSession) InvokeRPC(_ context.Context, at testpilot.Coordinate, role string, _ protoreflect.MethodDescriptor, _ proto.Message) (testpilot.EffectHandle, error) {
	if at.InstructionID == closingInstruction && role == sourceRole {
		if s.closingFails {
			return effect{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE, ProtocolCode: "deadline_exceeded"}}, nil
		}
		return effect{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, Response: &testpilotspb.CorrelatedEvidence{}}, nil
	}
	index, err := strconv.Atoi(strings.TrimPrefix(at.InstructionID, "read."))
	if err != nil || role != sourceRole || index < 0 || index >= len(s.script) {
		return nil, errUnscripted
	}
	switch item := s.script[index]; {
	case item.lost:
		return nil, errors.New("evidence source lost")
	case item.timeout:
		return effect{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE, ProtocolCode: "deadline_exceeded"}}, nil
	default:
		return effect{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, Response: proto.CloneOf(item.evidence)}, nil
	}
}

func (*sourceSession) Reserve(context.Context, testpilot.ReservationRequest) ([]testpilot.ReservationHandle, error) {
	return nil, errUnscripted
}
func (*sourceSession) PollRPC(context.Context, testpilot.Coordinate, string, protoreflect.MethodDescriptor, proto.Message, time.Duration, testpilot.PollPredicate) (testpilot.EffectHandle, error) {
	return nil, errUnscripted
}
func (*sourceSession) InvokeHandle(context.Context, testpilot.Coordinate, testpilot.OpaqueHandle, proto.Message) (testpilot.EffectHandle, error) {
	return nil, errUnscripted
}
func (*sourceSession) InjectFault(context.Context, testpilot.Coordinate, string, testpilotspb.FaultKind) (testpilot.EffectHandle, error) {
	return nil, errUnscripted
}
func (*sourceSession) Bridge(context.Context) (testpilot.HandleBridge, error)   { return nil, nil }
func (*sourceSession) Quarantine(context.Context, testpilot.EffectHandle) error { return nil }
func (*sourceSession) Close(context.Context) error                              { return nil }
func (*sourceSession) Diagnose(context.Context, string, *testpilotspb.RunDiagnostic) error {
	return nil
}

// effect is an effect that has already completed.
type effect testpilot.EffectResult

func (e effect) Wait(context.Context) (testpilot.EffectResult, error) {
	return testpilot.EffectResult{Outcome: proto.CloneOf(e.Outcome), Response: e.Response}, nil
}
func (effect) Cancel(context.Context) error { return nil }
func (effect) Drain(context.Context) error  { return nil }

// generous are ceilings no fixture reaches.
var generous = Limits{MaxEvents: 2048, MaxProperties: 16, MaxDuration: time.Minute, MaxCandidates: 1 << 16, MaxWork: 1 << 22, MaxReadings: 1 << 22}

// bound is one Query of one Model bound to a carrier Case, ready to run and replay.
type bound struct {
	source   *testpilotspb.Case
	plain    *testpilot.PreparedCase
	assessed *testpilot.AssessedCase
	factory  *Factory
}

func bind(t testing.TB, m *umpirespb.Model, query check.ClaimKey, source *testpilotspb.Case, limits Limits) *bound {
	t.Helper()
	plain, err := testpilot.Prepare(source, carrierProfile(t))
	require.NoError(t, err)
	factory, err := Prepare(m, query, source, limits)
	require.NoError(t, err)
	assessed, err := plain.WithAssessment(factory)
	require.NoError(t, err)
	return &bound{source: source, plain: plain, assessed: assessed, factory: factory}
}

// carrying is the Run Events of a Run that carry the named reads' evidence, ascending: the support an
// assessment may name, read off the Run and not off the assessment.
func carrying(t testing.TB, run *testpilotspb.Run, reads []read, names ...string) []int64 {
	t.Helper()
	var out []int64
	for _, event := range run.GetEvents() {
		for _, observation := range event.GetObservations() {
			evidence := &testpilotspb.CorrelatedEvidence{}
			require.NoError(t, observation.GetValue().GetObjectValue().UnmarshalTo(evidence))
			for _, item := range reads {
				if item.evidence != nil && proto.Equal(item.evidence, evidence) {
					for _, name := range names {
						if name == item.name {
							out = append(out, event.GetSequence())
						}
					}
				}
			}
		}
	}
	require.Len(t, out, len(names), "every named read's evidence is in the Run once")
	return out
}

func evidenceValue(t testing.TB, evidence *testpilotspb.CorrelatedEvidence) *celpb.Value {
	t.Helper()
	packed, err := anypb.New(evidence)
	require.NoError(t, err)
	return &celpb.Value{Kind: &celpb.Value_ObjectValue{ObjectValue: packed}}
}
