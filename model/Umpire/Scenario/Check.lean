import Umpire.Scenario

/-! Canonicalization and checking for authored Scenarios. -/

namespace Umpire

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def roleLe (left right : Scenario.Role) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def constraintLe (left right : SetupConstraint) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def occurrenceLe (left right : Scenario.Step) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def boundLe (left right : Scenario.Count) : Bool :=
  decide (left.action.value ≤ right.action.value)

private def orderLe (left right : Scenario.Order) : Bool :=
  decide (left.before.value < right.before.value) ||
    (left.before == right.before && decide (left.after.value ≤ right.after.value))

private def orderingEdge (ordering : Scenario.Order) : DefinitionGraph.Edge := {
  before := ordering.before
  after := ordering.after
}

private def occurrenceOrder (edge : DefinitionGraph.Edge) : Scenario.Order := {
  before := edge.before
  after := edge.after
}

private def bindingLe (left right : RoleBinding) : Bool :=
  decide (left.role.value ≤ right.role.value)

private def idListKey (ids : List DefinitionId) : String :=
  String.intercalate "\u001f" (ids.map DefinitionId.value)

private def idListLe (left right : List DefinitionId) : Bool :=
  decide (idListKey left ≤ idListKey right)

private def canonicalIdLists (lists : List (List DefinitionId)) : List (List DefinitionId) :=
  lists.mergeSort idListLe |>.eraseDups

private def maxRequiredOccurrences : Nat := 12

private def operandSortKey : SetupOperand → String
  | .role id => "role:" ++ quote id.value
  | .value value =>
      "value:" ++ quote value.definitionId.value ++ ":" ++ quote value.value

private def canonicalSetupConstraint (constraint : SetupConstraint) : SetupConstraint :=
  if operandSortKey constraint.left ≤ operandSortKey constraint.right then
    constraint
  else
    { constraint with left := constraint.right, right := constraint.left }

private def sourceJson (source : SourceLocation) : String :=
  "{\"path\":" ++ quote source.path ++
    ",\"line\":" ++ toString source.line ++
    ",\"column\":" ++ toString source.column ++
    ",\"provenance\":" ++ quote source.provenance ++ "}"

private def scenarioError
    (kind : ScenarioErrorKind)
    (owner : DefinitionId)
    (source : SourceLocation)
    (offendingValue : String)
    (relatedDefinitionIds : List DefinitionId := []) : ScenarioError := {
  kind
  definitionId := if owner.value == "" then
    DefinitionId.of "umpire.behavior.anonymous"
  else
    owner
  sourcePath := source.displayPath
  offendingValue
  relatedDefinitionIds := DefinitionId.canonicalSet relatedDefinitionIds
}

private def requireDefinitionId
    (owner : DefinitionId)
    (source : SourceLocation)
    (id : DefinitionId) : Except ScenarioError Unit :=
  match id.validate with
  | .error .empty =>
      .error (scenarioError .emptyDefinitionId owner source "<empty>" [id])
  | .error .malformed =>
      .error (scenarioError .invalidDefinitionId owner source id.value [id])
  | .ok () => .ok ()

private def requireUniqueIds
    (owner : DefinitionId)
    (source : SourceLocation)
    (ids : List DefinitionId) : Except ScenarioError Unit :=
  match DefinitionId.firstDuplicate ids with
  | some duplicate =>
      .error (scenarioError .duplicateDefinitionId owner source duplicate.value [duplicate])
  | none => .ok ()

private def findDefinition
    (context : ScenarioCheckContext)
    (id : DefinitionId) : Option DefinitionMetadata :=
  context.definitions.find? fun declaration => declaration.id == id

private def validateReferenceKind
    (context : ScenarioCheckContext)
    (owner : Scenario)
    (id : DefinitionId)
    (expected : DefinitionKind) : Except ScenarioError Unit := do
  requireDefinitionId owner.id owner.source id
  match findDefinition context id with
  | none =>
      throw (scenarioError .unknownReference owner.id owner.source id.value [id])
  | some metadata =>
      if metadata.kind != expected then
        throw (scenarioError .wrongReferenceKind owner.id owner.source
          (id.value ++ ": expected " ++ expected.name ++ ", found " ++ metadata.kind.name)
          [id])

private def findRole
    (roles : List Scenario.Role)
    (id : DefinitionId) : Option Scenario.Role :=
  roles.find? fun role => role.id == id

private def operandKind
    (context : ScenarioCheckContext)
    (owner : Scenario)
    (roles : List Scenario.Role) : SetupOperand → Except ScenarioError DefinitionKind
  | .role id =>
      match findRole roles id with
      | some role => pure role.valueKind
      | none => throw (scenarioError .invalidBinding owner.id owner.source id.value [id])
  | .value value => do
      requireDefinitionId owner.id owner.source value.definitionId
      match findDefinition context value.definitionId with
      | some metadata => pure metadata.kind
      | none =>
          throw (scenarioError .invalidBinding owner.id owner.source
            value.definitionId.value [value.definitionId])

private def validateSetupConstraint
    (context : ScenarioCheckContext)
    (owner : Scenario)
    (roles : List Scenario.Role)
    (constraint : SetupConstraint) : Except ScenarioError Unit := do
  requireDefinitionId owner.id owner.source constraint.id
  let leftKind ← operandKind context owner roles constraint.left
  let rightKind ← operandKind context owner roles constraint.right
  if leftKind != rightKind then
    throw (scenarioError .invalidBinding owner.id owner.source
      (leftKind.name ++ " != " ++ rightKind.name) [constraint.id])

private def validateOrdering
    (owner : Scenario)
    (analysis : DefinitionGraph.Analysis) : Except ScenarioError (List Scenario.Order) := do
  match analysis.edgeFindings.duplicate with
  | some edge =>
      throw (scenarioError .duplicateOrdering owner.id owner.source
        (edge.before.value ++ "->" ++ edge.after.value) [edge.before, edge.after])
  | none => pure ()
  match analysis.edgeFindings.self with
  | some edge =>
      throw (scenarioError .selfOrdering owner.id owner.source edge.before.value [edge.before])
  | none => pure ()
  match analysis.edgeFindings.unknownEndpoints with
  | finding :: _ =>
      let unknown := if !finding.beforeKnown then finding.edge.before else finding.edge.after
      throw (scenarioError .unknownReference owner.id owner.source unknown.value [unknown])
  | [] => pure ()
  match analysis.cycleEvidence with
  | some evidence =>
      let witness := evidence.residualPredecessorWitness
      throw (scenarioError .cyclicOrdering owner.id owner.source witness.value [witness])
  | none => pure (analysis.canonicalEdges.map occurrenceOrder)

private def validateBinding
    (context : ScenarioCheckContext)
    (owner : Scenario)
    (roles : List Scenario.Role)
    (binding : RoleBinding) : Except ScenarioError Unit := do
  let role ← match findRole roles binding.role with
    | some role => pure role
    | none =>
        throw (scenarioError .invalidBinding owner.id owner.source
          binding.role.value [binding.role])
  requireDefinitionId owner.id owner.source binding.value.definitionId
  match findDefinition context binding.value.definitionId with
  | none =>
      throw (scenarioError .invalidBinding owner.id owner.source
        binding.value.definitionId.value [binding.role, binding.value.definitionId])
  | some metadata =>
      if metadata.kind != role.valueKind then
        throw (scenarioError .invalidBinding owner.id owner.source
          (binding.role.value ++ ": expected " ++ role.valueKind.name ++
            ", found " ++ metadata.kind.name)
          [binding.role, binding.value.definitionId])

private def requireModelValueKind
    (context : ScenarioCheckContext)
    (owner : Scenario)
    (expected : DefinitionKind)
    (value : ModelValue) : Except ScenarioError Unit :=
  validateReferenceKind context owner value.definitionId expected

private def checkExactTrace
    (context : ScenarioCheckContext)
    (owner : Scenario)
    (roles : List Scenario.Role)
    (authored : AuthoredExactTrace) : Except ScenarioError Scenario.Trace := do
  requireUniqueIds owner.id owner.source (authored.setup.map RoleBinding.role)
  for binding in authored.setup do
    validateBinding context owner roles binding
  let boundRoles := authored.setup.map RoleBinding.role
  for role in roles do
    if !boundRoles.contains role.id then
      throw (scenarioError .incompleteExactTrace owner.id owner.source
        ("missing setup binding " ++ role.id.value) [role.id])
  if authored.setup.length != roles.length then
    throw (scenarioError .invalidBinding owner.id owner.source "unexpected setup binding" boundRoles)
  let initialState ← match authored.initialState with
    | some state => pure state
    | none =>
        throw (scenarioError .incompleteExactTrace owner.id owner.source
          "initial-state" [])
  requireModelValueKind context owner .state initialState
  let mut steps := []
  for (authoredStep, index) in authored.steps.zipIdx do
    let action ← match authoredStep.selectedAction with
      | some value => pure value
      | none =>
          throw (scenarioError .incompleteExactTrace owner.id owner.source
            ("step-" ++ toString index ++ ":selected-action") [])
    let outcome ← match authoredStep.outcome with
      | some value => pure value
      | none =>
          throw (scenarioError .incompleteExactTrace owner.id owner.source
            ("step-" ++ toString index ++ ":model-outcome") [])
    let state ← match authoredStep.resultingState with
      | some value => pure value
      | none =>
          throw (scenarioError .incompleteExactTrace owner.id owner.source
            ("step-" ++ toString index ++ ":resulting-state") [])
    let facts ← match authoredStep.observations with
      | some values => pure values
      | none =>
          throw (scenarioError .incompleteExactTrace owner.id owner.source
            ("step-" ++ toString index ++ ":observations") [])
    requireModelValueKind context owner .action action
    requireModelValueKind context owner .outcome outcome
    requireModelValueKind context owner .state state
    for fact in facts do
      requireModelValueKind context owner .fact fact
    steps := steps ++ [{
      selectedAction := action
      outcome := outcome
      state := state
      facts
    }]
  pure {
    setup := authored.setup.mergeSort bindingLe
    trace := { initialState, steps }
  }

private def countAction (action : DefinitionId) (actions : List DefinitionId) : Nat :=
  (actions.filter fun candidate => candidate == action).length

private def occurrenceIsReady
    (ordering : List Scenario.Order)
    (remaining : List Scenario.Step)
    (occurrence : Scenario.Step) : Bool :=
  ordering.all fun edge =>
    edge.after != occurrence.id ||
      !(remaining.any fun candidate => candidate.id == edge.before)

private structure OccurrenceAssignmentState where
  remaining : List Scenario.Step
  assignedRev : List (Option Scenario.Step)

private def insertOccurrenceState
    (states : List OccurrenceAssignmentState)
    (candidate : OccurrenceAssignmentState) : List OccurrenceAssignmentState :=
  if states.any fun state => state.remaining == candidate.remaining then
    states
  else
    states ++ [candidate]

private def advanceOccurrenceStates
    (ordering : List Scenario.Order)
    (action : DefinitionId)
    (states : List OccurrenceAssignmentState) : List OccurrenceAssignmentState :=
  states.foldl (init := []) fun next state =>
    let assignable := state.remaining.filter (fun occurrence =>
      occurrence.action == action && occurrenceIsReady ordering state.remaining occurrence)
      |>.mergeSort occurrenceLe
    let assigned := assignable.map fun occurrence => {
      remaining := state.remaining.erase occurrence
      assignedRev := some occurrence :: state.assignedRev
    }
    let skipped := { state with assignedRev := none :: state.assignedRev }
    (assigned ++ [skipped]).foldl insertOccurrenceState next

/--
Track canonical remaining-occurrence sets across the schedule. Deduplication makes equivalent
assignment permutations one state, avoiding factorial backtracking for repeated action labels.
-/
private def assignOccurrenceSlots
    (schedule : List DefinitionId)
    (ordering : List Scenario.Order)
    (occurrences : List Scenario.Step) : Option (List (Option Scenario.Step)) :=
  let countsSufficient := occurrences.all fun occurrence =>
    countAction occurrence.action schedule ≥
      (occurrences.filter fun candidate => candidate.action == occurrence.action).length
  if !countsSufficient then
    none
  else
    let initial : OccurrenceAssignmentState := {
      remaining := occurrences.mergeSort occurrenceLe
      assignedRev := []
    }
    let states := schedule.foldl (init := [initial]) fun states action =>
      advanceOccurrenceStates ordering action states
    (states.find? fun state => state.remaining.isEmpty).map fun state => state.assignedRev.reverse

private def hasOccurrenceAssignment
    (schedule : List DefinitionId)
    (ordering : List Scenario.Order)
    (occurrences : List Scenario.Step) : Bool :=
  (assignOccurrenceSlots schedule ordering occurrences).isSome

private def isSubsequence : List DefinitionId → List DefinitionId → Bool
  | [], _ => true
  | _, [] => false
  | expected :: rest, actual :: remaining =>
      if expected == actual then
        isSubsequence rest remaining
      else
        isSubsequence (expected :: rest) remaining

private def isPrefix : List DefinitionId → List DefinitionId → Bool
  | [], _ => true
  | _, [] => false
  | expected :: rest, actual :: remaining =>
      expected == actual && isPrefix rest remaining

private def containsAdjacent (expected : List DefinitionId) : List DefinitionId → Bool
  | [] => expected == []
  | actual@(_ :: remaining) => isPrefix expected actual || containsAdjacent expected remaining

private def validateActionConstraints
    (owner : Scenario)
    (allowed forbidden : List DefinitionId)
    (required : List Scenario.Step)
    (bounds : List Scenario.Count)
    (ordering : List Scenario.Order)
    (sequences adjacencies : List (List DefinitionId))
    (exactSchedule : Option (List DefinitionId)) : Except ScenarioError Unit := do
  for occurrence in required do
    if forbidden.contains occurrence.action then
      throw (scenarioError .forbiddenRequired owner.id owner.source
        occurrence.action.value [occurrence.id, occurrence.action])
    if allowed != [] && !allowed.contains occurrence.action then
      throw (scenarioError .contradictoryConstraint owner.id owner.source
        ("required action not allowed: " ++ occurrence.action.value)
        [occurrence.id, occurrence.action])
  for action in allowed do
    if forbidden.contains action then
      throw (scenarioError .contradictoryConstraint owner.id owner.source
        ("allowed and forbidden: " ++ action.value) [action])
  for bound in bounds do
    match bound.maximum with
    | some maximum =>
        if bound.minimum > maximum then
          throw (scenarioError .contradictoryOccurrenceBounds owner.id owner.source
            bound.action.value [bound.action])
    | none => pure ()
    let requiredCount := (required.filter fun occurrence => occurrence.action == bound.action).length
    match bound.maximum with
    | some maximum =>
        if requiredCount > maximum then
          throw (scenarioError .contradictoryOccurrenceBounds owner.id owner.source
            bound.action.value [bound.action])
    | none => pure ()
    if forbidden.contains bound.action && bound.minimum > 0 then
      throw (scenarioError .forbiddenRequired owner.id owner.source
        bound.action.value [bound.action])
  match exactSchedule with
  | none => pure ()
  | some actions =>
      for action in actions do
        if forbidden.contains action then
          throw (scenarioError .forbiddenRequired owner.id owner.source action.value [action])
        if allowed != [] && !allowed.contains action then
          throw (scenarioError .contradictoryConstraint owner.id owner.source
            ("exact action not allowed: " ++ action.value) [action])
      for occurrence in required do
        let requiredCount := (required.filter fun candidate =>
          candidate.action == occurrence.action).length
        if countAction occurrence.action actions < requiredCount then
          throw (scenarioError .contradictoryOccurrenceBounds owner.id owner.source
            occurrence.action.value [occurrence.action])
      for bound in bounds do
        let count := countAction bound.action actions
        if count < bound.minimum || bound.maximum.any fun maximum => count > maximum then
          throw (scenarioError .contradictoryOccurrenceBounds owner.id owner.source
            bound.action.value [bound.action])
      if !hasOccurrenceAssignment actions ordering required then
        throw (scenarioError .contradictoryConstraint owner.id owner.source
          "exact schedule violates occurrence ordering" (required.map Scenario.Step.id))
      for sequence in sequences do
        if !isSubsequence sequence actions then
          throw (scenarioError .contradictoryConstraint owner.id owner.source
            ("exact schedule omits sequence: " ++ idListKey sequence) sequence)
      for adjacency in adjacencies do
        if !containsAdjacent adjacency actions then
          throw (scenarioError .contradictoryConstraint owner.id owner.source
            ("exact schedule omits adjacency: " ++ idListKey adjacency) adjacency)

private def operandJson : SetupOperand → String
  | .role id => "{\"role\":" ++ quote id.value ++ "}"
  | .value value =>
      "{\"value\":{\"identity\":" ++ quote value.definitionId.value ++
        ",\"value\":" ++ quote value.value ++ "}}"

private def roleJson (role : Scenario.Role) : String :=
  "{\"id\":" ++ quote role.id.value ++
    ",\"valueKind\":" ++ quote role.valueKind.name ++ "}"

private def setupConstraintJson (constraint : SetupConstraint) : String :=
  "{\"id\":" ++ quote constraint.id.value ++
    ",\"relation\":" ++ quote constraint.relation.name ++
    ",\"left\":" ++ operandJson constraint.left ++
    ",\"right\":" ++ operandJson constraint.right ++ "}"

private def occurrenceJson (occurrence : Scenario.Step) : String :=
  "{\"id\":" ++ quote occurrence.id.value ++
    ",\"action\":" ++ quote occurrence.action.value ++ "}"

private def boundJson (bound : Scenario.Count) : String :=
  "{\"action\":" ++ quote bound.action.value ++
    ",\"minimum\":" ++ toString bound.minimum ++
    ",\"maximum\":" ++ (bound.maximum.map toString |>.getD "null") ++ "}"

private def orderJson (edge : Scenario.Order) : String :=
  "{\"before\":" ++ quote edge.before.value ++
    ",\"after\":" ++ quote edge.after.value ++ "}"

private def valueJson (value : ModelValue) : String :=
  "{\"identity\":" ++ quote value.definitionId.value ++
    ",\"value\":" ++ quote value.value ++ "}"

private def bindingJson (binding : RoleBinding) : String :=
  "{\"role\":" ++ quote binding.role.value ++
    ",\"value\":" ++ valueJson binding.value ++ "}"

private def traceStepJson
    (step : ModelTraceStep ModelValue ModelValue ModelValue ModelValue) : String :=
  "{\"selectedAction\":" ++ valueJson step.selectedAction ++
    ",\"outcome\":" ++ valueJson step.outcome ++
    ",\"state\":" ++ valueJson step.state ++
    ",\"facts\":" ++ array (step.facts.map valueJson) ++ "}"

private def behaviorTraceJson (trace : Scenario.Trace) : String :=
  "{\"setup\":" ++ array (trace.setup.mergeSort bindingLe |>.map bindingJson) ++
    ",\"initialState\":" ++ valueJson trace.trace.initialState ++
    ",\"steps\":" ++ array (trace.trace.steps.map traceStepJson) ++ "}"

private def actionListJson (actions : List DefinitionId) : String :=
  array (actions.map (quote ∘ DefinitionId.value))

private def equalNeighbors
    (constraints : List SetupConstraint)
    (operand : SetupOperand) : List SetupOperand :=
  constraints.flatMap fun constraint =>
    if constraint.relation != .equal then
      []
    else if constraint.left == operand then
      [constraint.right]
    else if constraint.right == operand then
      [constraint.left]
    else
      []

private def setupOperands (constraints : List SetupConstraint) : List SetupOperand :=
  constraints.flatMap (fun constraint => [constraint.left, constraint.right]) |>.eraseDups

private def operandsConnected
    (constraints : List SetupConstraint)
    (left right : SetupOperand) : Bool :=
  let rec visit (pending visited : List SetupOperand) : Nat → Bool
    | 0 => false
    | fuel + 1 =>
        match pending with
        | [] => false
        | current :: rest =>
            if current == right then
              true
            else if visited.contains current then
              visit rest visited fuel
            else
              visit (rest ++ equalNeighbors constraints current) (current :: visited) fuel
  let operandCount := setupOperands constraints |>.length
  visit [left] [] ((operandCount + 1) * (constraints.length + 1))

private def setupUnsatisfiable (constraints : List SetupConstraint) : Bool :=
  let unequalConflict := constraints.any fun constraint =>
    constraint.relation == .different &&
      operandsConnected constraints constraint.left constraint.right
  let literalOperands := (setupOperands constraints).filterMap fun operand =>
    match operand with
    | .value value => some (operand, value)
    | .role _ => none
  let literalConflict := literalOperands.any fun left =>
    literalOperands.any fun right =>
      left.2 != right.2 && operandsConnected constraints left.1 right.1
  unequalConflict || literalConflict

private def behaviorSemanticJson
    (id : DefinitionId)
    (version : Nat)
    (requires : List DefinitionId)
    (roles : List Scenario.Role)
    (setup : List SetupConstraint)
    (allowedActions : List DefinitionId)
    (requiredOccurrences : List Scenario.Step)
    (forbiddenActions : List DefinitionId)
    (occurrenceBounds : List Scenario.Count)
    (ordering : List Scenario.Order)
    (sequences adjacencies : List (List DefinitionId))
    (actionsExactly : Option (List DefinitionId))
    (traceExactly : Option Scenario.Trace)
    (spaceStatus : ScenarioStatus) : String :=
  "{\"id\":" ++ quote id.value ++
    ",\"version\":" ++ toString version ++
    ",\"requires\":" ++ actionListJson (DefinitionId.canonicalSet requires) ++
    ",\"roles\":" ++ array (roles.mergeSort roleLe |>.map roleJson) ++
    ",\"setup\":" ++ array (setup.mergeSort constraintLe |>.map setupConstraintJson) ++
    ",\"allowedActions\":" ++ actionListJson (DefinitionId.canonicalSet allowedActions) ++
    ",\"requiredOccurrences\":" ++
      array (requiredOccurrences.mergeSort occurrenceLe |>.map occurrenceJson) ++
    ",\"forbiddenActions\":" ++ actionListJson (DefinitionId.canonicalSet forbiddenActions) ++
    ",\"occurrenceBounds\":" ++
      array (occurrenceBounds.mergeSort boundLe |>.map boundJson) ++
    ",\"ordering\":" ++ array (ordering.mergeSort orderLe |>.map orderJson) ++
    ",\"sequences\":" ++ array (canonicalIdLists sequences |>.map actionListJson) ++
    ",\"adjacencies\":" ++ array (canonicalIdLists adjacencies |>.map actionListJson) ++
    ",\"actionsExactly\":" ++ (actionsExactly.map actionListJson |>.getD "null") ++
    ",\"traceExactly\":" ++ (traceExactly.map behaviorTraceJson |>.getD "null") ++
    ",\"spaceStatus\":" ++ quote spaceStatus.name ++ "}"

def canonicalScenarioJson (behavior : CheckedScenario) : String :=
  "{\"semantic\":" ++ behaviorSemanticJson behavior.id behavior.version behavior.requires
      behavior.roles behavior.setup behavior.allowedActions behavior.requiredOccurrences
      behavior.forbiddenActions behavior.occurrenceBounds behavior.ordering behavior.sequences
      behavior.adjacencies behavior.actionsExactly behavior.traceExactly behavior.spaceStatus ++
    ",\"source\":" ++ sourceJson behavior.source ++
    ",\"documentation\":" ++ quote behavior.documentation ++ "}"

def canonicalScenarioErrorJson (error : ScenarioError) : String :=
  "{\"kind\":" ++ quote error.kind.name ++
    ",\"definitionId\":" ++ quote error.definitionId.value ++
    ",\"sourcePath\":" ++ quote error.sourcePath ++
    ",\"offendingValue\":" ++ quote error.offendingValue ++
    ",\"relatedDefinitionIds\":" ++
      array (DefinitionId.canonicalSet error.relatedDefinitionIds |>.map
        (quote ∘ DefinitionId.value)) ++ "}"

/-- Check and canonicalize a behavior without selecting a target or enumerating any trace. -/
def Scenario.check
    (context : ScenarioCheckContext)
    (declaration : Scenario) : Except ScenarioError CheckedScenario := do
  requireDefinitionId declaration.id declaration.source declaration.id
  requireUniqueIds declaration.id declaration.source declaration.requires
  requireUniqueIds declaration.id declaration.source (declaration.roles.map Scenario.Role.id)
  requireUniqueIds declaration.id declaration.source (declaration.setup.map SetupConstraint.id)
  requireUniqueIds declaration.id declaration.source declaration.allowedActions
  requireUniqueIds declaration.id declaration.source declaration.forbiddenActions
  let orderingAnalysis := DefinitionGraph.analyze
    (declaration.requiredOccurrences.map Scenario.Step.id)
    (declaration.ordering.map orderingEdge)
  match orderingAnalysis.nodeFindings.duplicate with
  | some duplicate =>
      throw (scenarioError .duplicateDefinitionId declaration.id declaration.source
        duplicate.value [duplicate])
  | none => pure ()
  requireUniqueIds declaration.id declaration.source
    (declaration.occurrenceBounds.map Scenario.Count.action)
  for capability in declaration.requires do
    validateReferenceKind context declaration capability .capability
  let roles := declaration.roles.mergeSort roleLe
  for role in roles do
    requireDefinitionId declaration.id declaration.source role.id
  let setup := declaration.setup.map canonicalSetupConstraint |>.mergeSort constraintLe
  for constraint in setup do
    validateSetupConstraint context declaration roles constraint
  let allowed := DefinitionId.canonicalSet declaration.allowedActions
  let forbidden := DefinitionId.canonicalSet declaration.forbiddenActions
  let required := declaration.requiredOccurrences.mergeSort occurrenceLe
  if required.length > maxRequiredOccurrences then
    throw (scenarioError .occurrenceLimitExceeded declaration.id declaration.source
      (toString required.length ++ " > " ++ toString maxRequiredOccurrences)
      (required.map Scenario.Step.id))
  let bounds := declaration.occurrenceBounds.mergeSort boundLe
  for action in allowed ++ forbidden ++ required.map Scenario.Step.action ++
      bounds.map Scenario.Count.action do
    validateReferenceKind context declaration action .action
  for actions in declaration.sequences ++ declaration.adjacencies do
    for action in actions do
      validateReferenceKind context declaration action .action
  match declaration.actionsExactly with
  | some actions =>
      for action in actions do
        validateReferenceKind context declaration action .action
  | none => pure ()
  let ordering ← validateOrdering declaration orderingAnalysis
  let exactTrace ← declaration.traceExactly.mapM (checkExactTrace context declaration roles)
  let exactSchedule := declaration.actionsExactly <|>
    exactTrace.map fun trace => trace.trace.steps.map
      (fun step => step.selectedAction.definitionId)
  let sequences := canonicalIdLists declaration.sequences
  let adjacencies := canonicalIdLists declaration.adjacencies
  validateActionConstraints declaration allowed forbidden required bounds ordering sequences
    adjacencies exactSchedule
  match declaration.actionsExactly, exactTrace with
  | some actions, some trace =>
      let traceActions := trace.trace.steps.map fun step => step.selectedAction.definitionId
      if actions != traceActions then
        throw (scenarioError .contradictoryConstraint declaration.id declaration.source
          "actionsExactly != traceExactly actions" actions)
  | _, _ => pure ()
  let status := if setupUnsatisfiable setup then
    .unsatisfiable
  else
    .unclassified
  let semantic := behaviorSemanticJson declaration.id declaration.version declaration.requires
    roles setup allowed required forbidden bounds ordering sequences adjacencies
    declaration.actionsExactly exactTrace status
  let checked : CheckedScenario := {
    id := declaration.id
    source := declaration.source
    version := declaration.version
    requires := DefinitionId.canonicalSet declaration.requires
    roles
    setup
    allowedActions := allowed
    requiredOccurrences := required
    forbiddenActions := forbidden
    occurrenceBounds := bounds
    ordering
    sequences
    adjacencies
    actionsExactly := declaration.actionsExactly
    traceExactly := exactTrace
    spaceStatus := status
    documentation := declaration.documentation
    canonicalMetadata := ""
    behaviorFingerprint := behaviorFingerprintOf semantic
  }
  pure { checked with canonicalMetadata := canonicalScenarioJson checked }

/-- Produce a checked Behavior directly from an explicit proof that the typed checker succeeds.
Use `Scenario.check` when an invalid declaration's typed diagnostic is needed. -/
def Scenario.checked
    (context : ScenarioCheckContext)
    (declaration : Scenario)
    (valid : (Scenario.check context declaration).toOption.isSome = true) : CheckedScenario :=
  (Scenario.check context declaration).toOption.get valid

private def bindingFor (bindings : List RoleBinding) (role : DefinitionId) : Option ModelValue :=
  (bindings.find? fun binding => binding.role == role).map RoleBinding.value

private def resolveOperand (bindings : List RoleBinding) : SetupOperand → Option ModelValue
  | .role id => bindingFor bindings id
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

private def setupIsComplete (roles : List Scenario.Role) (bindings : List RoleBinding) : Bool :=
  bindings.length == roles.length &&
    roles.all (fun role => countAction role.id (bindings.map RoleBinding.role) == 1) &&
    bindings.all (fun binding => roles.any fun role => role.id == binding.role)

private def traceActions (trace : Scenario.Trace) : List DefinitionId :=
  trace.trace.steps.map fun step => step.selectedAction.definitionId

private def normalizedTrace (trace : Scenario.Trace) : Scenario.Trace :=
  { trace with setup := trace.setup.mergeSort bindingLe }

/-- Canonically attribute selected action positions to authored required occurrences. -/
def CheckedScenario.assignOccurrences
    (behavior : CheckedScenario)
    (schedule : List DefinitionId) : Option (List (Option Scenario.Step)) :=
  assignOccurrenceSlots schedule behavior.ordering behavior.requiredOccurrences

/-- Membership is a pure predicate over already semantic, target-owned trace data. -/
def CheckedScenario.admits (behavior : CheckedScenario) (candidate : Scenario.Trace) : Bool :=
  if behavior.spaceStatus == .unsatisfiable then
    false
  else
    let actions := traceActions candidate
    setupIsComplete behavior.roles candidate.setup &&
      behavior.setup.all (setupConstraintHolds candidate.setup) &&
      (behavior.allowedActions == [] ||
        actions.all fun action => behavior.allowedActions.contains action) &&
      actions.all (fun action => !behavior.forbiddenActions.contains action) &&
      behavior.occurrenceBounds.all (fun bound =>
        let count := countAction bound.action actions
        count ≥ bound.minimum && bound.maximum.all fun maximum => count ≤ maximum) &&
      hasOccurrenceAssignment actions behavior.ordering behavior.requiredOccurrences &&
      behavior.sequences.all (fun sequence => isSubsequence sequence actions) &&
      behavior.adjacencies.all (fun adjacency => containsAdjacent adjacency actions) &&
      behavior.actionsExactly.all (fun exact => actions == exact) &&
      behavior.traceExactly.all (fun exact => normalizedTrace candidate == exact)

def CheckedScenario.isUnsatisfiable (behavior : CheckedScenario) : Bool :=
  behavior.spaceStatus == .unsatisfiable

/-- The checker's diagnostic for an authored Scenario, or `none` when it is admitted. -/
def Scenario.error?
    (context : ScenarioCheckContext)
    (scenario : Scenario) : Option ScenarioError :=
  match Scenario.check context scenario with
  | .error error => some error
  | .ok _ => none

end Umpire
