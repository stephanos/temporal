import Umpire.Query.Tests.Fixtures

/-! Deterministic Query validation failures and exact-trace admission checks. -/

namespace Umpire.QueryTests

open Umpire

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
