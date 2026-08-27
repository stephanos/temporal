import Umpire.Query.Tests.Fixtures

/-! Canonical projection and semantic identity checks for checked Queries. -/

namespace Umpire.QueryTests

open Umpire

def canonicalOf
    (queryContext : QueryCheckContext (fun _ => True))
    (queryDeclaration : QueryDeclaration) : Option String :=
  (checkQuery queryContext queryDeclaration).toOption.map canonicalQueryJson

def digestOf
    (queryContext : QueryCheckContext (fun _ => True))
    (queryDeclaration : QueryDeclaration) : Option String :=
  (checkQuery queryContext queryDeclaration).toOption.map CheckedQuery.semanticDigest

def reorderedTarget : QueryTarget (fun _ => True) := {
  target with declarations := target.declarations.reverse
}

def incidentalContext : QueryCheckContext (fun _ => True) := {
  target := .checked { target := reorderedTarget, completeness := none }
}

def incidentalDeclaration : QueryDeclaration := {
  declaration (.witness { checkedProperty with documentation := "changed docs" }) with
  behavior := { checkedBehavior with documentation := "changed docs" }
  documentation := "changed query docs"
}

example : canonicalOf context (declaration (.witness checkedProperty)) =
    canonicalOf incidentalContext incidentalDeclaration := by
  native_decide

def orderedProperty : CheckedProperty := {
  checkedProperty with
  id := id "query.property.ordered"
  semanticDigest := "property/ordered-v1"
}

/-! Property source order does not change the canonical query projection. -/
example : canonicalOf context (declaration (.select [checkedProperty, orderedProperty])) =
    canonicalOf context (declaration (.select [orderedProperty, checkedProperty])) := by
  native_decide

def changedTarget
    (digest : String := target.semanticDigest)
    (kernelDigest : String := kernel.metadata.contractDigest)
    (composition : List DeclarationId := []) : QueryTarget (fun _ => True) := {
  target with
  semanticDigest := digest
  requiredCapabilities := composition
  kernel := { kernel with metadata := { kernel.metadata with contractDigest := kernelDigest } }
  planning := .unavailable
}

def contextFor (candidate : QueryTarget (fun _ => True)) : QueryCheckContext (fun _ => True) := {
  target := .checked { target := candidate, completeness := none }
}

def changedProperty : CheckedProperty := {
  checkedProperty with semanticDigest := "property/v2"
}

def changedBehavior : CheckedBehavior := {
  checkedBehavior with semanticDigest := "behavior/v2"
}

def changedBounds : QueryBounds := {
  bounds with behavior := {
    bounds.behavior with transitions := { value := 2, unit := .semanticTransitions }
  }
}

def changedBoundsDeclaration : QueryDeclaration := {
  declaration (.witness checkedProperty) with bounds := changedBounds
}

def changedStrategyDeclaration : QueryDeclaration := {
  declaration (.witness checkedProperty) with policy := { searchPolicy with strategy := .breadthFirst }
}

def changedSeedDeclaration : QueryDeclaration := {
  declaration (.witness checkedProperty) with policy := { searchPolicy with seed := 18 }
}

/-! Every consumed semantic input changes Query identity. -/
example :
    let baseline := digestOf context (declaration (.witness checkedProperty))
    [
      digestOf context (declaration (.witness changedProperty)),
      digestOf context (declaration (.witness checkedProperty) searchPolicy changedBehavior),
      digestOf context changedBoundsDeclaration,
      digestOf context changedStrategyDeclaration,
      digestOf context changedSeedDeclaration,
      digestOf (contextFor (changedTarget "target/v2"))
        (declaration (.witness checkedProperty)),
      digestOf (contextFor (changedTarget (composition := [id "query.capability.extra"])))
        (declaration (.witness checkedProperty)),
      digestOf (contextFor (changedTarget (kernelDigest := "query-kernel/v2")))
        (declaration (.witness checkedProperty))
    ].all (fun changed => changed.isSome && changed != baseline) := by
  native_decide

end Umpire.QueryTests
