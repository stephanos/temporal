import Temporal.Targets.NexusCancellationFencing
import Umpire3.Explorer
import Umpire3.FirstOrderView

namespace Umpire3.Temporal.Targets.NexusCancellationFencing

open Umpire3.Temporal.System.NexusCancellationFencing

private def lifecycleSort : FirstOrderSort where
  identifier := "lifecycle"
  kind := .enumeration
  values := ["open", "cancellation-accepted", "cancelled", "succeeded"]

private def taskSort : FirstOrderSort where
  identifier := "task-stage"
  kind := .enumeration
  values := ["idle", "dispatched", "returned"]

private def epochSort : FirstOrderSort where
  identifier := "epoch"
  kind := .enumeration
  values := ["none", "epoch-0", "epoch-1"]

def field (identifier : String) : FirstOrderTerm := .field identifier

def value (sort identifier : String) : FirstOrderTerm := .value sort identifier

def equalFieldValue (fieldName sortName identifier : String) : FirstOrderFormula :=
  .equal (field fieldName) (value sortName identifier)

def equalFields (left right : String) : FirstOrderFormula :=
  .equal (field left) (field right)

def all : List FirstOrderFormula → FirstOrderFormula
  | [] => .truth
  | formula :: formulas => formulas.foldl FirstOrderFormula.all formula

def any : List FirstOrderFormula → FirstOrderFormula
  | [] => .not .truth
  | formula :: formulas => formulas.foldl FirstOrderFormula.any formula

def setValue (fieldName sortName identifier : String) : FirstOrderUpdate where
  field := fieldName
  value := value sortName identifier

def copyValue (target source : String) : FirstOrderUpdate where
  field := target
  value := field source

def baseFirstOrderActions : List FirstOrderAction := [
  {
    identifier := "dispatch-task"
    guard := all [
      equalFieldValue "task" "task-stage" "idle",
      equalFieldValue "lifecycle" "lifecycle" "open",
    ]
    updates := [
      setValue "task" "task-stage" "dispatched",
      copyValue "worker-epoch" "owner-epoch",
    ]
  },
  {
    identifier := "request-cancellation"
    guard := equalFieldValue "lifecycle" "lifecycle" "open"
    updates := [setValue "lifecycle" "lifecycle" "cancellation-accepted"]
  },
  {
    identifier := "acquire-ownership"
    guard := all [
      any [
        equalFieldValue "lifecycle" "lifecycle" "open",
        equalFieldValue "lifecycle" "lifecycle" "cancellation-accepted",
        equalFieldValue "lifecycle" "lifecycle" "cancelled",
      ],
      equalFieldValue "owner-epoch" "epoch" "epoch-0",
    ]
    updates := [setValue "owner-epoch" "epoch" "epoch-1"]
  },
  {
    identifier := "commit-cancellation"
    guard := equalFieldValue "lifecycle" "lifecycle" "cancellation-accepted"
    updates := [setValue "lifecycle" "lifecycle" "cancelled"]
  },
  {
    identifier := "worker-returns-success"
    guard := equalFieldValue "task" "task-stage" "dispatched"
    updates := [
      setValue "task" "task-stage" "returned",
      copyValue "completion-epoch" "worker-epoch",
    ]
  },
]

def noStaleCompletionFormula : FirstOrderFormula := any [
  .not (equalFieldValue "lifecycle" "lifecycle" "succeeded"),
  equalFields "completion-epoch" "owner-epoch",
]

def firstOrderArtifact (variant : String) (persistGuard : FirstOrderFormula) : FirstOrderArtifact where
  target := "nexus-cancellation"
  property := "nexus.cancellation.won-excludes-success"
  world := "smoke"
  variant := variant
  canonicalModel := "Umpire3.Temporal.System.NexusCancellationFencing.behavior"
  resources := [
    { identifier := "operation", kind := "nexus-operation" },
    { identifier := "worker", kind := "nexus-worker" },
  ]
  liveOnlyActions := ["schedule-operation", "retry-task"]
  activatingFaults := if variant = "sound" then [] else ["stale-worker-completion"]
  bounds := { symbolicDepth := 6, concreteStateLimit := 512 }
  sorts := [lifecycleSort, taskSort, epochSort]
  stateFields := [
    { identifier := "lifecycle", sort := "lifecycle" },
    { identifier := "task", sort := "task-stage" },
    { identifier := "owner-epoch", sort := "epoch" },
    { identifier := "worker-epoch", sort := "epoch" },
    { identifier := "completion-epoch", sort := "epoch" },
  ]
  initial := all [
    equalFieldValue "lifecycle" "lifecycle" "open",
    equalFieldValue "task" "task-stage" "idle",
    equalFieldValue "owner-epoch" "epoch" "epoch-0",
    equalFieldValue "worker-epoch" "epoch" "none",
    equalFieldValue "completion-epoch" "epoch" "none",
  ]
  actions := baseFirstOrderActions ++ [{
    identifier := "persist-success"
    guard := persistGuard
    updates := [setValue "lifecycle" "lifecycle" "succeeded"]
  }]
  invariant := noStaleCompletionFormula

def soundFirstOrderArtifact : FirstOrderArtifact := firstOrderArtifact "sound" (all [
  equalFieldValue "task" "task-stage" "returned",
  equalFields "completion-epoch" "owner-epoch",
  any [
    equalFieldValue "lifecycle" "lifecycle" "open",
    equalFieldValue "lifecycle" "lifecycle" "cancellation-accepted",
  ],
])

def mutatedFirstOrderArtifact : FirstOrderArtifact := firstOrderArtifact
  "stale-completion-guard-removed" (all [
    equalFieldValue "task" "task-stage" "returned",
    .not (equalFieldValue "completion-epoch" "epoch" "none"),
  ])

private def encodeLifecycle : Lifecycle → String
  | .open => "open"
  | .cancellationAccepted => "cancellation-accepted"
  | .cancelled => "cancelled"
  | .succeeded => "succeeded"

private def encodeTask : TaskStage → String
  | .idle => "idle"
  | .dispatched => "dispatched"
  | .returned => "returned"

private def encodeEpoch (epoch : Nat) : String :=
  if epoch = 0 then "epoch-0" else "epoch-1"

private def encodeOptionalEpoch : Option Nat → String
  | none => "none"
  | some epoch => encodeEpoch epoch

def encodeState (state : SystemState) : FirstOrderState where
  fields := [
    { field := "lifecycle", value := encodeLifecycle state.lifecycle },
    { field := "task", value := encodeTask state.task },
    { field := "owner-epoch", value := encodeEpoch state.ownerEpoch },
    { field := "worker-epoch", value := encodeOptionalEpoch state.workerEpoch },
    { field := "completion-epoch", value := encodeOptionalEpoch state.completionEpoch },
  ]

private def boundedOptionalEpoch : Option Nat → Bool
  | none => true
  | some epoch => decide (epoch ≤ 1)

private def admissible (state : SystemState) : Bool :=
  decide (state.ownerEpoch ≤ 1) && boundedOptionalEpoch state.workerEpoch &&
    boundedOptionalEpoch state.completionEpoch

private theorem encodeEpoch_eq_iff {left right : Nat}
    (leftBound : left ≤ 1) (rightBound : right ≤ 1) :
    encodeEpoch left = encodeEpoch right ↔ left = right := by

  have leftCases : left = 0 ∨ left = 1 := by omega
  have rightCases : right = 0 ∨ right = 1 := by omega
  rcases leftCases with rfl | rfl <;> rcases rightCases with rfl | rfl <;> decide

private theorem encodeOptionalEpoch_eq_iff {left right : Option Nat}
    (leftBound : boundedOptionalEpoch left = true)
    (rightBound : boundedOptionalEpoch right = true) :
    encodeOptionalEpoch left = encodeOptionalEpoch right ↔ left = right := by
  cases left with
  | none =>
      cases right with
      | none => simp
      | some right =>
          simp only [boundedOptionalEpoch, decide_eq_true_eq] at rightBound
          have rightCases : right = 0 ∨ right = 1 := by omega
          rcases rightCases with rfl | rfl <;> decide
  | some left =>
      cases right with
      | none =>
          simp only [boundedOptionalEpoch, decide_eq_true_eq] at leftBound
          have leftCases : left = 0 ∨ left = 1 := by omega
          rcases leftCases with rfl | rfl <;> decide
      | some right =>
          simp only [boundedOptionalEpoch, decide_eq_true_eq] at leftBound rightBound
          simp [encodeOptionalEpoch, encodeEpoch_eq_iff leftBound rightBound]

private theorem initialAdmissible (state : SystemState)
    (initialState : behavior.Initial .smoke state) : admissible state = true := by
  change state = initial at initialState
  subst state
  decide

private theorem stepAdmissible (state : SystemState) (action : SystemAction)
    (nextState : SystemState) (stateAdmissible : admissible state = true)
    (step : behavior.Step .smoke state action nextState) : admissible nextState = true := by
  change nextState ∈ next .smoke state action at step
  cases action <;> simp only [next] at step <;> split at step
  all_goals try simp at step
  all_goals
    subst nextState
    simp only [admissible, Bool.and_eq_true, boundedOptionalEpoch] at stateAdmissible ⊢
    simp_all [Umpire3.Temporal.NexusCancellationFencing.World.maxOwnerEpoch]

private theorem artifactInitialPreserved (selected : FirstOrderArtifact)
    (selectedArtifact : selected = soundFirstOrderArtifact ∨
      selected = mutatedFirstOrderArtifact)
    (state : SystemState) (initialState : behavior.Initial .smoke state) :
    selected.initial.eval (encodeState state) = true := by
  change state = initial at initialState
  subst state
  rcases selectedArtifact with rfl | rfl <;> decide

private theorem soundStepPreserved (state : SystemState) (action : SystemAction)
    (nextState : SystemState) (stateAdmissible : admissible state = true)
    (step : behavior.Step .smoke state action nextState) :
    soundFirstOrderArtifact.next (encodeState state) (actionName action) =
      some (encodeState nextState) := by
  change nextState ∈ next .smoke state action at step
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases action <;> simp only [next] at step <;> split at step
  all_goals try simp at step
  all_goals
    subst nextState
    simp only [admissible, Bool.and_eq_true, boundedOptionalEpoch] at stateAdmissible
    simp_all [soundFirstOrderArtifact, firstOrderArtifact, baseFirstOrderActions, actionName,
      all, any, equalFieldValue, equalFields, field, value, setValue, copyValue,
      FirstOrderArtifact.next, FirstOrderAction.apply, FirstOrderFormula.eval,
      FirstOrderTerm.eval, FirstOrderState.read, FirstOrderState.write,
      applyFirstOrderUpdates, encodeState, encodeLifecycle, encodeTask,
      encodeOptionalEpoch, encodeEpoch,
      Umpire3.Temporal.NexusCancellationFencing.World.maxOwnerEpoch]
  all_goals cases lifecycle <;> cases completionEpoch <;> simp_all
  all_goals split <;> simp_all

private theorem mutatedStepPreserved (state : SystemState) (action : SystemAction)
    (nextState : SystemState) (stateAdmissible : admissible state = true)
    (step : mutatedBehavior.Step .smoke state action nextState) :
    mutatedFirstOrderArtifact.next (encodeState state) (actionName action) =
      some (encodeState nextState) := by
  change nextState ∈ mutatedNext .smoke state action at step
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases action <;> simp only [mutatedNext] at step
  all_goals try simp only [next] at step
  all_goals split at step
  all_goals try simp at step
  all_goals
    subst nextState
    simp only [admissible, Bool.and_eq_true, boundedOptionalEpoch] at stateAdmissible
    simp_all [mutatedFirstOrderArtifact, firstOrderArtifact, baseFirstOrderActions, actionName,
      all, any, equalFieldValue, field, value, setValue, copyValue,
      FirstOrderArtifact.next, FirstOrderAction.apply, FirstOrderFormula.eval,
      FirstOrderTerm.eval, FirstOrderState.read, FirstOrderState.write,
      applyFirstOrderUpdates, encodeState, encodeLifecycle, encodeTask,
      encodeOptionalEpoch, encodeEpoch,
      Umpire3.Temporal.NexusCancellationFencing.World.maxOwnerEpoch]
  all_goals cases lifecycle <;> cases completionEpoch <;> simp_all

  all_goals split <;> simp_all

private theorem propertyPreserved (state : SystemState)
    (stateAdmissible : admissible state = true) :
    noStaleCompletionFormula.eval (encodeState state) = noStaleCompletion state := by
  simp only [admissible, Bool.and_eq_true, boundedOptionalEpoch] at stateAdmissible
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  rcases stateAdmissible with ⟨⟨ownerBound, workerBound⟩, completionBound⟩
  simp only [decide_eq_true_eq] at ownerBound
  have ownerCases : ownerEpoch = 0 ∨ ownerEpoch = 1 := by omega
  rcases ownerCases with rfl | rfl
  all_goals cases completionEpoch with
    | none =>
        cases lifecycle <;> simp [noStaleCompletionFormula, noStaleCompletion, any,
          equalFieldValue, equalFields, field, value, FirstOrderFormula.eval,
          FirstOrderTerm.eval, FirstOrderState.read, encodeState, encodeLifecycle,
          encodeOptionalEpoch, encodeEpoch]
    | some completionEpoch =>
        simp only [decide_eq_true_eq] at completionBound
        have completionCases : completionEpoch = 0 ∨ completionEpoch = 1 := by omega
        rcases completionCases with rfl | rfl <;> cases lifecycle <;>
          simp [noStaleCompletionFormula, noStaleCompletion, any, equalFieldValue,
            equalFields, field, value, FirstOrderFormula.eval, FirstOrderTerm.eval,
            FirstOrderState.read, encodeState, encodeLifecycle, encodeOptionalEpoch, encodeEpoch]

private theorem actionComplete (identifier : String)
    (member : identifier ∈ soundFirstOrderArtifact.actionIdentifiers) :
    ∃ action, actionName action = identifier := by
  simp [soundFirstOrderArtifact, firstOrderArtifact, baseFirstOrderActions,
    FirstOrderArtifact.actionIdentifiers] at member
  rcases member with rfl | rfl | rfl | rfl | rfl | rfl
  · exact ⟨.dispatchTask, by simp [actionName]⟩
  · exact ⟨.acceptCancellation, by simp [actionName]⟩
  · exact ⟨.acquireOwnership, by simp [actionName]⟩
  · exact ⟨.commitCancellation, by simp [actionName]⟩
  · exact ⟨.returnSuccess, by simp [actionName]⟩
  · exact ⟨.persistSuccess, by simp [actionName]⟩

def soundFirstOrderView : FirstOrderView behavior .smoke noStaleCompletion where
  artifact := soundFirstOrderArtifact
  encodeState := encodeState
  encodeAction := actionName
  admissible := admissible
  initial_admissible := initialAdmissible
  step_admissible := stepAdmissible
  initial_preserved := artifactInitialPreserved _ (Or.inl rfl)
  step_preserved := soundStepPreserved
  property_preserved := propertyPreserved
  action_injective := soundFiniteView.actionName_injective
  action_total := by intro action; cases action <;> decide
  action_complete := actionComplete
  action_identifiers_unique := by decide

private theorem mutatedStepAdmissible (state : SystemState) (action : SystemAction)
    (nextState : SystemState) (stateAdmissible : admissible state = true)
    (step : mutatedBehavior.Step .smoke state action nextState) : admissible nextState = true := by
  change nextState ∈ mutatedNext .smoke state action at step
  cases action <;> simp only [mutatedNext] at step
  all_goals try simp only [next] at step
  all_goals split at step
  all_goals try simp at step
  all_goals
    subst nextState
    simp only [admissible, Bool.and_eq_true, boundedOptionalEpoch] at stateAdmissible ⊢
    simp_all [Umpire3.Temporal.NexusCancellationFencing.World.maxOwnerEpoch]

def mutatedFirstOrderView : FirstOrderView mutatedBehavior .smoke noStaleCompletion where
  artifact := mutatedFirstOrderArtifact
  encodeState := encodeState
  encodeAction := actionName
  admissible := admissible
  initial_admissible := initialAdmissible
  step_admissible := mutatedStepAdmissible
  initial_preserved := artifactInitialPreserved _ (Or.inr rfl)
  step_preserved := mutatedStepPreserved
  property_preserved := propertyPreserved
  action_injective := mutatedFiniteView.actionName_injective
  action_total := by intro action; cases action <;> decide
  action_complete := by
    intro identifier member
    apply actionComplete identifier
    simpa [mutatedFirstOrderArtifact, soundFirstOrderArtifact, firstOrderArtifact,
      FirstOrderArtifact.actionIdentifiers] using member
  action_identifiers_unique := by decide

def soundSearch := Exact.explore soundFiniteView noStaleCompletion {
  maxDepth := 16
  maxStates := 256
  maxTransitions := 4096
  maxStateBytes := 16384
}

def mutatedReachabilitySearch := Exact.explore mutatedFiniteView (fun _ => true) {
  maxDepth := 16
  maxStates := 256
  maxTransitions := 4096
  maxStateBytes := 16384
}

def soundFirstOrderExport : Option FirstOrderExport :=
  FirstOrderExport.ofSearch (resolved_first_order% soundFirstOrderView)
    soundFiniteView noStaleCompletion soundSearch encodeState

def mutatedFirstOrderExport : Option FirstOrderExport :=
  FirstOrderExport.ofSearch (resolved_first_order% mutatedFirstOrderView)
    mutatedFiniteView (fun _ => true) mutatedReachabilitySearch encodeState

end Umpire3.Temporal.Targets.NexusCancellationFencing
