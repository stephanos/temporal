import Umpire.Artifact.Set
import Umpire.Artifact.Tests.Result

/-! Complete Artifact set closure and deterministic manifest regressions. -/

namespace Umpire.Artifact.Tests.Set

open Umpire
open Umpire.Examples.Switch
open Umpire.Artifact.Tests.Runtime
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
      some "umpire.artifact-set.cbfd3dff30a3aae089edc0aaa0cb086861711344cbe665b45f0f7a7e43bf4dd1" &&
    executionSet.manifest?.map ArtifactSetManifest.artifactSetIdentity =
      some "umpire.artifact-set.60d47ed590ca697358d7c37458c05971cf3d612e9815faf7da0aec8182bda8a0" &&
    evaluationSet.manifest?.map ArtifactSetManifest.artifactSetIdentity =
      some "umpire.artifact-set.548b325722a3b47d5f1b1367c9ad90ab98d16ff5a5f9130c8c17c7619411db65" := by
  native_decide

example : evaluationSet.manifest?.any fun manifest =>
    manifest.artifactSetChecksum.render ==
      "sha256:d1387dc8e955186e2c033bb19633d1c528f0ce1a2b4e2bb4224e479d32d934fb" &&
    manifest.manifestSha256.render ==
      "sha256:6bde84aec927962a97f4a40d1dad9d0c9e6609aeaa2846892d3324518ff7967c" &&
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
