import Umpire.Command
import Umpire.Exploration

/-!
# A row plannable only under its second result

A transition contract binds every occurrence of its action, so a row whose prefix must take the
row's action to a different outcome cannot be planned under that outcome. Here the row from `far`
has two results, and reaching `far` takes the same action to the second one; the campaign plans
the row under its second result rather than calling it unreachable.
-/

namespace Umpire.ExplorationTests.Results

open Umpire Umpire.Command Umpire.Exploration

model_conventions root "umpire" under Umpire.ExplorationTests.Results

entity walker

enum Place
  | near
  | far

structure WalkState where
  place : Place
  deriving BEq, DecidableEq, Repr, Finite

/-- `held` sorts before `moved`, so it is the row's first result in canonical order. -/
enum StepOutcome
  | held
  | moved

enum Mark
  | marked

action go
  party: operator
  on: walker

/-- From near, a go holds first and moves to far second; from far, it holds first and moves back
second. The only way to far is a go that moved, so a row from far under `held` needs a prefix that
took `go` to `moved`, which a contract on `go` with outcome `held` forbids. -/
def goStep (state : WalkState) : List (Step WalkState StepOutcome Mark) :=
  match state.place with
  | .near => [{ outcome := .held, state, facts := [.marked] },
              { outcome := .moved, state := { place := .far }, facts := [.marked] }]
  | .far => [{ outcome := .held, state, facts := [.marked] },
             { outcome := .moved, state := { place := .near }, facts := [.marked] }]

machine walk
  for: walker
  state: WalkState
  starts: [near]
  ends: [far]
  steps:
    go: goStep

limits two
  steps: 2
  actions: 2
  search: 64

set walked
  purpose: exploratory
  bind:
    operator: driven
  machine: walk
  cover: rows
  budget: two

#guard walked.targets.map CoverageTarget.kind == ["row", "row"]

private def campaign? : Option (Campaign walk) := (Campaign.check walk walked two).toOption

#guard campaign?.isSome

private def candidateOf : Campaign.Next walk → Option (Candidate walk)
  | .candidate candidate _ => some candidate
  | _ => none

private def campaignOf : Campaign.Next walk → Option (Campaign walk)
  | .candidate _ campaign => some campaign
  | .exhausted campaign => some campaign
  | .toolingFailure _ campaign => some campaign

private def nearFirst? := campaign?.map Campaign.next

/-! The near row plans under its first result, `held`, in one step. -/
#guard (nearFirst?.bind candidateOf).any fun candidate =>
  candidate.checked.witness.any fun witness =>
    witness.trace.steps.map (·.outcome.value) == ["held"]

private def farCandidate? := do
  let campaign ← nearFirst?.bind campaignOf
  let candidate ← nearFirst?.bind candidateOf
  candidateOf (campaign.observe candidate .satisfied).next

/-! The far row cannot plan under `held`, whose prefix takes `go` to `moved`; it plans under its
second result, `moved`, and is not called unreachable. -/
#guard farCandidate?.any fun candidate =>
  candidate.checked.witness.any fun witness =>
    witness.trace.steps.map (·.outcome.value) == ["moved", "moved"]

/-! Nothing is unreachable, and a zero-step budget plans nothing. -/
#guard (do
  let campaign ← nearFirst?.bind campaignOf
  let candidate ← nearFirst?.bind candidateOf
  let next := (campaign.observe candidate .satisfied).next
  let far ← candidateOf next
  let campaign ← campaignOf next
  pure (campaign.observe far .satisfied).summary).any fun (summary : Campaign.Summary) =>
    summary.covered == 2 && summary.unreachable == 0 && summary.exhausted

#guard (walk.table.transitions.head?.bind fun row =>
  (goStep { place := .near }).head?.map fun result =>
    (pathTo walk 0 { row, result }).isNone).getD false

end Umpire.ExplorationTests.Results
