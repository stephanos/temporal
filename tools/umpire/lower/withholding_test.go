package lower

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	"google.golang.org/protobuf/proto"
)

func withheldTimeoutRetry(t *testing.T) *umpirespb.Model {
	t.Helper()
	m := loaded(t, "activity-standalone")
	timer := &umpirespb.ActionClass{Action: "temporal.features.activity.deadline.startToClose"}
	var stateFields []*umpirespb.Field
	for _, typ := range m.GetTypes() {
		if typ.GetName() == "temporal.features.activity.standalone.system.State" {
			stateFields = typ.GetRecord().GetFields()
		}
	}
	timeoutArgument := slices.IndexFunc(stateFields, func(field *umpirespb.Field) bool { return field.GetName() == "startToClose" })
	require.NotEqual(t, -1, timeoutArgument)
	hasMaxAttempts := slices.ContainsFunc(stateFields, func(field *umpirespb.Field) bool { return field.GetName() == "maxAttempts" })
	startTimeout := -1
	for _, action := range m.GetActions() {
		if action.GetId() == "temporal.features.activity.standalone.client.start" {
			startTimeout = slices.IndexFunc(action.GetInputs(), func(input *umpirespb.Param) bool { return input.GetName() == "startToClose" })
		}
	}
	require.NotEqual(t, -1, startTimeout)
	for _, scenario := range m.GetScenarios() {
		if scenario.GetName() == "retriedThenCompleted" {
			scenario.Actions[0].Inputs[startTimeout].GetEnum().Case = "expires"
			scenario.Actions[2] = proto.CloneOf(timer)
		}
	}
	for _, function := range m.GetFunctions() {
		if function.GetName() == "activitySystem.property.retryCompletes" {
			function.GetBody().GetBinary().GetLeft().GetBinary().GetRight().GetConstruct().Args[timeoutArgument].GetLiteral().GetEnum().Case = "expires"
		}
		// The checked-in fixture predates timeout retries; this isolates the bridge from model regeneration.
		if !hasMaxAttempts && function.GetName() == "activitySystem.rules.startToClose" {
			call := function.GetBody().GetIf().GetThen().GetCall()
			call.Function = "temporal.features.activity.standalone.system.ActivitySystem$.effects$.backOff"
			call.Args = call.GetArgs()[:1]
		}
	}
	r := realizationNamed(t, m, "standalone")
	script := scriptNamed(t, r, "attempts")
	script.Items = append([]*umpirespb.Item{{Position: script.GetPosition(), When: []*umpirespb.ActionClass{timer},
		Command: &umpirespb.Command{Id: "withhold-attempt", Position: script.GetPosition(), Instruction: &umpirespb.Command_AttemptWithheld{AttemptWithheld: &umpirespb.AttemptWithheld{}}}}}, script.GetItems()...)
	for _, step := range r.GetServerSteps() {
		if step.GetStep().GetAction() == timer.GetAction() {
			step.TimeoutBasis = umpirespb.TIMEOUT_BASIS_START_TO_CLOSE
		}
	}
	evidenceModel := &umpirespb.Model{Realizations: []*umpirespb.Realization{r}}
	first := evidenceOf(t, evidenceModel, "statusStarted")
	first.Confirms = []*umpirespb.Taking{{Step: proto.CloneOf(script.GetActivity().GetStarts()[0]), Occurrence: 1}}
	second := evidenceOf(t, evidenceModel, "attemptCount")
	second.Confirms[0].Step = proto.CloneOf(timer)
	controller := scriptNamed(t, r, "controller")
	controller.Items = slices.DeleteFunc(controller.GetItems(), func(item *umpirespb.Item) bool {
		return item.GetCommand().GetId() == "await-timed-out"
	})
	return m
}

func TestAWithheldTimeoutRetryLowersToTwoAttempts(t *testing.T) {
	p, err := NewProducer(withheldTimeoutRetry(t))
	require.NoError(t, err)
	a, _, err := p.ask("retry")
	require.NoError(t, err)
	path, problems := p.check(a, activityIdentity("retry"))
	require.Empty(t, problems)
	require.Empty(t, path.unanswered())
	require.Empty(t, path.late(), "the first delivery is recorded with withholding before the second delivery's timeout evidence")

	lowered, err := p.Lower("retry", activityIdentity("retry"))
	require.NoError(t, err)
	require.Equal(t, Lowered, lowered.Standing, "%v", lowered.Unsupported)
	require.Empty(t, lowered.Unsupported)
	c := lowered.Case
	require.Equal(t, map[string][]string{"controller": {"start-activity", "await-completed"}, "attempts": {"withhold-attempt", "complete-attempt"}}, instructionIDs(c))
	protorequire.ProtoEqual(t, &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptWithholding{ActivityAttemptWithholding: &testpilotspb.ActivityAttemptWithholding{}}},
		instruction(t, c, "attempts", "withhold-attempt").GetInstruction())
	require.NotNil(t, instruction(t, c, "attempts", "complete-attempt").GetInstruction().GetFinish())

	names := definitions(c)
	confirmed := map[string][]string{}
	for _, rule := range c.GetContract().GetCorrelated().GetProjectionRules() {
		kind := strings.TrimPrefix(defined(names, rule.GetKind()), activityEvidence)
		for _, output := range rule.GetOutputs() {
			confirmed[kind] = append(confirmed[kind], output.GetAction().GetValue())
		}
	}
	require.Equal(t, []string{"poll"}, confirmed["statusStarted"])
	require.Equal(t, []string{"startToClose", "backoff", "poll"}, confirmed["attemptCount"])
	require.NotContains(t, confirmed, "statusTimedOut", "a timeout that retries produces no terminal status evidence")
	var second *testpilotspb.EvidenceDeclaration
	for _, declaration := range c.GetProgram().GetEvidence() {
		if defined(names, declaration.GetEvidenceId()) == activityEvidence+"attemptCount" {
			second = declaration
		}
	}
	require.NotNil(t, second)
	require.NotNil(t, second.GetRunEvent())
	require.Len(t, second.GetRunEvent().GetGuard().GetAll().GetOperands(), 2)
	protorequire.ProtoEqual(t, cp.Equal(cp.Path(cp.ProjectedValue(), "activity_attempt.sdk_attempt"), cp.Literal(cp.SignedInteger(2))), second.GetRunEvent().GetGuard().GetAll().GetOperands()[1])
	preparedAsIs(t, c)
}

func TestAWithholdingCommandRequiresOneTimerOccurrence(t *testing.T) {
	p, err := NewProducer(withheldTimeoutRetry(t))
	require.NoError(t, err)
	a, _, err := p.ask("retry")
	require.NoError(t, err)
	path, problems := p.check(a, activityIdentity("retry"))
	require.Empty(t, problems)
	require.Empty(t, path.withholdingOccurrences())

	timer := path.adapter.classKey(&umpirespb.ActionClass{Action: "temporal.features.activity.deadline.startToClose"})
	path.keys = append(path.keys, timer)
	gaps := path.withholdingOccurrences()
	require.Len(t, gaps, 1)
	require.ErrorContains(t, gaps[0], "withholding command withhold-attempt requires exactly one occurrence")
	require.ErrorContains(t, gaps[0], "got 2")
	require.ErrorContains(t, gaps[0], activityRealizationAt+":")
}

func TestWithholdingRefusesAnUnprovenSDKContextTimeoutBasis(t *testing.T) {
	m := withheldTimeoutRetry(t)
	for _, step := range realizationNamed(t, m, "standalone").GetServerSteps() {
		step.TimeoutBasis = umpirespb.TIMEOUT_BASIS_UNSPECIFIED
	}
	_, err := NewProducer(m)
	require.ErrorContains(t, err, "timeout basis")
	require.ErrorContains(t, err, "withhold-attempt")
}
