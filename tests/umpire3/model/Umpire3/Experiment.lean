import Lean.Data.Json
import Umpire3.Executable
import Umpire3.Manifest
import Umpire3.Value

namespace Umpire3

structure SemanticResource where
  identifier : String
  kind : String

structure SemanticAction where
  identifier : String
  kind : String
  arguments : List SemanticNamedValue := []
  bindings : List SemanticBinding := []
  requiredCapabilities : List String
  preCheckpoint : Option String
  postCheckpoint : Option String
  responseMode : String := "synchronous"
  maxBlockNanos : Nat := 0

structure SemanticPolicy where
  identifier : String
  kind : String
  scope : List String
  arguments : List SemanticNamedValue := []

structure SemanticFault where
  identifier : String
  kind : String
  policy : Option String := none
  safetyClass : String
  scopeResources : List String
  scopeEndpoints : List String := []
  scopeTaskQueues : List String := []
  scopeServices : List String := []
  scopeRoutes : List String := []
  scopeParticipants : List String := []
  scopeAttempts : List Nat := []
  occurrenceFirst : Nat
  occurrenceCount : Nat
  intervalStartAction : String
  intervalStopAction : String
  arguments : List SemanticNamedValue := []
  requiredCapabilities : List String

structure SemanticOrderConstraint where
  before : String
  after : String
  relation : String

structure SemanticCheckpoint where
  identifier : String
  observation : String
  ordering : String
  omissionPolicy : String

structure SemanticExperiment where
  identifier : String
  modelModules : List String
  sourceRevision : String := "umpire3-v2"
  propertyIdentifier : String
  propertyStatementHash : String
  claim : String := "implementation-conformance"
  scope : ExplorationScope
  strategy : String
  seed : Int := 0
  resources : List SemanticResource
  actions : List SemanticAction
  policies : List SemanticPolicy := []
  faults : List SemanticFault := []
  order : List SemanticOrderConstraint := []
  checkpoints : List SemanticCheckpoint
  provenanceKind : String
  proofManifest : String
  redactionClass : String := "semantic-only"
  maxArtifactBytes : Nat := 1048576

def SemanticExperiment.WellFormed (experiment : SemanticExperiment) : Prop :=
  experiment.identifier ≠ "" ∧
    experiment.modelModules ≠ [] ∧
    experiment.propertyIdentifier ≠ "" ∧
    experiment.propertyStatementHash ≠ "" ∧
    experiment.strategy ≠ "" ∧
    experiment.actions ≠ [] ∧
    experiment.checkpoints ≠ [] ∧
    experiment.provenanceKind ≠ "" ∧
    experiment.proofManifest ≠ "" ∧
    ∀ action ∈ experiment.actions,
      ((action.responseMode = "synchronous" ∨ action.responseMode = "asynchronous" ∨
          action.responseMode = "deferred" ∨ action.responseMode = "failure") ∧
          action.maxBlockNanos = 0 ∨
        action.responseMode = "blocking" ∧ action.maxBlockNanos > 0) ∧
      (∀ checkpoint, action.preCheckpoint = some checkpoint →
        ∃ candidate ∈ experiment.checkpoints, candidate.identifier = checkpoint) ∧
      (∀ checkpoint, action.postCheckpoint = some checkpoint →
        ∃ candidate ∈ experiment.checkpoints, candidate.identifier = checkpoint)

private def stringsJson (values : List String) : Lean.Json :=
  Lean.Json.arr (values.map Lean.toJson).toArray

private def namedValuesJson (values : List SemanticNamedValue) : Lean.Json :=
  Lean.Json.arr (values.map SemanticNamedValue.toJson).toArray

private def bindingsJson (bindings : List SemanticBinding) : Lean.Json :=
  Lean.Json.arr (bindings.map SemanticBinding.toJson).toArray

private def SemanticResource.toJson (resource : SemanticResource) : Lean.Json :=
  Lean.Json.mkObj [("identifier", resource.identifier), ("kind", resource.kind)]

private def SemanticAction.toJson (action : SemanticAction) : Lean.Json :=
  let fields : List (String × Lean.Json) := [
    ("identifier", action.identifier),
    ("kind", action.kind),
    ("arguments", namedValuesJson action.arguments),
    ("bindings", bindingsJson action.bindings),
    ("requiredCapabilities", stringsJson action.requiredCapabilities),
    ("responseMode", action.responseMode),
  ]
  let fields := if action.maxBlockNanos = 0 then fields else
    fields ++ [("maxBlockNanos", Lean.toJson action.maxBlockNanos)]
  let fields := match action.preCheckpoint with
    | none => fields
    | some checkpoint => fields ++ [("preCheckpoint", Lean.Json.str checkpoint)]
  let fields := match action.postCheckpoint with
    | none => fields
    | some checkpoint => fields ++ [("postCheckpoint", Lean.Json.str checkpoint)]
  Lean.Json.mkObj fields

private def SemanticPolicy.toJson (policy : SemanticPolicy) : Lean.Json := Lean.Json.mkObj [
  ("identifier", policy.identifier),
  ("kind", policy.kind),
  ("scope", stringsJson policy.scope),
  ("arguments", namedValuesJson policy.arguments),
]

private def SemanticFault.toJson (fault : SemanticFault) : Lean.Json :=
  let fields : List (String × Lean.Json) := [
    ("identifier", fault.identifier),
    ("kind", fault.kind),
    ("safetyClass", fault.safetyClass),
    ("scope", Lean.Json.mkObj [
      ("resources", stringsJson fault.scopeResources),
      ("endpoints", stringsJson fault.scopeEndpoints),
      ("taskQueues", stringsJson fault.scopeTaskQueues),
      ("services", stringsJson fault.scopeServices),
      ("routes", stringsJson fault.scopeRoutes),
      ("participants", stringsJson fault.scopeParticipants),
      ("attempts", Lean.Json.arr (fault.scopeAttempts.map Lean.toJson).toArray),
    ]),
    ("occurrence", Lean.Json.mkObj [
      ("first", Lean.toJson fault.occurrenceFirst),
      ("count", Lean.toJson fault.occurrenceCount),
    ]),
    ("interval", Lean.Json.mkObj [
      ("startAction", fault.intervalStartAction),
      ("stopAction", fault.intervalStopAction),
    ]),
    ("arguments", namedValuesJson fault.arguments),
    ("requiredCapabilities", stringsJson fault.requiredCapabilities),
  ]
  let fields := match fault.policy with
    | none => fields
    | some policy => fields ++ [("policy", Lean.Json.str policy)]
  Lean.Json.mkObj fields

private def SemanticOrderConstraint.toJson (order : SemanticOrderConstraint) : Lean.Json :=
  Lean.Json.mkObj [
    ("before", order.before),
    ("after", order.after),
    ("relation", order.relation),
  ]

private def SemanticCheckpoint.toJson (checkpoint : SemanticCheckpoint) : Lean.Json :=
  Lean.Json.mkObj [
    ("identifier", checkpoint.identifier),
    ("observation", checkpoint.observation),
    ("ordering", checkpoint.ordering),
    ("omissionPolicy", checkpoint.omissionPolicy),
  ]

private def Assumption.toJson (assumption : Assumption) : Lean.Json :=
  Lean.Json.mkObj [
    ("identifier", assumption.identifier),
    ("statementHash", assumption.statementHash),
  ]

def SemanticExperiment.toJson (experiment : SemanticExperiment)
    (semanticHash catalogHash : String) : Lean.Json := Lean.Json.mkObj [
  ("formatVersion", "umpire3/v2"),
  ("experimentID", experiment.identifier),
  ("model", Lean.Json.mkObj [
    ("modules", stringsJson experiment.modelModules),
    ("sourceRevision", experiment.sourceRevision),
    ("semanticHash", semanticHash),
    ("catalogHash", catalogHash),
    ("leanVersion", leanVersion),
  ]),
  ("property", Lean.Json.mkObj [
    ("identifier", experiment.propertyIdentifier),
    ("statementHash", experiment.propertyStatementHash),
    ("claim", experiment.claim),
  ]),
  ("scope", Lean.Json.mkObj [
    ("bounds", Lean.Json.mkObj [
      ("maxDepth", Lean.toJson experiment.scope.bound.maxDepth),
      ("maxResults", Lean.toJson experiment.scope.bound.maxResults),
    ]),
    ("assumptions", Lean.Json.arr (experiment.scope.assumptions.map Assumption.toJson).toArray),
    ("strategy", experiment.strategy),
    ("seed", Lean.toJson experiment.seed),
  ]),
  ("resources", Lean.Json.arr (experiment.resources.map SemanticResource.toJson).toArray),
  ("actions", Lean.Json.arr (experiment.actions.map SemanticAction.toJson).toArray),
  ("policies", Lean.Json.arr (experiment.policies.map SemanticPolicy.toJson).toArray),
  ("faults", Lean.Json.arr (experiment.faults.map SemanticFault.toJson).toArray),
  ("order", Lean.Json.arr (experiment.order.map SemanticOrderConstraint.toJson).toArray),
  ("checkpoints", Lean.Json.arr (experiment.checkpoints.map SemanticCheckpoint.toJson).toArray),
  ("provenance", Lean.Json.mkObj [
    ("kind", experiment.provenanceKind),
    ("proofManifest", experiment.proofManifest),
  ]),
  ("retention", Lean.Json.mkObj [
    ("redactionClass", experiment.redactionClass),
    ("maxArtifactBytes", Lean.toJson experiment.maxArtifactBytes),
  ]),
]

def SemanticExperiment.json (experiment : SemanticExperiment)
    (semanticHash catalogHash : String) : String :=
  (experiment.toJson semanticHash catalogHash).compress

end Umpire3
