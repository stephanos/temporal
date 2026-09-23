import Umpire.KnownGap
import Umpire.Json
import Umpire.Fingerprint

/-!
# Evaluation Profiles

An Evaluation Profile is the policy an offline Claim Assessment applies to one closed Run of one
Case: the claim it would make, the trust basis it asserts, the Known Gap kinds that keep it from
accepting, and an ordered reason table. Each reason names one condition from a closed set and the
decision it forces, `rejected` or `incomplete`; the table's order is the precedence the assessment
reports its reasons in. A subject for which no reason holds is accepted.

A `Declaration` is the authored form. `Profile.declare` checks it and is the only way to a
`Profile`, so every Profile has a valid name, a claim, a trust basis, a non-empty table of
uniquely named reasons over distinct conditions, and a Known Gap policy that is not contradictory:
blocking kinds exactly when a `known-gap-blocking` reason exists.

A Profile carries no catalog, endpoint, credential, path, Limit, Driver or execution authority, and
nothing Temporal: it is data a Go assessor reads. `Profile.render` is its canonical JSON (compact,
fields in a fixed order, one trailing newline) and `Profile.identity` is `sha256:` followed by the
hex SHA-256 of those bytes, so the same Profile always has the same identity and any other Profile
is an independent assessment.
-/

namespace Umpire.Evaluation

/-- The closed set of status-specific conditions a reason can name. Each reads one recorded value of
the subject; none re-derives a Verdict. -/
inductive Condition where
  /-- The recorded Verdict is violated. -/
  | verdictViolated
  /-- The recorded Verdict is inconclusive. -/
  | verdictInconclusive
  /-- The Run was stopped by its Monitor. -/
  | dispositionStopped
  /-- The Run did not complete. -/
  | dispositionIncomplete
  /-- The Run's cleanup did not succeed. -/
  | cleanupUnclosed
  /-- The Case declares a Known Gap of a kind the Profile blocks on. -/
  | knownGapBlocking
  /-- A rule the Verdict names at a terminal state has no supporting event. -/
  | unsupportedRule
  deriving BEq, DecidableEq, Repr

/-- The condition's stable spelling in a rendered Profile. -/
def Condition.name : Condition → String
  | .verdictViolated => "verdict-violated"
  | .verdictInconclusive => "verdict-inconclusive"
  | .dispositionStopped => "disposition-stopped"
  | .dispositionIncomplete => "disposition-incomplete"
  | .cleanupUnclosed => "cleanup-unclosed"
  | .knownGapBlocking => "known-gap-blocking"
  | .unsupportedRule => "unsupported-rule"

/-- The decision a reason forces. Acceptance is never forced: it is what remains when no reason
holds. -/
inductive Forced where
  | rejected
  | incomplete
  deriving BEq, DecidableEq, Repr

/-- The forced decision's stable spelling in a rendered Profile. -/
def Forced.name : Forced → String
  | .rejected => "rejected"
  | .incomplete => "incomplete"

/-- One row of a reason table: its name, the condition it tests and the decision it forces. -/
structure Reason where
  name : String
  condition : Condition
  decision : Forced
  deriving BEq, DecidableEq, Repr

/-- An authored Evaluation Profile, not yet checked. -/
structure Declaration where
  /-- The exact name a Profile is selected by: lowercase letters, digits and inner hyphens. -/
  name : String
  /-- The claim an accepted assessment makes. -/
  claim : String
  /-- The trust basis the Profile asserts; asserted, never checked against the subject. -/
  trust : String
  /-- The Known Gap kinds that keep a subject from acceptance. -/
  blockingGaps : List KnownGapKind
  /-- The reason table, in precedence order. -/
  reasons : List Reason
  deriving BEq, Repr

/-- Why a declaration is not a Profile. Each names what it found. -/
inductive ProfileError where
  | invalidName (name : String)
  | emptyClaim
  | emptyTrust
  | emptyTable
  | emptyReasonName
  | duplicateReason (name : String)
  | repeatedCondition (condition : Condition)
  | duplicateBlockingKind (kind : KnownGapKind)
  /-- A `known-gap-blocking` reason with no blocking kind: it could never hold. -/
  | blockingWithoutKinds
  /-- Blocking kinds with no `known-gap-blocking` reason: they could never block. -/
  | kindsWithoutBlocking
  deriving BEq, DecidableEq, Repr

/-- The error as one line naming what was found. -/
def ProfileError.render : ProfileError → String
  | .invalidName name =>
      s!"Profile name '{name}' is not lowercase letters, digits and inner hyphens"
  | .emptyClaim => "the Profile states no claim"
  | .emptyTrust => "the Profile states no trust basis"
  | .emptyTable => "the Profile's reason table is empty"
  | .emptyReasonName => "a reason has an empty name"
  | .duplicateReason name => s!"reason '{name}' is declared twice"
  | .repeatedCondition condition => s!"condition '{condition.name}' is named by two reasons"
  | .duplicateBlockingKind kind => s!"Known Gap kind '{kind.name}' is blocking twice"
  | .blockingWithoutKinds => "a 'known-gap-blocking' reason is declared with no blocking kind"
  | .kindsWithoutBlocking => "blocking Known Gap kinds are declared with no 'known-gap-blocking' reason"

/-- A checked Evaluation Profile. Only `Profile.declare` constructs one. -/
structure Profile where
  private mk ::
  private declaration : Declaration
  deriving BEq, Repr

private def validName (name : String) : Bool :=
  let characters := name.toList
  !characters.isEmpty && characters.head? != some '-' && characters.getLast? != some '-' &&
    characters.all fun character => character.isLower || character.isDigit || character == '-'

private def firstDuplicate [BEq α] : List α → Option α
  | [] => none
  | item :: rest => if rest.contains item then some item else firstDuplicate rest

/-- Check a declaration. The first problem found, in the order the fields are declared, is the
error. -/
def Profile.declare (declaration : Declaration) : Except ProfileError Profile := do
  unless validName declaration.name do throw (.invalidName declaration.name)
  if declaration.claim.isEmpty then throw .emptyClaim
  if declaration.trust.isEmpty then throw .emptyTrust
  if declaration.reasons.isEmpty then throw .emptyTable
  if declaration.reasons.any (·.name.isEmpty) then throw .emptyReasonName
  if let some name := firstDuplicate (declaration.reasons.map (·.name)) then
    throw (.duplicateReason name)
  if let some condition := firstDuplicate (declaration.reasons.map (·.condition)) then
    throw (.repeatedCondition condition)
  if let some kind := firstDuplicate declaration.blockingGaps then
    throw (.duplicateBlockingKind kind)
  let blocking := declaration.reasons.any (·.condition == .knownGapBlocking)
  if blocking && declaration.blockingGaps.isEmpty then throw .blockingWithoutKinds
  if !blocking && !declaration.blockingGaps.isEmpty then throw .kindsWithoutBlocking
  pure ⟨declaration⟩

namespace Profile

/-- The exact name the Profile is selected by. -/
def name (profile : Profile) : String := profile.declaration.name

/-- The claim an accepted assessment makes. -/
def claim (profile : Profile) : String := profile.declaration.claim

/-- The asserted trust basis. -/
def trust (profile : Profile) : String := profile.declaration.trust

/-- The reason table, in precedence order. -/
def reasons (profile : Profile) : List Reason := profile.declaration.reasons

private def kindRank : KnownGapKind → Nat
  | .capability => 0
  | .input => 1
  | .interpretation => 2
  | .claim => 3

/-- The blocking Known Gap kinds in the fixed kind order, whatever order they were declared in. -/
def blockingGaps (profile : Profile) : List KnownGapKind :=
  profile.declaration.blockingGaps.mergeSort fun left right => kindRank left ≤ kindRank right

/-- The rendered Profile format version; a reader rejects any other. -/
def formatVersion : Nat := 1

/-- The Profile as ordered canonical JSON. -/
def json (profile : Profile) : CanonicalJson :=
  .object [
    ("version", .natural formatVersion),
    ("name", .string profile.name),
    ("claim", .string profile.claim),
    ("trust", .string profile.trust),
    ("blockingKnownGaps", .array (profile.blockingGaps.map (.string ·.name))),
    ("reasons", .array (profile.reasons.map fun reason => .object [
      ("name", .string reason.name),
      ("condition", .string reason.condition.name),
      ("decision", .string reason.decision.name)
    ]))
  ]

/-- The Profile's canonical bytes: compact JSON and one trailing newline. -/
def render (profile : Profile) : String := profile.json.compact ++ "\n"

/-- `sha256:` and the hex SHA-256 of the canonical bytes. -/
def identity (profile : Profile) : String :=
  "sha256:" ++ Fingerprint.sha256Hex profile.render

end Profile

end Umpire.Evaluation
