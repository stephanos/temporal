import Temporal.Families.TaskAcknowledgement.Feature
import Temporal.Families.TaskAcknowledgement.System
import Umpire3.Refinement

namespace Umpire3.Temporal.Refinement.TaskAck

namespace Feature


abbrev State := Feature.TaskAck.State
abbrev Action := Feature.TaskAck.Command
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Feature.TaskAck.feature

end Feature

namespace Protocol

namespace System

abbrev State := Temporal.System.TaskAck.Protocol.State
abbrev Action := Temporal.System.TaskAck.Protocol.Action
abbrev behavior := Temporal.System.TaskAck.Protocol.behavior
abbrev mutatedBehavior := Temporal.System.TaskAck.Protocol.mutatedBehavior
abbrev next := Temporal.System.TaskAck.Protocol.next

end System

def project : System.State → Feature.State
  | .idle => .empty
  | .messageQueued => .queued
  | .deliveryIssued => .delivered
  | .completionStored => .acknowledged
  | .completionStoredWithBacklog => .acknowledgedWithBacklog

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .enqueueMessage => .one .enqueue
  | .issueDelivery => .one .deliver
  | .storeCompletion | .storeCompletionWithoutRemovingBacklog => .one .acknowledge

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Feature.TaskAck.feature (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.TaskAck.Protocol.next] at step
  all_goals subst nextState
  · exact Runs.cons (by trivial) (Runs.nil _)
  · exact Runs.cons (by trivial) (Runs.nil _)
  · exact Runs.cons (by trivial) (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨.empty, rfl, rfl⟩

theorem stepSimulates : StepSimulation System.behavior Feature.behavior Projects actionMap := by
  intro _ systemState featureState action nextSystemState related systemStep
  subst featureState
  exact ⟨project nextSystemState, mappedRun systemStep, rfl⟩

def soundSimulation : SafetySimulation System.behavior Feature.behavior where
  Relates := Projects
  mapAction := actionMap
  initial := fun _ initialState => initialProjects initialState
  step := stepSimulates

theorem mutationBreaksDeclaredSimulation :
    ¬StepSimulation System.mutatedBehavior Feature.behavior Projects actionMap := by
  intro simulation
  have transition : System.mutatedBehavior.Step () .deliveryIssued
      .storeCompletionWithoutRemovingBacklog .completionStoredWithBacklog := by decide
  rcases simulation .deliveryIssued .delivered .storeCompletionWithoutRemovingBacklog
      .completionStoredWithBacklog rfl transition with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState = Feature.TaskAck.State.acknowledgedWithBacklog := projects.symm
  subst nextFeatureState
  rcases run.uncons with ⟨middle, invalidStep, tail⟩
  have middleIsFinal : middle = Feature.TaskAck.State.acknowledgedWithBacklog :=
    Runs.empty tail
  subst middle
  change Feature.TaskAck.feature.Step .delivered .acknowledge .acknowledgedWithBacklog
    at invalidStep
  simp [Feature.TaskAck.step] at invalidStep

end Protocol

namespace History

namespace System

abbrev State := Temporal.System.TaskAck.History.State
abbrev Action := Temporal.System.TaskAck.History.Action
abbrev behavior := Temporal.System.TaskAck.History.behavior
abbrev mutatedBehavior := Temporal.System.TaskAck.History.mutatedBehavior
abbrev next := Temporal.System.TaskAck.History.next

end System

def project : System.State → Feature.State
  | .empty => .empty
  | .scheduledObserved => .queued
  | .startedObserved => .delivered
  | .completedObserved => .acknowledged
  | .completedObservedWithBacklog => .acknowledgedWithBacklog

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .observeScheduled => .one .enqueue
  | .observeStarted => .one .deliver
  | .observeCompleted | .observeCompletedWithoutRemovingBacklog => .one .acknowledge

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Feature.TaskAck.feature (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.TaskAck.History.next] at step
  all_goals subst nextState
  · exact Runs.cons (by trivial) (Runs.nil _)
  · exact Runs.cons (by trivial) (Runs.nil _)
  · exact Runs.cons (by trivial) (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨.empty, rfl, rfl⟩

theorem stepSimulates : StepSimulation System.behavior Feature.behavior Projects actionMap := by
  intro _ systemState featureState action nextSystemState related systemStep
  subst featureState
  exact ⟨project nextSystemState, mappedRun systemStep, rfl⟩

def soundSimulation : SafetySimulation System.behavior Feature.behavior where
  Relates := Projects
  mapAction := actionMap
  initial := fun _ initialState => initialProjects initialState
  step := stepSimulates

theorem mutationBreaksDeclaredSimulation :
    ¬StepSimulation System.mutatedBehavior Feature.behavior Projects actionMap := by
  intro simulation
  have transition : System.mutatedBehavior.Step () .startedObserved
      .observeCompletedWithoutRemovingBacklog .completedObservedWithBacklog := by decide
  rcases simulation .startedObserved .delivered .observeCompletedWithoutRemovingBacklog
      .completedObservedWithBacklog rfl transition with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState = Feature.TaskAck.State.acknowledgedWithBacklog := projects.symm
  subst nextFeatureState
  rcases run.uncons with ⟨middle, invalidStep, tail⟩
  have middleIsFinal : middle = Feature.TaskAck.State.acknowledgedWithBacklog :=
    Runs.empty tail
  subst middle
  change Feature.TaskAck.feature.Step .delivered .acknowledge .acknowledgedWithBacklog
    at invalidStep
  simp [Feature.TaskAck.step] at invalidStep

end History

structure Simulations where
  protocol : SafetySimulation Protocol.System.behavior Feature.behavior
  history : SafetySimulation History.System.behavior Feature.behavior

def soundSimulations : Simulations where
  protocol := Protocol.soundSimulation
  history := History.soundSimulation

theorem mutationsBreakDeclaredSimulations :
    (¬StepSimulation Protocol.System.mutatedBehavior Feature.behavior
      Protocol.Projects Protocol.actionMap) ∧
    (¬StepSimulation History.System.mutatedBehavior Feature.behavior
      History.Projects History.actionMap) :=
  ⟨Protocol.mutationBreaksDeclaredSimulation, History.mutationBreaksDeclaredSimulation⟩

structure NonVacuity where
  protocolGood : Runs (Protocol.System.behavior.at ()) .idle
    [.enqueueMessage, .issueDelivery, .storeCompletion] .completionStored
  protocolBad : Runs (Protocol.System.mutatedBehavior.at ()) .idle
    [.enqueueMessage, .issueDelivery, .storeCompletionWithoutRemovingBacklog]
    .completionStoredWithBacklog
  historyGood : Runs (History.System.behavior.at ()) .empty
    [.observeScheduled, .observeStarted, .observeCompleted] .completedObserved
  historyBad : Runs (History.System.mutatedBehavior.at ()) .empty
    [.observeScheduled, .observeStarted, .observeCompletedWithoutRemovingBacklog]
    .completedObservedWithBacklog
  breaks :
    (¬StepSimulation Protocol.System.mutatedBehavior Feature.behavior
      Protocol.Projects Protocol.actionMap) ∧
    (¬StepSimulation History.System.mutatedBehavior Feature.behavior
      History.Projects History.actionMap)

theorem nonVacuity : NonVacuity where
  protocolGood := Temporal.System.TaskAck.Protocol.completionRun
  protocolBad := Temporal.System.TaskAck.Protocol.backlogRetentionMutationRun
  historyGood := Temporal.System.TaskAck.History.completionRun
  historyBad := Temporal.System.TaskAck.History.backlogRetentionMutationRun
  breaks := mutationsBreakDeclaredSimulations

end Umpire3.Temporal.Refinement.TaskAck
