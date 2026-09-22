import Umpire.Exploration.Ledger
import Umpire.Artifact

/-!
# One campaign over one exploratory set

A campaign is one exploratory set, the Model it covers, the limits its budget names and a ledger
over its targets. It hands out candidates one at a time -- the first pending target's Query, checked
and searched -- and takes back what each candidate's Run said. Selection is the enumeration order
and nothing else: no scoring, no seed, and a target is planned at most once.

Everything here is pure and Temporal-free. Producing the candidate's Case, running it and reading
the Run belong to the bridge and the coordinator; what they return is an `Observation`.
-/

namespace Umpire.Exploration

open Umpire.Command

variable {Setup State Action Outcome Fact : Type}
variable [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
variable [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
variable [DecidableEq Outcome] [DecidableEq Fact]

/-- Why a set is not a campaign. -/
inductive CampaignError where
  | notExploratory (set : DefinitionId) (purpose : String)
  | wrongMachine (set : DefinitionId) (machine : Option DefinitionId) (model : DefinitionId)
  | unknownTarget (set : DefinitionId) (key : String)
  | noBudget (set : DefinitionId)
  deriving BEq, Repr

def CampaignError.render : CampaignError → String
  | .notExploratory set purpose => s!"set {set.value} is {purpose}, not exploratory"
  | .wrongMachine set machine model =>
      s!"set {set.value} covers {(machine.map (·.value)).getD "no machine"}, not {model.value}"
  | .unknownTarget set key => s!"set {set.value} names a target the Model does not declare: {key}"
  | .noBudget set => s!"set {set.value} names no budget"

/-- A defect of the campaign's own Query, not of the Model or the deployment. It ends the campaign,
because every later candidate would be built the same way. -/
structure ToolingFailure where
  target : String
  reason : String
  deriving BEq, Repr

/-- One candidate: the target it was planned for, its checked Model and admission, its Plan (whose
checksum is its identity) and every target on its planned witness path. -/
structure Candidate (model : DeclaredModel Setup State Action Outcome Fact) where
  selected : CoverageTarget
  queryKey : String
  admitted : AdmittedModel model
  plan : Plan
  covers : List CoverageTarget

namespace Candidate

variable {model : DeclaredModel Setup State Action Outcome Fact}

def identity (candidate : Candidate model) : ArtifactChecksum := candidate.plan.artifactChecksum

def binding (candidate : Candidate model) : ArtifactBinding := candidate.plan.artifactBinding

def checked (candidate : Candidate model) : Umpire.Command.CheckedModel model :=
  candidate.admitted.checked

end Candidate

/-- The checked inputs and the ledger. -/
structure Campaign (model : DeclaredModel Setup State Action Outcome Fact) where
  set : SetDeclaration
  limits : Limits
  ledger : Ledger
  /-- Candidates planned so far, in order, each with the observation it received. -/
  history : List (ArtifactChecksum × String × Option Observation) := []

namespace Campaign

variable {model : DeclaredModel Setup State Action Outcome Fact}

private def declaredIds (model : DeclaredModel Setup State Action Outcome Fact) : List DefinitionId :=
  model.stateIds ++ model.actionIds ++ model.outcomeIds

private def targetIds : CoverageTarget → List DefinitionId
  | .row _ state action results => state :: action :: results
  | .result outcome => [outcome]
  | .classMember member _ _ _ _ => [member]

/-- Check one exploratory set against the Model it names and the limits its budget names. -/
def check (model : DeclaredModel Setup State Action Outcome Fact) (set : SetDeclaration)
    (limits : Limits) : Except CampaignError (Campaign model) := do
  unless set.purpose == .exploratory do
    throw (.notExploratory set.id set.purpose.name)
  -- A set names its machine by the machine's own identity, which is not the Model's target.
  let machineId := model.origin.family.id "machine" model.key
  unless set.machine == some machineId do
    throw (.wrongMachine set.id set.machine machineId)
  if set.budget.isNone then throw (.noBudget set.id)
  let known := declaredIds model
  for target in set.targets do
    for id in targetIds target do
      unless known.contains id do throw (.unknownTarget set.id (targetKey target))
  pure { set, limits, ledger := Ledger.ofTargets set.targets }

/-- What `next` returns: a candidate, exhaustion, or the campaign's own defect. -/
inductive Next (model : DeclaredModel Setup State Action Outcome Fact) where
  | candidate (candidate : Candidate model) (campaign : Campaign model)
  | exhausted (campaign : Campaign model)
  | toolingFailure (failure : ToolingFailure) (campaign : Campaign model)

/-- How one admission error ends a candidate. `notSelected` is the one honest "nothing here": the
target is unreachable under the limits. Every other error is a Query the campaign itself formed
wrongly, which no later target would form better, so it is the campaign's defect. -/
def admissionFailure (target : String) : AdmissionError → Option ToolingFailure
  | .notSelected .. => none
  | .invalidTarget _ => some { target, reason := "invalid-target" }
  | .invalidVocabulary _ => some { target, reason := "invalid-vocabulary" }
  | .admission diagnostic => some { target, reason := "admission: " ++ (repr diagnostic).pretty }
  | .instances reason => some { target, reason := "instances: " ++ reason }

private def queryKeyFor (set : SetDeclaration) (target : CoverageTarget) : String :=
  set.name ++ "." ++ (targetKey target).replace ":" "."

/-- Plan the first pending target. A target no path reaches, or whose Query selects nothing, is
marked unreachable and the next pending target is tried; any other admission error, a candidate
without a Plan, or a witness that does not contain its own target is a tooling failure. Each
retry marks one pending target unreachable, so the fuel of one per target never runs out. -/
def nextWith : Nat → Campaign model → Next model
  | 0, campaign => .exhausted campaign
  | fuel + 1, campaign =>
    match campaign.ledger.nextPending with
    | none => .exhausted campaign
    | some target =>
      match chooseRow model campaign.limits.steps.value target with
      | none => nextWith fuel { campaign with ledger := campaign.ledger.markUnreachable target }
      | some (lead, final) =>
          let queryKey := queryKeyFor campaign.set target
          let (propertyAuthor, behaviorAuthor) := targetAuthors model queryKey lead final
          match checkAdmitted model queryKey campaign.limits propertyAuthor behaviorAuthor with
          | .error error =>
              match admissionFailure (targetKey target) error with
              | none => nextWith fuel { campaign with ledger := campaign.ledger.markUnreachable target }
              | some failure => .toolingFailure failure campaign
          | .ok admitted =>
              match admitted.checked.run.artifact, admitted.checked.witness with
              | some plan, some witness =>
                  let covers := coveredTargets campaign.set.targets witness
                  if covers.any fun covered => targetKey covered == targetKey target then
                    let candidate : Candidate model :=
                      { selected := target, queryKey, admitted, plan, covers }
                    .candidate candidate { campaign with
                      ledger := campaign.ledger.markPlanned target
                      history := campaign.history ++ [(plan.artifactChecksum, targetKey target, none)] }
                  else
                    .toolingFailure {
                      target := targetKey target
                      reason := "planned witness does not reach the selected target" } campaign
              | _, _ =>
                  .toolingFailure {
                    target := targetKey target
                    reason := "selected Query carries no Plan or witness" } campaign

def next (campaign : Campaign model) : Next model :=
  nextWith (campaign.ledger.entries.length + 1) campaign

/-- Take back what a candidate's Run said. -/
def observe (campaign : Campaign model) (candidate : Candidate model) (observation : Observation) :
    Campaign model :=
  { campaign with
    ledger := campaign.ledger.credit candidate.identity candidate.covers observation
    history := campaign.history.map fun (identity, key, recorded) =>
      if identity == candidate.identity then (identity, key, some observation)
      else (identity, key, recorded) }

/-- The campaign's counts, in one closed record. -/
structure Summary where
  targets : Nat
  selected : Nat
  covered : Nat
  unreachable : Nat
  violated : Nat
  attempted : Nat
  pending : Nat
  counterexamples : List Counterexample
  exhausted : Bool
  deriving BEq, Repr

def summary (campaign : Campaign model) : Summary := {
  targets := campaign.ledger.entries.length
  selected := campaign.history.length
  covered := campaign.ledger.count .covered
  unreachable := campaign.ledger.count .unreachable
  violated := campaign.ledger.count .violated
  attempted := campaign.ledger.count .attempted
  pending := campaign.ledger.count .pending + campaign.ledger.count .planned
  counterexamples := campaign.ledger.counterexamples
  exhausted := campaign.ledger.exhausted }

end Campaign

end Umpire.Exploration
