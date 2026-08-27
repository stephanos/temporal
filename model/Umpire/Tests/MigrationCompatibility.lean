import Umpire.Examples.SwitchTests

/-!
Executable compatibility matrix for the domain-neutral Switch migration.

This cross-layer fixture intentionally lives outside `Umpire.Target.*`: it exercises the checked
Target through Query, Planning, and Artifact while the focused Target suite stays import-pure.
-/

namespace Umpire.Tests.MigrationCompatibility

open Umpire
open Umpire.Examples.Switch

/-- The domain-neutral part of the closed migration inventory. -/
def compatibilityFamilies : List String := ["switch"]

private def occurrenceAt
    (declarationId owner : DeclarationId)
    (role : AuthoringOccurrenceRole)
    (line column : Nat) : AuthoringOccurrence := {
  id := {
    sourcePath := "Umpire/Tests/MigrationCompatibility.lean"
    line
    column
    endLine := line
    endColumn := column + 8
    localOrdinal := 0
  }
  declarationId
  path := { role, owner }
}

private def authoringAt (line column : Nat) : AuthoredTarget LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  targetAuthoring with
  occurrences := [occurrenceAt targetId targetId .targetDeclaration line column]
}

private def checkedSummary
    (result : Except AuthoringDiagnostic (QueryTarget LawStatement)) :
    Option (String × String × Option (List SemanticValue × String × String)) :=
  result.toOption.map fun checked =>
    (checked.canonicalMetadata, checked.semanticDigest,
      match checked.planning with
      | .unavailable => none
      | .available capability => some
          (capability.actions, capability.roleDomainDigest, capability.actionDomainDigest))

/-! Moving a compiler-only occurrence cannot change any checked semantic product. -/
example : [
    checkedSummary (checkTarget (authoringAt 12 3)),
    checkedSummary (checkTarget (authoringAt 420 19))
  ] == [
    some (canonicalCheckedTargetJson target, target.semanticDigest,
      some ([flipAction], "switch-role-domain/v1", "switch-action-domain/v1")),
    some (canonicalCheckedTargetJson target, target.semanticDigest,
      some ([flipAction], "switch-role-domain/v1", "switch-action-domain/v1"))
  ] := by
  native_decide

example : target.source = source ∧
    (checkTarget (authoringAt 420 19)).toOption.map CheckedTarget.source = some source := by
  native_decide

private def earlyTarget : QueryTarget LawStatement :=
  checkedTarget (authoringAt 12 3)

private def relocatedTarget : QueryTarget LawStatement :=
  checkedTarget (authoringAt 420 19)

private def exactActionDeclaration : QueryDeclaration := {
  id := exactActionQueryId
  source
  target := targetId
  form := .witness flipProperty
  behavior := exactActionBehavior
  bounds
  policy := shortestPolicy
}

/-! Query canonical bytes depend on stable target semantics, not elaboration layout. -/
example : [
    (checkQuery (.ofTarget earlyTarget) exactActionDeclaration).toOption.map
      canonicalQueryJson,
    (checkQuery (.ofTarget relocatedTarget) exactActionDeclaration).toOption.map
      canonicalQueryJson
  ] = [some (canonicalQueryJson exactActionQuery), some (canonicalQueryJson exactActionQuery)] := by
  native_decide

private def wrongKindDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  targetDeclaration with requiredCapabilities := [flipActionId]
}

private def wrongKindAuthoringAt (line column : Nat) : AuthoredTarget LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  declaration := wrongKindDeclaration
  occurrences := [occurrenceAt flipActionId targetId .capabilityRequirement line column]
}

private def diagnosticSummary
    (result : Except AuthoringDiagnostic (QueryTarget LawStatement)) :
    Option (DeclarationErrorKind × AuthoringOccurrenceRole × String × Nat × Nat) :=
  match result with
  | .ok _ => none
  | .error diagnostic => some
      (diagnostic.error.kind, diagnostic.path.role, diagnostic.offending.sourcePath,
        diagnostic.offending.line, diagnostic.offending.column)

/-! The diagnostic follows the authored occurrence while stable provenance remains unchanged. -/
example : [
    diagnosticSummary (checkTarget (wrongKindAuthoringAt 31 4)),
    diagnosticSummary (checkTarget (wrongKindAuthoringAt 503 27))
  ] = [
    some (.wrongKind, .capabilityRequirement,
      "Umpire/Tests/MigrationCompatibility.lean", 31, 4),
    some (.wrongKind, .capabilityRequirement,
      "Umpire/Tests/MigrationCompatibility.lean", 503, 27)
  ] := by
  native_decide

example : wrongKindDeclaration.source = source := by
  rfl

private def invalidIdentityMetadata : DeclarationMetadata := {
  id := DeclarationId.of "action"
  kind := .action
  source
  contractDigest := "invalid-identity/v1"
}

private def invalidIdentityDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  targetDeclaration with
  declarations := invalidIdentityMetadata :: targetDeclaration.declarations
}

private def missingProviderDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  targetDeclaration with providers := []
}

private def providerWithoutLaw : CapabilityProvider LawStatement := {
  switchProvider with lawWitnesses := []
}

private def missingLawDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  targetDeclaration with providers := [providerWithoutLaw]
}

private def incompleteKernelDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  targetDeclaration with
  kernel := .incomplete transitionKernel.metadata
    [DeclarationId.of "umpire.kernel-proof.step-complete"]
}

private def targetErrorKind
    (result : Except DeclarationError (QueryTarget LawStatement)) : Option DeclarationErrorKind :=
  match result with
  | .ok _ => none
  | .error failure => some failure.kind

/-! Target-owned invalid identity, provider, law, and kernel availability stay typed at Target. -/
example : [
    targetErrorKind (composeTarget invalidIdentityDeclaration),
    targetErrorKind (composeTarget missingProviderDeclaration),
    targetErrorKind (composeTarget missingLawDeclaration),
    targetErrorKind (composeTarget incompleteKernelDeclaration)
  ] = [some .invalidIdentity, some .missingProvider, some .missingLaw, some .incompleteKernel] := by
  native_decide

private def queryErrorKind
    (result : Except QueryError (CheckedQuery LawStatement)) : Option QueryErrorKind :=
  match result with
  | .ok _ => none
  | .error failure => some failure.kind

private def invalidBounds : QueryBounds := {
  bounds with
  behavior := {
    bounds.behavior with transitions := { value := 0, unit := .semanticTransitions }
  }
}

private def invalidBoundDeclaration : QueryDeclaration := {
  exactActionDeclaration with bounds := invalidBounds
}

private def exhaustiveDeclaration : QueryDeclaration := {
  exactActionDeclaration with policy := { shortestPolicy with strategy := .exhaustive }
}

private def noFinitePlanningTarget : QueryTarget LawStatement := {
  target with planning := .unavailable
}

private def mismatchedTrace : BehaviorTrace := {
  setup := switchSetup
  trace := {
    initialState := offState
    steps := [{
      selectedAction := flipAction
      modelOutcome := appliedOutcome
      resultingState := offState
      observations := [powerOffObservation]
    }]
  }
}

private def mismatchedBehavior : CheckedBehavior := {
  exactTraceBehavior with
  traceExactly := some mismatchedTrace
  semanticDigest := "switch-behavior-target-kernel-mismatch/v1"
}

private def mismatchedDeclaration : QueryDeclaration := {
  exactActionDeclaration with behavior := mismatchedBehavior
}

/-! Query owns bounds, finite-completeness, and exact-trace/kernel mismatch failures. -/
example : [
    queryErrorKind (checkQuery (.ofTarget target) invalidBoundDeclaration),
    queryErrorKind (checkQuery (.ofTarget noFinitePlanningTarget) exhaustiveDeclaration),
    queryErrorKind (checkQuery (.ofTarget target) mismatchedDeclaration)
  ] = [some .invalidBound, some .missingFiniteCompleteness, some .targetKernelMismatch] := by
  native_decide

/-!
Switch's imported golden tests pin its exact Query and artifact bytes and its planner outcomes.
This inventory assertion makes omission of the domain-neutral family an executable failure.
-/
example : compatibilityFamilies = ["switch"] := by
  rfl

end Umpire.Tests.MigrationCompatibility
