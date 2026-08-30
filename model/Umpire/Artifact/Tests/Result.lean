import Umpire.Artifact.Result
import Umpire.Artifact.Tests.Evidence

/-! Evidence and Result exact v2 bytes, checksums, closures, and independent status axes. -/

namespace Umpire.Artifact.Tests.Result

open Umpire
open Umpire.Examples.Switch
open Umpire.Artifact.Tests.Runtime
open Umpire.Artifact.Tests.Evidence

private def id (value : String) : DefinitionId := DefinitionId.of value

private def fingerprint (value : String) : BehaviorFingerprint :=
  (BehaviorFingerprint.parse? value).getD (behaviorFingerprintOf "invalid")

private def checksum (value : String) : ArtifactChecksum :=
  (ArtifactChecksum.parse? value).getD (drivePlanChecksumOf "invalid")

private def emptyChecksum : ArtifactChecksum :=
  checksum "sha256:0000000000000000000000000000000000000000000000000000000000000000"

private def evidenceLimit : ArtifactLimit := { value := 2, unit := "evidence-records" }

private def fieldReference : ArtifactFieldReference := {
  kindDefinitionId := id "umpire.evidence.kind.history"
  fieldDefinitionId := id "umpire.evidence.field.event"
}

private def observationProgram : ArtifactDefinitionReference := {
  definitionId := runtimeConfiguration.observation.programDefinitionId
  behaviorFingerprint := runtimeConfiguration.observation.programBehaviorFingerprint
}

private def mapping : ArtifactDefinitionReference := {
  definitionId := runtimeConfiguration.observation.mappingDefinitionId
  behaviorFingerprint := runtimeConfiguration.observation.mappingBehaviorFingerprint
}

private def initialCoordinate : ArtifactModelCoordinate := {
  kind := "initial-state"
  step := none
  position := none
}

private def initialEvidenceLink : ArtifactEvidenceLink := {
  coordinate := initialCoordinate
  mappingDefinitionId := mapping.definitionId
  mappingVersion := 1
  mappingBehaviorFingerprint := mapping.behaviorFingerprint
  profileDefinitionId := runtimeConfiguration.observation.profileDefinitionId
  profileVersion := 1
  evidenceDefinitionIds := [id "switch.evidence.history.1"]
  ruleDefinitionId := id "switch.observation.rule.initial-state"
  bindingDefinitionIds := []
  orderingSupport := [{
    factDefinitionId := id "switch.evidence.history.1"
    kindDefinitionId := id "umpire.evidence.kind.history"
    ordinal := 0
    causalFactDefinitionIds := []
  }]
  closureSupport := [{
    kindDefinitionId := id "umpire.evidence.kind.history"
    lastOrdinal := 0
  }]
  appliedDispositions := [{
    field := fieldReference
    kind := "retained"
    normalizedValue := some "flip-requested"
    digestPolicyDefinitionId := none
    digestToken := none
  }]
  appliedLimit := evidenceLimit
  meaningBehaviorFingerprint := behaviorFingerprintOf "switch.state.off/v1"
}

private def evidenceDraft : EvidenceArtifact := {
  formatVersion := "umpire-evidence/v2"
  runIdentity := experimentRun.runIdentity
  behaviorFingerprint :=
    fingerprint "sha256:0aa42f873839132836c028886c9be5ad63e5dc66dbc967182ae139159501c8ab"
  experiment := compiledArtifact.artifactBinding
  runtimeConfiguration := runtimeConfiguration.artifactBinding
  run := experimentRun.artifactBinding
  rawEvidence := rawEvidence.artifactBinding
  observationProgram
  mapping
  observationEvaluationStatus := "accepted"
  evidenceBackedModelTrace := some {
    traceId := "switch.trace.accepted"
    observationPlan := observationProgram
    mappingDefinitionId := mapping.definitionId
    mappingVersion := 1
    mappingBehaviorFingerprint := mapping.behaviorFingerprint
    source := {
      path := "Umpire/Artifact/Tests/Result.lean"
      line := 1
      column := 1
      provenance := "lean-model"
    }
    profileDefinitionId := runtimeConfiguration.observation.profileDefinitionId
    profileVersion := 1
    sourceClosed := true
    vocabulary := [{
      definitionId := compiledArtifact.plan.initialState.definitionId
      kind := .state
      canonicalBehavior := "switch.state.off/v1"
    }]
    appliedLimit := evidenceLimit
    evidenceDefinitionIds := [id "switch.evidence.history.1"]
    trace := {
      traceId := "switch.trace.accepted"
      initialState := compiledArtifact.plan.initialState
      steps := []
    }
  }
  evidenceLinks := [initialEvidenceLink]
  dispositions := [{
    field := fieldReference
    disposition := "retain"
    digestPolicyDefinitionId := none
  }, {
    field := {
      kindDefinitionId := id "umpire.evidence.kind.participant-output"
      fieldDefinitionId := id "umpire.evidence.field.rejected"
    }
    disposition := "reject"
    digestPolicyDefinitionId := none
  }]
  diagnostics := []
  knownGaps := []
  provenance := {
    sourceDefinitionIds := [id "switch.evidence.interpreted"]
    sourceLocations := [{
      path := "Umpire/Artifact/Tests/Result.lean"
      line := 1
      column := 1
      provenance := "lean-model"
    }]
  }
  provenanceChecksum := emptyChecksum
  artifactChecksum := emptyChecksum
}

def evidence : EvidenceArtifact := evidenceDraft.seal

private def propertyVerdict (property : PortableProperty) : ArtifactPropertyVerdict := {
  queryDefinitionId := compiledArtifact.plan.queryDefinitionId
  propertyDefinitionId := property.definitionId
  propertyBehaviorFingerprint := property.behaviorFingerprint
  traceId := some "switch.trace.accepted"
  status := "satisfied"
  queryLimits := compiledArtifact.plan.expandedLimits
  evidenceLimit := some evidenceLimit
  provenanceDefinitionIds := [property.definitionId, compiledArtifact.plan.queryDefinitionId]
  clauses := [{
    propertyDefinitionId := property.definitionId
    clauseDefinitionId := id (property.definitionId.value ++ ".clause")
    status := "satisfied"
    coordinates := [initialCoordinate]
    queryLimits := compiledArtifact.plan.expandedLimits
    propertyLimit := some { value := 1, unit := "observation-positions" }
    evidenceLimit
    provenanceDefinitionIds := [property.definitionId]
    evidenceLinks := [initialEvidenceLink]
  }]
  diagnostic := none
}

private def propertyVerdicts : List ArtifactPropertyVerdict :=
  compiledArtifact.properties.map propertyVerdict

private def querySummary : ArtifactQuerySummary := {
  queryDefinitionId := compiledArtifact.plan.queryDefinitionId
  status := "satisfied"
  queryLimits := compiledArtifact.plan.expandedLimits
  requiredPropertyDefinitionIds := compiledArtifact.properties.map PortableProperty.definitionId
  propertyVerdicts
  missingPropertyDefinitionIds := []
  duplicatePropertyDefinitionIds := []
  unexpectedPropertyDefinitionIds := []
  divergentPropertyDefinitionIds := []
  wrongQueryResultDefinitionIds := []
  traceIds := ["switch.trace.accepted"]
}

private def resultDraft : ResultArtifact := {
  formatVersion := "umpire-result/v2"
  runIdentity := experimentRun.runIdentity
  behaviorFingerprint :=
    fingerprint "sha256:f6fbf2847d73f198dd50a9c466e6f1834f67042db0df0a54965c2bcb6b4f7a41"
  experiment := compiledArtifact.artifactBinding
  runtimeConfiguration := runtimeConfiguration.artifactBinding
  run := experimentRun.artifactBinding
  rawEvidence := rawEvidence.artifactBinding
  evidence := evidence.artifactBinding
  operationalStatus := experimentRun.operationalStatus.name
  observationEvaluationStatus := evidence.observationEvaluationStatus
  implementationLink := {
    definitionId := id "switch.implementation-link.system-to-feature"
    behaviorFingerprint :=
      fingerprint "sha256:0ec0f5e52dc5ed18516f1ffb9ae2973a98c5c7469a5482e2f0ef53f522f37d69"
    sourceTarget := {
      definitionId := compiledArtifact.plan.targetDefinitionId
      kind := .target
      behaviorFingerprint := compiledArtifact.plan.targetBehaviorFingerprint
    }
    destinationTarget := {
      definitionId := id "switch.target.feature"
      kind := .target
      behaviorFingerprint :=
        fingerprint "sha256:bf5ea7369835e8267f27e21cc1fb185505c83a6558905fb82b57fb55bd014828"
    }
    diagnostic := none
  }
  implementationLinkStatus := "applied"
  propertyVerdicts
  querySummary
  semanticStatus := "satisfied"
  limits := [
    { stage := "observation-evaluation", limit := evidenceLimit },
    { stage := "query", limit := {
      value := compiledArtifact.plan.expandedLimits.search.value
      unit := compiledArtifact.plan.expandedLimits.search.unit.name
    } }
  ]
  knownGaps := []
  cleanupStatus := experimentRun.cleanup.status.name
  evaluationOutcomeChecksum := none
  provenance := {
    sourceDefinitionIds := [id "switch.result.interpreted"]
    sourceLocations := [{
      path := "Umpire/Artifact/Tests/Result.lean"
      line := 1
      column := 1
      provenance := "lean-model"
    }]
  }
  provenanceChecksum := emptyChecksum
  artifactChecksum := emptyChecksum
}

private def resultWithOutcome : ResultArtifact := {
  resultDraft with evaluationOutcomeChecksum := resultDraft.expectedEvaluationOutcomeChecksum evidence compiledArtifact
}

def result : ResultArtifact := resultWithOutcome.seal

private def incompleteQuerySummary : ArtifactQuerySummary := {
  querySummary with
  status := "incomplete"
  propertyVerdicts := []
  missingPropertyDefinitionIds := querySummary.requiredPropertyDefinitionIds
  traceIds := []
}

private def incompleteResult : ResultArtifact := ({
  resultDraft with
  implementationLinkStatus := "not-evaluated"
  propertyVerdicts := []
  querySummary := incompleteQuerySummary
  semanticStatus := "incomplete"
  evaluationOutcomeChecksum := none
} : ResultArtifact).seal

/-! Lean owns the authoritative interpreted Evidence and Result fixture bytes. -/
example : canonicalEvidenceArtifactBytes evidence = include_str "Fixtures/EvidenceV2.json" := by
  native_decide

example : canonicalResultArtifactBytes result = include_str "Fixtures/ResultV2.json" := by
  native_decide

/-! Every checksum consumes exact deterministic pretty bytes with one terminal LF. -/
example : evidence.hasValidChecksums && result.hasValidChecksums &&
    result.evaluationOutcomeChecksum == result.expectedEvaluationOutcomeChecksum evidence compiledArtifact := by
  native_decide

private def planOnlyMutation : ExperimentSpec :=
  let planDraft := { compiledArtifact.plan with selectionReason := .behaviorSelection }
  let plan := { planDraft with artifactChecksum := planDraft.expectedArtifactChecksum }
  let experimentDraft := { compiledArtifact with plan }
  { experimentDraft with artifactChecksum := experimentDraft.expectedArtifactChecksum }

/-! Stable accepted-outcome identity binds the exact sealed DrivePlan, not only its Properties. -/
example : planOnlyMutation.hasValidArtifactChecksum && planOnlyMutation.plan.hasValidArtifactChecksum &&
    result.expectedEvaluationOutcomeChecksum evidence compiledArtifact !=
      result.expectedEvaluationOutcomeChecksum evidence planOnlyMutation := by
  native_decide

/-! Exact parent bindings close without interpreting RawEvidence or applying an Implementation Link. -/
example : evidence.isValidTransport &&
    evidence.closes compiledArtifact runtimeConfiguration experimentRun rawEvidence &&
    result.isValidTransport &&
    result.closes compiledArtifact runtimeConfiguration experimentRun rawEvidence evidence := by
  native_decide

/-! Accepted Evidence still closes an unresolved Result without inventing an outcome checksum. -/
example : incompleteResult.expectedEvaluationOutcomeChecksum evidence compiledArtifact = none := by
  native_decide

example : incompleteResult.isValidTransport := by
  native_decide

example : incompleteResult.closes
    compiledArtifact runtimeConfiguration experimentRun rawEvidence evidence := by
  native_decide

/-! Operational failure and semantic non-success remain independent transport axes. -/
example :
    ({ result with operationalStatus := "failed" } : ResultArtifact).seal.isValidTransport &&
    !({ result with semanticStatus := "incomplete" } : ResultArtifact).seal.isValidTransport := by
  native_decide

end Umpire.Artifact.Tests.Result
