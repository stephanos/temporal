import Umpire.Query.Tests.Fixtures

/-! Deterministic Query validation failures and exact-trace admission checks. -/

namespace Umpire.QueryTests

open Umpire

def checkedFixtureQuery : CheckedQuery (fun _ => True) :=
  checkedQuery target (declaration (.verify checkedProperty) exhaustivePolicy)
    (by native_decide)

/-- Checked Query authoring re-ascribes the dependent target at the language boundary. -/
example : checkedFixtureQuery.target = target := by
  rfl

#guard_msgs (error, substring := true) in
def queryWithoutValidityProof : CheckedQuery (fun _ => True) :=
  checkedQuery target (declaration (.verify checkedProperty) exhaustivePolicy)

private def queryErrorOf
    (result : Except QueryError (CheckedQuery (fun _ => True))) : Option QueryError :=
  match result with
  | .error failure => some failure
  | .ok _ => none

private def queryErrorJsonOf
    (result : Except QueryError (CheckedQuery (fun _ => True))) : Option String :=
  queryErrorOf result |>.map canonicalQueryErrorJson

def alphaProperty : CheckedProperty := {
  checkedProperty with id := id "query.property.alpha"
}

def zetaProperty : CheckedProperty := {
  checkedProperty with id := id "query.property.zeta"
}

def exactAdapterFailures : List (Option QueryError) := [
  queryErrorOf (checkQuery context {
    declaration (.witness checkedProperty) with id := id "", source := { path := "" }
  }),
  queryErrorOf (checkQuery context {
    declaration (.witness checkedProperty) with id := id "query"
  }),
  queryErrorOf (checkQuery context (declaration
    (.select [zetaProperty, alphaProperty, zetaProperty, alphaProperty]))),
  queryErrorOf (checkQuery context {
    declaration (.witness checkedProperty) with target := id "zeta.target.mismatch"
  })
]

/-- Shared identity adapters retain Query's exact payloads and deterministic duplicate witness. -/
example : exactAdapterFailures = [
  some {
    kind := .emptyDefinitionId
    definitionId := id "umpire.query.anonymous"
    sourcePath := "<unknown>"
    offendingValue := "<empty>"
    relatedDefinitionIds := []
  },
  some {
    kind := .invalidDefinitionId
    definitionId := id "query"
    sourcePath := "Umpire/Query/Tests.lean"
    offendingValue := "query"
    relatedDefinitionIds := [id "query"]
  },
  some {
    kind := .duplicateProperty
    definitionId := id "query.declaration.fixture"
    sourcePath := "Umpire/Query/Tests.lean"
    offendingValue := "query.property.alpha"
    relatedDefinitionIds := [id "query.property.alpha"]
  },
  some {
    kind := .targetMismatch
    definitionId := id "query.declaration.fixture"
    sourcePath := "Umpire/Query/Tests.lean"
    offendingValue := "zeta.target.mismatch != query.target.fixture"
    relatedDefinitionIds := [id "query.target.fixture", id "zeta.target.mismatch"]
  }
] := by
  native_decide

/-- Canonical Query diagnostics retain field order and canonical related-ID order. -/
example : queryErrorJsonOf (checkQuery context {
    declaration (.witness checkedProperty) with target := id "zeta.target.mismatch"
  }) = some ("{\"kind\":\"target-mismatch\",\"definitionId\":" ++
    "\"query.declaration.fixture\",\"sourcePath\":\"Umpire/Query/Tests.lean\"," ++
    "\"offendingValue\":\"zeta.target.mismatch != query.target.fixture\"," ++
    "\"relatedDefinitionIds\":[\"query.target.fixture\",\"zeta.target.mismatch\"]}") := by
  native_decide

def invalidLimits : QueryLimits := {
  limits with behavior := {
    limits.behavior with transitions := { value := 0, unit := .semanticTransitions }
  }
}

/-! Invalid limits and unsupported verify strategies retain distinct deterministic failures. -/
example : [
    errorKindOf (checkQuery context {
      declaration (.witness checkedProperty) with limits := invalidLimits
    }),
    errorKindOf (checkQuery context (declaration (.verify checkedProperty)))
  ] = [some .invalidLimit, some .incompatibleStrategy] := by
  native_decide

def exactTrace (outcome : ModelValue := acceptedValue) : BehaviorTrace := {
  setup
  trace := {
    initialState := initial
    steps := [{
      selectedAction := requestValue
      modelOutcome := outcome
      resultingState := completed
      observations := [observedValue]
    }]
  }
}

def invalidExactBehavior : CheckedBehavior := {
  checkedBehavior with
  traceExactly := some (exactTrace (value accepted "not-admitted"))
  behaviorFingerprint := behaviorFingerprintOf "behavior/invalid-exact-v1"
}

/-! Structural exactness is insufficient: the selected kernel must admit the complete step. -/
example : errorKindOf (checkQuery context
    (declaration (.select [checkedProperty]) searchPolicy invalidExactBehavior)) =
      some .targetKernelMismatch := by
  native_decide

end Umpire.QueryTests
