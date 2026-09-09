import Testpilot.Authoring
import Umpire.Case.Provenance

/-!
Shared Testpilot producers lower their checked inputs through these closed Case construction
mechanics. Runtime coordinates, clients, credentials, and callback authority remain Host-owned.
-/

namespace Temporal.Testpilot.CaseSupport

open temporal.server.api.testpilot.v1
open Testpilot.Authoring

def source : Umpire.SourceLocation := {
  path := "Temporal/Testpilot.lean", line := 1, column := 1, provenance := "checked-model"
}

def binding (id fingerprint : String) (kind : Umpire.Case.CaseDefinitionKind) :
    Umpire.Case.CaseDefinitionBinding :=
  { definitionId := id, behaviorFingerprint := fingerprint, kind }

def textType : ValueType := Types.singular (Types.scalar .SCALAR_KIND_TEXT)
def statusType : ValueType :=
  Types.singular (Types.enumeration "temporal.server.api.testpilot.v1.InstructionOutcomeStatus")
def statusOutcome : InstructionOutcomeDefinition :=
  Program.outcome #[Program.outcomeField .INSTRUCTION_OUTCOME_FIELD_STATUS statusType]

def bounds (timeout : Int64 := 5000) (emitted : Int64 := 8) : InstructionLimits :=
  Program.instructionLimits timeout 1 emitted 4096

def field (name : String) : FieldPath := Path.make #[Path.field name]

def boolean (value : Bool) : ProgramExpression :=
  ProgramExpr.literal (Value.boolean value)

def project
    (source : FieldPath)
    (observationId : String)
    (cardinality : ProjectionKind := .PROJECTION_KIND_ONE) : ResponseProjection :=
  Program.responseProjection source cardinality #[Program.observationTarget observationId]

def historyEventType : ValueType :=
  Types.singular (Types.messageType "temporal.api.history.v1.HistoryEvent")

def nested (names : List String) : FieldPath :=
  Path.make (names.map Path.field).toArray

def historyEvents : FieldPath :=
  Path.make #[Path.field "history", Path.repeated "events"]

def historyAttribute (selected name : String) : FieldPath :=
  Path.make #[Path.oneofSelector "attributes" selected, Path.field name]

def text (value : String) : ProgramExpression := ProgramExpr.literal (Value.text value)
def signedInteger (value : Int) : ProgramExpression :=
  ProgramExpr.literal (Value.signedInteger value)
def observed (id : String) : ContractExpression := ContractExpr.observation id
def captured (id : String) : ContractExpression := ContractExpr.capture id
def runId : ProgramExpression := ProgramExpr.run
def projected (value : ContractExpression) (path : FieldPath) : ContractExpression :=
  ContractExpr.path value path

def succeeded (entrypoint instruction : String) : ProgramExpression :=
  let status := ProgramExpr.outcome (Ref.instruction entrypoint instruction)
    .INSTRUCTION_OUTCOME_FIELD_STATUS
  ProgramExpr.all #[ProgramExpr.present status,
    ProgramExpr.equals status (ProgramExpr.literal (Value.enumeration 1))]

def assign (target : FieldPath) (value : ProgramExpression) : RequestAssignment :=
  Program.requestAssignment target value

def programLimits : ProgramLimits :=
  Program.limits 4 16 24 8 16 256 12 32 32768 4096 30000 5000

def contractLimits : ContractLimits :=
  Monitor.limits 4 16 16 12 100000 1000000000 4 8192

def provenance (producerId producerVersion : String)
    (definitions : List Umpire.Case.CaseDefinitionBinding)
    (sources : List Umpire.SourceLocation)
    (knownGaps : List Umpire.Case.CaseKnownGap) : CaseProvenance :=
  Umpire.Case.Provenance.make { producerId, producerVersion, definitions, sources, knownGaps }

end Temporal.Testpilot.CaseSupport
