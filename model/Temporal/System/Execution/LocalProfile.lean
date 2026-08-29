import Umpire.Artifact.Runtime

namespace Temporal.System.Execution

open _root_.Umpire

/-!
# Ephemeral local execution profile

This module owns the portable, inert description of the one local execution authority. It contains
no server address, namespace, credential, executable, callback, or runtime IO. The Temporal adapter
owns the in-memory test-server lifecycle; the profile only fixes the values checked before that
lifecycle may start. This first adapter therefore uses the in-memory Temporal test suite rather than
a developer-server lifecycle, and its evidence is limited to the closed sources configured by the
later System-owned participant binding.
-/

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

/-- Stable identity, version, and derived behavior of one execution profile. -/
structure ExecutionProfileReference where
  definitionId : DefinitionId
  version : Nat
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- The only four commands admitted by the first participant program. -/
inductive ParticipantCommandKind where
  | prepare
  | realize
  | observe
  | cleanup
  deriving BEq, DecidableEq, Ord, Repr

def ParticipantCommandKind.name : ParticipantCommandKind → String
  | .prepare => "prepare"
  | .realize => "realize"
  | .observe => "observe"
  | .cleanup => "cleanup"

/-- Closed cardinality, protocol, and command requirements for a local participant program. -/
structure ParticipantProgramRequirements where
  participantCount : Nat
  programCount : Nat
  protocolVersion : Nat
  commands : List ParticipantCommandKind
  deriving BEq, DecidableEq, Repr

/-- Unchecked profile data accepted only through `checkLocalProfile`. -/
structure LocalProfileDefinition where
  reference : ExecutionProfileReference
  requiredCapabilities : List DefinitionId
  phaseLimits : List PhaseLimit
  seed : Nat
  attempt : Nat
  participantProgramRequirements : ParticipantProgramRequirements
  /-- Authority-like fields are represented only on the unchecked side and must remain empty. -/
  authorityFieldDefinitionIds : List DefinitionId := []
  deriving BEq, DecidableEq, Repr

/-- Stable preflight categories produced without performing runtime IO. -/
inductive LocalProfileErrorKind where
  | profile
  | configuration
  | capability
  | budget
  | seed
  | attempt
  | participant
  | protocol
  | authority
  deriving BEq, DecidableEq, Ord, Repr

def LocalProfileErrorKind.name : LocalProfileErrorKind → String
  | .profile => "profile"
  | .configuration => "configuration"
  | .capability => "capability"
  | .budget => "budget"
  | .seed => "seed"
  | .attempt => "attempt"
  | .participant => "participant"
  | .protocol => "protocol"
  | .authority => "authority"

/-- One deterministic profile-check failure. -/
structure LocalProfileError where
  kind : LocalProfileErrorKind
  subject : DefinitionId
  deriving BEq, DecidableEq, Repr

private def profileError
    (kind : LocalProfileErrorKind)
    (subject : DefinitionId) : LocalProfileError := { kind, subject }

/-- Exact generic capabilities owned by the reusable runtime and Temporal lifecycle adapter. -/
def profileRequiredCapabilities : List DefinitionId := [
  DefinitionId.of "umpire.runtime.capability.complete-workflow-history-read",
  DefinitionId.of "umpire.runtime.capability.ephemeral-server-lifecycle",
  DefinitionId.of "umpire.runtime.capability.sdk-worker-lifecycle"
]

/-- The five fixed local phase budgets in execution order. -/
def localPhaseLimits : List PhaseLimit := [
  { phase := .preparation, durationMilliseconds := 30000, maxAttempts := 1,
    maxRecords := 128, maxBytes := 1048576 },
  { phase := .realization, durationMilliseconds := 30000, maxAttempts := 1,
    maxRecords := 128, maxBytes := 1048576 },
  { phase := .observation, durationMilliseconds := 30000, maxAttempts := 1,
    maxRecords := 3584, maxBytes := 12582912 },
  { phase := .isolation, durationMilliseconds := 15000, maxAttempts := 1,
    maxRecords := 128, maxBytes := 1048576 },
  { phase := .cleanup, durationMilliseconds := 15000, maxAttempts := 1,
    maxRecords := 128, maxBytes := 1048576 }
]

/-- Seed zero is the sole policy admitted by the first local profile. -/
def localSeed : Nat := 0

/-- Attempt one is the sole policy admitted by the first local profile. -/
def localAttempt : Nat := 1

/-- Exactly one protocol-v2 program supplies the four closed participant commands. -/
def localParticipantProgramRequirements : ParticipantProgramRequirements := {
  participantCount := 1
  programCount := 1
  protocolVersion := 2
  commands := [.prepare, .realize, .observe, .cleanup]
}

private def phaseLimitJson (limit : PhaseLimit) : String :=
  "{\"phase\":" ++ quote limit.phase.name ++
    ",\"durationMilliseconds\":" ++ toString limit.durationMilliseconds ++
    ",\"maxAttempts\":" ++ toString limit.maxAttempts ++
    ",\"maxRecords\":" ++ toString limit.maxRecords ++
    ",\"maxBytes\":" ++ toString limit.maxBytes ++ "}"

private def participantProgramRequirementsJson
    (requirements : ParticipantProgramRequirements) : String :=
  "{\"participantCount\":" ++ toString requirements.participantCount ++
    ",\"programCount\":" ++ toString requirements.programCount ++
    ",\"protocolVersion\":" ++ toString requirements.protocolVersion ++
    ",\"commands\":" ++ array (requirements.commands.map (quote ∘ ParticipantCommandKind.name)) ++
    "}"

/--
Return the non-self-referential Generated View used to derive the profile Behavior Fingerprint.

The view deliberately excludes the Behavior Fingerprint itself and unchecked authority-field input.
-/
def localProfileGeneratedViewBytes (definition : LocalProfileDefinition) : String :=
  Json.prettyBytes <|
    "{\"definitionId\":" ++ quote definition.reference.definitionId.value ++
      ",\"version\":" ++ toString definition.reference.version ++
      ",\"requiredCapabilityDefinitionIds\":" ++
        array (definition.requiredCapabilities.map (quote ∘ DefinitionId.value)) ++
      ",\"phaseLimits\":" ++ array (definition.phaseLimits.map phaseLimitJson) ++
      ",\"seed\":" ++ toString definition.seed ++
      ",\"attempt\":" ++ toString definition.attempt ++
      ",\"participantProgramRequirements\":" ++
        participantProgramRequirementsJson definition.participantProgramRequirements ++ "}"

/-- Recompute the profile fingerprint solely from its exact pretty Generated View bytes. -/
def LocalProfileDefinition.expectedBehaviorFingerprint
    (definition : LocalProfileDefinition) : BehaviorFingerprint :=
  behaviorFingerprintOf (localProfileGeneratedViewBytes definition)

private def canonicalLocalProfileDraft : LocalProfileDefinition := {
  reference := {
    definitionId := DefinitionId.of "temporal.runtime-profile.ephemeral-local"
    version := 2
    behaviorFingerprint := behaviorFingerprintOf "unsealed-local-profile"
  }
  requiredCapabilities := profileRequiredCapabilities
  phaseLimits := localPhaseLimits
  seed := localSeed
  attempt := localAttempt
  participantProgramRequirements := localParticipantProgramRequirements
}

/-- Canonical unchecked fixture used by composition and mutation tests. -/
def canonicalLocalProfileDefinition : LocalProfileDefinition := {
  canonicalLocalProfileDraft with
  reference := {
    canonicalLocalProfileDraft.reference with
    behaviorFingerprint := canonicalLocalProfileDraft.expectedBehaviorFingerprint
  }
}

private def phaseTotal
    (projection : PhaseLimit → Nat)
    (limits : List PhaseLimit) : Nat :=
  limits.foldl (fun total limit => total + projection limit) 0

private def budgetsHaveExactTotals (limits : List PhaseLimit) : Bool :=
  phaseTotal PhaseLimit.durationMilliseconds limits == 120000 &&
    phaseTotal PhaseLimit.maxAttempts limits == 5 &&
    phaseTotal PhaseLimit.maxRecords limits == 4096 &&
    phaseTotal PhaseLimit.maxBytes limits == 16777216

private structure CheckedLocalProfilePayload where
  definition : LocalProfileDefinition

/-- A profile whose complete portable meaning has passed the exact local checker. -/
structure CheckedLocalProfile where
  private mk ::
  private payload : CheckedLocalProfilePayload

/-- Check one profile definition before a runtime adapter may perform IO. -/
def checkLocalProfile
    (definition : LocalProfileDefinition) : Except LocalProfileError CheckedLocalProfile := do
  let subject := definition.reference.definitionId
  if definition.authorityFieldDefinitionIds != [] then
    throw (profileError .authority subject)
  if definition.reference.definitionId != canonicalLocalProfileDraft.reference.definitionId ||
      definition.reference.version != canonicalLocalProfileDraft.reference.version then
    throw (profileError .profile subject)
  if definition.requiredCapabilities != profileRequiredCapabilities then
    throw (profileError .capability subject)
  if definition.phaseLimits != localPhaseLimits ||
      !budgetsHaveExactTotals definition.phaseLimits then
    throw (profileError .budget subject)
  if definition.seed != localSeed then
    throw (profileError .seed subject)
  if definition.attempt != localAttempt then
    throw (profileError .attempt subject)
  if definition.participantProgramRequirements.participantCount != 1 ||
      definition.participantProgramRequirements.programCount != 1 ||
      definition.participantProgramRequirements.commands != [.prepare, .realize, .observe, .cleanup]
    then
      throw (profileError .participant subject)
  if definition.participantProgramRequirements.protocolVersion != 2 then
    throw (profileError .protocol subject)
  if definition.reference.behaviorFingerprint != definition.expectedBehaviorFingerprint then
    throw (profileError .profile subject)
  pure (.mk { definition })

private theorem canonicalLocalProfile_isSome :
    (checkLocalProfile canonicalLocalProfileDefinition).toOption.isSome = true := by
  native_decide

/-- The sole checked local profile exposed to runtime/configuration composition. -/
def ephemeralLocalProfile : CheckedLocalProfile :=
  (checkLocalProfile canonicalLocalProfileDefinition).toOption.get canonicalLocalProfile_isSome

/-- Return the exact semantic reference carried by a checked profile. -/
def CheckedLocalProfile.reference (profile : CheckedLocalProfile) : ExecutionProfileReference :=
  profile.payload.definition.reference

/-- Return the exact canonical capability list carried by a checked profile. -/
def CheckedLocalProfile.requiredCapabilities
    (profile : CheckedLocalProfile) : List DefinitionId :=
  profile.payload.definition.requiredCapabilities

/-- Return the exact five canonical phase limits carried by a checked profile. -/
def CheckedLocalProfile.phaseLimits (profile : CheckedLocalProfile) : List PhaseLimit :=
  profile.payload.definition.phaseLimits

/-- Return the only accepted seed. -/
def CheckedLocalProfile.seed (profile : CheckedLocalProfile) : Nat :=
  profile.payload.definition.seed

/-- Return the only accepted attempt. -/
def CheckedLocalProfile.attempt (profile : CheckedLocalProfile) : Nat :=
  profile.payload.definition.attempt

/-- Return the closed participant/program requirements. -/
def CheckedLocalProfile.participantProgramRequirements
    (profile : CheckedLocalProfile) : ParticipantProgramRequirements :=
  profile.payload.definition.participantProgramRequirements

/-- Return the exact pretty Generated View bytes that identify a checked profile. -/
def CheckedLocalProfile.generatedViewBytes (profile : CheckedLocalProfile) : String :=
  localProfileGeneratedViewBytes profile.payload.definition

/-- Project a checked local profile onto fn-18's inert RuntimeConfiguration authority transport. -/
def CheckedLocalProfile.authorityProfile (profile : CheckedLocalProfile) : AuthorityProfile := {
  definitionId := profile.reference.definitionId
  version := profile.reference.version
  behaviorFingerprint := profile.reference.behaviorFingerprint
  requiredCapabilityDefinitionIds := profile.requiredCapabilities
}

/--
Check the fn-18 transport and exact profile-owned portion of one RuntimeConfiguration.

Nexus-specific observation and participant program closure remains the responsibility of its later
System-owned composition module.
-/
def CheckedLocalProfile.checkRuntimeConfiguration
    (profile : CheckedLocalProfile)
    (configuration : RuntimeConfiguration) : Except LocalProfileError Unit := do
  let subject := configuration.configurationDefinitionId
  if !configuration.isValidTransport then
    throw (profileError .configuration subject)
  if configuration.authorityProfile.definitionId != profile.reference.definitionId ||
      configuration.authorityProfile.version != profile.reference.version ||
      configuration.authorityProfile.behaviorFingerprint != profile.reference.behaviorFingerprint then
    throw (profileError .profile subject)
  if configuration.authorityProfile.requiredCapabilityDefinitionIds != profile.requiredCapabilities then
    throw (profileError .capability subject)
  if configuration.phaseLimits != profile.phaseLimits then
    throw (profileError .budget subject)
  pure ()

end Temporal.System.Execution
