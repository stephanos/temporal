import Umpire.Replay.Edits
import Umpire.Exploration.Promotion

/-!
# One reduction of one violated Query

A replay subject is a violated Run of a Case that one admitted Query produced. Reduction takes that
Query's Scenario through one sweep of `dropPrefixStep` edits, last prefix step first, each tried
once against the candidate retained so far: the edited Query is re-admitted through the Query's own
source; one the Model does not admit is `inapplicable` and produces no Case; an admitted one is
produced as a whole Case, named by its Plan checksum, which a coordinator runs twice and classes.
A candidate whose Runs reproduce the subject's violation is retained and becomes what the next edit
applies to; one that does not is recorded and never retried; a dropped step is never reintroduced.

The sweep ends `minimized` when at least one edit was retained, `irreducible` when none was, and
`incomplete` when an edit stays undecided, the coordinator stops early or the edit cap cut the
sweep short. `minimized` means no single edit of that one sweep reproduced, not that no subset
would: nothing is re-enumerated after a retention.

Everything here is pure: the reduction is a value, and the classes it takes are what a coordinator
decided about Runs it never shows here.
-/

namespace Umpire.Replay

open Umpire.Command

/-- What one edit came to. -/
inductive Fate where
  /-- The Model does not admit the edited Query; no Case was produced. -/
  | inapplicable (reason : String)
  /-- The edited Query was admitted, and its Case was rejected by the Producer or by preparation. -/
  | rejected (reason : String)
  /-- Both Runs reproduced the subject's violation; the candidate is what later edits apply to. -/
  | retained
  /-- A Run said conclusively that the violation is not the subject's. -/
  | notReproduced
  /-- A Run stayed indeterminate after its one retry; the reduction ends incomplete here. -/
  | undecided
  deriving BEq, Repr

def Fate.name : Fate → String
  | .inapplicable _ => "inapplicable"
  | .rejected _ => "rejected"
  | .retained => "retained"
  | .notReproduced => "not-reproduced"
  | .undecided => "undecided"

def Fate.reason : Fate → String
  | .inapplicable reason | .rejected reason => reason
  | _ => ""

/-- One edit with its fate and, where a Case was produced, the candidate's digest. -/
structure Settled where
  edit : Edit
  fate : Fate
  candidate : Option String := none
  deriving BEq, Repr

inductive Result where
  | minimized
  | irreducible
  | incomplete (reason : String)
  deriving BEq, Repr

def Result.name : Result → String
  | .minimized => "minimized"
  | .irreducible => "irreducible"
  | .incomplete _ => "incomplete"

def Result.reason : Result → String
  | .incomplete reason => reason
  | _ => ""

/-- The most edits one sweep enumerates. -/
def editCap : Nat := 8

/-- One sweep in progress: the edits in order, the positions the retained candidate keeps, what
each tried edit came to, what is left to try, and the end, once it is decided early. -/
structure Reduction where
  sweep : List Edit
  /-- More prefix steps than the cap: the sweep cannot end minimized or irreducible. -/
  capped : Bool
  kept : List Nat
  settled : List Settled := []
  pending : List Edit
  ended : Option Result := none
  deriving BEq, Repr

namespace Reduction

/-- The sweep over a subject whose exact action sequence is `actions`. -/
def start (actions : List DefinitionId) (cap : Nat := editCap) : Reduction :=
  let all := sweepOf actions
  { sweep := all.take cap
    capped := all.length > cap
    kept := List.range actions.length
    pending := all.take cap }

/-- The next edit to try, unless the reduction has ended. -/
def next? (reduction : Reduction) : Option Edit :=
  if reduction.ended.isSome then none else reduction.pending.head?

/-- The positions a candidate of `edit` keeps: the retained candidate's, less the edit's. -/
def candidateKept (reduction : Reduction) (edit : Edit) : List Nat :=
  reduction.kept.erase edit.index

/-- Record what `edit` came to. A retained edit's step is dropped for good; an undecided edit ends
the reduction incomplete, naming it. -/
def settle (reduction : Reduction) (edit : Edit) (fate : Fate) (candidate : Option String := none) :
    Reduction :=
  { reduction with
    pending := reduction.pending.erase edit
    settled := reduction.settled ++ [{ edit, fate, candidate }]
    kept := if fate == .retained then reduction.kept.erase edit.index else reduction.kept
    ended := match reduction.ended, fate with
      | some ended, _ => some ended
      | none, .undecided => some (.incomplete s!"{edit.name} is undecided")
      | none, _ => none }

/-- End the reduction early; the first end decided stands. -/
def stop (reduction : Reduction) (reason : String) : Reduction :=
  if reduction.ended.isSome then reduction else { reduction with ended := some (.incomplete reason) }

/-- How the sweep ended. -/
def result (reduction : Reduction) : Result :=
  match reduction.ended with
  | some ended => ended
  | none =>
      if !reduction.pending.isEmpty then .incomplete "the sweep did not finish"
      else if reduction.capped then .incomplete s!"the sweep was capped at {editCap} edits"
      else if reduction.settled.any (·.fate == .retained) then .minimized
      else .irreducible

/-- Settling never brings a dropped step back: the retained positions only shrink. -/
theorem settle_kept_sublist (reduction : Reduction) (edit : Edit) (fate : Fate)
    (candidate : Option String) : (reduction.settle edit fate candidate).kept.Sublist reduction.kept := by
  unfold settle
  split
  · exact List.erase_sublist
  · exact List.Sublist.refl _

/-- Settling never lets an edit be tried again: what is pending only shrinks. -/
theorem settle_pending_sublist (reduction : Reduction) (edit : Edit) (fate : Fate)
    (candidate : Option String) :
    (reduction.settle edit fate candidate).pending.Sublist reduction.pending :=
  List.erase_sublist

end Reduction

/-! ### Admitting and producing a candidate -/

variable {Setup State Action Outcome Fact : Type}
variable [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
variable [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
variable [DecidableEq Outcome] [DecidableEq Fact]
variable {model : DeclaredModel Setup State Action Outcome Fact}

/-- What produces a Case of the Query: the realization and what the `case` block produces under. -/
structure Production where
  realization : Umpire.Case.Producer.Realization
  evidence : Umpire.Case.Producer.Vocabulary → List Umpire.Case.Producer.EvidenceMapping :=
    fun _ => []
  claims : List Umpire.Case.Producer.ClassClaim := []
  catalog : List (String × String) := []
  relations : List Umpire.Case.Producer.FieldRelation := []

/-- One admitted form of the Query: the admission, the Plan the search delivered and its digest,
which names the Case produced from it. -/
structure Admitted (model : DeclaredModel Setup State Action Outcome Fact) where
  admitted : AdmittedModel model
  plan : Plan
  digest : String

/-- Why the Model did not admit an edited Query, briefly. -/
def admissionReason : AdmissionError → String
  | .notSelected .. => "not-selected: the Model selects no trace of the edited Scenario"
  | .invalidTarget _ => "invalid-target"
  | .invalidVocabulary _ => "invalid-vocabulary"
  | .admission diagnostic =>
      "admission: " ++ String.ofList (((repr diagnostic).pretty).toList.take 240)
  | .instances reason => "instances: " ++ reason

/-- Admit the Query with its Scenario restricted to `kept`. -/
def admitKept (source : QuerySource model) (kept : List Nat) : Except String (Admitted model) := do
  let admitted ← (restrictSource source kept).admit.mapError admissionReason
  let some plan := admitted.checked.run.artifact
    | throw "the admitted Query carries no Plan"
  pure { admitted, plan, digest := Umpire.Exploration.candidateDigest plan.artifactChecksum }

/-- Produce the admitted form's Case under `identity`. Evidence lines for an Action the edited path
no longer selects are left out, since a line for an unselected Action is a production error. -/
def Admitted.produce (admitted : Admitted model) (identity : Umpire.Case.Producer.Identity)
    (production : Production) :
    Except Umpire.Case.Compiler.Error temporal.server.api.testpilot.v1.Case :=
  let checked := admitted.admitted.checked
  let selected := checked.behavior.actionsExactly.getD []
  Umpire.Command.produce checked identity production.realization
    (fun vocabulary => (production.evidence vocabulary).filter fun mapping =>
      selected.contains mapping.action.definitionId)
    (claims := production.claims) (evidenceCatalog := production.catalog)
    (relations := production.relations)

end Umpire.Replay
