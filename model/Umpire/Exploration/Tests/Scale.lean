import Umpire.Command
import Umpire.Exploration

/-!
# Ten times the switch

A counter with ten positions and two actions has twenty rows, ten times the switch's two. Its
campaign covers every row and result it enumerates within the same shape of limits, one bounded
Search per candidate and no table of paths between targets, and a candidate for a far row covers
the rows on its way, so the campaign spends fewer candidates than there are targets.
-/

namespace Umpire.ExplorationTests.Scale

open Umpire Umpire.Command Umpire.Exploration

model_conventions root "umpire" under Umpire.ExplorationTests.Scale

entity counter

enum Slot
  | s0
  | s1
  | s2
  | s3
  | s4
  | s5
  | s6
  | s7
  | s8
  | s9

structure CounterState where
  slot : Slot
  deriving BEq, DecidableEq, Repr, Finite

enum CountOutcome
  | moved
  | stayed

enum Tick
  | ticked

action advance
  party: operator
  on: counter

action reset
  party: operator
  on: counter

def Slot.next : Slot → Option Slot
  | .s0 => some .s1
  | .s1 => some .s2
  | .s2 => some .s3
  | .s3 => some .s4
  | .s4 => some .s5
  | .s5 => some .s6
  | .s6 => some .s7
  | .s7 => some .s8
  | .s8 => some .s9
  | .s9 => none

def advanceStep (state : CounterState) : List (Step CounterState CountOutcome Tick) :=
  match state.slot.next with
  | some slot => [{ outcome := .moved, state := { slot }, facts := [.ticked] }]
  | none => [{ outcome := .stayed, state, facts := [.ticked] }]

def resetStep (state : CounterState) : List (Step CounterState CountOutcome Tick) :=
  if state.slot == .s0 then [{ outcome := .stayed, state, facts := [.ticked] }]
  else [{ outcome := .moved, state := { slot := .s0 }, facts := [.ticked] }]

machine counterMachine
  for: counter
  state: CounterState
  starts: [s0]
  ends: [s9]
  steps:
    advance: advanceStep
    reset: resetStep

limits ten
  steps: 10
  actions: 10
  search: 8192

set scale
  purpose: exploratory
  bind:
    operator: driven
  machine: counterMachine
  cover: rows | results
  budget: ten

#guard counterMachine.table.transitions.length == 20
#guard scale.targets.length == 22

private def campaign? : Option (Campaign counterMachine) :=
  (Campaign.check counterMachine scale ten).toOption

#guard campaign?.isSome

private def walk (campaign : Campaign counterMachine) : Nat → Nat → Nat × Campaign.Summary
  | 0, selected => (selected, campaign.summary)
  | fuel + 1, selected =>
      match campaign.next with
      | .candidate candidate campaign =>
          walk (campaign.observe candidate .satisfied) fuel (selected + 1)
      | .exhausted campaign => (selected, campaign.summary)
      | .toolingFailure _ campaign => (selected, campaign.summary)

private def outcome? := campaign?.map (walk · 30 0)

private def finalLedger (campaign : Campaign counterMachine) : Nat → Ledger
  | 0 => campaign.ledger
  | fuel + 1 =>
      match campaign.next with
      | .candidate candidate campaign => finalLedger (campaign.observe candidate .satisfied) fuel
      | .exhausted campaign => campaign.ledger
      | .toolingFailure _ campaign => campaign.ledger

#guard (campaign?.map (finalLedger · 30)).any fun ledger =>
  (ledger.entries.filter (·.status == .unreachable)).map (targetKey ·.target) ==
    ["row:s9-advance"]

/-! Every target but one is covered, in fewer candidates than targets because each candidate's
path covers the rows it walks through. The one left is the advance that stays at the last slot:
reaching that slot takes nine advances that move, and a transition contract on `advance` binds
every occurrence, so no Query of this form can select it and the campaign says so without a Run. -/
#guard outcome?.any fun (selected, summary) =>
  summary.exhausted && summary.covered == 21 && summary.unreachable == 1 &&
    selected == summary.selected && selected < 22
#guard (campaign?.map fun campaign => (walk campaign 30 0).2).any fun summary =>
  summary.covered + summary.unreachable == summary.targets

end Umpire.ExplorationTests.Scale
