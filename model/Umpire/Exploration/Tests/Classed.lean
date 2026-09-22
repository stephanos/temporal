import Umpire.Command
import Umpire.Exploration

/-!
# A class member that the deployment contradicts

The switch has no classed action, so this Model is the one the class ledger is pinned on: a lamp
whose one action carries a `mode` class with two members and an example for each. Its exploratory
set covers class members, its campaign plans one candidate per member, and a `violated` Run on a
member is the counterexample an exploration exists to find.
-/

namespace Umpire.ExplorationTests.Classed

open Umpire Umpire.Command Umpire.Exploration

model_conventions root "umpire" under Umpire.ExplorationTests.Classed

entity lamp

enum Level
  | dim
  | bright

/-- How hard the lamp is toggled. Each member is a class the action claims behaves alike. -/
enum Mode
  | soft
  | hard

structure LampState where
  level : Level
  deriving BEq, DecidableEq, Repr, Finite

enum ToggleOutcome
  | changed
  | held

enum Glow
  | low
  | high

action toggle
  party: operator
  on: lamp
  input:
    mode: Mode
  examples:
    soft → gentle
    hard → firm

/-- A soft toggle brightens a dim lamp and leaves a bright one; a hard toggle dims a bright lamp
and leaves a dim one. -/
def toggleStep (state : LampState) (mode : Mode) : List (Step LampState ToggleOutcome Glow) :=
  match mode, state.level with
  | .soft, .dim => [{ outcome := .changed, state := { level := .bright }, facts := [.high] }]
  | .soft, .bright => [{ outcome := .held, state, facts := [.high] }]
  | .hard, .bright => [{ outcome := .changed, state := { level := .dim }, facts := [.low] }]
  | .hard, .dim => [{ outcome := .held, state, facts := [.low] }]

machine lampMachine
  for: lamp
  state: LampState
  starts: [dim]
  ends: [bright]
  steps:
    toggle: toggleStep

limits two
  steps: 2
  actions: 2
  search: 64

set classed
  purpose: exploratory
  bind:
    operator: driven
  machine: lampMachine
  cover: classMembers
  budget: two

private instance : Inhabited CoverageTarget := ⟨.result unknownId⟩

/-! ### The targets are the two classes -/

#guard classed.targets.map CoverageTarget.kind == ["classMember", "classMember"]
/-! The classes are enumerated in the machine's action order, which is by member name. -/
#guard classed.targets.map (fun target => match target with
  | .classMember _ _ _ className sample => (className, sample)
  | _ => ("", "")) == [("hard", "firm"), ("soft", "gentle")]

private def campaign? : Option (Campaign lampMachine) :=
  (Campaign.check lampMachine classed two).toOption

#guard campaign?.isSome

private def hardTarget : CoverageTarget := classed.targets[0]!
private def softTarget : CoverageTarget := classed.targets[1]!

private def candidateOf : Campaign.Next lampMachine → Option (Candidate lampMachine)
  | .candidate candidate _ => some candidate
  | _ => none

private def campaignOf : Campaign.Next lampMachine → Option (Campaign lampMachine)
  | .candidate _ campaign => some campaign
  | .exhausted campaign => some campaign
  | .toolingFailure _ campaign => some campaign

private def first? := campaign?.map Campaign.next
private def hardCandidate? := first?.bind candidateOf
private def afterHard? := first?.bind campaignOf

/-! The hard member's candidate is the first reachable row whose action is the member: a hard
toggle from dim, one step, its Property naming that row's outcome, so the Case has a clause. -/
#guard hardCandidate?.any fun candidate =>
  targetKey candidate.selected == targetKey hardTarget &&
    candidate.covers.map targetKey == [targetKey hardTarget] &&
    candidate.checked.witness.any (fun witness => witness.trace.steps.length == 1) &&
    candidate.checked.property.clauses.length == 1

/-! ### A violated member is a counterexample -/

private def violated? := do
  let campaign ← afterHard?
  let candidate ← hardCandidate?
  pure (campaign.observe candidate .violated)

#guard violated?.any fun campaign =>
  campaign.ledger.status? hardTarget == some TargetStatus.violated &&
    campaign.ledger.classes.map (fun entry => (entry.className, entry.verdict)) ==
      [("hard", Observation.violated)]
#guard (do
  let campaign ← violated?
  let candidate ← hardCandidate?
  pure (campaign.ledger.counterexamples.map fun (sample : Counterexample) =>
    (sample.className, targetKey sample.target, sample.candidate == candidate.identity))).any
    (· == [("hard", targetKey hardTarget, true)])

/-! The admission the counterexample's candidate was checked through is retained beside its
checked Model, which is what Promotion re-answers through. -/
#guard hardCandidate?.any fun candidate =>
  candidate.admitted.admitted.query.id == candidate.checked.query.id

/-- A satisfied member is a class verdict and no counterexample; the second member's candidate is
the soft toggle from dim, which brightens. -/
private def finished? := do
  let campaign ← violated?
  let next := campaign.next
  let candidate ← candidateOf next
  let campaign ← campaignOf next
  pure (campaign.observe candidate .satisfied)

#guard finished?.any fun campaign =>
  campaign.ledger.classes.map (fun entry => (entry.className, entry.verdict)) ==
      [("hard", Observation.violated), ("soft", Observation.satisfied)] &&
    campaign.summary.counterexamples.length == 1 &&
    campaign.summary.covered == 1 &&
    campaign.summary.violated == 1 &&
    campaign.summary.exhausted

/-! ### A violation is a counterexample whatever the class ledger already said

The hard member satisfied first, then the soft member violated: the soft counterexample is
recorded, and the hard class keeps its verdict. And a class satisfied by one candidate and
violated by a later one is a violated class with its counterexample, because a class one Run
contradicted is contradicted. -/

private def satisfiedThenViolated? := do
  let campaign ← afterHard?
  let candidate ← hardCandidate?
  let campaign := campaign.observe candidate .satisfied
  let next := campaign.next
  let soft ← candidateOf next
  let campaign ← campaignOf next
  pure (campaign.observe soft .violated)

#guard satisfiedThenViolated?.any fun campaign =>
  campaign.ledger.classes.map (fun entry => (entry.className, entry.verdict)) ==
      [("hard", Observation.satisfied), ("soft", Observation.violated)] &&
    campaign.ledger.counterexamples.map (·.className) == ["soft"]

/-! Rows and class members together: the row candidate for the hard toggle from dim also crosses
the hard class member, so its verdicts are the class's. Satisfied first sets the class; a later
violated candidate over the same member supersedes it and is a counterexample; the same candidate
observed twice is one counterexample. -/
set classedRows
  purpose: exploratory
  bind:
    operator: driven
  machine: lampMachine
  cover: rows | classMembers
  budget: two

private def rowsCampaign? : Option (Campaign lampMachine) :=
  (Campaign.check lampMachine classedRows two).toOption

#guard rowsCampaign?.isSome
#guard classedRows.targets.map CoverageTarget.kind ==
  ["row", "row", "row", "row", "classMember", "classMember"]

private def rowsFirst? := rowsCampaign?.map Campaign.next
private def rowsCandidate? := rowsFirst?.bind candidateOf
private def rowsAfter? := rowsFirst?.bind campaignOf

#guard rowsCandidate?.any fun candidate =>
  candidate.covers.map CoverageTarget.kind == ["row", "classMember"]

private def supersededClass? := do
  let campaign ← rowsAfter?
  let candidate ← rowsCandidate?
  let satisfied := campaign.observe candidate .satisfied
  let violated := satisfied.observe candidate .violated
  pure (satisfied.ledger.classes.map fun (entry : ClassVerdict) => entry.verdict,
    violated.ledger.classes.map fun (entry : ClassVerdict) => entry.verdict,
    (violated.observe candidate .violated).ledger.counterexamples.length)

#guard supersededClass? == some ([Observation.satisfied], [Observation.violated], 1)

/-! A non-decisive Run is no class verdict. -/
#guard (do
  let campaign ← afterHard?
  let candidate ← hardCandidate?
  pure (campaign.observe candidate .inconclusive).ledger.classes.isEmpty).getD false

end Umpire.ExplorationTests.Classed
