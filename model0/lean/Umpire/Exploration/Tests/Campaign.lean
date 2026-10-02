import Umpire.Exploration
import Umpire.Examples.Switch
import Umpire.Promotion.Tests.Fixtures.CompiledSource

/-!
# The campaign over the switch's exploratory set

The switch's set covers the rows and results one flip from `off` reaches: one row and its two
results. That is enough to pin what a campaign does at every step -- which target it plans, what
its candidate's planned path covers, what each observation credits, and when it is exhausted --
and to pin that it does the same thing twice.

Every value below is an `Option`, read through `any`, so a step that did not happen fails the pin
that reads it rather than a proof somewhere else.
-/

namespace Umpire.ExplorationTests

open Umpire Umpire.Command Umpire.Exploration Umpire.Examples.Switch

private instance : Inhabited CoverageTarget := ⟨.result unknownId⟩

/-! ### The set and the campaign -/

#guard switchExploration.purpose == .exploratory
#guard switchExploration.targets.map CoverageTarget.kind == ["row", "result", "result"]

private def campaign? : Option (Campaign twoState) :=
  (Campaign.check twoState switchExploration one).toOption

#guard campaign?.isSome

/-- The row target and the two result targets, as the set enumerated them. -/
private def rowTarget : CoverageTarget := switchExploration.targets[0]!
private def appliedTarget : CoverageTarget := switchExploration.targets[1]!
private def deferredTarget : CoverageTarget := switchExploration.targets[2]!

private def candidateOf : Campaign.Next twoState → Option (Candidate twoState)
  | .candidate candidate _ => some candidate
  | _ => none

private def campaignOf : Campaign.Next twoState → Option (Campaign twoState)
  | .candidate _ campaign => some campaign
  | .exhausted campaign => some campaign
  | .toolingFailure _ campaign => some campaign

private def isExhausted : Campaign.Next twoState → Bool
  | .exhausted _ => true
  | _ => false

private def selects (target : CoverageTarget) (candidate : Candidate twoState) : Bool :=
  targetKey candidate.selected == targetKey target

/-! ### One walk, step by step -/

private def first? := campaign?.map Campaign.next
private def firstCandidate? := first?.bind candidateOf
private def afterFirst? := first?.bind campaignOf

/-! The first candidate is the row's: one flip from off, taking the row's first result, which is
`applied`; its planned path covers the row and the `applied` result, not `deferred`. -/
#guard firstCandidate?.any (selects rowTarget)
#guard firstCandidate?.any fun candidate =>
  candidate.covers.map targetKey == [targetKey rowTarget, targetKey appliedTarget]
#guard firstCandidate?.any fun candidate =>
  candidate.checked.witness.any fun witness => witness.trace.steps.length == 1
#guard afterFirst?.any fun campaign =>
  campaign.ledger.status? rowTarget == some TargetStatus.planned &&
    campaign.ledger.status? appliedTarget == some TargetStatus.pending &&
    campaign.history.length == 1

/-! A second `next` while the first is unobserved plans the next pending target, which is the
`applied` result: a campaign does not stop a caller from planning ahead; the session does. -/
#guard (afterFirst?.bind fun campaign => candidateOf campaign.next).any (selects appliedTarget)

private def afterSatisfied? := do
  let campaign ← afterFirst?
  let candidate ← firstCandidate?
  pure (campaign.observe candidate .satisfied)

#guard afterSatisfied?.any fun campaign =>
  campaign.ledger.status? rowTarget == some TargetStatus.covered &&
    campaign.ledger.status? appliedTarget == some TargetStatus.covered &&
    campaign.ledger.status? deferredTarget == some TargetStatus.pending &&
    campaign.history.map (·.2.2) == [some .satisfied]

private def second? := afterSatisfied?.map Campaign.next
private def secondCandidate? := second?.bind candidateOf
private def afterSecond? := second?.bind campaignOf

/-! The `deferred` result's row is the same row with its other result; the candidate is a
different Plan, and its path covers the row again and `deferred`. -/
#guard secondCandidate?.any (selects deferredTarget)
#guard secondCandidate?.any fun candidate =>
  candidate.covers.map targetKey == [targetKey rowTarget, targetKey deferredTarget]
#guard (do
  let first ← firstCandidate?
  let second ← secondCandidate?
  pure (first.identity != second.identity)).getD false

private def finished? := do
  let campaign ← afterSecond?
  let candidate ← secondCandidate?
  pure (campaign.observe candidate .satisfied)

#guard finished?.any fun campaign => isExhausted campaign.next
#guard finished?.any fun campaign => campaign.summary == {
  targets := 3
  selected := 2
  covered := 3
  unreachable := 0
  violated := 0
  attempted := 0
  unrealizable := 0
  pending := 0
  counterexamples := []
  exhausted := true }

/-! ### The other observations -/

private def afterViolated? := do
  let campaign ← afterFirst?
  let candidate ← firstCandidate?
  pure (campaign.observe candidate .violated)

/-! A violated Run marks the candidate's targets and credits nothing. -/
#guard afterViolated?.any fun campaign =>
  campaign.ledger.status? rowTarget == some TargetStatus.violated &&
    campaign.ledger.status? appliedTarget == some TargetStatus.violated &&
    campaign.ledger.status? deferredTarget == some TargetStatus.pending

private def violatedThenInconclusive? := do
  let campaign ← afterViolated?
  let next := campaign.next
  let candidate ← candidateOf next
  let campaign ← campaignOf next
  pure (campaign.observe candidate .inconclusive)

/-! A non-decisive result marks what was still pending or planned as attempted and leaves a
violated target violated; the campaign is then exhausted with nothing covered. -/
#guard violatedThenInconclusive?.any fun campaign =>
  campaign.ledger.status? deferredTarget == some TargetStatus.attempted &&
    campaign.ledger.status? rowTarget == some TargetStatus.violated &&
    isExhausted campaign.next &&
    campaign.summary.covered == 0 &&
    campaign.summary.violated == 2 &&
    campaign.summary.attempted == 1

/-! A preparation rejection is the same as any other non-decisive result. -/
#guard (do
  let campaign ← afterFirst?
  let candidate ← firstCandidate?
  pure ((campaign.observe candidate .prepareRejected).ledger.status? rowTarget)).any
    (· == some TargetStatus.attempted)

/-! A covered target stays covered when a later candidate over the same row is violated. -/
#guard (do
  let campaign ← afterSecond?
  let candidate ← secondCandidate?
  let ledger := (campaign.observe candidate .violated).ledger
  pure (ledger.status? rowTarget == some TargetStatus.covered &&
    ledger.status? deferredTarget == some TargetStatus.violated)).getD false

/-! ### Unreachable targets

The flip from `on` is two steps from `off`; under `one` no path reaches it, so a set that names
its row is told so without a Run, and the campaign moves on to the next pending target. -/

private def onRowKey : String := ((twoState.table.transitions.map (·.key))[1]?).getD ""

private def onRowTarget : CoverageTarget :=
  .row onRowKey onState.definitionId flipActionId [appliedOutcomeId]

private def withUnreachable : SetDeclaration :=
  { switchExploration with targets := onRowTarget :: switchExploration.targets }

private def unreachableFirst? :=
  (Campaign.check twoState withUnreachable one).toOption.map Campaign.next

#guard (unreachableFirst?.bind candidateOf).any (selects rowTarget)
#guard (unreachableFirst?.bind campaignOf).any fun campaign =>
  campaign.ledger.status? onRowTarget == some TargetStatus.unreachable

/-! ### What is not a campaign -/

private def errorOf (result : Except CampaignError (Campaign twoState)) : Option CampaignError :=
  match result with
  | .ok _ => none
  | .error error => some error

private def machineId : DefinitionId := twoState.origin.family.id "machine" twoState.key

#guard errorOf (Campaign.check twoState { switchExploration with purpose := .functional } one) ==
  some (.notExploratory switchExploration.id "functional")
#guard errorOf (Campaign.check twoState { switchExploration with machine := none } one) ==
  some (.wrongMachine switchExploration.id none machineId)
#guard errorOf (Campaign.check twoState { switchExploration with budget := none } one) ==
  some (.noBudget switchExploration.id)
#guard errorOf (Campaign.check twoState
    { switchExploration with targets := [.result (DefinitionId.of "umpire.absent")] } one) ==
  some (.unknownTarget switchExploration.id "result:umpire.absent")

/-! Only `notSelected` is an unreachable target; every other admission error is the campaign's
own defect. -/
#guard (Campaign.admissionFailure "t" (.notSelected .noneFound {} one)).isNone
#guard (Campaign.admissionFailure "t" (.instances "two")).map (·.reason) == some "instances: two"

/-! ### Determinism

The same inputs and the same observations walk the same candidates to the same summary. -/

private def walk (campaign : Campaign twoState) : Nat → List (ArtifactChecksum × String) →
    List (ArtifactChecksum × String) × Campaign.Summary
  | 0, seen => (seen, campaign.summary)
  | fuel + 1, seen =>
      match campaign.next with
      | .candidate candidate campaign =>
          walk (campaign.observe candidate .satisfied) fuel
            (seen ++ [(candidate.identity, targetKey candidate.selected)])
      | .exhausted campaign => (seen, campaign.summary)
      | .toolingFailure _ campaign => (seen, campaign.summary)

private def walked? := campaign?.map (walk · 8 [])

#guard walked? == campaign?.map (walk · 8 [])
#guard walked?.any fun (seen, summary) =>
  seen.map (·.2) == [targetKey rowTarget, targetKey deferredTarget] && summary.exhausted

/-- What a replay records of one candidate: its identity, the target it was selected for, its
Query's identity and the observation it was read as. -/
private structure Replayed where
  identity : ArtifactChecksum
  target : String
  query : DefinitionId
  observation : Umpire.Exploration.Observation
  deriving BEq, Repr

/-- What a replay ends with: the candidates in order, every target's status, and the summary. -/
private structure Replay where
  seen : List Replayed
  statuses : List (String × Option TargetStatus)
  summary : Campaign.Summary
  deriving BEq, Repr

private def statusesOf (campaign : Campaign twoState) : List (String × Option TargetStatus) :=
  switchExploration.targets.map fun target => (targetKey target, campaign.ledger.status? target)

/-- Replay a scripted observation stream: each entry is the decisive observation the candidate of
that identity is read as, consumed in order. The walk ends when the stream does, before another
candidate is planned, the way a campaign its caller stops ends after the completed prefix; a
candidate the stream does not name at its position ends the walk too, so that a stream keyed by
the wrong identities records nothing past the mismatch. -/
private def replay (campaign : Campaign twoState) :
    List (ArtifactChecksum × Umpire.Exploration.Observation) →
    List Replayed → Replay
  | [], seen => { seen, statuses := statusesOf campaign, summary := campaign.summary }
  | (identity, observation) :: rest, seen =>
      match campaign.next with
      | .candidate candidate campaign =>
          if candidate.identity != identity then
            { seen, statuses := statusesOf campaign, summary := campaign.summary }
          else
            let replayed : Replayed := {
              identity := candidate.identity
              target := targetKey candidate.selected
              query := candidate.checked.query.id
              observation }
            replay (campaign.observe candidate observation) rest (seen ++ [replayed])
      | .exhausted campaign => { seen, statuses := statusesOf campaign, summary := campaign.summary }
      | .toolingFailure _ campaign =>
          { seen, statuses := statusesOf campaign, summary := campaign.summary }

/-- The stream the pins replay: the identities the first walk planned, the row candidate satisfied
and the deferred result's candidate violated. -/
private def stream? : Option (List (ArtifactChecksum × Umpire.Exploration.Observation)) :=
  walked?.map fun (seen, _) =>
    (seen.map (·.1)).zip [Umpire.Exploration.Observation.satisfied, .violated]

private def replayedOnce? : Option Replay := do
  let campaign ← (Campaign.check twoState switchExploration one).toOption
  pure (replay campaign (← stream?) [])

private def replayedAgain? : Option Replay := do
  let campaign ← (Campaign.check twoState switchExploration one).toOption
  pure (replay campaign (← stream?) [])

/-! Two campaigns checked from the same declarations and replayed over the same stream record the
same candidates in the same order, the same status for every target and the same summary. -/
#guard replayedOnce?.isSome
#guard replayedOnce? == replayedAgain?
#guard replayedOnce?.any fun replayed =>
  replayed.seen.map (·.observation) == [Umpire.Exploration.Observation.satisfied, .violated] &&
    replayed.statuses.map (·.2) == [some TargetStatus.covered, some .covered, some .violated] &&
    replayed.summary.selected == 2 && replayed.summary.covered == 2 &&
    replayed.summary.violated == 1 && replayed.summary.exhausted

/-- The stream cut after its first observation: the campaign stopped by its caller. -/
private def replayedPrefix? : Option Replay := do
  let campaign ← (Campaign.check twoState switchExploration one).toOption
  pure (replay campaign ((← stream?).take 1) [])

/-! A stream that ends early records a prefix of the full replay's candidates, with the same
identities, and credits the same targets for that prefix; what the full replay observed later is
still pending here, and the campaign is not exhausted. -/
#guard ((do
  let full ← replayedOnce?
  let cut ← replayedPrefix?
  pure (cut.seen == full.seen.take 1 &&
    cut.statuses.map (·.2) == [some TargetStatus.covered, some .covered, some .pending] &&
    cut.summary.selected == 1 && cut.summary.covered == 2 && !cut.summary.exhausted)) : Option Bool).getD false

/-! A stream whose identities are not the campaign's records nothing: a stale or crossed script
is not replayed. -/
#guard ((do
  let campaign ← (Campaign.check twoState switchExploration one).toOption
  let replayed := replay campaign [(planChecksumOf "elsewhere", Umpire.Exploration.Observation.satisfied)] []
  pure replayed.seen.isEmpty) : Option Bool).getD false

/-! ### Pinned regressions are outside the campaign

The switch's compiled regression source promotes its exact-action Query under a fresh name. That
Query is pinned by `Umpire.PromotionTests`, not selected by the campaign: no candidate is it, and
the campaign's count of selected candidates is the count of its own candidates, so a pinned
regression consumes none of the campaign's budget. -/

private def promotedQuery? : Option (CheckedQuery LawStatement) :=
  (Umpire.Promotion.Tests.Fixtures.CompiledSource.promotedQueryResult exactActionQuery).toOption

#guard promotedQuery?.isSome
#guard ((do
  let promoted ← promotedQuery?
  let replayed ← replayedOnce?
  pure (replayed.seen.all (fun candidate =>
      candidate.query != promoted.id && candidate.query != exactActionQuery.id) &&
    replayed.summary.selected == replayed.seen.length &&
    replayed.summary.targets == switchExploration.targets.length)) : Option Bool).getD false

/-! ### The session: one candidate at a time -/

private def sessionCandidateOf : Session.Step twoState → Option (Candidate twoState)
  | .candidate candidate _ => some candidate
  | _ => none

private def sessionOf : Session.Step twoState → Option (Session twoState)
  | .candidate _ session => some session
  | .exhausted session => some session
  | .toolingFailure _ session => some session
  | .outstanding => none

private def isOutstanding : Session.Step twoState → Bool
  | .outstanding => true
  | _ => false

private def sessionFirst? := campaign?.map fun campaign => (Session.begin campaign).next
private def outstandingCandidate? := sessionFirst?.bind sessionCandidateOf
private def outstandingSession? := sessionFirst?.bind sessionOf

/-! `next` refuses while a candidate is outstanding; `observe` admits only that candidate's exact
binding, once, and a stale or crossed binding leaves the session as it was. -/
#guard outstandingSession?.any fun session => isOutstanding session.next
#guard outstandingSession?.any fun session => (session.observe [] .satisfied).isNone
#guard (do
  let session ← outstandingSession?
  let candidate ← outstandingCandidate?
  pure ((session.observe [candidate.binding, candidate.binding] .satisfied).isNone &&
    (session.observe [{ candidate.binding with formatVersion := "unsupported-format" }]
      .satisfied).isNone)).getD false
#guard (do
  let session ← outstandingSession?
  let crossed ← secondCandidate?
  pure (session.observe [crossed.binding] .satisfied).isNone).getD false
#guard (do
  let session ← outstandingSession?
  let candidate ← outstandingCandidate?
  let observed ← session.observe [candidate.binding] .satisfied
  pure (observed.outstanding.isNone &&
    observed.campaign.ledger.status? rowTarget == some TargetStatus.covered &&
    (sessionCandidateOf observed.next).any (selects deferredTarget))).getD false

end Umpire.ExplorationTests
