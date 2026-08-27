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
    (definitionId owner : DefinitionId)
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
  definitionId
  path := { role, owner }
}

private def authoringAt (line column : Nat) : AuthoredTarget LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  targetAuthoring.withOccurrences [occurrenceAt targetId targetId .targetDefinition line column]

private def checkedSummary
    (result : Except AuthoringDiagnostic (QueryTarget LawStatement)) :
    Option (String × String × Option (List ModelValue × String × String)) :=
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

private def earlyQueryResult : Except QueryError (CheckedQuery LawStatement) :=
  checkQuery (.ofTarget earlyTarget) exactActionDeclaration

private theorem earlyQueryResult_isSome : earlyQueryResult.toOption.isSome = true := by
  native_decide

private def materializeEarlyQuery
    (checked : CheckedQuery LawStatement) : CheckedQuery LawStatement := {
  checked with
  target := earlyTarget
  completeness := (CheckedQueryTarget.ofTarget earlyTarget).completeness
}

private def earlyQuery : CheckedQuery LawStatement :=
  materializeEarlyQuery (earlyQueryResult.toOption.get earlyQueryResult_isSome)

private def earlyKernel? : Option (IncrementalPlannerKernel earlyQuery.target) :=
  IncrementalPlannerKernel.ofCheckedQuery? earlyQuery
    (by
      intro evidence evidenceEq
      simp [earlyQuery, materializeEarlyQuery, CheckedQueryTarget.ofTarget, earlyTarget,
        checkedTarget, authoringAt, AuthoredTarget.withOccurrences, targetAuthoring,
        AuthoredTarget.make, targetDefinition] at evidenceEq
      cases Option.some.inj evidenceEq
      simp [finitePlanning])
    (by
      intro _ _ setup
      simp only [earlyQuery, materializeEarlyQuery, earlyTarget, checkedTarget, authoringAt,
        AuthoredTarget.withOccurrences, targetAuthoring, AuthoredTarget.make, targetDefinition,
        transitionKernel, initialStates]
      split <;> simp)
    (by
      intro _ _ state action
      by_cases selectedAction : action = flipAction
      · subst action
        by_cases selectedOff : state = offState
        · subst state
          simpa [earlyQuery, materializeEarlyQuery, earlyTarget, checkedTarget, authoringAt,
            AuthoredTarget.withOccurrences, targetAuthoring, AuthoredTarget.make,
            targetDefinition, transitionKernel, stepResults] using appliedResult_ordered
        · by_cases selectedOn : state = onState
          · subst state
            simpa [earlyQuery, materializeEarlyQuery, earlyTarget, checkedTarget, authoringAt,
              AuthoredTarget.withOccurrences, targetAuthoring, AuthoredTarget.make,
              targetDefinition, transitionKernel, stepResults, onState_ne_offState] using
                appliedFromOnResult_ordered
          · simp [earlyQuery, materializeEarlyQuery, earlyTarget, checkedTarget, authoringAt,
              AuthoredTarget.withOccurrences, targetAuthoring, AuthoredTarget.make,
              targetDefinition, transitionKernel, stepResults, selectedOff, selectedOn]
      · simp [earlyQuery, materializeEarlyQuery, earlyTarget, checkedTarget, authoringAt,
          AuthoredTarget.withOccurrences, targetAuthoring, AuthoredTarget.make,
          targetDefinition, transitionKernel, stepResults, selectedAction])

private theorem earlyKernel?_isSome : earlyKernel?.isSome = true := by
  rfl

private def earlyKernel : IncrementalPlannerKernel earlyQuery.target :=
  earlyKernel?.get earlyKernel?_isSome

private def earlyRun : PlannerRun := plan earlyQuery earlyKernel

private def relocatedQueryResult : Except QueryError (CheckedQuery LawStatement) :=
  checkQuery (.ofTarget relocatedTarget) exactActionDeclaration

private theorem relocatedQueryResult_isSome :
    relocatedQueryResult.toOption.isSome = true := by
  native_decide

private def materializeRelocatedQuery
    (checked : CheckedQuery LawStatement) : CheckedQuery LawStatement := {
  checked with
  target := relocatedTarget
  completeness := (CheckedQueryTarget.ofTarget relocatedTarget).completeness
}

private def relocatedQuery : CheckedQuery LawStatement :=
  materializeRelocatedQuery
    (relocatedQueryResult.toOption.get relocatedQueryResult_isSome)

private def relocatedKernel? : Option (IncrementalPlannerKernel relocatedQuery.target) :=
  IncrementalPlannerKernel.ofCheckedQuery? relocatedQuery
    (by
      intro evidence evidenceEq
      simp [relocatedQuery, materializeRelocatedQuery, CheckedQueryTarget.ofTarget,
        relocatedTarget, checkedTarget, authoringAt, AuthoredTarget.withOccurrences,
        targetAuthoring, AuthoredTarget.make, targetDefinition] at evidenceEq
      cases Option.some.inj evidenceEq
      simp [finitePlanning])
    (by
      intro _ _ setup
      simp only [relocatedQuery, materializeRelocatedQuery, relocatedTarget, checkedTarget,
        authoringAt, AuthoredTarget.withOccurrences, targetAuthoring, AuthoredTarget.make,
        targetDefinition, transitionKernel, initialStates]
      split <;> simp)
    (by
      intro _ _ state action
      by_cases selectedAction : action = flipAction
      · subst action
        by_cases selectedOff : state = offState
        · subst state
          simpa [relocatedQuery, materializeRelocatedQuery, relocatedTarget, checkedTarget,
            authoringAt, AuthoredTarget.withOccurrences, targetAuthoring, AuthoredTarget.make,
            targetDefinition, transitionKernel, stepResults] using
              appliedResult_ordered
        · by_cases selectedOn : state = onState
          · subst state
            simpa [relocatedQuery, materializeRelocatedQuery, relocatedTarget, checkedTarget,
              authoringAt, AuthoredTarget.withOccurrences, targetAuthoring,
              AuthoredTarget.make, targetDefinition, transitionKernel, stepResults,
              onState_ne_offState] using appliedFromOnResult_ordered
          · simp [relocatedQuery, materializeRelocatedQuery, relocatedTarget, checkedTarget,
              authoringAt, AuthoredTarget.withOccurrences, targetAuthoring,
              AuthoredTarget.make, targetDefinition, transitionKernel, stepResults,
              selectedOff, selectedOn]
      · simp [relocatedQuery, materializeRelocatedQuery, relocatedTarget, checkedTarget,
          authoringAt, AuthoredTarget.withOccurrences, targetAuthoring, AuthoredTarget.make,
          targetDefinition, transitionKernel, stepResults, selectedAction])

private theorem relocatedKernel?_isSome : relocatedKernel?.isSome = true := by
  rfl

private def relocatedKernel : IncrementalPlannerKernel relocatedQuery.target :=
  relocatedKernel?.get relocatedKernel?_isSome

private def relocatedRun : PlannerRun := plan relocatedQuery relocatedKernel

private def expectedSwitchArtifactJson : String :=
  include_str "../Examples/Fixtures/SwitchCompiledArtifact.json"

/-! Planning both layouts preserves the committed canonical artifact bytes. -/
example : [
    earlyRun.artifact.map (fun artifact => canonicalExperimentSpecJson artifact ++ "\n"),
    relocatedRun.artifact.map (fun artifact => canonicalExperimentSpecJson artifact ++ "\n")
  ] = [some expectedSwitchArtifactJson, some expectedSwitchArtifactJson] := by
  native_decide

private def wrongKindDefinition : TargetDefinition
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  targetDefinition with requiredCapabilities := [flipActionId]
}

private def wrongKindAuthoringAt (line column : Nat) : AuthoredTarget LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  AuthoredTarget.make wrongKindDefinition targetComposition (occurrences := [
    occurrenceAt flipActionId targetId .capabilityRequirement line column
  ])

private def diagnosticSummary
    (result : Except AuthoringDiagnostic (QueryTarget LawStatement)) :
    Option (DefinitionErrorKind × AuthoringOccurrenceRole × String × Nat × Nat) :=
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

example : wrongKindDefinition.source = source := by
  rfl

private def expertTargetDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  id := targetDefinition.id
  source := targetDefinition.source
  definitions := targetDefinition.definitions
  requiredCapabilities := targetDefinition.requiredCapabilities
  providers := [switchProvider]
  connectors := []
  resolvedSetups := targetDefinition.resolvedSetups
  kernel := targetDefinition.kernel
}

private def invalidDefinitionIdMetadata : DefinitionMetadata := {
  id := DefinitionId.of "action"
  kind := .action
  source
  contractDigest := "invalid-definition-id/v1"
}

private def invalidDefinitionIdTarget : TargetDeclaration LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  expertTargetDeclaration with
  definitions := invalidDefinitionIdMetadata :: expertTargetDeclaration.definitions
}

private def missingProviderDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  expertTargetDeclaration with providers := []
}

private def providerWithoutLaw : CapabilityProvider LawStatement := {
  switchProvider with lawWitnesses := []
}

private def missingLawDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  expertTargetDeclaration with providers := [providerWithoutLaw]
}

private def incompleteKernelDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  expertTargetDeclaration with
  kernel := .incomplete transitionKernel.metadata
    [DefinitionId.of "umpire.kernel-proof.step-complete"]
}

private def targetErrorKind
    (result : Except DefinitionError (QueryTarget LawStatement)) : Option DefinitionErrorKind :=
  match result with
  | .ok _ => none
  | .error failure => some failure.kind

/-! Target-owned invalid Definition IDs, providers, laws, and kernel availability stay typed at Target. -/
example : [
    targetErrorKind (composeTarget invalidDefinitionIdTarget),
    targetErrorKind (composeTarget missingProviderDeclaration),
    targetErrorKind (composeTarget missingLawDeclaration),
    targetErrorKind (composeTarget incompleteKernelDeclaration)
  ] = [some .invalidDefinitionId, some .missingProvider, some .missingLaw, some .incompleteKernel] := by
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

private def noFinitePlanningTarget : QueryTarget LawStatement :=
  checkedTarget targetAuthoring.withoutPlanning

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
