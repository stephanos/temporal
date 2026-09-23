import Umpire.Command.Authoring
import Umpire.Promotion

/-!
# From an admitted Query to its review-only proposal

Every proposal a campaign or a replay compiles comes from the same thing: one admitted Query, its
checked Model and the Plan its search delivered. The anchor is read off that planning, the
promoted identities are fresh names under the Model's family keyed by the Plan checksum's digest,
and the rendered bytes are their own expectation, so `Umpire.Promotion.compilePromotionSource`
proves that replanning the unchanged Query reproduces the anchor and that the bytes and their
SHA-256 are what they were. The proposal renders the Model's expected trace; no observed Run
reaches it. Nothing here touches a file: whoever compiled it writes the bytes where it names.

The exploration campaign and the replay bridge both call `propose`, so a counterexample's proposal
and a reduced subject's are the same shape under the same names.
-/

namespace Umpire.Command.Promotion

open Umpire Umpire.Command

variable {Setup State Action Outcome Fact : Type}
variable [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
variable [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
variable [DecidableEq Outcome] [DecidableEq Fact]
variable {model : DeclaredModel Setup State Action Outcome Fact}

/-- A Plan checksum's digest, its identity without the algorithm prefix: what fresh promoted names,
the source file and a candidate's Case are keyed by. -/
def digestOf (identity : ArtifactChecksum) : String :=
  let rendered := identity.render
  if rendered.startsWith "sha256:" then String.ofList (rendered.toList.drop 7) else rendered

/-- The anchor of one admitted Query's planning: its identities, the Plan result and the Plan, the
witness and the reason it was selected. `none` when the result was not a found trace. -/
def anchor (admitted : AdmittedModel model) (plan : Plan) : Option PromotionBaseAnchor :=
  let query := admitted.checked.query
  match admitted.checked.run.result.outcome with
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
      planResult := admitted.checked.run
      plan
      expectedTrace := trace
      selectionReason := reason }
  | _ => none

/-- The fresh names one proposal is written under: the source, the promoted Behavior and the
promoted Query, each under the Model's family and keyed by the digest, and the location the
caller names the source at. -/
def spec (model : DeclaredModel Setup State Action Outcome Fact) (digest : String)
    (location : SourceLocation) : PromotionSourceSpec :=
  let family := model.origin.family
  { sourceDefinitionId := family.id "promotion-source" digest
    sourceLocation := location
    promotedBehaviorDefinitionId := family.id "behavior" ("regression-" ++ digest)
    promotedQueryDefinitionId := family.id "query" ("regression-" ++ digest) }

/-- One compiled proposal: the Plan it came from, the names it is written under, and the source
with its digest. -/
structure Proposal where
  identity : ArtifactChecksum
  spec : PromotionSourceSpec
  bytes : String
  sha256 : String
  deriving Repr

/-- Render and compile one admitted Query's proposal. -/
def propose (admitted : AdmittedModel model) (plan : Plan) (location : SourceLocation) :
    Except PromotionError Proposal := do
  let some anchor := anchor admitted plan
    | throw {
        kind := .nonFoundResult
        subject := admitted.checked.query.id
        detail := "the Query's planning result is not a found trace" }
  let spec := spec model (digestOf plan.artifactChecksum) location
  let bytes := renderPromotionSource spec anchor.expectedTrace
  let compiled ← compilePromotionSource admitted.admitted anchor spec
    { bytes, sha256 := promotionSourceSha256 bytes }
  pure { identity := plan.artifactChecksum, spec, bytes := compiled.sourceBytes, sha256 := compiled.sourceSha256 }

/-- The file a proposal is named at: `<set>-<digest>.lean`, relative to wherever the caller writes. -/
def path (setName digest : String) : String := setName ++ "-" ++ digest ++ ".lean"

/-- The location a proposal is named at, with the tool that compiled it as its provenance. -/
def location (setName digest provenance : String) : SourceLocation :=
  { path := path setName digest, line := 1, column := 1, provenance }

end Umpire.Command.Promotion
