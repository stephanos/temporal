import Umpire.Replay
import Umpire.Exploration.Tests.Classed
import Umpire.Examples.Switch

/-!
# What one sweep of edits does

The lamp pins an admitted edit: toggling a dim lamp hard holds it dim, so dropping that prefix step
still reaches the soft toggle that brightens it, and the edited Query is admitted with one step and
a Plan of its own. The switch pins a Query with nothing to drop. The reduction itself is pinned on
names alone: the order of the sweep, what each fate does to the retained positions, and how the
sweep ends.
-/

namespace Umpire.ReplayTests

open Umpire Umpire.Command Umpire.Replay
open Umpire.ExplorationTests.Classed

model_conventions root "umpire" under Umpire.ReplayTests

property softBrightens
  machine: lampMachine
  when: toggle (soft)
  holds: fun step => step.state.level == .bright

scenario hardThenSoft
  model: lampMachine
  starts: dim
  actions: [toggle (hard), toggle (soft)]

limits lampThree
  steps: 3
  actions: 3
  search: 64

query softAfterHard
  find: softBrightens
  in: hardThenSoft
  limits: lampThree

private def subject? := softAfterHard.source.admit.toOption

#guard subject?.isSome

private def actions? : Option (List DefinitionId) :=
  subject?.bind fun admitted =>
    (editable (softAfterHard.source.behavior admitted.checked.vocabulary)).toOption

/-! The Scenario's two steps; the sweep tries the one before the target row. -/
#guard actions?.map (·.length) == some 2
#guard (actions?.map sweepOf).map (·.map (·.index)) == some [0]

/-! Keeping every position is the subject's own Query: the same Plan, so the same digest. -/
private def whole? := (admitKept softAfterHard.source [0, 1]).toOption
private def subjectDigest? : Option String :=
  subject?.bind fun admitted => admitted.checked.run.artifact.map fun plan =>
    Umpire.Exploration.candidateDigest plan.artifactChecksum
#guard whole?.map (·.digest) == subjectDigest?

/-! Dropping the held hard toggle is admitted: one step, a Plan of its own. -/
private def dropped? := (admitKept softAfterHard.source [1]).toOption
#guard dropped?.map (·.admitted.checked.behavior.actionsExactly.map (·.length)) == some (some 1)
#guard (dropped?.map (·.digest)).isSome && dropped?.map (·.digest) != subjectDigest?

/-! The switch's one-step Query has no prefix, so its sweep is empty and it is irreducible. -/
open Umpire.Examples.Switch in
#guard ((exactAction.source.admit.toOption).bind fun admitted =>
    (editable (exactAction.source.behavior admitted.checked.vocabulary)).toOption).map
    (fun actions => (sweepOf actions).length) == some 0

/-! ### The reduction on names alone -/

private def act (name : String) : DefinitionId := DefinitionId.of s!"replay.action.{name}"
private def four : List DefinitionId := [act "a", act "b", act "c", act "d"]
private def fresh : Reduction := Reduction.start four

/-! The sweep is the three prefix steps, last first, over all four positions. -/
#guard fresh.sweep.map (·.index) == [2, 1, 0]
#guard fresh.kept == [0, 1, 2, 3]
#guard fresh.next?.map (·.index) == some 2
#guard (fresh.next?.map fresh.candidateKept) == some [0, 1, 3]

private def edit (index : Nat) : Edit := { index, action := four.getD index (act "none") }

/-! A retained edit drops its step for good, and the next edit applies to what was retained. -/
private def afterRetained := fresh.settle (edit 2) .retained (some "c")
#guard afterRetained.kept == [0, 1, 3]
#guard afterRetained.next?.map afterRetained.candidateKept == some [0, 3]

/-! A non-reproducing or inapplicable edit keeps the positions and is never tried again. -/
private def swept := (afterRetained.settle (edit 1) .notReproduced (some "b")).settle (edit 0)
  (.inapplicable "not-selected")
#guard swept.kept == [0, 1, 3]
#guard swept.next? == none
#guard swept.result == .minimized
#guard swept.settled.map (·.fate.name) == ["retained", "not-reproduced", "inapplicable"]

/-! Nothing retained is irreducible. -/
#guard ((fresh.settle (edit 2) .notReproduced).settle (edit 1) (.rejected "production")
  |>.settle (edit 0) (.inapplicable "not-selected")).result == .irreducible

/-! An undecided edit ends the sweep incomplete, naming it; nothing more is handed out. -/
private def undecided := fresh.settle (edit 2) .undecided (some "c")
#guard undecided.result == .incomplete "dropPrefixStep 2 is undecided"
#guard undecided.next? == none

/-! Stopping early is incomplete; the first end stands. -/
#guard (fresh.stop "stopped").result == .incomplete "stopped"
#guard (undecided.stop "stopped").result == .incomplete "dropPrefixStep 2 is undecided"
#guard fresh.result == .incomplete "the sweep did not finish"

/-! More prefix steps than the cap: the sweep takes the cap and cannot end minimized. -/
private def long := Reduction.start ((List.range 11).map fun index => act (toString index)) (cap := 8)
#guard long.sweep.length == 8 && long.capped
#guard long.sweep.head?.map (·.index) == some 9
#guard (long.sweep.foldl (fun reduction edit => reduction.settle edit .notReproduced) long).result ==
  .incomplete "the sweep was capped at 8 edits"

end Umpire.ReplayTests
