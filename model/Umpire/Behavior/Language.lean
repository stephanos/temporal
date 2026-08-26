import Umpire.Core

/-! Implementation behind the `Umpire.Behavior` public facade. -/

namespace Umpire

/-! Checked, portable constraints over pure semantic traces. -/

inductive BehaviorErrorKind where
  | emptyIdentity
  | invalidIdentity
  | duplicateIdentity
  | unknownReference
  | wrongReferenceKind
  | invalidBinding
  | contradictoryOccurrenceBounds
  | contradictoryConstraint
  | forbiddenRequired
  | duplicateOrdering
  | selfOrdering
  | cyclicOrdering
  | occurrenceLimitExceeded
  | incompleteExactTrace
  deriving BEq, DecidableEq, Ord, Repr

def BehaviorErrorKind.name : BehaviorErrorKind → String
  | .emptyIdentity => "empty-identity"
  | .invalidIdentity => "invalid-identity"
  | .duplicateIdentity => "duplicate-identity"
  | .unknownReference => "unknown-reference"
  | .wrongReferenceKind => "wrong-reference-kind"
  | .invalidBinding => "invalid-binding"
  | .contradictoryOccurrenceBounds => "contradictory-occurrence-bounds"
  | .contradictoryConstraint => "contradictory-constraint"
  | .forbiddenRequired => "forbidden-required"
  | .duplicateOrdering => "duplicate-ordering"
  | .selfOrdering => "self-ordering"
  | .cyclicOrdering => "cyclic-ordering"
  | .occurrenceLimitExceeded => "occurrence-limit-exceeded"
  | .incompleteExactTrace => "incomplete-exact-trace"

structure BehaviorError where
  kind : BehaviorErrorKind
  declarationId : DeclarationId
  sourcePath : String
  offendingValue : String
  relatedIdentities : List DeclarationId
  deriving BEq, DecidableEq, Repr

/-- A symbolic setup role retains the kind of semantic value that may bind it. -/
structure ResourceRole where
  id : DeclarationId
  valueKind : DeclarationKind
  deriving BEq, DecidableEq, Repr

structure RoleBinding where
  role : DeclarationId
  value : SemanticValue
  deriving BEq, DecidableEq, Repr

inductive SetupOperand where
  | role (id : DeclarationId)
  | value (value : SemanticValue)
  deriving BEq, DecidableEq, Repr

inductive SetupRelation where
  | equal
  | different
  deriving BEq, DecidableEq, Ord, Repr

def SetupRelation.name : SetupRelation → String
  | .equal => "equal"
  | .different => "different"

structure SetupConstraint where
  id : DeclarationId
  relation : SetupRelation
  left : SetupOperand
  right : SetupOperand
  deriving BEq, DecidableEq, Repr

/-- A required action occurrence has a stable identity independent of its action identity. -/
structure NamedOccurrence where
  id : DeclarationId
  action : DeclarationId
  deriving BEq, DecidableEq, Repr

structure OccurrenceBound where
  action : DeclarationId
  minimum : Nat := 0
  maximum : Option Nat := none
  deriving BEq, DecidableEq, Repr

namespace OccurrenceBound

def exactly (action : DeclarationId) (count : Nat) : OccurrenceBound :=
  { action, minimum := count, maximum := some count }

def atLeast (action : DeclarationId) (count : Nat) : OccurrenceBound :=
  { action, minimum := count }

def atMost (action : DeclarationId) (count : Nat) : OccurrenceBound :=
  { action, maximum := some count }

end OccurrenceBound

structure OccurrenceOrder where
  before : DeclarationId
  after : DeclarationId
  deriving BEq, DecidableEq, Repr

/-- Optional fields keep malformed promoted witnesses representable until checking. -/
structure AuthoredExactTraceStep where
  selectedAction : Option SemanticValue
  modelOutcome : Option SemanticValue
  resultingState : Option SemanticValue
  observations : Option (List SemanticValue)
  deriving BEq, DecidableEq, Repr

structure AuthoredExactTrace where
  setup : List RoleBinding
  initialState : Option SemanticValue
  steps : List AuthoredExactTraceStep
  deriving BEq, DecidableEq, Repr

/-- A complete pure trace together with the symbolic setup bindings that selected it. -/
structure BehaviorTrace where
  setup : List RoleBinding
  trace : SemanticTrace SemanticValue SemanticValue SemanticValue SemanticValue
  deriving BEq, DecidableEq, Repr

structure BehaviorDeclaration where
  id : DeclarationId
  source : SemanticSource
  version : Nat := 1
  requires : List DeclarationId := []
  roles : List ResourceRole := []
  setup : List SetupConstraint := []
  allowedActions : List DeclarationId := []
  requiredOccurrences : List NamedOccurrence := []
  forbiddenActions : List DeclarationId := []
  occurrenceBounds : List OccurrenceBound := []
  ordering : List OccurrenceOrder := []
  sequences : List (List DeclarationId) := []
  adjacencies : List (List DeclarationId) := []
  actionsExactly : Option (List DeclarationId) := none
  traceExactly : Option AuthoredExactTrace := none
  documentation : String := ""
  deriving BEq, DecidableEq, Repr

structure BehaviorCheckContext where
  declarations : List DeclarationMetadata
  deriving BEq, DecidableEq, Repr

inductive BehaviorSpaceStatus where
  | unclassified
  | unsatisfiable
  deriving BEq, DecidableEq, Ord, Repr

def BehaviorSpaceStatus.name : BehaviorSpaceStatus → String
  | .unclassified => "unclassified"
  | .unsatisfiable => "unsatisfiable"

structure CheckedBehavior where
  id : DeclarationId
  source : SemanticSource
  version : Nat
  requires : List DeclarationId
  roles : List ResourceRole
  setup : List SetupConstraint
  allowedActions : List DeclarationId
  requiredOccurrences : List NamedOccurrence
  forbiddenActions : List DeclarationId
  occurrenceBounds : List OccurrenceBound
  ordering : List OccurrenceOrder
  sequences : List (List DeclarationId)
  adjacencies : List (List DeclarationId)
  actionsExactly : Option (List DeclarationId)
  traceExactly : Option BehaviorTrace
  spaceStatus : BehaviorSpaceStatus
  documentation : String
  canonicalMetadata : String
  semanticDigest : String
  deriving BEq, DecidableEq, Repr

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def idLe (left right : DeclarationId) : Bool :=
  decide (left.value ≤ right.value)

private def roleLe (left right : ResourceRole) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def constraintLe (left right : SetupConstraint) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def occurrenceLe (left right : NamedOccurrence) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def boundLe (left right : OccurrenceBound) : Bool :=
  decide (left.action.value ≤ right.action.value)

private def orderLe (left right : OccurrenceOrder) : Bool :=
  decide (left.before.value < right.before.value) ||
    (left.before == right.before && decide (left.after.value ≤ right.after.value))

private def bindingLe (left right : RoleBinding) : Bool :=
  decide (left.role.value ≤ right.role.value)

private def idListKey (ids : List DeclarationId) : String :=
  String.intercalate "\u001f" (ids.map DeclarationId.value)

private def idListLe (left right : List DeclarationId) : Bool :=
  decide (idListKey left ≤ idListKey right)

private def canonicalIds (ids : List DeclarationId) : List DeclarationId :=
  ids.mergeSort idLe |>.eraseDups

private def canonicalIdLists (lists : List (List DeclarationId)) : List (List DeclarationId) :=
  lists.mergeSort idListLe |>.eraseDups

private def maxRequiredOccurrences : Nat := 12

private def operandSortKey : SetupOperand → String
  | .role id => "role:" ++ quote id.value
  | .value value =>
      "value:" ++ quote value.identity.value ++ ":" ++ quote value.value

private def canonicalSetupConstraint (constraint : SetupConstraint) : SetupConstraint :=
  if operandSortKey constraint.left ≤ operandSortKey constraint.right then
    constraint
  else
    { constraint with left := constraint.right, right := constraint.left }

private def sourcePath (source : SemanticSource) : String :=
  if source.path == "" then "<unknown>" else source.path

private def sourceJson (source : SemanticSource) : String :=
  "{\"path\":" ++ quote source.path ++
    ",\"line\":" ++ toString source.line ++
    ",\"column\":" ++ toString source.column ++
    ",\"provenance\":" ++ quote source.provenance ++ "}"

private def behaviorError
    (kind : BehaviorErrorKind)
    (owner : DeclarationId)
    (source : SemanticSource)
    (offendingValue : String)
    (relatedIdentities : List DeclarationId := []) : BehaviorError := {
  kind
  declarationId := if owner.value == "" then
    DeclarationId.of "umpire.behavior.anonymous"
  else
    owner
  sourcePath := sourcePath source
  offendingValue
  relatedIdentities := canonicalIds relatedIdentities
}

private def firstDuplicateId : List DeclarationId → Option DeclarationId
  | first :: second :: rest =>
      if first == second then some first else firstDuplicateId (second :: rest)
  | _ => none

private def firstDuplicateOrder : List OccurrenceOrder → Option OccurrenceOrder
  | first :: second :: rest =>
      if first == second then some first else firstDuplicateOrder (second :: rest)
  | _ => none

private def requireIdentity
    (owner : DeclarationId)
    (source : SemanticSource)
    (id : DeclarationId) : Except BehaviorError Unit :=
  if id.value == "" then
    .error (behaviorError .emptyIdentity owner source "<empty>" [id])
  else if !id.isNamespaced then
    .error (behaviorError .invalidIdentity owner source id.value [id])
  else
    .ok ()

private def requireUniqueIds
    (owner : DeclarationId)
    (source : SemanticSource)
    (ids : List DeclarationId) : Except BehaviorError Unit :=
  match firstDuplicateId (ids.mergeSort idLe) with
  | some duplicate =>
      .error (behaviorError .duplicateIdentity owner source duplicate.value [duplicate])
  | none => .ok ()

private def findDeclaration
    (context : BehaviorCheckContext)
    (id : DeclarationId) : Option DeclarationMetadata :=
  context.declarations.find? fun declaration => declaration.id == id

private def validateReferenceKind
    (context : BehaviorCheckContext)
    (owner : BehaviorDeclaration)
    (id : DeclarationId)
    (expected : DeclarationKind) : Except BehaviorError Unit := do
  requireIdentity owner.id owner.source id
  match findDeclaration context id with
  | none =>
      throw (behaviorError .unknownReference owner.id owner.source id.value [id])
  | some metadata =>
      if metadata.kind != expected then
        throw (behaviorError .wrongReferenceKind owner.id owner.source
          (id.value ++ ": expected " ++ expected.name ++ ", found " ++ metadata.kind.name)
          [id])

private def findRole
    (roles : List ResourceRole)
    (id : DeclarationId) : Option ResourceRole :=
  roles.find? fun role => role.id == id

private def operandKind
    (context : BehaviorCheckContext)
    (owner : BehaviorDeclaration)
    (roles : List ResourceRole) : SetupOperand → Except BehaviorError DeclarationKind
  | .role id =>
      match findRole roles id with
      | some role => pure role.valueKind
      | none => throw (behaviorError .invalidBinding owner.id owner.source id.value [id])
  | .value value => do
      requireIdentity owner.id owner.source value.identity
      match findDeclaration context value.identity with
      | some metadata => pure metadata.kind
      | none =>
          throw (behaviorError .invalidBinding owner.id owner.source
            value.identity.value [value.identity])

private def validateSetupConstraint
    (context : BehaviorCheckContext)
    (owner : BehaviorDeclaration)
    (roles : List ResourceRole)
    (constraint : SetupConstraint) : Except BehaviorError Unit := do
  requireIdentity owner.id owner.source constraint.id
  let leftKind ← operandKind context owner roles constraint.left
  let rightKind ← operandKind context owner roles constraint.right
  if leftKind != rightKind then
    throw (behaviorError .invalidBinding owner.id owner.source
      (leftKind.name ++ " != " ++ rightKind.name) [constraint.id])

private structure OrderingGraph where
  indegree : Std.HashMap DeclarationId Nat
  outgoing : Std.HashMap DeclarationId (List DeclarationId)
  incoming : Std.HashMap DeclarationId (List DeclarationId)

private def buildOrderingGraph
    (occurrences : List DeclarationId)
    (ordering : List OccurrenceOrder) : OrderingGraph :=
  ordering.foldl (init := {
    indegree := occurrences.foldl (init := {}) fun degrees occurrence =>
      degrees.insert occurrence 0
    outgoing := {}
    incoming := {}
  }) fun graph edge => {
    indegree := graph.indegree.modify edge.after (fun count => count + 1)
    outgoing := graph.outgoing.insert edge.before (edge.after :: graph.outgoing.getD edge.before [])
    incoming := graph.incoming.insert edge.after (edge.before :: graph.incoming.getD edge.after [])
  }

private def countTopologically
    (outgoing : Std.HashMap DeclarationId (List DeclarationId))
    (indegree : Std.HashMap DeclarationId Nat)
    (pending : List DeclarationId)
    (count : Nat) : Nat → Nat × Std.HashMap DeclarationId Nat
  | 0 => (count, indegree)
  | fuel + 1 =>
      match pending with
      | [] => (count, indegree)
      | current :: rest =>
          let (indegree, pending) := (outgoing.getD current []).foldl (init := (indegree, rest))
            fun (degrees, pending) next =>
              let remaining := degrees.getD next 0 - 1
              let degrees := degrees.insert next remaining
              let pending := if remaining == 0 then next :: pending else pending
              (degrees, pending)
          countTopologically outgoing indegree pending (count + 1) fuel

private def followResidualPredecessors
    (incoming : Std.HashMap DeclarationId (List DeclarationId))
    (indegree : Std.HashMap DeclarationId Nat)
    (current : DeclarationId)
    (visited : List DeclarationId) : Nat → Option DeclarationId
  | 0 => none
  | fuel + 1 =>
      if visited.contains current then
        some current
      else
        let predecessors := (incoming.getD current []).filter
          (fun predecessor => decide (indegree.getD predecessor 0 > 0))
        match predecessors.mergeSort idLe with
        | predecessor :: _ =>
            followResidualPredecessors incoming indegree predecessor (current :: visited) fuel
        | [] => none

private def orderingCycleWitness?
    (occurrences : List DeclarationId)
    (ordering : List OccurrenceOrder) : Option DeclarationId :=
  let graph := buildOrderingGraph occurrences ordering
  let pending := occurrences.filter (fun occurrence => graph.indegree.getD occurrence 0 == 0)
  let (count, indegree) :=
    countTopologically graph.outgoing graph.indegree pending 0 occurrences.length
  if count == occurrences.length then
    none
  else
    match occurrences.find? (fun occurrence => indegree.getD occurrence 0 > 0) with
    | some start =>
        followResidualPredecessors graph.incoming indegree start [] (occurrences.length + 1)
    | none => none

private def validateOrdering
    (owner : BehaviorDeclaration)
    (occurrences : List NamedOccurrence) :
    List OccurrenceOrder → Except BehaviorError (List OccurrenceOrder)
  | ordering => do
      let canonical := ordering.mergeSort orderLe
      match firstDuplicateOrder canonical with
      | some edge =>
          throw (behaviorError .duplicateOrdering owner.id owner.source
            (edge.before.value ++ "->" ++ edge.after.value) [edge.before, edge.after])
      | none => pure ()
      match canonical.find? fun edge => edge.before == edge.after with
      | some edge =>
          throw (behaviorError .selfOrdering owner.id owner.source edge.before.value [edge.before])
      | none => pure ()
      let occurrenceIds := occurrences.map NamedOccurrence.id
      for edge in canonical do
        if !occurrenceIds.contains edge.before then
          throw (behaviorError .unknownReference owner.id owner.source
            edge.before.value [edge.before])
        if !occurrenceIds.contains edge.after then
          throw (behaviorError .unknownReference owner.id owner.source
            edge.after.value [edge.after])
      match orderingCycleWitness? occurrenceIds canonical with
      | some witness =>
          throw (behaviorError .cyclicOrdering owner.id owner.source witness.value [witness])
      | none => pure canonical

private def validateBinding
    (context : BehaviorCheckContext)
    (owner : BehaviorDeclaration)
    (roles : List ResourceRole)
    (binding : RoleBinding) : Except BehaviorError Unit := do
  let role ← match findRole roles binding.role with
    | some role => pure role
    | none =>
        throw (behaviorError .invalidBinding owner.id owner.source
          binding.role.value [binding.role])
  requireIdentity owner.id owner.source binding.value.identity
  match findDeclaration context binding.value.identity with
  | none =>
      throw (behaviorError .invalidBinding owner.id owner.source
        binding.value.identity.value [binding.role, binding.value.identity])
  | some metadata =>
      if metadata.kind != role.valueKind then
        throw (behaviorError .invalidBinding owner.id owner.source
          (binding.role.value ++ ": expected " ++ role.valueKind.name ++
            ", found " ++ metadata.kind.name)
          [binding.role, binding.value.identity])

private def requireSemanticValueKind
    (context : BehaviorCheckContext)
    (owner : BehaviorDeclaration)
    (expected : DeclarationKind)
    (value : SemanticValue) : Except BehaviorError Unit :=
  validateReferenceKind context owner value.identity expected

private def checkExactTrace
    (context : BehaviorCheckContext)
    (owner : BehaviorDeclaration)
    (roles : List ResourceRole)
    (authored : AuthoredExactTrace) : Except BehaviorError BehaviorTrace := do
  requireUniqueIds owner.id owner.source (authored.setup.map RoleBinding.role)
  for binding in authored.setup do
    validateBinding context owner roles binding
  let boundRoles := authored.setup.map RoleBinding.role
  for role in roles do
    if !boundRoles.contains role.id then
      throw (behaviorError .incompleteExactTrace owner.id owner.source
        ("missing setup binding " ++ role.id.value) [role.id])
  if authored.setup.length != roles.length then
    throw (behaviorError .invalidBinding owner.id owner.source "unexpected setup binding" boundRoles)
  let initialState ← match authored.initialState with
    | some state => pure state
    | none =>
        throw (behaviorError .incompleteExactTrace owner.id owner.source
          "initial-state" [])
  requireSemanticValueKind context owner .state initialState
  let mut steps := []
  for (authoredStep, index) in authored.steps.zipIdx do
    let action ← match authoredStep.selectedAction with
      | some value => pure value
      | none =>
          throw (behaviorError .incompleteExactTrace owner.id owner.source
            ("step-" ++ toString index ++ ":selected-action") [])
    let outcome ← match authoredStep.modelOutcome with
      | some value => pure value
      | none =>
          throw (behaviorError .incompleteExactTrace owner.id owner.source
            ("step-" ++ toString index ++ ":model-outcome") [])
    let state ← match authoredStep.resultingState with
      | some value => pure value
      | none =>
          throw (behaviorError .incompleteExactTrace owner.id owner.source
            ("step-" ++ toString index ++ ":resulting-state") [])
    let observations ← match authoredStep.observations with
      | some values => pure values
      | none =>
          throw (behaviorError .incompleteExactTrace owner.id owner.source
            ("step-" ++ toString index ++ ":observations") [])
    requireSemanticValueKind context owner .action action
    requireSemanticValueKind context owner .outcome outcome
    requireSemanticValueKind context owner .state state
    for observation in observations do
      requireSemanticValueKind context owner .observation observation
    steps := steps ++ [{
      selectedAction := action
      modelOutcome := outcome
      resultingState := state
      observations
    }]
  pure {
    setup := authored.setup.mergeSort bindingLe
    trace := { initialState, steps }
  }

private def countAction (action : DeclarationId) (actions : List DeclarationId) : Nat :=
  (actions.filter fun candidate => candidate == action).length

private def occurrenceIsReady
    (ordering : List OccurrenceOrder)
    (remaining : List NamedOccurrence)
    (occurrence : NamedOccurrence) : Bool :=
  ordering.all fun edge =>
    edge.after != occurrence.id ||
      !(remaining.any fun candidate => candidate.id == edge.before)

private structure OccurrenceAssignmentState where
  remaining : List NamedOccurrence
  assignedRev : List (Option NamedOccurrence)

private def insertOccurrenceState
    (states : List OccurrenceAssignmentState)
    (candidate : OccurrenceAssignmentState) : List OccurrenceAssignmentState :=
  if states.any fun state => state.remaining == candidate.remaining then
    states
  else
    states ++ [candidate]

private def advanceOccurrenceStates
    (ordering : List OccurrenceOrder)
    (action : DeclarationId)
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
    (schedule : List DeclarationId)
    (ordering : List OccurrenceOrder)
    (occurrences : List NamedOccurrence) : Option (List (Option NamedOccurrence)) :=
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
    (schedule : List DeclarationId)
    (ordering : List OccurrenceOrder)
    (occurrences : List NamedOccurrence) : Bool :=
  (assignOccurrenceSlots schedule ordering occurrences).isSome

private def isSubsequence : List DeclarationId → List DeclarationId → Bool
  | [], _ => true
  | _, [] => false
  | expected :: rest, actual :: remaining =>
      if expected == actual then
        isSubsequence rest remaining
      else
        isSubsequence (expected :: rest) remaining

private def isPrefix : List DeclarationId → List DeclarationId → Bool
  | [], _ => true
  | _, [] => false
  | expected :: rest, actual :: remaining =>
      expected == actual && isPrefix rest remaining

private def containsAdjacent (expected : List DeclarationId) : List DeclarationId → Bool
  | [] => expected == []
  | actual@(_ :: remaining) => isPrefix expected actual || containsAdjacent expected remaining

private def validateActionConstraints
    (owner : BehaviorDeclaration)
    (allowed forbidden : List DeclarationId)
    (required : List NamedOccurrence)
    (bounds : List OccurrenceBound)
    (ordering : List OccurrenceOrder)
    (sequences adjacencies : List (List DeclarationId))
    (exactSchedule : Option (List DeclarationId)) : Except BehaviorError Unit := do
  for occurrence in required do
    if forbidden.contains occurrence.action then
      throw (behaviorError .forbiddenRequired owner.id owner.source
        occurrence.action.value [occurrence.id, occurrence.action])
    if allowed != [] && !allowed.contains occurrence.action then
      throw (behaviorError .contradictoryConstraint owner.id owner.source
        ("required action not allowed: " ++ occurrence.action.value)
        [occurrence.id, occurrence.action])
  for action in allowed do
    if forbidden.contains action then
      throw (behaviorError .contradictoryConstraint owner.id owner.source
        ("allowed and forbidden: " ++ action.value) [action])
  for bound in bounds do
    match bound.maximum with
    | some maximum =>
        if bound.minimum > maximum then
          throw (behaviorError .contradictoryOccurrenceBounds owner.id owner.source
            bound.action.value [bound.action])
    | none => pure ()
    let requiredCount := (required.filter fun occurrence => occurrence.action == bound.action).length
    match bound.maximum with
    | some maximum =>
        if requiredCount > maximum then
          throw (behaviorError .contradictoryOccurrenceBounds owner.id owner.source
            bound.action.value [bound.action])
    | none => pure ()
    if forbidden.contains bound.action && bound.minimum > 0 then
      throw (behaviorError .forbiddenRequired owner.id owner.source
        bound.action.value [bound.action])
  match exactSchedule with
  | none => pure ()
  | some actions =>
      for action in actions do
        if forbidden.contains action then
          throw (behaviorError .forbiddenRequired owner.id owner.source action.value [action])
        if allowed != [] && !allowed.contains action then
          throw (behaviorError .contradictoryConstraint owner.id owner.source
            ("exact action not allowed: " ++ action.value) [action])
      for occurrence in required do
        let requiredCount := (required.filter fun candidate =>
          candidate.action == occurrence.action).length
        if countAction occurrence.action actions < requiredCount then
          throw (behaviorError .contradictoryOccurrenceBounds owner.id owner.source
            occurrence.action.value [occurrence.action])
      for bound in bounds do
        let count := countAction bound.action actions
        if count < bound.minimum || bound.maximum.any fun maximum => count > maximum then
          throw (behaviorError .contradictoryOccurrenceBounds owner.id owner.source
            bound.action.value [bound.action])
      if !hasOccurrenceAssignment actions ordering required then
        throw (behaviorError .contradictoryConstraint owner.id owner.source
          "exact schedule violates occurrence ordering" (required.map NamedOccurrence.id))
      for sequence in sequences do
        if !isSubsequence sequence actions then
          throw (behaviorError .contradictoryConstraint owner.id owner.source
            ("exact schedule omits sequence: " ++ idListKey sequence) sequence)
      for adjacency in adjacencies do
        if !containsAdjacent adjacency actions then
          throw (behaviorError .contradictoryConstraint owner.id owner.source
            ("exact schedule omits adjacency: " ++ idListKey adjacency) adjacency)

private def operandJson : SetupOperand → String
  | .role id => "{\"role\":" ++ quote id.value ++ "}"
  | .value value =>
      "{\"value\":{\"identity\":" ++ quote value.identity.value ++
        ",\"value\":" ++ quote value.value ++ "}}"

private def roleJson (role : ResourceRole) : String :=
  "{\"id\":" ++ quote role.id.value ++
    ",\"valueKind\":" ++ quote role.valueKind.name ++ "}"

private def setupConstraintJson (constraint : SetupConstraint) : String :=
  "{\"id\":" ++ quote constraint.id.value ++
    ",\"relation\":" ++ quote constraint.relation.name ++
    ",\"left\":" ++ operandJson constraint.left ++
    ",\"right\":" ++ operandJson constraint.right ++ "}"

private def occurrenceJson (occurrence : NamedOccurrence) : String :=
  "{\"id\":" ++ quote occurrence.id.value ++
    ",\"action\":" ++ quote occurrence.action.value ++ "}"

private def boundJson (bound : OccurrenceBound) : String :=
  "{\"action\":" ++ quote bound.action.value ++
    ",\"minimum\":" ++ toString bound.minimum ++
    ",\"maximum\":" ++ (bound.maximum.map toString |>.getD "null") ++ "}"

private def orderJson (edge : OccurrenceOrder) : String :=
  "{\"before\":" ++ quote edge.before.value ++
    ",\"after\":" ++ quote edge.after.value ++ "}"

private def valueJson (value : SemanticValue) : String :=
  "{\"identity\":" ++ quote value.identity.value ++
    ",\"value\":" ++ quote value.value ++ "}"

private def bindingJson (binding : RoleBinding) : String :=
  "{\"role\":" ++ quote binding.role.value ++
    ",\"value\":" ++ valueJson binding.value ++ "}"

private def traceStepJson
    (step : SemanticTraceStep SemanticValue SemanticValue SemanticValue SemanticValue) : String :=
  "{\"selectedAction\":" ++ valueJson step.selectedAction ++
    ",\"modelOutcome\":" ++ valueJson step.modelOutcome ++
    ",\"resultingState\":" ++ valueJson step.resultingState ++
    ",\"observations\":" ++ array (step.observations.map valueJson) ++ "}"

private def behaviorTraceJson (trace : BehaviorTrace) : String :=
  "{\"setup\":" ++ array (trace.setup.mergeSort bindingLe |>.map bindingJson) ++
    ",\"initialState\":" ++ valueJson trace.trace.initialState ++
    ",\"steps\":" ++ array (trace.trace.steps.map traceStepJson) ++ "}"

private def actionListJson (actions : List DeclarationId) : String :=
  array (actions.map (quote ∘ DeclarationId.value))

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
    (id : DeclarationId)
    (version : Nat)
    (requires : List DeclarationId)
    (roles : List ResourceRole)
    (setup : List SetupConstraint)
    (allowedActions : List DeclarationId)
    (requiredOccurrences : List NamedOccurrence)
    (forbiddenActions : List DeclarationId)
    (occurrenceBounds : List OccurrenceBound)
    (ordering : List OccurrenceOrder)
    (sequences adjacencies : List (List DeclarationId))
    (actionsExactly : Option (List DeclarationId))
    (traceExactly : Option BehaviorTrace)
    (spaceStatus : BehaviorSpaceStatus) : String :=
  "{\"id\":" ++ quote id.value ++
    ",\"version\":" ++ toString version ++
    ",\"requires\":" ++ actionListJson (canonicalIds requires) ++
    ",\"roles\":" ++ array (roles.mergeSort roleLe |>.map roleJson) ++
    ",\"setup\":" ++ array (setup.mergeSort constraintLe |>.map setupConstraintJson) ++
    ",\"allowedActions\":" ++ actionListJson (canonicalIds allowedActions) ++
    ",\"requiredOccurrences\":" ++
      array (requiredOccurrences.mergeSort occurrenceLe |>.map occurrenceJson) ++
    ",\"forbiddenActions\":" ++ actionListJson (canonicalIds forbiddenActions) ++
    ",\"occurrenceBounds\":" ++
      array (occurrenceBounds.mergeSort boundLe |>.map boundJson) ++
    ",\"ordering\":" ++ array (ordering.mergeSort orderLe |>.map orderJson) ++
    ",\"sequences\":" ++ array (canonicalIdLists sequences |>.map actionListJson) ++
    ",\"adjacencies\":" ++ array (canonicalIdLists adjacencies |>.map actionListJson) ++
    ",\"actionsExactly\":" ++ (actionsExactly.map actionListJson |>.getD "null") ++
    ",\"traceExactly\":" ++ (traceExactly.map behaviorTraceJson |>.getD "null") ++
    ",\"spaceStatus\":" ++ quote spaceStatus.name ++ "}"

def canonicalBehaviorJson (behavior : CheckedBehavior) : String :=
  "{\"semantic\":" ++ behaviorSemanticJson behavior.id behavior.version behavior.requires
      behavior.roles behavior.setup behavior.allowedActions behavior.requiredOccurrences
      behavior.forbiddenActions behavior.occurrenceBounds behavior.ordering behavior.sequences
      behavior.adjacencies behavior.actionsExactly behavior.traceExactly behavior.spaceStatus ++
    ",\"source\":" ++ sourceJson behavior.source ++
    ",\"documentation\":" ++ quote behavior.documentation ++ "}"

def canonicalBehaviorErrorJson (error : BehaviorError) : String :=
  "{\"kind\":" ++ quote error.kind.name ++
    ",\"declarationId\":" ++ quote error.declarationId.value ++
    ",\"sourcePath\":" ++ quote error.sourcePath ++
    ",\"offendingValue\":" ++ quote error.offendingValue ++
    ",\"relatedIdentities\":" ++
      array (canonicalIds error.relatedIdentities |>.map (quote ∘ DeclarationId.value)) ++ "}"

/-- Check and canonicalize a behavior without selecting a target or enumerating any trace. -/
def checkBehavior
    (context : BehaviorCheckContext)
    (declaration : BehaviorDeclaration) : Except BehaviorError CheckedBehavior := do
  requireIdentity declaration.id declaration.source declaration.id
  requireUniqueIds declaration.id declaration.source declaration.requires
  requireUniqueIds declaration.id declaration.source (declaration.roles.map ResourceRole.id)
  requireUniqueIds declaration.id declaration.source (declaration.setup.map SetupConstraint.id)
  requireUniqueIds declaration.id declaration.source declaration.allowedActions
  requireUniqueIds declaration.id declaration.source declaration.forbiddenActions
  requireUniqueIds declaration.id declaration.source
    (declaration.requiredOccurrences.map NamedOccurrence.id)
  requireUniqueIds declaration.id declaration.source
    (declaration.occurrenceBounds.map OccurrenceBound.action)
  for capability in declaration.requires do
    validateReferenceKind context declaration capability .capability
  let roles := declaration.roles.mergeSort roleLe
  for role in roles do
    requireIdentity declaration.id declaration.source role.id
  let setup := declaration.setup.map canonicalSetupConstraint |>.mergeSort constraintLe
  for constraint in setup do
    validateSetupConstraint context declaration roles constraint
  let allowed := canonicalIds declaration.allowedActions
  let forbidden := canonicalIds declaration.forbiddenActions
  let required := declaration.requiredOccurrences.mergeSort occurrenceLe
  if required.length > maxRequiredOccurrences then
    throw (behaviorError .occurrenceLimitExceeded declaration.id declaration.source
      (toString required.length ++ " > " ++ toString maxRequiredOccurrences)
      (required.map NamedOccurrence.id))
  let bounds := declaration.occurrenceBounds.mergeSort boundLe
  for action in allowed ++ forbidden ++ required.map NamedOccurrence.action ++
      bounds.map OccurrenceBound.action do
    validateReferenceKind context declaration action .action
  for actions in declaration.sequences ++ declaration.adjacencies do
    for action in actions do
      validateReferenceKind context declaration action .action
  match declaration.actionsExactly with
  | some actions =>
      for action in actions do
        validateReferenceKind context declaration action .action
  | none => pure ()
  let ordering ← validateOrdering declaration required declaration.ordering
  let exactTrace ← declaration.traceExactly.mapM (checkExactTrace context declaration roles)
  let exactSchedule := declaration.actionsExactly <|>
    exactTrace.map fun trace => trace.trace.steps.map
      (fun step => step.selectedAction.identity)
  let sequences := canonicalIdLists declaration.sequences
  let adjacencies := canonicalIdLists declaration.adjacencies
  validateActionConstraints declaration allowed forbidden required bounds ordering sequences
    adjacencies exactSchedule
  match declaration.actionsExactly, exactTrace with
  | some actions, some trace =>
      let traceActions := trace.trace.steps.map fun step => step.selectedAction.identity
      if actions != traceActions then
        throw (behaviorError .contradictoryConstraint declaration.id declaration.source
          "actionsExactly != traceExactly actions" actions)
  | _, _ => pure ()
  let status := if setupUnsatisfiable setup then
    .unsatisfiable
  else
    .unclassified
  let semantic := behaviorSemanticJson declaration.id declaration.version declaration.requires
    roles setup allowed required forbidden bounds ordering sequences adjacencies
    declaration.actionsExactly exactTrace status
  let checked : CheckedBehavior := {
    id := declaration.id
    source := declaration.source
    version := declaration.version
    requires := canonicalIds declaration.requires
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
    semanticDigest := semanticDigestOf semantic
  }
  pure { checked with canonicalMetadata := canonicalBehaviorJson checked }

private def bindingFor (bindings : List RoleBinding) (role : DeclarationId) : Option SemanticValue :=
  (bindings.find? fun binding => binding.role == role).map RoleBinding.value

private def resolveOperand (bindings : List RoleBinding) : SetupOperand → Option SemanticValue
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

private def setupIsComplete (roles : List ResourceRole) (bindings : List RoleBinding) : Bool :=
  bindings.length == roles.length &&
    roles.all (fun role => countAction role.id (bindings.map RoleBinding.role) == 1) &&
    bindings.all (fun binding => roles.any fun role => role.id == binding.role)

private def traceActions (trace : BehaviorTrace) : List DeclarationId :=
  trace.trace.steps.map fun step => step.selectedAction.identity

private def normalizedTrace (trace : BehaviorTrace) : BehaviorTrace :=
  { trace with setup := trace.setup.mergeSort bindingLe }

/-- Canonically attribute selected action positions to authored required occurrences. -/
def CheckedBehavior.assignOccurrences
    (behavior : CheckedBehavior)
    (schedule : List DeclarationId) : Option (List (Option NamedOccurrence)) :=
  assignOccurrenceSlots schedule behavior.ordering behavior.requiredOccurrences

/-- Membership is a pure predicate over already semantic, target-owned trace data. -/
def CheckedBehavior.admits (behavior : CheckedBehavior) (candidate : BehaviorTrace) : Bool :=
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

def CheckedBehavior.isUnsatisfiable (behavior : CheckedBehavior) : Bool :=
  behavior.spaceStatus == .unsatisfiable

end Umpire
