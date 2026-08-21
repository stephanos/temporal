import Umpire3.Catalog
import Temporal.Product.Nexus
import Temporal.Product.TaskAck
import Temporal.Refinement.TaskAck
import Temporal.Feature.UpdateLifecycle
import Temporal.Inventory
import Temporal.Refinement.MigratedFamilies

namespace Umpire3.Temporal

private def capability (identifier description : String) : CapabilityDeclaration :=
  { identifier, description }

private def action (identifier description : String)
    (requiredCapabilities : List String) : ActionDeclaration := {
  identifier, description, parameters := [], requiredCapabilities
}

private def actionAfter (identifier description : String) (requiredCapabilities dependencies : List String)
    (projections : List ProjectionDeclaration := []) : ActionDeclaration := {
  identifier, description, requiredCapabilities, dependencies, projections
}

private def actionAfterIdentity (identifier description : String)
    (requiredCapabilities dependencies : List String) (parameter : String) : ActionDeclaration := {
  identifier
  description
  parameters := [{ name := parameter, type := "identity", required := false }]
  requiredCapabilities
  dependencies
}

private def entity (identifier description : String) : EntityDeclaration :=
  { identifier, description }

private def observation (identifier description : String) : ObservationDeclaration :=
  { identifier, description }

private def evidence (identifier description : String) : EvidenceDeclaration :=
  { identifier, description }

private def property (identifier description : String) (requirements : List String)
    (proof : ResolvedTheorem) : PropertyDeclaration :=
  (RegisteredProperty.mk identifier description requirements proof).declaration

private def module (identifier description : String) : ModuleDeclaration :=
  { identifier, description }

def catalog : SemanticCatalog where
  version := "temporal/umpire3/catalog/v1"
  types := [
    { identifier := "string", kind := "scalar", description := "Unicode text" },
    { identifier := "integer", kind := "scalar", description := "Signed integer" },
    { identifier := "boolean", kind := "scalar", description := "Boolean" },
    { identifier := "duration", kind := "scalar", description := "Nanosecond duration" },
    { identifier := "enum", kind := "scalar", description := "Named or unknown enum number" },
    { identifier := "bytes-digest", kind := "scalar", description := "Digest of redacted bytes" },
    { identifier := "identity", kind := "symbol", description := "Runtime-grounded entity identity" },
    { identifier := "list", kind := "collection", description := "Ordered typed values" },
    { identifier := "record", kind := "collection", description := "Named typed values" },
  ]
  capabilities := [
    capability "nexus" "Nexus operation authority",
    capability "nexus-worker-control" "Nexus worker task authority",
    capability "nexus-observation" "Nexus state observation",
    capability "failover-control" "Ownership failover authority",
    capability "update" "Workflow Update authority",
    capability "workflow-task-control" "Workflow Task authority",
    capability "history-observation" "Workflow history observation",
    capability "fault-rpc" "Scoped RPC and HTTP fault authority",
    capability "fault-process" "Isolated participant process fault authority",
    capability "fault-network" "Isolated network partition authority",
    capability "fault-clock" "Isolated clock-control authority",
    capability "fault-persistence" "Selected persistence fault authority",
  ]
  actions := [
    actionAfter "schedule-operation" "Schedule a Nexus operation" ["nexus"] []
      [{ name := "operation-id", type := "identity" }],
    actionAfter "dispatch-task" "Dispatch a Nexus task" ["nexus-worker-control"]
      ["schedule-operation"] [{ name := "operation-id", type := "identity" }],
    actionAfterIdentity "worker-returns-success" "Return Nexus worker success" ["nexus-worker-control"]
      ["dispatch-task"] "operation",
    {
      identifier := "request-cancellation"
      description := "Request Nexus cancellation"
      parameters := [{ name := "reason", type := "string", required := false }]
      dependencies := ["dispatch-task"]
      requiredCapabilities := ["nexus"]
    },
    actionAfterIdentity "commit-cancellation" "Commit Nexus cancellation" ["nexus-observation"]
      ["request-cancellation"] "operation",
    actionAfterIdentity "persist-success" "Persist Nexus success" ["nexus-observation"]
      ["worker-returns-success"] "operation",
    actionAfterIdentity "retry-task" "Retry Nexus task delivery" ["nexus-worker-control"]
      ["acquire-ownership"] "operation",
    actionAfterIdentity "acquire-ownership" "Acquire a new ownership epoch" ["failover-control"]
      ["commit-cancellation"] "operation",
    action "crash-owner" "Crash the current owner" ["failover-control"],
    actionAfter "recover-owner" "Recover an owner" ["failover-control"] ["crash-owner"],
    actionAfterIdentity "ack-task" "Acknowledge task delivery" ["nexus-worker-control"]
      ["dispatch-task"] "operation",
    actionAfter "start-update" "Start a Workflow Update" ["update"] []
      [{ name := "update-id", type := "identity" }],
    actionAfterIdentity "accept-update" "Accept a Workflow Update" ["update"]
      ["dispatch-workflow-task"] "update",
    actionAfterIdentity "complete-update" "Complete a Workflow Update" ["update"]
      ["complete-workflow-task"] "update",
    actionAfterIdentity "record-update-history" "Record Update history" ["history-observation"]
      ["accept-update"] "update",
    actionAfterIdentity "dispatch-workflow-task" "Dispatch a Workflow Task" ["workflow-task-control"]
      ["start-update"] "update",
    actionAfterIdentity "complete-workflow-task" "Complete a Workflow Task" ["workflow-task-control"]
      ["record-update-history"] "update",
  ] ++ Product.TaskAck.declaration.actions ++ Inventory.actions
  entities := [
    entity "nexus-operation" "Nexus operation",
    entity "nexus-worker" "Nexus task worker",
    entity "workflow" "Workflow execution",
    entity "workflow-update" "Workflow Update",
  ] ++ [Product.TaskAck.declaration.entity] ++ Inventory.entities
  relations := [{
    identifier := "task-delivery.current-completion"
    source := "nexus-worker"
    target := "nexus-operation"
    description := "A completion belongs to the current ownership epoch"
  }] ++ Inventory.relations
  observations := [
    observation "cancellation-accepted" "Cancellation was accepted",
    observation "cancellation-won" "Cancellation became authoritative",
    observation "stale-success-absent" "No stale success became visible",
    observation "update-accepted" "Workflow Update was accepted",
    observation "update-completed" "Workflow Update completed",
  ] ++ Product.TaskAck.declaration.observations ++ Inventory.observations
  evidence := [
    evidence "source-sequence" "Authoritative order from one source",
    evidence "causal" "Explicit causal reference",
    evidence "identity-lineage" "Grounded entity identity and lineage",
  ]
  properties := [
    property "nexus.cancellation.won-excludes-success"
      "Cancellation excludes a later stale success"
      ["causal", "identity-lineage"]
      (resolved_theorem% Umpire3.Temporal.Product.Nexus.cancellation_won_excludes_success),
    property "workflow-update.accepted-completes-through-history"
      "Accepted Update completion is represented in history"
      ["source-sequence", "identity-lineage"]
      (resolved_theorem% Umpire3.Temporal.Feature.UpdateLifecycle.historyBackedSafe),
  ] ++ Product.TaskAck.declaration.properties ++ Inventory.properties
  policies := [{ identifier := "during", description := "Policy active during a bounded action interval" }]
  faults := [{
    identifier := "stale-worker-completion"
    description := "A stale worker returns completion after ownership changes"
    safetyClass := "controlled"
    scopeDimensions := ["namespace", "participant", "attempt", "interval"]
    requiredCapabilities := ["nexus-worker-control", "failover-control"]
  }, {
    identifier := "drop"
    description := "Drop a selected RPC or HTTP occurrence"
    safetyClass := "controlled"
    scopeDimensions := ["namespace", "service", "route", "occurrence", "interval"]
    requiredCapabilities := ["fault-rpc"]
  }, {
    identifier := "delay"
    description := "Delay a selected RPC or HTTP occurrence"
    safetyClass := "controlled"
    scopeDimensions := ["namespace", "service", "route", "occurrence", "interval"]
    requiredCapabilities := ["fault-rpc"]
  }, {
    identifier := "duplicate"
    description := "Duplicate a selected RPC or HTTP occurrence"
    safetyClass := "controlled"
    scopeDimensions := ["namespace", "service", "route", "occurrence", "interval"]
    requiredCapabilities := ["fault-rpc"]
  }, {
    identifier := "reorder"
    description := "Reorder selected RPC or HTTP occurrences"
    safetyClass := "controlled"
    scopeDimensions := ["namespace", "service", "route", "occurrence", "interval"]
    requiredCapabilities := ["fault-rpc"]
  }, {
    identifier := "hold-release"
    description := "Hold and deterministically release a selected occurrence"
    safetyClass := "controlled"
    scopeDimensions := ["namespace", "route", "occurrence", "interval"]
    requiredCapabilities := ["fault-rpc"]
  }, {
    identifier := "rejection"
    description := "Reject a selected RPC or HTTP occurrence"
    safetyClass := "controlled"
    scopeDimensions := ["namespace", "service", "route", "occurrence", "interval"]
    requiredCapabilities := ["fault-rpc"]
  }, {
    identifier := "process-crash"
    description := "Crash an isolated participant process"
    safetyClass := "controlled"
    scopeDimensions := ["namespace", "participant", "occurrence", "interval"]
    requiredCapabilities := ["fault-process"]
  }, {
    identifier := "restart"
    description := "Restart an isolated participant process"
    safetyClass := "controlled"
    scopeDimensions := ["namespace", "participant", "occurrence", "interval"]
    requiredCapabilities := ["fault-process"]
  }, {
    identifier := "partition"
    description := "Partition selected isolated endpoints"
    safetyClass := "restricted"
    scopeDimensions := ["namespace", "endpoint", "service", "interval"]
    requiredCapabilities := ["fault-network"]
  }, {
    identifier := "failover"
    description := "Trigger a selected ownership failover"
    safetyClass := "controlled"
    scopeDimensions := ["namespace", "service", "occurrence", "interval"]
    requiredCapabilities := ["failover-control"]
  }, {
    identifier := "clock-skew"
    description := "Skew an isolated participant clock"
    safetyClass := "restricted"
    scopeDimensions := ["namespace", "participant", "interval"]
    requiredCapabilities := ["fault-clock"]
  }, {
    identifier := "persistence-error"
    description := "Inject an approved selected persistence error"
    safetyClass := "restricted"
    scopeDimensions := ["namespace", "service", "occurrence", "interval"]
    requiredCapabilities := ["fault-persistence"]
  }]
  modules := [
    module "Temporal.Product.Nexus" "Nexus product contract",
    module "Temporal.System.NexusTasks" "Nexus task mechanism",
    module "Temporal.Refinement.NexusTasks" "Nexus system refinement",
    module "Temporal.Feature.UpdateLifecycle" "History-backed Workflow Update contract",
    module "Temporal.System.MigratedFamilies.UpdateLifecycle" "Independent Workflow Update mechanism",
    module "Temporal.Refinement.MigratedFamilies.UpdateLifecycle" "Workflow Update mechanism refinement",
    module "Temporal.System.TaskDelivery" "Shared current-completion delivery guarantee",
  ] ++ [Product.TaskAck.declaration.module] ++ Inventory.modules
    ++ [
      module "Temporal.System.TaskAck" "Independent Workflow Task acknowledgement mechanisms",
      module "Temporal.Refinement.TaskAck" "Workflow Task acknowledgement refinements",
    ]
  targets := [
    {
      identifier := "nexus-cancellation"
      modules := ["Temporal.Product.Nexus", "Temporal.System.NexusTasks", "Temporal.Refinement.NexusTasks"]
      properties := ["nexus.cancellation.won-excludes-success"]
    },
    {
      identifier := "workflow-update-lifecycle"
      modules := ["Temporal.Feature.UpdateLifecycle", "Temporal.System.TaskDelivery",
        "Temporal.System.MigratedFamilies.UpdateLifecycle",
        "Temporal.Refinement.MigratedFamilies.UpdateLifecycle"]
      properties := ["workflow-update.accepted-completes-through-history"]
    },
  ] ++ [Product.TaskAck.declaration.target] ++ Inventory.targets

set_option maxRecDepth 100000 in
theorem catalogWellFormed : catalog.WellFormed := by rfl

def json (semanticHash : String) : String :=
  (catalog.toJson semanticHash).compress

def main : IO Unit := do
  let some semanticHash ← IO.getEnv "UMPIRE3_SEMANTIC_HASH"
    | throw (IO.userError "UMPIRE3_SEMANTIC_HASH is required")
  IO.println (json semanticHash)

end Umpire3.Temporal
