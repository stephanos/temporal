package lint

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	_ "go.temporal.io/api/workflowservice/v1"
	_ "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

const apiActivityStatus = "temporal.api.activity.v1.ActivityExecutionInfo.status"

// apiRegistryLowering reads evidence elements and paths from the linked descriptors, as the command's
// Lowering does through lower, which this package may not import. It records every field a path
// reaches.
func apiRegistryLowering(reached *[]string) Lowering {
	message := func(name string) (protoreflect.MessageDescriptor, error) {
		d, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(name))
		if err != nil {
			return nil, err
		}
		return d.(protoreflect.MessageDescriptor), nil
	}
	field := func(_ *umpirespb.Position, md protoreflect.MessageDescriptor, path string) (protoreflect.FieldDescriptor, error) {
		fd, err := apiWalk(md, path)
		if err == nil && reached != nil {
			*reached = append(*reached, string(fd.FullName()))
		}
		return fd, err
	}
	response := func(read *umpirespb.ReadSource) (protoreflect.MessageDescriptor, error) {
		service, method, _ := strings.Cut(strings.TrimPrefix(read.GetMethod(), "/"), "/")
		d, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(service))
		if err != nil {
			return nil, err
		}
		fd, err := apiWalk(d.(protoreflect.ServiceDescriptor).Methods().ByName(protoreflect.Name(method)).Output(), read.GetPath())
		if err != nil {
			return nil, err
		}
		return fd.Message(), nil
	}
	return Lowering{
		Element: func(_ *umpirespb.Realization, e *umpirespb.Evidence) (protoreflect.MessageDescriptor, error) {
			switch from := e.GetFrom().(type) {
			case *umpirespb.Evidence_Read:
				return response(from.Read)
			case *umpirespb.Evidence_Single:
				return response(from.Single)
			case *umpirespb.Evidence_RunEvent:
				return message("temporal.server.api.testpilot.v1.InstructionOutcome")
			case *umpirespb.Evidence_History:
				return message("temporal.api.history.v1.HistoryEvent")
			default:
				return nil, fmt.Errorf("evidence %s is recorded nowhere", e.GetId())
			}
		},
		Field: field,
	}
}

var apiSegment = regexp.MustCompile(`^([a-z0-9_]+)(\[\*\]|<([a-z0-9_]+)>)?$`)

// apiWalk reads the path grammar of a realization: `field`, `field[*]` and `oneof<member>`.
func apiWalk(md protoreflect.MessageDescriptor, path string) (protoreflect.FieldDescriptor, error) {
	var fd protoreflect.FieldDescriptor
	for i, s := range strings.Split(path, ".") {
		if i > 0 {
			if md = fd.Message(); md == nil {
				return nil, fmt.Errorf("%s is no message", fd.FullName())
			}
		}
		m := apiSegment.FindStringSubmatch(s)
		if m == nil {
			return nil, fmt.Errorf("%q is no segment", s)
		}
		if m[3] != "" {
			oneof := md.Oneofs().ByName(protoreflect.Name(m[1]))
			if oneof == nil {
				return nil, fmt.Errorf("%s has no oneof %s", md.FullName(), m[1])
			}
			fd = oneof.Fields().ByName(protoreflect.Name(m[3]))
		} else {
			fd = md.Fields().ByName(protoreflect.Name(m[1]))
		}
		if fd == nil {
			return nil, fmt.Errorf("%s has no field %s", md.FullName(), s)
		}
	}
	return fd, nil
}

// apiActivityIR is a copy of the standalone activity's IR, whose realization polls the activity's status
// for every terminal value and PAUSED, and whose Run Event guards test the instruction's outcome, a
// field of the run's own record that no count includes.
func apiActivityIR(t *testing.T) (*umpirespb.Model, *umpirespb.Realization) {
	t.Helper()
	ir, err := model.Load("../../../model/ir/activity.json")
	require.NoError(t, err)
	ir = proto.Clone(ir).(*umpirespb.Model)
	require.Len(t, ir.GetRealizations(), 1)
	return ir, ir.GetRealizations()[0]
}

func apiTallyOf(t *testing.T, ir *umpirespb.Model, lowering Lowering) Tally {
	t.Helper()
	m, err := Of("", ir, lowering, Options{})
	require.NoError(t, err)
	tallies, err := unmodeledAPIValues(m)
	require.NoError(t, err)
	require.Len(t, tallies, 1)
	require.Equal(t, "activitySystem", tallies[0].Owner)
	return tallies[0]
}

func apiSubjects(t Tally) []string {
	var out []string
	for _, f := range t.Findings {
		out = append(out, f.Subject)
	}
	return out
}

func apiEvidence(t *testing.T, r *umpirespb.Realization, suffix string) *umpirespb.Evidence {
	t.Helper()
	for _, e := range r.GetEvidence() {
		if strings.HasSuffix(e.GetId(), suffix) {
			return e
		}
	}
	require.FailNow(t, "no evidence "+suffix)
	return nil
}

// apiCommands is every command of the realization's scripts, items and performances alike.
func apiCommands(r *umpirespb.Realization) []*umpirespb.Command {
	var out []*umpirespb.Command
	for _, s := range r.GetScripts() {
		for _, item := range s.GetItems() {
			out = append(out, item.GetCommand())
			for _, p := range item.GetPerforms() {
				out = append(out, p.GetCommand())
			}
		}
	}
	return out
}

func apiProjected(path string) *umpirespb.Operand {
	return &umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{
		Of: &umpirespb.Operand{Kind: &umpirespb.Operand_Projected{Projected: &umpirespb.Empty{}}}, Path: path}}}
}

func apiEnum(name string) *umpirespb.Operand {
	return &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_EnumName{EnumName: name}}}}
}

func apiStatusIs(name string) *umpirespb.Operand {
	return &umpirespb.Operand{Kind: &umpirespb.Operand_Equal{Equal: &umpirespb.Equal{Left: apiProjected("status"), Right: apiEnum(name)}}}
}

// apiAlsoUntil widens every poll of the evidence kind to test one more condition.
func apiAlsoUntil(r *umpirespb.Realization, evidence string, condition *umpirespb.Operand) {
	for _, c := range apiCommands(r) {
		if poll := c.GetPoll(); poll != nil && poll.GetEvidence() == evidence {
			poll.Until = &umpirespb.Operand{Position: poll.GetUntil().GetPosition(), Kind: &umpirespb.Operand_All{
				All: &umpirespb.All{Operands: []*umpirespb.Operand{poll.GetUntil(), condition}}}}
		}
	}
}

func TestUnmodeledAPIValueReportsAValueAPollWaitsThrough(t *testing.T) {
	ir, _ := apiActivityIR(t)
	tally := apiTallyOf(t, ir, apiRegistryLowering(nil))
	require.Contains(t, apiSubjects(tally), apiActivityStatus+" ACTIVITY_EXECUTION_STATUS_RUNNING")
	for _, f := range tally.Findings {
		if f.Subject == apiActivityStatus+" ACTIVITY_EXECUTION_STATUS_RUNNING" {
			require.Equal(t, UnmodeledAPIValue, f.Kind)
			require.Equal(t, "ActivityExecutionInfo.status ACTIVITY_EXECUTION_STATUS_RUNNING is mapped to a fact by no poll or guard", f.Message)
			require.Equal(t, "model/temporal/features/standaloneactivity/Realization.scala:57", f.Position)
		}
	}
	for _, mapped := range []string{"PAUSED", "COMPLETED", "FAILED", "CANCELED", "TERMINATED", "TIMED_OUT"} {
		require.NotContains(t, apiSubjects(tally), apiActivityStatus+" ACTIVITY_EXECUTION_STATUS_"+mapped)
	}
}

func TestUnmodeledAPIValueIsQuietWhenEveryValueIsMapped(t *testing.T) {
	ir, r := apiActivityIR(t)
	for _, e := range r.GetEvidence() {
		if source := e.GetRunEvent(); source != nil {
			source.Guard = nil
		}
	}
	apiAlsoUntil(r, apiEvidence(t, r, ".statusPaused").GetId(), &umpirespb.Operand{Kind: &umpirespb.Operand_Not{
		Not: &umpirespb.Not{Of: apiStatusIs("ACTIVITY_EXECUTION_STATUS_RUNNING")}}})
	tally := apiTallyOf(t, ir, apiRegistryLowering(nil))
	require.Equal(t, 7, tally.Population)
	require.Empty(t, tally.Findings)
}

func TestUnmodeledAPIValueNeverReadsAWrittenRequestField(t *testing.T) {
	ir, r := apiActivityIR(t)
	written := 0
	for _, c := range apiCommands(r) {
		if rpc := c.GetRpc(); rpc != nil && strings.HasSuffix(rpc.GetMethod(), "/StartActivityExecution") {
			rpc.Assign = append(rpc.Assign, &umpirespb.Assignment{Target: "id_reuse_policy", Value: apiEnum("ACTIVITY_ID_REUSE_POLICY_REJECT_DUPLICATE")})
			written++
		}
	}
	require.Positive(t, written)
	var reached []string
	tally := apiTallyOf(t, ir, apiRegistryLowering(&reached))
	require.Equal(t, 7, tally.Population, "the activity's 7 statuses")
	for _, s := range append(apiSubjects(tally), reached...) {
		require.NotContains(t, s, "StartActivityExecutionRequest")
		require.NotContains(t, s, "ActivityIdReusePolicy")
	}
}

func TestUnmodeledAPIValueCountsNoZeroValue(t *testing.T) {
	ir, r := apiActivityIR(t)
	apiAlsoUntil(r, apiEvidence(t, r, ".statusPaused").GetId(), apiStatusIs("ACTIVITY_EXECUTION_STATUS_UNSPECIFIED"))
	tally := apiTallyOf(t, ir, apiRegistryLowering(nil))
	require.Equal(t, 7, tally.Population)
	for _, s := range apiSubjects(tally) {
		require.NotContains(t, s, "UNSPECIFIED")
	}
}

func TestUnmodeledAPIValueIsNotMappedByAKindThatRecordsNoFact(t *testing.T) {
	ir, r := apiActivityIR(t)
	paused := apiEvidence(t, r, ".statusPaused")
	// The kind stays, recording what the machine's evidence function names no fact by.
	paused.Records = "statusPausedUnmodeled"
	tally := apiTallyOf(t, ir, apiRegistryLowering(nil))
	require.Contains(t, apiSubjects(tally), apiActivityStatus+" ACTIVITY_EXECUTION_STATUS_PAUSED")
	require.NotContains(t, apiSubjects(tally), apiActivityStatus+" ACTIVITY_EXECUTION_STATUS_COMPLETED")
}

func TestUnmodeledAPIValueCountsAOneofMemberAPollSelects(t *testing.T) {
	ir, r := apiActivityIR(t)
	// The paused kind reads the activity's outcome instead, and its poll waits for a failure.
	paused := apiEvidence(t, r, ".statusPaused")
	paused.From = &umpirespb.Evidence_Single{Single: &umpirespb.ReadSource{
		Method: "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution", Path: "outcome"}}
	for _, c := range apiCommands(r) {
		if poll := c.GetPoll(); poll != nil && poll.GetEvidence() == paused.GetId() {
			poll.Until = &umpirespb.Operand{Kind: &umpirespb.Operand_Present{Present: &umpirespb.Present{Of: apiProjected("value<failure>")}}}
		}
	}
	tally := apiTallyOf(t, ir, apiRegistryLowering(nil))
	require.Equal(t, 9, tally.Population, "the activity's 7 statuses, and the two members of the outcome's oneof")
	require.NotContains(t, apiSubjects(tally), "temporal.api.activity.v1.ActivityExecutionOutcome.value failure")
	require.Contains(t, apiSubjects(tally), "temporal.api.activity.v1.ActivityExecutionOutcome.value result")
	for _, f := range tally.Findings {
		if f.Subject == "temporal.api.activity.v1.ActivityExecutionOutcome.value result" {
			require.Equal(t, "ActivityExecutionOutcome.value result is mapped to a fact by no poll or guard", f.Message)
		}
	}
}

func TestUnmodeledAPIValueCountsNoValueOfTheRunsOwnRecord(t *testing.T) {
	ir, r := apiActivityIR(t)
	// A guard over a oneof of the instruction outcome, the run's own record, as its status is.
	scheduled := apiEvidence(t, r, ".statusScheduled").GetRunEvent()
	scheduled.Guard = &umpirespb.Operand{Kind: &umpirespb.Operand_All{All: &umpirespb.All{Operands: []*umpirespb.Operand{
		scheduled.GetGuard(), {Kind: &umpirespb.Operand_Present{Present: &umpirespb.Present{Of: apiProjected("value.value<enum_value>")}}}}}}}
	tally := apiTallyOf(t, ir, apiRegistryLowering(nil))
	require.Equal(t, 7, tally.Population)
	for _, s := range apiSubjects(tally) {
		require.NotContains(t, s, "temporal.server.api.testpilot.v1")
	}
}

func TestUnmodeledAPIValueNeedsLoweringsDescriptors(t *testing.T) {
	ir, _ := apiActivityIR(t)
	m, err := Of("", ir, Lowering{}, Options{})
	require.NoError(t, err)
	_, err = unmodeledAPIValues(m)
	require.ErrorContains(t, err, "lowering's descriptors")
}

func TestUnmodeledAPIValueCountsNoHistoryAttributesOneof(t *testing.T) {
	ir, r := apiActivityIR(t)
	paused := apiEvidence(t, r, ".statusPaused")
	// A read of the history's events, polled until one member of the attributes oneof is present, and
	// a history kind lifting another member of it.
	paused.From = &umpirespb.Evidence_Read{Read: &umpirespb.ReadSource{
		Method: "/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory", Path: "history.events[*]"}}
	terminated := apiEvidence(t, r, ".statusTerminated")
	terminated.From = &umpirespb.Evidence_History{History: "activity_task_started_event_attributes"}
	for _, c := range apiCommands(r) {
		if poll := c.GetPoll(); poll != nil && (poll.GetEvidence() == paused.GetId() || poll.GetEvidence() == terminated.GetId()) {
			poll.Evidence = paused.GetId()
			poll.Until = &umpirespb.Operand{Kind: &umpirespb.Operand_Present{Present: &umpirespb.Present{
				Of: apiProjected("attributes<activity_task_scheduled_event_attributes>")}}}
		}
	}
	tally := apiTallyOf(t, ir, apiRegistryLowering(nil))
	for _, s := range apiSubjects(tally) {
		require.NotContains(t, s, "attributes")
	}
	require.Equal(t, 7, tally.Population, "the activity's statuses the other polls test")
	require.Contains(t, apiSubjects(tally), apiActivityStatus+" ACTIVITY_EXECUTION_STATUS_TERMINATED", "no poll tests it any more")
}
