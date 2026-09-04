import Umpire.Case.Value

/-!
The bounded multi-context Program IR.

Each entrypoint is a Host-activated DAG in exactly one execution context. Instructions and their
outcomes are closed data; endpoint coordinates, clients, credentials, and callbacks are absent.
-/

namespace Umpire.Case

/-- The symbolic environment roles a Program may require from a Host. -/
inductive SymbolicRoleKind where
  | endpoint
  | worker
  | taskQueue
  | participant
  deriving BEq, DecidableEq, Repr

/-- One symbolic role whose concrete binding remains Host-owned. -/
structure ProgramRole where
  roleId : String
  kind : SymbolicRoleKind
  deriving BEq, DecidableEq, Repr

/-- The four isolated execution contexts supported by version one. -/
inductive EntrypointContext where
  | controller
  | workflow
  | activity
  | nexusHandler
  deriving BEq, DecidableEq, Repr

/-- Host activation data for a controller entrypoint. -/
structure ControllerActivation where
  deriving BEq, DecidableEq, Repr

/-- Symbolic Host activation data for a workflow entrypoint. -/
structure WorkflowActivation where
  workflowType : String
  workerRoleId : String
  taskQueueRoleId : String
  deriving BEq, DecidableEq, Repr

/-- Symbolic Host activation data for an activity entrypoint. -/
structure ActivityActivation where
  activityType : String
  workerRoleId : String
  taskQueueRoleId : String
  deriving BEq, DecidableEq, Repr

/-- Symbolic Host activation data for a Nexus-handler entrypoint. -/
structure NexusHandlerActivation where
  service : String
  operation : String
  workerRoleId : String
  taskQueueRoleId : String
  deriving BEq, DecidableEq, Repr

/-- The closed Host activation binding for one entrypoint context. -/
inductive ActivationBinding where
  | controller (activation : ControllerActivation := {})
  | workflow (activation : WorkflowActivation)
  | activity (activation : ActivityActivation)
  | nexusHandler (activation : NexusHandlerActivation)
  deriving BEq, Repr

/-- One typed expression assigned to a descriptor-resolved request path. -/
structure RequestAssignment where
  target : FieldPath
  value : ValueExpression
  deriving BEq, Repr

/-- Whether a response projection selects one value or emits every repeated element in order. -/
inductive ProjectionCardinality where
  | one
  | emitEach
  deriving BEq, DecidableEq, Repr

/-- A response projection destination. -/
inductive ProjectionSink where
  | slot (slotId : String)
  | observation (observationId : String)
  deriving BEq, DecidableEq, Repr

/-- One typed response path projected to Slots, Observations, or both. -/
structure ResponseProjection where
  source : FieldPath
  cardinality : ProjectionCardinality := .one
  sinks : List ProjectionSink
  deriving BEq, Repr

/-- The closed generic outcomes an instruction may expose to later guards. -/
inductive InstructionOutcomeStatus where
  | succeeded
  | protocolNonSuccess
  | sdkFailure
  | timedOut
  | canceled
  deriving BEq, DecidableEq, Repr

/-- One instruction outcome containing only fields declared by its schema. -/
structure InstructionOutcome where
  status : InstructionOutcomeStatus
  protocolCode : String := ""
  sdkFailureCode : String := ""
  detail : String := ""
  value : Option Value := none
  deriving BEq, Repr

/-- Per-instruction resource bounds checked during preparation. -/
structure InstructionBounds where
  timeoutMilliseconds : Nat
  maxAttempts : Nat
  maxEmittedEvents : Nat
  maxResponseBytes : Nat := 0
  deriving BEq, DecidableEq, Repr

/-- A descriptor-resolved, Host-authorized unary protobuf call. -/
structure InvokeRPC where
  endpointRoleId : String
  method : String
  requestAssignments : List RequestAssignment
  responseProjections : List ResponseProjection
  deriving BEq, Repr

/-- A bounded wait for one private or ordinary Slot to become available. -/
structure AwaitSlot where
  slotId : String
  deriving BEq, DecidableEq, Repr

/-- A controller-side Nexus completion using opaque Host-owned authority. -/
structure CompleteNexusOperation where
  capabilitySlotId : String
  result : ValueExpression
  deriving BEq, Repr

/-- A workflow-SDK Nexus operation start. -/
structure StartNexusOperation where
  endpointRoleId : String
  service : String
  operation : String
  input : ValueExpression
  deriving BEq, Repr

/-- A workflow-SDK wait for one earlier instruction outcome. -/
structure Await where
  instruction : InstructionReference
  deriving BEq, DecidableEq, Repr

/-- A workflow completion value. -/
structure Finish where
  result : ValueExpression
  deriving BEq, Repr

/-- The closed Nexus-handler response forms. -/
inductive NexusResponseKind where
  | synchronous
  | asynchronous
  | error
  deriving BEq, DecidableEq, Repr

/-- A Nexus-handler response and optional opaque completion-authority publication. -/
structure RespondNexus where
  kind : NexusResponseKind
  result : ValueExpression
  capabilitySlotId : String := ""
  deriving BEq, Repr

/-- The complete version-one Program instruction table. -/
inductive Instruction where
  | invokeRPC (instruction : InvokeRPC)
  | awaitSlot (instruction : AwaitSlot)
  | completeNexusOperation (instruction : CompleteNexusOperation)
  | startNexusOperation (instruction : StartNexusOperation)
  | awaitOutcome (instruction : Await)
  | finish (instruction : Finish)
  | respondNexus (instruction : RespondNexus)
  deriving BEq, Repr

/-- Worker activation authority reserved before one controller effect is dispatched. -/
structure ActivationReservation where
  entrypointId : String
  count : Nat
  deriving BEq, DecidableEq, Repr

/-- One instruction and its context-local dependencies, guard, outcome schema, and bounds. -/
structure InstructionNode where
  instructionId : String
  dependencies : List InstructionReference
  guard : Option ValueExpression := none
  instruction : Instruction
  outcome : InstructionOutcomeSchema
  bounds : InstructionBounds
  activationReservations : List ActivationReservation := []
  deriving BEq, Repr

/-- One Host-activated, context-local acyclic instruction graph. -/
structure Entrypoint where
  entrypointId : String
  context : EntrypointContext
  activation : ActivationBinding
  nodes : List InstructionNode
  deriving BEq, Repr

/-- The always-run cleanup DAG and the context in which the Host activates it. -/
structure CleanupGraph where
  entrypointId : String
  context : EntrypointContext
  nodes : List InstructionNode
  deriving BEq, Repr

/-- Global Program ceilings checked before target I/O. -/
structure ProgramLimits where
  maxEntrypoints : Nat
  maxNodes : Nat
  maxEdges : Nat
  maxActivations : Nat
  maxAttempts : Nat
  maxRunEvents : Nat
  maxExpressionDepth : Nat
  maxPathFanout : Nat
  maxRequestBytes : Nat
  maxResponseBytes : Nat
  maxTotalDurationMilliseconds : Nat
  maxCleanupDurationMilliseconds : Nat
  deriving BEq, DecidableEq, Repr

/-- A bounded collection of context-local DAGs and one always-run cleanup graph. -/
structure Program where
  programId : String
  roles : List ProgramRole
  slots : List SlotSchema
  observations : List ObservationSchema
  entrypoints : List Entrypoint
  cleanup : CleanupGraph
  limits : ProgramLimits
  deriving BEq, Repr

end Umpire.Case
