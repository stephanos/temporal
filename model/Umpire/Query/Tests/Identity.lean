import Umpire.Query.Tests.Fixtures

/-! Canonical projection and semantic identity checks for checked Queries. -/

namespace Umpire.QueryTests

open Umpire

example : Limits.bounded 1 2 3 = ({
    steps := { value := 1, unit := .steps }
    actions := { value := 2, unit := .actions }
    search := { value := 3, unit := .search }
  } : Limits) := by
  rfl

example : Limits.bounded 0 0 0 = ({
    steps := { value := 0, unit := .steps }
    actions := { value := 0, unit := .actions }
    search := { value := 0, unit := .search }
  } : Limits) := by
  rfl

def canonicalOf
    (queryContext : QueryCheckContext (fun _ => True))
    (authoredQuery : Query) : Option String :=
  (Query.check queryContext authoredQuery).toOption.map canonicalQueryJson

def fingerprintOf
    (queryContext : QueryCheckContext (fun _ => True))
    (authoredQuery : Query) : Option BehaviorFingerprint :=
  (Query.check queryContext authoredQuery).toOption.map CheckedQuery.behaviorFingerprint

example : PlannerPolicy.shortest = {
    strategy := .shortest
    seed := 17
  } := by
  rfl

example : PlannerPolicy.exhaustive = {
    strategy := .exhaustive
    seed := 17
  } := by
  rfl

example : PlannerPolicy.seeded = {
    strategy := .seeded
    seed := 17
  } := by
  rfl

example : PlannerPolicy.seeded 0 = {
    strategy := .seeded
    seed := 0
  } := by
  rfl

def reorderedModelSpec : ModelSpec (fun _ => True)
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  modelSpec with definitions := modelSpec.definitions.reverse
}

def reorderedTarget : QueryModel (fun _ => True) :=
  model (DraftModel.make reorderedModelSpec modelProviders)

def incidentalContext : QueryCheckContext (fun _ => True) := .ofTarget reorderedTarget

def incidentalDeclaration : Query := {
  declaration (.find { Property.checked with documentation := "changed docs" }) with
  behavior := { Scenario.checked with documentation := "changed docs" }
  documentation := "changed query docs"
}

example : canonicalOf context (declaration (.find Property.checked)) =
    canonicalOf incidentalContext incidentalDeclaration := by
  native_decide

def noSetupModelSpec : ModelSpec (fun _ => True)
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  modelSpec with resolvedSetups := []
}

def noSetupTargetAuthoring : DraftModel (fun _ => True)
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  DraftModel.make noSetupModelSpec modelProviders
    (.available kernel rfl finitePlanning)

def noSetupContext : QueryCheckContext (fun _ => True) :=
  .ofTarget (model noSetupTargetAuthoring)

/-- Query fingerprints bind the exact finite role assignments Planning will enumerate. -/
example : fingerprintOf context (declaration (.find Property.checked)) !=
    fingerprintOf noSetupContext (declaration (.find Property.checked)) := by
  native_decide

def orderedProperty : CheckedProperty := {
  Property.checked with
  id := id "query.property.ordered"
  behaviorFingerprint := behaviorFingerprintOf "property/ordered-v1"
}

/-! Property source order does not change the canonical query projection. -/
example : canonicalOf context (declaration (.pick [Property.checked, orderedProperty])) =
    canonicalOf context (declaration (.pick [orderedProperty, Property.checked])) := by
  native_decide

def definitionsWithCanonicalBehavior
    (definitionId : DefinitionId)
    (digest : String) : List DefinitionMetadata :=
  targetDefinitions.map fun definition =>
    if definition.id == definitionId then { definition with behaviorVersion := digest }
    else definition

def changedSemanticModelSpec : ModelSpec (fun _ => True)
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  modelSpec with definitions := definitionsWithCanonicalBehavior targetId "query-target/v2"
}

def changedSemanticTarget : QueryModel (fun _ => True) :=
  model (DraftModel.make changedSemanticModelSpec modelProviders)

def changedCompositionModelSpec : ModelSpec (fun _ => True)
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  modelSpec with requiredCapabilities := [extraCapabilityId]
}

def changedCompositionTarget : QueryModel (fun _ => True) :=
  model (DraftModel.make changedCompositionModelSpec modelProviders)

def changedKernel : Machine
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := kernel

def changedKernelModelSpec : ModelSpec (fun _ => True)
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  modelSpec with
  definitions := definitionsWithCanonicalBehavior kernelId "query-kernel/v2"
  machine := .checked changedKernel
}

def changedKernelTarget : QueryModel (fun _ => True) :=
  model (DraftModel.make changedKernelModelSpec modelProviders)

def contextFor (candidate : QueryModel (fun _ => True)) : QueryCheckContext (fun _ => True) := {
  target := .checked { target := candidate, completeness := none }
}

def changedProperty : CheckedProperty := {
  Property.checked with behaviorFingerprint := behaviorFingerprintOf "property/v2"
}

def changedBehavior : CheckedScenario := {
  Scenario.checked with behaviorFingerprint := behaviorFingerprintOf "behavior/v2"
}

def changedLimits : Limits := { limits with steps := { value := 2, unit := .steps } }

def changedLimitsDeclaration : Query := {
  declaration (.find Property.checked) with limits := changedLimits
}

def changedStrategyDeclaration : Query := {
  declaration (.find Property.checked) with policy := { searchPolicy with strategy := .seeded }
}

def changedSeedDeclaration : Query := {
  declaration (.find Property.checked) with policy := { searchPolicy with seed := 18 }
}

/-! Every consumed semantic input changes Query identity. -/
example :
    let baseline := fingerprintOf context (declaration (.find Property.checked))
    [
      fingerprintOf context (declaration (.find changedProperty)),
      fingerprintOf context (declaration (.find Property.checked) searchPolicy changedBehavior),
      fingerprintOf context changedLimitsDeclaration,
      fingerprintOf context changedStrategyDeclaration,
      fingerprintOf context changedSeedDeclaration,
      fingerprintOf (contextFor changedSemanticTarget)
        (declaration (.find Property.checked)),
      fingerprintOf (contextFor changedCompositionTarget)
        (declaration (.find Property.checked)),
      fingerprintOf (contextFor changedKernelTarget)
        (declaration (.find Property.checked))
    ].all (fun changed => changed.isSome && changed != baseline) := by
  native_decide

end Umpire.QueryTests
