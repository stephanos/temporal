package migration

import (
	"fmt"
	"slices"

	"go.temporal.io/server/tests/umpire3/compiler"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/regress"
)

type BehaviorContract struct {
	Behavior             string
	ModelTarget          string
	Properties           []string
	Entities             []string
	Actions              []string
	Faults               []string
	Relations            []string
	Variants             []string
	RequiredCapabilities []string
	ExpectedVerdict      string
	NegativeControl      string
	Evidence             []string
	Fidelity             protocol.Fidelity
	EvidenceLevel        protocol.EvidenceLevel
}

func behaviorContracts() map[string]BehaviorContract {
	contracts := []BehaviorContract{
		contract("PlanAndDriveKitchenSinkNexusOperation", "feature-nexus", "nexus-operation.closure",
			[]string{"nexus-operation", "workflow"}, []string{"schedule-operation", "worker-returns-success", "persist-success"},
			[]string{"participant-program"}, []string{"chasm"}, nexusCapabilities(), "conforming"),
		contract("PlanAndDriveKitchenSinkWorkflow", "foundation-delivery-safety", "entity.progress",
			[]string{"workflow"}, []string{"progress-entity"}, []string{"participant-program"}, nil,
			[]string{"history-observation", "workflow-task-control"}, "conforming"),
		contract("PlanAndDriveNexusOperationCHASM", "feature-nexus", "nexus-operation.closure",
			[]string{"nexus-operation", "workflow"}, []string{"schedule-operation", "worker-returns-success", "persist-success"},
			[]string{"terminal-state"}, []string{"chasm"}, nexusCapabilities(), "conforming"),
		contract("PlanAndDriveWorkflowToCompletion", "foundation-delivery-safety", "entity.progress",
			[]string{"workflow"}, []string{"progress-entity"}, []string{"terminal-state"}, nil,
			[]string{"history-observation", "workflow-task-control"}, "conforming"),
		contractWithFault("ProbeNexusCoverageGuidedFaults", "feature-nexus", "nexus-operation.closure",
			[]string{"close-nexus-operation"}, []string{"drop"}, "conforming"),
		contract("ProbeNexusDegraded", "feature-nexus", "nexus-operation.closure",
			[]string{"nexus-operation", "workflow"}, []string{"close-nexus-operation"},
			[]string{"operation-failure", "terminal-state"}, nil, nexusCapabilities(), "conforming"),
		contract("ProbeNexusExploration", "feature-nexus", "nexus-operation.closure",
			[]string{"nexus-operation"}, []string{"close-nexus-operation"}, []string{"bounded-exploration"}, nil,
			nexusCapabilities(), "conforming"),
		contractWithFault("ProbeNexusFaultAction", "feature-nexus", "nexus-operation.closure",
			[]string{"close-nexus-operation"}, []string{"hold-release"}, "conforming"),
		contract("ProbeNexusFlagged", "feature-nexus", "nexus-operation.closure",
			[]string{"nexus-operation", "workflow"}, []string{"close-nexus-operation"},
			[]string{"retryable-failure", "unreached-terminal", "liveness-violation"}, nil,
			nexusCapabilities(), "violating"),
		contract("ProbeNexusGeneratedCompletion", "feature-nexus", "nexus-operation.closure",
			[]string{"nexus-operation"}, []string{"close-nexus-operation"}, []string{"generated-completion"}, nil,
			nexusCapabilities(), "conforming"),
		contractWithFault("ProbeNexusHTTPFaultSeam", "feature-nexus", "nexus-operation.closure",
			[]string{"close-nexus-operation"}, []string{"hold-release"}, "conforming"),
		contract("ProbeNexusLearnedFootprint", "feature-nexus", "nexus-operation.closure",
			[]string{"nexus-operation"}, []string{"close-nexus-operation"}, []string{"learned-rpc-http-footprint"}, nil,
			nexusCapabilities(), "conforming"),
		contract("ProbeNexusRandomized", "feature-nexus", "nexus-operation.closure",
			[]string{"nexus-operation"}, []string{"close-nexus-operation"},
			[]string{"seeded-campaign"}, nil, nexusCapabilities(), "conforming"),
		contract("ProbeNexusReflectedDurationVariant", "feature-nexus", "nexus-operation.closure",
			[]string{"nexus-operation"}, []string{"schedule-operation"}, []string{"protobuf-duration"}, nil,
			nexusCapabilities(), "conforming"),
		contract("ProbeNexusReflectedVariant", "feature-nexus", "nexus-operation.closure",
			[]string{"nexus-operation"}, []string{"schedule-operation"}, []string{"protobuf-required-field"}, nil,
			nexusCapabilities(), "conforming"),
		contract("ProbeNexusRejectedStart", "feature-nexus", "nexus-operation.closure",
			[]string{"nexus-operation"}, []string{"schedule-operation"},
			[]string{"unknown-endpoint", "start-rejection", "terminal-state"}, nil,
			nexusCapabilities(), "conforming"),
		contractWithFault("ProbeNexusResilience", "feature-nexus", "nexus-operation.closure",
			[]string{"close-nexus-operation"}, []string{"drop"}, "conforming"),
		contract("ProbeWorkflowContinueAsNew", "foundation-routing-isolation", "workflow-run.continuation-lineage",
			[]string{"workflow-run"}, []string{"continue-workflow"}, []string{"continuation-lineage"}, nil,
			[]string{"history-observation"}, "conforming"),
		contract("ProbeWorkflowContinueAsNewGenerated", "foundation-routing-isolation", "workflow-task.routing-isolation",
			[]string{"workflow-run", "workflow-task"}, []string{"route-workflow-task", "continue-workflow"},
			[]string{"generated-continuation-lineage", "task-queue-route"}, nil,
			[]string{"history-observation"}, "conforming"),
		contract("ProbeWorkflowGenerated", "workflow-update-lifecycle", "workflow-update.accepted-completes-through-history",
			[]string{"workflow", "workflow-update"},
			[]string{"start-update", "dispatch-workflow-task", "accept-update", "record-update-history", "complete-workflow-task", "complete-update"},
			[]string{"history-completion"}, nil, []string{"history-observation", "update", "workflow-task-control"}, "conforming"),
		contract("ProbeWorkflowReset", "foundation-routing-isolation", "workflow-run.reset-lineage",
			[]string{"workflow-run"}, []string{"reset-workflow"}, []string{"reset-lineage"}, nil,
			[]string{"history-observation"}, "conforming"),
		contract("SparseRegressionBidirectionalNexusActivityLinks", "integration-nexus-activity", "nexus-activity.link-consistency",
			[]string{"nexus-operation", "activity"}, []string{"link-nexus-activity"},
			[]string{"nexus-to-activity", "activity-to-nexus"}, []string{"hsm", "chasm"},
			[]string{"history-observation"}, "conforming"),
		contract("SparseRegressionCallbackAfterCallerCompletion", "integration-callback-workflow", "callback.response-consistency",
			[]string{"nexus-operation", "callback", "workflow"}, []string{"register-callback", "record-callback-response"},
			[]string{"caller-terminal-before-callback", "callback-failure"}, []string{"hsm", "chasm"},
			[]string{"history-observation"}, "conforming"),
		contractWithFault("SparseRegressionCancellationRetry", "nexus-cancellation", "nexus.cancellation.won-excludes-success",
			[]string{"request-cancellation", "commit-cancellation", "retry-task", "persist-success"}, []string{"drop"}, "conforming"),
		contract("SparseRegressionCompletionBeforeStartResponse", "feature-nexus", "nexus-operation.closure",
			[]string{"nexus-operation"}, []string{"worker-returns-success", "persist-success"},
			[]string{"completion-before-start-response", "late-start-response"}, nil, nexusCapabilities(), "conforming"),
		contract("SparseRegressionOrdinaryNexusCompletion", "feature-nexus", "nexus-operation.closure",
			[]string{"nexus-operation", "workflow"}, []string{"schedule-operation", "worker-returns-success", "persist-success"},
			[]string{"result-digest", "endpoint-link", "embedded-storage-absent"}, nil, nexusCapabilities(), "conforming"),
		contract("SparseRegressionSharedHandlerWorkflow", "integration-callback-workflow", "callback.response-consistency",
			[]string{"nexus-operation", "callback", "workflow"}, []string{"register-callback", "record-callback-response"},
			[]string{"shared-handler", "callback-reference", "parallel-start"}, []string{"hsm"},
			[]string{"history-observation"}, "conforming"),
		contract("SparseRegressionStartToCloseTimeout", "integration-nexus-timeout", "nexus-operation.timeout-semantics",
			[]string{"nexus-operation"}, []string{"schedule-operation", "timeout-nexus-operation"},
			[]string{"start-to-close-timeout"}, []string{"hsm", "chasm"}, nexusCapabilities(), "conforming"),
	}
	result := make(map[string]BehaviorContract, len(contracts))
	for _, value := range contracts {
		result[value.Behavior] = value
	}
	return result
}

func Contract(behavior string) (BehaviorContract, bool) {
	contract, exists := behaviorContracts()[behavior]
	return contract, exists
}

func Scenario(behavior string, variant string) (compiler.Scenario, error) {
	contract, exists := Contract(behavior)
	if !exists {
		return compiler.Scenario{}, fmt.Errorf("unknown Umpire3 behavior contract %q", behavior)
	}
	if len(contract.Properties) != 1 || len(contract.Entities) == 0 || len(contract.Actions) == 0 {
		return compiler.Scenario{}, fmt.Errorf("behavior contract %q is not executable", behavior)
	}
	if len(contract.Variants) == 0 {
		if variant != "" {
			return compiler.Scenario{}, fmt.Errorf("behavior contract %q has no variant %q", behavior, variant)
		}
	} else if !slices.Contains(contract.Variants, variant) {
		return compiler.Scenario{}, fmt.Errorf("behavior contract %q does not declare variant %q", behavior, variant)
	}
	identifier := "behavior-contract-" + behavior
	if variant != "" {
		identifier += "-" + variant
	}
	resources := make([]regress.Resource, len(contract.Entities))
	for index, entity := range contract.Entities {
		resources[index] = regress.Resource{
			Identifier: identifier + "-" + entity, Kind: protocol.EntityKind(entity),
		}
	}
	actions := make([]regress.Term, len(contract.Actions))
	for index, action := range contract.Actions {
		var options []regress.ActionOption
		if behavior == "ProbeNexusFlagged" {
			options = append(options, regress.Asynchronously())
		}
		actions[index] = regress.Action(
			fmt.Sprintf("%s-action-%02d", identifier, index+1), protocol.ActionKind(action), options...)
	}
	actions = append(actions, regress.Require(protocol.PropertyID(contract.Properties[0])))
	root := regress.OnePath(actions...)
	catalog, err := protocol.DefaultCatalog()
	if err != nil {
		return compiler.Scenario{}, err
	}
	for index := len(contract.Faults) - 1; index >= 0; index-- {
		faultKind := protocol.FaultKind(contract.Faults[index])
		declaration, known := catalog.Fault(string(faultKind))
		if !known {
			return compiler.Scenario{}, fmt.Errorf("behavior contract %q has unknown fault %q", behavior, faultKind)
		}
		requiredCapabilities := make([]string, len(declaration.RequiredCapabilities))
		for capabilityIndex, capability := range declaration.RequiredCapabilities {
			requiredCapabilities[capabilityIndex] = string(capability)
		}
		resourceScope := make([]string, len(resources))
		for resourceIndex, resource := range resources {
			resourceScope[resourceIndex] = resource.Identifier
		}
		root = regress.During(
			regress.ConfiguredFault(protocol.Fault{
				Identifier: fmt.Sprintf("%s-fault-%02d", identifier, index+1), Kind: string(faultKind),
				SafetyClass: declaration.SafetyClass,
				Scope: protocol.FaultScope{
					Resources: resourceScope, Endpoints: []string{"umpire3-nexus-endpoint"},
					TaskQueues: []string{"umpire3-nexus-task-queue"}, Services: []string{"nexus"},
					Routes: []string{"/service/operation"}, Participants: resourceScope, Attempts: []int{1},
				},
				Occurrence:           protocol.FaultOccurrence{First: 1, Count: 1},
				RequiredCapabilities: requiredCapabilities,
			}),
			root,
		)
	}
	return regress.NewScenario(identifier, protocol.TargetID(contract.ModelTarget), resources, root), nil
}

func contract(
	behavior string,
	target string,
	property string,
	entities []string,
	actions []string,
	relations []string,
	variants []string,
	capabilities []string,
	verdict string,
) BehaviorContract {
	return BehaviorContract{
		Behavior: behavior, ModelTarget: target, Properties: []string{property}, Entities: entities,
		Actions: actions, Relations: relations, Variants: variants, RequiredCapabilities: capabilities,
		ExpectedVerdict: verdict, NegativeControl: "qualified-opposite-observation",
		Evidence: []string{"source-identity", "source-sequence", "entity-identity", "identity-lineage"},
		Fidelity: behaviorFidelity(behavior), EvidenceLevel: protocol.EvidenceLocalIntegration,
	}
}

func behaviorFidelity(behavior string) protocol.Fidelity {
	switch behavior {
	case "ProbeNexusCoverageGuidedFaults", "ProbeNexusRandomized":
		return protocol.FidelityPartial
	case "PlanAndDriveKitchenSinkNexusOperation", "PlanAndDriveKitchenSinkWorkflow",
		"PlanAndDriveNexusOperationCHASM", "PlanAndDriveWorkflowToCompletion", "ProbeNexusLearnedFootprint":
		return protocol.FidelitySemanticEquivalent
	case "ProbeNexusDegraded", "ProbeNexusExploration", "ProbeNexusFaultAction", "ProbeNexusFlagged", "ProbeNexusResilience",
		"ProbeNexusGeneratedCompletion", "ProbeNexusHTTPFaultSeam",
		"ProbeNexusReflectedDurationVariant", "ProbeNexusReflectedVariant", "ProbeNexusRejectedStart",
		"ProbeWorkflowContinueAsNew", "ProbeWorkflowContinueAsNewGenerated", "ProbeWorkflowGenerated",
		"ProbeWorkflowReset", "SparseRegressionBidirectionalNexusActivityLinks",
		"SparseRegressionCallbackAfterCallerCompletion", "SparseRegressionCancellationRetry",
		"SparseRegressionCompletionBeforeStartResponse", "SparseRegressionOrdinaryNexusCompletion",
		"SparseRegressionSharedHandlerWorkflow", "SparseRegressionStartToCloseTimeout":
		return protocol.FidelityExact
	default:
		return protocol.FidelityInventoryOnly
	}
}

func contractWithFault(
	behavior string,
	target string,
	property string,
	actions []string,
	faults []string,
	verdict string,
) BehaviorContract {
	result := contract(behavior, target, property, []string{"nexus-operation"}, actions,
		[]string{"scoped-fault-effect"}, nil, append(nexusCapabilities(), "fault-rpc"), verdict)
	result.Faults = faults
	return result
}

func nexusCapabilities() []string {
	return []string{"nexus", "nexus-observation", "nexus-worker-control"}
}
