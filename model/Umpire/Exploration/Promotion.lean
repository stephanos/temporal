import Umpire.Exploration.Campaign
import Umpire.Command.Promotion

/-!
# From a counterexample to its promotion source

A counterexample is a class-member target whose candidate's Run was violated. What the campaign
retained for it is the whole candidate: the admission its Query passed, its checked Model, its
Plan and its witness. That is exactly what `Umpire.Promotion` compiles a review-only regression
source from: the base anchor is read off the candidate, the promoted identities are fresh names
under the Model's family keyed by the candidate's digest, and the rendered bytes are their own
expectation, so `compilePromotionSource` proves that replanning the unchanged Query reproduces the
anchor and that the bytes and their SHA-256 are what they were. Nothing here touches a file: the
bytes go out with the summary, and whoever runs the campaign writes them where it names.
-/

namespace Umpire.Exploration

open Umpire.Command

variable {Setup State Action Outcome Fact : Type}
variable [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
variable [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
variable [DecidableEq Outcome] [DecidableEq Fact]
variable {model : DeclaredModel Setup State Action Outcome Fact}

/-- The candidate's digest, its identity without the algorithm prefix: what its fresh promoted
names and its source file are keyed by. -/
def candidateDigest (identity : ArtifactChecksum) : String :=
  Umpire.Command.Promotion.digestOf identity

/-- The anchor of the candidate's own planning, read off its admission and Plan. -/
def promotionAnchor (candidate : Candidate model) : Option PromotionBaseAnchor :=
  Umpire.Command.Promotion.anchor candidate.admitted candidate.plan

/-- The fresh names one candidate's proposal is written under. -/
def promotionSpec (candidate : Candidate model) (location : SourceLocation) : PromotionSourceSpec :=
  Umpire.Command.Promotion.spec model (candidateDigest candidate.identity) location

/-- One counterexample's proposal: the shape every proposal has. -/
abbrev Proposal := Umpire.Command.Promotion.Proposal

/-- Render and compile one candidate's proposal through the shared compiler. -/
def propose (candidate : Candidate model) (location : SourceLocation) :
    Except PromotionError Proposal :=
  Umpire.Command.Promotion.propose candidate.admitted candidate.plan location

/-- The file a proposal is named at: `<set>-<digest>.lean`, relative to wherever the caller writes. -/
def proposalPath (setName : String) (identity : ArtifactChecksum) : String :=
  Umpire.Command.Promotion.path setName (candidateDigest identity)

/-- The location a campaign's proposals are named at. -/
def proposalLocation (setName : String) (identity : ArtifactChecksum) : SourceLocation :=
  { path := proposalPath setName identity, line := 1, column := 1, provenance := "umpire-explore" }

namespace Campaign

/-- Every counterexample's proposal, in the order the counterexamples were observed, each compiled
or the reason it did not compile. -/
def proposals (campaign : Campaign model) : List (ArtifactChecksum × Except PromotionError Proposal) :=
  campaign.counterexampleCandidates.map fun candidate =>
    (candidate.identity, propose candidate (proposalLocation campaign.set.name candidate.identity))

end Campaign

end Umpire.Exploration
