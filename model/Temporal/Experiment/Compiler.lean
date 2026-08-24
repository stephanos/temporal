import Temporal.Experiment.DSL
import Temporal.Experiment.Json

namespace Temporal.Experiment

private def compileError (kind : CompileErrorKind) (subject context : String) : CompileError :=
  { kind, subject, context }

private def requirePresent (context : String) (names : List String) : Except CompileError Unit :=
  match names.find? (fun name => name == "") with
  | some name => .error (compileError .missingIdentity name context)
  | none => .ok ()

private def firstDuplicate : List String → Option String
  | first :: second :: rest =>
      if first == second then some first else firstDuplicate (second :: rest)
  | _ => none

private def requireUnique (context : String) (names : List String) : Except CompileError Unit :=
  match firstDuplicate names.mergeSort with
  | some name => .error (compileError .duplicateIdentity name context)
  | none => .ok ()

private def validateIdentities (target : ModelTarget) (regression : Regression) : Except CompileError Unit := do
  requirePresent "regression" [regression.id.value]
  requirePresent "regression.target" [regression.target.value]
  requirePresent "regression.resource" (regression.resources.map ResourceId.value)
  requirePresent "regression.action" (regression.actionAttempts.map ActionId.value)
  requirePresent "regression.property" (regression.expectedProperties.items.map PropertyId.value)
  requirePresent "regression.ordering" (regression.ordering.flatMap fun edge => [edge.before.value, edge.after.value])
  requirePresent "target" [target.id.value, target.declaration]
  requirePresent "target.resource" (target.resources.map fun binding => binding.id.value)
  requirePresent "target.action" (target.actionProjections.map fun projection => projection.id.value)
  requirePresent "target.property" (target.propertyObservations.map fun observation => observation.id.value)
  requireUnique "regression.resource" (regression.resources.map ResourceId.value)
  requireUnique "regression.action" (regression.actionAttempts.map ActionId.value)
  requireUnique "regression.property" (regression.expectedProperties.items.map PropertyId.value)
  requireUnique "target.resource" (target.resources.map fun binding => binding.id.value)
  requireUnique "target.action" (target.actionProjections.map fun projection => projection.id.value)
  requireUnique "target.property" (target.propertyObservations.map fun observation => observation.id.value)

private def validateExpectations (regression : Regression) : Except CompileError Unit :=
  if regression.expectedProperties.items.isEmpty then
    .error (compileError .emptyExpectations regression.id.value "expectedProperties")
  else
    .ok ()

private def validateTarget (target : ModelTarget) (regression : Regression) : Except CompileError Unit :=
  if regression.target == target.id then
    .ok ()
  else
    .error (compileError .targetMismatch regression.target.value target.id.value)

private def validateBounds (regression : Regression) : Except CompileError Unit := do
  if regression.bounds.resources == 0 then
    throw (compileError .invalidBound "resources" "must be positive")
  if regression.bounds.actions == 0 then
    throw (compileError .invalidBound "actions" "must be positive")
  if regression.resources.length > regression.bounds.resources then
    throw (compileError .boundExceeded "resources"
      (toString regression.resources.length ++ " > " ++ toString regression.bounds.resources))
  if regression.actionAttempts.length > regression.bounds.actions then
    throw (compileError .boundExceeded "actions"
      (toString regression.actionAttempts.length ++ " > " ++ toString regression.bounds.actions))
  if regression.ordering.length > regression.bounds.precedenceEdges then
    throw (compileError .boundExceeded "precedenceEdges"
      (toString regression.ordering.length ++ " > " ++ toString regression.bounds.precedenceEdges))

private def resolveResources
    (target : ModelTarget)
    (resources : List ResourceId) : Except CompileError ResolvedSetup := do
  let resolved ← resources.mapM fun id =>
    match target.resources.find? (fun binding => binding.id == id) with
    | some binding => .ok { id, value := binding.value }
    | none => .error (compileError .unresolvedResource id.value target.id.value)
  pure ⟨resolved⟩

private def resolveProperties
    (target : ModelTarget)
    (properties : List PropertyId) : Except CompileError (List ExpectedProperty) :=
  properties.mapM fun id =>
    match target.propertyObservations.find? (fun observation => observation.id == id) with
    | some observation => .ok { propertyId := id, observationContract := observation.contract }
    | none => .error (compileError .unresolvedProperty id.value target.id.value)

private def projectActions
    (target : ModelTarget)
    (setup : ResolvedSetup)
    (actions : List ActionId) : Except CompileError (List ProjectedOutcome) :=
  actions.mapM fun id =>
    match target.actionProjections.find? (fun projection => projection.id == id) with
    | none => .error (compileError .unmappedAction id.value target.id.value)
    | some projection =>
        match projection.project setup with
        | none => .error (compileError .impossibleAction id.value (canonicalResolvedSetup setup))
        | some outcome => .ok { actionId := id, outcome }

private def edgeSubject (edge : PrecedenceEdge) : String :=
  edge.before.value ++ "->" ++ edge.after.value

private def firstDuplicateEdge : List PrecedenceEdge → Option PrecedenceEdge
  | first :: second :: rest =>
      if first == second then some first else firstDuplicateEdge (second :: rest)
  | _ => none

private def resourceLe (left right : ResourceId) : Bool :=
  decide (left.value ≤ right.value)

private def actionLe (left right : ActionId) : Bool :=
  decide (left.value ≤ right.value)

private def propertyLe (left right : PropertyId) : Bool :=
  decide (left.value ≤ right.value)

private def edgeLe (left right : PrecedenceEdge) : Bool :=
  decide (left.before.value < right.before.value) ||
    (left.before == right.before && decide (left.after.value ≤ right.after.value))

private def reaches
    (edges : List PrecedenceEdge)
    (current goal : ActionId)
    (visited : List ActionId) : Nat → Bool
  | 0 => false
  | fuel + 1 =>
      if current == goal then
        true
      else if visited.contains current then
        false
      else
        edges.any fun edge =>
          edge.before == current && reaches edges edge.after goal (current :: visited) fuel

private def validateOrdering
    (actions : List ActionId)
    (ordering : List PrecedenceEdge) : Except CompileError (List PrecedenceEdge) := do
  let canonical := ordering.mergeSort edgeLe
  match firstDuplicateEdge canonical with
  | some edge => throw (compileError .duplicateOrdering (edgeSubject edge) "ordering")
  | none => pure ()
  match canonical.find? (fun edge => edge.before == edge.after) with
  | some edge => throw (compileError .selfOrdering (edgeSubject edge) "ordering")
  | none => pure ()
  for edge in canonical do
    if !actions.contains edge.before then
      throw (compileError .unresolvedAction edge.before.value "ordering")
    if !actions.contains edge.after then
      throw (compileError .unresolvedAction edge.after.value "ordering")
  match canonical.find? (fun edge => reaches canonical edge.after edge.before [] canonical.length) with
  | some edge => throw (compileError .cyclicOrdering (edgeSubject edge) "ordering")
  | none => pure canonical

def compile (target : ModelTarget) (regression : Regression) : Except CompileError ExperimentSpec := do
  validateIdentities target regression
  validateExpectations regression
  validateTarget target regression
  validateBounds regression
  let resources := regression.resources.mergeSort resourceLe
  let actions := regression.actionAttempts.mergeSort actionLe
  let properties := regression.expectedProperties.items.mergeSort propertyLe
  let setup ← resolveResources target resources
  let expectedProperties ← resolveProperties target properties
  let projectedOutcomes ← projectActions target setup actions
  let ordering ← validateOrdering actions regression.ordering
  let modelIdentity := deriveModelIdentity target.id target.declaration setup projectedOutcomes expectedProperties
  pure {
    formatVersion := "temporal-experiment/v1"
    regressionId := regression.id
    targetId := target.id
    modelIdentity
    resources
    resolvedSetup := setup
    actionAttempts := actions
    projectedOutcomes
    ordering
    expectedProperties
    bounds := regression.bounds
    omissions := regression.omissions.mergeSort
    provenance := target.provenance
  }

end Temporal.Experiment
