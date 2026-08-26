import Umpire.Query.Tests.Fixtures

/-! Finite-domain and kernel-completeness checks for exhaustive Query planning. -/

namespace Umpire.QueryTests

open Umpire

def incompleteContext (missing : CompletenessRequirement) :
    QueryCheckContext (fun _ => True) := {
  target := .incomplete target.id source [missing]
}

/-! Exhaustive mode fails closed for every missing finite or kernel-completeness obligation. -/
example : [
    .roleDomain,
    .actionDomain,
    .initialEnumeration,
    .stepEnumeration,
    .kernelRelation
  ].all (fun missing =>
    errorKindOf (checkQuery (incompleteContext missing)
      (declaration (.verify checkedProperty) exhaustivePolicy)) ==
        some .missingFiniteCompleteness) := by
  native_decide

def noFiniteDomains : QueryCheckContext (fun _ => True) := {
  target := .checked { target, completeness := none }
}

example : errorKindOf (checkQuery noFiniteDomains
    (declaration (.verify checkedProperty) exhaustivePolicy)) =
      some .missingFiniteCompleteness := by
  native_decide

/-! The checked planner input retains the exact certified domains, not only their digests. -/
example : ((checkQuery exhaustiveContext
    (declaration (.verify checkedProperty) exhaustivePolicy)).toOption.bind fun query =>
      query.completeness.map fun evidence =>
        (evidence.roleAssignments.length, evidence.actions.length)) = some (1, 1) := by
  native_decide

/-! Completeness follows the exhaustive strategy, not a particular query form. -/
example : [
    QueryForm.verify checkedProperty,
    .witness checkedProperty,
    .counterexample checkedProperty,
    .select [checkedProperty]
  ].all (fun form =>
    errorKindOf (checkQuery noFiniteDomains (declaration form exhaustivePolicy)) ==
      some .missingFiniteCompleteness) := by
  native_decide

end Umpire.QueryTests
