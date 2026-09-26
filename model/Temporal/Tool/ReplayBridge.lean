import Temporal.Feature.Nexus.Control.Model
import Temporal.Tool.Bridge
import Temporal.Tool.ExplorationBridge
import Testpilot.ProtoJSON
import Umpire.Fingerprint
import Umpire.Replay

/-!
# The replay bridge

One reduction of one replay subject, driven from outside one frame at a time. The coordinator is
Go's: it admitted a violated Run, reruns Cases and classes their Runs, and knows no Query, edit or
coordinate. What it sees is this protocol, framed as the exploration bridge frames its own: one
line of canonical JSON each way, each naming the set and the frame's sequence number, the Profile
identity echoed on every frame the bridge writes.

The frames are `admit`, `next`, `observe` and `finish`. `admit` names the set and either the Query
(a functional set's) or the exploration target key (an exploratory set's) whose Case the subject
is, and the subject's identity: the SHA-256, in lowercase hex, of the Case's compact canonical
bytes. The bridge recovers the admitted Query through its binding table, re-produces the Case under
the set's realization and admits the subject only when the bytes it produces are the subject's;
otherwise it answers `crossed` and the protocol ends. `admitted` lists the sweep's edits in the
order they are tried. `next` hands out the next candidate -- the edit, the candidate's digest (the
edited Query's Plan checksum), its Case ID `temporal.case.<set>.<digest>` and fixture
`<set>-<digest>`, its Case checksum as `identity`, and the whole Case -- after passing over, and
listing, every edit the Model does not admit (`inapplicable`) or whose Case the Producer rejects
(`rejected`); or it says the sweep is `exhausted`. `observe` takes back what the coordinator
decided about the outstanding candidate's Runs, a class (`reproduced`, `not-reproduced` or
`indeterminate`, after the one retry) or the fact that its preparation was rejected, and answers
`settled`. `finish` reports the result -- `minimized`, `irreducible` or `incomplete` -- the digest
retained, and every edit's fate, an edit never settled being `not-tried` (or `unsettled`, with its
candidate's digest, when its candidate was outstanding at the end); a `minimized` or `irreducible`
result carries the retained candidate's review-only proposal, compiled by
`Umpire.Command.Promotion.propose` from its admitted Query and named `<set>-<digest>.lean` by its
digest, or why it did not compile. The proposal renders the Model's expected trace, never a Run.

Frames are exact, as the exploration bridge's are: each kind admits a closed set of keys, so the
coordinator cannot name an edit, a coordinate or a Case. A frame out of sequence or duplicated, one
naming another set or Profile, an `observe` for a candidate that is not outstanding or was already
settled, a `next` while a candidate is outstanding or after the reduction ended, and a line longer
than the bridge parses are rejected before any Query is admitted, and leave the reduction as it was.
Go receives whole Cases and returns classes; it never edits a Case.
-/

namespace Temporal.Tool.ReplayBridge

open Umpire Umpire.Command Umpire.Replay
open Temporal.Tool.Bridge (jsonString jsonObject jsonArray header Envelope)

/-! ### The binding table -/

/-- One functional Query the bridge can admit a subject of: its source, the identity its `case`
block produced it under, and what the block produces under. -/
structure QueryBinding {Setup State Action Outcome Fact : Type}
    [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (model : DeclaredModel Setup State Action Outcome Fact) where
  name : String
  source : QuerySource model
  identity : Umpire.Case.Producer.Identity
  production : Production

/-- One functional set over a Model, with its Queries. -/
structure FunctionalBinding {Setup State Action Outcome Fact : Type}
    [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (model : DeclaredModel Setup State Action Outcome Fact) where
  set : SetDeclaration
  queries : List (QueryBinding model)

/-- What `admit` names: a functional set's Query, or an exploratory set's target key. -/
inductive Named where
  | query (name : String)
  | target (key : String)

/-- What one edit's candidate came to before any Run: not admitted, or produced. -/
inductive Attempt where
  | inapplicable (reason : String)
  | produced (digest caseId fixture : String)
      (produced : Except String temporal.server.api.testpilot.v1.Case)

/-- A recovered subject with its Model's types erased: the Case its Query produces, the sweep over
its Scenario, and how one candidate is admitted and produced. -/
structure Recovered where
  digest : String
  caseId : String
  fixture : String
  produced : Except String temporal.server.api.testpilot.v1.Case
  reduction : Reduction
  attempt : List Nat → Attempt
  /-- The review-only proposal of the candidate that keeps `kept`, compiled from its admitted
  Query and named by its digest. -/
  propose : List Nat → Except String Umpire.Command.Promotion.Proposal

/-- One set the bridge can admit a subject of. -/
structure Bound where
  name : String
  recover : Named → Except String Recovered

private def productionReason (error : Umpire.Case.Compiler.Error) : String :=
  s!"production: {error.construct} at {error.sourceDefinitionId}"

/-- A candidate's Case identity: the Temporal Case root, the set and the candidate's digest. -/
def candidateIdentity (setName digest : String) : Umpire.Case.Producer.Identity :=
  { caseId := Temporal.Case.caseIdRoot ++ "." ++ setName ++ "." ++ digest
    fixture := setName ++ "-" ++ digest }

section
variable {Setup State Action Outcome Fact : Type}
variable [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
variable [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
variable [DecidableEq Outcome] [DecidableEq Fact]
variable {model : DeclaredModel Setup State Action Outcome Fact}

/-- Recover one Query: admit its source, produce its Case under `identity` (or under its own digest
when none), and set out the sweep over its Scenario. -/
def recoverSource (setName : String) (source : QuerySource model)
    (identity : Option Umpire.Case.Producer.Identity) (production : Production) :
    Except String Recovered := do
  let admitted ← source.admit.mapError admissionReason
  let some plan := admitted.checked.run.artifact
    | throw "the admitted Query carries no Plan"
  let digest := Umpire.Exploration.candidateDigest plan.artifactChecksum
  let actions ← editable (source.behavior admitted.checked.vocabulary)
  let identity := identity.getD (candidateIdentity setName digest)
  let subjectForm : Admitted model := { admitted, plan, digest }
  pure {
    digest
    caseId := identity.caseId
    fixture := identity.fixture
    produced := (subjectForm.produce identity production).mapError productionReason
    reduction := Reduction.start actions
    attempt := fun kept =>
      match admitKept source kept with
      | .error reason => .inapplicable reason
      | .ok candidate =>
          let named := candidateIdentity setName candidate.digest
          .produced candidate.digest named.caseId named.fixture
            ((candidate.produce named production).mapError productionReason)
    propose := fun kept => do
      let retained ← admitKept source kept
      let location := Umpire.Command.Promotion.location setName retained.digest "umpire-replay"
      (Umpire.Command.Promotion.propose retained.admitted retained.plan location).mapError
        fun error => s!"{repr error.kind}: {error.detail}" }

/-- Bind one functional set: `admit` names one of its Queries. -/
def Bound.functional (binding : FunctionalBinding model) : Bound :=
  { name := binding.set.name
    recover := fun named =>
      match named with
      | .target key => .error s!"set {binding.set.name} is functional; name a Query, not target {key}"
      | .query name =>
          match binding.queries.find? (·.name == name) with
          | none => .error s!"set {binding.set.name} has no Query {name}; it binds {binding.queries.map (·.name)}"
          | some query => recoverSource binding.set.name query.source (some query.identity) query.production }

/-- Bind one exploratory set: `admit` names a target key, whose candidate the campaign plans the
way the exploration bridge plans it, and whose Case is named by its digest. -/
def Bound.exploratory (binding : Temporal.Tool.ExplorationBridge.Binding model) : Bound :=
  { name := binding.set.name
    recover := fun named =>
      match named with
      | .query name => .error s!"set {binding.set.name} is exploratory; name a target, not Query {name}"
      | .target key =>
          match binding.set.targets.find? (Umpire.Exploration.targetKey · == key) with
          | none => .error s!"set {binding.set.name} has no target {key}"
          | some target =>
              match Umpire.Exploration.Campaign.planTarget model binding.set binding.limits target with
              | none => .error s!"target {key} is unreachable within the set's limits"
              | some (queryKey, property, behavior) =>
                  recoverSource binding.set.name
                    ({ key := queryKey, limits := binding.limits, property, behavior } : QuerySource model)
                    none
                    { realization := binding.production.realization
                      claims := binding.production.claims
                      catalog := binding.production.catalog
                      relations := binding.production.relations } }

end

/-! ### Frames -/

/-- What an `observe` frame says of the outstanding candidate's Runs. -/
inductive Decided where
  | reproduced
  | notReproduced
  | indeterminate
  | prepareRejected (detail : String)

def Decided.fate : Decided → Fate
  | .reproduced => .retained
  | .notReproduced => .notReproduced
  | .indeterminate => .undecided
  | .prepareRejected detail => .rejected s!"preparation: {detail}"

inductive Request where
  | admit (profile : String) (named : Named) (identity : String)
  | next
  | observe (candidate : String) (profile : String) (decided : Decided)
  | finish (status : Option String)

structure Frame where
  seq : Nat
  setName : String
  request : Request

def admittedKeys : String → List String
  | "admit" => ["frame", "seq", "set", "profile", "query", "target", "identity"]
  | "next" => ["frame", "seq", "set"]
  | "observe" => ["frame", "seq", "set", "candidate", "profile", "class", "prepareRejected"]
  | "finish" => ["frame", "seq", "set", "status"]
  | _ => []

/-- The statuses `finish` may name when the coordinator ends a reduction early. -/
def stopStatuses : List String := ["stopped", "limit-reached"]

/-- The classes `observe` names. -/
def classes : List String := ["reproduced", "not-reproduced", "indeterminate"]

/-- The longest line the bridge reads: an `observe` carries a class, never a Run. -/
def maxLineBytes : Nat := 65536

private def isHexDigest (value : String) : Bool :=
  value.length == 64 && value.all fun c => c.isDigit || ('a' ≤ c && c ≤ 'f')

def parseFrame (line : String) : Except String (Except (Nat × String) Frame) := do
  let envelope? ← Temporal.Tool.Bridge.parseEnvelope admittedKeys line
  pure <| envelope?.bind fun (envelope : Envelope) =>
    let request : Except String Request := do
      match envelope.kind with
      | "admit" =>
          let profile ← envelope.string "profile"
          let identity ← envelope.string "identity"
          unless isHexDigest identity do
            throw "admit `identity` is not a lowercase hex SHA-256"
          let named ← match ← envelope.string? "query", ← envelope.string? "target" with
            | some query, none => pure (Named.query query)
            | none, some target => pure (Named.target target)
            | some _, some _ => throw "admit names both a `query` and a `target`"
            | none, none => throw "admit names neither a `query` nor a `target`"
          pure (.admit profile named identity)
      | "next" => pure .next
      | "observe" =>
          let candidate ← envelope.string "candidate"
          let profile ← envelope.string "profile"
          match ← envelope.string? "class", ← envelope.string? "prepareRejected" with
          | some "reproduced", none => pure (.observe candidate profile .reproduced)
          | some "not-reproduced", none => pure (.observe candidate profile .notReproduced)
          | some "indeterminate", none => pure (.observe candidate profile .indeterminate)
          | some other, none => throw s!"observe names class `{other}`; one of {classes}"
          | none, some detail => pure (.observe candidate profile (.prepareRejected detail))
          | some _, some _ => throw "observe carries both `class` and `prepareRejected`"
          | none, none => throw "observe carries neither `class` nor `prepareRejected`"
      | "finish" =>
          match ← envelope.string? "status" with
          | none => pure (.finish none)
          | some status =>
              if stopStatuses.contains status then pure (.finish (some status))
              else throw s!"finish names status `{status}`; one of {stopStatuses}"
      | other => throw s!"unknown frame `{other}`"
    match request with
    | .ok request => .ok { seq := envelope.seq, setName := envelope.setName, request }
    | .error reason => .error (envelope.seq, reason)

/-! ### Rendering -/

/-- One candidate as `next` hands it out: its edit, its digest (the edited Query's Plan checksum),
its Case identity, its Case checksum and its canonical bytes. -/
structure Candidate where
  edit : Edit
  digest : String
  caseId : String
  fixture : String
  identity : String
  encoded : String


def renderRejected (seq : Nat) (reason : String) : String :=
  Temporal.Tool.Bridge.renderRejected seq reason

private def editMembers (edit : Edit) : List (String × String) :=
  [("edit", jsonString edit.name), ("index", toString edit.index),
    ("action", jsonString edit.action.value)]

private def settledRow (settled : Settled) : String :=
  jsonObject (editMembers settled.edit ++ [
    ("fate", jsonString settled.fate.name),
    ("reason", jsonString settled.fate.reason),
    ("candidate", match settled.candidate with
      | some digest => jsonString digest
      | none => "null")])

private def settledRows (settled : List Settled) : String := jsonArray (settled.map settledRow)

def renderAdmitted (seq : Nat) (setName profile : String) (recovered : Recovered) (identity : String) :
    String :=
  jsonObject (header "admitted" seq setName profile ++ [
    ("subject", jsonString recovered.digest),
    ("caseId", jsonString recovered.caseId),
    ("fixture", jsonString recovered.fixture),
    ("identity", jsonString identity),
    ("edits", jsonArray (recovered.reduction.sweep.map fun edit => jsonObject (editMembers edit))),
    ("capped", if recovered.reduction.capped then "true" else "false")])

def renderCrossed (seq : Nat) (setName profile reason : String) : String :=
  jsonObject (header "crossed" seq setName profile ++ [("reason", jsonString reason)])

def renderCandidate (seq : Nat) (setName profile : String) (candidate : Candidate)
    (skipped : List Settled) : String :=
  jsonObject (header "candidate" seq setName profile ++ [("candidate", jsonString candidate.digest)] ++
    editMembers candidate.edit ++ [
    ("caseId", jsonString candidate.caseId),
    ("fixture", jsonString candidate.fixture),
    ("identity", jsonString candidate.identity),
    ("skipped", settledRows skipped),
    ("case", candidate.encoded)])

def renderExhausted (seq : Nat) (setName profile : String) (skipped : List Settled) : String :=
  jsonObject (header "exhausted" seq setName profile ++ [("skipped", settledRows skipped)])

def renderSettled (seq : Nat) (setName profile : String) (settled : Settled) (retained : String) :
    String :=
  jsonObject (header "settled" seq setName profile ++ [
    ("candidate", match settled.candidate with
      | some digest => jsonString digest
      | none => "null")] ++ editMembers settled.edit ++ [
    ("fate", jsonString settled.fate.name),
    ("reason", jsonString settled.fate.reason),
    ("retained", jsonString retained)])

/-- An edit the sweep never settled: the one whose candidate was outstanding when the reduction
ended is `unsettled` with that candidate's digest; the rest are `not-tried`. -/
private def unsettledRow (outstanding : Option (Edit × String)) (edit : Edit) : String :=
  let (fate, candidate) := match outstanding with
    | some (pending, digest) => if pending == edit then ("unsettled", jsonString digest) else ("not-tried", "null")
    | none => ("not-tried", "null")
  jsonObject (editMembers edit ++ [("fate", jsonString fate), ("reason", jsonString ""),
    ("candidate", candidate)])

/-- The proposal member: the retained candidate's compiled source with its SHA-256 and path, or
why it did not compile, or `null` when the result was incomplete and nothing is proposed. -/
private def proposalMember (retained : String)
    (proposal : Option (Except String Umpire.Command.Promotion.Proposal)) : String :=
  match proposal with
  | none => "null"
  | some (.ok compiled) => jsonObject [
      ("digest", jsonString retained),
      ("promotionSourceSha256", jsonString compiled.sha256),
      ("promotionSourcePath", jsonString compiled.spec.sourceLocation.path),
      ("promotionSource", jsonString compiled.bytes)]
  | some (.error reason) => jsonObject [
      ("digest", jsonString retained),
      ("promotionSourceSha256", "null"),
      ("promotionError", jsonString reason)]

/-- The result, with every edit of the sweep -- each settled edit's fate in the order it was
settled, then each edit the sweep never settled -- and, for a `minimized` or `irreducible` result,
the retained candidate's proposal. -/
def renderFinished (seq : Nat) (setName profile : String) (result : Result) (subject retained : String)
    (reduction : Reduction) (outstanding : Option (Edit × String))
    (proposal : Option (Except String Umpire.Command.Promotion.Proposal)) : String :=
  jsonObject (header "finished" seq setName profile ++ [
    ("status", jsonString result.name),
    ("reason", jsonString result.reason),
    ("subject", jsonString subject),
    ("retained", jsonString retained),
    ("edits", jsonArray (reduction.settled.map settledRow ++
      reduction.pending.map (unsettledRow outstanding))),
    ("proposal", proposalMember retained proposal)])

/-! ### The protocol -/

inductive Phase where
  | closed
  | reducing
  /-- The sweep has no edit left, or the reduction ended; only `finish` remains. -/
  | ended
  deriving BEq, Repr

structure State where
  phase : Phase := .closed
  expected : Nat := 1
  setName : String := ""
  profile : String := ""
  recovered : Option Recovered := none
  reduction : Reduction := Reduction.start []
  /-- The digest later edits apply to: the subject's, then each retained candidate's. -/
  retained : String := ""
  outstanding : Option (Edit × String) := none
  /-- Candidates already settled, whose digests a later `observe` is stale for. -/
  seen : List String := []

abbrev Outcome := Temporal.Tool.Bridge.Outcome State

/-- The reason a frame is rejected, or none when it is in order. Checked before any Query is
admitted. -/
def rejection (state : State) (frame : Frame) : Option String :=
  (Temporal.Tool.Bridge.sequenceRejection state.expected frame.seq) <|>
  match state.phase, frame.request with
  | .closed, .admit .. => none
  | .closed, _ => some "no subject is admitted; send `admit` first"
  | _, .admit .. => some s!"a subject of {state.setName} is already admitted"
  | phase, request =>
    if frame.setName != state.setName then
      some s!"frame names set {frame.setName}; the subject is of {state.setName}"
    else
      let ended : Option String :=
        if phase == .ended then some "the reduction ended; send `finish`" else none
      match request with
      | .finish _ => none
      | .next =>
          ended <|> match state.outstanding with
            | some (_, digest) => some s!"candidate {digest} is outstanding; send `observe`"
            | none => none
      | .observe candidate profile _ =>
          (Temporal.Tool.Bridge.profileRejection state.profile profile) <|>
          match state.outstanding with
          | none =>
              if state.seen.contains candidate then
                some s!"stale observe: candidate {candidate} was already settled"
              else ended <|> some "no candidate is outstanding; send `next`"
          | some (_, digest) =>
              if digest == candidate then none
              else some s!"crossed observe: candidate {digest} is outstanding, not {candidate}"
      | .admit .. => none

/-- The Case's identity: the SHA-256 of its compact canonical bytes, in lowercase hex. -/
def caseIdentity (encoded : String) : String := Umpire.Fingerprint.sha256Hex encoded

/-- Hand out the next candidate, passing over and recording every edit that produces no Case. -/
def advance (reduction : Reduction) (recovered : Recovered) :
    IO (Reduction × List Settled × Option Candidate) := do
  let mut reduction := reduction
  let mut skipped : List Settled := []
  for _ in [0:reduction.pending.length] do
    let some edit := reduction.next? | break
    match recovered.attempt (reduction.candidateKept edit) with
    | .inapplicable reason =>
        let fate := Fate.inapplicable reason
        reduction := reduction.settle edit fate
        skipped := skipped ++ [({ edit, fate } : Settled)]
    | .produced digest caseId fixture produced =>
        let encoded ← match produced with
          | .error reason => pure (Except.error reason)
          | .ok value =>
              match ← Testpilot.ProtoJSON.canonical value with
              | .ok encoded => pure (.ok encoded)
              | .error error => pure (.error s!"encoding: {error}")
        match encoded with
        | .error reason =>
            let fate := Fate.rejected reason
            reduction := reduction.settle edit fate (some digest)
            skipped := skipped ++ [({ edit, fate, candidate := some digest } : Settled)]
        | .ok encoded =>
            return (reduction, skipped,
              some { edit, digest, caseId, fixture, identity := caseIdentity encoded, encoded })
  pure (reduction, skipped, none)

/-- Apply one in-order frame. -/
def step (effects : Temporal.Tool.Bridge.Effects) (bound : List Bound) (state : State)
    (frame : Frame) : IO Outcome := do
  let seq := frame.seq
  let accepted (line : String) (state : State) : Outcome :=
    .reply line { state with expected := seq + 1 }
  match frame.request with
  | .admit profile named identity =>
      if profile.isEmpty then
        return .reply (renderRejected seq "admit names an empty `profile`") state
      let some found := bound.find? (·.name == frame.setName)
        | return .reply (renderRejected seq
            s!"unknown set {frame.setName}; the bridge binds {bound.map (·.name)}") state
      match found.recover named with
      | .error reason => return .reply (renderRejected seq reason) state
      | .ok recovered =>
          let encoded ← match recovered.produced with
            | .error reason => pure (Except.error reason)
            | .ok value =>
                match ← Testpilot.ProtoJSON.canonical value with
                | .ok encoded => pure (.ok encoded)
                | .error error => pure (.error s!"encoding: {error}")
          match encoded with
          | .error reason =>
              return .finished (renderCrossed seq frame.setName profile
                s!"the set's Query does not produce a Case: {reason}")
          | .ok encoded =>
              let produced := caseIdentity encoded
              if produced != identity then
                return .finished (renderCrossed seq frame.setName profile
                  s!"the set produces {recovered.caseId} as {produced}, not the subject's {identity}")
              effects.writeProgress s!"admitted {recovered.digest} {recovered.reduction.sweep.length} edits"
              return accepted (renderAdmitted seq frame.setName profile recovered identity)
                { state with
                  phase := .reducing
                  setName := frame.setName
                  profile := profile
                  recovered := some recovered
                  reduction := recovered.reduction
                  retained := recovered.digest }
  | .next =>
      let some recovered := state.recovered
        | return .reply (renderRejected seq "no subject is admitted") state
      let (reduction, skipped, candidate) ← advance state.reduction recovered
      for entry in skipped do
        effects.writeProgress s!"skipped {entry.edit.name} {entry.fate.name} {entry.fate.reason}"
      let seen := state.seen ++ skipped.filterMap (·.candidate)
      match candidate with
      | none =>
          return accepted (renderExhausted seq state.setName state.profile skipped)
            { state with phase := .ended, reduction, seen }
      | some candidate =>
          effects.writeProgress s!"candidate {candidate.digest} {candidate.edit.name}"
          return accepted (renderCandidate seq state.setName state.profile candidate skipped)
            { state with reduction, seen, outstanding := some (candidate.edit, candidate.digest) }
  | .observe _ _ decided =>
      let some (edit, digest) := state.outstanding
        | return .reply (renderRejected seq "no candidate is outstanding") state
      let fate := decided.fate
      let reduction := state.reduction.settle edit fate (some digest)
      let retained := if fate == .retained then digest else state.retained
      let settled : Settled := { edit, fate, candidate := some digest }
      effects.writeProgress s!"settled {digest} {edit.name} {fate.name}"
      return accepted (renderSettled seq state.setName state.profile settled retained)
        { state with
          reduction
          retained
          outstanding := none
          seen := state.seen ++ [digest]
          phase := if reduction.ended.isSome then .ended else state.phase }
  | .finish requested =>
      let reduction := match state.outstanding, requested with
        | some (_, digest), _ => state.reduction.stop s!"candidate {digest} was never settled"
        | none, some status => state.reduction.stop status
        | none, none => state.reduction
      let subject := (state.recovered.map (·.digest)).getD ""
      let result := reduction.result
      -- Only a finished sweep proposes: an incomplete one retained a candidate nobody may review
      -- as the reduction's answer.
      let proposal := match result with
        | .minimized | .irreducible => state.recovered.map (·.propose reduction.kept)
        | .incomplete _ => none
      return .finished (renderFinished seq state.setName state.profile result subject
        state.retained reduction state.outstanding proposal)

/-- The diagnostic prefix every failure outside a frame carries. -/
def diagnosticPrefix : String := "umpire-replay:"

def protocol (effects : Temporal.Tool.Bridge.Effects) (bound : List Bound) :
    Temporal.Tool.Bridge.Protocol State Frame :=
  { diagnosticPrefix
    maxLineBytes := some maxLineBytes
    parse := parseFrame
    seqOf := (·.seq)
    reject := rejection
    step := step effects bound }

/-- Serve frames until `finish`, or until `admit` answers `crossed`. -/
def serve (effects : Temporal.Tool.Bridge.Effects) (bound : List Bound) : IO UInt32 :=
  Temporal.Tool.Bridge.serve effects (protocol effects bound) {}

/-! ### The sets the bridge binds -/

open Temporal.Feature.Nexus in
/-- The negative control's set: its one Query, produced as its `case` block produces it. -/
def controlBinding : FunctionalBinding Control.nexusControl :=
  { set := Control.nexusCallerControl
    queries := [{
      name := "forgedCompletion"
      source := Control.forgedCompletion.source
      identity := Control.nexusCallerControlCases.forgedCompletion.identity
      production := {
        realization := Control.nexusCallerControlCases.realization
        evidence := Control.nexusCallerControlCases.forgedCompletion.evidence
        claims := Control.nexusCallerControlCases.forgedCompletion.claims
        catalog := Control.nexusCallerControlCases.forgedCompletion.catalog
        relations := Control.nexusCallerControlCases.forgedCompletion.relations } }] }

open Temporal.Feature.Nexus.Caller in
/-- One caller Query as its functional `case` block produces it. -/
private def callerQuery (name : String) (source : QuerySource nexusProtocol)
    (identity : Umpire.Case.Producer.Identity)
    (evidence : Umpire.Case.Producer.Vocabulary → List Umpire.Case.Producer.EvidenceMapping)
    (claims : List Umpire.Case.Producer.ClassClaim) (catalog : List (String × String))
    (relations : List Umpire.Case.Producer.FieldRelation) : QueryBinding nexusProtocol :=
  { name, source, identity
    production := { realization := nexusCallerCases.realization, evidence, claims, catalog, relations } }

open Temporal.Feature.Nexus.Caller in
/-- The caller Model's functional set: the Queries whose Scenario has a prefix to reduce. -/
def callerBinding : FunctionalBinding nexusProtocol :=
  { set := nexusCallerTests
    queries := [
      callerQuery "retry" retry.source nexusCallerCases.retry.identity
        nexusCallerCases.retry.evidence nexusCallerCases.retry.claims
        nexusCallerCases.retry.catalog nexusCallerCases.retry.relations,
      callerQuery "scheduleToStartTimeout" scheduleToStartTimeout.source
        nexusCallerCases.scheduleToStartTimeout.identity
        nexusCallerCases.scheduleToStartTimeout.evidence
        nexusCallerCases.scheduleToStartTimeout.claims
        nexusCallerCases.scheduleToStartTimeout.catalog
        nexusCallerCases.scheduleToStartTimeout.relations,
      callerQuery "startToCloseTimeout" startToCloseTimeout.source
        nexusCallerCases.startToCloseTimeout.identity
        nexusCallerCases.startToCloseTimeout.evidence
        nexusCallerCases.startToCloseTimeout.claims
        nexusCallerCases.startToCloseTimeout.catalog
        nexusCallerCases.startToCloseTimeout.relations] }

/-- Every set the bridge admits subjects of. -/
def boundSets : List Bound :=
  [Bound.functional controlBinding, Bound.functional callerBinding,
    Bound.exploratory Temporal.Tool.ExplorationBridge.nexusCallerBinding]

end Temporal.Tool.ReplayBridge
