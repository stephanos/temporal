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

def noFiniteDomains : QueryCheckContext (fun _ => True) :=
  .ofTarget targetWithoutPlanning

/-! Planning availability is additive: non-exhaustive semantic queries still consume the target. -/
example : (checkQuery noFiniteDomains
    (declaration (.witness checkedProperty) searchPolicy)).isOk := by
  native_decide

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

/-! Query copies Target's stable compatibility tokens and finite domains verbatim. -/
example : ((checkQuery exhaustiveContext
    (declaration (.verify checkedProperty) exhaustivePolicy)).toOption.bind fun query =>
      query.completeness.map fun evidence =>
        (evidence.roleAssignments, evidence.actions,
          evidence.roleDomainFingerprint.render, evidence.actionDomainFingerprint.render)) =
      some ([setup], [requestValue],
        (behaviorFingerprintOf <|
          "query-role-domain/v1\n[[{\"role\":\"query.role.operation\",\"value\":" ++
            "{\"definitionId\":\"query.state.phase\",\"value\":\"operation-a\"}}]]").render,
        (behaviorFingerprintOf <|
          "query-action-domain/v1\n[{\"definitionId\":\"query.action.request\"," ++
            "\"value\":\"request\"}]").render) := by
  native_decide

def duplicateActionPlanning : FinitePlanningCapability kernel.authoritativeStep := {
  actions := [requestValue, requestValue]
  actionSound := by
    intro action member
    simp only [List.mem_cons, List.not_mem_nil, or_false] at member
    rcases member with member | member <;> subst action
    · exact ⟨initial, transition, rfl, rfl, rfl⟩
    · exact ⟨initial, transition, rfl, rfl, rfl⟩
  actionComplete := by
    intro state action result admitted
    simp [admitted.2.1]
}

def duplicateActionAuthoring : AuthoredTarget (fun _ => True)
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  AuthoredTarget.make targetDefinition targetComposition
    (.available kernel rfl duplicateActionPlanning)

def duplicateActionContext : QueryCheckContext (fun _ => True) :=
  .ofTarget (checkedTarget duplicateActionAuthoring)

/-- Duplicate finite actions reject before Planning can enumerate a different candidate domain. -/
example : errorKindOf (checkQuery duplicateActionContext
    (declaration (.verify checkedProperty) exhaustivePolicy)) = some .duplicateFiniteDomain := by
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
