package lower

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	commandpb "go.temporal.io/api/command/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

const realizationAt = "model/temporal/features/nexuscaller/Realization.scala:"

// functionalQueries is the functional set of model/temporal/features/nexuscaller/system/System.scala,
// in its declaration order.
var functionalQueries = []string{"syncCompletion", "asyncCompletion", "asyncFailure", "handlerError", "retry",
	"scheduleToStartTimeout", "startToCloseTimeout"}

func loaded(t *testing.T, name string) *umpirespb.Model {
	t.Helper()
	m, err := umpiremodel.Load(filepath.Join("..", "..", "..", "model", "ir", name+".json"))
	require.NoError(t, err)
	return m
}

func nexusIdentity(query string) cp.Identity {
	return cp.IdentityFor("temporal.case", "nexusCallerTests", query)
}

func lowered(t *testing.T, p *Producer, query string) *testpilotspb.Case {
	t.Helper()
	l, err := p.Lower(query, nexusIdentity(query))
	require.NoError(t, err)
	require.Equal(t, Lowered, l.Standing)
	require.Empty(t, l.Unsupported)
	require.NotNil(t, l.Case)
	return l.Case
}

// The seven functional Queries of the Nexus caller Model are every find Query its IR declares, so a
// test over this list leaves none out.
func TestTheFunctionalSetIsEveryFindQueryOfTheNexusCallerModel(t *testing.T) {
	var declared []string
	for _, q := range loaded(t, "nexus-caller").GetQueries() {
		if q.GetForm() == umpirespb.Query_FORM_FIND {
			declared = append(declared, q.GetName())
		}
	}
	require.ElementsMatch(t, functionalQueries, declared)
}

// Every lowered Case decodes strictly and prepares, unchanged, under the Profile derived from it:
// what a black-box consumer does with a Case. The environment names the handler's own task queue,
// which the realization binds apart from the caller workflow's.
func TestLoweredCasesPrepareUnderTheirDerivedProfile(t *testing.T) {
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	p, err := NewProducer(loaded(t, "nexus-caller"))
	require.NoError(t, err)
	for _, query := range functionalQueries {
		t.Run(query, func(t *testing.T) {
			encoded, err := protojson.Marshal(lowered(t, p, query))
			require.NoError(t, err)
			source, err := testpilot.DecodeCaseProtoJSON(encoded)
			require.NoError(t, err)
			profile, err := temporal.DeriveProfile(source, catalog, temporal.Environment{
				Identity: query + "-profile", Namespace: "namespace", TaskQueue: "task-queue",
				HandlerTaskQueue: "task-queue-handler", NexusEndpoint: "nexus-endpoint"})
			require.NoError(t, err)
			prepared, err := testpilot.Prepare(source, profile)
			require.NoError(t, err)
			require.True(t, proto.Equal(source, prepared.Snapshot()), "preparation carries the Case unchanged")
		})
	}
}

// Identical generation inputs give identical bytes: the Model read twice, lowered by two producers.
func TestLoweringIsDeterministic(t *testing.T) {
	bytesOf := func(t *testing.T, query string) []byte {
		p, err := NewProducer(loaded(t, "nexus-caller"))
		require.NoError(t, err)
		encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(lowered(t, p, query))
		require.NoError(t, err)
		return encoded
	}
	for _, query := range functionalQueries {
		t.Run(query, func(t *testing.T) {
			first := bytesOf(t, query)
			require.NotEmpty(t, first)
			require.Equal(t, first, bytesOf(t, query))
		})
	}
}

func instructionIDs(c *testpilotspb.Case) map[string][]string {
	out := map[string][]string{}
	for _, e := range c.GetProgram().GetEntrypoints() {
		ids := []string{}
		for _, n := range e.GetInstructions() {
			ids = append(ids, n.GetInstructionId())
		}
		out[e.GetEntrypointId()] = ids
	}
	return out
}

func instruction(t *testing.T, c *testpilotspb.Case, entrypoint, id string) *testpilotspb.InstructionNode {
	t.Helper()
	for _, e := range c.GetProgram().GetEntrypoints() {
		for _, n := range e.GetInstructions() {
			if e.GetEntrypointId() == entrypoint && n.GetInstructionId() == id {
				return n
			}
		}
	}
	require.FailNow(t, "no instruction "+entrypoint+"/"+id)
	return nil
}

// What each Case carries is read off the Scala: the scripts of Realization.scala with the classes
// each Scenario of Claims.scala pins placed in them, and the clauses each Property fixes, bounded by
// where the Scenario places the Property's action.
func TestALoweredCaseCarriesWhatTheScalaDeclares(t *testing.T) {
	const property = "temporal.nexus.caller.property."
	workflow := []string{"start-nexus-operation", "await-nexus-operation", "finish-workflow"}
	controller := func(between ...string) []string {
		return append(append([]string{"start-workflow", "await-scheduled"}, between...), "await-close", "history")
	}
	type rule struct {
		id    string
		bound int64
	}
	cases := []struct {
		query      string
		controller []string
		handler    []string
		rules      []rule
	}{
		{"syncCompletion", controller(), []string{"respond-sync"}, []rule{{"syncSucceeds.fact-nexusOperationCompleted", 1}}},
		{"asyncCompletion", controller("await-completion-authority", "complete-nexus-operation"), []string{"respond-async"},
			[]rule{{"completionSucceeds.fact-nexusOperationCompleted", 2}}},
		{"asyncFailure", controller("await-completion-authority", "fail-nexus-operation"), []string{"respond-async"},
			[]rule{{"completionFails.fact-nexusOperationFailed", 2}}},
		{"handlerError", controller(), []string{"respond-error"}, []rule{{"handlerErrorFails.fact-nexusOperationFailed", 1}}},
		{"retry", controller("pending-attempts"), []string{"respond-error-retryable", "respond-sync"},
			[]rule{{"retrySucceeds.fact-nexusOperationCompleted", 3}, {"retrySucceeds.state-succeeded-1-unset-unset-unset", 3}}},
		{"scheduleToStartTimeout", append([]string{"stop-handler-worker"}, controller()...), []string{},
			[]rule{{"scheduleToStartFires.fact-nexusOperationTimedOut-scheduleToStart", 2}}},
		{"startToCloseTimeout", controller(), []string{"respond-async"},
			[]rule{{"startToCloseFires.fact-nexusOperationTimedOut-startToClose", 2}}},
	}
	p, err := NewProducer(loaded(t, "nexus-caller"))
	require.NoError(t, err)
	for _, c := range cases {
		t.Run(c.query, func(t *testing.T) {
			lowered := lowered(t, p, c.query)
			require.Equal(t, map[string][]string{"controller": c.controller, "workflow": workflow, "handler": c.handler},
				instructionIDs(lowered))
			var rules []rule
			bounds := map[string]int64{}
			for _, r := range lowered.GetContract().GetCorrelated().GetRules() {
				bounds[r.GetRuleId()] = r.GetBound()
			}
			local := map[string]string{}
			for _, n := range lowered.GetProvenance().GetLocalNames() {
				local[n.GetDefinitionId()] = n.GetLocalName()
			}
			for _, b := range lowered.GetProvenance().GetCorrelatedRules() {
				name, ok := local[b.GetRuleId()]
				if !ok {
					name = b.GetRuleId()
				}
				rules = append(rules, rule{strings.TrimPrefix(b.GetRuleId(), property), bounds[name]})
			}
			require.Equal(t, c.rules, rules)
			require.Equal(t, "temporal.case.nexusCallerTests."+c.query, lowered.GetCaseId())
			// Each Case starts a workflow type of its own: the Name the Scala scopes to the Case's fixture.
			workflowType := "umpire-nexusCallerTests-" + c.query + "-workflow"
			require.Equal(t, workflowType, lowered.GetProgram().GetEntrypoints()[1].GetWorkflow().GetWorkflowType())
			started := instruction(t, lowered, "controller", "start-workflow").GetInstruction().GetInvokeRpc().GetRequestAssignments()
			require.Equal(t, "workflow_type.name", started[2].GetTarget())
			require.Equal(t, workflowType, started[2].GetValue().GetLiteral().GetTextValue())
			require.Equal(t, "temporal.nexus.caller.testpilot", lowered.GetProvenance().GetProducerId())
		})
	}
}

// A field the Scala does not set stays unset in the Case: a deadline is present exactly where the
// class the Scenario pins sets it.
func TestAWrittenMessageKeepsItsPresence(t *testing.T) {
	p, err := NewProducer(loaded(t, "nexus-caller"))
	require.NoError(t, err)
	schedule := func(query string) *commandpb.ScheduleNexusOperationCommandAttributes {
		node := instruction(t, lowered(t, p, query), "workflow", "start-nexus-operation")
		return node.GetInstruction().GetWorkflowCommand().GetCommand().GetScheduleNexusOperationCommandAttributes()
	}
	type deadlines struct{ scheduleToStart, startToClose, scheduleToClose bool }
	of := func(a *commandpb.ScheduleNexusOperationCommandAttributes) deadlines {
		return deadlines{a.GetScheduleToStartTimeout() != nil, a.GetStartToCloseTimeout() != nil, a.GetScheduleToCloseTimeout() != nil}
	}
	require.Equal(t, deadlines{}, of(schedule("syncCompletion")))
	require.Equal(t, deadlines{scheduleToStart: true}, of(schedule("scheduleToStartTimeout")))
	require.Equal(t, deadlines{startToClose: true}, of(schedule("startToCloseTimeout")))
	require.EqualValues(t, 2, schedule("scheduleToStartTimeout").GetScheduleToStartTimeout().GetSeconds())
	require.Equal(t, "temporal.nexus-endpoint", schedule("syncCompletion").GetEndpoint())
	require.Equal(t, map[string][]byte{"encoding": []byte("json/plain")}, schedule("syncCompletion").GetInput().GetMetadata())
}

// The handle an asynchronous reply binds is one learned value: the handler binds it, the controller
// waits for it and the completion reads it.
func TestALearnedHandleIsBoundOnceAndReadByItsDependents(t *testing.T) {
	p, err := NewProducer(loaded(t, "nexus-caller"))
	require.NoError(t, err)
	c := lowered(t, p, "asyncCompletion")
	require.Len(t, c.GetProgram().GetSlots(), 1)
	require.Equal(t, "completion-authority", c.GetProgram().GetSlots()[0].GetSlotId())
	require.NotNil(t, c.GetProgram().GetSlots()[0].GetOpaqueHandle())
	const slot = "completion-authority"
	require.Equal(t, slot, instruction(t, c, "handler", "respond-async").GetInstruction().GetNexusHandlerReply().GetHandleSlotId())
	require.Equal(t, slot, instruction(t, c, "controller", "await-completion-authority").GetInstruction().GetAwaitSlot().GetSlotId())
	require.Equal(t, slot, instruction(t, c, "controller", "complete-nexus-operation").GetInstruction().GetNexusOperationCompletion().GetHandleSlotId())
}

// onPathKinds is, for each functional Query, the evidence kinds its Case confirms: those a step of its
// path records.
var onPathKinds = map[string][]string{
	"syncCompletion":         {"scheduled", "completed"},
	"asyncCompletion":        {"scheduled", "started", "completed"},
	"asyncFailure":           {"scheduled", "started", "failed"},
	"handlerError":           {"scheduled", "failed"},
	"retry":                  {"scheduled", "pendingAttempts", "completed"},
	"scheduleToStartTimeout": {"scheduled", "timedOut"},
	"startToCloseTimeout":    {"scheduled", "started", "timedOut"},
}

// offPathKinds is, for each functional Query, the history kinds its Case carries off its path: the
// five history events less the ones a step of the path records.
var offPathKinds = map[string][]string{
	"syncCompletion":         {"started", "failed", "canceled", "timedOut"},
	"asyncCompletion":        {"failed", "canceled", "timedOut"},
	"asyncFailure":           {"completed", "canceled", "timedOut"},
	"handlerError":           {"started", "completed", "canceled", "timedOut"},
	"retry":                  {"started", "failed", "canceled", "timedOut"},
	"scheduleToStartTimeout": {"started", "completed", "failed", "canceled"},
	"startToCloseTimeout":    {"completed", "failed", "canceled"},
}

// The Scala realization declares the five history kinds exhaustive (Realization.scala,
// historySource), so a Case confirms the kinds its path records and carries every other of the five,
// declared, lifted by the history read and given no meaning in the Contract.
func TestALoweredCaseDeclaresTheHistoryKindsOffItsPath(t *testing.T) {
	history := []string{"started", "completed", "failed", "canceled", "timedOut"}
	p, err := NewProducer(loaded(t, "nexus-caller"))
	require.NoError(t, err)
	for _, query := range functionalQueries {
		t.Run(query, func(t *testing.T) {
			c := lowered(t, p, query)
			names := definitions(c)
			kinds := map[testpilotspb.CorrelatedEvidenceMeaning][]string{}
			for _, rule := range c.GetContract().GetCorrelated().GetProjectionRules() {
				kinds[rule.GetMeaning()] = append(kinds[rule.GetMeaning()],
					strings.TrimPrefix(defined(names, rule.GetKind()), "temporal.nexus.caller.evidence."))
			}
			require.Len(t, kinds, 2)
			confirmed, off := kinds[testpilotspb.CORRELATED_EVIDENCE_MEANING_CONFIRMED], kinds[testpilotspb.CORRELATED_EVIDENCE_MEANING_IRRELEVANT]
			require.ElementsMatch(t, onPathKinds[query], confirmed)
			require.ElementsMatch(t, offPathKinds[query], off)
			for _, kind := range history {
				require.NotEqual(t, slices.Contains(confirmed, kind), slices.Contains(off, kind), "%s is confirmed or off the path", kind)
			}
			require.Len(t, c.GetProgram().GetEvidence(), len(confirmed)+len(off))
		})
	}
}

// What a realization writes is checked against the protobuf descriptors it names, at the Scala line
// that wrote it, before any Case exists.
func TestADescriptorARealizationCrossesIsRejectedWhereItIsWritten(t *testing.T) {
	script := func(r *umpirespb.Realization, id string) *umpirespb.Script {
		for _, s := range r.GetScripts() {
			if s.GetId() == id {
				return s
			}
		}
		return nil
	}
	command := func(r *umpirespb.Realization, scriptID, id string) *umpirespb.Command {
		for _, item := range script(r, scriptID).GetItems() {
			if item.GetCommand().GetId() == id {
				return item.GetCommand()
			}
			for _, p := range item.GetPerforms() {
				if p.GetCommand().GetId() == id {
					return p.GetCommand()
				}
			}
		}
		return nil
	}
	schedule := func(r *umpirespb.Realization) *umpirespb.Proto {
		return command(r, "workflow", "start-nexus-operation").GetWorkflowCommand().GetCommand()
	}
	attributes := func(r *umpirespb.Realization) *umpirespb.Proto {
		return schedule(r).GetFields()[1].GetValue().GetMessage()
	}
	text := func(s string) *umpirespb.Operand {
		return &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_Text{Text: s}}}}
	}
	cases := []struct {
		name   string
		mutate func(r *umpirespb.Realization)
		want   string
	}{
		{"a message the descriptors do not have", func(r *umpirespb.Realization) { schedule(r).Message = "temporal.api.command.v1.Nope" },
			"no protobuf message temporal.api.command.v1.Nope"},
		{"a message written where another belongs", func(r *umpirespb.Realization) { attributes(r).Message = "temporal.api.common.v1.Payload" },
			"temporal.api.common.v1.Payload is written where a temporal.api.command.v1.ScheduleNexusOperationCommandAttributes belongs"},
		{"a field the message does not have", func(r *umpirespb.Realization) { attributes(r).GetFields()[0].Name = "endpont" },
			"temporal.api.command.v1.ScheduleNexusOperationCommandAttributes has no field endpont"},
		{"a value of another kind than the field", func(r *umpirespb.Realization) {
			attributes(r).GetFields()[1].Value = &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_Flag{Flag: true}}
		}, "temporal.api.command.v1.ScheduleNexusOperationCommandAttributes.service is of kind string, and a flag is written into it"},
		{"an enum value the enum does not have", func(r *umpirespb.Realization) {
			schedule(r).GetFields()[0].Value = &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_EnumName{EnumName: "COMMAND_TYPE_NOPE"}}
		}, "temporal.api.enums.v1.CommandType has no value COMMAND_TYPE_NOPE"},
		{"one value written into a map", func(r *umpirespb.Realization) { attributes(r).GetFields()[1].Name = "nexus_header" },
			"temporal.api.command.v1.ScheduleNexusOperationCommandAttributes.nexus_header holds several values, and one is written into it"},
		{"a role written into a field that is no text", func(r *umpirespb.Realization) {
			attributes(r).GetFields()[0].Name = "schedule_to_close_timeout"
		}, "temporal.api.command.v1.ScheduleNexusOperationCommandAttributes.schedule_to_close_timeout is of kind message, and a role is written into it"},
		{"a history event kind the event does not have", func(r *umpirespb.Realization) {
			r.Evidence[2].From = &umpirespb.Evidence_History{History: "nexus_operation_finished_event_attributes"}
		}, "evidence temporal.nexus.caller.evidence.completed: a history event has no attributes nexus_operation_finished_event_attributes"},
		{"an operation key the recorded message does not have", func(r *umpirespb.Realization) { r.Evidence[0].Operation = "event_number" },
			"temporal.api.history.v1.HistoryEvent has no field event_number"},
		{"an operation key that is a message", func(r *umpirespb.Realization) { r.Evidence[0].Operation = "event_time" },
			"evidence temporal.nexus.caller.evidence.scheduled keys its operation by event_time, which is no single scalar of temporal.api.history.v1.HistoryEvent"},
		{"evidence read from one value", func(r *umpirespb.Realization) { r.Evidence[0].GetRead().Path = "history" },
			"evidence temporal.nexus.caller.evidence.scheduled is read from history, which is no repeated message"},
		{"a method the service does not have", func(r *umpirespb.Realization) {
			command(r, "controller", "start-workflow").GetRpc().Method = "/temporal.api.workflowservice.v1.WorkflowService/BeginWorkflowExecution"
		}, "temporal.api.workflowservice.v1.WorkflowService has no method BeginWorkflowExecution"},
		{"an observation of a message the descriptors do not have", func(r *umpirespb.Realization) {
			r.Observations[0].Message = "temporal.api.history.v1.HistoricEvent"
		}, "no protobuf message temporal.api.history.v1.HistoricEvent"},
		{"a value observed into an observation of another message", func(r *umpirespb.Realization) {
			r.Observations[0].Message = "temporal.api.common.v1.Payload"
		}, "command history observes history.events[*] into history-event, which is a temporal.api.common.v1.Payload, and the path reads temporal.api.history.v1.History.events"},
		{"an assignment to a field the request does not have", func(r *umpirespb.Realization) {
			command(r, "controller", "start-workflow").GetRpc().GetAssign()[1].Target = "workflow_name"
		}, "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest has no field workflow_name"},
		{"a text assigned to a number", func(r *umpirespb.Realization) {
			command(r, "controller", "await-close").GetRpc().GetAssign()[2].Value = text("many")
		}, "command await-close assigns a text to temporal.api.workflowservice.v1.GetWorkflowExecutionHistoryRequest.maximum_page_size, which is of kind int32"},
		{"a poll condition over a field the element does not have", func(r *umpirespb.Realization) {
			command(r, "controller", "pending-attempts").GetPoll().GetUntil().GetEqual().GetLeft().GetPath().Path = "attempts"
		}, "temporal.api.workflow.v1.PendingNexusOperationInfo has no field attempts"},
		{"a reply that is no reply", func(r *umpirespb.Realization) {
			command(r, "handler", "respond-sync").GetNexusReply().GetReply().Message = "temporal.api.common.v1.Payload"
		}, "command respond-sync answers with temporal.api.common.v1.Payload; a Nexus handler answers with a start response or a handler error"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := loaded(t, "nexus-caller")
			c.mutate(m.GetRealizations()[0])
			p, err := NewProducer(m)
			require.NoError(t, err)
			_, err = p.Lower("retry", nexusIdentity("retry"))
			require.ErrorContains(t, err, c.want)
			var located *umpiremodel.Error
			require.ErrorAs(t, err, &located)
			// A poll the realization waits in through the kit's await is placed where the kit writes it.
			requireDeclaredIn(t, located.Position, realizationAt)
		})
	}
}

// Every Query of a Model has a standing, so none is left unlowered without a reason: a verify Query
// is searched and never realized, and a find Query lowers only through a realization of its machine.
// The activity's system designs and the Nexus close designs declare no realization yet; the standing
// of each Query of the activity Model, which declares one, is in activity_cases_test.go.
func TestEveryQueryHasAStanding(t *testing.T) {
	cases := []struct {
		model              string
		verify, unrealized int
	}{
		// The capabilities' generated verifies replace each design's and composition's two `any` ones,
		// and add the product's three laws; the designs waive closed rejection.
		{"activity-system", 64, 23},
		{"nexus-close", 84, 72},
	}
	for _, c := range cases {
		t.Run(c.model, func(t *testing.T) {
			m := loaded(t, c.model)
			require.Empty(t, m.GetRealizations())
			p, err := NewProducer(m)
			require.NoError(t, err)
			counts := map[Standing]int{}
			for _, q := range m.GetQueries() {
				l, err := p.Lower(q.GetName(), cp.IdentityFor("temporal.case", c.model, q.GetName()))
				require.NoError(t, err)
				want := NothingToRealize
				if q.GetForm() == umpirespb.Query_FORM_FIND {
					want = NoRealization
				}
				require.Equal(t, &Lowering{Standing: want}, l, q.GetName())
				counts[l.Standing]++
			}
			require.Equal(t, map[Standing]int{NothingToRealize: c.verify, NoRealization: c.unrealized}, counts)
		})
	}
	p, err := NewProducer(loaded(t, "nexus-caller"))
	require.NoError(t, err)
	_, err = p.Lower("nope", nexusIdentity("nope"))
	var located *umpiremodel.Error
	require.ErrorAs(t, err, &located)
	require.Equal(t, umpiremodel.Error{Position: loaded(t, "nexus-caller").GetSource(), Message: "no Query nope"}, *located)
}

// A read lifts the history kinds a path's rules confirm. A path that records no history kind lifts
// nothing: a lift with no rule is a Case preparation rejects.
func TestALiftCarriesTheHistoryKindsOfThePathOrIsLeftOut(t *testing.T) {
	observed := cp.ObservationTarget("history-event")
	node := cp.Node("history", cp.InvokeRPC("service", "/method", nil, []*testpilotspb.ResponseRead{
		cp.ResponseRead("history.events[*]", testpilotspb.READ_CARDINALITY_EMIT_EACH, observed,
			&testpilotspb.ReadTarget{Target: &testpilotspb.ReadTarget_CorrelatedEvidence{
				CorrelatedEvidence: &testpilotspb.CorrelatedEvidenceProjection{ObservationId: "correlated-evidence"}}})}))
	read := &cp.EvidenceSource{KindID: "polled", Recorded: cp.Recorded{Method: "/method", Path: "items"}}
	recorded := &cp.EvidenceSource{KindID: "completed", Recorded: cp.Recorded{HistoryAttributes: "completed_attributes"}}
	targets := func(rules ...cp.EvidenceRule) []*testpilotspb.ReadTarget {
		lowered := built{node}.with("history-2", rules)
		require.Equal(t, "history-2", lowered.GetInstructionId())
		return lowered.GetInstruction().GetInvokeRpc().GetResponseReads()[0].GetTargets()
	}
	require.Empty(t, cmp.Diff([]*testpilotspb.ReadTarget{observed}, targets(), protocmp.Transform()))
	require.Empty(t, cmp.Diff([]*testpilotspb.ReadTarget{observed}, targets(cp.EvidenceRule{Source: read}), protocmp.Transform()))
	rules := []cp.EvidenceRule{{Source: read}, {Source: recorded}}
	require.Empty(t, cmp.Diff([]*testpilotspb.ReadTarget{observed, cp.EvidenceTarget("correlated-evidence", rules)}, targets(rules...), protocmp.Transform()))
	require.Equal(t, "history", node.GetInstructionId(), "the declared command is not changed")
}

// One declaration that crosses a descriptor hides none after it: an observation, a kind of evidence
// and a command that each name what the descriptors do not have are all reported, each where it was
// written.
func TestEveryDescriptorARealizationCrossesIsReported(t *testing.T) {
	m := loaded(t, "nexus-caller")
	r := m.GetRealizations()[0]
	r.Observations[1].Message = "temporal.server.api.testpilot.v1.Nope"
	r.Evidence[2].From = &umpirespb.Evidence_History{History: "nexus_operation_begun_event_attributes"}
	for _, s := range r.GetScripts() {
		for _, item := range s.GetItems() {
			switch item.GetCommand().GetId() {
			case "start-workflow":
				item.GetCommand().GetRpc().Method = "/temporal.api.workflowservice.v1.WorkflowService/BeginWorkflowExecution"
			case "await-close":
				item.GetCommand().GetRpc().GetAssign()[0].Target = "name_space"
			default:
			}
		}
	}
	p, err := NewProducer(m)
	require.NoError(t, err)
	l, err := p.Lower("syncCompletion", nexusIdentity("syncCompletion"))
	require.Nil(t, l)
	for _, want := range []string{
		"no protobuf message temporal.server.api.testpilot.v1.Nope",
		"a history event has no attributes nexus_operation_begun_event_attributes",
		"temporal.api.workflowservice.v1.WorkflowService has no method BeginWorkflowExecution",
		"temporal.api.workflowservice.v1.GetWorkflowExecutionHistoryRequest has no field name_space",
	} {
		require.ErrorContains(t, err, want)
	}
	// The kind of evidence that crosses its descriptor is reported as that, once: the producer is not
	// asked about a realization that is not all there, so the path's fact is not also said to have no
	// evidence.
	require.NotContains(t, err.Error(), "evidence.kind-unknown")
}
