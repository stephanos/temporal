import Umpire.Case.Projection

/-!
Closed Nexus evidence at the SDK/history boundary. `Binding` freezes one Testpilot Run and the
already-started operation handles it may observe. `Record.toEvent` checks all correlation fields
before preserving the source identity, causal parents, and exact Run Event sequences. No raw
history, callback transport, wall clock, or exploration-selected outcome enters this module.
-/

namespace Temporal.System.Nexus.Evidence

open Umpire Case.Projection

/-- Testpilot Run identity is separate from Temporal workflow execution identity. -/
structure Scope where
  executionId : String
  «namespace» : String
  workflowId : String
  runId : String
  deriving BEq, DecidableEq, Repr

/-- One SDK operation handle, never a workflow cancellation or activation shutdown handle. -/
structure Operation where
  scheduledEventId : Nat
  operationId : String
  requestId : String
  deriving BEq, DecidableEq, Repr

/-- Immutable Run-local correlation authority supplied by the authorized Temporal adapter. -/
structure Binding where
  scope : Scope
  operations : List Operation
  deriving BEq, DecidableEq, Repr

/-- Source order is local to each declared record stream. -/
inductive Source where
  | sdk
  | history
  deriving BEq, DecidableEq, Repr

/-- Workflow cancellation and activation shutdown are explicitly non-operation evidence. -/
inductive Kind where
  | cancellationSubmitted
  | cancellationConfirmed
  | canceled
  | completed
  | unrelated
  | workflowCancellation
  | activationShutdown
  | unsupported (name : String)
  deriving BEq, DecidableEq, Repr

/-- Stable source record identity; its ordinal is not a supporting Run Event sequence. -/
structure SourceEvent where
  scope : Scope
  source : Source
  ordinal : Nat
  deriving BEq, DecidableEq, Repr

/-- Only the declared correlation and provenance survive raw-history projection. -/
structure Record where
  identity : SourceEvent
  operation : Operation
  kind : Kind
  parents : List SourceEvent := []
  runSequences : List Nat
  deriving BEq, DecidableEq, Repr

/-- Correlation failures remain separate from generic projection diagnostics and Property verdicts. -/
inductive Error where
  | invalidBinding
  | wrongScope
  | wrongOperation
  | wrongSource
  | unsupportedEvidence (name : String)
  deriving BEq, DecidableEq, Repr

/-- Stable declaration names shared by correlation admission and the checked correspondence leaf. -/
def field (name : String) : DefinitionId := .of ("temporal.nexus.evidence." ++ name)

/-- Complete scope, including the owning Testpilot Run, used for source deduplication. -/
def Scope.fields (scope : Scope) : List (DefinitionId × String) := [
  (field "execution", scope.executionId), (field "namespace", scope.namespace),
  (field "run", scope.runId), (field "workflow", scope.workflowId)]

/-- Length-safe canonical identity preserves all three operation coordinates without delimiters. -/
def Operation.key (operation : Operation) : String := Lean.Json.compress (.arr #[
  Lean.toJson operation.scheduledEventId, Lean.toJson operation.operationId, Lean.toJson operation.requestId])

/-- Source ownership is closed: SDK submissions cannot masquerade as history confirmations. -/
def Source.id : Source → DefinitionId
  | .sdk => field "sdk"
  | .history => field "history"

/-- Stable evidence kinds participate in the checked projector's canonical provenance. -/
def Kind.id : Kind → DefinitionId
  | .cancellationSubmitted => field "cancellation-submitted"
  | .cancellationConfirmed => field "cancellation-confirmed"
  | .canceled => field "canceled"
  | .completed => field "completed"
  | .unrelated => field "unrelated"
  | .workflowCancellation => field "workflow-cancellation"
  | .activationShutdown => field "activation-shutdown"
  | .unsupported name => field ("unsupported." ++ name)

/-- Preserve the source coordinates without substituting arrival order. -/
def SourceEvent.identity (event : SourceEvent) : Identity :=
  { scope := event.scope.fields, source := event.source.id, ordinal := event.ordinal }

/-- Reject empty, ambiguous, or over-capacity authority before allocating a projection Run. -/
def Binding.validate (binding : Binding) (maxOperations : Nat) : Except Error Unit := do
  if binding.scope.fields.any (fun value => value.2.isEmpty) || binding.operations.isEmpty ||
      binding.operations.length > maxOperations ||
      binding.operations.any (fun operation => operation.scheduledEventId == 0 ||
        operation.operationId.isEmpty || operation.requestId.isEmpty) ||
      (binding.operations.map Operation.scheduledEventId).eraseDups.length != binding.operations.length ||
      (binding.operations.map Operation.operationId).eraseDups.length != binding.operations.length then
    throw .invalidBinding

/-- Validate full correlation and source authority, retaining exact causal and Run support. -/
def Record.toEvent (record : Record) (binding : Binding) : Except Error Event := do
  unless record.identity.scope == binding.scope do throw .wrongScope
  unless binding.operations.contains record.operation do throw .wrongOperation
  if let .unsupported name := record.kind then throw (.unsupportedEvidence name)
  match record.kind, record.identity.source with
  | .cancellationSubmitted, .history => throw .wrongSource
  | .cancellationConfirmed, .sdk | .canceled, .sdk | .completed, .sdk => throw .wrongSource
  | _, _ => pure ()
  pure {
    identity := record.identity.identity
    operation := record.operation.key
    kind := record.kind.id
    parents := record.parents.map SourceEvent.identity
    runSequences := record.runSequences
  }

end Temporal.System.Nexus.Evidence
