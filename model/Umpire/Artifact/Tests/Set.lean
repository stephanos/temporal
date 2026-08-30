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
      some "umpire.artifact-set.4b7c7fb8319e64bbab53abc7f0f73f3b22733b08c11caa9cbd508fe1f59c7775" &&
    executionSet.manifest?.map ArtifactSetManifest.artifactSetIdentity =
      some "umpire.artifact-set.3dda4efe07ac24ef454f7dc4227440277cb59caf4a4d671ac09d5bc11555f2f0" &&
    evaluationSet.manifest?.map ArtifactSetManifest.artifactSetIdentity =
      some "umpire.artifact-set.c5ecf27335cb3441e768818b0eb4025eeed3de5c6bdea6e0f3dc2a66f9922c69" := by
  native_decide

example : evaluationSet.manifest?.any fun manifest =>
    manifest.artifactSetChecksum.render ==
      "sha256:084ed42fd042c6c08bd62a1e5942a15a0c53a3da813041965f3d58c1cb73d172" &&
    manifest.manifestSha256.render ==
      "sha256:ed15c7a3c10b3442d769ad99496e07c57f70648d1aa43197857b220185f5ce66" &&
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
