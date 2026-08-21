import Temporal.Refinement.MigratedFamilies
import Umpire3.Experiment
import Umpire3.Manifest

namespace Umpire3.Temporal.Experiments.UpdateLifecycle

open System.MigratedFamilies.UpdateLifecycle

def actions : List Action := [
  .start,
  .dispatchTask,
  .accept,
  .recordHistory,
  .completeWorkflowTask,
  .complete,
]

theorem executableTraceIsValid :
    executable.follow () [initial] actions = [State.completed] := by decide

def proofManifest : SemanticProofManifest where
  identifier := "update-lifecycle-simulation-v1"
  proof := resolved_simulation% Refinement.MigratedFamilies.UpdateLifecycle.soundSimulation
  assumptions := [{
    identifier := System.TaskDelivery.guarantee.identifier
    statementHash := "derived:" ++ System.TaskDelivery.guarantee.identifier
  }]

def experiment : SemanticExperiment where
  identifier := "workflow-update-lifecycle-v1"
  modelModules := [
    "Temporal.Feature.UpdateLifecycle",
    "Temporal.System.TaskDelivery",
    "Temporal.System.MigratedFamilies.UpdateLifecycle",
    "Temporal.Refinement.MigratedFamilies.UpdateLifecycle",
  ]
  propertyIdentifier := "workflow-update.accepted-completes-through-history"
  propertyStatementHash := "derived"
  scope := {
    bound := { maxDepth := 6, maxResults := 1000 }
    assumptions := [{
      identifier := System.TaskDelivery.guarantee.identifier
      statementHash := "derived:" ++ System.TaskDelivery.guarantee.identifier
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
  proofManifest := "update-lifecycle-simulation-v1"

theorem experimentWellFormed : experiment.WellFormed := by
  simp [SemanticExperiment.WellFormed, experiment, System.TaskDelivery.guarantee]

def main : IO Unit := do
  let some semanticHash ← IO.getEnv "UMPIRE3_SEMANTIC_HASH"
    | throw (IO.userError "UMPIRE3_SEMANTIC_HASH is required")
  let some catalogHash ← IO.getEnv "UMPIRE3_CATALOG_HASH"
    | throw (IO.userError "UMPIRE3_CATALOG_HASH is required")
  IO.println (experiment.json semanticHash catalogHash)

end Umpire3.Temporal.Experiments.UpdateLifecycle
