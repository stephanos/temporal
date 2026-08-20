package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	chasmnexus "go.temporal.io/server/chasm/lib/nexusoperation"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/nexus/nexustest"
	"go.temporal.io/server/tests/testcore"
	"go.temporal.io/server/tests/umpire3/campaign"
	"go.temporal.io/server/tests/umpire3/compiler"
	"go.temporal.io/server/tests/umpire3/explore"
	umpire3fault "go.temporal.io/server/tests/umpire3/fault"
	"go.temporal.io/server/tests/umpire3/protocol"
	umpire3runtime "go.temporal.io/server/tests/umpire3/runtime"
	"go.temporal.io/server/tests/umpire3/wirecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

func (s *Umpire3TestSuite) TestProbeNexusGeneratedCompletion() {
	runUmpire3Behavior(s.T(), "ProbeNexusGeneratedCompletion", "")
}

func (s *Umpire3TestSuite) TestProbeNexusRejectedStart() {
	t := s.T()
	env := newNexusTestEnv(t, true,
		testcore.WithDynamicConfig(dynamicconfig.EnableChasm, true),
		testcore.WithDynamicConfig(chasmnexus.EnableChasmWorkflowOperations, true),
		testcore.WithDynamicConfig(chasmnexus.Enabled, true),
	)
	requestID := "umpire3-rejected-" + uuid.NewString()
	_, err := env.FrontendClient().StartNexusOperationExecution(t.Context(),
		&workflowservice.StartNexusOperationExecutionRequest{
			Namespace: env.Namespace().String(), OperationId: requestID,
			Endpoint: "umpire3-nonexistent-endpoint", Service: "service", Operation: "operation",
			RequestId: requestID, ScheduleToCloseTimeout: durationpb.New(10 * time.Second),
		})
	require.Error(t, err)
	require.Equal(t, codes.NotFound, serviceerror.ToStatus(err).Code())
	outcome, err := protocol.ClassifyOutcome(protocol.ClaimConforming, &protocol.TerminalEvidence{
		State: "rejected", Disposition: protocol.TerminalDispositionFailure,
		Reference: "request/" + requestID, EntityIdentity: requestID,
	})
	require.NoError(t, err)
	requireUmpire3RejectedStartBehaviorContract(t, "ProbeNexusRejectedStart", outcome)
}

func requireUmpire3RejectedStartBehaviorContract(
	t *testing.T,
	behavior string,
	outcome protocol.Outcome,
) {
	t.Helper()
	require.Contains(t, t.Name(), behavior)
	require.Equal(t, protocol.OutcomeDegraded, outcome.Kind)
	require.Equal(t, "rejected", outcome.Terminal)
}

// TestProbeNexusReflectedVariant retains descriptor-derived invalid-input coverage. The generated
// request mutation is sent to the real public API and retained only as digest-bound provenance.
func (s *Umpire3TestSuite) TestProbeNexusReflectedVariant() {
	t := s.T()
	env := newNexusTestEnv(t, true,
		testcore.WithDynamicConfig(dynamicconfig.EnableChasm, true),
		testcore.WithDynamicConfig(dynamicconfig.EnableCHASMCallbacks, true),
		testcore.WithDynamicConfig(chasmnexus.EnableChasmWorkflowOperations, true),
		testcore.WithDynamicConfig(chasmnexus.Enabled, true),
	)
	endpoint := env.createRandomExternalNexusServer(t.Context(), t, nexustest.Handler{
		OnStartOperation: func(
			context.Context,
			string,
			string,
			*nexus.LazyValue,
			nexus.StartOperationOptions,
		) (nexus.HandlerStartOperationResult[any], error) {
			return &nexus.HandlerStartOperationResultSync[any]{Value: "ok"}, nil
		},
	})
	mutation := requireUmpire3WireMutation(t,
		"temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest",
		"operation_id", wirecase.MutationEmptyString)
	result, err := wirecase.Drive(t.Context(), &workflowservice.StartNexusOperationExecutionRequest{
		Namespace: env.Namespace().String(), OperationId: "umpire3-reflected-" + uuid.NewString(),
		Endpoint: endpoint, Service: "service", Operation: "operation", RequestId: uuid.NewString(),
		ScheduleToCloseTimeout: durationpb.New(10 * time.Second),
		StartToCloseTimeout:    durationpb.New(time.Second),
	}, mutation, func(ctx context.Context, request proto.Message) (proto.Message, error) {
		return env.FrontendClient().StartNexusOperationExecution(ctx,
			request.(*workflowservice.StartNexusOperationExecutionRequest))
	})
	require.NoError(t, err)
	requireUmpire3WireBehaviorContract(t, "ProbeNexusReflectedVariant", result)
	require.Equal(t, wirecase.ResponseRejected, result.Response, "%+v", result)
	require.Equal(t, "InvalidArgument", result.Code)
	require.NotEmpty(t, result.Provenance.RequestDigest)
}

// TestProbeNexusReflectedDurationVariant sends a generated negative-duration case to the real
// standalone Nexus start API and binds the response to descriptor and request digests.
func (s *Umpire3TestSuite) TestProbeNexusReflectedDurationVariant() {
	t := s.T()
	env := newNexusTestEnv(t, true,
		testcore.WithDynamicConfig(dynamicconfig.EnableChasm, true),
		testcore.WithDynamicConfig(dynamicconfig.EnableCHASMCallbacks, true),
		testcore.WithDynamicConfig(chasmnexus.EnableChasmWorkflowOperations, true),
		testcore.WithDynamicConfig(chasmnexus.Enabled, true),
	)
	endpoint := env.createRandomExternalNexusServer(t.Context(), t, nexustest.Handler{
		OnStartOperation: func(
			context.Context,
			string,
			string,
			*nexus.LazyValue,
			nexus.StartOperationOptions,
		) (nexus.HandlerStartOperationResult[any], error) {
			return &nexus.HandlerStartOperationResultSync[any]{Value: "ok"}, nil
		},
	})
	mutation := requireUmpire3WireMutation(t,
		"temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest",
		"start_to_close_timeout", wirecase.MutationNegativeDuration)
	result, err := wirecase.Drive(t.Context(), &workflowservice.StartNexusOperationExecutionRequest{
		Namespace: env.Namespace().String(), OperationId: "umpire3-duration-" + uuid.NewString(),
		Endpoint: endpoint, Service: "service", Operation: "operation", RequestId: uuid.NewString(),
		ScheduleToCloseTimeout: durationpb.New(10 * time.Second),
		StartToCloseTimeout:    durationpb.New(time.Second),
	}, mutation, func(ctx context.Context, request proto.Message) (proto.Message, error) {
		return env.FrontendClient().StartNexusOperationExecution(ctx,
			request.(*workflowservice.StartNexusOperationExecutionRequest))
	})
	require.NoError(t, err)
	requireUmpire3WireBehaviorContract(t, "ProbeNexusReflectedDurationVariant", result)
	require.Equal(t, wirecase.ResponseRejected, result.Response, "%+v", result)
	require.Equal(t, "InvalidArgument", result.Code)
	require.NotEmpty(t, result.Provenance.RequestDigest)
}

func requireUmpire3WireMutation(
	t *testing.T,
	message string,
	field string,
	kind wirecase.MutationKind,
) wirecase.Mutation {
	t.Helper()
	mutations, err := wirecase.Catalog(message)
	require.NoError(t, err)
	for _, mutation := range mutations {
		if mutation.Field == field && mutation.Kind == kind {
			return mutation
		}
	}
	require.FailNow(t, "generated protobuf mutation is unavailable", "%s.%s/%s", message, field, kind)
	return wirecase.Mutation{}
}

func requireUmpire3WireBehaviorContract(t *testing.T, behavior string, result wirecase.Result) {
	t.Helper()
	require.Contains(t, t.Name(), behavior)
	require.NotEqual(t, wirecase.ResponseTransportFailed, result.Response)
	require.NotEmpty(t, result.Provenance.DescriptorDigest)
	require.NotEmpty(t, result.Provenance.RequestDigest)
}

func (s *Umpire3TestSuite) TestProbeNexusFaultAction() {
	runUmpire3Behavior(s.T(), "ProbeNexusFaultAction", "")
	term := umpire3DropTerm()
	s.NoError(umpire3fault.Preflight(term,
		[]protocol.CapabilityID{protocol.CapabilityIDFaultRpc}, false))
	realizer := &umpire3RootFaultRealizer{}
	s.NoError(umpire3fault.Run(context.Background(), term, realizer, umpire3fault.Options{
		Capabilities: []protocol.CapabilityID{protocol.CapabilityIDFaultRpc}, CleanupTimeout: time.Second,
	}, func(context.Context) error { return nil }))
	s.Equal([]string{"install", "activate", "release", "cleanup"}, realizer.events)
}

func (s *Umpire3TestSuite) TestProbeNexusResilience() {
	t := s.T()
	declared := umpire3DeclaredFootprint(t, protocol.ActionKindCloseNexusOperation)
	baselineFactory := &umpire3RequiredFootprintFactory{
		umpire3SDKRootFactory: newUmpire3SDKRootFactory(t, false), declared: declared,
	}
	baseline := evaluateUmpire3BehaviorIn(t, "ProbeNexusLearnedFootprint", "", baselineFactory)
	require.Equal(t, umpire3runtime.ClaimConforming, baseline.Claim.Kind, baseline.Claim.Reason)
	require.NotNil(t, baseline.Footprint)
	targets := umpire3fault.FaultTargets(baseline.Footprint.Calls, 17, 6)
	require.NotEmpty(t, targets)

	faultFactory := &umpire3RequiredFootprintFactory{
		umpire3SDKRootFactory: newUmpire3SDKRootFactory(t, false), declared: declared,
	}
	faulted := evaluateUmpire3BehaviorIn(t, "ProbeNexusResilience", "", faultFactory)
	require.Equal(t, umpire3runtime.ClaimConforming, faulted.Claim.Kind, faulted.Claim.Reason)
	require.NotNil(t, faulted.Footprint)
	require.Len(t, faulted.Faults, 1)
	require.True(t, faulted.Faults[0].Realized)
	require.True(t, faulted.Faults[0].CleanupComplete)
}

func (s *Umpire3TestSuite) TestProbeNexusDegraded() {
	result := evaluateUmpire3Behavior(s.T(), "ProbeNexusDegraded", "", false)
	s.Equal(umpire3runtime.ClaimConforming, result.Claim.Kind)
	s.Equal(umpire3runtime.OutcomeDegraded, result.Outcome.Kind)
	s.Equal("failed", result.Outcome.Terminal)
}

func (s *Umpire3TestSuite) TestProbeNexusFlagged() {
	result := evaluateUmpire3Behavior(s.T(), "ProbeNexusFlagged", "", false)
	s.Equal(umpire3runtime.ClaimViolating, result.Claim.Kind)
	s.Equal(umpire3runtime.OutcomeFlagged, result.Outcome.Kind)
	s.Empty(result.Outcome.Terminal)
}

func (s *Umpire3TestSuite) TestProbeNexusHTTPFaultSeam() {
	runUmpire3Behavior(s.T(), "ProbeNexusHTTPFaultSeam", "")
	selected := umpire3fault.SelectFootprints([]umpire3fault.Footprint{
		{Protocol: "http", Service: "nexus", Route: "/service/operation", Risk: 10},
	}, 17, 1)
	s.Equal([]umpire3fault.Footprint{{
		Protocol: "http", Service: "nexus", Route: "/service/operation", Risk: 10,
		RealizationEvidence: true,
	}}, selected)
}

func (s *Umpire3TestSuite) TestProbeNexusExploration() {
	t := s.T()
	observer := newUmpire3NexusLifecycleObserver(t)
	values, err := explore.NexusLifecycleValues()
	require.NoError(t, err)
	report, err := explore.Run(context.Background(), explore.Template{
		Identifier: "umpire3-root-nexus-exploration",
		Goal: explore.Goal{
			Kind: explore.GoalTransitionCoverage, Target: protocol.TargetIDFeatureNexus,
			Property: protocol.PropertyIDNexusOperationClosure,
		},
		Holes: []explore.Hole{{
			Identifier: "edge", Kind: explore.HoleAction, Values: values,
		}},
		Build: func(assignment explore.Assignment) (compiler.Scenario, error) {
			value := assignment["edge"]
			return umpire3AssuranceScenario("umpire3-explore-"+strings.ReplaceAll(value.Key, "/", "-"),
				protocol.TargetIDFeatureNexus, protocol.PropertyIDNexusOperationClosure,
				protocol.EntityKindNexusOperation, umpire3NexusCoverageAction(value.Text)), nil
		},
		Observe: func(ctx context.Context, candidate explore.Candidate) ([]string, error) {
			if len(candidate.Coverage) != 1 {
				return nil, fmt.Errorf("generated Nexus candidate declares %d edges, want 1", len(candidate.Coverage))
			}
			if err := observer.Observe(ctx, candidate.Coverage[0]); err != nil {
				return nil, fmt.Errorf("edge %s: %w", candidate.Coverage[0], err)
			}
			return candidate.Coverage, nil
		},
	}, explore.Bounds{MaxAssignments: len(values), Compiler: umpire3CompilerLimits()})
	require.NoError(t, err)
	requireUmpire3ExplorationBehaviorContract(t, "ProbeNexusExploration", report)
	require.True(t, report.Complete)
	require.True(t, report.Coverage.Complete)
	require.Len(t, report.Candidates, 17)
}

func requireUmpire3ExplorationBehaviorContract(
	t *testing.T,
	behavior string,
	report explore.Report,
) {
	t.Helper()
	require.Contains(t, t.Name(), behavior)
	require.Equal(t, protocol.TargetIDFeatureNexus, report.Coverage.Target)
	require.Equal(t, protocol.PropertyIDNexusOperationClosure, report.Coverage.Property)
}

func umpire3NexusCoverageAction(action string) protocol.ActionKind {
	switch action {
	case "schedule", "reject":
		return protocol.ActionKindScheduleOperation
	case "attempt-failed":
		return protocol.ActionKindRetryTask
	case "start":
		return protocol.ActionKindDispatchTask
	case "succeed":
		return protocol.ActionKindPersistSuccess
	case "fail", "terminate":
		return protocol.ActionKindCloseNexusOperation
	case "cancel":
		return protocol.ActionKindCommitCancellation
	case "timeout":
		return protocol.ActionKindTimeoutNexusOperation
	default:
		return ""
	}
}

func (s *Umpire3TestSuite) TestProbeWorkflowGenerated() {
	runUmpire3Behavior(s.T(), "ProbeWorkflowGenerated", "")
}

func (s *Umpire3TestSuite) TestProbeWorkflowContinueAsNew() {
	runUmpire3Behavior(s.T(), "ProbeWorkflowContinueAsNew", "")
}

func (s *Umpire3TestSuite) TestProbeWorkflowContinueAsNewGenerated() {
	runUmpire3Behavior(s.T(), "ProbeWorkflowContinueAsNewGenerated", "")
}

func (s *Umpire3TestSuite) TestProbeWorkflowReset() {
	runUmpire3Behavior(s.T(), "ProbeWorkflowReset", "")
}

func (s *Umpire3TestSuite) TestProbeNexusLearnedFootprint() {
	t := s.T()
	declared := umpire3DeclaredFootprint(t, protocol.ActionKindCloseNexusOperation)
	factory := &umpire3RequiredFootprintFactory{
		umpire3SDKRootFactory: newUmpire3SDKRootFactory(t, false), declared: declared,
	}
	result := evaluateUmpire3BehaviorIn(t, "ProbeNexusLearnedFootprint", "", factory)
	require.Equal(t, umpire3runtime.ClaimConforming, result.Claim.Kind, result.Claim.Reason)
	calls, digest := factory.learnedFootprint()
	require.NotEmpty(t, calls)
	require.NotEmpty(t, digest)
	require.Empty(t, umpire3fault.ReconcileFootprints(declared, calls, nil))
	require.NotNil(t, result.Footprint)
	require.True(t, result.Footprint.Complete)
	require.Equal(t, digest, result.Footprint.FootprintDigest)
	require.NotEmpty(t, result.Footprint.ReconciliationDigest)
	targets := umpire3fault.FaultTargets(calls, 99, 2)
	require.Len(t, targets, 2)
	require.Contains(t, targets, umpire3fault.Footprint{
		Protocol: "http", Service: "nexus", Route: "/service/operation", Risk: 8,
		RealizationEvidence: true,
	})
}

func umpire3DeclaredFootprint(t *testing.T, actionKind protocol.ActionKind) []umpire3fault.Footprint {
	t.Helper()
	catalog, err := protocol.DefaultCatalog()
	require.NoError(t, err)
	action, found := catalog.Action(string(actionKind))
	require.True(t, found)
	declared := make([]umpire3fault.Footprint, len(action.Footprint))
	for index, call := range action.Footprint {
		risk := 5
		if call.Protocol == "http" {
			risk = 8
		}
		declared[index] = umpire3fault.Footprint{
			Protocol: call.Protocol, Service: call.Service, Route: call.Route, Risk: risk,
		}
	}
	return declared
}

func (s *Umpire3TestSuite) TestProbeNexusCoverageGuidedFaults() {
	runUmpire3Behavior(s.T(), "ProbeNexusCoverageGuidedFaults", "")
	report := runUmpire3RootCampaign(s.T(), 23, 1)
	s.Len(report.Executions, 1)
	s.Len(report.Dropped, 1)
	s.Equal(campaign.DropBudget, report.Dropped[0].Reason)
}

func (s *Umpire3TestSuite) TestProbeNexusRandomized() {
	runUmpire3Behavior(s.T(), "ProbeNexusRandomized", "")
	first := runUmpire3RootCampaign(s.T(), 8675309, 2)
	second := runUmpire3RootCampaign(s.T(), 8675309, 2)
	s.Equal(deterministicCampaignView(first), deterministicCampaignView(second))
}

type umpire3CampaignExecution struct {
	CandidateID string
	Digest      string
	Coverage    []campaign.CoveragePoint
}

type umpire3CampaignView struct {
	CoverageBefore []campaign.CoveragePoint
	CoverageAfter  []campaign.CoveragePoint
	CoverageDelta  []campaign.CoveragePoint
	Executions     []umpire3CampaignExecution
	Dropped        []campaign.Dropped
}

func deterministicCampaignView(report campaign.Report) umpire3CampaignView {
	view := umpire3CampaignView{
		CoverageBefore: report.CoverageBefore, CoverageAfter: report.CoverageAfter,
		CoverageDelta: report.CoverageDelta, Dropped: report.Dropped,
	}
	for _, execution := range report.Executions {
		view.Executions = append(view.Executions, umpire3CampaignExecution{
			CandidateID: execution.CandidateID, Digest: execution.Digest, Coverage: execution.Coverage,
		})
	}
	return view
}

func runUmpire3RootCampaign(t *testing.T, seed int64, budget int) campaign.Report {
	t.Helper()
	candidates := []campaign.Candidate{
		{Identifier: "callback", Scenario: umpire3AssuranceScenario(
			"umpire3-campaign-callback", protocol.TargetIDIntegrationCallbackWorkflow,
			protocol.PropertyIDCallbackResponseConsistency, protocol.EntityKindCallback,
			protocol.ActionKindRecordCallbackResponse),
			Coverage: []campaign.CoveragePoint{{Kind: campaign.CoverageAction, Identifier: "record-callback-response"}}},
		{Identifier: "nexus", Scenario: umpire3AssuranceScenario(
			"umpire3-campaign-nexus", protocol.TargetIDFeatureNexus,
			protocol.PropertyIDNexusOperationClosure, protocol.EntityKindNexusOperation,
			protocol.ActionKindCloseNexusOperation),
			Coverage: []campaign.CoveragePoint{{Kind: campaign.CoverageAction, Identifier: "close-nexus-operation"}}},
	}
	report, err := campaign.Run(context.Background(), campaign.Request{
		Candidates: candidates, Seed: seed, Workers: 2, MaxExecutions: budget,
		CompilerLimits: umpire3CompilerLimits(),
		Executor: func(ctx context.Context, experiment protocol.Experiment) (umpire3runtime.Result, []campaign.CoveragePoint, error) {
			result, err := umpire3runtime.Run(ctx, umpire3runtime.Request{
				Experiment: experiment, Environment: newUmpire3RootEnvironment(t, false),
			})
			return result, []campaign.CoveragePoint{{Kind: campaign.CoverageAction, Identifier: experiment.Actions[0].Kind}}, err
		},
	})
	require.NoError(t, err)
	return report
}

func umpire3DescriptorHasField(t *testing.T, fullName string) bool {
	t.Helper()
	data, err := os.ReadFile("umpire3/model/Temporal/API/Generated/field-dispositions.json")
	require.NoError(t, err)
	var document struct {
		Messages []struct {
			Fields []struct {
				FullName    string `json:"fullName"`
				Disposition string `json:"disposition"`
			} `json:"fields"`
		} `json:"messages"`
	}
	require.NoError(t, json.Unmarshal(data, &document))
	for _, message := range document.Messages {
		for _, field := range message.Fields {
			if field.FullName == fullName {
				return field.Disposition != ""
			}
		}
	}
	return false
}

func umpire3DropTerm() umpire3fault.Term {
	return umpire3fault.Term{
		Kind: protocol.FaultKindDrop,
		Scope: umpire3fault.Scope{
			Namespaces: []string{"umpire3-root"}, Services: []string{"nexus"},
			Routes: []string{"/service/operation"},
		},
		Occurrence: umpire3fault.Occurrence{First: 1, Count: 1},
		Interval:   umpire3fault.Interval{Start: 1, Stop: 2},
	}
}

type umpire3RootFaultRealizer struct {
	events []string
}

func (r *umpire3RootFaultRealizer) Install(context.Context, umpire3fault.Term) (string, error) {
	r.events = append(r.events, "install")
	return "fault", nil
}

func (r *umpire3RootFaultRealizer) Activate(context.Context, string) error {
	r.events = append(r.events, "activate")
	return nil
}

func (r *umpire3RootFaultRealizer) Release(context.Context, string) error {
	r.events = append(r.events, "release")
	return nil
}

func (r *umpire3RootFaultRealizer) Cleanup(context.Context, string) error {
	r.events = append(r.events, "cleanup")
	return nil
}

func umpire3CompilerLimits() compiler.Limits {
	return compiler.Limits{
		MaxPaths: 8, MaxActions: 64, MaxStates: 1000, MaxMemoryBytes: 8 << 20, MaxTime: 5 * time.Second,
	}
}
