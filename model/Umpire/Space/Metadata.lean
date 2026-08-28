import Umpire.Space.Language

/-! Canonical in-memory metadata rows projected from one checked Experiment Space. -/

namespace Umpire

/-- A source-backed semantic reference without a copy of the referenced semantic body. -/
structure SpaceSemanticReference where
  id : DefinitionId
  source : SourceLocation
  version : Nat
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- A typed Model Definition reference without its canonical behavior body. -/
structure SpaceDefinitionReference extends SpaceSemanticReference where
  kind : DefinitionKind
  deriving BEq, DecidableEq, Repr

/-- Canonical metadata for one checked Space and the Query/Behavior/Target it closes over. -/
structure SpaceMetadataRow where
  id : DefinitionId
  source : SourceLocation
  version : Nat
  baseQuery : SpaceSemanticReference
  baseBehavior : SpaceSemanticReference
  target : SpaceSemanticReference
  properties : List SpaceSemanticReference
  limits : SpaceLimits
  pointCount : Nat
  baseBehaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- Canonical metadata for one checked variation axis. -/
structure SpaceAxisMetadataRow where
  id : DefinitionId
  source : SourceLocation
  version : Nat
  role : Option ResourceRole
  choices : List DefinitionId
  baseBehaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- Canonical metadata for one checked choice and its requested effects. -/
structure SpaceChoiceMetadataRow where
  id : DefinitionId
  source : SourceLocation
  version : Nat
  axis : DefinitionId
  baseline : Bool
  binding : Option RoleBinding
  faults : List DefinitionId
  baseBehaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- Canonical metadata for one checked request-only fault. -/
structure SpaceFaultMetadataRow where
  id : DefinitionId
  source : SourceLocation
  version : Nat
  occurrence : NamedOccurrence
  capability : SpaceDefinitionReference
  incompatibleWith : List DefinitionId
  baseBehaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- Reference-closed subject of one canonical coverage-goal row. -/
inductive SpaceCoverageMetadataSubject where
  | axisChoice (axis choice : DefinitionId)
  | fault (id : DefinitionId)
  | definition (reference : SpaceDefinitionReference)
  | property (reference : SpaceSemanticReference)
  deriving BEq, DecidableEq, Repr

/-- Canonical metadata for one checked seek-only coverage goal. -/
structure SpaceCoverageGoalMetadataRow where
  id : DefinitionId
  source : SourceLocation
  version : Nat
  subject : SpaceCoverageMetadataSubject
  minimum : Nat
  baseBehaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- Stable categories for stale, missing, extra, or mismatched Space metadata. -/
inductive SpaceMetadataErrorKind where
  | baseDigestMismatch
  | missingRow
  | extraRow
  | staleRow
  | behaviorFingerprintMismatch
  deriving BEq, DecidableEq, Ord, Repr

/-- Stable serialized name of a Space metadata failure category. -/
def SpaceMetadataErrorKind.name : SpaceMetadataErrorKind → String
  | .baseDigestMismatch => "base-digest-mismatch"
  | .missingRow => "missing-row"
  | .extraRow => "extra-row"
  | .staleRow => "stale-row"
  | .behaviorFingerprintMismatch => "behavior-fingerprint-mismatch"

/-- Canonical typed failure returned when a metadata projection is not self-consistent. -/
structure SpaceMetadataError where
  kind : SpaceMetadataErrorKind
  definitionId : DefinitionId
  sourcePath : String
  offendingValue : String
  relatedDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

/-- Unchecked rows can be inspected or transported, but fn-5 consumes only the checked aggregate. -/
structure SpaceMetadataProjection where
  space : SpaceMetadataRow
  axes : List SpaceAxisMetadataRow
  choices : List SpaceChoiceMetadataRow
  faults : List SpaceFaultMetadataRow
  coverageGoals : List SpaceCoverageGoalMetadataRow
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- Canonical metadata for one checked Space. Its constructor is intentionally not public. -/
structure CheckedSpaceMetadata where
  private mk ::
  space : SpaceMetadataRow
  axes : List SpaceAxisMetadataRow
  choices : List SpaceChoiceMetadataRow
  faults : List SpaceFaultMetadataRow
  coverageGoals : List SpaceCoverageGoalMetadataRow
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

private def idLe (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

private def propertyLe (left right : CheckedProperty) : Bool := idLe left.id right.id
private def axisRowLe (left right : SpaceAxisMetadataRow) : Bool := idLe left.id right.id
private def choiceRowLe (left right : SpaceChoiceMetadataRow) : Bool := idLe left.id right.id
private def faultRowLe (left right : SpaceFaultMetadataRow) : Bool := idLe left.id right.id
private def goalRowLe (left right : SpaceCoverageGoalMetadataRow) : Bool := idLe left.id right.id

private def canonicalIds (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort idLe

private def sourcePath (source : SourceLocation) : String :=
  if source.path == "" then "<unknown>" else source.path

private def metadataError
    (kind : SpaceMetadataErrorKind)
    (space : SpaceMetadataRow)
    (offendingValue : String)
    (relatedDefinitionIds : List DefinitionId := []) : SpaceMetadataError := {
  kind
  definitionId := space.id
  sourcePath := sourcePath space.source
  offendingValue
  relatedDefinitionIds := canonicalIds relatedDefinitionIds
}

private def semanticReference
    (id : DefinitionId)
    (source : SourceLocation)
    (version : Nat)
    (behaviorFingerprint : BehaviorFingerprint) : SpaceSemanticReference := {
  id
  source
  version
  behaviorFingerprint
}

private def definitionReference (metadata : DefinitionMetadata) : SpaceDefinitionReference := {
  toSpaceSemanticReference := semanticReference metadata.id metadata.source metadata.version
    (behaviorFingerprintOf metadata.canonicalBehavior)
  kind := metadata.kind
}

private def targetReference (space : CheckedExperimentSpace LawStatement) : SpaceSemanticReference :=
  let target := space.baseQuery.target
  let version := (target.definitions.find? fun metadata =>
    metadata.id == target.id && metadata.kind == .target).map DefinitionMetadata.version |>.getD 1
  semanticReference target.id target.source version target.behaviorFingerprint

private def coverageSubject : CheckedCoverageSubject → SpaceCoverageMetadataSubject
  | .axisChoice axis choice => .axisChoice axis choice
  | .fault id => .fault id
  | .definition metadata => .definition (definitionReference metadata)
  | .property reference => .property (semanticReference reference.id reference.source
      reference.version reference.behaviorFingerprint)

private def spaceRow (space : CheckedExperimentSpace LawStatement) : SpaceMetadataRow := {
  id := space.id
  source := space.source
  version := space.version
  baseQuery := semanticReference space.baseQuery.id space.baseQuery.source space.baseQuery.version
    space.baseQuery.behaviorFingerprint
  baseBehavior := semanticReference space.baseQuery.behavior.id space.baseQuery.behavior.source
    space.baseQuery.behavior.version space.baseQuery.behavior.behaviorFingerprint
  target := targetReference space
  properties := space.baseQuery.form.properties.mergeSort propertyLe |>.map fun property =>
    semanticReference property.id property.source property.version property.behaviorFingerprint
  limits := space.limits
  pointCount := space.pointCount
  baseBehaviorFingerprint := space.behaviorFingerprint
}

private def axisRows (space : CheckedExperimentSpace LawStatement) : List SpaceAxisMetadataRow :=
  space.axes.map (fun axis => {
    id := axis.id
    source := axis.source
    version := axis.version
    role := axis.role
    choices := axis.choices.map CheckedChoice.id |>.mergeSort idLe
    baseBehaviorFingerprint := axis.behaviorFingerprint
  }) |>.mergeSort axisRowLe

private def choiceRows (space : CheckedExperimentSpace LawStatement) : List SpaceChoiceMetadataRow :=
  space.axes.flatMap (fun axis => axis.choices.map fun choice => {
    id := choice.id
    source := choice.source
    version := choice.version
    axis := axis.id
    baseline := choice.baseline
    binding := choice.binding
    faults := choice.faults.mergeSort idLe
    baseBehaviorFingerprint := choice.behaviorFingerprint
  }) |>.mergeSort choiceRowLe

private def faultRows (space : CheckedExperimentSpace LawStatement) : List SpaceFaultMetadataRow :=
  space.faults.map (fun fault => {
    id := fault.id
    source := fault.source
    version := fault.version
    occurrence := fault.occurrence
    capability := definitionReference fault.capability
    incompatibleWith := fault.incompatibleWith.mergeSort idLe
    baseBehaviorFingerprint := fault.behaviorFingerprint
  }) |>.mergeSort faultRowLe

private def coverageGoalRows
    (space : CheckedExperimentSpace LawStatement) : List SpaceCoverageGoalMetadataRow :=
  space.coverageGoals.map (fun goal => {
    id := goal.id
    source := goal.source
    version := goal.version
    subject := coverageSubject goal.subject
    minimum := goal.minimum
    baseBehaviorFingerprint := goal.behaviorFingerprint
  }) |>.mergeSort goalRowLe

/-- Build the canonical unchecked rows used as input to the fail-closed metadata checker. -/
def canonicalSpaceMetadataProjection
    (space : CheckedExperimentSpace LawStatement) : SpaceMetadataProjection := {
  space := spaceRow space
  axes := axisRows space
  choices := choiceRows space
  faults := faultRows space
  coverageGoals := coverageGoalRows space
  behaviorFingerprint := space.behaviorFingerprint
}

private def firstDuplicate : List DefinitionId → Option DefinitionId
  | first :: second :: rest =>
      if first == second then some second else firstDuplicate (second :: rest)
  | _ => none

private def validateBijection
    (space : SpaceMetadataRow)
    (rowKind : String)
    (expected actual : List DefinitionId) : Except SpaceMetadataError Unit := do
  let canonicalActual := canonicalIds actual
  match firstDuplicate canonicalActual with
  | some duplicate =>
      throw (metadataError .extraRow space (rowKind ++ ":" ++ duplicate.value) [duplicate])
  | none => pure ()
  match expected.find? fun id => !canonicalActual.contains id with
  | some missing =>
      throw (metadataError .missingRow space (rowKind ++ ":" ++ missing.value) [missing])
  | none => pure ()
  match canonicalActual.find? fun id => !expected.contains id with
  | some extra =>
      throw (metadataError .extraRow space (rowKind ++ ":" ++ extra.value) [extra])
  | none => pure ()

private def staleBaseDigest
    (expected actual : SpaceMetadataRow) : Option DefinitionId :=
  if expected.baseQuery.behaviorFingerprint != actual.baseQuery.behaviorFingerprint then
    some expected.baseQuery.id
  else if expected.baseBehavior.behaviorFingerprint != actual.baseBehavior.behaviorFingerprint then
    some expected.baseBehavior.id
  else if expected.target.behaviorFingerprint != actual.target.behaviorFingerprint then
    some expected.target.id
  else
    (expected.properties.zip actual.properties).findSome? fun pair =>
      if pair.1.behaviorFingerprint != pair.2.behaviorFingerprint then some pair.1.id else none

private def firstStaleAxis
    (expected actual : List SpaceAxisMetadataRow) : Option DefinitionId :=
  expected.findSome? fun row =>
    match actual.find? fun candidate => candidate.id == row.id with
    | some candidate => if candidate == row then none else some row.id
    | none => none

private def firstStaleChoice
    (expected actual : List SpaceChoiceMetadataRow) : Option DefinitionId :=
  expected.findSome? fun row =>
    match actual.find? fun candidate => candidate.id == row.id with
    | some candidate => if candidate == row then none else some row.id
    | none => none

private def firstStaleFault
    (expected actual : List SpaceFaultMetadataRow) : Option DefinitionId :=
  expected.findSome? fun row =>
    match actual.find? fun candidate => candidate.id == row.id with
    | some candidate => if candidate == row then none else some row.id
    | none => none

private def firstStaleGoal
    (expected actual : List SpaceCoverageGoalMetadataRow) : Option DefinitionId :=
  expected.findSome? fun row =>
    match actual.find? fun candidate => candidate.id == row.id with
    | some candidate => if candidate == row then none else some row.id
    | none => none

/-- Validate row completeness, exact references, and digests against the checked Space. -/
def checkSpaceMetadataProjection
    (space : CheckedExperimentSpace LawStatement)
    (candidate : SpaceMetadataProjection) :
    Except SpaceMetadataError CheckedSpaceMetadata := do
  let expected := canonicalSpaceMetadataProjection space
  match staleBaseDigest expected.space candidate.space with
  | some stale =>
      throw (metadataError .baseDigestMismatch expected.space stale.value [stale])
  | none => pure ()
  if candidate.space != expected.space then
    throw (metadataError .staleRow expected.space ("space:" ++ expected.space.id.value)
      [expected.space.id])
  validateBijection expected.space "axis" (expected.axes.map SpaceAxisMetadataRow.id)
    (candidate.axes.map SpaceAxisMetadataRow.id)
  validateBijection expected.space "choice" (expected.choices.map SpaceChoiceMetadataRow.id)
    (candidate.choices.map SpaceChoiceMetadataRow.id)
  validateBijection expected.space "fault" (expected.faults.map SpaceFaultMetadataRow.id)
    (candidate.faults.map SpaceFaultMetadataRow.id)
  validateBijection expected.space "coverage-goal"
    (expected.coverageGoals.map SpaceCoverageGoalMetadataRow.id)
    (candidate.coverageGoals.map SpaceCoverageGoalMetadataRow.id)
  let axes := candidate.axes.mergeSort axisRowLe
  let choices := candidate.choices.mergeSort choiceRowLe
  let faults := candidate.faults.mergeSort faultRowLe
  let goals := candidate.coverageGoals.mergeSort goalRowLe
  match firstStaleAxis expected.axes axes with
  | some stale => throw (metadataError .staleRow expected.space ("axis:" ++ stale.value) [stale])
  | none => pure ()
  match firstStaleChoice expected.choices choices with
  | some stale => throw (metadataError .staleRow expected.space ("choice:" ++ stale.value) [stale])
  | none => pure ()
  match firstStaleFault expected.faults faults with
  | some stale => throw (metadataError .staleRow expected.space ("fault:" ++ stale.value) [stale])
  | none => pure ()
  match firstStaleGoal expected.coverageGoals goals with
  | some stale =>
      throw (metadataError .staleRow expected.space ("coverage-goal:" ++ stale.value) [stale])
  | none => pure ()
  if candidate.behaviorFingerprint != expected.behaviorFingerprint then
    throw (metadataError .behaviorFingerprintMismatch expected.space
      candidate.behaviorFingerprint.render [expected.space.id])
  pure {
    space := expected.space
    axes
    choices
    faults
    coverageGoals := goals
    behaviorFingerprint := expected.behaviorFingerprint
  }

/-- Project one checked Space to its deterministic source-backed metadata rows. -/
def projectCheckedSpaceMetadata
    (space : CheckedExperimentSpace LawStatement) :
    Except SpaceMetadataError CheckedSpaceMetadata :=
  checkSpaceMetadataProjection space (canonicalSpaceMetadataProjection space)

end Umpire
