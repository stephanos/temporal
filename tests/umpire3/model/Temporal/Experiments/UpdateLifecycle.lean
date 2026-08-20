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

def experiment : SemanticExperiment where
  identifier := "workflow-update-lifecycle-v1"
  modelModules := [
    "Temporal.Product.Update",
    "Temporal.System.UpdateTasks",
    "Temporal.Refinement.UpdateTasks",
    "Temporal.System.TaskDelivery",
  ]
  propertyIdentifier := "workflow-update.accepted-completes-through-history"
  scope := {
    bound := { maxDepth := 6, maxResults := 1000 }
    assumptions := [{
      identifier := System.TaskDelivery.guarantee.identifier
      statementHash := System.TaskDelivery.guarantee.statementHash
    }]
  }
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
  checkpoints := [
    { identifier := "update-accepted", observation := "update-accepted",
      ordering := "source-sequence", omissionPolicy := "required" },
    { identifier := "update-completed", observation := "update-completed",
      ordering := "causal", omissionPolicy := "required" },
  ]
  provenance := "proof"

theorem experimentWellFormed : experiment.WellFormed := by
  simp [SemanticExperiment.WellFormed, experiment, System.TaskDelivery.guarantee]

def json (semanticHash : String) : String :=
  "{" ++
  "\"formatVersion\":\"" ++ formatVersion ++ "\"," ++
  "\"experimentID\":\"workflow-update-lifecycle-v1\"," ++
  "\"model\":{" ++
    "\"modules\":[\"Temporal.Product.Update\",\"Temporal.System.UpdateTasks\",\"Temporal.Refinement.UpdateTasks\",\"Temporal.System.TaskDelivery\"]," ++
    "\"sourceRevision\":\"umpire3-v1\"," ++
    "\"semanticHash\":\"" ++ semanticHash ++ "\"," ++
    "\"leanVersion\":\"" ++ leanVersion ++ "\"}," ++
  "\"property\":{" ++
    "\"identifier\":\"workflow-update.accepted-completes-through-history\"," ++
    "\"statementHash\":\"sha256:d36d6ddd20e5d51e0a4bd591bd2b59ce90f953417042a7b0bc4bff99485c351d\"," ++
    "\"claim\":\"implementation-conformance\"}," ++
  "\"scope\":{" ++
    "\"bounds\":{\"maxDepth\":6,\"maxResults\":1000}," ++
    "\"assumptions\":[{\"identifier\":\"task-delivery.current-completion-only\"," ++
      "\"statementHash\":\"sha256:31e3bc0f50ed17ad15f1848d8fd3cc753e18c21729d606f10fc10d5b71d9bc93\"}]," ++
    "\"strategy\":\"explicit-trace\",\"seed\":0}," ++
  "\"resources\":[{\"identifier\":\"workflow\",\"kind\":\"workflow\"}," ++
    "{\"identifier\":\"update\",\"kind\":\"workflow-update\"}]," ++
  "\"actions\":[" ++
    "{\"identifier\":\"u1\",\"kind\":\"start-update\",\"requiredCapabilities\":[\"update\"]}," ++
    "{\"identifier\":\"u2\",\"kind\":\"dispatch-workflow-task\",\"requiredCapabilities\":[\"workflow-task-control\"]}," ++
    "{\"identifier\":\"u3\",\"kind\":\"accept-update\",\"requiredCapabilities\":[\"update\"],\"postCheckpoint\":\"update-accepted\"}," ++
    "{\"identifier\":\"u4\",\"kind\":\"record-update-history\",\"requiredCapabilities\":[\"history-observation\"],\"preCheckpoint\":\"update-accepted\"}," ++
    "{\"identifier\":\"u5\",\"kind\":\"complete-workflow-task\",\"requiredCapabilities\":[\"workflow-task-control\"]}," ++
    "{\"identifier\":\"u6\",\"kind\":\"complete-update\",\"requiredCapabilities\":[\"update\"],\"postCheckpoint\":\"update-completed\"}]," ++
  "\"checkpoints\":[" ++
    "{\"identifier\":\"update-accepted\",\"observation\":\"update-accepted\",\"ordering\":\"source-sequence\",\"omissionPolicy\":\"required\"}," ++
    "{\"identifier\":\"update-completed\",\"observation\":\"update-completed\",\"ordering\":\"causal\",\"omissionPolicy\":\"required\"}]," ++
  "\"provenance\":{\"kind\":\"proof\",\"proofManifest\":\"update-tasks-refinement-v1\"}," ++
  "\"retention\":{\"redactionClass\":\"semantic-only\",\"maxArtifactBytes\":1048576}" ++
  "}"

def main : IO Unit := do
  let some semanticHash ← IO.getEnv "UMPIRE3_SEMANTIC_HASH"
    | throw (IO.userError "UMPIRE3_SEMANTIC_HASH is required")
  IO.println (json semanticHash)

end Umpire3.Temporal.Experiments.UpdateLifecycle
