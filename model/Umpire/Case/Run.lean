import Umpire.Case.Program

/-!
The immutable Run record produced by executing an Umpire Program.

Event sequence numbers and elapsed times are monotonic coordinates assigned by the Executor. Rule
verdict support cites those coordinates rather than mutable runtime objects.
-/

namespace Umpire.Case

/-- The closed event vocabulary emitted by a version-one Executor. -/
inductive RunEventKind where
  | runOpened
  | activationOpened
  | instructionStarted
  | instructionCompleted
  | instructionTimedOut
  | activationClosed
  | cleanupStarted
  | cleanupCompleted
  | runClosed
  | diagnostic
  deriving BEq, DecidableEq, Repr

/-- Optional context-local coordinates attached to one Run Event. -/
structure RunCoordinates where
  entrypointId : String := ""
  activationId : String := ""
  instructionId : String := ""
  attempt : Nat := 0
  emittedIndex : Nat := 0
  deriving BEq, DecidableEq, Repr

/-- One declared Observation value recorded on a Run Event. -/
structure ObservationValue where
  observationId : String
  value : Value
  deriving BEq, Repr

/-- An immutable fact at one Executor-recorded monotonic coordinate. -/
structure RunEvent where
  sequence : Nat
  elapsedMilliseconds : Nat
  kind : RunEventKind
  coordinates : RunCoordinates
  sourceId : String := ""
  causalSourceIds : List String := []
  outcome : Option InstructionOutcome := none
  observations : List ObservationValue := []
  deriving BEq, Repr

/-- Why execution stopped producing ordinary Program events. -/
inductive RunDisposition where
  | completed
  | stoppedByMonitor
  | incomplete
  deriving BEq, DecidableEq, Repr

/-- The normative outcome of the always-run cleanup graph. -/
inductive RunCleanupStatus where
  | succeeded
  | failed
  | timedOut
  deriving BEq, DecidableEq, Repr

/-- Cleanup status and any diagnostics that explain it. -/
structure CleanupOutcome where
  status : RunCleanupStatus
  diagnosticIds : List String := []
  deriving BEq, Repr

/-- The component or invariant that produced a diagnostic. -/
inductive RunDiagnosticKind where
  | execution
  | monitor
  | recorder
  | invariant
  | limit
  | hostContract
  | postCloseEvent
  deriving BEq, DecidableEq, Repr

/-- One stable diagnostic with an optional supporting Run Event coordinate. -/
structure RunDiagnostic where
  diagnosticId : String
  kind : RunDiagnosticKind
  code : String
  detail : String
  supportingEventSequence : Option Nat := none
  deriving BEq, Repr

/-- The terminal or pending result of one Contract rule. -/
inductive RuleVerdictKind where
  | pending
  | satisfied
  | violated
  | inconclusive
  deriving BEq, DecidableEq, Repr

/-- One rule result and the minimal Run Event coordinates supporting it. -/
structure RuleVerdict where
  ruleId : String
  kind : RuleVerdictKind
  terminalStateId : String := ""
  supportingEventSequences : List Nat := []
  deriving BEq, Repr

/-- The aggregate Contract verdict for one Run. -/
inductive VerdictKind where
  | satisfied
  | violated
  | inconclusive
  deriving BEq, DecidableEq, Repr

/-- The aggregate verdict, per-rule results, and their supporting coordinates. -/
structure Verdict where
  kind : VerdictKind
  rules : List RuleVerdict
  supportingEventSequences : List Nat := []
  deriving BEq, Repr

/-- The authoritative append-only record of one attempted Program execution. -/
structure Run where
  runId : String
  caseId : String
  programId : String
  events : List RunEvent
  disposition : RunDisposition
  cleanup : CleanupOutcome
  verdict : Verdict
  diagnostics : List RunDiagnostic := []
  deriving BEq, Repr

end Umpire.Case
