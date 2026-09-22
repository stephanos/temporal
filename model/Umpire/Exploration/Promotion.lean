import Umpire.Exploration.Campaign
import Umpire.Promotion

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
  let rendered := identity.render
  if rendered.startsWith "sha256:" then String.ofList (rendered.toList.drop 7) else rendered

/-- The anchor of the candidate's own planning: its Query's identities, the Plan result and the
Plan the campaign kept, the witness and the reason it was selected. `none` when the candidate's
result was not a found trace, which a planned candidate's never is. -/
def promotionAnchor (candidate : Candidate model) : Option PromotionBaseAnchor :=
  let query := candidate.checked.query
  match candidate.checked.run.result.outcome with
  | .found trace reason => some {
      queryDefinitionId := query.id
      queryBehaviorFingerprint := query.behaviorFingerprint
      queryCanonicalMetadata := query.canonicalMetadata
      behaviorDefinitionId := query.behavior.id
      behaviorFingerprint := query.behavior.behaviorFingerprint
      targetDefinitionId := query.target.id
      targetBehaviorFingerprint := query.target.behaviorFingerprint
      kernelDefinitionId := query.target.machine.metadata.id
      kernelBehaviorFingerprint := query.target.behaviorFingerprint
      planResult := candidate.checked.run
      plan := candidate.plan
      expectedTrace := trace
      selectionReason := reason }
  | _ => none

/-- The fresh names one candidate's proposal is written under: the source, the promoted Behavior
and the promoted Query, each under the Model's family and keyed by the digest, and the location
the source is named at, which the caller of the campaign decides. -/
def promotionSpec (candidate : Candidate model) (location : SourceLocation) : PromotionSourceSpec :=
  let digest := candidateDigest candidate.identity
  let family := model.origin.family
  { sourceDefinitionId := family.id "promotion-source" digest
    sourceLocation := location
    promotedBehaviorDefinitionId := family.id "behavior" ("regression-" ++ digest)
    promotedQueryDefinitionId := family.id "query" ("regression-" ++ digest) }

/-- One counterexample's proposal: the candidate it came from, the names it is written under, and
the compiled source with its digest. -/
structure Proposal where
  identity : ArtifactChecksum
  spec : PromotionSourceSpec
  bytes : String
  sha256 : String
  deriving Repr

/-- Render and compile one candidate's proposal. The expectation is the rendering itself: the
compiler replans the unchanged Query, checks the result against the anchor read off the candidate,
re-renders, and seals the bytes and their SHA-256 only when every one of those agrees. -/
def propose (candidate : Candidate model) (location : SourceLocation) :
    Except PromotionError Proposal := do
  let some anchor := promotionAnchor candidate
    | throw {
        kind := .nonFoundResult
        subject := candidate.checked.query.id
        detail := "the candidate's planning result is not a found trace" }
  let spec := promotionSpec candidate location
  let bytes := renderPromotionSource spec anchor.expectedTrace
  let compiled ← compilePromotionSource candidate.admitted.admitted anchor spec
    { bytes, sha256 := promotionSourceSha256 bytes }
  pure { identity := candidate.identity, spec, bytes := compiled.sourceBytes, sha256 := compiled.sourceSha256 }

/-- The file a proposal is named at: `<set>-<digest>.lean`, relative to wherever the caller writes. -/
def proposalPath (setName : String) (identity : ArtifactChecksum) : String :=
  setName ++ "-" ++ candidateDigest identity ++ ".lean"

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
