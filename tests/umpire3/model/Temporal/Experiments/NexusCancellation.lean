import Temporal.Refinement.NexusTasks
import Umpire3.Experiment
import Umpire3.Manifest

namespace Umpire3.Temporal.Experiments.NexusCancellation

open System.NexusTasks

def scheduled : State := { initial with scheduled := true }
def dispatched : State := { scheduled with task := .dispatched, workerEpoch := some 0 }
def cancellationRequested : State := { dispatched with
  cancellation := { noProgress with attempted := true, applied := true }
  visible := .cancellationAccepted }
def cancellationCommitted : State := { cancellationRequested with
  cancellation := { cancellationRequested.cancellation with
    committed := true, persisted := true, observed := true }
  visible := .cancelled }
def ownershipChanged : State := { cancellationCommitted with
  ownerEpoch := 1
  staleWorkerEpoch := some 0 }
def retried : State := { ownershipChanged with
  task := .dispatched
  attempt := 1
  workerEpoch := some 1
  completionEpoch := none }
def staleReturned : State := { retried with
  task := .returned
  staleWorkerEpoch := none
  completionEpoch := some 0
  success := { noProgress with attempted := true, applied := true } }
def unsafePersisted : State := { staleReturned with
  success := { staleReturned.success with
    committed := true, persisted := true, observed := true }
  visible := .succeeded }

def counterexampleActions : List Action := [
  .ScheduleOperation,
  .DispatchTask,
  .RequestCancellation,
  .CommitCancellation,
  .AcquireOwnership,
  .RetryTask,
  .WorkerReturnsSuccess,
  .PersistSuccess,
]

theorem scheduleUnsafe : unsafeSystem.Step initial .ScheduleOperation scheduled := by
  simp [unsafeStep, unsafeNext, next, transitions, scheduled, initial, stutterResult]

theorem dispatchUnsafe : unsafeSystem.Step scheduled .DispatchTask dispatched := by
  simp [unsafeStep, unsafeNext, next, transitions, dispatched, scheduled, initial,
    stutterResult]

theorem requestUnsafe :
    unsafeSystem.Step dispatched .RequestCancellation cancellationRequested := by
  simp [unsafeStep, unsafeNext, next, transitions, cancellationRequested,
    dispatched, scheduled, initial, noProgress, productResult]

theorem commitUnsafe :
    unsafeSystem.Step cancellationRequested .CommitCancellation cancellationCommitted := by
  simp [unsafeStep, unsafeNext, next, transitions, cancellationCommitted,
    cancellationRequested, dispatched, scheduled, initial, noProgress, productResult]

theorem ownershipUnsafe :
    unsafeSystem.Step cancellationCommitted .AcquireOwnership ownershipChanged := by
  simp [unsafeStep, unsafeNext, next, transitions, ownershipChanged,
    cancellationCommitted, cancellationRequested, dispatched, scheduled, initial, noProgress,
    stutterResult]

theorem retryUnsafe : unsafeSystem.Step ownershipChanged .RetryTask retried := by
  simp [unsafeStep, unsafeNext, next, transitions, retried, ownershipChanged,
    cancellationCommitted, cancellationRequested, dispatched, scheduled, initial, noProgress,
    stutterResult]

theorem returnUnsafe : unsafeSystem.Step retried .WorkerReturnsSuccess staleReturned := by
  simp [unsafeStep, unsafeNext, next, transitions, returnedEpoch, staleReturned,
    retried, ownershipChanged, cancellationCommitted, cancellationRequested, dispatched, scheduled,
    initial, noProgress, stutterResult]

theorem persistUnsafe : unsafeSystem.Step staleReturned .PersistSuccess unsafePersisted := by
  simp [unsafeStep, unsafeNext, unsafePersisted, staleReturned, retried,
    ownershipChanged, cancellationCommitted, cancellationRequested, dispatched, scheduled, initial,
    noProgress]

theorem unsafeCounterexample :
    Runs unsafeSystem initial counterexampleActions unsafePersisted :=
  Runs.cons scheduleUnsafe (Runs.cons dispatchUnsafe (Runs.cons requestUnsafe
    (Runs.cons commitUnsafe (Runs.cons ownershipUnsafe (Runs.cons retryUnsafe
      (Runs.cons returnUnsafe (Runs.cons persistUnsafe
        (Runs.nil (model := unsafeSystem) unsafePersisted))))))))

theorem soundModelRejectsCounterexample :
    next staleReturned .PersistSuccess = [] := by
  simp [next, transitions, staleReturned, retried, ownershipChanged, cancellationCommitted,
    cancellationRequested, dispatched, scheduled, initial, noProgress]

theorem explorerFindsUnsafeCounterexample :
    unsafeExecutable.follow [initial] counterexampleActions = [unsafePersisted] := by decide

theorem explorerRejectsSoundCounterexample :
    executable.follow [initial] counterexampleActions = [] := by decide

def experiment : SemanticExperiment where
  identifier := "nexus-cancellation-stale-completion-v1"
  modelModules := [
    "Temporal.Product.Nexus",
    "Temporal.System.NexusTasks",
    "Temporal.Refinement.NexusTasks",
  ]
  propertyIdentifier := "nexus.cancellation.won-excludes-success"
  scope := {
    bound := { maxDepth := 8, maxResults := 10000 }
    assumptions := System.NexusTasks.assumptions
  }
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
    { identifier := "a7", kind := "worker-returns-success", requiredCapabilities := ["nexus-worker-control"],
      preCheckpoint := none, postCheckpoint := none },
    { identifier := "a8", kind := "persist-success", requiredCapabilities := ["nexus-observation"],
      preCheckpoint := some "cancellation-won", postCheckpoint := some "no-stale-success" },
  ]
  checkpoints := [
    { identifier := "cancellation-accepted", observation := "cancellation-accepted",
      ordering := "source-sequence", omissionPolicy := "required" },
    { identifier := "cancellation-won", observation := "cancellation-won",
      ordering := "causal", omissionPolicy := "required" },
    { identifier := "no-stale-success", observation := "stale-success-absent",
      ordering := "causal", omissionPolicy := "required" },
  ]
  provenance := "counterexample"

theorem experimentWellFormed : experiment.WellFormed := by
  simp [SemanticExperiment.WellFormed, experiment, System.NexusTasks.assumptions]

def json (semanticHash : String) : String :=
  "{" ++
  "\"formatVersion\":\"" ++ formatVersion ++ "\"," ++
  "\"experimentID\":\"nexus-cancellation-stale-completion-v1\"," ++
  "\"model\":{" ++
    "\"modules\":[\"Temporal.Product.Nexus\",\"Temporal.System.NexusTasks\",\"Temporal.Refinement.NexusTasks\"]," ++
    "\"sourceRevision\":\"umpire3-v1\"," ++
    "\"semanticHash\":\"" ++ semanticHash ++ "\"," ++
    "\"leanVersion\":\"" ++ leanVersion ++ "\"}," ++
  "\"property\":{" ++
    "\"identifier\":\"nexus.cancellation.won-excludes-success\"," ++
    "\"statementHash\":\"sha256:ee1e668005a68fd1dd72bbd4dd2758d035c89f67f6492223c1b615ef094225d8\"," ++
    "\"claim\":\"implementation-conformance\"}," ++
  "\"scope\":{" ++
    "\"bounds\":{\"maxDepth\":8,\"maxResults\":10000}," ++
    "\"assumptions\":[" ++
      "{\"identifier\":\"persistence-commit-atomicity\",\"statementHash\":\"sha256:61db93cf9c68dfc18c4d379f71dc5b03ae45854bb70c94c1b390e30ba52e50ad\"}," ++
      "{\"identifier\":\"task-at-least-once-delivery\",\"statementHash\":\"sha256:4bb260aba066d97231a70090468b1c6bd1ad4bba387742a54f8110a484270415\"}," ++
      "{\"identifier\":\"task-delivery.current-completion-only\",\"statementHash\":\"sha256:31e3bc0f50ed17ad15f1848d8fd3cc753e18c21729d606f10fc10d5b71d9bc93\"}]," ++
    "\"strategy\":\"explicit-breadth-first\",\"seed\":0}," ++
  "\"resources\":[" ++
    "{\"identifier\":\"operation\",\"kind\":\"nexus-operation\"}," ++
    "{\"identifier\":\"worker\",\"kind\":\"nexus-worker\"}]," ++
  "\"actions\":[" ++
    "{\"identifier\":\"a1\",\"kind\":\"schedule-operation\",\"requiredCapabilities\":[\"nexus\"]}," ++
    "{\"identifier\":\"a2\",\"kind\":\"dispatch-task\",\"requiredCapabilities\":[\"nexus-worker-control\"]}," ++
    "{\"identifier\":\"a3\",\"kind\":\"request-cancellation\",\"requiredCapabilities\":[\"nexus\"],\"postCheckpoint\":\"cancellation-accepted\"}," ++
    "{\"identifier\":\"a4\",\"kind\":\"commit-cancellation\",\"requiredCapabilities\":[\"nexus-observation\"],\"preCheckpoint\":\"cancellation-accepted\",\"postCheckpoint\":\"cancellation-won\"}," ++
    "{\"identifier\":\"a5\",\"kind\":\"acquire-ownership\",\"requiredCapabilities\":[\"failover-control\"]}," ++
    "{\"identifier\":\"a6\",\"kind\":\"retry-task\",\"requiredCapabilities\":[\"nexus-worker-control\"]}," ++
    "{\"identifier\":\"a7\",\"kind\":\"worker-returns-success\",\"requiredCapabilities\":[\"nexus-worker-control\"]}," ++
    "{\"identifier\":\"a8\",\"kind\":\"persist-success\",\"requiredCapabilities\":[\"nexus-observation\"],\"preCheckpoint\":\"cancellation-won\",\"postCheckpoint\":\"no-stale-success\"}]," ++
  "\"checkpoints\":[" ++
    "{\"identifier\":\"cancellation-accepted\",\"observation\":\"cancellation-accepted\",\"ordering\":\"source-sequence\",\"omissionPolicy\":\"required\"}," ++
    "{\"identifier\":\"cancellation-won\",\"observation\":\"cancellation-won\",\"ordering\":\"causal\",\"omissionPolicy\":\"required\"}," ++
    "{\"identifier\":\"no-stale-success\",\"observation\":\"stale-success-absent\",\"ordering\":\"causal\",\"omissionPolicy\":\"required\"}]," ++
  "\"provenance\":{\"kind\":\"counterexample\",\"proofManifest\":\"nexus-tasks-refinement-v1\"}," ++
  "\"retention\":{\"redactionClass\":\"semantic-only\",\"maxArtifactBytes\":1048576}" ++
  "}"

def main : IO Unit := do
  let some semanticHash ← IO.getEnv "UMPIRE3_SEMANTIC_HASH"
    | throw (IO.userError "UMPIRE3_SEMANTIC_HASH is required")
  IO.println (json semanticHash)

end Umpire3.Temporal.Experiments.NexusCancellation
