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
      (declaration (.verify checkedProperty) exhaustivePolicy)),
    summaryOf (checkQuery context (declaration (.witness checkedProperty))),
    summaryOf (checkQuery context (declaration (.counterexample checkedProperty))),
    summaryOf (checkQuery context (declaration (.select [checkedProperty])))
  ] = [
    some (.universal, .verifiedWithinBounds),
    some (.existential, .satisfyingWitness),
    some (.existential, .violatingCounterexample),
    some (.exploratory, .boundedSelection)
  ] := by
  native_decide

end Umpire.QueryTests
