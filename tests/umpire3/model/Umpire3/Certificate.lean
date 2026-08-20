import Umpire3.FiniteView
import Umpire3.Execution
import Umpire3.Property

namespace Umpire3

namespace Exact

structure Node {World : Type} (model : Behavior World) (world : World) where
  initial : model.State world
  actions : List (model.Action world)
  state : model.State world

def Node.depth {model : Behavior World} {world : World} (node : Node model world) : Nat :=
  node.actions.length

structure Certificate {World : Type} (model : Behavior World) (world : World) where
  nodes : List (Node model world)

def Certificate.Contains {model : Behavior World} {world : World}
    (certificate : Certificate model world) (state : model.State world) : Prop :=
  ∃ node ∈ certificate.nodes, node.state = state

inductive Scope where
  | exhaustive
  | depth (value : Nat)
  deriving DecidableEq, Repr

def Scope.contains : Scope → Nat → Bool
  | .exhaustive, _ => true
  | .depth limit, value => decide (value ≤ limit)

def Scope.closes : Scope → Nat → Bool
  | .exhaustive, _ => true
  | .depth limit, value => decide (value < limit)

def Node.check {model : Behavior World} {world : World}
    (view : FiniteView model world) (node : Node model world) : Bool :=
  letI : DecidableEq (model.State world) := view.identity.decidableEq
  letI : DecidableEq (model.Action world) := view.actionDecidableEq
  decide (node.initial ∈ view.initials) &&
    decide (node.state ∈ view.executable.follow world [node.initial] node.actions)

def Certificate.containsState {model : Behavior World} {world : World}
    (view : FiniteView model world) (certificate : Certificate model world)
    (state : model.State world) : Bool :=
  letI : DecidableEq (model.State world) := view.identity.decidableEq
  certificate.nodes.any fun node => decide (node.state = state)

def Certificate.containsInitial {model : Behavior World} {world : World}
    (view : FiniteView model world) (certificate : Certificate model world)
    (state : model.State world) : Bool :=
  letI : DecidableEq (model.State world) := view.identity.decidableEq
  certificate.nodes.any fun node =>
    decide (node.initial = state ∧ node.state = state ∧ node.actions = [])

def Certificate.containsSuccessor {model : Behavior World} {world : World}
    (view : FiniteView model world) (certificate : Certificate model world)
    (depth : Nat) (state : model.State world) : Bool :=
  letI : DecidableEq (model.State world) := view.identity.decidableEq
  certificate.nodes.any fun node =>
    decide (node.state = state ∧ node.depth ≤ depth + 1)

def Certificate.checkReachable {model : Behavior World} {world : World}
    (view : FiniteView model world) (certificate : Certificate model world) : Bool :=
  certificate.nodes.all fun node => node.check view

def Certificate.before {model : Behavior World} {world : World}
    (certificate : Certificate model world) (depth : Nat) : Certificate model world where
  nodes := certificate.nodes.filter fun node => decide (node.depth < depth)

def Certificate.distinct {model : Behavior World} {world : World}
    (view : FiniteView model world) : List (Node model world) → Bool
  | [] => true
  | node :: nodes =>
      letI : DecidableEq view.identity.Code := view.identity.codeDecidableEq
      nodes.all (fun other =>
        decide (view.identity.encode node.state ≠ view.identity.encode other.state)) &&
        Certificate.distinct view nodes

def Certificate.check {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (scope : Scope) (certificate : Certificate model world) : Bool :=
  letI : DecidableEq (model.State world) := view.identity.decidableEq
  Certificate.distinct view certificate.nodes &&
    certificate.nodes.all (fun node => node.check view) &&
    view.initials.all (fun state => certificate.containsInitial view state) &&
    certificate.nodes.all (fun node =>
      property node.state && scope.contains node.depth &&
        (!scope.closes node.depth ||
          (view.successors node.state).all (fun successor =>
            certificate.containsSuccessor view node.depth successor.2)))

def Certificate.checkBefore {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (certificate : Certificate model world) : Nat → Bool
  | 0 => true
  | depth + 1 => (certificate.before (depth + 1)).check view property (.depth depth)

def Certificate.Valid {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (scope : Scope) (certificate : Certificate model world) : Prop :=
  certificate.check view property scope = true

theorem ExecutableView.follow_sound {model : Behavior World} (view : ExecutableView model)
    (world : World) [DecidableEq (model.Action world)]
    {states : List (model.State world)} {actions : List (model.Action world)}
    {final : model.State world}
    (member : final ∈ view.follow world states actions) :
    ∃ initial, initial ∈ states ∧ Runs (model.at world) initial actions final := by
  induction actions generalizing states with
  | nil =>
      exact ⟨final, member, Runs.nil (model := model.at world) final⟩
  | cons action actions inductionHypothesis =>
      change final ∈ view.follow world
        (states.flatMap fun state =>
          (view.successors world state).filterMap fun successor =>
            if successor.1 = action then some successor.2 else none)
        actions at member
      rcases inductionHypothesis member with ⟨nextState, nextMember, tail⟩
      rcases List.mem_flatMap.mp nextMember with ⟨state, stateMember, filteredMember⟩
      rcases List.mem_filterMap.mp filteredMember with
        ⟨successor, successorMember, selected⟩
      by_cases sameAction : successor.1 = action
      · have nextEquality : successor.2 = nextState := by
          simpa [sameAction] using selected
        subst nextState
        have step : model.Step world state action successor.2 := by
          rw [← sameAction]
          exact (view.successors_exact world state successor.1 successor.2).mp successorMember
        exact ⟨state, stateMember, Runs.cons step tail⟩
      · simp [sameAction] at selected

theorem Node.reachable {model : Behavior World} {world : World}
    (view : FiniteView model world) (node : Node model world)
    (checked : node.check view = true) : model.Reachable world node.state := by
  letI : DecidableEq (model.State world) := view.identity.decidableEq
  letI : DecidableEq (model.Action world) := view.actionDecidableEq
  simp only [Node.check, Bool.and_eq_true, decide_eq_true_eq] at checked
  rcases ExecutableView.follow_sound view.executable world checked.2 with
    ⟨initial, member, run⟩
  simp at member
  subst initial
  exact ⟨node.initial, node.actions,
    (view.executable.initials_exact world node.initial).mp checked.1, run⟩

theorem Certificate.checked_nodes_reachable {model : Behavior World} {world : World}
    (view : FiniteView model world) (certificate : Certificate model world)
    (checked : certificate.checkReachable view = true)
    {node : Node model world} (member : node ∈ certificate.nodes) :
    model.Reachable world node.state := by
  simp only [Certificate.checkReachable, List.all_eq_true] at checked
  exact node.reachable view (checked node member)

theorem Certificate.valid_nodes_reachable {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (scope : Scope) (certificate : Certificate model world)
    (valid : certificate.Valid view property scope)
    {node : Node model world} (member : node ∈ certificate.nodes) :
    model.Reachable world node.state := by
  letI : DecidableEq (model.State world) := view.identity.decidableEq
  simp only [Certificate.Valid, Certificate.check, Bool.and_eq_true,
    List.all_eq_true] at valid
  exact node.reachable view (valid.1.1.2 node member)

theorem Certificate.valid_safe {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (scope : Scope) (certificate : Certificate model world)
    (valid : certificate.Valid view property scope)
    {node : Node model world} (member : node ∈ certificate.nodes) :
    property node.state = true := by
  letI : DecidableEq (model.State world) := view.identity.decidableEq
  simp only [Certificate.Valid, Certificate.check, Bool.and_eq_true,
    List.all_eq_true] at valid
  have checked := valid.2 node member
  exact checked.1.1

theorem Certificate.valid_initial {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (scope : Scope) (certificate : Certificate model world)
    (valid : certificate.Valid view property scope)
    {state : model.State world} (initial : model.Initial world state) :
    ∃ node ∈ certificate.nodes,
      node.initial = state ∧ node.state = state ∧ node.actions = [] := by
  letI : DecidableEq (model.State world) := view.identity.decidableEq
  simp only [Certificate.Valid, Certificate.check, Bool.and_eq_true,
    List.all_eq_true] at valid
  have member : state ∈ view.initials :=
    (view.executable.initials_exact world state).mpr initial
  have covered := valid.1.2 state member
  simpa [Certificate.containsInitial] using covered

theorem Certificate.valid_closed {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (certificate : Certificate model world)
    (valid : certificate.Valid view property .exhaustive)
    {node : Node model world} (nodeMember : node ∈ certificate.nodes)
    {action : model.Action world} {nextState : model.State world}
    (successorMember : (action, nextState) ∈ view.successors node.state) :
    ∃ nextNode ∈ certificate.nodes,
      nextNode.state = nextState ∧ nextNode.depth ≤ node.depth + 1 := by
  letI : DecidableEq (model.State world) := view.identity.decidableEq
  simp only [Certificate.Valid, Certificate.check, Bool.and_eq_true,
    List.all_eq_true] at valid
  have checked := valid.2 node nodeMember
  have closed := checked.2
  simp only [Scope.closes, Bool.not_true, Bool.false_or, List.all_eq_true] at closed
  have contained := closed (action, nextState) successorMember
  simpa [Certificate.containsSuccessor] using contained

theorem Certificate.valid_depth_closed {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (depth : Nat) (certificate : Certificate model world)
    (valid : certificate.Valid view property (.depth depth))
    {node : Node model world} (nodeMember : node ∈ certificate.nodes)
    (within : node.depth < depth)
    {action : model.Action world} {nextState : model.State world}
    (successorMember : (action, nextState) ∈ view.successors node.state) :
    ∃ nextNode ∈ certificate.nodes,
      nextNode.state = nextState ∧ nextNode.depth ≤ node.depth + 1 := by
  letI : DecidableEq (model.State world) := view.identity.decidableEq
  simp only [Certificate.Valid, Certificate.check, Bool.and_eq_true,
    List.all_eq_true] at valid
  have checked := valid.2 node nodeMember
  have closed := checked.2
  have closes : Scope.closes (.depth depth) node.depth = true := by
    simp [Scope.closes, within]
  simp only [closes, Bool.not_true, Bool.false_or, List.all_eq_true] at closed
  have contained := closed (action, nextState) successorMember
  simpa [Certificate.containsSuccessor] using contained

theorem Certificate.valid_exhaustive_covers_run {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (certificate : Certificate model world)
    (valid : certificate.Valid view property .exhaustive)
    {start final : model.State world} {actions : List (model.Action world)}
    (run : Runs (model.at world) start actions final)
    (covered : certificate.Contains start) : certificate.Contains final := by
  induction actions generalizing start with
  | nil =>
      have equality := Runs.empty run
      subst final
      exact covered
  | cons action actions inductionHypothesis =>
      rcases Runs.uncons run with ⟨next, step, tail⟩
      change model.Step world start action next at step
      rcases covered with ⟨node, nodeMember, nodeState⟩
      have successorMember : (action, next) ∈ view.successors node.state := by
        apply (view.executable.successors_exact world node.state action next).mpr
        simpa [nodeState] using step
      rcases certificate.valid_closed view property valid nodeMember successorMember with
        ⟨nextNode, nextMember, nextState, _⟩
      exact inductionHypothesis tail ⟨nextNode, nextMember, nextState⟩

theorem Certificate.valid_exhaustive_covers_reachable
    {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (certificate : Certificate model world)
    (valid : certificate.Valid view property .exhaustive)
    {state : model.State world} (reachable : model.Reachable world state) :
    certificate.Contains state := by
  rcases reachable with ⟨initial, actions, initialState, run⟩
  rcases certificate.valid_initial view property .exhaustive valid initialState with
    ⟨node, member, _, nodeState, _⟩
  exact certificate.valid_exhaustive_covers_run view property valid run
    ⟨node, member, nodeState⟩

theorem Certificate.valid_exhaustive_exact
    {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (certificate : Certificate model world)
    (valid : certificate.Valid view property .exhaustive)
    (state : model.State world) :
    certificate.Contains state ↔ model.Reachable world state := by
  constructor
  · rintro ⟨node, member, nodeState⟩
    rw [← nodeState]
    exact certificate.valid_nodes_reachable view property .exhaustive valid member
  · exact certificate.valid_exhaustive_covers_reachable view property valid

theorem Certificate.valid_exhaustive_safety
    {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (certificate : Certificate model world)
    (valid : certificate.Valid view property .exhaustive) :
    Safety (model.at world) (fun state => property state = true) := by
  intro state reachable
  rcases certificate.valid_exhaustive_covers_reachable view property valid reachable with
    ⟨node, member, nodeState⟩
  rw [← nodeState]
  exact certificate.valid_safe view property .exhaustive valid member

def ReachableWithin (model : Behavior World) (world : World) (depth : Nat)
    (state : model.State world) : Prop :=
  ∃ initial actions, model.Initial world initial ∧
    Runs (model.at world) initial actions state ∧ actions.length ≤ depth

def ReachableBefore (model : Behavior World) (world : World) (depth : Nat)
    (state : model.State world) : Prop :=
  ∃ initial actions, model.Initial world initial ∧
    Runs (model.at world) initial actions state ∧ actions.length < depth

theorem Certificate.valid_depth_covers_run {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (depth : Nat) (certificate : Certificate model world)
    (valid : certificate.Valid view property (.depth depth))
    {start final : model.State world} {actions : List (model.Action world)}
    (run : Runs (model.at world) start actions final)
    (covered : ∃ node ∈ certificate.nodes,
      node.state = start ∧ node.depth + actions.length ≤ depth) :
    certificate.Contains final := by
  induction actions generalizing start with
  | nil =>
      have equality := Runs.empty run
      subst final
      rcases covered with ⟨node, member, nodeState, _⟩
      exact ⟨node, member, nodeState⟩
  | cons action actions inductionHypothesis =>
      rcases Runs.uncons run with ⟨next, step, tail⟩
      change model.Step world start action next at step
      rcases covered with ⟨node, nodeMember, nodeState, remaining⟩
      simp only [List.length_cons] at remaining
      have within : node.depth < depth := by omega
      have successorMember : (action, next) ∈ view.successors node.state := by
        apply (view.executable.successors_exact world node.state action next).mpr
        simpa [nodeState] using step
      rcases certificate.valid_depth_closed view property depth valid nodeMember within
          successorMember with ⟨nextNode, nextMember, nextState, nextDepth⟩
      apply inductionHypothesis tail
      exact ⟨nextNode, nextMember, nextState, by omega⟩

theorem Certificate.valid_depth_covers_reachable
    {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (depth : Nat) (certificate : Certificate model world)
    (valid : certificate.Valid view property (.depth depth))
    {state : model.State world} (reachable : ReachableWithin model world depth state) :
    certificate.Contains state := by
  rcases reachable with ⟨initial, actions, initialState, run, within⟩
  rcases certificate.valid_initial view property (.depth depth) valid initialState with
    ⟨node, member, _, nodeState, emptyActions⟩
  apply certificate.valid_depth_covers_run view property depth valid
    (actions := actions) run
  refine ⟨node, member, nodeState, ?_⟩
  simp only [Node.depth, emptyActions, List.length_nil, Nat.zero_add]
  exact within

theorem Certificate.valid_depth_safety
    {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (depth : Nat) (certificate : Certificate model world)
    (valid : certificate.Valid view property (.depth depth)) :
    ∀ state, ReachableWithin model world depth state → property state = true := by
  intro state reachable
  rcases certificate.valid_depth_covers_reachable view property depth valid reachable with
    ⟨node, member, nodeState⟩
  rw [← nodeState]
  exact certificate.valid_safe view property (.depth depth) valid member

theorem Certificate.checked_before_safety
    {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (certificate : Certificate model world) (depth : Nat)
    (checked : certificate.checkBefore view property depth = true) :
    ∀ state, ReachableBefore model world depth state → property state = true := by
  cases depth with
  | zero =>
      intro state reachable
      rcases reachable with ⟨initial, actions, initialState, run, before⟩
      omega
  | succ depth =>
      intro state reachable
      apply (certificate.before (depth + 1)).valid_depth_safety view property depth checked
      rcases reachable with ⟨initial, actions, initialState, run, before⟩
      exact ⟨initial, actions, initialState, run, by omega⟩

end Exact

end Umpire3
