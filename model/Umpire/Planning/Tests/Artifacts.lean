import Umpire.Planning.Tests.Fixtures

/-! Inspectability, optional occurrences, byte stability, and semantic identity checks. -/

namespace Umpire.PlanningTests

open Umpire

def witnessSpec (seed : Nat := 17) : Option ExperimentSpec :=
  (run 2 (.witness property) .shortest 10 seed false).artifact

def incidentalWitnessSpec : Option ExperimentSpec :=
  let query := checkedQuery 2 (.witness property) .shortest 10 17 false
  let incidental : CheckedQuery (fun _ => True) := {
    query with
    documentation := "changed query documentation"
    behavior := { query.behavior with documentation := "changed behavior documentation" }
    form := .witness { property with documentation := "changed property documentation" }
  }
  (plan incidental (incrementalKernel 2)).artifact

def selectedArtifactIsInspectable : Bool :=
  match witnessSpec with
  | none => false
  | some spec =>
      spec.plan.initialState == initial &&
      spec.plan.requestedActions == [requestValue] &&
      spec.plan.modelOutcomes == [acceptedValue] &&
      spec.plan.linearExtension.map PlannedOccurrence.definitionId == [occurrence] &&
      spec.plan.linearExtension.map PlannedOccurrence.action == [request] &&
      spec.plan.linearExtension.length == spec.plan.requestedActions.length &&
      spec.plan.bindings == setup &&
      spec.plan.symbolicRoles == [] &&
      spec.plan.expandedBounds == bounds &&
      spec.plan.selectionReason == .satisfyingWitness &&
      spec.plan.checkpoints.length == 1 &&
      spec.plan.omissions == canonicalPlannerOmissions &&
      spec.properties.map PortableProperty.definitionId == [property.id]

/-! A selected trace is compiled into an inspectable plan that separates requests from outcomes. -/
example : selectedArtifactIsInspectable := by
  native_decide

def optionalBehavior : CheckedBehavior := {
  behavior with
  requiredOccurrences := []
  semanticDigest := "behavior/optional-v1"
}

/-! The linear extension contains every selected action, including optional occurrences. -/
example :
    ((run 2 (.select [property]) .shortest 10 17 false optionalBehavior).artifact.map fun spec =>
      (spec.plan.linearExtension.length,
        spec.plan.linearExtension.map PlannedOccurrence.action)) =
      some (1, [request]) := by
  native_decide

/-! Independent planning and rendering of semantically identical checked inputs is byte-identical. -/
example :
    witnessSpec.map canonicalExperimentSpecJson =
      incidentalWitnessSpec.map canonicalExperimentSpecJson := by
  native_decide

/-! A meaning-bearing Query input is part of the artifact semantic identity. -/
example :
    witnessSpec.map ExperimentSpec.semanticIdentity !=
      (witnessSpec 18).map ExperimentSpec.semanticIdentity := by
  native_decide

end Umpire.PlanningTests
