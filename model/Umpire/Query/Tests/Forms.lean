import Umpire.Query.Tests.Fixtures

/-! Quantifier and claim checks for every public Query form. -/

namespace Umpire.QueryTests

open Umpire

def summaryOf
    (result : Except QueryError (CheckedQuery (fun _ => True))) :
    Option (QueryQuantifier × QueryClaim) :=
  result.toOption.map fun query => (query.quantifier, query.claim)

/-! Every public form fixes its quantifier and claim before planning. -/
example : [
    summaryOf (checkQuery exhaustiveContext
      (declaration (.verify Property.checked) exhaustivePolicy)),
    summaryOf (checkQuery context (declaration (.witness Property.checked))),
    summaryOf (checkQuery context (declaration (.counterexample Property.checked))),
    summaryOf (checkQuery context (declaration (.select [Property.checked])))
  ] = [
    some (.universal, .verifiedWithinLimits),
    some (.existential, .satisfyingWitness),
    some (.existential, .violatingCounterexample),
    some (.exploratory, .limitedSelection)
  ] := by
  native_decide

end Umpire.QueryTests
