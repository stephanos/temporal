import Umpire.Exploration.Target

/-!
# What a campaign has learned about each target

A target starts pending. Planning it moves it to `planned`, and it is never planned again: whatever
its candidate's Run says, the answer is recorded and the campaign moves on. `covered` is the only
status a Run earns; `unreachable` is decided without one; `violated` and `attempted` say a Run was
spent and did not confirm the planned path -- the first because the deployment contradicted it, the
second because nothing decisive came back; `unrealizable` says no Run could be spent, because the
realization the campaign produces Cases under binds nothing for a member on the planned path.

A class ledger sits beside the targets: per claimed class, the decisive verdict of the candidate
planned for its member target. A `violated` there is a counterexample, the one thing an exploration
exists to find.
-/

namespace Umpire.Exploration

open Umpire.Command

/-- The status of one target. -/
inductive TargetStatus where
  | pending
  | planned
  | covered
  | unreachable
  | violated
  | attempted
  | unrealizable
  deriving BEq, DecidableEq, Repr

def TargetStatus.name : TargetStatus → String
  | .pending => "pending"
  | .planned => "planned"
  | .covered => "covered"
  | .unreachable => "unreachable"
  | .violated => "violated"
  | .attempted => "attempted"
  | .unrealizable => "unrealizable"

/-- What a candidate's Run said, once its cleanup is closed. Anything that is not a decisive
Verdict on a completed Run credits nothing. `unrealizable` is the one observation made without a
Run: the realization cannot perform a member on the candidate's path. -/
inductive Observation where
  | satisfied
  | violated
  | prepareRejected
  | inconclusive
  | unrealizable
  deriving BEq, DecidableEq, Repr

def Observation.name : Observation → String
  | .satisfied => "satisfied"
  | .violated => "violated"
  | .prepareRejected => "prepare-rejected"
  | .inconclusive => "inconclusive"
  | .unrealizable => "unrealizable"

/-- One target and what the campaign knows about it. -/
structure LedgerEntry where
  target : CoverageTarget
  status : TargetStatus
  deriving BEq, Repr

/-- The verdict recorded for one claimed class. -/
structure ClassVerdict where
  className : String
  member : DefinitionId
  verdict : Observation
  deriving BEq, Repr

/-- A violated class-member target: the class, the target and the candidate that violated it. -/
structure Counterexample where
  className : String
  target : CoverageTarget
  candidate : ArtifactChecksum
  deriving BEq, Repr

/-- The targets in enumeration order with their statuses, and the class ledger. -/
structure Ledger where
  entries : List LedgerEntry
  classes : List ClassVerdict := []
  counterexamples : List Counterexample := []
  deriving BEq, Repr

namespace Ledger

def ofTargets (targets : List CoverageTarget) : Ledger :=
  { entries := targets.map fun target => { target, status := .pending } }

/-- The first pending target in enumeration order. -/
def nextPending (ledger : Ledger) : Option CoverageTarget :=
  (ledger.entries.find? (·.status == .pending)).map (·.target)

def status? (ledger : Ledger) (target : CoverageTarget) : Option TargetStatus :=
  (ledger.entries.find? fun entry => targetKey entry.target == targetKey target).map (·.status)

private def setStatus (ledger : Ledger) (target : CoverageTarget) (status : TargetStatus) : Ledger :=
  { ledger with entries := ledger.entries.map fun entry =>
      if targetKey entry.target == targetKey target then { entry with status } else entry }

def markPlanned (ledger : Ledger) (target : CoverageTarget) : Ledger :=
  ledger.setStatus target .planned

def markUnreachable (ledger : Ledger) (target : CoverageTarget) : Ledger :=
  ledger.setStatus target .unreachable

/-- Credit one candidate's observation to the targets on its planned path. A `satisfied` Run
covers every one of them, whatever a previous candidate said; a `violated` Run marks the ones not
already covered; `unrealizable` marks the ones still pending or planned as unrealizable; anything
else marks them attempted.

A decisive Run is also a class verdict for every class member on the path: the first decisive
verdict stands, except that a violation supersedes an earlier satisfaction, because a class one
Run contradicted is contradicted. Every violated Run that crosses a class member is a
counterexample for that class, once per candidate, whatever the class ledger already said. -/
def credit (ledger : Ledger) (candidate : ArtifactChecksum) (covers : List CoverageTarget)
    (observation : Observation) : Ledger := Id.run do
  let keys := covers.map targetKey
  let mut entries := ledger.entries.map fun entry =>
    if !keys.contains (targetKey entry.target) then entry
    else match observation, entry.status with
      | .satisfied, _ => { entry with status := .covered }
      | .violated, .covered => entry
      | .violated, _ => { entry with status := .violated }
      | .unrealizable, .pending => { entry with status := .unrealizable }
      | .unrealizable, .planned => { entry with status := .unrealizable }
      | _, .pending => { entry with status := .attempted }
      | _, .planned => { entry with status := .attempted }
      | _, _ => entry
  let mut classes := ledger.classes
  let mut counterexamples := ledger.counterexamples
  if observation == .satisfied || observation == .violated then
    for target in covers do
      if let .classMember member _ _ className _ := target then
        match classes.find? (·.className == className) with
        | none => classes := classes ++ [{ className, member, verdict := observation }]
        | some recorded =>
            if recorded.verdict == .satisfied && observation == .violated then
              classes := classes.map fun entry =>
                if entry.className == className then { entry with verdict := .violated } else entry
        if observation == .violated then
          let sample : Counterexample := { className, target, candidate }
          unless counterexamples.any fun seen =>
              seen.className == className && seen.candidate == candidate do
            counterexamples := counterexamples ++ [sample]
  return ({ entries, classes, counterexamples } : Ledger)

def count (ledger : Ledger) (status : TargetStatus) : Nat :=
  (ledger.entries.filter (·.status == status)).length

/-- Nothing is left to plan. -/
def exhausted (ledger : Ledger) : Bool :=
  ledger.entries.all fun entry => entry.status != .pending && entry.status != .planned

end Ledger

end Umpire.Exploration
