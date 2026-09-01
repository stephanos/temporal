import Temporal.Tool.PortableEvaluationContract
import Umpire.Artifact.Tests.PortableEvaluationContract

namespace Temporal.Tool.PortableEvaluationContractTests

open Umpire
open Umpire.Artifact.PortableEvaluationContract
open Temporal.Tool.PortableEvaluationContract

private theorem normalContract_isSome : normalContract.toOption.isSome = true := by
  native_decide

private def normal : Contract :=
  normalContract.toOption.get normalContract_isSome

private theorem duplicateContract_isSome : duplicateContract.toOption.isSome = true := by
  native_decide

private def duplicate : Contract :=
  duplicateContract.toOption.get duplicateContract_isSome

example : canonicalProtoJSON normal = canonicalProtoJSON normal := by
  rfl

private def normalBytesExact : Bool :=
  match normalContractProtoJSON with
  | .ok bytes => bytes == canonicalProtoJSON normal
  | .error _ => false

example : normalBytesExact = true := by
  native_decide

private def duplicateBytesExact : Bool :=
  match duplicateContractProtoJSON with
  | .ok bytes => bytes == canonicalProtoJSON duplicate
  | .error _ => false

example : duplicateBytesExact = true := by
  native_decide

example : canonicalProtoJSON normal != canonicalProtoJSON duplicate := by
  native_decide

private def unsupportedRejected : Bool :=
  match lowerObservationExpression
      (DefinitionId.of "unsupported.test")
      { path := "PortableEvaluationContractTests.lean", line := 1, column := 1 }
      (.boolean true) with
  | .error failure => failure == {
      sourceDefinitionId := DefinitionId.of "unsupported.test"
      source := { path := "PortableEvaluationContractTests.lean", line := 1, column := 1 }
      construct := "observation.literal-boolean"
    }
  | .ok _ => false

example : unsupportedRejected = true := by
  native_decide

private def definitionMutation : Contract := {
  normal with
  query := { normal.query with definitionId := DefinitionId.of "mutation.definition" }
}

private def fingerprintMutation : Contract := {
  normal with
  query := {
    normal.query with
    behaviorFingerprint := behaviorFingerprintOf "mutation.fingerprint"
  }
}

private def clauseMutation : Contract := {
  normal with
  properties := normal.properties.map fun property => {
    property with
    clauses := property.clauses.map fun clause => {
      clause with definitionId := clause.definitionId ++ ".mutation"
    }
  }
}

private def closureMutation : Contract := {
  normal with
  observation := {
    normal.observation with
    profile := {
      normal.observation.profile with
      sources := normal.observation.profile.sources.drop 1
    }
  }
}

private def limitMutation : Contract := {
  normal with limits := { normal.limits with maxEvaluationWork := normal.limits.maxEvaluationWork + 1 }
}

private def knownGapMutation : Contract := {
  normal with
  knownGaps := {
    kind := .interpretation
    code := "mutation"
    subject := "mutation"
    detail := "mutation"
  } :: normal.knownGaps
}

example :
    [definitionMutation, fingerprintMutation, clauseMutation, closureMutation, limitMutation,
      knownGapMutation].all
      (Umpire.Artifact.Tests.PortableEvaluationContract.mutationChangesBytes normal) = true := by
  native_decide

end Temporal.Tool.PortableEvaluationContractTests
