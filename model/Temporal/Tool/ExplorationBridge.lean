import Temporal.Feature.Nexus.Caller.Model
import Temporal.Tool.Bridge
import Testpilot.ProtoJSON
import Umpire.Exploration

/-!
# The exploration bridge

One exploratory campaign, driven from outside one frame at a time. The coordinator that runs the
Cases is Go's and knows no target, coordinate or Case family; what it sees is this protocol: a
frame in, a frame out, each one line of canonical JSON, each naming the set, the frame's sequence
number and, where one is in play, the candidate's opaque identity.

The frames are `initialize`, `next`, `observe` and `finish`. `initialize` names the set and opens
its campaign; `next` hands out the next candidate as one whole Lean-produced Case with its identity
and the target keys its planned path covers, or says the campaign is exhausted, or reports the
campaign's own defect; `observe` takes back the exact closed Run of the one outstanding candidate,
or the fact that its preparation was rejected, and answers with what was credited; `finish`
renders the summary and the counterexamples. A `next` while a candidate is outstanding, an
`observe` for a candidate that is not the outstanding one, a frame out of sequence, a frame naming
another set, or a frame after the campaign ended is rejected before any campaign call and leaves
the campaign as it was.

The Case each candidate carries is produced the way a `case` block's Cases are, from the
candidate's checked Model under the realization the block over the exploratory set names, with the
machine's claims, evidence catalog and field relations the same block emits. Its identity is the
candidate's: the Case ID `temporal.case.<set>.<digest>` and the fixture `<set>-<digest>`, where the
digest is the candidate's Plan checksum, so every candidate's run scope and workflow type are its
own and nothing is registered in the Temporal Case Registry.

A realization binds the class members it can perform, and a machine's table enumerates every
member: a planned path that performs a member with no binding and no timer behind it is one the
realization cannot run, and the Producer would assemble a Program without that step. The bridge
reads the bindings before producing: such a candidate is credited `unrealizable` at once -- to the
target it was planned for and to the class members not bound, never to the rest of its path, which
another candidate may reach -- reported in the `skipped` list of the frame that follows it, and
the campaign moves on within the same `next`.

The frames are exact: each kind admits a closed set of keys, and an object carrying any other key
-- one that would let the coordinator name a target, a coordinate or a Case family -- is rejected.
`initialize` names the Profile the coordinator runs under, by its identity; the bridge echoes it
on every frame it writes and refuses an `observe` under another. The budget's Limits are written
out by value on `initialized`, so the coordinator holds the bounds the campaign searched under.

What a Run says is read off its disposition, its cleanup and its Verdict alone: a completed Run
whose cleanup succeeded with a `satisfied` Verdict is `satisfied`; a Run whose cleanup succeeded
with a `violated` Verdict, completed or stopped by the Monitor, is `violated`; a preparation
rejection is `prepare-rejected`; anything else, including a well-formed Run for another Case, is
`inconclusive`. A Run the bridge cannot read -- one that does not decode, or names no Run or Case
-- is no observation at all: the frame is rejected and the candidate stays outstanding. The bridge
reads no Run Event: credit is the planned witness path.
-/

namespace Temporal.Tool.ExplorationBridge

open Umpire Umpire.Exploration
open Umpire.Command (DeclaredModel SetDeclaration)
open Temporal.Tool.Bridge (jsonString jsonObject jsonArray jsonStrings header Envelope)

/-! ### What produces a candidate's Case -/

/-- What the `case` block over an exploratory set emits: the realization and, for the set's
machine, the claims its actions make, its `evidence:` catalog and its Properties' field relations. -/
structure Production where
  realization : Umpire.Case.Producer.Realization
  claims : List Umpire.Case.Producer.ClassClaim := []
  catalog : List (String × String) := []
  relations : List Umpire.Case.Producer.FieldRelation := []
  /-- The machine's timers by name: a path step that fires one is the platform's, not an
  instruction the realization must bind. -/
  timers : List String := []

/-- One exploratory set bound to the Model it covers, the limits its budget names and what produces
its Cases. -/
structure Binding {Setup State Action Outcome Fact : Type}
    [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (model : DeclaredModel Setup State Action Outcome Fact) where
  set : SetDeclaration
  limits : Limits
  production : Production

/-- The Definition ID root every candidate's Case ID hangs off: the Temporal Case root, as a `case`
block's Cases use it. -/
def caseIdRoot : String := Temporal.Case.caseIdRoot

/-- The digest a candidate's identity contributes to its Case ID and fixture: the checksum's hex
digits, because a Case ID admits no colon. -/
def digestOf (identity : ArtifactChecksum) : String :=
  let rendered := identity.render
  if rendered.startsWith "sha256:" then String.ofList (rendered.toList.drop 7) else rendered

def caseIdOf (setName : String) (identity : ArtifactChecksum) : String :=
  caseIdRoot ++ "." ++ setName ++ "." ++ digestOf identity

def fixtureOf (setName : String) (identity : ArtifactChecksum) : String :=
  setName ++ "-" ++ digestOf identity

/-- A candidate as a frame carries it: its identity, the target it was planned for, every target
key its planned path covers, its Case identity and the Case itself, or the production error. -/
structure CandidateView where
  identity : ArtifactChecksum
  target : String
  covers : List String
  caseId : String
  fixture : String
  produced : Except Umpire.Case.Compiler.Error temporal.server.api.testpilot.v1.Case

/-- A candidate the bridge passed over: planned, found unrealizable, credited as such. -/
structure Skipped where
  identity : ArtifactChecksum
  target : String
  reason : String

/-! ### A campaign with its types erased

The Model a campaign covers fixes the types of its states, actions and outcomes, and the protocol
must not: `initialize` names the set by a string. A runner is a session with its types erased
into the four operations the protocol needs, each of which returns the next runner. -/

/-- What the summary says of one counterexample's proposal: its compiled source and digest, or why
it did not compile. -/
inductive ProposalReport where
  | compiled (proposal : Proposal)
  | failed (error : PromotionError)

mutual

inductive Runner where
  | mk (next : Unit → Step) (observe : ArtifactChecksum → Observation → Observed)
       (summary : Unit → Campaign.Summary) (ledger : Unit → List (String × TargetStatus))
       (proposals : Unit → List (ArtifactChecksum × ProposalReport))

/-- What `next` returns, each with the candidates passed over on the way to it. -/
inductive Step where
  | candidate (view : CandidateView) (skipped : List Skipped) (runner : Runner)
  | exhausted (skipped : List Skipped) (runner : Runner)
  | toolingFailure (failure : ToolingFailure) (skipped : List Skipped) (runner : Runner)
  | outstanding

/-- What `observe` returns: the target keys the observation covered, the statuses of every target
on the candidate's planned path after crediting, and the runner after; or a rejection, when the
identity is not the outstanding candidate's. -/
inductive Observed where
  | credited (covered : List String) (statuses : List (String × TargetStatus)) (runner : Runner)
  | rejected

end

namespace Runner

def next : Runner → Step
  | .mk next _ _ _ _ => next ()

def observe : Runner → ArtifactChecksum → Observation → Observed
  | .mk _ observe _ _ _, identity, observation => observe identity observation

def summary : Runner → Campaign.Summary
  | .mk _ _ summary _ _ => summary ()

def ledger : Runner → List (String × TargetStatus)
  | .mk _ _ _ ledger _ => ledger ()

/-- Every counterexample's proposal, compiled from the candidate the campaign retained. -/
def proposals : Runner → List (ArtifactChecksum × ProposalReport)
  | .mk _ _ _ _ proposals => proposals ()

private def emptySummary : Campaign.Summary :=
  { targets := 0, selected := 0, covered := 0, unreachable := 0, violated := 0, attempted := 0,
    unrealizable := 0, pending := 0, counterexamples := [], exhausted := true }

instance : Inhabited Runner :=
  ⟨.mk (fun _ => .outstanding) (fun _ _ => .rejected) (fun _ => emptySummary) (fun _ => []) (fun _ => [])⟩

instance : Inhabited Step := ⟨.outstanding⟩

variable {Setup State Action Outcome Fact : Type}
variable [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
variable [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
variable [DecidableEq Outcome] [DecidableEq Fact]
variable {model : DeclaredModel Setup State Action Outcome Fact}

/-- The class members on the candidate's path that no action binding resolves to and no timer
names, each by its Definition ID and its spelling: what the realization cannot perform. The path
is read the way the Producer reads it, the stated program or else the exact action sequence. -/
def unrealizable (production : Production) (checked : Umpire.Command.CheckedModel model) :
    List (DefinitionId × String) :=
  let input := Umpire.Command.producerInput checked
  let bound := production.realization.actions.map (·.resolve (some input.vocabulary))
  let path := (input.program.map (·.map (·.1))).getD (input.scenario.actionsExactly.getD [])
  (path.filterMap fun action =>
    if bound.contains action then none
    else match input.vocabulary.actions.find? (·.definitionId == action) with
      | some value => if production.timers.contains value.value then none else some (action, value.value)
      | none => some (action, action.value)).eraseDups

/-- The targets an unrealizable candidate is credited to: the one it was planned for, and the
class members on its path that the unbound members are. -/
def unrealizableCovers (candidate : Candidate model) (unbound : List (DefinitionId × String)) :
    List Umpire.Command.CoverageTarget :=
  candidate.selected :: candidate.covers.filter fun target =>
    match target with
    | Umpire.Command.CoverageTarget.classMember member _ _ _ _ =>
        targetKey target != targetKey candidate.selected && unbound.any (·.1 == member)
    | _ => false

/-- One candidate's Case, produced from its checked Model the way a `case` block's Cases are. The
machine's `evidence:` catalog confirms each fact along the witness, so the Case writes no evidence
lines of its own. -/
def produceCandidate (binding : Binding model) (candidate : Candidate model) : CandidateView :=
  let identity := candidate.identity
  let caseId := caseIdOf binding.set.name identity
  let fixture := fixtureOf binding.set.name identity
  { identity
    target := targetKey candidate.selected
    covers := candidate.covers.map targetKey
    caseId
    fixture
    produced := Umpire.Command.produce candidate.checked
      ({ caseId, fixture } : Umpire.Case.Producer.Identity)
      binding.production.realization (fun _ => [])
      (claims := binding.production.claims)
      (evidenceCatalog := binding.production.catalog)
      (relations := binding.production.relations) }

private def ledgerOf (session : Session model) : List (String × TargetStatus) :=
  session.campaign.ledger.entries.map fun entry => (targetKey entry.target, entry.status)

mutual

/-- The next realizable candidate: an unrealizable one is credited `unrealizable` on the spot,
without a Run, recorded as skipped, and the session moves on. -/
partial def advance (binding : Binding model) (session : Session model) (skipped : List Skipped) :
    Step :=
  match session.next with
  | .candidate candidate next =>
      match unrealizable binding.production candidate.checked with
      | [] => .candidate (produceCandidate binding candidate) skipped (ofSession binding next)
      | unbound =>
        let target := targetKey candidate.selected
        let reason := "unrealizable: the realization binds no " ++ ", ".intercalate (unbound.map (·.2))
        match next.observe [candidate.binding] .unrealizable (some (unrealizableCovers candidate unbound)) with
        | some after =>
            advance binding after (skipped ++ [{ identity := candidate.identity, target, reason }])
        | none =>
            .toolingFailure { target, reason := "a planned candidate could not be credited" }
              skipped (ofSession binding next)
  | .exhausted next => .exhausted skipped (ofSession binding next)
  | .toolingFailure failure next => .toolingFailure failure skipped (ofSession binding next)
  | .outstanding => .outstanding

/-- The session as a runner. -/
partial def ofSession (binding : Binding model) (session : Session model) : Runner :=
  .mk
    (fun _ => advance binding session [])
    (fun identity observation =>
      match session.outstanding with
      | some candidate =>
          if candidate.identity != identity then .rejected
          else match session.observe [candidate.binding] observation with
            | none => .rejected
            | some next =>
                let keys := candidate.covers.map targetKey
                let after := ledgerOf next
                let covered := if observation == .satisfied then keys else []
                .credited covered (after.filter fun (key, _) => keys.contains key) (ofSession binding next)
      | none => .rejected)
    (fun _ => session.campaign.summary)
    (fun _ => ledgerOf session)
    (fun _ => session.campaign.proposals.map fun (identity, proposal) =>
      (identity, match proposal with
        | .ok compiled => .compiled compiled
        | .error error => .failed error))

end

end Runner

/-- One set the bridge can open: its name, what `initialize` reports about it, and the campaign
over it, checked when opened. -/
structure Bound where
  name : String
  machine : String
  budget : String
  limits : Limits
  targets : List String
  campaign : Unit → Except CampaignError Runner

/-- Bind one exploratory set to its Model. -/
def Bound.of {Setup State Action Outcome Fact : Type}
    [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Fact]
    {model : DeclaredModel Setup State Action Outcome Fact} (binding : Binding model) : Bound :=
  { name := binding.set.name
    machine := ((binding.set.machine.map (·.value)).getD "")
    budget := binding.set.budget.getD ""
    limits := binding.limits
    targets := binding.set.targets.map targetKey
    campaign := fun _ =>
      (Campaign.check model binding.set binding.limits).map fun campaign =>
        Runner.ofSession binding (Session.begin campaign) }

/-! ### Reading a Run -/

/-- What a closed Run says about its candidate, read off the disposition, the cleanup and the
Verdict alone. -/
def observationOf (run : temporal.server.api.testpilot.v1.Run) : Observation :=
  let cleanupClosed := run.cleanup.any fun cleanup =>
    cleanup.status == temporal.server.api.testpilot.v1.CleanupStatus.CLEANUP_STATUS_SUCCEEDED
  let verdict : Option temporal.server.api.testpilot.v1.VerdictStatus := run.verdict.map (·.status)
  if !cleanupClosed then .inconclusive
  else match run.disposition, verdict with
    | .RUN_DISPOSITION_COMPLETED, some .VERDICT_STATUS_SATISFIED => .satisfied
    | .RUN_DISPOSITION_COMPLETED, some .VERDICT_STATUS_VIOLATED => .violated
    | .RUN_DISPOSITION_STOPPED_BY_MONITOR, some .VERDICT_STATUS_VIOLATED => .violated
    | _, _ => .inconclusive

/-- What a closed Run for one Case says, with why it credited nothing: a Run for another Case is
`inconclusive` and says so, a Run whose cleanup is not closed likewise, a decisive Run has no
detail. -/
def readRun (expectedCaseId : String) (run : temporal.server.api.testpilot.v1.Run) :
    Observation × String :=
  if run.case_id != expectedCaseId then
    (.inconclusive, s!"run {run.run_id} names Case {run.case_id}, not {expectedCaseId}")
  else match observationOf run with
    | .satisfied => (.satisfied, "")
    | .violated => (.violated, "")
    | _ =>
        if run.cleanup.all fun cleanup =>
            cleanup.status != temporal.server.api.testpilot.v1.CleanupStatus.CLEANUP_STATUS_SUCCEEDED
        then (.inconclusive, "cleanup is not closed")
        else (.inconclusive, "the Run is not a completed Run with a decisive Verdict")

private def parseOptions : Protobuf.Json.ParseOptions :=
  Protobuf.Json.ParseOptions.withGeneratedPool { discardUnknownFields := false, allowPartial := false }

/-- Decode one Run from its ProtoJSON object. -/
def decodeRun (json : Lean.Json) : IO (Except String temporal.server.api.testpilot.v1.Run) := do
  match ← Protobuf.Json.fromJson json temporal.server.api.testpilot.v1.Run parseOptions with
  | .ok run => pure (.ok run)
  | .error error => pure (.error (toString error))

/-! ### Frames -/

/-- What an `observe` frame carries for the outstanding candidate: its closed Run, or the fact that
its preparation was rejected. -/
inductive Result where
  | run (json : Lean.Json)
  | prepareRejected (detail : String)

/-- The frames the coordinator sends. -/
inductive Request where
  | initialize (profile : String)
  | next
  | observe (candidate : String) (profile : String) (result : Result)
  | finish (status : Option String)

/-- One frame as read: its sequence number, the set it names and what it asks. -/
structure Frame where
  seq : Nat
  setName : String
  request : Request

/-- The keys each frame kind admits, and no other: a frame is exact, so a key the bridge does not
read -- one that would name a target, a coordinate or a Case family -- rejects rather than being
dropped. -/
def admittedKeys : String → List String
  | "initialize" => ["frame", "seq", "set", "profile"]
  | "next" => ["frame", "seq", "set"]
  | "observe" => ["frame", "seq", "set", "candidate", "profile", "run", "prepareRejected"]
  | "finish" => ["frame", "seq", "set", "status"]
  | _ => []

/-- The terminal statuses `finish` may name when the campaign ended with targets pending: the
coordinator stopped it, or a campaign counter of its own tripped. -/
def stopStatuses : List String := ["stopped", "limit-reached"]

/-- Read one line as a frame, through the envelope every bridge reads. -/
def parseFrame (line : String) : Except String (Except (Nat × String) Frame) := do
  let envelope? ← Temporal.Tool.Bridge.parseEnvelope admittedKeys line
  pure <| envelope?.bind fun (envelope : Envelope) =>
    let request : Except String Request := do
      match envelope.kind with
      | "initialize" => pure (.initialize (← envelope.string "profile"))
      | "next" => pure .next
      | "observe" =>
          let candidate ← (envelope.string "candidate").mapError fun _ =>
            "observe names no `candidate`"
          let profile ← envelope.string "profile"
          match envelope.json.getObjVal? "run", envelope.json.getObjVal? "prepareRejected" with
          | .ok run, .error _ => pure (.observe candidate profile (.run run))
          | .error _, .ok (.str detail) => pure (.observe candidate profile (.prepareRejected detail))
          | .ok _, .ok _ => throw "observe carries both `run` and `prepareRejected`"
          | _, _ => throw "observe carries neither `run` nor a `prepareRejected` string"
      | "finish" =>
          match envelope.json.getObjVal? "status" with
          | .error _ => pure (.finish none)
          | .ok (.str status) =>
              if stopStatuses.contains status then pure (.finish (some status))
              else throw s!"finish names status `{status}`; one of {stopStatuses}"
          | .ok _ => throw "finish `status` is not a string"
      | other => throw s!"unknown frame `{other}`"
    match request with
    | .ok request => .ok { seq := envelope.seq, setName := envelope.setName, request }
    | .error reason => .error (envelope.seq, reason)

/-! ### Rendering -/

private def statusRows (statuses : List (String × TargetStatus)) : String :=
  jsonArray (statuses.map fun (target, status) =>
    jsonObject [("target", jsonString target), ("status", jsonString status.name)])

def renderRejected (seq : Nat) (reason : String) : String :=
  Temporal.Tool.Bridge.renderRejected seq reason

def renderInitialized (seq : Nat) (bound : Bound) (profile : String) : String :=
  jsonObject (header "initialized" seq bound.name profile ++ [
    ("machine", jsonString bound.machine),
    ("budget", jsonString bound.budget),
    ("limits", jsonObject [
      ("steps", toString bound.limits.steps.value),
      ("actions", toString bound.limits.actions.value),
      ("search", toString bound.limits.search.value)]),
    ("targets", jsonStrings bound.targets)])

private def skippedRows (skipped : List Skipped) : String :=
  jsonArray (skipped.map fun entry =>
    jsonObject [("candidate", jsonString entry.identity.render), ("target", jsonString entry.target),
      ("reason", jsonString entry.reason)])

/-- The candidate frame; `encodedCase` is the Case's canonical ProtoJSON, embedded verbatim. -/
def renderCandidate (seq : Nat) (setName profile : String) (view : CandidateView) (skipped : List Skipped)
    (encodedCase : String) : String :=
  jsonObject (header "candidate" seq setName profile ++ [
    ("candidate", jsonString view.identity.render),
    ("target", jsonString view.target),
    ("covers", jsonStrings view.covers),
    ("caseId", jsonString view.caseId),
    ("fixture", jsonString view.fixture),
    ("skipped", skippedRows skipped),
    ("case", encodedCase)])

def renderExhausted (seq : Nat) (setName profile : String) (skipped : List Skipped) : String :=
  jsonObject (header "exhausted" seq setName profile ++ [("skipped", skippedRows skipped)])

def renderToolingFailure (seq : Nat) (setName profile : String) (failure : ToolingFailure)
    (skipped : List Skipped) : String :=
  jsonObject (header "toolingFailure" seq setName profile ++ [
    ("target", jsonString failure.target),
    ("reason", jsonString failure.reason),
    ("skipped", skippedRows skipped)])

def renderCredited (seq : Nat) (setName profile : String) (identity : ArtifactChecksum)
    (observation : Observation) (detail : String) (covered : List String)
    (statuses : List (String × TargetStatus)) : String :=
  jsonObject (header "credited" seq setName profile ++ [
    ("candidate", jsonString identity.render),
    ("observation", jsonString observation.name),
    ("detail", jsonString detail),
    ("credited", jsonStrings covered),
    ("statuses", statusRows statuses)])

/-- The summary frame. Each counterexample carries its proposal: the promotion source's SHA-256
and bytes when it compiled, or the reason it did not, so that whoever runs the campaign writes
the source where it names and compares the digest across runs. -/
def renderFinished (seq : Nat) (setName profile : String) (status : String) (summary : Campaign.Summary)
    (ledger : List (String × TargetStatus)) (proposals : List (ArtifactChecksum × ProposalReport)) : String :=
  jsonObject (header "finished" seq setName profile ++ [
    ("status", jsonString status),
    ("summary", jsonObject [
      ("targets", toString summary.targets),
      ("selected", toString summary.selected),
      ("covered", toString summary.covered),
      ("unreachable", toString summary.unreachable),
      ("violated", toString summary.violated),
      ("attempted", toString summary.attempted),
      ("unrealizable", toString summary.unrealizable),
      ("pending", toString summary.pending),
      ("exhausted", if summary.exhausted then "true" else "false")]),
    ("counterexamples", jsonArray (summary.counterexamples.map fun sample =>
      let proposal := (proposals.find? (·.1 == sample.candidate)).map (·.2)
      jsonObject ([
        ("className", jsonString sample.className),
        ("target", jsonString (targetKey sample.target)),
        ("candidate", jsonString sample.candidate.render)] ++
        match proposal with
        | some (.compiled compiled) => [
            ("promotionSourceSha256", jsonString compiled.sha256),
            ("promotionSourcePath", jsonString compiled.spec.sourceLocation.path),
            ("promotionSource", jsonString compiled.bytes)]
        | some (.failed error) => [
            ("promotionSourceSha256", "null"),
            ("promotionError", jsonString s!"{repr error.kind}: {error.detail}")]
        | none => [
            ("promotionSourceSha256", "null"),
            ("promotionError", jsonString "the campaign retained no candidate for this counterexample")]))),
    ("ledger", statusRows ledger)])

/-! ### The protocol -/

/-- Where the campaign stands between frames. -/
inductive Phase where
  /-- No `initialize` yet. -/
  | closed
  | running
  /-- `next` said exhausted; only `finish` remains. -/
  | exhausted
  /-- The campaign's own defect ended it; only `finish` remains. -/
  | failed (failure : ToolingFailure)
  deriving Repr

structure State where
  phase : Phase := .closed
  /-- The sequence number the next frame must carry. -/
  expected : Nat := 1
  setName : String := ""
  /-- The identity of the Profile the coordinator runs under, named at `initialize`. -/
  profile : String := ""
  runner : Runner := default
  /-- The outstanding candidate, whose Case the coordinator holds. -/
  outstanding : Option CandidateView := none
  /-- Candidates already observed, whose identities a later `observe` is stale for. -/
  seen : List ArtifactChecksum := []

/-- What one accepted frame produces: the frame to write and the state after, and whether the
protocol is complete. -/
abbrev Outcome := Temporal.Tool.Bridge.Outcome State

/-- The effects the bridge runs under, injected so a test measures what reached each stream. -/
abbrev Effects := Temporal.Tool.Bridge.Effects

/-- The reason a frame is rejected, or none when it is in order. Checked before any campaign call. -/
def rejection (state : State) (frame : Frame) : Option String :=
  (Temporal.Tool.Bridge.sequenceRejection state.expected frame.seq) <|>
  match state.phase, frame.request with
    | .closed, .initialize _ => none
    | .closed, _ => some "no campaign is open; send `initialize` first"
    | _, .initialize _ => some s!"campaign over {state.setName} is already open"
    | phase, request =>
      if frame.setName != state.setName then
        some s!"frame names set {frame.setName}; the campaign is over {state.setName}"
      else
        let ended : Option String := match phase with
          | .exhausted => some "the campaign is exhausted; send `finish`"
          | .failed failure => some s!"the campaign ended on a tooling failure ({failure.reason}); send `finish`"
          | _ => none
        match request with
        | .finish _ => none
        | .next =>
            ended <|> match state.outstanding with
              | some view => some s!"candidate {view.identity.render} is outstanding; send `observe`"
              | none => none
        | .observe candidate profile _ =>
            ended <|>
              (Temporal.Tool.Bridge.profileRejection state.profile profile) <|>
              match state.outstanding with
              | none =>
                  if state.seen.any (·.render == candidate) then
                    some s!"stale observe: candidate {candidate} was already observed"
                  else some "no candidate is outstanding; send `next`"
              | some view =>
                  if view.identity.render == candidate then none
                  else some s!"crossed observe: candidate {view.identity.render} is outstanding, not {candidate}"
        | .initialize _ => none

/-- The terminal status `finish` reports. -/
def finishStatus (state : State) (requested : Option String) : String :=
  match state.phase with
  | .failed _ => "tooling-failure"
  | .exhausted => "exhausted"
  | _ => if state.runner.summary.exhausted then "exhausted" else requested.getD "stopped"

/-- Apply one in-order frame. -/
def step (effects : Effects) (bound : List Bound) (state : State) (frame : Frame) : IO Outcome := do
  let seq := frame.seq
  let accepted (line : String) (state : State) : Outcome :=
    .reply line { state with expected := seq + 1 }
  -- The campaign's own defect, wherever it surfaced, ends it the same way.
  let failWith (failure : ToolingFailure) (skipped : List Skipped) (runner : Runner)
      (seen : List ArtifactChecksum) : Outcome :=
    accepted (renderToolingFailure seq state.setName state.profile failure skipped)
      { state with phase := .failed failure, runner, seen }
  match frame.request with
  | .initialize profile =>
      if profile.isEmpty then
        pure (Temporal.Tool.Bridge.Outcome.reply (renderRejected seq "initialize names an empty `profile`") state)
      else match bound.find? (·.name == frame.setName) with
      | none =>
          pure (Temporal.Tool.Bridge.Outcome.reply (renderRejected seq s!"unknown set {frame.setName}; the bridge binds {bound.map (·.name)}") state)
      | some found =>
          match found.campaign () with
          | .error error => pure (Temporal.Tool.Bridge.Outcome.reply (renderRejected seq s!"set {frame.setName} is not a campaign: {error.render}") state)
          | .ok runner =>
              pure (accepted (renderInitialized seq found profile)
                { state with phase := .running, setName := found.name, profile, runner })
  | .next =>
      let step := state.runner.next
      let skipped := match step with
        | .candidate _ skipped _ => skipped
        | .exhausted skipped _ => skipped
        | .toolingFailure _ skipped _ => skipped
        | .outstanding => []
      for entry in skipped do
        effects.writeProgress s!"skipped {entry.identity.render} {entry.target} {entry.reason}"
      let seen := state.seen ++ skipped.map (·.identity)
      match step with
      | .outstanding =>
          pure (Temporal.Tool.Bridge.Outcome.reply (renderRejected seq "a candidate is outstanding") state)
      | .exhausted skipped runner =>
          pure (accepted (renderExhausted seq state.setName state.profile skipped)
            { state with phase := .exhausted, runner, seen })
      | .toolingFailure failure skipped runner => pure (failWith failure skipped runner seen)
      | .candidate view skipped runner =>
          match view.produced with
          | .error error =>
              let failure : ToolingFailure :=
                { target := view.target, reason := s!"production: {error.construct} at {error.sourceDefinitionId}" }
              pure (failWith failure skipped runner seen)
          | .ok produced =>
              match ← Testpilot.ProtoJSON.canonical produced with
              | .error error =>
                  pure (failWith { target := view.target, reason := s!"encoding: {error}" } skipped runner seen)
              | .ok encoded =>
                  effects.writeProgress s!"candidate {view.identity.render} {view.target}"
                  pure (accepted (renderCandidate seq state.setName state.profile view skipped encoded)
                    { state with runner, outstanding := some view, seen })
  | .observe candidate _ result =>
      let some view := state.outstanding
        | pure (Temporal.Tool.Bridge.Outcome.reply (renderRejected seq "no candidate is outstanding") state)
      -- A Run the bridge cannot read is no observation: the frame is rejected and the candidate
      -- stays outstanding, so a garbled frame spends nothing.
      let read : Except String (Observation × String) ← match result with
        | .prepareRejected _ => pure (.ok (Observation.prepareRejected, "preparation was rejected"))
        | .run json =>
            match ← decodeRun json with
            | .error reason => pure (.error s!"run does not decode: {reason}")
            | .ok run =>
                if run.run_id.isEmpty || run.case_id.isEmpty then
                  pure (.error "run names no `runId` or no `caseId`")
                else pure (.ok (readRun view.caseId run))
      match read with
      | .error reason => pure (Temporal.Tool.Bridge.Outcome.reply (renderRejected seq reason) state)
      | .ok (observation, detail) =>
          match state.runner.observe view.identity observation with
          | .rejected =>
              pure (Temporal.Tool.Bridge.Outcome.reply (renderRejected seq s!"candidate {candidate} is not the outstanding candidate") state)
          | .credited covered statuses runner =>
              effects.writeProgress s!"observed {view.identity.render} {observation.name}"
              pure (accepted (renderCredited seq state.setName state.profile view.identity observation
                  detail covered statuses)
                { state with runner, outstanding := none, seen := state.seen ++ [view.identity] })
  | .finish requested =>
      pure (Temporal.Tool.Bridge.Outcome.finished (renderFinished seq state.setName state.profile (finishStatus state requested)
        state.runner.summary state.runner.ledger state.runner.proposals))

/-- The diagnostic prefix every failure outside a frame carries. -/
def diagnosticPrefix : String := "umpire-explore:"

/-- The exploration protocol over the sets the bridge binds. -/
def protocol (effects : Effects) (bound : List Bound) : Temporal.Tool.Bridge.Protocol State Frame :=
  { diagnosticPrefix
    parse := parseFrame
    seqOf := (·.seq)
    reject := rejection
    step := step effects bound }

/-- Serve frames until `finish`. A line that is no frame, or stdin closing before `finish`, is a
failure outside the protocol: a diagnostic on stderr and a non-zero exit. -/
def serve (effects : Effects) (bound : List Bound) : IO UInt32 :=
  Temporal.Tool.Bridge.serve effects (protocol effects bound) {}

/-! ### The sets the bridge binds -/

/-- The caller Model's exploratory set under the functional set's realization, with what the
exploratory `case` block emits for the protocol machine. -/
def nexusCallerBinding : Binding Temporal.Feature.Nexus.Caller.nexusProtocol :=
  { set := Temporal.Feature.Nexus.Caller.nexusCallerExploration
    limits := Temporal.Feature.Nexus.Caller.four
    production := {
      realization := Temporal.Feature.Nexus.Caller.nexusCallerExplorationCases.realization
      claims := Temporal.Feature.Nexus.Caller.nexusCallerExplorationCases.claims
      catalog := Temporal.Feature.Nexus.Caller.nexusCallerExplorationCases.catalog
      relations := Temporal.Feature.Nexus.Caller.nexusCallerExplorationCases.relations
      timers := Temporal.Feature.Nexus.Caller.nexusCallerExplorationCases.timers } }

/-- Every set the bridge opens. -/
def boundSets : List Bound := [Bound.of nexusCallerBinding]

/-- The effects of the process boundary. -/
def processEffects : IO Effects := Temporal.Tool.Bridge.processEffects

end Temporal.Tool.ExplorationBridge
