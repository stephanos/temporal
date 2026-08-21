import Temporal.Refinement.TaskAck

namespace Umpire3.Temporal.TaskAck.Tests

example : Safety Product.TaskAck.product Product.TaskAck.AcknowledgementConsistent :=
  Product.TaskAck.acknowledged_removes_backlog

example : SafetySimulation Refinement.TaskAck.Protocol.System.behavior
    Refinement.TaskAck.Feature.behavior :=
  Refinement.TaskAck.Protocol.soundSimulation

example : SafetySimulation Refinement.TaskAck.History.System.behavior
    Refinement.TaskAck.Feature.behavior :=
  Refinement.TaskAck.History.soundSimulation

example : Runs (System.TaskAck.Protocol.behavior.at ()) .idle
    [.enqueueMessage, .issueDelivery, .storeCompletion] .completionStored :=
  System.TaskAck.Protocol.completionRun

example : Runs (System.TaskAck.History.behavior.at ()) .empty
    [.observeScheduled, .observeStarted, .observeCompleted] .completedObserved :=
  System.TaskAck.History.completionRun

example : Runs (System.TaskAck.Protocol.mutatedBehavior.at ()) .idle
    [.enqueueMessage, .issueDelivery, .storeCompletionWithoutRemovingBacklog]
    .completionStoredWithBacklog :=
  System.TaskAck.Protocol.backlogRetentionMutationRun

example : Runs (System.TaskAck.History.mutatedBehavior.at ()) .empty
    [.observeScheduled, .observeStarted, .observeCompletedWithoutRemovingBacklog]
    .completedObservedWithBacklog :=
  System.TaskAck.History.backlogRetentionMutationRun

example :
    (¬StepSimulation Refinement.TaskAck.Protocol.System.mutatedBehavior
      Refinement.TaskAck.Feature.behavior Refinement.TaskAck.Protocol.Projects
      Refinement.TaskAck.Protocol.actionMap) ∧
    (¬StepSimulation Refinement.TaskAck.History.System.mutatedBehavior
      Refinement.TaskAck.Feature.behavior Refinement.TaskAck.History.Projects
      Refinement.TaskAck.History.actionMap) :=
  Refinement.TaskAck.mutationsBreakDeclaredSimulations

end Umpire3.Temporal.TaskAck.Tests
