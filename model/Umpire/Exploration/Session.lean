import Umpire.Exploration.Campaign

/-!
# One candidate at a time

A session is a campaign with at most one candidate outstanding. `next` refuses while one is out;
`observe` admits only the exact binding of the one that is, with what its Run said, and rejects a
crossed, stale or malformed binding without touching the campaign.
-/

namespace Umpire.Exploration

open Umpire.Command

variable {Setup State Action Outcome Fact : Type}
variable [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
variable [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
variable [DecidableEq Outcome] [DecidableEq Fact]

structure Session (model : DeclaredModel Setup State Action Outcome Fact) where
  campaign : Campaign model
  outstanding : Option (Candidate model) := none

namespace Session

variable {model : DeclaredModel Setup State Action Outcome Fact}

/-- Open a session over a checked campaign. -/
def begin (campaign : Campaign model) : Session model := { campaign }

/-- What `next` returns: the candidate now outstanding, exhaustion, a tooling failure, or nothing
because a candidate is already outstanding. -/
inductive Step (model : DeclaredModel Setup State Action Outcome Fact) where
  | candidate (candidate : Candidate model) (session : Session model)
  | exhausted (session : Session model)
  | toolingFailure (failure : ToolingFailure) (session : Session model)
  | outstanding

def next (session : Session model) : Step model :=
  match session.outstanding with
  | some _ => .outstanding
  | none =>
      match session.campaign.next with
      | .candidate candidate campaign => .candidate candidate { campaign, outstanding := some candidate }
      | .exhausted campaign => .exhausted { campaign, outstanding := none }
      | .toolingFailure failure campaign => .toolingFailure failure { campaign, outstanding := none }

/-- Admit exactly the outstanding candidate's binding with its observation. -/
def observe (session : Session model) (bindings : List ArtifactBinding) (observation : Observation) :
    Option (Session model) :=
  match session.outstanding, bindings with
  | some candidate, [binding] =>
      if binding == candidate.binding then
        some { campaign := session.campaign.observe candidate observation, outstanding := none }
      else none
  | _, _ => none

end Session

end Umpire.Exploration
