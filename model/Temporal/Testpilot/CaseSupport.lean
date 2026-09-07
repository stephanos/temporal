import Umpire.Case.Compiler

/-!
Shared Testpilot producers lower their checked inputs through these closed Case construction
mechanics. Runtime coordinates, clients, credentials, and callback authority remain Host-owned.
-/

namespace Temporal.Testpilot.CaseSupport

open Umpire
open Umpire.Case

def source : SourceLocation := {
  path := "Temporal/Testpilot.lean", line := 1, column := 1, provenance := "checked-model"
}

def binding (id fingerprint : String) (kind : CaseDefinitionKind) : CaseDefinitionBinding :=
  { definitionId := id, behaviorFingerprint := fingerprint, kind }

def textType : ValueType := .singular (.scalar .text)
def statusType : ValueType :=
  .singular (.enumeration "temporal.server.api.testpilot.v1.InstructionOutcomeStatus")
def statusOutcome : InstructionOutcomeSchema :=
  { fields := [{ field := .status, type := statusType }] }

def bounds (timeout := 5000) (emitted := 8) : InstructionBounds := {
  timeoutMilliseconds := timeout, maxAttempts := 1, maxEmittedEvents := emitted,
  maxResponseBytes := 4096
}

def field (name : String) : FieldPath := { segments := [{ field := name }] }

def boolean (value : Bool) : ValueExpression := .literal (.boolean value)

def project
    (source : FieldPath)
    (observationId : String)
    (cardinality := ProjectionCardinality.one) : ResponseProjection :=
  { source, cardinality, sinks := [.observation observationId] }

def programLimits : ProgramLimits := {
  maxEntrypoints := 4, maxNodes := 16, maxEdges := 24, maxActivations := 8,
  maxAttempts := 16, maxRunEvents := 256, maxExpressionDepth := 12, maxPathFanout := 32,
  maxRequestBytes := 32768, maxResponseBytes := 4096,
  maxTotalDurationMilliseconds := 30000, maxCleanupDurationMilliseconds := 5000
}

def contractLimits : ContractLimits := {
  maxRules := 4, maxStates := 16, maxTransitions := 16, maxExpressionDepth := 12,
  maxWorkPerEvent := 100000, maxTotalWork := 1000000000,
  maxCaptures := 4, maxCaptureBytes := 8192
}

end Temporal.Testpilot.CaseSupport
