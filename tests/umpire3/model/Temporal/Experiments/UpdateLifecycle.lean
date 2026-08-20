import Temporal.Refinement.UpdateTasks
import Umpire3.Experiment
import Umpire3.Manifest

namespace Umpire3.Temporal.Experiments.UpdateLifecycle

open System.UpdateTasks

def requestedState : State := { initial with visible := .requested }
def dispatchedState : State := { requestedState with taskDispatched := true }
def acceptedState : State := { dispatchedState with visible := .accepted }
def recordedState : State := { acceptedState with historyRecorded := true }
def taskCompletedState : State := { recordedState with completionEpoch := some 0 }
def completedState : State := { taskCompletedState with visible := .completed }

def actions : List Action := [
  .StartUpdate,
  .DispatchWorkflowTask,
  .AcceptUpdate,
  .RecordUpdateHistory,
  .CompleteWorkflowTask,
  .CompleteUpdate,
]

theorem executableTraceIsValid :
    executable.follow [initial] actions = [completedState] := by decide

def proofManifest : SemanticProofManifest where
  identifier := "update-tasks-refinement-v1"
  proof := resolved_refinement% System.UpdateTasks.updateTasksRefinesProduct
  assumptions := [{
    identifier := System.TaskDelivery.guarantee.identifier
    statementHash := System.TaskDelivery.guarantee.statementHash
  }]

def experiment : SemanticExperiment where
  identifier := "workflow-update-lifecycle-v1"
  modelModules := [
    "Temporal.Product.Update",
    "Temporal.System.UpdateTasks",
    "Temporal.Refinement.UpdateTasks",
    "Temporal.System.TaskDelivery",
  ]
  propertyIdentifier := "workflow-update.accepted-completes-through-history"
  propertyStatementHash := "sha256:d36d6ddd20e5d51e0a4bd591bd2b59ce90f953417042a7b0bc4bff99485c351d"
  scope := {
    bound := { maxDepth := 6, maxResults := 1000 }
    assumptions := [{
      identifier := System.TaskDelivery.guarantee.identifier
      statementHash := System.TaskDelivery.guarantee.statementHash
    }]
  }
  strategy := "explicit-trace"
  resources := [
    { identifier := "workflow", kind := "workflow" },
    { identifier := "update", kind := "workflow-update" },
  ]
  actions := [
    { identifier := "u1", kind := "start-update", requiredCapabilities := ["update"],
      preCheckpoint := none, postCheckpoint := none },
    { identifier := "u2", kind := "dispatch-workflow-task",
      requiredCapabilities := ["workflow-task-control"], preCheckpoint := none,
      postCheckpoint := none },
    { identifier := "u3", kind := "accept-update", requiredCapabilities := ["update"],
      preCheckpoint := none, postCheckpoint := some "update-accepted" },
    { identifier := "u4", kind := "record-update-history",
      requiredCapabilities := ["history-observation"], preCheckpoint := some "update-accepted",
      postCheckpoint := none },
    { identifier := "u5", kind := "complete-workflow-task",
      requiredCapabilities := ["workflow-task-control"], preCheckpoint := none,
      postCheckpoint := none },
    { identifier := "u6", kind := "complete-update", requiredCapabilities := ["update"],
      preCheckpoint := none, postCheckpoint := some "update-completed" },
  ]
  order := [
    { before := "u1", after := "u2", relation := "semantic" },
    { before := "u2", after := "u3", relation := "semantic" },
    { before := "u3", after := "u4", relation := "semantic" },
    { before := "u4", after := "u5", relation := "semantic" },
    { before := "u5", after := "u6", relation := "semantic" },
  ]
  checkpoints := [
    { identifier := "update-accepted", observation := "update-accepted",
      ordering := "source-sequence", omissionPolicy := "required" },
    { identifier := "update-completed", observation := "update-completed",
      ordering := "causal", omissionPolicy := "required" },
  ]
  provenanceKind := "proof"
  proofManifest := "update-tasks-refinement-v1"

theorem experimentWellFormed : experiment.WellFormed := by
  simp [SemanticExperiment.WellFormed, experiment, System.TaskDelivery.guarantee]

def main : IO Unit := do
  let some semanticHash ← IO.getEnv "UMPIRE3_SEMANTIC_HASH"
    | throw (IO.userError "UMPIRE3_SEMANTIC_HASH is required")
  let some catalogHash ← IO.getEnv "UMPIRE3_CATALOG_HASH"
    | throw (IO.userError "UMPIRE3_CATALOG_HASH is required")
  IO.println (experiment.json semanticHash catalogHash)

end Umpire3.Temporal.Experiments.UpdateLifecycle
