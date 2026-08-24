import Temporal.Families.NexusCancellation.Refinement
import Temporal.Mechanisms.TaskDelivery
import Umpire3.Experiment
import Umpire3.Manifest

namespace Umpire3.Temporal.Experiments.NexusCancellation

def assumptions : List Assumption := [
  {
    identifier := "persistence-commit-atomicity"
    statementHash := "sha256:61db93cf9c68dfc18c4d379f71dc5b03ae45854bb70c94c1b390e30ba52e50ad"
  },
  {
    identifier := "task-at-least-once-delivery"
    statementHash := "sha256:4bb260aba066d97231a70090468b1c6bd1ad4bba387742a54f8110a484270415"
  },
  {
    identifier := Mechanisms.TaskDelivery.guarantee.identifier
    statementHash := "derived:" ++ Mechanisms.TaskDelivery.guarantee.identifier
  },
]

def proofAssumptions : List SemanticProofDependency := [
  {
    identifier := "persistence-commit-atomicity"
    statementHash := "sha256:61db93cf9c68dfc18c4d379f71dc5b03ae45854bb70c94c1b390e30ba52e50ad"
  },
  {
    identifier := "task-at-least-once-delivery"
    statementHash := "sha256:4bb260aba066d97231a70090468b1c6bd1ad4bba387742a54f8110a484270415"
  },
  {
    identifier := Mechanisms.TaskDelivery.guarantee.identifier
    statementHash := "derived:" ++ Mechanisms.TaskDelivery.guarantee.identifier
  },
]

def proofManifest : SemanticProofManifest where
  identifier := "nexus-cancellation-refinement-v2"
  proof := resolved_simulation% Refinement.NexusCancellationFencing.soundSimulation
  assumptions := proofAssumptions

def experiment : SemanticExperiment where
  identifier := "nexus-cancellation-stale-completion-v1"
  modelModules := [
    "Temporal.Feature.NexusCancellationFencing",
    "Temporal.System.NexusCancellationFencing",
    "Temporal.Refinement.NexusCancellationFencing",
  ]
  propertyIdentifier := "nexus.cancellation.won-excludes-success"
  propertyStatementHash := "derived"
  scope := {
    bound := { maxDepth := 8, maxResults := 10000 }
    assumptions := assumptions
  }
  strategy := "explicit-breadth-first"
  resources := [
    { identifier := "operation", kind := "nexus-operation" },
    { identifier := "worker", kind := "nexus-worker" },
  ]
  actions := [
    { identifier := "a1", kind := "schedule-operation", requiredCapabilities := ["nexus"],
      preCheckpoint := none, postCheckpoint := none },
    { identifier := "a2", kind := "dispatch-task", requiredCapabilities := ["nexus-worker-control"],
      preCheckpoint := none, postCheckpoint := none },
    { identifier := "a3", kind := "request-cancellation", requiredCapabilities := ["nexus"],
      preCheckpoint := none, postCheckpoint := some "cancellation-accepted" },
    { identifier := "a4", kind := "commit-cancellation", requiredCapabilities := ["nexus-observation"],
      preCheckpoint := some "cancellation-accepted", postCheckpoint := some "cancellation-won" },
    { identifier := "a5", kind := "acquire-ownership", requiredCapabilities := ["failover-control"],
      preCheckpoint := none, postCheckpoint := none },
    { identifier := "a6", kind := "retry-task", requiredCapabilities := ["nexus-worker-control"],
      preCheckpoint := none, postCheckpoint := none },
    { identifier := "a7", kind := "worker-returns-success",
      allowedOutcomes := ["applied", "suppressed", "rejected", "retried", "fault-intercepted"],
      requiredCapabilities := ["nexus-worker-control"],
      preCheckpoint := none, postCheckpoint := none },
    { identifier := "a8", kind := "persist-success",
      allowedOutcomes := ["applied", "suppressed", "rejected", "retried", "fault-intercepted"],
      requiredCapabilities := ["nexus-observation"],
      preCheckpoint := some "cancellation-won", postCheckpoint := some "no-stale-success" },
  ]
  policies := [{
    identifier := "ownership-change"
    kind := "during"
    scope := ["a5", "a6", "a7", "a8"]
  }]
  faults := [{
    identifier := "stale-completion"
    kind := "stale-worker-completion"
    policy := some "ownership-change"
    safetyClass := "controlled"
    scopeResources := ["operation", "worker"]
    scopeParticipants := ["worker"]
    scopeAttempts := [1]
    occurrenceFirst := 1
    occurrenceCount := 1
    intervalStartAction := "a5"
    intervalStopAction := "a8"
    requiredCapabilities := ["nexus-worker-control", "failover-control"]
  }]
  order := [
    { before := "a1", after := "a2", relation := "semantic" },
    { before := "a2", after := "a3", relation := "semantic" },
    { before := "a3", after := "a4", relation := "semantic" },
    { before := "a4", after := "a5", relation := "semantic" },
    { before := "a5", after := "a6", relation := "semantic" },
    { before := "a6", after := "a7", relation := "semantic" },
    { before := "a7", after := "a8", relation := "semantic" },
  ]
  checkpoints := [
    { identifier := "cancellation-accepted", observation := "cancellation-accepted",
      ordering := "source-sequence", omissionPolicy := "required" },
    { identifier := "cancellation-won", observation := "cancellation-won",
      ordering := "causal", omissionPolicy := "required" },
    { identifier := "no-stale-success", observation := "stale-success-absent",
      ordering := "causal", omissionPolicy := "required" },
  ]
  provenanceKind := "counterexample"
  proofManifest := "nexus-cancellation-refinement-v2"

theorem experimentWellFormed : experiment.WellFormed := by
  simp [SemanticExperiment.WellFormed, experiment, assumptions]

def main : IO Unit := do
  let some semanticHash ← IO.getEnv "UMPIRE3_SEMANTIC_HASH"
    | throw (IO.userError "UMPIRE3_SEMANTIC_HASH is required")
  let some catalogHash ← IO.getEnv "UMPIRE3_CATALOG_HASH"
    | throw (IO.userError "UMPIRE3_CATALOG_HASH is required")
  IO.println (experiment.json semanticHash catalogHash)

end Umpire3.Temporal.Experiments.NexusCancellation
