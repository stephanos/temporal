import Umpire.Query

/-! Checked finite variation declarations over one existing Query closure. -/

namespace Umpire

structure ChoiceDeclaration where
  id : DefinitionId
  source : SourceLocation
  version : Nat := 1
  baseline : Bool := false
  binding : Option ModelValue := none
  faults : List DefinitionId := []
  documentation : String := ""
  deriving BEq, DecidableEq, Repr

structure VariationAxisDeclaration where
  id : DefinitionId
  source : SourceLocation
  version : Nat := 1
  role : Option DefinitionId := none
  choices : List ChoiceDeclaration
  documentation : String := ""
  deriving BEq, DecidableEq, Repr

structure FaultIntentDeclaration where
  id : DefinitionId
  source : SourceLocation
  version : Nat := 1
  occurrence : DefinitionId
  action : DefinitionId
  capability : DefinitionId
  incompatibleWith : List DefinitionId := []
  documentation : String := ""
  deriving BEq, DecidableEq, Repr

inductive CoverageSubject where
  | axisChoice (axis choice : DefinitionId)
  | fault (id : DefinitionId)
  | state (id : DefinitionId)
  | action (id : DefinitionId)
  | outcome (id : DefinitionId)
  | observation (id : DefinitionId)
  | relation (id : DefinitionId)
  | property (id : DefinitionId)
  deriving BEq, DecidableEq, Repr

structure CoverageGoalDeclaration where
  id : DefinitionId
  source : SourceLocation
  version : Nat := 1
  subject : CoverageSubject
  minimum : Nat
  documentation : String := ""
  deriving BEq, DecidableEq, Repr

structure ExperimentSpaceDeclaration where
  id : DefinitionId
  source : SourceLocation
  version : Nat := 1
  baseQuery : DefinitionId
  axes : List VariationAxisDeclaration
  faults : List FaultIntentDeclaration := []
  coverageGoals : List CoverageGoalDeclaration
  documentation : String := ""
  deriving BEq, DecidableEq, Repr

structure SpaceLimits where
  minimumAxes : Nat
  maximumAxes : Nat
  minimumChoicesPerAxis : Nat
  maximumChoicesPerAxis : Nat
  maximumPoints : Nat
  maximumFaults : Nat
  minimumCoverageGoals : Nat
  maximumCoverageGoals : Nat
  deriving BEq, DecidableEq, Repr

def SpaceLimits.v1 : SpaceLimits := {
  minimumAxes := 1
  maximumAxes := 8
  minimumChoicesPerAxis := 2
  maximumChoicesPerAxis := 16
  maximumPoints := 256
  maximumFaults := 12
  minimumCoverageGoals := 1
  maximumCoverageGoals := 64
}

inductive SpaceErrorKind where
  | emptyDefinitionId
  | invalidDefinitionId
  | duplicateDefinitionId
  | baseQueryMismatch
  | axisCountOutOfRange
  | choiceCountOutOfRange
  | pointCountExceeded
  | faultCountOutOfRange
  | coverageGoalCountOutOfRange
  | unknownRole
  | unwritableRole
  | duplicateControlledRole
  | invalidChoiceBinding
  | unknownValue
  | wrongValueKind
  | unavailableValue
  | conflictingBinding
  | baselineHasEffect
  | multipleBaseline
  | emptyChoiceEffect
  | duplicateChoiceEffect
  | unknownFault
  | duplicateFaultSelection
  | unknownOccurrence
  | occurrenceActionMismatch
  | unknownCapability
  | wrongReferenceKind
  | missingCapability
  | asymmetricFaultIncompatibility
  | incompatibleFaultSelection
  | invalidCoverageMinimum
  | unknownCoverageSubject
  | wrongCoverageSubjectKind
  | impossibleCoverageGoal
  deriving BEq, DecidableEq, Ord, Repr

def SpaceErrorKind.name : SpaceErrorKind → String
  | .emptyDefinitionId => "empty-definition-id"
  | .invalidDefinitionId => "invalid-definition-id"
  | .duplicateDefinitionId => "duplicate-definition-id"
  | .baseQueryMismatch => "base-query-mismatch"
  | .axisCountOutOfRange => "axis-count-out-of-range"
  | .choiceCountOutOfRange => "choice-count-out-of-range"
  | .pointCountExceeded => "point-count-exceeded"
  | .faultCountOutOfRange => "fault-count-out-of-range"
  | .coverageGoalCountOutOfRange => "coverage-goal-count-out-of-range"
  | .unknownRole => "unknown-role"
  | .unwritableRole => "unwritable-role"
  | .duplicateControlledRole => "duplicate-controlled-role"
  | .invalidChoiceBinding => "invalid-choice-binding"
  | .unknownValue => "unknown-value"
  | .wrongValueKind => "wrong-value-kind"
  | .unavailableValue => "unavailable-value"
  | .conflictingBinding => "conflicting-binding"
  | .baselineHasEffect => "baseline-has-effect"
  | .multipleBaseline => "multiple-baseline"
  | .emptyChoiceEffect => "empty-choice-effect"
  | .duplicateChoiceEffect => "duplicate-choice-effect"
  | .unknownFault => "unknown-fault"
  | .duplicateFaultSelection => "duplicate-fault-selection"
  | .unknownOccurrence => "unknown-occurrence"
  | .occurrenceActionMismatch => "occurrence-action-mismatch"
  | .unknownCapability => "unknown-capability"
  | .wrongReferenceKind => "wrong-reference-kind"
  | .missingCapability => "missing-capability"
  | .asymmetricFaultIncompatibility => "asymmetric-fault-incompatibility"
  | .incompatibleFaultSelection => "incompatible-fault-selection"
  | .invalidCoverageMinimum => "invalid-coverage-minimum"
  | .unknownCoverageSubject => "unknown-coverage-subject"
  | .wrongCoverageSubjectKind => "wrong-coverage-subject-kind"
  | .impossibleCoverageGoal => "impossible-coverage-goal"

structure SpaceError where
  kind : SpaceErrorKind
  definitionId : DefinitionId
  sourcePath : String
  offendingValue : String
  relatedDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

structure CheckedChoice where
  id : DefinitionId
  source : SourceLocation
  version : Nat
  baseline : Bool
  binding : Option RoleBinding
  faults : List DefinitionId
  documentation : String
  canonicalMetadata : String
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

structure CheckedVariationAxis where
  id : DefinitionId
  source : SourceLocation
  version : Nat
  role : Option ResourceRole
  choices : List CheckedChoice
  documentation : String
  canonicalMetadata : String
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

structure CheckedFaultIntent where
  id : DefinitionId
  source : SourceLocation
  version : Nat
  occurrence : NamedOccurrence
  capability : DefinitionMetadata
  incompatibleWith : List DefinitionId
  documentation : String
  canonicalMetadata : String
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

structure CheckedPropertyReference where
  id : DefinitionId
  source : SourceLocation
  version : Nat
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

inductive CheckedCoverageSubject where
  | axisChoice (axis choice : DefinitionId)
  | fault (id : DefinitionId)
  | definition (metadata : DefinitionMetadata)
  | property (reference : CheckedPropertyReference)
  deriving BEq, DecidableEq, Repr

structure CheckedCoverageGoal where
  id : DefinitionId
  source : SourceLocation
  version : Nat
  subject : CheckedCoverageSubject
  minimum : Nat
  documentation : String
  canonicalMetadata : String
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

structure SpaceCheckContext (LawStatement : LawDefinition → Prop) where
  baseQuery : CheckedQuery LawStatement

def SpaceCheckContext.ofQuery
    (query : CheckedQuery LawStatement) : SpaceCheckContext LawStatement := {
  baseQuery := query
}

structure CheckedExperimentSpace (LawStatement : LawDefinition → Prop) where
  private mk ::
  id : DefinitionId
  source : SourceLocation
  version : Nat
  baseQuery : CheckedQuery LawStatement
  axes : List CheckedVariationAxis
  faults : List CheckedFaultIntent
  coverageGoals : List CheckedCoverageGoal
  limits : SpaceLimits
  pointCount : Nat
  documentation : String
  canonicalMetadata : String
  behaviorFingerprint : BehaviorFingerprint

instance : BEq (CheckedExperimentSpace LawStatement) where
  beq left right := left.canonicalMetadata == right.canonicalMetadata &&
    left.behaviorFingerprint == right.behaviorFingerprint

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def idLe (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

private def canonicalIds (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort idLe

private def identityKey (id : DefinitionId) : String := id.value.toLower

private def identityLe (left right : DefinitionId) : Bool :=
  decide (identityKey left < identityKey right) ||
    (identityKey left == identityKey right && decide (left.value ≤ right.value))

private def sourcePath (source : SourceLocation) : String :=
  if source.path == "" then "<unknown>" else source.path

private def sourceJson (source : SourceLocation) : String :=
  "{\"path\":" ++ quote source.path ++
    ",\"line\":" ++ toString source.line ++
    ",\"column\":" ++ toString source.column ++
    ",\"provenance\":" ++ quote source.provenance ++ "}"

private def spaceError
    (kind : SpaceErrorKind)
    (owner : DefinitionId)
    (source : SourceLocation)
    (offendingValue : String)
    (relatedDefinitionIds : List DefinitionId := []) : SpaceError := {
  kind
  definitionId := if owner.value == "" then
    DefinitionId.of "umpire.space.anonymous"
  else
    owner
  sourcePath := sourcePath source
  offendingValue
  relatedDefinitionIds := canonicalIds relatedDefinitionIds
}

def canonicalSpaceErrorJson (error : SpaceError) : String :=
  "{\"kind\":" ++ quote error.kind.name ++
    ",\"definitionId\":" ++ quote error.definitionId.value ++
    ",\"sourcePath\":" ++ quote error.sourcePath ++
    ",\"offendingValue\":" ++ quote error.offendingValue ++
    ",\"relatedDefinitionIds\":" ++
      array (canonicalIds error.relatedDefinitionIds |>.map (quote ∘ DefinitionId.value)) ++ "}"

private def requireDefinitionId
    (owner : DefinitionId)
    (source : SourceLocation)
    (candidate : DefinitionId) : Except SpaceError Unit :=
  if candidate.value == "" then
    .error (spaceError .emptyDefinitionId owner source "<empty>" [candidate])
  else if !candidate.isNamespaced then
    .error (spaceError .invalidDefinitionId owner source candidate.value [candidate])
  else
    .ok ()

private def firstCaseCollision : List DefinitionId → Option (DefinitionId × DefinitionId)
  | first :: second :: rest =>
      if identityKey first == identityKey second then
        some (first, second)
      else
        firstCaseCollision (second :: rest)
  | _ => none

private def requireUniqueIdsAs
    (kind : SpaceErrorKind)
    (owner : DefinitionId)
    (source : SourceLocation)
    (ids : List DefinitionId) : Except SpaceError Unit :=
  match firstCaseCollision (ids.mergeSort identityLe) with
  | some (first, second) =>
      .error (spaceError kind owner source second.value [first, second])
  | none => .ok ()

private def requireUniqueIds
    (owner : DefinitionId)
    (source : SourceLocation)
    (ids : List DefinitionId) : Except SpaceError Unit :=
  requireUniqueIdsAs .duplicateDefinitionId owner source ids

private def choiceLe (left right : ChoiceDeclaration) : Bool := idLe left.id right.id
private def axisLe (left right : VariationAxisDeclaration) : Bool := idLe left.id right.id
private def faultLe (left right : FaultIntentDeclaration) : Bool := idLe left.id right.id
private def goalLe (left right : CoverageGoalDeclaration) : Bool := idLe left.id right.id

private def checkedChoiceLe (left right : CheckedChoice) : Bool := idLe left.id right.id
private def checkedAxisLe (left right : CheckedVariationAxis) : Bool := idLe left.id right.id
private def checkedFaultLe (left right : CheckedFaultIntent) : Bool := idLe left.id right.id
private def checkedGoalLe (left right : CheckedCoverageGoal) : Bool := idLe left.id right.id

private def valueJson (value : ModelValue) : String :=
  "{\"definitionId\":" ++ quote value.definitionId.value ++
    ",\"value\":" ++ quote value.value ++ "}"

private def bindingJson (binding : RoleBinding) : String :=
  "{\"role\":" ++ quote binding.role.value ++
    ",\"value\":" ++ valueJson binding.value ++ "}"

private def roleJson (role : ResourceRole) : String :=
  "{\"id\":" ++ quote role.id.value ++ ",\"valueKind\":" ++ quote role.valueKind.name ++ "}"

private def choiceSemanticJson
    (id : DefinitionId)
    (version : Nat)
    (baseline : Bool)
    (binding : Option RoleBinding)
    (faults : List DefinitionId) : String :=
  "{\"id\":" ++ quote id.value ++
    ",\"version\":" ++ toString version ++
    ",\"baseline\":" ++ (if baseline then "true" else "false") ++
    ",\"binding\":" ++ (binding.map bindingJson |>.getD "null") ++
    ",\"faults\":" ++ array (canonicalIds faults |>.map (quote ∘ DefinitionId.value)) ++ "}"

private def canonicalChoiceJson (choice : CheckedChoice) : String :=
  "{\"semantic\":" ++ choiceSemanticJson choice.id choice.version choice.baseline
      choice.binding choice.faults ++
    ",\"source\":" ++ sourceJson choice.source ++
    ",\"documentation\":" ++ quote choice.documentation ++ "}"

private def axisSemanticJson
    (id : DefinitionId)
    (version : Nat)
    (role : Option ResourceRole)
    (choices : List CheckedChoice) : String :=
  "{\"id\":" ++ quote id.value ++
    ",\"version\":" ++ toString version ++
    ",\"role\":" ++ (role.map roleJson |>.getD "null") ++
    ",\"choices\":" ++ array (choices.mergeSort checkedChoiceLe |>.map fun choice =>
      "{\"id\":" ++ quote choice.id.value ++
        ",\"behaviorFingerprint\":" ++ quote choice.behaviorFingerprint.render ++ "}") ++ "}"

private def canonicalAxisJson (axis : CheckedVariationAxis) : String :=
  "{\"semantic\":" ++ axisSemanticJson axis.id axis.version axis.role axis.choices ++
    ",\"source\":" ++ sourceJson axis.source ++
    ",\"documentation\":" ++ quote axis.documentation ++ "}"

private def faultSemanticJson
    (id : DefinitionId)
    (version : Nat)
    (occurrence : NamedOccurrence)
    (capability : DefinitionMetadata)
    (incompatibleWith : List DefinitionId) : String :=
  "{\"id\":" ++ quote id.value ++
    ",\"version\":" ++ toString version ++
    ",\"occurrence\":{\"id\":" ++ quote occurrence.id.value ++
      ",\"action\":" ++ quote occurrence.action.value ++ "}" ++
    ",\"capability\":{\"id\":" ++ quote capability.id.value ++
      ",\"version\":" ++ toString capability.version ++
      ",\"canonicalBehavior\":" ++ quote capability.canonicalBehavior ++ "}" ++
    ",\"incompatibleWith\":" ++
      array (canonicalIds incompatibleWith |>.map (quote ∘ DefinitionId.value)) ++ "}"

private def canonicalFaultJson (fault : CheckedFaultIntent) : String :=
  "{\"semantic\":" ++ faultSemanticJson fault.id fault.version fault.occurrence
      fault.capability fault.incompatibleWith ++
    ",\"source\":" ++ sourceJson fault.source ++
    ",\"documentation\":" ++ quote fault.documentation ++ "}"

private def coverageSubjectJson : CheckedCoverageSubject → String
  | .axisChoice axis choice =>
      "{\"kind\":\"axis-choice\",\"axis\":" ++ quote axis.value ++
        ",\"choice\":" ++ quote choice.value ++ "}"
  | .fault id => "{\"kind\":\"fault\",\"id\":" ++ quote id.value ++ "}"
  | .definition metadata =>
      "{\"kind\":" ++ quote metadata.kind.name ++
        ",\"id\":" ++ quote metadata.id.value ++
        ",\"version\":" ++ toString metadata.version ++
        ",\"canonicalBehavior\":" ++ quote metadata.canonicalBehavior ++ "}"
  | .property reference =>
      "{\"kind\":\"property\",\"id\":" ++ quote reference.id.value ++
        ",\"version\":" ++ toString reference.version ++
        ",\"behaviorFingerprint\":" ++ quote reference.behaviorFingerprint.render ++ "}"

private def goalSemanticJson
    (id : DefinitionId)
    (version : Nat)
    (subject : CheckedCoverageSubject)
    (minimum : Nat) : String :=
  "{\"id\":" ++ quote id.value ++
    ",\"version\":" ++ toString version ++
    ",\"subject\":" ++ coverageSubjectJson subject ++
    ",\"minimum\":" ++ toString minimum ++ "}"

private def canonicalGoalJson (goal : CheckedCoverageGoal) : String :=
  "{\"semantic\":" ++ goalSemanticJson goal.id goal.version goal.subject goal.minimum ++
    ",\"source\":" ++ sourceJson goal.source ++
    ",\"documentation\":" ++ quote goal.documentation ++ "}"

private def limitsJson (limits : SpaceLimits) : String :=
  "{\"axes\":{\"minimum\":" ++ toString limits.minimumAxes ++
      ",\"maximum\":" ++ toString limits.maximumAxes ++ "}" ++
    ",\"choicesPerAxis\":{\"minimum\":" ++ toString limits.minimumChoicesPerAxis ++
      ",\"maximum\":" ++ toString limits.maximumChoicesPerAxis ++ "}" ++
    ",\"maximumPoints\":" ++ toString limits.maximumPoints ++
    ",\"maximumFaults\":" ++ toString limits.maximumFaults ++
    ",\"coverageGoals\":{\"minimum\":" ++ toString limits.minimumCoverageGoals ++
      ",\"maximum\":" ++ toString limits.maximumCoverageGoals ++ "}}"

private def propertyReferenceJson (property : CheckedProperty) : String :=
  "{\"id\":" ++ quote property.id.value ++
    ",\"behaviorFingerprint\":" ++ quote property.behaviorFingerprint.render ++ "}"

private def spaceSemanticJson
    (id : DefinitionId)
    (version : Nat)
    (query : CheckedQuery LawStatement)
    (axes : List CheckedVariationAxis)
    (faults : List CheckedFaultIntent)
    (goals : List CheckedCoverageGoal)
    (limits : SpaceLimits)
    (pointCount : Nat) : String :=
  "{\"id\":" ++ quote id.value ++
    ",\"version\":" ++ toString version ++
    ",\"baseQuery\":{\"id\":" ++ quote query.id.value ++
      ",\"behaviorFingerprint\":" ++ quote query.behaviorFingerprint.render ++ "}" ++
    ",\"baseBehavior\":{\"id\":" ++ quote query.behavior.id.value ++
      ",\"behaviorFingerprint\":" ++ quote query.behavior.behaviorFingerprint.render ++ "}" ++
    ",\"target\":{\"id\":" ++ quote query.target.id.value ++
      ",\"behaviorFingerprint\":" ++ quote query.target.behaviorFingerprint.render ++ "}" ++
    ",\"properties\":" ++ array (query.form.properties.mergeSort (fun left right =>
      idLe left.id right.id) |>.map propertyReferenceJson) ++
    ",\"axes\":" ++ array (axes.mergeSort checkedAxisLe |>.map fun axis =>
      "{\"id\":" ++ quote axis.id.value ++
        ",\"behaviorFingerprint\":" ++ quote axis.behaviorFingerprint.render ++ "}") ++
    ",\"faults\":" ++ array (faults.mergeSort checkedFaultLe |>.map fun fault =>
      "{\"id\":" ++ quote fault.id.value ++
        ",\"behaviorFingerprint\":" ++ quote fault.behaviorFingerprint.render ++ "}") ++
    ",\"coverageGoals\":" ++ array (goals.mergeSort checkedGoalLe |>.map fun goal =>
      "{\"id\":" ++ quote goal.id.value ++
        ",\"behaviorFingerprint\":" ++ quote goal.behaviorFingerprint.render ++ "}") ++
    ",\"limits\":" ++ limitsJson limits ++
    ",\"pointCount\":" ++ toString pointCount ++ "}"

def canonicalExperimentSpaceJson (space : CheckedExperimentSpace LawStatement) : String :=
  space.canonicalMetadata

private def allDeclarationIds (declaration : ExperimentSpaceDeclaration) : List DefinitionId :=
  [declaration.id] ++ declaration.axes.flatMap (fun axis =>
    [axis.id] ++ axis.choices.map ChoiceDeclaration.id) ++
    declaration.faults.map FaultIntentDeclaration.id ++
    declaration.coverageGoals.map CoverageGoalDeclaration.id

private def checkPointCount
    (declaration : ExperimentSpaceDeclaration)
    (axes : List VariationAxisDeclaration)
    (limits : SpaceLimits) : Except SpaceError Nat := do
  let mut count := 1
  for axis in axes do
    let choiceCount := axis.choices.length
    if choiceCount < limits.minimumChoicesPerAxis || choiceCount > limits.maximumChoicesPerAxis then
      throw (spaceError .choiceCountOutOfRange axis.id axis.source
        (toString choiceCount ++ " not in " ++ toString limits.minimumChoicesPerAxis ++ ".." ++
          toString limits.maximumChoicesPerAxis) [axis.id])
    if choiceCount > limits.maximumPoints / count then
      throw (spaceError .pointCountExceeded declaration.id declaration.source
        (toString count ++ " * " ++ toString choiceCount ++ " > " ++
          toString limits.maximumPoints) (axes.map VariationAxisDeclaration.id))
    count := count * choiceCount
  pure count

private def findDefinition
    (query : CheckedQuery LawStatement)
    (candidate : DefinitionId) : Option DefinitionMetadata :=
  query.target.definitions.find? fun metadata => metadata.id == candidate

private def findRole
    (query : CheckedQuery LawStatement)
    (candidate : DefinitionId) : Option ResourceRole :=
  query.behavior.roles.find? fun role => role.id == candidate

private def bindingFor (bindings : List RoleBinding) (role : DefinitionId) : Option ModelValue :=
  (bindings.find? fun binding => binding.role == role).map RoleBinding.value

private def resolveOperand (bindings : List RoleBinding) : SetupOperand → Option ModelValue
  | .role role => bindingFor bindings role
  | .value value => some value

private def setupConstraintHolds
    (bindings : List RoleBinding)
    (constraint : SetupConstraint) : Bool :=
  match resolveOperand bindings constraint.left, resolveOperand bindings constraint.right with
  | some left, some right =>
      match constraint.relation with
      | .equal => left == right
      | .different => left != right
  | _, _ => false

private def targetAllowsBinding
    (query : CheckedQuery LawStatement)
    (binding : RoleBinding) : Bool :=
  query.target.resolvedSetups.any fun setup =>
    setup.contains binding && query.behavior.setup.all (setupConstraintHolds setup)

private def isSpaceMetadataKind : DefinitionKind → Bool
  | .experimentSpace | .variationAxis | .choice | .fault | .coverageGoal => true
  | _ => false

private def checkedChoiceEffectKey (choice : CheckedChoice) : String :=
  (choice.binding.map bindingJson |>.getD "null") ++ ":" ++
    array (choice.faults.map (quote ∘ DefinitionId.value))

private def firstDuplicateChoiceEffect : List CheckedChoice → Option (DefinitionId × DefinitionId)
  | first :: rest =>
      match rest.find? fun other => checkedChoiceEffectKey first == checkedChoiceEffectKey other with
      | some other => some (first.id, other.id)
      | none => firstDuplicateChoiceEffect rest
  | [] => none

private def checkChoice
    (query : CheckedQuery LawStatement)
    (role : Option ResourceRole)
    (declaredFaults : List FaultIntentDeclaration)
    (choice : ChoiceDeclaration) : Except SpaceError CheckedChoice := do
  requireUniqueIdsAs .duplicateFaultSelection choice.id choice.source choice.faults
  let faults := canonicalIds choice.faults
  if choice.baseline && (choice.binding.isSome || !faults.isEmpty) then
    throw (spaceError .baselineHasEffect choice.id choice.source choice.id.value [choice.id])
  if !choice.baseline && choice.binding.isNone && faults.isEmpty then
    throw (spaceError .emptyChoiceEffect choice.id choice.source choice.id.value [choice.id])
  for faultId in faults do
    if !(declaredFaults.any fun fault => fault.id == faultId) then
      throw (spaceError .unknownFault choice.id choice.source faultId.value [faultId])
  let binding ← match role, choice.binding with
    | none, none => pure none
    | none, some value =>
        throw (spaceError .invalidChoiceBinding choice.id choice.source
          value.definitionId.value [choice.id, value.definitionId])
    | some _role, none => pure none
    | some role, some value => do
        let metadata ← match findDefinition query value.definitionId with
          | some metadata => pure metadata
          | none =>
              throw (spaceError .unknownValue choice.id choice.source
                value.definitionId.value [value.definitionId])
        if metadata.kind != role.valueKind then
          throw (spaceError .wrongValueKind choice.id choice.source
            (value.definitionId.value ++ ": expected " ++ role.valueKind.name ++
              ", found " ++ metadata.kind.name) [role.id, value.definitionId])
        let binding := { role := role.id, value }
        if !(query.target.resolvedSetups.any fun setup => setup.contains binding) then
          throw (spaceError .unavailableValue choice.id choice.source
            value.definitionId.value [role.id, value.definitionId])
        if !targetAllowsBinding query binding then
          throw (spaceError .conflictingBinding choice.id choice.source
            value.definitionId.value [role.id, value.definitionId])
        pure (some binding)
  let semantic := choiceSemanticJson choice.id choice.version choice.baseline binding faults
  let checked : CheckedChoice := {
    id := choice.id
    source := choice.source
    version := choice.version
    baseline := choice.baseline
    binding
    faults
    documentation := choice.documentation
    canonicalMetadata := ""
    behaviorFingerprint := behaviorFingerprintOf semantic
  }
  pure { checked with canonicalMetadata := canonicalChoiceJson checked }

private def checkAxis
    (query : CheckedQuery LawStatement)
    (declaredFaults : List FaultIntentDeclaration)
    (axis : VariationAxisDeclaration) : Except SpaceError CheckedVariationAxis := do
  let role ← match axis.role with
    | none => pure none
    | some roleId =>
        match findRole query roleId with
        | none => throw (spaceError .unknownRole axis.id axis.source roleId.value [roleId])
        | some role =>
            if role.valueKind == .outcome || role.valueKind == .observation ||
                isSpaceMetadataKind role.valueKind then
              throw (spaceError .unwritableRole axis.id axis.source
                (role.id.value ++ ": " ++ role.valueKind.name) [role.id])
            pure (some role)
  let authoredChoices := axis.choices.mergeSort choiceLe
  match authoredChoices.find? fun choice =>
      choice.baseline && (choice.binding.isSome || !choice.faults.isEmpty) with
  | some choice =>
      throw (spaceError .baselineHasEffect choice.id choice.source choice.id.value [choice.id])
  | none => pure ()
  let baselineCount := authoredChoices.countP ChoiceDeclaration.baseline
  if baselineCount > 1 then
    throw (spaceError .multipleBaseline axis.id axis.source (toString baselineCount)
      (authoredChoices.filter ChoiceDeclaration.baseline |>.map ChoiceDeclaration.id))
  let mut choices := []
  for choice in authoredChoices do
    choices := choices ++ [← checkChoice query role declaredFaults choice]
  let effectful := choices.filter fun choice => !choice.baseline
  match firstDuplicateChoiceEffect effectful with
  | some (first, second) =>
      throw (spaceError .duplicateChoiceEffect axis.id axis.source second.value [first, second])
  | none => pure ()
  let semantic := axisSemanticJson axis.id axis.version role choices
  let checked : CheckedVariationAxis := {
    id := axis.id
    source := axis.source
    version := axis.version
    role
    choices := choices.mergeSort checkedChoiceLe
    documentation := axis.documentation
    canonicalMetadata := ""
    behaviorFingerprint := behaviorFingerprintOf semantic
  }
  pure { checked with canonicalMetadata := canonicalAxisJson checked }

private def targetHasCapability
    (query : CheckedQuery LawStatement)
    (capability : DefinitionId) : Bool :=
  query.target.requiredCapabilities.contains capability ||
    query.target.providers.any fun provider => provider.contract.id == capability

private def checkFault
    (query : CheckedQuery LawStatement)
    (declaration : FaultIntentDeclaration) : Except SpaceError CheckedFaultIntent := do
  requireUniqueIds declaration.id declaration.source declaration.incompatibleWith
  let occurrence ← match query.behavior.requiredOccurrences.find? fun occurrence =>
      occurrence.id == declaration.occurrence with
    | some occurrence => pure occurrence
    | none => throw (spaceError .unknownOccurrence declaration.id declaration.source
        declaration.occurrence.value [declaration.occurrence])
  if occurrence.action != declaration.action then
    throw (spaceError .occurrenceActionMismatch declaration.id declaration.source
      (declaration.action.value ++ " != " ++ occurrence.action.value)
      [declaration.occurrence, declaration.action, occurrence.action])
  let capability ← match findDefinition query declaration.capability with
    | none => throw (spaceError .unknownCapability declaration.id declaration.source
        declaration.capability.value [declaration.capability])
    | some metadata => pure metadata
  if capability.kind != .capability then
    throw (spaceError .wrongReferenceKind declaration.id declaration.source
      (capability.id.value ++ ": expected capability, found " ++ capability.kind.name)
      [capability.id])
  if !targetHasCapability query capability.id then
    throw (spaceError .missingCapability declaration.id declaration.source
      capability.id.value [capability.id, query.target.id])
  let incompatibleWith := canonicalIds declaration.incompatibleWith
  let semantic := faultSemanticJson declaration.id declaration.version occurrence capability
    incompatibleWith
  let checked : CheckedFaultIntent := {
    id := declaration.id
    source := declaration.source
    version := declaration.version
    occurrence
    capability
    incompatibleWith
    documentation := declaration.documentation
    canonicalMetadata := ""
    behaviorFingerprint := behaviorFingerprintOf semantic
  }
  pure { checked with canonicalMetadata := canonicalFaultJson checked }

private def validateFaultIncompatibilities
    (owner : ExperimentSpaceDeclaration)
    (faults : List CheckedFaultIntent) : Except SpaceError Unit := do
  for fault in faults do
    for incompatibleId in fault.incompatibleWith do
      let incompatible ← match faults.find? fun candidate => candidate.id == incompatibleId with
        | some candidate => pure candidate
        | none => throw (spaceError .unknownFault fault.id fault.source
            incompatibleId.value [incompatibleId])
      if incompatible.id == fault.id || !incompatible.incompatibleWith.contains fault.id then
        throw (spaceError .asymmetricFaultIncompatibility owner.id owner.source
          (fault.id.value ++ "<->" ++ incompatible.id.value) [fault.id, incompatible.id])

private def validateSelections
    (owner : ExperimentSpaceDeclaration)
    (axes : List CheckedVariationAxis)
    (faults : List CheckedFaultIntent) : Except SpaceError Unit := do
  for fault in faults do
    let selectingAxes := axes.filter fun axis =>
      axis.choices.any fun choice => choice.faults.contains fault.id
    if selectingAxes.length > 1 then
      throw (spaceError .duplicateFaultSelection owner.id owner.source fault.id.value
        (fault.id :: selectingAxes.map CheckedVariationAxis.id))
  for (leftAxis, leftIndex) in axes.zipIdx do
    for (rightAxis, rightIndex) in axes.zipIdx do
      if leftIndex < rightIndex then
        for leftChoice in leftAxis.choices do
          for rightChoice in rightAxis.choices do
            for leftFaultId in leftChoice.faults do
              let leftFault := faults.find? (fun fault => fault.id == leftFaultId)
              for rightFaultId in rightChoice.faults do
                if leftFault.any fun fault => fault.incompatibleWith.contains rightFaultId then
                  throw (spaceError .incompatibleFaultSelection owner.id owner.source
                    (leftFaultId.value ++ "<->" ++ rightFaultId.value)
                    [leftAxis.id, leftChoice.id, rightAxis.id, rightChoice.id,
                      leftFaultId, rightFaultId])
  for axis in axes do
    for choice in axis.choices do
      for leftFaultId in choice.faults do
        let leftFault := faults.find? (fun fault => fault.id == leftFaultId)
        for rightFaultId in choice.faults do
          if leftFaultId != rightFaultId &&
              leftFault.any (fun fault => fault.incompatibleWith.contains rightFaultId) then
            throw (spaceError .incompatibleFaultSelection choice.id choice.source
              (leftFaultId.value ++ "<->" ++ rightFaultId.value)
              [choice.id, leftFaultId, rightFaultId])

private def findSemanticSubject
    (query : CheckedQuery LawStatement)
    (owner : CoverageGoalDeclaration)
    (id : DefinitionId)
    (expected : DefinitionKind) : Except SpaceError CheckedCoverageSubject :=
  match findDefinition query id with
  | none => .error (spaceError .unknownCoverageSubject owner.id owner.source id.value [id])
  | some metadata =>
      if metadata.kind == expected then
        .ok (.definition metadata)
      else
        .error (spaceError .wrongCoverageSubjectKind owner.id owner.source
          (id.value ++ ": expected " ++ expected.name ++ ", found " ++ metadata.kind.name) [id])

private def faultMatchingPoints
    (axes : List CheckedVariationAxis)
    (pointCount : Nat)
    (faultId : DefinitionId) : Nat :=
  axes.foldl (fun result axis =>
    let selecting := (axis.choices.filter fun choice => choice.faults.contains faultId).length
    result + selecting * (pointCount / axis.choices.length)) 0

private def checkGoal
    (query : CheckedQuery LawStatement)
    (axes : List CheckedVariationAxis)
    (faults : List CheckedFaultIntent)
    (pointCount : Nat)
    (declaration : CoverageGoalDeclaration) : Except SpaceError CheckedCoverageGoal := do
  if declaration.minimum == 0 || declaration.minimum > SpaceLimits.v1.maximumPoints ||
      declaration.minimum > pointCount then
    throw (spaceError .invalidCoverageMinimum declaration.id declaration.source
      (toString declaration.minimum) [declaration.id])
  let subject ← match declaration.subject with
    | .axisChoice axisId choiceId =>
        match axes.find? fun axis => axis.id == axisId with
        | none => throw (spaceError .unknownCoverageSubject declaration.id declaration.source
            axisId.value [axisId, choiceId])
        | some axis =>
            if !(axis.choices.any fun choice => choice.id == choiceId) then
              throw (spaceError .unknownCoverageSubject declaration.id declaration.source
                choiceId.value [axisId, choiceId])
            let possible := pointCount / axis.choices.length
            if declaration.minimum > possible then
              throw (spaceError .impossibleCoverageGoal declaration.id declaration.source
                (toString declaration.minimum ++ " > " ++ toString possible) [axisId, choiceId])
            pure (.axisChoice axisId choiceId)
    | .fault faultId =>
        if !(faults.any fun fault => fault.id == faultId) then
          throw (spaceError .unknownCoverageSubject declaration.id declaration.source
            faultId.value [faultId])
        let possible := faultMatchingPoints axes pointCount faultId
        if declaration.minimum > possible then
          throw (spaceError .impossibleCoverageGoal declaration.id declaration.source
            (toString declaration.minimum ++ " > " ++ toString possible) [faultId])
        pure (.fault faultId)
    | .state id => findSemanticSubject query declaration id .state
    | .action id => findSemanticSubject query declaration id .action
    | .outcome id => findSemanticSubject query declaration id .outcome
    | .observation id => findSemanticSubject query declaration id .observation
    | .relation id => findSemanticSubject query declaration id .relation
    | .property propertyId =>
        match query.form.properties.find? fun property => property.id == propertyId with
        | none => throw (spaceError .unknownCoverageSubject declaration.id declaration.source
            propertyId.value [propertyId])
        | some property => pure (.property {
            id := property.id
            source := property.source
            version := property.version
            behaviorFingerprint := property.behaviorFingerprint
          })
  let semantic := goalSemanticJson declaration.id declaration.version subject declaration.minimum
  let checked : CheckedCoverageGoal := {
    id := declaration.id
    source := declaration.source
    version := declaration.version
    subject
    minimum := declaration.minimum
    documentation := declaration.documentation
    canonicalMetadata := ""
    behaviorFingerprint := behaviorFingerprintOf semantic
  }
  pure { checked with canonicalMetadata := canonicalGoalJson checked }

/-- Check one complete finite Space without enumerating its Cartesian assignments. -/
def checkExperimentSpace
    (context : SpaceCheckContext LawStatement)
    (declaration : ExperimentSpaceDeclaration) :
    Except SpaceError (CheckedExperimentSpace LawStatement) := do
  let limits := SpaceLimits.v1
  requireDefinitionId declaration.id declaration.source declaration.id
  let axes := declaration.axes.mergeSort axisLe
  let faults := declaration.faults.mergeSort faultLe
  let goals := declaration.coverageGoals.mergeSort goalLe
  if axes.length < limits.minimumAxes || axes.length > limits.maximumAxes then
    throw (spaceError .axisCountOutOfRange declaration.id declaration.source
      (toString axes.length) (axes.map VariationAxisDeclaration.id))
  if faults.length > limits.maximumFaults then
    throw (spaceError .faultCountOutOfRange declaration.id declaration.source
      (toString faults.length) (faults.map FaultIntentDeclaration.id))
  if goals.length < limits.minimumCoverageGoals || goals.length > limits.maximumCoverageGoals then
    throw (spaceError .coverageGoalCountOutOfRange declaration.id declaration.source
      (toString goals.length) (goals.map CoverageGoalDeclaration.id))
  let ids := allDeclarationIds declaration
  for candidate in ids.mergeSort identityLe do
    requireDefinitionId declaration.id declaration.source candidate
  requireUniqueIds declaration.id declaration.source ids
  let pointCount ← checkPointCount declaration axes limits
  if declaration.baseQuery != context.baseQuery.id then
    throw (spaceError .baseQueryMismatch declaration.id declaration.source
      (declaration.baseQuery.value ++ " != " ++ context.baseQuery.id.value)
      [declaration.baseQuery, context.baseQuery.id])
  let controlledRoles := axes.filterMap VariationAxisDeclaration.role
  requireUniqueIdsAs .duplicateControlledRole declaration.id declaration.source controlledRoles
  let mut builtFaults := []
  for fault in faults do
    builtFaults := builtFaults ++ [← checkFault context.baseQuery fault]
  let checkedFaults := builtFaults.mergeSort checkedFaultLe
  validateFaultIncompatibilities declaration checkedFaults
  let mut builtAxes := []
  for axis in axes do
    builtAxes := builtAxes ++ [← checkAxis context.baseQuery faults axis]
  let checkedAxes := builtAxes.mergeSort checkedAxisLe
  validateSelections declaration checkedAxes checkedFaults
  let mut builtGoals := []
  for goal in goals do
    builtGoals := builtGoals ++ [← checkGoal context.baseQuery checkedAxes checkedFaults pointCount goal]
  let checkedGoals := builtGoals.mergeSort checkedGoalLe
  let semantic := spaceSemanticJson declaration.id declaration.version context.baseQuery checkedAxes
    checkedFaults checkedGoals limits pointCount
  let canonical := "{\"semantic\":" ++ semantic ++
    ",\"source\":" ++ sourceJson declaration.source ++
    ",\"documentation\":" ++ quote declaration.documentation ++ "}"
  pure {
    id := declaration.id
    source := declaration.source
    version := declaration.version
    baseQuery := context.baseQuery
    axes := checkedAxes
    faults := checkedFaults
    coverageGoals := checkedGoals
    limits
    pointCount
    documentation := declaration.documentation
    canonicalMetadata := canonical
    behaviorFingerprint := behaviorFingerprintOf semantic
  }

end Umpire
