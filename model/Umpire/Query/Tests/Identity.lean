import Umpire.Query.Tests.Fixtures

/-! Canonical projection and semantic identity checks for checked Queries. -/

namespace Umpire.QueryTests

open Umpire

example : QueryLimits.bounded 1 2 3 = ({
    behavior := {
      transitions := { value := 1, unit := .semanticTransitions }
      selectedActions := { value := 2, unit := .selectedActions }
    }
    search := { value := 3, unit := .candidateEvaluations }
  } : QueryLimits) := by
  rfl

example : QueryLimits.bounded 0 0 0 = ({
    behavior := {
      transitions := { value := 0, unit := .semanticTransitions }
      selectedActions := { value := 0, unit := .selectedActions }
    }
    search := { value := 0, unit := .candidateEvaluations }
  } : QueryLimits) := by
  rfl

def canonicalOf
    (queryContext : QueryCheckContext (fun _ => True))
    (queryDeclaration : QueryDeclaration) : Option String :=
  (checkQuery queryContext queryDeclaration).toOption.map canonicalQueryJson

def fingerprintOf
    (queryContext : QueryCheckContext (fun _ => True))
    (queryDeclaration : QueryDeclaration) : Option BehaviorFingerprint :=
  (checkQuery queryContext queryDeclaration).toOption.map CheckedQuery.behaviorFingerprint

example : PlannerPolicy.shortest = {
    strategy := .shortest
    seed := 17
    tieBreak := .definitionId
  } := by
  rfl

example : PlannerPolicy.exhaustive = {
    strategy := .exhaustive
    seed := 17
    tieBreak := .definitionId
  } := by
  rfl

example : PlannerPolicy.seeded = {
    strategy := .seeded
    seed := 17
    tieBreak := .definitionId
  } := by
  rfl

example : PlannerPolicy.seeded 0 = {
    strategy := .seeded
    seed := 0
    tieBreak := .definitionId
  } := by
  rfl

def reorderedTargetDefinition : TargetDefinition
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  targetDefinition with definitions := targetDefinition.definitions.reverse
}

def reorderedTarget : QueryTarget (fun _ => True) :=
  checkedTarget (AuthoredTarget.make reorderedTargetDefinition targetComposition)

def incidentalContext : QueryCheckContext (fun _ => True) := .ofTarget reorderedTarget

def incidentalDeclaration : QueryDeclaration := {
  declaration (.witness { checkedProperty with documentation := "changed docs" }) with
  behavior := { checkedBehavior with documentation := "changed docs" }
  documentation := "changed query docs"
}

example : canonicalOf context (declaration (.witness checkedProperty)) =
    canonicalOf incidentalContext incidentalDeclaration := by
  native_decide

def noSetupTargetDefinition : TargetDefinition
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  targetDefinition with resolvedSetups := []
}

def noSetupTargetAuthoring : AuthoredTarget (fun _ => True)
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  AuthoredTarget.make noSetupTargetDefinition targetComposition
    (.available kernel rfl finitePlanning)

def noSetupContext : QueryCheckContext (fun _ => True) :=
  .ofTarget (checkedTarget noSetupTargetAuthoring)

/-- Query fingerprints bind the exact finite role assignments Planning will enumerate. -/
example : fingerprintOf context (declaration (.witness checkedProperty)) !=
    fingerprintOf noSetupContext (declaration (.witness checkedProperty)) := by
  native_decide

def orderedProperty : CheckedProperty := {
  checkedProperty with
  id := id "query.property.ordered"
  behaviorFingerprint := behaviorFingerprintOf "property/ordered-v1"
}

/-! Property source order does not change the canonical query projection. -/
example : canonicalOf context (declaration (.select [checkedProperty, orderedProperty])) =
    canonicalOf context (declaration (.select [orderedProperty, checkedProperty])) := by
  native_decide

def definitionsWithCanonicalBehavior
    (definitionId : DefinitionId)
    (digest : String) : List DefinitionMetadata :=
  targetDefinitions.map fun definition =>
    if definition.id == definitionId then { definition with canonicalBehavior := digest }
    else definition

def changedSemanticTargetDefinition : TargetDefinition
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  targetDefinition with definitions := definitionsWithCanonicalBehavior targetId "query-target/v2"
}

def changedSemanticTarget : QueryTarget (fun _ => True) :=
  checkedTarget (AuthoredTarget.make changedSemanticTargetDefinition targetComposition)

def changedCompositionTargetDefinition : TargetDefinition
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  targetDefinition with requiredCapabilities := [extraCapabilityId]
}

def changedCompositionTarget : QueryTarget (fun _ => True) :=
  checkedTarget (AuthoredTarget.make changedCompositionTargetDefinition targetComposition)

def changedKernel : TransitionKernel
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := kernel

def changedKernelTargetDefinition : TargetDefinition
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  targetDefinition with
  definitions := definitionsWithCanonicalBehavior kernelId "query-kernel/v2"
  kernel := .checked changedKernel
}

def changedKernelTarget : QueryTarget (fun _ => True) :=
  checkedTarget (AuthoredTarget.make changedKernelTargetDefinition targetComposition)

def contextFor (candidate : QueryTarget (fun _ => True)) : QueryCheckContext (fun _ => True) := {
  target := .checked { target := candidate, completeness := none }
}

def changedProperty : CheckedProperty := {
  checkedProperty with behaviorFingerprint := behaviorFingerprintOf "property/v2"
}

def changedBehavior : CheckedBehavior := {
  checkedBehavior with behaviorFingerprint := behaviorFingerprintOf "behavior/v2"
}

def changedLimits : QueryLimits := {
  limits with behavior := {
    limits.behavior with transitions := { value := 2, unit := .semanticTransitions }
  }
}

def changedLimitsDeclaration : QueryDeclaration := {
  declaration (.witness checkedProperty) with limits := changedLimits
}

def changedStrategyDeclaration : QueryDeclaration := {
  declaration (.witness checkedProperty) with policy := { searchPolicy with strategy := .seeded }
}

def changedSeedDeclaration : QueryDeclaration := {
  declaration (.witness checkedProperty) with policy := { searchPolicy with seed := 18 }
}

/-! Every consumed semantic input changes Query identity. -/
example :
    let baseline := fingerprintOf context (declaration (.witness checkedProperty))
    [
      fingerprintOf context (declaration (.witness changedProperty)),
      fingerprintOf context (declaration (.witness checkedProperty) searchPolicy changedBehavior),
      fingerprintOf context changedLimitsDeclaration,
      fingerprintOf context changedStrategyDeclaration,
      fingerprintOf context changedSeedDeclaration,
      fingerprintOf (contextFor changedSemanticTarget)
        (declaration (.witness checkedProperty)),
      fingerprintOf (contextFor changedCompositionTarget)
        (declaration (.witness checkedProperty)),
      fingerprintOf (contextFor changedKernelTarget)
        (declaration (.witness checkedProperty))
    ].all (fun changed => changed.isSome && changed != baseline) := by
  native_decide

end Umpire.QueryTests
