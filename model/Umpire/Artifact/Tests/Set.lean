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
      some "umpire.artifact-set.9d8f2d956cb604e0707178a997503734308d9e4114560d0a167bae9e325daff9" &&
    executionSet.manifest?.map ArtifactSetManifest.artifactSetIdentity =
      some "umpire.artifact-set.b90d742a6da8a54292caf52c479ca29a90c1af66baada3a26a333a53b8ca495e" &&
    evaluationSet.manifest?.map ArtifactSetManifest.artifactSetIdentity =
      some "umpire.artifact-set.6f3d719ac73050750cacfcb82ce20c442c2ff5089a2e4e9dd6030454b047dcbd" := by
  native_decide

example : evaluationSet.manifest?.any fun manifest =>
    manifest.artifactSetChecksum.render ==
      "sha256:f4ee0216139e1f58c53a4a91167a8b8da575b985629f17b6dca7f95b4816b67f" &&
    manifest.manifestSha256.render ==
      "sha256:d64ba1932f1929850fa74b9e5c8cc29f4fd24bc514bb4255514638ead1bef6fe" &&
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
