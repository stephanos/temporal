import Umpire3.Certificate

namespace Umpire3

namespace Exact

structure Limits where
  maxDepth : Nat
  maxStates : Nat
  maxTransitions : Nat
  maxStateBytes : Nat
  maxWork : Nat := 100000
  deriving DecidableEq, Repr

inductive Resource where
  | states
  | transitions
  | stateBytes
  | work
  deriving DecidableEq, Repr

structure Statistics {World : Type} (model : Behavior World) (world : World) where
  visited : Nat
  transitions : Nat
  stateBytes : Nat
  buckets : Nat
  collisions : Nat
  coverage : List String
  deadlocks : List (model.State world)

inductive Classification where
  | traceWitness
  | finiteExhaustive
  | boundedSafe (depth : Nat)
  | resourceLimited (resource : Resource)
  | invalidCertificate
  | internalError
  deriving DecidableEq, Repr

inductive Result {World : Type} (model : Behavior World) (world : World) where
  | counterexample (witness : Node model world) (certificate : Certificate model world)
      (statistics : Statistics model world)
  | exhausted (certificate : Certificate model world) (statistics : Statistics model world)
  | depthComplete (depth : Nat) (certificate : Certificate model world)
      (statistics : Statistics model world)
  | resourceLimited (resource : Resource) (certificate : Certificate model world)
      (statistics : Statistics model world)
  | internalError (certificate : Certificate model world) (statistics : Statistics model world)

def Result.witness? {model : Behavior World} {world : World} :
    Result model world → Option (Node model world)
  | .counterexample witness _ _ => some witness
  | _ => none

def Result.certificate {model : Behavior World} {world : World} :
    Result model world → Certificate model world
  | .counterexample _ certificate _ => certificate
  | .exhausted certificate _ => certificate
  | .depthComplete _ certificate _ => certificate
  | .resourceLimited _ certificate _ => certificate
  | .internalError certificate _ => certificate

def Result.statistics {model : Behavior World} {world : World} :
    Result model world → Statistics model world
  | .counterexample _ _ statistics => statistics
  | .exhausted _ statistics => statistics
  | .depthComplete _ _ statistics => statistics
  | .resourceLimited _ _ statistics => statistics
  | .internalError _ statistics => statistics

def Result.mapCertificate {model : Behavior World} {world : World}
    (result : Result model world)
    (map : Certificate model world → Certificate model world) : Result model world :=
  match result with
  | .counterexample witness certificate statistics =>
      .counterexample witness (map certificate) statistics
  | .exhausted certificate statistics => .exhausted (map certificate) statistics
  | .depthComplete depth certificate statistics =>
      .depthComplete depth (map certificate) statistics
  | .resourceLimited resource certificate statistics =>
      .resourceLimited resource (map certificate) statistics
  | .internalError certificate statistics => .internalError (map certificate) statistics

def Node.checkViolation {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (node : Node model world) : Bool :=
  node.check view && !property node.state

theorem Node.checkedViolation {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (node : Node model world) (checked : node.checkViolation view property = true) :
    model.Reachable world node.state ∧ property node.state = false := by
  simp only [Node.checkViolation, Bool.and_eq_true] at checked
  refine ⟨node.reachable view checked.1, ?_⟩
  simpa only [Bool.not_eq_true'] using checked.2

def Result.Accepted {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (result : Result model world) : Prop :=
  result.certificate.checkReachable view = true ∧
    match result with
    | .counterexample witness certificate _ =>
        witness.checkViolation view property = true ∧
          certificate.checkBefore view property witness.depth = true
    | .exhausted certificate _ => certificate.Valid view property .exhaustive
    | .depthComplete depth certificate _ => certificate.Valid view property (.depth depth)
    | .resourceLimited _ _ _ => True
    | .internalError certificate _ => certificate.nodes = []

private def rejected {model : Behavior World} {world : World}
    (statistics : Statistics model world) : Result model world :=
  .internalError { nodes := [] } statistics

private def Result.validateExplorer {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (result : Result model world) : Result model world :=
  if result.certificate.checkReachable view then
    match result with
    | .counterexample witness certificate statistics =>
        if witness.checkViolation view property &&
            certificate.checkBefore view property witness.depth then
          result
        else
          rejected statistics
    | .exhausted certificate statistics =>
        if certificate.check view property .exhaustive then result else rejected statistics
    | .depthComplete depth certificate statistics =>
        if certificate.check view property (.depth depth) then result else rejected statistics
    | .resourceLimited _ _ _ => result
    | .internalError _ statistics => rejected statistics
  else
    rejected result.statistics

private theorem rejected_accepted {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (statistics : Statistics model world) :
    Result.Accepted view property (rejected statistics) := by
  simp [Result.Accepted, rejected, Result.certificate, Certificate.checkReachable]

private theorem Result.validateExplorer_accepted
    {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (result : Result model world) :
    Result.Accepted view property (Result.validateExplorer view property result) := by
  cases result with
  | counterexample witness certificate statistics =>
      by_cases reachable : certificate.checkReachable view = true
      · by_cases violation : witness.checkViolation view property = true
        · by_cases before : certificate.checkBefore view property witness.depth = true
          · simp only [Result.validateExplorer, Result.certificate, reachable, if_true,
              violation, before, Bool.and_self]
            exact ⟨reachable, violation, before⟩
          · simpa [Result.validateExplorer, Result.certificate, Result.statistics,
              reachable, violation, before] using
              rejected_accepted view property statistics
        · simpa [Result.validateExplorer, Result.certificate, Result.statistics,
            reachable, violation] using
            rejected_accepted view property statistics
      · simpa [Result.validateExplorer, Result.certificate, Result.statistics, reachable] using
          rejected_accepted view property statistics
  | exhausted certificate statistics =>
      by_cases reachable : certificate.checkReachable view = true
      · by_cases valid : certificate.check view property .exhaustive = true
        · simp only [Result.validateExplorer, Result.certificate, reachable, if_true, valid]
          exact ⟨reachable, valid⟩
        · simpa [Result.validateExplorer, Result.certificate, Result.statistics,
            reachable, valid] using
            rejected_accepted view property statistics
      · simpa [Result.validateExplorer, Result.certificate, Result.statistics, reachable] using
          rejected_accepted view property statistics
  | depthComplete depth certificate statistics =>
      by_cases reachable : certificate.checkReachable view = true
      · by_cases valid : certificate.check view property (.depth depth) = true
        · simp only [Result.validateExplorer, Result.certificate, reachable, if_true, valid]
          exact ⟨reachable, valid⟩
        · simpa [Result.validateExplorer, Result.certificate, Result.statistics,
            reachable, valid] using
            rejected_accepted view property statistics
      · simpa [Result.validateExplorer, Result.certificate, Result.statistics, reachable] using
          rejected_accepted view property statistics
  | resourceLimited resource certificate statistics =>
      by_cases reachable : certificate.checkReachable view = true
      · simp only [Result.validateExplorer, Result.certificate, reachable, if_true]
        exact ⟨reachable, trivial⟩
      · simpa [Result.validateExplorer, Result.certificate, Result.statistics, reachable] using
          rejected_accepted view property statistics
  | internalError certificate statistics =>
      by_cases reachable : certificate.checkReachable view = true <;>
        simpa [Result.validateExplorer, Result.certificate, Result.statistics, reachable] using
          rejected_accepted view property statistics

def classify {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool) :
    Result model world → Classification
  | .counterexample witness _ _ =>
      if witness.checkViolation view property then .traceWitness else .invalidCertificate
  | .exhausted certificate _ =>
      if certificate.check view property .exhaustive then
        .finiteExhaustive
      else .invalidCertificate
  | .depthComplete depth certificate _ =>
      if certificate.check view property (.depth depth) then
        .boundedSafe depth
      else .invalidCertificate
  | .resourceLimited resource _ _ => .resourceLimited resource
  | .internalError _ _ => .internalError

theorem classify_finiteExhaustive_sound {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (result : Result model world)
    (classified : classify view property result = .finiteExhaustive) :
    Safety (model.at world) (fun state => property state = true) := by
  cases result with
  | counterexample witness certificate statistics =>
      simp only [classify] at classified
      split at classified <;> contradiction
  | exhausted certificate statistics =>
      simp only [classify] at classified
      split at classified
      · rename_i checked
        exact certificate.valid_exhaustive_safety view property checked
      · contradiction
  | depthComplete depth certificate statistics =>
      simp only [classify] at classified
      split at classified <;> contradiction
  | resourceLimited resource certificate statistics =>
      simp [classify] at classified
  | internalError certificate statistics =>
      simp [classify] at classified

theorem classify_boundedSafe_sound {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (result : Result model world) (depth : Nat)
    (classified : classify view property result = .boundedSafe depth) :
    ∀ state, ReachableWithin model world depth state → property state = true := by
  cases result with
  | counterexample witness certificate statistics =>
      simp only [classify] at classified
      split at classified <;> contradiction
  | exhausted certificate statistics =>
      simp only [classify] at classified
      split at classified <;> contradiction
  | depthComplete checkedDepth certificate statistics =>
      simp only [classify] at classified
      split at classified
      · rename_i checked
        injection classified with sameDepth
        subst depth
        exact certificate.valid_depth_safety view property checkedDepth checked
      · contradiction
  | resourceLimited resource certificate statistics =>
      simp [classify] at classified
  | internalError certificate statistics =>
      simp [classify] at classified

theorem classify_traceWitness_sound {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (result : Result model world)
    (classified : classify view property result = .traceWitness) :
    ∃ state, model.Reachable world state ∧ property state = false := by
  cases result with
  | counterexample witness certificate statistics =>
      simp only [classify] at classified
      split at classified
      · rename_i checked
        exact ⟨witness.state, witness.checkedViolation view property checked⟩
      · contradiction
  | exhausted certificate statistics =>
      simp only [classify] at classified
      split at classified <;> contradiction
  | depthComplete depth certificate statistics =>
      simp only [classify] at classified
      split at classified <;> contradiction
  | resourceLimited resource certificate statistics =>
      simp [classify] at classified
  | internalError certificate statistics =>
      simp [classify] at classified

private structure StateStore (State : Type) where
  buckets : List (Nat × List State) := []
  collisions : Nat := 0

private def StateStore.contains {model : Behavior World} {world : World}
    (view : FiniteView model world) (store : StateStore (model.State world))
    (state : model.State world) : Bool :=
  letI : DecidableEq (model.State world) := view.identity.decidableEq
  let fingerprint := view.identity.fingerprint (view.identity.encode state)
  store.buckets.any fun bucket =>
    decide (bucket.1 = fingerprint) && bucket.2.any fun candidate => decide (candidate = state)

private def StateStore.insert {model : Behavior World} {world : World}
    (view : FiniteView model world) (store : StateStore (model.State world))
    (state : model.State world) : StateStore (model.State world) :=
  let fingerprint := view.identity.fingerprint (view.identity.encode state)
  let rec insertBucket : List (Nat × List (model.State world)) →
      List (Nat × List (model.State world)) × Bool
    | [] => ([(fingerprint, [state])], false)
    | bucket :: buckets =>
        if bucket.1 = fingerprint then
          ((bucket.1, bucket.2 ++ [state]) :: buckets, !bucket.2.isEmpty)
        else
          let inserted := insertBucket buckets
          (bucket :: inserted.1, inserted.2)
  let inserted := insertBucket store.buckets
  { buckets := inserted.1, collisions := store.collisions + if inserted.2 then 1 else 0 }

private structure SearchState {World : Type} (model : Behavior World) (world : World) where
  store : StateStore (model.State world)
  queue : List (Node model world)
  nodes : List (Node model world)
  transitions : Nat
  stateBytes : Nat
  coverage : List String
  deadlocks : List (model.State world)
  depthTruncated : Bool

private def SearchState.statistics {model : Behavior World} {world : World}
    (state : SearchState model world) : Statistics model world where
  visited := state.nodes.length
  transitions := state.transitions
  stateBytes := state.stateBytes
  buckets := state.store.buckets.length
  collisions := state.store.collisions
  coverage := state.coverage
  deadlocks := state.deadlocks

private def SearchState.certificate {model : Behavior World} {world : World}
    (state : SearchState model world) : Certificate model world where
  nodes := state.nodes

private def insertCoverage (coverage : List String) (identifier : String) : List String :=
  if coverage.contains identifier then coverage else coverage ++ [identifier]

private inductive Expansion {World : Type} (model : Behavior World) (world : World) where
  | ready (state : SearchState model world)
  | limited (resource : Resource) (state : SearchState model world)

private def addSuccessors {model : Behavior World} {world : World}
    (view : FiniteView model world) (limits : Limits) (parent : Node model world) :
    SearchState model world → List (model.Action world × model.State world) →
      Expansion model world
  | state, [] => .ready state
  | state, successor :: successors =>
      let state := {
        state with
        transitions := state.transitions + 1
        coverage := insertCoverage state.coverage (view.actionName successor.1)
      }
      if state.transitions > limits.maxTransitions then
        .limited .transitions state
      else if state.store.contains view successor.2 then
        addSuccessors view limits parent state successors
      else if parent.depth ≥ limits.maxDepth then
        addSuccessors view limits parent { state with depthTruncated := true } successors
      else if state.nodes.length ≥ limits.maxStates then
        .limited .states state
      else
        let encodedSize := view.identity.encodedSize (view.identity.encode successor.2)
        if state.stateBytes + encodedSize > limits.maxStateBytes then
          .limited .stateBytes state
        else
          let node : Node model world := {
            initial := parent.initial
            actions := parent.actions ++ [successor.1]
            state := successor.2
          }
          let state := {
            state with
            store := state.store.insert view successor.2
            queue := state.queue ++ [node]
            nodes := state.nodes ++ [node]
            stateBytes := state.stateBytes + encodedSize
          }
          addSuccessors view limits parent state successors

private inductive Seed {World : Type} (model : Behavior World) (world : World) where
  | ready (state : SearchState model world)
  | limited (resource : Resource) (state : SearchState model world)

private def seedInitials {model : Behavior World} {world : World}
    (view : FiniteView model world) (limits : Limits) :
    SearchState model world → List (model.State world) → Seed model world
  | state, [] => .ready state
  | state, initial :: initials =>
      if state.store.contains view initial then
        seedInitials view limits state initials
      else if state.nodes.length ≥ limits.maxStates then
        .limited .states state
      else
        let encodedSize := view.identity.encodedSize (view.identity.encode initial)
        if state.stateBytes + encodedSize > limits.maxStateBytes then
          .limited .stateBytes state
        else
          let node : Node model world := { initial := initial, actions := [], state := initial }
          let state := {
            state with
            store := state.store.insert view initial
            queue := state.queue ++ [node]
            nodes := state.nodes ++ [node]
            stateBytes := state.stateBytes + encodedSize
          }
          seedInitials view limits state initials

private def finish {model : Behavior World} {world : World} (limits : Limits)
    (state : SearchState model world) : Result model world :=
  if state.depthTruncated then
    .depthComplete limits.maxDepth state.certificate state.statistics
  else
    .exhausted state.certificate state.statistics

private def search {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (limits : Limits) : Nat → SearchState model world → Result model world
  | 0, state => .resourceLimited .work state.certificate state.statistics
  | fuel + 1, state =>
      match state.queue with
      | [] => finish limits state
      | node :: queue =>
          let state := { state with queue := queue }
          if !property node.state then
            .counterexample node state.certificate state.statistics
          else
            let successors := view.successors node.state
            let state := if successors.isEmpty then
              { state with deadlocks := state.deadlocks ++ [node.state] }
            else state
            match addSuccessors view limits node state successors with
            | .limited resource state =>
                .resourceLimited resource state.certificate state.statistics
            | .ready state => search view property limits fuel state

private def exploreRaw {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (limits : Limits) : Result model world :=
  let empty : SearchState model world := {
    store := {}
    queue := []
    nodes := []
    transitions := 0
    stateBytes := 0
    coverage := []
    deadlocks := []
    depthTruncated := false
  }
  match seedInitials view limits empty view.initials with
  | .limited resource state => .resourceLimited resource state.certificate state.statistics
  | .ready state => search view property limits (min limits.maxWork (limits.maxStates + 1)) state

def explore {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (limits : Limits) : Result model world :=
  Result.validateExplorer view property (exploreRaw view property limits)

theorem explore_accepted {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (limits : Limits) : Result.Accepted view property (explore view property limits) :=
  Result.validateExplorer_accepted view property (exploreRaw view property limits)

theorem explore_nodes_reachable {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (limits : Limits) {node : Node model world}
    (member : node ∈ (explore view property limits).certificate.nodes) :
    model.Reachable world node.state :=
  (explore view property limits).certificate.checked_nodes_reachable view
    (explore_accepted view property limits).1 member

theorem explore_finiteExhaustive_exact {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (limits : Limits)
    (classified : classify view property (explore view property limits) = .finiteExhaustive)
    (state : model.State world) :
    (explore view property limits).certificate.Contains state ↔
      model.Reachable world state := by
  generalize produced : explore view property limits = result
  rw [produced] at classified
  have accepted := explore_accepted view property limits
  rw [produced] at accepted
  cases result with
  | counterexample witness certificate statistics =>
      simp only [classify] at classified
      split at classified <;> contradiction
  | exhausted certificate statistics =>
      exact certificate.valid_exhaustive_exact view property accepted.2 state
  | depthComplete depth certificate statistics =>
      simp only [classify] at classified
      split at classified <;> contradiction
  | resourceLimited resource certificate statistics =>
      simp [classify] at classified
  | internalError certificate statistics =>
      simp [classify] at classified

theorem explore_boundedSafe_covers {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (limits : Limits) (depth : Nat)
    (classified : classify view property (explore view property limits) = .boundedSafe depth)
    (state : model.State world) (reachable : ReachableWithin model world depth state) :
    (explore view property limits).certificate.Contains state := by
  generalize produced : explore view property limits = result
  rw [produced] at classified
  have accepted := explore_accepted view property limits
  rw [produced] at accepted
  cases result with
  | counterexample witness certificate statistics =>
      simp only [classify] at classified
      split at classified <;> contradiction
  | exhausted certificate statistics =>
      simp only [classify] at classified
      split at classified <;> contradiction
  | depthComplete checkedDepth certificate statistics =>
      simp only [classify] at classified
      split at classified
      · injection classified with sameDepth
        subst depth
        exact certificate.valid_depth_covers_reachable view property checkedDepth
          accepted.2 reachable
      · contradiction
  | resourceLimited resource certificate statistics =>
      simp [classify] at classified
  | internalError certificate statistics =>
      simp [classify] at classified

theorem explore_traceWitness_shortest {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (limits : Limits)
    (classified : classify view property (explore view property limits) = .traceWitness) :
    ∃ witness, (explore view property limits).witness? = some witness ∧
      model.Reachable world witness.state ∧ property witness.state = false ∧
      ∀ state, ReachableBefore model world witness.depth state → property state = true := by
  generalize produced : explore view property limits = result
  rw [produced] at classified
  have accepted := explore_accepted view property limits
  rw [produced] at accepted
  cases result with
  | counterexample witness certificate statistics =>
      simp only [classify] at classified
      split at classified
      · have violation := witness.checkedViolation view property accepted.2.1
        exact ⟨witness, rfl, violation.1, violation.2,
          certificate.checked_before_safety view property witness.depth accepted.2.2⟩
      · contradiction
  | exhausted certificate statistics =>
      simp only [classify] at classified
      split at classified <;> contradiction
  | depthComplete depth certificate statistics =>
      simp only [classify] at classified
      split at classified <;> contradiction
  | resourceLimited resource certificate statistics =>
      simp [classify] at classified
  | internalError certificate statistics =>
      simp [classify] at classified

end Exact

end Umpire3
