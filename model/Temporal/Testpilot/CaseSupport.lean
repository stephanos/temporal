import Testpilot.Authoring
import Umpire.Provenance
import Umpire.Operation

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

def binding (id fingerprint : String) (kind : Umpire.Provenance.DefinitionKind) :
    Umpire.Provenance.DefinitionBinding :=
  { definitionId := id, behaviorFingerprint := fingerprint, kind }

def textType : ValueType := Types.singular (Types.scalar .SCALAR_KIND_TEXT)
def statusType : ValueType :=
  Types.singular (Types.enumeration "temporal.server.api.testpilot.v1.InstructionOutcomeStatus")
def statusOutcome : InstructionOutcomeDefinition :=
  Program.outcome #[Program.outcomeField .INSTRUCTION_OUTCOME_FIELD_STATUS statusType]

def bounds (timeout : Int64 := 5000) (emitted : Int64 := 8) : InstructionLimits :=
  Program.instructionLimits timeout 1 emitted 4096

def field (name : String) : FieldPath := Path.make #[Path.field name]

def boolean (value : Bool) : Expression :=
  Expr.literal (Value.boolean value)

def project
    (source : FieldPath)
    (observationId : String)
    (cardinality : ReadCardinality := .READ_CARDINALITY_ONE) : ResponseRead :=
  Program.responseRead source cardinality #[Program.observationTarget observationId]

/-- The history event message a history read projects into one Observation per event. -/
def historyEventNode := "temporal.api.history.v1.HistoryEvent"

def historyEventType : ValueType :=
  Types.singular (Types.messageType historyEventNode)

def nested (names : List String) : FieldPath :=
  Path.make (names.map Path.field).toArray

def historyEvents : FieldPath :=
  Path.make #[Path.field "history", Path.repeated "events"]

def historyAttribute (selected name : String) : FieldPath :=
  Path.make #[Path.oneofSelector "attributes" selected, Path.field name]

/-- The gRPC transport path of a checked generated method, derived from its own admitted full name
rather than copied beside it. -/
def methodPath (schema : Umpire.Operation.RpcSchema) : String :=
  let segments := schema.fullName.splitOn "."
  "/" ++ ".".intercalate segments.dropLast ++ "/" ++ (segments.getLast?.getD "")

def text (value : String) : Expression := Expr.literal (Value.text value)
def signedInteger (value : Int) : Expression :=
  Expr.literal (Value.signedInteger value)
def observed (id : String) : Expression := Expr.observation id
def captured (id : String) : Expression := Expr.capture id
def runId : Expression := Expr.run
def projected (value : Expression) (path : FieldPath) : Expression :=
  Expr.path value path

def succeeded (entrypoint instruction : String) : Expression :=
  let status := Expr.outcome (Ref.instruction entrypoint instruction)
    .INSTRUCTION_OUTCOME_FIELD_STATUS
  Expr.all #[Expr.present status,
    Expr.equal status (Expr.literal (Value.enumeration 1))]

def assign (target : FieldPath) (value : Expression) : RequestAssignment :=
  Program.requestAssignment target value

def programLimits : ProgramLimits :=
  Program.limits 4 16 24 8 16 256 12 32 32768 4096 30000 5000

def contractLimits : ContractLimits :=
  Contract.limits 4 16 16 12 100000 1000000000 4 8192

def provenance (producerId producerVersion : String)
    (definitions : List Umpire.Provenance.DefinitionBinding)
    (sources : List Umpire.SourceLocation)
    (knownGaps : List Umpire.Provenance.KnownGap) : CaseProvenance :=
  Umpire.Provenance.make { producerId, producerVersion, definitions, sources, knownGaps }

end Temporal.Testpilot.CaseSupport
