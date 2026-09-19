import Temporal.Testpilot.CaseSupport
import Umpire.Case.Producer

/-!
# Lifting evidence out of a recorded history read

The generic Producer resolves an `evidence` line into an `EvidenceRule`: which Action a recorded
event confirms, and which recorded data carries it. The Producer declares each kind once on the
Program, so the history read's lift names the declarations of the history kinds among them and
spells nothing itself: the guard is the presence of the declared attributes arm, and the scope, key
and fields are the declaration's. A Run Event kind is lifted by the runtime as it records the event,
and a read kind by the instruction that polls it, so neither is named here.

Every realization builds its history node's evidence target through here, so the rules the Program
lifts and the rules the Contract reads are the same declarations by construction.
-/

namespace Temporal.Case.Evidence

open Umpire
open Temporal.Testpilot.CaseSupport
open Testpilot.Authoring
open temporal.server.api.testpilot.v1

/-- Whether a resolved rule reads a history event, the only kind a history read lifts. -/
def readsHistory (resolved : Umpire.Case.Producer.EvidenceRule) : Bool :=
  match resolved.source.recorded with
  | .historyEvent _ => true
  | _ => false

/-- One resolved history rule as the Program's lift rule: the name of its declaration. -/
def rule (resolved : Umpire.Case.Producer.EvidenceRule) : CorrelatedEvidenceRule :=
  Program.declaredEvidenceRule resolved.source.kindId.value

/-- The history read's evidence target: the history kinds among the resolved rules, in the order
the `evidence` block declared them. -/
def target
    (observationId : String)
    (resolved : List Umpire.Case.Producer.EvidenceRule) : ReadTarget :=
  Program.correlatedEvidenceTarget observationId
    ((resolved.filter readsHistory).map rule).toArray

end Temporal.Case.Evidence
