import Umpire.Query

/-! Canonicalization and checking for authored Queries. -/

namespace Umpire

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def propertyLe (left right : CheckedProperty) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def queryError
    (kind : QueryErrorKind)
    (owner : Query)
    (offendingValue : String)
    (relatedDefinitionIds : List DefinitionId := []) : QueryError := {
  kind
  definitionId := if owner.id.value == "" then
    DefinitionId.of "umpire.query.anonymous"
  else
    owner.id
  sourcePath := owner.source.displayPath
  offendingValue
  relatedDefinitionIds := DefinitionId.canonicalSet relatedDefinitionIds
}

private def requirementIds (missing : List CompletenessRequirement) : List DefinitionId :=
  missing.map fun requirement => DefinitionId.of ("umpire.query." ++ requirement.name)

private def firstDuplicate [BEq α] : List α → Option α
  | first :: second :: rest =>
      if first == second then some first else firstDuplicate (second :: rest)
  | _ => none

private def validateFiniteDomains
    (declaration : Query)
    (evidence : Option (FiniteCompletenessEvidence LawStatement target)) :
    Except QueryError Unit := do
  match evidence with
  | none => pure ()
  | some evidence =>
      match firstDuplicate
          (Query.FiniteDomain.canonicalRoleAssignments evidence.roleAssignments) with
      | some duplicate =>
          throw (queryError .duplicateFiniteDomain declaration
            ("role-assignment:" ++ Query.FiniteDomain.roleAssignmentJson duplicate)
            (duplicate.map RoleBinding.role))
      | none => pure ()
      match firstDuplicate (Query.FiniteDomain.canonicalActions evidence.actions) with
      | some duplicate =>
          throw (queryError .duplicateFiniteDomain declaration
            ("action:" ++ Query.FiniteDomain.valueJson duplicate) [duplicate.definitionId])
      | none => pure ()

private def validateDefinitionId (declaration : Query) : Except QueryError Unit :=
  match declaration.id.validate with
  | .error .empty =>
      .error (queryError .emptyDefinitionId declaration "<empty>")
  | .error .malformed =>
      .error (queryError .invalidDefinitionId declaration declaration.id.value [declaration.id])
  | .ok () => .ok ()

private def validateProperties
    (declaration : Query)
    (target : QueryModel LawStatement) : Except QueryError Unit := do
  let properties := declaration.form.properties.mergeSort propertyLe
  if properties.isEmpty then
    throw (queryError .missingProperty declaration "properties")
  match DefinitionId.firstDuplicate (properties.map CheckedProperty.id) with
  | some duplicate =>
      throw (queryError .duplicateProperty declaration duplicate.value [duplicate])
  | none => pure ()
  for capability in declaration.behavior.requires ++ properties.flatMap CheckedProperty.requires do
    if !target.requiredCapabilities.contains capability then
      throw (queryError .missingCapability declaration capability.value [capability, target.id])

private def validateLimits (declaration : Query) : Except QueryError Unit := do
  let limits := declaration.limits
  if limits.steps.value == 0 then
    throw (queryError .invalidLimit declaration "steps=0")
  if limits.actions.value == 0 then
    throw (queryError .invalidLimit declaration "actions=0")
  if limits.search.value == 0 then
    throw (queryError .invalidLimit declaration "search=0")
  if limits.steps.unit != .steps then
    throw (queryError .unitMismatch declaration ("steps:" ++ limits.steps.unit.name))
  if limits.actions.unit != .actions then
    throw (queryError .unitMismatch declaration ("actions:" ++ limits.actions.unit.name))
  if limits.search.unit != .search then
    throw (queryError .unitMismatch declaration ("search:" ++ limits.search.unit.name))

private def validateStrategy (declaration : Query) : Except QueryError Unit :=
  match declaration.form, declaration.policy.strategy with
  | .verify _, .exhaustive => .ok ()
  | .verify _, strategy =>
      .error (queryError .incompatibleStrategy declaration strategy.name)
  | _, _ => .ok ()

private def modelProviders (target : QueryModel LawStatement) : List DefinitionId :=
  DefinitionId.canonicalSet (target.requiredCapabilities ++
    target.providers.map Provider.id ++
    target.connectors.map Connector.id)

private def validateExactTrace
    (declaration : Query)
    (target : QueryModel LawStatement) : Except QueryError Unit := do
  match declaration.behavior.traceExactly with
  | none => pure ()
  | some exact =>
      if !target.resolvedSetups.contains exact.setup then
        throw (queryError .targetKernelMismatch declaration "setup" [target.id])
      if !((target.machine.initialStates exact.setup).contains exact.trace.initialState) then
        throw (queryError .targetKernelMismatch declaration "initial-state"
          [target.id, target.machine.metadata.id])
      let mut current := exact.trace.initialState
      for (step, index) in exact.trace.steps.zipIdx do
        let expected : Step ModelValue ModelValue ModelValue := {
          outcome := step.outcome
          state := step.state
          facts := step.facts
        }
        if !((target.machine.steps current step.selectedAction).contains expected) then
          throw (queryError .targetKernelMismatch declaration
            ("step-" ++ toString index)
            [target.id, target.machine.metadata.id, step.selectedAction.definitionId])
        current := step.state

private def stringListJson (items : List String) : String :=
  array (items.map quote)

private def propertyJson (property : CheckedProperty) : String :=
  "{\"id\":" ++ quote property.id.value ++
    ",\"behaviorFingerprint\":" ++ quote property.behaviorFingerprint.render ++ "}"

private def limitsJson (limits : Limits) : String :=
  "{\"steps\":" ++ canonicalLimitJson limits.steps ++
    ",\"actions\":" ++ canonicalLimitJson limits.actions ++
    ",\"search\":" ++ canonicalLimitJson limits.search ++ "}"

private def policyJson (policy : PlannerPolicy) : String :=
  "{\"strategy\":" ++ quote policy.strategy.name ++
    ",\"seed\":" ++ toString policy.seed ++ "}"

private def completenessJson
    (evidence : Option (FiniteCompletenessEvidence LawStatement target)) : String :=
  match evidence with
  | none => "null"
  | some evidence =>
      "{\"roleDomainFingerprint\":" ++ quote evidence.roleDomainFingerprint.render ++
        ",\"actionDomainFingerprint\":" ++ quote evidence.actionDomainFingerprint.render ++ "}"

private def querySemanticJson
    (id : DefinitionId)
    (version : Nat)
    (form : Query.Form)
    (behavior : CheckedScenario)
    (target : QueryModel LawStatement)
    (composition : List DefinitionId)
    (limits : Limits)
    (policy : PlannerPolicy)
    (completeness : Option (FiniteCompletenessEvidence LawStatement target)) : String :=
  let properties := form.properties.mergeSort propertyLe
  "{\"id\":" ++ quote id.value ++
    ",\"version\":" ++ toString version ++
    ",\"form\":" ++ quote form.name ++
    ",\"properties\":" ++ array (properties.map propertyJson) ++
    ",\"behavior\":{\"id\":" ++ quote behavior.id.value ++
      ",\"behaviorFingerprint\":" ++ quote behavior.behaviorFingerprint.render ++ "}" ++
    ",\"limits\":" ++ limitsJson limits ++
    ",\"policy\":" ++ policyJson policy ++
    ",\"target\":{\"id\":" ++ quote target.id.value ++
      ",\"behaviorFingerprint\":" ++ quote target.behaviorFingerprint.render ++
      ",\"composition\":" ++
        stringListJson (composition.map DefinitionId.value) ++
      ",\"kernel\":{\"id\":" ++ quote target.machine.metadata.id.value ++ "}}" ++
    ",\"finiteCompleteness\":" ++ completenessJson completeness ++ "}"

/-- Query JSON is the canonical semantic projection; source order and documentation stay outside
the persisted identity. -/
def canonicalQueryJson (query : CheckedQuery LawStatement) : String :=
  query.canonicalMetadata

def canonicalQueryErrorJson (error : QueryError) : String :=
  "{\"kind\":" ++ quote error.kind.name ++
    ",\"definitionId\":" ++ quote error.definitionId.value ++
    ",\"sourcePath\":" ++ quote error.sourcePath ++
    ",\"offendingValue\":" ++ quote error.offendingValue ++
    ",\"relatedDefinitionIds\":" ++
      stringListJson (DefinitionId.canonicalSet error.relatedDefinitionIds |>.map
        DefinitionId.value) ++ "}"

/-- Freeze every meaning-bearing input and reject invalid exhaustive or exact-trace queries before
the search backend can be initialized. -/
def Query.check
    (context : QueryCheckContext LawStatement)
    (declaration : Query) : Except QueryError (CheckedQuery LawStatement) := do
  validateDefinitionId declaration
  let model ← match context.target with
    | .checked target => pure target
    | .incomplete targetId _ missing =>
        throw (queryError .missingFiniteCompleteness declaration
          (String.intercalate "," (missing.map CompletenessRequirement.name))
          (targetId :: requirementIds missing))
  let target := model.target
  if declaration.target != target.id then
    throw (queryError .targetMismatch declaration
      (declaration.target.value ++ " != " ++ target.id.value)
      [declaration.target, target.id])
  validateLimits declaration
  validateStrategy declaration
  validateProperties declaration target
  validateExactTrace declaration target
  validateFiniteDomains declaration model.completeness
  if declaration.policy.strategy == .exhaustive && model.completeness.isNone then
    throw (queryError .missingFiniteCompleteness declaration "finite role/action domains"
      [target.id, target.machine.metadata.id])
  let completeness := model.completeness
  let composition := modelProviders target
  let legacySemantic := querySemanticJson declaration.id declaration.version declaration.form
    declaration.behavior target composition declaration.limits declaration.policy completeness
  let semantic := if declaration.ending == .final && !declaration.requireFiring then
    legacySemantic
  else
    (legacySemantic.dropEnd 1).toString ++ ",\"endingPolicy/v1\":{\"ending\":" ++
      quote declaration.ending.name ++ ",\"requireFiring\":" ++
      (if declaration.requireFiring then "true" else "false") ++ "}}"
  pure {
    id := declaration.id
    source := declaration.source
    version := declaration.version
    form := declaration.form
    behavior := declaration.behavior
    target
    limits := declaration.limits
    policy := declaration.policy
    ending := declaration.ending
    requireFiring := declaration.requireFiring
    authoredKnownGaps := declaration.authoredKnownGaps
    modelProviders := composition
    completeness
    documentation := declaration.documentation
    canonicalMetadata := semantic
    behaviorFingerprint := behaviorFingerprintOf semantic
  }

/-- Produce a checked Query directly from an explicit proof that the typed checker succeeds. Model
re-ascription stays inside this boundary so dependent search APIs see the selected Model. Use
`Query.check` when an invalid declaration's typed diagnostic is needed. -/
def Query.checked
    (target : QueryModel LawStatement)
    (declaration : Query)
    (valid : (Query.check (.ofTarget target) declaration).toOption.isSome = true) :
    CheckedQuery LawStatement :=
  let checked := (Query.check (.ofTarget target) declaration).toOption.get valid
  {
    checked with
    target
    completeness := (ModelCompleteness.ofTarget target).completeness
  }

def Query.error?
    (declaration : Query)
    (target : QueryModel LawStatement) : Option QueryError :=
  match Query.check (.ofTarget target) declaration with
  | .error error => some error
  | .ok _ => none

end Umpire
