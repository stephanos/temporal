import Temporal.Testpilot.CaseSupport
import Umpire.Case.Producer

/-!
# Lifting evidence out of a recorded history read

The generic Producer resolves an `evidence` line into an `EvidenceRule`: which Action a recorded
event confirms, and which generated attributes field carries it. Turning that into the Program's
projection target is Temporal's job, because the guard is a path into the `HistoryEvent.Attributes`
oneof and the scope is this Case's own Run.

Every realization builds its history node's evidence target through here, so the rules the Program
lifts and the rules the Contract reads are the same list by construction.
-/

namespace Temporal.Case.Evidence

open Umpire
open Temporal.Testpilot.CaseSupport
open Testpilot.Authoring
open temporal.server.api.testpilot.v1

/-- One resolved evidence rule as the Program's lift rule. The guard selects the oneof arm, so the
kind is the literal the selected shape denotes rather than a value read out of it. -/
def rule
    (scopeField : DefinitionId)
    (identity : Umpire.Case.Producer.Identity)
    (resolved : Umpire.Case.Producer.EvidenceRule) : CorrelatedEvidenceRule :=
  Program.correlatedEvidenceRule
    (guard := Path.make #[Path.oneofSelector "attributes" resolved.source.attributesField])
    (source := resolved.source.sourceId.value)
    (kind := resolved.source.kindId.value)
    (operation := resolved.source.operationKeyPath)
    (scope := #[Program.correlatedEvidenceLiteral scopeField.value identity.runScope])

/-- The history read's evidence target, in the order the `evidence` block declared. -/
def target
    (scopeField : DefinitionId)
    (observationId : String)
    (identity : Umpire.Case.Producer.Identity)
    (resolved : List Umpire.Case.Producer.EvidenceRule) : ProjectionTarget :=
  Program.correlatedEvidenceTarget observationId
    (resolved.map (rule scopeField identity)).toArray

end Temporal.Case.Evidence
