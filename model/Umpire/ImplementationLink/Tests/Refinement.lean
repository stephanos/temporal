import Umpire.ImplementationLink.Refinement

/-! A small pair of tables, one refining the other with a stutter, and the decided witness. -/

namespace Umpire.ImplementationLinkTests.Refinement

open Umpire

inductive Detailed
  | idle
  | busy
  | retrying
  | done
  deriving DecidableEq, Repr

inductive Simple
  | idle
  | busy
  | done
  deriving DecidableEq, Repr

inductive Move
  | go
  | fail
  | retry
  | finish
  deriving DecidableEq, Repr

inductive Seen
  | ok
  deriving DecidableEq, Repr

inductive Note
  | started
  | failed
  | finished
  deriving DecidableEq, Repr

/-- The detailed machine retries: a failure backs it off and the retry returns it to work. Finishing
records that it had failed as well as that it finished. -/
def detailed : FiniteTable Unit Detailed Move Seen Note := {
  setups := [⟨(), "default"⟩]
  states := [⟨.idle, "idle"⟩, ⟨.busy, "busy"⟩, ⟨.retrying, "retrying"⟩, ⟨.done, "done"⟩]
  actions := [⟨.go, "go"⟩, ⟨.fail, "fail"⟩, ⟨.retry, "retry"⟩, ⟨.finish, "finish"⟩]
  outcomes := [⟨.ok, "ok"⟩]
  facts := [⟨.started, "started"⟩, ⟨.failed, "failed"⟩, ⟨.finished, "finished"⟩]
  initial := [⟨(), [.idle]⟩]
  transitions := [
    ⟨"idle-go", .idle, .go, [{ outcome := .ok, state := .busy, facts := [.started] }]⟩,
    ⟨"busy-fail", .busy, .fail, [{ outcome := .ok, state := .retrying, facts := [.failed] }]⟩,
    ⟨"retrying-retry", .retrying, .retry, [{ outcome := .ok, state := .busy, facts := [] }]⟩,
    ⟨"busy-finish", .busy, .finish,
      [{ outcome := .ok, state := .done, facts := [.finished, .failed] }]⟩]
}

/-- The simple machine cannot see a retry, and records only that it finished. -/
def simple : FiniteTable Unit Simple Move Seen Note := {
  setups := [⟨(), "default"⟩]
  states := [⟨.idle, "idle"⟩, ⟨.busy, "busy"⟩, ⟨.done, "done"⟩]
  actions := [⟨.go, "go"⟩, ⟨.finish, "finish"⟩]
  outcomes := [⟨.ok, "ok"⟩]
  facts := [⟨.started, "started"⟩, ⟨.finished, "finished"⟩]
  initial := [⟨(), [.idle]⟩]
  transitions := [
    ⟨"idle-go", .idle, .go, [{ outcome := .ok, state := .busy, facts := [.started] }]⟩,
    ⟨"busy-finish", .busy, .finish, [{ outcome := .ok, state := .done, facts := [.finished] }]⟩]
}

/-- Retrying is still busy; the failure fact has no simple counterpart. -/
def morphism : RefinementMorphism Unit Detailed Seen Note Unit Simple Seen Note := {
  mapSetup := id
  mapState := fun
    | .idle => .idle
    | .busy | .retrying => .busy
    | .done => .done
  mapOutcome := some
  mapObservation := fun
    | .started => some .started
    | .failed => none
    | .finished => some .finished
}

/- The failure and the retry stutter, and the finish is carried although the detailed machine records
more than the simple one. -/
#guard detailed.refines simple morphism

/-- The witness, decided rather than written. -/
theorem refined : TableRefinement detailed simple morphism := .ofChecked (by decide)

/- A trace through the retry is seen by the simple machine as two steps. -/
#guard morphism.visibleStates (SourceAction := Move) .idle [
  { selectedAction := .go, outcome := .ok, state := .busy, facts := [.started] },
  { selectedAction := .fail, outcome := .ok, state := .retrying, facts := [.failed] },
  { selectedAction := .retry, outcome := .ok, state := .busy, facts := [] },
  { selectedAction := .finish, outcome := .ok, state := .done, facts := [.finished, .failed] }] ==
  [.busy, .done]

/-- A map under which backing off is done: the failure row then reaches `done` with a fact the
simple machine's finish does not record, from a state it does not finish from. -/
def wrongMorphism : RefinementMorphism Unit Detailed Seen Note Unit Simple Seen Note := {
  morphism with
  mapState := fun
    | .idle => .idle
    | .busy => .busy
    | .retrying | .done => .done
}

#guard !detailed.refines simple wrongMorphism

/-- A simple machine that does not begin where the detailed one does. -/
def startsBusy : FiniteTable Unit Simple Move Seen Note := { simple with initial := [⟨(), [.busy]⟩] }

#guard !detailed.refines startsBusy morphism

end Umpire.ImplementationLinkTests.Refinement
