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
      some "umpire.artifact-set.ff833b3c322019881a67443e013ed5060a09aebd4246edba34dae22cb0d0eb80" &&
    executionSet.manifest?.map ArtifactSetManifest.artifactSetIdentity =
      some "umpire.artifact-set.9e4db86fd3bc6b41654ae0aa2f6601805f20cd0e189ff1821b727d82805674f6" &&
    evaluationSet.manifest?.map ArtifactSetManifest.artifactSetIdentity =
      some "umpire.artifact-set.b235748c7bc21083e1c650186d6f8c98a1bca123b46cd0c62b69f18885567372" := by
  native_decide

example : evaluationSet.manifest?.any fun manifest =>
    manifest.artifactSetChecksum.render ==
      "sha256:48a20a42604e2f6d483562fe886df504ca36b6423bccc86b99833210fb0da593" &&
    manifest.manifestSha256.render ==
      "sha256:cf53d048c8dcdbfe680002ad99e892cb1aebba99ed18bfb12b9d063212160da0" &&
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
