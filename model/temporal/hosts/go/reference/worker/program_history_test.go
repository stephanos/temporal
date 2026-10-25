package worker

import (
	"testing"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	deploymentpb "go.temporal.io/api/deployment/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

func historyAny(t *testing.T, message proto.Message) *anypb.Any {
	t.Helper()
	packed, err := anypb.New(message)
	require.NoError(t, err)
	return packed
}
func historyKeyFor(bits ...bool) *temporalv1.ActivityTaskKey {
	return &temporalv1.ActivityTaskKey{Scenario: "history", Attempt: 2, ActivityType: "activity", HeartbeatHistory: bits}
}
func historyProgram(t *testing.T) *temporalv1.WorkerProgram {
	t.Helper()
	heartbeat := historyAny(t, &workflowservice.RecordActivityTaskHeartbeatRequest{Details: &commonpb.Payloads{Payloads: []*commonpb.Payload{{Data: []byte("heartbeat")}}}})
	complete := historyAny(t, &workflowservice.RespondActivityTaskCompletedRequest{Result: &commonpb.Payloads{Payloads: []*commonpb.Payload{{Data: []byte("complete")}}}})
	canceled := historyAny(t, &workflowservice.RespondActivityTaskCanceledRequest{})
	p := &temporalv1.WorkerProgram{Namespace: "ns", TaskQueue: "queue"}
	for _, row := range []struct {
		history  []bool
		response *anypb.Any
	}{
		{response: heartbeat}, {history: []bool{false}, response: heartbeat}, {history: []bool{true}, response: complete},
		{history: []bool{false, false}, response: complete}, {history: []bool{false, true}, response: heartbeat},
		{history: []bool{false, true, true}, response: canceled},
	} {
		p.Rules = append(p.Rules, &temporalv1.Rule{Key: historyAny(t, historyKeyFor(row.history...)), Respond: []*anypb.Any{row.response}})
	}
	domains := []*temporalv1.ActivityTaskSuccessors{
		{Key: historyKeyFor(), CancelRequested: []bool{false, true}},
		{Key: historyKeyFor(false), CancelRequested: []bool{false, true}},
		{Key: historyKeyFor(false, true), CancelRequested: []bool{true}},
	}
	for _, domain := range domains {
		p.ContinuationDomains = append(p.ContinuationDomains, historyAny(t, domain))
	}
	return p
}

func changeDomain(t *testing.T, p *temporalv1.WorkerProgram, index int, edit func(*temporalv1.ActivityTaskSuccessors)) {
	t.Helper()
	domain := new(temporalv1.ActivityTaskSuccessors)
	require.NoError(t, p.ContinuationDomains[index].UnmarshalTo(domain))
	edit(domain)
	p.ContinuationDomains[index] = historyAny(t, domain)
}

func TestProgramExactHistoryAndDeclaredSingletonDomain(t *testing.T) {
	p := historyProgram(t)
	table, err := Decode(programBytes(t, p))
	require.NoError(t, err)
	require.Equal(t, 3, table.MaxHeartbeats())
	for _, row := range p.GetRules() {
		key := new(temporalv1.ActivityTaskKey)
		require.NoError(t, row.GetKey().UnmarshalTo(key))
		got := table.Match(key.GetScenario(), key.GetAttempt(), key.GetActivityType(), key.GetHeartbeatHistory()...)
		require.True(t, proto.Equal(row.GetRespond()[0], got))
	}
	require.Nil(t, table.Match("history", 2, "activity", false, true, false))
	rows := table.Rows()
	rows[0].Key.Scenario = "changed"
	rows[0].Response.Value = nil
	p.Rules[0].Respond[0].Value = nil
	require.NotNil(t, table.Match("history", 2, "activity"))
	require.NotEmpty(t, table.Match("history", 2, "activity").GetValue())
}

func TestProgramRejectsIncompleteOrAmbiguousHistory(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*temporalv1.WorkerProgram)
		want string
	}{
		{name: "duplicate-key", edit: func(p *temporalv1.WorkerProgram) { p.Rules = append(p.Rules, p.Rules[0]) }, want: "duplicate exact"},
		{name: "missing-domain", edit: func(p *temporalv1.WorkerProgram) { p.ContinuationDomains = p.ContinuationDomains[1:] }, want: "no declared"},
		{name: "empty-domain", edit: func(p *temporalv1.WorkerProgram) {
			changeDomain(t, p, 0, func(domain *temporalv1.ActivityTaskSuccessors) { domain.CancelRequested = nil })
		}, want: "nonempty unique"},
		{name: "duplicate-bit", edit: func(p *temporalv1.WorkerProgram) {
			changeDomain(t, p, 0, func(domain *temporalv1.ActivityTaskSuccessors) { domain.CancelRequested = []bool{true, true} })
		}, want: "duplicate declared bit"},
		{name: "duplicate-domain", edit: func(p *temporalv1.WorkerProgram) {
			p.ContinuationDomains = append(p.ContinuationDomains, p.ContinuationDomains[0])
		}, want: "duplicate declared domain"},
		{name: "missing-declared-row", edit: func(p *temporalv1.WorkerProgram) { p.Rules = p.Rules[:len(p.Rules)-1] }, want: "has no response row"},
		{name: "orphan-parent", edit: func(p *temporalv1.WorkerProgram) {
			p.Rules = p.Rules[1:]
			p.ContinuationDomains = p.ContinuationDomains[1:]
		}, want: "no heartbeat parent"},
		{name: "terminal-domain", edit: func(p *temporalv1.WorkerProgram) { p.Rules[0].Respond = p.Rules[2].Respond }, want: "requires an exact heartbeat"},
		{name: "undeclared-child", edit: func(p *temporalv1.WorkerProgram) {
			changeDomain(t, p, 0, func(domain *temporalv1.ActivityTaskSuccessors) { domain.CancelRequested = []bool{true} })
		}, want: "outside its declared"},
		{name: "unknown-domain-type", edit: func(p *temporalv1.WorkerProgram) {
			p.ContinuationDomains[0] = historyAny(t, &workflowservice.RespondActivityTaskCompletedRequest{})
		}, want: "successor 0 domain"},
		{name: "malformed-domain", edit: func(p *temporalv1.WorkerProgram) { p.ContinuationDomains[0].Value = []byte{0xff} }, want: "successor 0 domain"},
		{name: "unknown-domain-field", edit: func(p *temporalv1.WorkerProgram) {
			changeDomain(t, p, 0, func(domain *temporalv1.ActivityTaskSuccessors) { domain.ProtoReflect().SetUnknown([]byte{0x18, 1}) })
		}, want: "unknown fields"},
		{name: "unknown-key-field", edit: func(p *temporalv1.WorkerProgram) {
			key := historyKeyFor()
			key.ProtoReflect().SetUnknown([]byte{0x28, 1})
			p.Rules[0].Key = historyAny(t, key)
		}, want: "unknown fields"},
		{name: "unknown-nested-response", edit: func(p *temporalv1.WorkerProgram) {
			payload := &commonpb.Payload{Data: []byte("x")}
			payload.ProtoReflect().SetUnknown([]byte{0xf8, 0x7, 1})
			p.Rules[2].Respond = []*anypb.Any{historyAny(t, &workflowservice.RespondActivityTaskCompletedRequest{Result: &commonpb.Payloads{Payloads: []*commonpb.Payload{payload}}})}
		}, want: "unknown fields"},
		{name: "unsupported-request-field", edit: func(p *temporalv1.WorkerProgram) {
			p.Rules[2].Respond = []*anypb.Any{historyAny(t, &workflowservice.RespondActivityTaskCompletedRequest{DeploymentOptions: &deploymentpb.WorkerDeploymentOptions{DeploymentName: "forbidden"}})}
		}, want: "unsupported activity response field"},
		{name: "zero-attempt", edit: func(p *temporalv1.WorkerProgram) {
			key := historyKeyFor()
			key.Attempt = 0
			p.Rules[0].Key = historyAny(t, key)
		}, want: "positive attempt"},
		{name: "missing-response", edit: func(p *temporalv1.WorkerProgram) { p.Rules[0].Respond = nil }, want: "one response"},
		{name: "multiple-responses", edit: func(p *temporalv1.WorkerProgram) {
			p.Rules[0].Respond = append(p.Rules[0].Respond, p.Rules[2].Respond[0])
		}, want: "one response"},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := historyProgram(t)
			test.edit(p)
			_, err := Decode(programBytes(t, p))
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestProgramFollowsExplicitBinaryDomainAfterTrueWithoutModelInference(t *testing.T) {
	p := historyProgram(t)
	changeDomain(t, p, 2, func(domain *temporalv1.ActivityTaskSuccessors) { domain.CancelRequested = []bool{false, true} })
	response := historyAny(t, &workflowservice.RespondActivityTaskCompletedRequest{})
	p.Rules = append(p.Rules, &temporalv1.Rule{Key: historyAny(t, historyKeyFor(false, true, false)), Respond: []*anypb.Any{response}})
	table, err := Decode(programBytes(t, p))
	require.NoError(t, err)
	require.True(t, proto.Equal(response, table.Match("history", 2, "activity", false, true, false)))
}
