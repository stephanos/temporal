import Umpire.Artifact.Result

namespace Umpire

/-! Exact closed v2 Artifact sets and their deterministic manifest. -/

private def quoteSet (value : String) : String := Lean.Json.compress (.str value)

private def setArray (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

/-- One exact member row in the deterministic Artifact set manifest. -/
structure ArtifactSetManifestMember where
  path : String
  formatVersion : String
  artifactChecksum : ArtifactChecksum
  behaviorFingerprint : BehaviorFingerprint
  provenanceChecksum : ArtifactChecksum
  deriving BEq, DecidableEq, Repr

/-- The complete deterministic manifest for one admitted Artifact set. -/
structure ArtifactSetManifest where
  formatVersion : String
  artifactSetIdentity : String
  members : List ArtifactSetManifestMember
  artifactSetChecksum : ArtifactChecksum
  deriving BEq, DecidableEq, Repr

/-- The retained documents that may form one exact executable, execution, or evaluation closure. -/
structure ArtifactSet where
  experiment : ExperimentSpec
  runtimeConfiguration : RuntimeConfiguration
  experimentRun : Option ExperimentRun := none
  rawEvidence : Option RawEvidence := none
  evidence : Option EvidenceArtifact := none
  result : Option ResultArtifact := none
  deriving BEq, DecidableEq, Repr

private def resultArtifactBinding (result : ResultArtifact) : ArtifactBinding := {
  formatVersion := result.formatVersion
  artifactChecksum := result.artifactChecksum
  behaviorFingerprint := result.behaviorFingerprint
  provenanceChecksum := result.provenanceChecksum
}

private def manifestMember
    (path : String)
    (binding : ArtifactBinding) : ArtifactSetManifestMember := {
  path
  formatVersion := binding.formatVersion
  artifactChecksum := binding.artifactChecksum
  behaviorFingerprint := binding.behaviorFingerprint
  provenanceChecksum := binding.provenanceChecksum
}

private def setDefinitionIdLe (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

private def setModelValueLe (left right : ModelValue) : Bool :=
  decide (left.definitionId.value < right.definitionId.value) ||
    (left.definitionId == right.definitionId && decide (left.value ≤ right.value))

private def setBindingLe (left right : RoleBinding) : Bool :=
  decide (left.role.value < right.role.value) ||
    (left.role == right.role && setModelValueLe left.value right.value)

private def setPropertyLe (left right : PortableProperty) : Bool :=
  decide (left.definitionId.value ≤ right.definitionId.value)

private def setDefinitionIdsValid (ids : List DefinitionId) : Bool :=
  ids == (ids.mergeSort setDefinitionIdLe).eraseDups && ids.all DefinitionId.isNamespaced

private def setModelValueValid (value : ModelValue) : Bool :=
  value.definitionId.isNamespaced

private def setOperandValid : SetupOperand → Bool
  | .role definitionId => definitionId.isNamespaced
  | .value value => setModelValueValid value

private def setRoleKindValid : DefinitionKind → Bool
  | .state | .action | .outcome | .observation | .relation | .capability | .provider | .law |
      .connector | .target | .kernel => true
  | _ => false

private def drivePlanCollectionsValid (plan : DrivePlan) : Bool :=
  plan.bindings == plan.bindings.mergeSort setBindingLe &&
    plan.bindings.all fun binding =>
      binding.role.isNamespaced && setModelValueValid binding.value &&
    plan.symbolicRoles.all fun role =>
      role.id.isNamespaced && setRoleKindValid role.valueKind &&
    plan.modelPreconditions.all fun precondition =>
      precondition.id.isNamespaced && setOperandValid precondition.left &&
        setOperandValid precondition.right &&
    setModelValueValid plan.initialState &&
    [plan.requestedActions, plan.modelOutcomes, plan.resultingStates, plan.selectedChoices,
      plan.selectedVariants, plan.requestedFaults].all fun values =>
        values.all setModelValueValid &&
    plan.linearExtension.all fun occurrence =>
      occurrence.definitionId.isNamespaced && occurrence.actionDefinitionId.isNamespaced &&
        occurrence.authoredDefinitionId.all DefinitionId.isNamespaced &&
    plan.checkpoints.all fun checkpoint => checkpoint.observations.all setModelValueValid

/-- Check the complete canonical DrivePlan transport retained inside an Artifact set. -/
def DrivePlan.isValidTransport (plan : DrivePlan) : Bool :=
  plan.formatVersion == "umpire-drive-plan/v2" && plan.queryDefinitionId.isNamespaced &&
    plan.behaviorDefinitionId.isNamespaced && plan.targetDefinitionId.isNamespaced &&
    plan.kernelDefinitionId.isNamespaced && drivePlanCollectionsValid plan &&
    setDefinitionIdsValid plan.capabilityRequirementDefinitionIds &&
    plan.provenance.isValidTransport && (validateKnownGaps plan.knownGaps).isOk &&
    plan.hasValidArtifactChecksum

private def experimentPropertiesValid (properties : List PortableProperty) : Bool :=
  properties != [] && properties == properties.mergeSort setPropertyLe &&
    properties.all fun property =>
      property.definitionId.isNamespaced &&
        setDefinitionIdsValid property.requirementDefinitionIds

/-- Check the complete canonical ExperimentSpec transport retained inside an Artifact set. -/
def ExperimentSpec.isValidTransport (experiment : ExperimentSpec) : Bool :=
  experiment.formatVersion == "umpire-experiment/v2" && experiment.plan.isValidTransport &&
    experiment.queryBehaviorFingerprint == experiment.plan.queryBehaviorFingerprint &&
    experimentPropertiesValid experiment.properties &&
    setDefinitionIdsValid experiment.observationRequirementDefinitionIds &&
    experiment.provenance.isValidTransport && experiment.hasValidArtifactChecksum

/-- Check one exact 2-, 4-, or 6-member closure without executing or interpreting any member. -/
def ArtifactSet.isValidClosure (set : ArtifactSet) : Bool :=
  set.experiment.isValidTransport && set.runtimeConfiguration.isValidTransport &&
    set.runtimeConfiguration.closesExperiment set.experiment &&
    match set.experimentRun, set.rawEvidence, set.evidence, set.result with
    | none, none, none, none => true
    | some run, some rawEvidence, none, none =>
        run.isValidTransport && rawEvidence.isValidTransport &&
          rawEvidence.closes set.experiment set.runtimeConfiguration run
    | some run, some rawEvidence, some evidence, some result =>
        run.isValidTransport && rawEvidence.isValidTransport && evidence.isValidTransport &&
          result.isValidTransport &&
          result.closes set.experiment set.runtimeConfiguration run rawEvidence evidence &&
          result.implementationLink.sourceTarget.definitionId ==
            set.experiment.plan.targetDefinitionId &&
          result.implementationLink.sourceTarget.behaviorFingerprint ==
            set.experiment.plan.targetBehaviorFingerprint
    | _, _, _, _ => false

private def ArtifactSet.manifestMembers? (set : ArtifactSet) : Option (List ArtifactSetManifestMember) :=
  if !set.isValidClosure then none
  else
    let initial := [
      manifestMember "artifacts/experiment.json" set.experiment.artifactBinding,
      manifestMember "artifacts/runtime-configuration.json"
        set.runtimeConfiguration.artifactBinding
    ]
    match set.experimentRun, set.rawEvidence, set.evidence, set.result with
    | none, none, none, none => some initial
    | some run, some rawEvidence, none, none =>
        some (initial ++ [
          manifestMember "artifacts/experiment-run.json" run.artifactBinding,
          manifestMember "artifacts/raw-evidence.json" rawEvidence.artifactBinding
        ])
    | some run, some rawEvidence, some evidence, some result =>
        some (initial ++ [
          manifestMember "artifacts/experiment-run.json" run.artifactBinding,
          manifestMember "artifacts/raw-evidence.json" rawEvidence.artifactBinding,
          manifestMember "artifacts/evidence.json" evidence.artifactBinding,
          manifestMember "artifacts/result.json" (resultArtifactBinding result)
        ])
    | _, _, _, _ => none

private def ArtifactSetManifestMember.canonicalJson
    (member : ArtifactSetManifestMember) : String :=
  "{\"path\":" ++ quoteSet member.path ++
    ",\"formatVersion\":" ++ quoteSet member.formatVersion ++
    ",\"artifactChecksum\":" ++ quoteSet member.artifactChecksum.render ++
    ",\"behaviorFingerprint\":" ++ quoteSet member.behaviorFingerprint.render ++
    ",\"provenanceChecksum\":" ++ quoteSet member.provenanceChecksum.render ++ "}"

private def manifestMembersJson (members : List ArtifactSetManifestMember) : String :=
  setArray (members.map ArtifactSetManifestMember.canonicalJson)

private def artifactSetIdentity (members : List ArtifactSetManifestMember) : String :=
  "umpire.artifact-set." ++ Fingerprint.sha256Hex
    ("umpire.artifact-set-identity/v2\n" ++ Json.prettyBytes (manifestMembersJson members))

private def artifactSetChecksumOf (canonicalContent : String) : ArtifactChecksum :=
  (ArtifactChecksum.parse? ("sha256:" ++ Fingerprint.sha256Hex
    ("umpire.artifact-set/v2\n" ++ canonicalContent))).getD (drivePlanChecksumOf "")

private def artifactSetManifestContentJson (manifest : ArtifactSetManifest) : String :=
  "{\"formatVersion\":" ++ quoteSet manifest.formatVersion ++
    ",\"artifactSetIdentity\":" ++ quoteSet manifest.artifactSetIdentity ++
    ",\"members\":" ++ manifestMembersJson manifest.members ++ "}"

def ArtifactSetManifest.expectedChecksum (manifest : ArtifactSetManifest) : ArtifactChecksum :=
  artifactSetChecksumOf (Json.prettyBytes (artifactSetManifestContentJson manifest))

private def sealArtifactSetManifest
    (members : List ArtifactSetManifestMember) : ArtifactSetManifest :=
  let withoutChecksum : ArtifactSetManifest := {
    formatVersion := "umpire-artifact-set/v2"
    artifactSetIdentity := artifactSetIdentity members
    members
    artifactSetChecksum := drivePlanChecksumOf ""
  }
  { withoutChecksum with artifactSetChecksum := withoutChecksum.expectedChecksum }

/-- Derive the sole valid manifest when the retained documents form an exact closed set. -/
def ArtifactSet.manifest? (set : ArtifactSet) : Option ArtifactSetManifest :=
  set.manifestMembers?.map sealArtifactSetManifest

/-- Check that a supplied manifest is exactly the one derived from a closed retained set. -/
def ArtifactSetManifest.isValidFor
    (manifest : ArtifactSetManifest)
    (set : ArtifactSet) : Bool :=
  set.manifest? == some manifest

/-- Encode an Artifact set manifest in its sole deterministic pretty representation. -/
def canonicalArtifactSetManifestJson (manifest : ArtifactSetManifest) : String :=
  let content := artifactSetManifestContentJson manifest
  Json.pretty ((content.dropEnd 1).toString ++
    ",\"artifactSetChecksum\":" ++ quoteSet manifest.artifactSetChecksum.render ++ "}")

def canonicalArtifactSetManifestBytes (manifest : ArtifactSetManifest) : String :=
  canonicalArtifactSetManifestJson manifest ++ "\n"

/-- Return the raw SHA-256 of the complete canonical manifest bytes. -/
def ArtifactSetManifest.manifestSha256 (manifest : ArtifactSetManifest) : ArtifactChecksum :=
  (ArtifactChecksum.parse? ("sha256:" ++ Fingerprint.sha256Hex
    (canonicalArtifactSetManifestBytes manifest))).getD (drivePlanChecksumOf "")

end Umpire
