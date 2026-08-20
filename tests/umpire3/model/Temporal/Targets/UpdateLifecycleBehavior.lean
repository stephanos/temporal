import Temporal.System.UpdateTasks
import Umpire3.ExecutableView

namespace Umpire3.Temporal.Targets.UpdateLifecycleBehavior

def featureBehavior : Behavior Unit :=
  Behavior.ofTransitionSystem Temporal.Product.Update.product

def featureBounded : BoundedModel Temporal.Product.Update.product where
  toExecutableModel := Temporal.Product.Update.executable
  initials := [Temporal.Product.Update.initial]
  initial_iff := by
    intro state
    exact List.mem_singleton
  actions := [.request, .accept, .complete]
  action_complete := by
    intro _ action _ _
    cases action <;> simp

def featureExecutable : ExecutableView featureBehavior :=
  ExecutableView.ofBoundedModel featureBounded

def systemBehavior : Behavior Unit :=
  Behavior.ofTransitionSystem Temporal.System.UpdateTasks.system

def systemBounded : BoundedModel Temporal.System.UpdateTasks.system where
  toExecutableModel := Temporal.System.UpdateTasks.executable
  initials := [Temporal.System.UpdateTasks.initial]
  initial_iff := by
    intro state
    exact List.mem_singleton
  actions := [
    .StartUpdate,
    .DispatchWorkflowTask,
    .AcceptUpdate,
    .RecordUpdateHistory,
    .CompleteWorkflowTask,
    .CompleteUpdate,
  ]
  action_complete := by
    intro _ action _ _
    cases action <;> simp

def systemExecutable : ExecutableView systemBehavior :=
  ExecutableView.ofBoundedModel systemBounded

end Umpire3.Temporal.Targets.UpdateLifecycleBehavior
