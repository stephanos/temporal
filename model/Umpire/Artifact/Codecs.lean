import Umpire.Artifact.Types
import Umpire.Json

namespace Umpire

/-! Canonical v2 bytes and Artifact Checksum derivation for the retained planning Artifacts. -/

private def valueLe (left right : ModelValue) : Bool :=
  decide (left.definitionId.value < right.definitionId.value) ||
    (left.definitionId == right.definitionId && decide (left.value ≤ right.value))

private def bindingLe (left right : RoleBinding) : Bool :=
  decide (left.role.value < right.role.value) ||
    (left.role == right.role && valueLe left.value right.value)

private def sourceLe (left right : SourceLocation) : Bool :=
  decide (left.path < right.path) ||
    (left.path == right.path && decide (left.line < right.line)) ||
    (left.path == right.path && left.line == right.line && decide (left.column < right.column)) ||
    (left.path == right.path && left.line == right.line && left.column == right.column &&
      decide (left.provenance ≤ right.provenance))

private def propertyLe (left right : PortableProperty) : Bool :=
  decide (left.definitionId.value ≤ right.definitionId.value)

private def valueJson (value : ModelValue) : CanonicalJson :=
  .object [
    ("definitionId", .string value.definitionId.value),
    ("value", .string value.value)
  ]

private def roleJson (role : ResourceRole) : CanonicalJson :=
  .object [
    ("definitionId", .string role.id.value),
    ("valueKind", .string role.valueKind.name)
  ]

private def bindingJson (binding : RoleBinding) : CanonicalJson :=
  .object [
    ("roleDefinitionId", .string binding.role.value),
    ("value", valueJson binding.value)
  ]

private def operandJson : SetupOperand → CanonicalJson
  | .role identity =>
      .object [
        ("kind", .string "role"),
        ("definitionId", .string identity.value)
      ]
  | .value value =>
      .object [
        ("kind", .string "value"),
        ("value", valueJson value)
      ]

private def preconditionJson (constraint : SetupConstraint) : CanonicalJson :=
  .object [
    ("definitionId", .string constraint.id.value),
    ("relation", .string constraint.relation.name),
    ("left", operandJson constraint.left),
    ("right", operandJson constraint.right)
  ]

private def limitJson (bound : Limit) : CanonicalJson :=
  .object [
    ("value", .natural bound.value),
    ("unit", .string bound.unit.name)
  ]

private def limitsJson (limits : QueryLimits) : CanonicalJson :=
  .object [
    ("behavior", .object [
      ("transitions", limitJson limits.behavior.transitions),
      ("selectedActions", limitJson limits.behavior.selectedActions)
    ]),
    ("search", .object [
      ("value", .natural limits.search.value),
      ("unit", .string limits.search.unit.name)
    ])
  ]

private def exploredJson (explored : ExploredCounts) : CanonicalJson :=
  .object [
    ("setups", .natural explored.setups),
    ("traces", .natural explored.traces),
    ("transitions", .natural explored.transitions),
    ("propertyEvaluations", .natural explored.propertyEvaluations)
  ]

private def checkpointJson (checkpoint : ObservationCheckpoint) : CanonicalJson :=
  .object [
    ("transition", .natural checkpoint.transition),
    ("observations", .array (checkpoint.observations.map valueJson))
  ]

private def plannedOccurrenceJson (occurrence : PlannedOccurrence) : CanonicalJson :=
  .object [
    ("definitionId", .string occurrence.definitionId.value),
    ("actionDefinitionId", .string occurrence.actionDefinitionId.value),
    ("position", .natural occurrence.position),
    ("authoredDefinitionId", CanonicalJson.ofOption
      (fun authoredDefinitionId => .string authoredDefinitionId.value)
      occurrence.authoredDefinitionId)
  ]

private def sourceJson (source : SourceLocation) : CanonicalJson :=
  .object [
    ("path", .string source.path),
    ("line", .natural source.line),
    ("column", .natural source.column),
    ("provenance", .string source.provenance)
  ]

private def artifactProvenanceJson (provenance : ArtifactProvenance) : CanonicalJson :=
  .object [
    ("sourceDefinitionIds", .array (DefinitionId.canonicalSet provenance.sourceDefinitionIds |>.map
      fun definitionId => .string definitionId.value)),
    ("sourceLocations", .array
      (provenance.sourceLocations.mergeSort sourceLe |>.eraseDups |>.map sourceJson))
  ]

/-- Encode one ArtifactProvenance with the canonical v2 field and collection order. -/
def ArtifactProvenance.canonicalJson (provenance : ArtifactProvenance) : String :=
  (artifactProvenanceJson provenance).compact

private def propertyJson (property : PortableProperty) : CanonicalJson :=
  .object [
    ("definitionId", .string property.definitionId.value),
    ("behaviorFingerprint", .string property.behaviorFingerprint.render),
    ("requirementDefinitionIds", .array
      (DefinitionId.canonicalSet property.requirementDefinitionIds |>.map fun definitionId =>
        .string definitionId.value))
  ]

private def drivePlanContentFields (plan : DrivePlan) : List (String × CanonicalJson) := [
  ("formatVersion", .string plan.formatVersion),
  ("queryDefinitionId", .string plan.queryDefinitionId.value),
  ("queryBehaviorFingerprint", .string plan.queryBehaviorFingerprint.render),
  ("behaviorDefinitionId", .string plan.behaviorDefinitionId.value),
  ("behaviorFingerprint", .string plan.behaviorFingerprint.render),
  ("targetDefinitionId", .string plan.targetDefinitionId.value),
  ("targetBehaviorFingerprint", .string plan.targetBehaviorFingerprint.render),
  ("kernelDefinitionId", .string plan.kernelDefinitionId.value),
  ("kernelBehaviorFingerprint", .string plan.kernelBehaviorFingerprint.render),
  ("bindings", .array (plan.bindings.mergeSort bindingLe |>.map bindingJson)),
  ("symbolicRoles", .array (plan.symbolicRoles.map roleJson)),
  ("modelPreconditions", .array (plan.modelPreconditions.map preconditionJson)),
  ("initialState", valueJson plan.initialState),
  ("requestedActions", .array (plan.requestedActions.map valueJson)),
  ("modelOutcomes", .array (plan.modelOutcomes.map valueJson)),
  ("resultingStates", .array (plan.resultingStates.map valueJson)),
  ("linearExtension", .array (plan.linearExtension.map plannedOccurrenceJson)),
  ("selectedChoices", .array (plan.selectedChoices.map valueJson)),
  ("selectedVariants", .array (plan.selectedVariants.map valueJson)),
  ("requestedFaults", .array (plan.requestedFaults.map valueJson)),
  ("capabilityRequirementDefinitionIds", .array
    (DefinitionId.canonicalSet plan.capabilityRequirementDefinitionIds |>.map fun definitionId =>
      .string definitionId.value)),
  ("expandedLimits", limitsJson plan.expandedLimits),
  ("checkpoints", .array (plan.checkpoints.map checkpointJson)),
  ("selectionReason", .string plan.selectionReason.name),
  ("explored", exploredJson plan.explored),
  ("knownGaps", .array (plan.knownGaps.toList.map KnownGap.canonicalJsonValue)),
  ("provenance", artifactProvenanceJson plan.provenance)
]

private def drivePlanContentJson (plan : DrivePlan) : CanonicalJson :=
  .object (drivePlanContentFields plan)

def DrivePlan.expectedArtifactChecksum (plan : DrivePlan) : ArtifactChecksum :=
  drivePlanChecksumOf (drivePlanContentJson plan).prettyBytes

def DrivePlan.hasValidArtifactChecksum (plan : DrivePlan) : Bool :=
  plan.artifactChecksum == plan.expectedArtifactChecksum

private def sealedDrivePlanJson (plan : DrivePlan) : CanonicalJson :=
  .object (drivePlanContentFields plan ++ [
    ("artifactChecksum", .string plan.artifactChecksum.render)
  ])

def canonicalDrivePlanJson (plan : DrivePlan) : String :=
  (sealedDrivePlanJson plan).pretty

def canonicalDrivePlanBytes (plan : DrivePlan) : String :=
  (sealedDrivePlanJson plan).prettyBytes

private def experimentSpecContentFields (spec : ExperimentSpec) : List (String × CanonicalJson) := [
  ("formatVersion", .string spec.formatVersion),
  ("queryBehaviorFingerprint", .string spec.queryBehaviorFingerprint.render),
  ("plan", sealedDrivePlanJson spec.plan),
  ("properties", .array (spec.properties.mergeSort propertyLe |>.map propertyJson)),
  ("observationRequirementDefinitionIds", .array
    (DefinitionId.canonicalSet spec.observationRequirementDefinitionIds |>.map fun definitionId =>
      .string definitionId.value)),
  ("provenance", artifactProvenanceJson spec.provenance)
]

private def experimentSpecContentJson (spec : ExperimentSpec) : CanonicalJson :=
  .object (experimentSpecContentFields spec)

def ExperimentSpec.expectedArtifactChecksum (spec : ExperimentSpec) : ArtifactChecksum :=
  experimentSpecChecksumOf (experimentSpecContentJson spec).prettyBytes

def ExperimentSpec.hasValidArtifactChecksum (spec : ExperimentSpec) : Bool :=
  spec.artifactChecksum == spec.expectedArtifactChecksum

private def sealedExperimentSpecJson (spec : ExperimentSpec) : CanonicalJson :=
  .object (experimentSpecContentFields spec ++ [
    ("artifactChecksum", .string spec.artifactChecksum.render)
  ])

def canonicalExperimentSpecJson (spec : ExperimentSpec) : String :=
  (sealedExperimentSpecJson spec).pretty

def canonicalExperimentSpecBytes (spec : ExperimentSpec) : String :=
  (sealedExperimentSpecJson spec).prettyBytes

end Umpire
