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
    errorKindOf (Query.check (incompleteContext missing)
      (declaration (.verify Property.checked) exhaustivePolicy)) ==
        some .missingFiniteCompleteness) := by
  native_decide

def noFiniteDomains : QueryCheckContext (fun _ => True) :=
  .ofTarget targetWithoutPlanning

/-! Planning availability is additive: non-exhaustive semantic queries still consume the target. -/
example : (Query.check noFiniteDomains
    (declaration (.find Property.checked) searchPolicy)).isOk := by
  native_decide

example : errorKindOf (Query.check noFiniteDomains
    (declaration (.verify Property.checked) exhaustivePolicy)) =
      some .missingFiniteCompleteness := by
  native_decide

/-! The checked planner input retains the exact certified domains, not only their digests. -/
example : ((Query.check exhaustiveContext
    (declaration (.verify Property.checked) exhaustivePolicy)).toOption.bind fun query =>
      query.completeness.map fun evidence =>
        (evidence.roleAssignments.length, evidence.actions.length)) = some (1, 1) := by
  native_decide

/-! Query copies Target's stable compatibility tokens and finite domains verbatim. -/
example : ((Query.check exhaustiveContext
    (declaration (.verify Property.checked) exhaustivePolicy)).toOption.bind fun query =>
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

def duplicateActionAuthoring : DraftModel (fun _ => True)
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  DraftModel.make modelSpec modelProviders
    (.available kernel rfl duplicateActionPlanning)

def duplicateActionContext : QueryCheckContext (fun _ => True) :=
  .ofTarget (model duplicateActionAuthoring)

/-- Duplicate finite actions reject before Planning can enumerate a different candidate domain. -/
example : errorKindOf (Query.check duplicateActionContext
    (declaration (.verify Property.checked) exhaustivePolicy)) = some .duplicateFiniteDomain := by
  native_decide

/-! Completeness follows the exhaustive strategy, not a particular query form. -/
example : [
    Query.Form.verify Property.checked,
    .find Property.checked,
    .findViolation Property.checked,
    .pick [Property.checked]
  ].all (fun form =>
    errorKindOf (Query.check noFiniteDomains (declaration form exhaustivePolicy)) ==
      some .missingFiniteCompleteness) := by
  native_decide

end Umpire.QueryTests
