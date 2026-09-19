import Umpire.Artifact.Set
import Umpire.Artifact.Tests.Result

/-! Complete Artifact set closure and deterministic manifest regressions. -/

namespace Umpire.Artifact.Tests.Set

open Umpire
open Umpire.Examples.Switch
open Umpire.Artifact.Tests.RunRecord
open Umpire.Artifact.Tests.Evidence
open Umpire.Artifact.Tests.Result

def evaluationSet : ArtifactSet := {
  experiment := compiledArtifact
  runtimeConfiguration
  experimentRun := some experimentRun
  rawEvidence := some rawEvidence
  evidence := some evidence
  result := some result
}

def executableSet : ArtifactSet := {
  experiment := compiledArtifact
  runtimeConfiguration
}

def executionSet : ArtifactSet := {
  executableSet with
  experimentRun := some experimentRun
  rawEvidence := some rawEvidence
}

example : executableSet.isValidClosure && executionSet.isValidClosure &&
    evaluationSet.isValidClosure := by
  native_decide

example :
    executableSet.manifest?.map ArtifactSetManifest.artifactSetIdentity =
      some "umpire.artifact-set.9d8f2d956cb604e0707178a997503734308d9e4114560d0a167bae9e325daff9" &&
    executionSet.manifest?.map ArtifactSetManifest.artifactSetIdentity =
      some "umpire.artifact-set.b90d742a6da8a54292caf52c479ca29a90c1af66baada3a26a333a53b8ca495e" &&
    evaluationSet.manifest?.map ArtifactSetManifest.artifactSetIdentity =
      some "umpire.artifact-set.878837dedb569529846729de78f97fea9d3dddaeb99248bcacec1542875c29d4" := by
  native_decide

example : evaluationSet.manifest?.any fun manifest =>
    manifest.artifactSetChecksum.render ==
      "sha256:aae8228624fb7aee586be45608a68d6b6a0b4573a8e4d7943948fca93f7c872f" &&
    manifest.manifestSha256.render ==
      "sha256:45cb02c642dc4753e0bd3d3b4cfb3ce929513c0dbb2cb98f09bbf7cbc974b783" &&
    canonicalArtifactSetManifestBytes manifest == include_str "Fixtures/ArtifactSetV2.json" := by
  native_decide

/-! Partial and stale document families produce no manifest or partial admitted value. -/
example :
    !({ executionSet with rawEvidence := none } : ArtifactSet).isValidClosure &&
    ({ executionSet with rawEvidence := none } : ArtifactSet).manifest?.isNone &&
    !({ evaluationSet with result := none } : ArtifactSet).isValidClosure &&
    !({ executableSet with runtimeConfiguration := {
      runtimeConfiguration with experiment := runtimeConfiguration.artifactBinding
    }} : ArtifactSet).isValidClosure := by
  native_decide

/-! Lean rejects the same checksum-preserving noncanonical Experiment collections as Go. -/
example :
    let duplicateObservationRequirements := {
      compiledArtifact with
      observationRequirementDefinitionIds :=
        compiledArtifact.observationRequirementDefinitionIds ++
          compiledArtifact.observationRequirementDefinitionIds
    }
    let duplicatePlanCapabilities := {
      compiledArtifact with plan := {
        compiledArtifact.plan with
        capabilityRequirementDefinitionIds :=
          compiledArtifact.plan.capabilityRequirementDefinitionIds ++
            compiledArtifact.plan.capabilityRequirementDefinitionIds
      }
    }
    let reversedProvenance := {
      compiledArtifact with provenance := {
        compiledArtifact.provenance with
        sourceDefinitionIds := compiledArtifact.provenance.sourceDefinitionIds.reverse
      }
    }
    !({ evaluationSet with experiment := duplicateObservationRequirements } : ArtifactSet).isValidClosure &&
    !({ evaluationSet with experiment := duplicatePlanCapabilities } : ArtifactSet).isValidClosure &&
    !({ evaluationSet with experiment := reversedProvenance } : ArtifactSet).isValidClosure := by
  native_decide

/-! Exact member paths and order are part of the admitted manifest, not presentation metadata. -/
example : evaluationSet.manifest?.any fun manifest =>
    !({ manifest with members := manifest.members.reverse }).isValidFor evaluationSet := by
  native_decide

/-! The exact retained Experiment target may close through the Implementation Link destination. -/
example :
    let destinationDraft : ResultArtifact := {
      result with
      implementationLink := {
        result.implementationLink with
        sourceTarget := result.implementationLink.destinationTarget
        destinationTarget := result.implementationLink.sourceTarget
      }
      evaluationOutcomeChecksum := none
    }
    let destinationWithOutcome := {
      destinationDraft with
      evaluationOutcomeChecksum :=
        destinationDraft.expectedEvaluationOutcomeChecksum evidence compiledArtifact
    }
    let destinationResult := destinationWithOutcome.seal
    ({ evaluationSet with result := some destinationResult } : ArtifactSet).isValidClosure := by
  native_decide

/-! A Result may resolve its Implementation Link source only through the retained Experiment. -/
example :
    let staleDraft : ResultArtifact := {
      result with
      implementationLink := {
        result.implementationLink with
        sourceTarget := result.implementationLink.destinationTarget
      }
      evaluationOutcomeChecksum := none
    }
    let staleWithOutcome := {
      staleDraft with
      evaluationOutcomeChecksum := staleDraft.expectedEvaluationOutcomeChecksum evidence compiledArtifact
    }
    let staleResult := staleWithOutcome.seal
    !({ evaluationSet with result := some staleResult } : ArtifactSet).isValidClosure := by
  native_decide

end Umpire.Artifact.Tests.Set
