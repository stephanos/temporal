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

def field (name : String) : String := Path.make #[Path.field name]

def boolean (value : Bool) : Expression :=
  Expr.literal (Value.boolean value)

def project
    (source : String)
    (observationId : String)
    (cardinality : ReadCardinality := .READ_CARDINALITY_ONE) : ResponseRead :=
  Program.responseRead source cardinality #[Program.observationTarget observationId]

/-- The history event message a history read projects into one Observation per event. -/
def historyEventNode := "temporal.api.history.v1.HistoryEvent"

def historyEventType : ValueType :=
  Types.singular (Types.messageType historyEventNode)

def nested (names : List String) : String :=
  Path.make (names.map Path.field).toArray

def historyEvents : String :=
  Path.make #[Path.field "history", Path.repeated "events"]

def historyAttribute (selected name : String) : String :=
  Path.make #[Path.oneofMember "attributes" selected, Path.field name]

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
def projected (value : Expression) (path : String) : Expression :=
  Expr.path value path

def assign (target : String) (value : Expression) : RequestAssignment :=
  Program.requestAssignment target value

def provenance (producerId producerVersion : String)
    (definitions : List Umpire.Provenance.DefinitionBinding)
    (sources : List Umpire.SourceLocation)
    (knownGaps : List Umpire.Provenance.KnownGap) : Except Umpire.SourceLocation CaseProvenance :=
  Umpire.Provenance.make { producerId, producerVersion, definitions, sources, knownGaps }

end Temporal.Testpilot.CaseSupport
