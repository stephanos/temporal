import Temporal.API
import Temporal.Shared
import Temporal.Testpilot.CaseSupport
import Umpire.Case.Compiler
import Umpire.Case.Scoped
import Umpire.Case.Observed
import Umpire.Property.Elab
import Umpire.Property.Evaluate
import Umpire.Property.Scoped
import Umpire.Property.Scoped
import Umpire.Observation.Projection.Coverage
import Umpire.Model.Table

/-!
# Two workflow-owned Nexus operations qualified by captured field relations

The workflow schedules two Nexus operations on one endpoint and awaits both. Each is a separate
modeled operation identity, and this module keeps that identity typed on both sides of the
boundary a Nexus operation actually has.

The submission is an SDK command, not an RPC: `Umpire.Operation.sdkCommand` declares one identity
per scheduled operation, and `Umpire.Operation.event` declares the two semantic events (scheduled
and completed) the operation later records. None of the three pretends to be a unary RPC, and no
generated RPC stands in for one. The RPCs this Case references are environment work: the workflow
the operations run in is started through `StartWorkflowExecution`, and `GetWorkflowExecutionHistory`
supplies the generated response schema through which every event field below is read. Neither
schedules, starts or completes an operation.

Two independent requirements sit on that evidence, and they fail in different ways:

* The **Link** is the scoped clause's correlation. An operation retains the operation identity its
  own scheduled evidence recorded, keyed by the operation, and a later step belongs to that
  operation only when the retained identity is one of the two the model declares. A step carrying
  an identity the model never declared is not one of this operation's semantic steps at all, so it
  fails admission rather than reporting a product violation.
* The **authored field requirement** is the same-step clause. A completion must reference the
  scheduled event its own operation was scheduled at: the `scheduled_event_id` the completed event
  records equals the `event_id` the operation's prior state carries. The Target owns both the
  correlated and the crossed completion, so selecting the await Action never selects which one
  arrives, and the crossed one is a violation of this clause rather than a rejection.

The bounded response is the scoped clause itself: a scheduled operation must complete within its
declared window of semantic transitions. A run that has only been scheduled is unresolved, and a
completion that arrives after the window closes is violated -- neither answer is manufactured from
a synthetic deadline.

The Contract the Case carries reads exactly the fields the authored requirement names: every field
path in a monitor rule is `Umpire.Case.Observed.pathOf` applied to the same `PropertyFieldPath` the
model Property compares, so editing a coordinate moves the runtime read with it. The rule structure
around those paths -- its states, its capture, the presence checks over the derived value paths and
the operation literal each rule matches -- is authored here. The declared Observation carries one
history event, so a model coordinate that only walks the response wrapper around that event has no
derived read path at all.

The bounded-response clause has no runtime counterpart in this Case: the Driver evaluates a scoped
capability only from declared `ScopedEvidence` Observations, which no instruction of this Program
emits. The window is therefore model-only, and the Case declares that as a Known Gap rather than
letting its provenance imply an online reading it does not have.
-/

namespace Temporal.Feature.Nexus3.TypedNexus

open Umpire
open Umpire.Operation
open Umpire.Value
open Temporal.Testpilot.CaseSupport
open Testpilot.Authoring
open temporal.server.api.testpilot.v1

/-! ### The one generated reference: history observation -/

/-- The generated unary method whose response supplies every observed event field below. Fetching
history is observation work; it does not schedule, start or complete an operation. -/
abbrev historyMethod := Temporal.Api.Workflowservice.V1.WorkflowService.getWorkflowExecutionHistory

def historyReference : Temporal.API.MethodReference historyMethod := by constructor

def historyWitness : Temporal.API.rpcOwner.Witness
    Temporal.Api.Workflowservice.V1.GetWorkflowExecutionHistoryRequest
    Temporal.Api.Workflowservice.V1.GetWorkflowExecutionHistoryResponse :=
  ⟨historyMethod, historyReference⟩

/-- Admit the generated history declaration against the generator's own structural selection. -/
def historyBinding : Except Operation.Error
    (CheckedRpc Temporal.API.rpcOwner historyWitness) :=
  Temporal.API.bindUnary historyMethod historyReference

def historySchema : RpcSchema := Temporal.API.rpcOwner.schema historyWitness

def source : SourceLocation :=
  Temporal.Shared.sourceLocation "Temporal/Feature/Nexus3/TypedNexus.lean"

/-- Semantic value bounds for the admitted history payloads. -/
def valueLimits : Limits := ⟨16, 20000000, 262144, 512⟩

/-! ### Structural coordinates, taken from the generated descriptors -/

def historyResponseRoot := "temporal.api.workflowservice.v1.GetWorkflowExecutionHistoryResponse"
def historyNode := "temporal.api.history.v1.History"
def historyEventNode := "temporal.api.history.v1.HistoryEvent"
def scheduledAttributesNode := "temporal.api.history.v1.NexusOperationScheduledEventAttributes"
def completedAttributesNode := "temporal.api.history.v1.NexusOperationCompletedEventAttributes"

/-- The oneof group every history event's attributes belong to. -/
def attributesGroup := "attributes"

/-! ### The two modeled operation identities -/

/-- The Nexus service both operations are addressed to. -/
def nexusService := "umpire.case.service"

/-- The two Nexus operations the workflow schedules, in declaration order. -/
def firstOperation := "complete"
def secondOperation := "confirm"

def firstCommandId : DefinitionId := .of "temporal.nexus3.typed-nexus.command.schedule-complete"
def secondCommandId : DefinitionId := .of "temporal.nexus3.typed-nexus.command.schedule-confirm"
def scheduledEventDeclarationId : DefinitionId :=
  .of "temporal.nexus3.typed-nexus.event.operation-scheduled"
def completedEventDeclarationId : DefinitionId :=
  .of "temporal.nexus3.typed-nexus.event.operation-completed"

/-- One scheduled operation's declared SDK command. Its submission, return and error signature is
declared separately from any RPC, and its identity is what the Link correlation retains. -/
def scheduleCommand (identity : DefinitionId) :
    Except Operation.Error (Declaration .sdkCommand String String Empty) :=
  Operation.sdkCommand String String Empty identity

/-- The two semantic events this operation records. An event has no service response and no
transport failure signature, so it is declared as an event rather than as a command or an RPC. -/
def scheduledEventDeclaration : Except Operation.Error (Declaration .event String Unit Empty) :=
  Operation.event String scheduledEventDeclarationId
def completedEventDeclaration : Except Operation.Error (Declaration .event String Unit Empty) :=
  Operation.event String completedEventDeclarationId

/-! ### Model identities -/

def pendingStateId : DefinitionId := .of "temporal.nexus3.typed-nexus.state.pending"
def scheduledStateId : DefinitionId := .of "temporal.nexus3.typed-nexus.state.scheduled"
def completedStateId : DefinitionId := .of "temporal.nexus3.typed-nexus.state.completed"
def scheduleActionId : DefinitionId := .of "temporal.nexus3.typed-nexus.action.schedule"
def awaitActionId : DefinitionId := .of "temporal.nexus3.typed-nexus.action.await-completion"
def pollActionId : DefinitionId := .of "temporal.nexus3.typed-nexus.action.poll-history"
def scheduledOutcomeId : DefinitionId := .of "temporal.nexus3.typed-nexus.outcome.scheduled"
def completedOutcomeId : DefinitionId := .of "temporal.nexus3.typed-nexus.outcome.completed"
def noProgressOutcomeId : DefinitionId := .of "temporal.nexus3.typed-nexus.outcome.no-progress"
def operationRoleId : DefinitionId := .of "temporal.nexus3.typed-nexus.role.operation"
def targetId : DefinitionId := .of "temporal.nexus3.typed-nexus.target"
def kernelId : DefinitionId := .of "temporal.nexus3.typed-nexus.kernel"
def capabilityId : DefinitionId := .of "temporal.nexus3.typed-nexus.capability"
def providerId : DefinitionId := .of "temporal.nexus3.typed-nexus.provider"
def runFieldId : DefinitionId := .of "temporal.nexus3.typed-nexus.scope.run"
def operationFieldId : DefinitionId := .of "temporal.nexus3.typed-nexus.scope.operation"
def projectionId : DefinitionId := .of "temporal.nexus3.typed-nexus.projection"
def evidenceSourceId : DefinitionId := .of "temporal.nexus3.typed-nexus.source.history"
def operationIdentityFieldId : DefinitionId :=
  .of "temporal.nexus3.typed-nexus.evidence.operation-identity"
def completedEvidenceKindId : DefinitionId :=
  .of "temporal.nexus3.typed-nexus.evidence.completed"
def linkPropertyId : DefinitionId := .of "temporal.nexus3.typed-nexus.property.bounded-completion"
def linkClauseId : DefinitionId := .of "temporal.nexus3.typed-nexus.clause.bounded-completion"
def captureId : DefinitionId := .of "temporal.nexus3.typed-nexus.capture.scheduled-operation"
def fieldPropertyId : DefinitionId := .of "temporal.nexus3.typed-nexus.property.correlated-completion"
def groupId : DefinitionId := .of "temporal.nexus3.typed-nexus.property.group"
def caseId : DefinitionId := .of "temporal.nexus3.typed-nexus.property.case"
def clauseId : DefinitionId := .of "temporal.nexus3.typed-nexus.clause.correlated-completion"

def pendingState : ModelValue := .named pendingStateId "pending"
def completedState : ModelValue := .named completedStateId "completed"
def noProgressOutcome : ModelValue := .named noProgressOutcomeId "no-progress"
def awaitAction : ModelValue := .named awaitActionId "await-completion"
def pollAction : ModelValue := .named pollActionId "poll-history"

/-- One scheduled operation's Action payload: the same modeled Action definition carrying the exact
SDK command identity it submitted, so one trigger pattern covers both operations while their
declared command identities stay distinct. -/
def scheduleAction (command : Declaration .sdkCommand String String Empty) : ModelValue :=
  ⟨scheduleActionId, command.identity.value⟩

/-! ### Exact evidence payloads -/

/-- The scheduled evidence of one operation, read through the generated history response: the
event's own id and the endpoint/service/operation/request the scheduling command addressed. -/
def scheduledPayload (operation : String) (eventId : Int) (requestId : String) : Raw :=
  Value.message historyResponseRoot [
    (1, Value.message historyNode [
      (1, Value.repeated [
        Value.message historyEventNode [
          (1, Value.literal (.integer .int64 eventId)),
          (53, Value.message scheduledAttributesNode [
            (2, Value.literal (.text nexusService)),
            (3, Value.literal (.text operation)),
            (8, Value.literal (.text requestId))])]])])]

/-- The completed evidence of one operation: the scheduled event it references and its request. -/
def completedPayload (scheduledEventId : Int) (requestId : String) : Raw :=
  Value.message historyResponseRoot [
    (1, Value.message historyNode [
      (1, Value.repeated [
        Value.message historyEventNode [
          (1, Value.literal (.integer .int64 (scheduledEventId + 1))),
          (55, Value.message completedAttributesNode [
            (1, Value.literal (.integer .int64 scheduledEventId)),
            (3, Value.literal (.text requestId))])]])])]

/-! ### Checked cursors over the admitted payloads

Every operand below is built by a real cursor walk over a real admitted payload, so the declared
coordinates and the admitted ones are the same coordinates. -/

private def fieldError (reason : String) : Field.Error := ⟨source, "typed-nexus", reason⟩

private abbrev HistoryProjection :=
  PropertyFieldProjection Temporal.API.rpcOwner historyWitness

private abbrev HistoryCursor (type : Singular) :=
  Field.Cursor Temporal.API.rpcOwner historyWitness .response valueLimits type .singular .available

/-- Walk one admitted history payload to its single event, returning the optional `history`
presence the walk established and the event cursor itself. -/
private def eventCursor (payload : Raw) :
    Except Field.Error (HistoryCursor .boolean × HistoryCursor (.message historyEventNode)) := do
  let value ← (Value.check Temporal.API.rpcOwner historyWitness .response valueLimits payload).mapError
    fun error => Field.Error.mk source error.path error.reason
  let historyReference ← Field.reference Temporal.API.rpcOwner historyWitness .response
    historyResponseRoot 1 source
  let eventsReference ← Field.reference Temporal.API.rpcOwner historyWitness .response
    historyNode 1 source
  let root ← (Field.root value).refine (.message historyResponseRoot) .singular .available source
  let history ← root.field historyReference source
  let history ← history.refine (.message historyNode) .singular .optional source
  let presence ← history.present source
  let established ← history.establish source
  let events ← established.field eventsReference source
  let events ← events.refine (.message historyEventNode) .repeated .available source
  let event ← events.index 0 source
  pure (presence, event)

/-- Select one event's attributes oneof member, returning the member presence and the member. -/
private def attributeCursor (event : HistoryCursor (.message historyEventNode))
    (number : Nat) (node : String) :
    Except Field.Error (HistoryCursor .boolean × HistoryCursor (.message node)) := do
  let reference ← Field.reference Temporal.API.rpcOwner historyWitness .response
    historyEventNode number source
  let attributes ← event.field reference source
  let attributes ← attributes.refine (.message node) .singular (.oneof attributesGroup) source
  let presence ← attributes.present source
  let selected ← attributes.select attributesGroup source
  pure (presence, selected)

/-- Read one scalar leaf of an available message cursor. -/
private def scalarCursor (node : String) (parent : HistoryCursor (.message node))
    (number : Nat) (type : Singular) : Except Field.Error (HistoryCursor type) := do
  let reference ← Field.reference Temporal.API.rpcOwner historyWitness .response node number source
  let field ← parent.field reference source
  field.refine type .singular .available source

/-! ### Structural paths

Each path is the coordinates of one admitted cursor above; the tests require the two to agree. -/

/-- The coordinates of one scheduled-event attribute, from the whole history response payload. -/
def scheduledSteps (number : Nat) : List Field.Step :=
  [.field historyResponseRoot 1, .establish, .field historyNode 1, .index 0,
    .field historyEventNode 53, .select attributesGroup, .field scheduledAttributesNode number]

/-- The coordinates of one completed-event attribute, from the whole history response payload. -/
def completedSteps (number : Nat) : List Field.Step :=
  [.field historyResponseRoot 1, .establish, .field historyNode 1, .index 0,
    .field historyEventNode 55, .select attributesGroup, .field completedAttributesNode number]

private def eventIdSteps : List Field.Step :=
  [.field historyResponseRoot 1, .establish, .field historyNode 1, .index 0,
    .field historyEventNode 1]

private def historyPresenceSteps : List Field.Step := [.field historyResponseRoot 1, .present]

private def basePath (root : PropertyFieldRoot) (reference : DefinitionId) : PropertyFieldPath :=
  { root, reference, schema := historySchema, side := .response, steps := [], type := .text }

/-- The operation identity the scheduled evidence recorded. This is the value each operation
retains under its own key, and the Link correlation reads it back. -/
def scheduledOperationPath : PropertyFieldPath :=
  { basePath .outcome scheduledOutcomeId with steps := scheduledSteps 3 }

def scheduledHistoryPresencePath : PropertyFieldPath :=
  { basePath .outcome scheduledOutcomeId with
    steps := historyPresenceSteps, type := .boolean }

def scheduledAttributesPresencePath : PropertyFieldPath :=
  { basePath .outcome scheduledOutcomeId with
    steps := (scheduledSteps 3).take 5 ++ [.present], type := .boolean }

/-- The scheduled event's own id, carried by the model state a scheduled operation is in. -/
def scheduledEventIdPath : PropertyFieldPath :=
  { basePath .priorState scheduledStateId with
    steps := eventIdSteps, type := .integer .int64 }

def scheduledStatePresencePath : PropertyFieldPath :=
  { basePath .priorState scheduledStateId with
    steps := historyPresenceSteps, type := .boolean }

/-- The scheduled event a completion references. -/
def completedScheduledEventIdPath : PropertyFieldPath :=
  { basePath .outcome completedOutcomeId with
    steps := completedSteps 1, type := .integer .int64 }

def completedHistoryPresencePath : PropertyFieldPath :=
  { basePath .outcome completedOutcomeId with
    steps := historyPresenceSteps, type := .boolean }

def completedAttributesPresencePath : PropertyFieldPath :=
  { basePath .outcome completedOutcomeId with
    steps := (completedSteps 1).take 5 ++ [.present], type := .boolean }

/-! ### Admitted projections -/

/-- The scheduled evidence one operation records, as admitted projections: the two presence facts
the read traverses and the operation identity itself. -/
def scheduledProjections (operation : String) (eventId : Int) (requestId : String) :
    Except Field.Error (List HistoryProjection) := do
  let (historyPresence, event) ← eventCursor (scheduledPayload operation eventId requestId)
  let (attributesPresence, attributes) ← attributeCursor event 53 scheduledAttributesNode
  let name ← scalarCursor scheduledAttributesNode attributes 3 .text
  pure [
    ← PropertyFieldProjection.ofCursor .outcome scheduledOutcomeId historyPresence (by decide) source,
    ← PropertyFieldProjection.ofCursor .outcome scheduledOutcomeId attributesPresence (by decide)
      source,
    ← PropertyFieldProjection.ofCursor .outcome scheduledOutcomeId name (by decide) source]

/-- The scheduled event id an operation's model state carries, as admitted projections. -/
def scheduledStateProjections (operation : String) (eventId : Int) (requestId : String) :
    Except Field.Error (List HistoryProjection) := do
  let (historyPresence, event) ← eventCursor (scheduledPayload operation eventId requestId)
  let identity ← scalarCursor historyEventNode event 1 (.integer .int64)
  pure [
    ← PropertyFieldProjection.ofCursor .priorState scheduledStateId historyPresence (by decide) source,
    ← PropertyFieldProjection.ofCursor .priorState scheduledStateId identity (by decide) source]

/-- The completion evidence, as admitted projections: the two presence facts and the scheduled
event the completion references. -/
def completedProjections (scheduledEventId : Int) (requestId : String) :
    Except Field.Error (List HistoryProjection) := do
  let (historyPresence, event) ← eventCursor (completedPayload scheduledEventId requestId)
  let (attributesPresence, attributes) ← attributeCursor event 55 completedAttributesNode
  let referenced ← scalarCursor completedAttributesNode attributes 1 (.integer .int64)
  pure [
    ← PropertyFieldProjection.ofCursor .outcome completedOutcomeId historyPresence (by decide) source,
    ← PropertyFieldProjection.ofCursor .outcome completedOutcomeId attributesPresence (by decide)
      source,
    ← PropertyFieldProjection.ofCursor .outcome completedOutcomeId referenced (by decide) source]

private def lastModelValue (projections : Except Field.Error (List HistoryProjection)) :
    Except Field.Error ModelValue := do
  let some last := (← projections).getLast? | throw (fieldError "missing projection")
  pure last.modelValue

/-! ### The two operations this Case runs -/

/-- One modeled operation: its declared SDK command, the operation name it addresses, and the exact
scheduled event id and request id its own evidence records. -/
structure OperationCase where
  command : DefinitionId
  operation : String
  eventId : Int
  requestId : String
  deriving BEq, DecidableEq, Repr

def firstCase : OperationCase := ⟨firstCommandId, firstOperation, 5, "umpire-typed-nexus-a"⟩
def secondCase : OperationCase := ⟨secondCommandId, secondOperation, 9, "umpire-typed-nexus-b"⟩

def operationCases : List OperationCase := [firstCase, secondCase]

def OperationCase.scheduledOutcome (entry : OperationCase) : Except Field.Error ModelValue :=
  lastModelValue (scheduledProjections entry.operation entry.eventId entry.requestId)

def OperationCase.scheduledState (entry : OperationCase) : Except Field.Error ModelValue :=
  lastModelValue (scheduledStateProjections entry.operation entry.eventId entry.requestId)

def OperationCase.completedOutcome (entry : OperationCase) : Except Field.Error ModelValue :=
  lastModelValue (completedProjections entry.eventId entry.requestId)

/-! ### The Target -/

private def structuralKinds : List (DefinitionId × DefinitionKind) := [
  (targetId, .target), (kernelId, .machine), (providerId, .provider), (capabilityId, .capability)]

/-- The modeled vocabulary a Property clause may name. -/
private def vocabularyKinds : List (DefinitionId × DefinitionKind) := [
  (pendingStateId, .state), (scheduledStateId, .state), (completedStateId, .state),
  (scheduleActionId, .action), (awaitActionId, .action), (pollActionId, .action),
  (scheduledOutcomeId, .outcome), (completedOutcomeId, .outcome), (noProgressOutcomeId, .outcome)]

private def definitions : List DefinitionMetadata :=
  (structuralKinds ++ vocabularyKinds).map fun (id, kind) =>
    Temporal.Shared.definitionMetadata id kind source id.value

private def provider : Provider (fun _ => True) := {
  id := providerId
  source
  contract := { id := capabilityId, behaviorVersion := "temporal-nexus3-typed-nexus/v1"
                requiredLaws := [] }
  meanings := vocabularyKinds.map fun (id, kind) =>
    { definitionId := id, kind, behaviorVersion := id.value ++ "/meaning-v1" }
  lawProofs := []
}

private def modelSpec : TableModelSpec := {
  id := targetId
  source
  definitions
  requiredCapabilities := [capabilityId]
  metadata := { id := kernelId, source }
}

/-- Every way this authored example can fail admission, named by its owner. -/
inductive AdmissionError where
  | operation (error : Operation.Error)
  | field (error : Field.Error)
  | target (error : TableAdmissionError)
  | property (error : PropertyError)
  | scoped (error : Property.Scoped.Error)
  | projection (error : Observation.Projection.Error)
  | coverage (error : Observation.Projection.CoverageError)
  | inconsistent (reason : String)

private abbrev TypedTarget :=
  CheckedModel (fun _ => True) Unit ModelValue ModelValue ModelValue ModelValue

/-! ### The independent field requirement -/

private def presenceHolds (path : PropertyFieldPath) : PropertyPredicate :=
  PropertyPredicate.compareFields .equal (.field path source) (.literal (.boolean true) source)
    source

private def selects (reference : DefinitionId) : PropertyPredicate :=
  .atom { field := .selectedAction, reference }

/-- A completion references the scheduled event its own operation was scheduled at. The three
presence atoms establish exactly the optional and oneof steps the two reads traverse. -/
def completionReferencesSchedule : PropertyPredicate := .all [
  presenceHolds scheduledStatePresencePath,
  presenceHolds completedHistoryPresencePath,
  presenceHolds completedAttributesPresencePath,
  PropertyPredicate.compareFields .equal (.field scheduledEventIdPath source)
    (.field completedScheduledEventIdPath source) source]

/-- The authored same-step Property. Its one clause applies exactly to the await step. -/
def fieldDeclaration : Property := {
  id := fieldPropertyId
  source
  version := 2
  requires := [capabilityId]
  clauses := [.branches {
    id := groupId, source, guard := selects awaitActionId
    cases := [{
      id := caseId, source, guard := selects awaitActionId
      clauses := [⟨clauseId, source, completionReferencesSchedule⟩] }] }]
}

/-! ### The bounded-response Property and its Link correlation -/

/-- The typed earlier command field each operation retains under its own key: the operation
identity its scheduled evidence recorded. -/
def scheduledOperationCapture : PropertyScopedCapture :=
  { name := captureId, key := operationFieldId, path := scheduledOperationPath, lifetime := 2 }

private def capturedOperationIs (operation : String) : PropertyPredicate :=
  PropertyPredicate.compareFields .equal
    (.field { scheduledOperationPath with capture := some ⟨captureId, 0⟩ } source)
    (.literal (.text operation) source) source

/-- The Link. A step belongs to this operation only when the identity the operation retained at its
own scheduling is one of the two identities the model declares. The scheduling disjunct decides the
step that creates occurrence zero, which therefore never has to read it.

The admitted set is the two declared identities, not this operation's own: a correlation operand
cannot name the scope key, so an operation keyed `complete` whose scheduled evidence recorded
`confirm` still satisfies the Link and is caught by the authored field requirement instead. The
runtime rules are per-identity and reject it outright; `Tests/TypedNexus.lean` pins both answers. -/
def declaredOperationIdentity : PropertyPredicate :=
  .any (selects scheduleActionId :: operationCases.map fun entry => capturedOperationIs entry.operation)

/-- The bounded response: a scheduled operation completes within its declared window of semantic
transitions. Closing an unfinished prefix leaves it unresolved rather than inventing a deadline. -/
def boundedCompletion : PropertyScopedClause := {
  id := linkClauseId
  source
  trigger := selects scheduleActionId
  response := .atom { field := .outcome, reference := completedOutcomeId }
  scope := [runFieldId]
  key := operationFieldId
  bound := 2
  endpoint := .«partial»
  captures := [scheduledOperationCapture]
  correlation := some declaredOperationIdentity
}

def linkDeclaration : Property := {
  id := linkPropertyId
  source
  requires := [capabilityId]
  clauses := []
  scopedClauses := [boundedCompletion]
}

/-! ### The declared evidence projection

The scoped capability reads `ScopedEvidence` the Program lifts out of the very history the monitor
rules read, so the bounded response is answered from recorded evidence rather than in the model
alone. One rule per recorded shape: each scheduled event selects itself by the operation identity it
records, and a completion selects itself by its own attributes member.

The operation key is the scheduled event's own id. A completion records the scheduled event it
answers and nothing else that names its operation, so keying on that id is the only way both sides
of one operation land under the same key -- the same indistinguishability the crossed-completion
Known Gap names, read here as the key rather than as a rule. -/

/-- The evidence kind one operation's scheduled event records. -/
def OperationCase.scheduledEvidenceKindId (entry : OperationCase) : DefinitionId :=
  .of ("temporal.nexus3.typed-nexus.evidence.scheduled-" ++ entry.operation)

/-- The operation identity a scheduled event retains, the one field the Link correlation reads. -/
def retainedOperationIdentity : List (EvidenceFieldDeclaration × FieldDisposition) :=
  [(⟨operationIdentityFieldId, .text⟩, .retain)]

def projectionLimits : Observation.Projection.Limits := {
  events := 64, buffered := 32, keys := 8, support := 256
  work := 1000000000, eventSize := 512 }

/-- The declared projection. A completion names one representative completed step: the Target owns
both completions from either scheduled state, and the bounded-response clause reads the completed
outcome by its declared identity, so which of the two the projector releases never changes its
answer. Separating them would need the operation identity a completed event does not record. -/
def projectionDeclaration : Except AdmissionError
    (Observation.Projection.Declaration ModelValue ModelValue ModelValue ModelValue) := do
  let scheduled ← operationCases.mapM fun entry => do
    let command ← (scheduleCommand entry.command).mapError AdmissionError.operation
    let state ← entry.scheduledState.mapError AdmissionError.field
    let outcome ← entry.scheduledOutcome.mapError AdmissionError.field
    pure ({ kind := entry.scheduledEvidenceKindId
            fields := retainedOperationIdentity
            meaning := .confirmed none [(scheduleAction command,
              { state := state, outcome := outcome, facts := [] })] } :
      Observation.Projection.Rule ModelValue ModelValue ModelValue ModelValue)
  let completedOutcome ← firstCase.completedOutcome.mapError AdmissionError.field
  pure {
    id := projectionId
    scopeFields := [runFieldId]
    operationField := operationFieldId
    sources := [evidenceSourceId]
    rules := scheduled ++ [{
      kind := completedEvidenceKindId
      meaning := .confirmed none [(awaitAction,
        { state := completedState, outcome := completedOutcome
          facts := [] })] }]
    limits := projectionLimits }

/-- The one covered coordinate: the operation identity the scheduled evidence records is the value
the Link capture retains, and `operationIdentityFieldId` is the declared field that supplies it. -/
def coverageMappings : List Observation.Projection.FieldMapping :=
  [⟨scheduledOperationPath, operationIdentityFieldId⟩]

/-! ### Admission -/

/-- The complete checked model: the Target that owns both completions, the same-step field
requirement, the compiled bounded-response consumer carrying the Link, and the checked evidence
projection the scoped capability runs on. -/
structure Model where
  target : TypedTarget
  fieldProperty : CheckedFieldProperty
  link : CheckedProperty
  compiled : Property.Scoped.Compiled target
  plan : Observation.Projection.Checked target
  coverage : Observation.Projection.Coverage plan

private def fieldBindings : List PropertyFieldBinding :=
  [scheduledOutcomeId, completedOutcomeId, scheduledStateId].map
    (PropertyFieldBinding.ofWitness Temporal.API.rpcOwner historyWitness)

/-- Evaluation ceilings for the scoped consumer, separate from the semantic bound above. -/
def runLimits : Property.Scoped.Limits :=
  { transitions := 32, obligations := 16, work := 100000000, captures := 16 }

/-- Admit the whole authored example: the SDK command and event declarations, the Target whose
completions are its own alternatives, the authored field requirement, and the Link. -/
def checked : Except AdmissionError Model := do
  let _ ← scheduledEventDeclaration.mapError AdmissionError.operation
  let _ ← completedEventDeclaration.mapError AdmissionError.operation
  let _ ← historyBinding.mapError AdmissionError.operation
  let commands ← operationCases.mapM fun entry =>
    (scheduleCommand entry.command).mapError AdmissionError.operation
  let scheduledStates ← operationCases.mapM fun entry =>
    entry.scheduledState.mapError AdmissionError.field
  let scheduledOutcomes ← operationCases.mapM fun entry =>
    entry.scheduledOutcome.mapError AdmissionError.field
  let completedOutcomes ← operationCases.mapM fun entry =>
    entry.completedOutcome.mapError AdmissionError.field
  unless scheduledStates.length == 2 && completedOutcomes.length == 2 do
    throw (.inconsistent "this example declares exactly two operations")
  let scheduleRows := (((commands.zip scheduledStates).zip scheduledOutcomes)).zipIdx.map
    fun (((command, state), outcome), index) =>
      ({ key := "schedule-" ++ toString index, source := pendingState
         action := scheduleAction command
         results := [{ state := state, outcome := outcome, facts := [] }] } :
        FiniteTransitionRow ModelValue ModelValue ModelValue ModelValue)
  let pollRows := scheduledStates.zipIdx.map fun (state, index) =>
    ({ key := "poll-" ++ toString index, source := state, action := pollAction
       results := [{ state := state, outcome := noProgressOutcome
                     facts := [] }] } :
      FiniteTransitionRow ModelValue ModelValue ModelValue ModelValue)
  -- The Target owns both completions: selecting the await Action never selects which scheduled
  -- event the completion that arrives references, so the crossed one is a clause violation.
  let awaitRows := scheduledStates.zipIdx.map fun (state, index) =>
    ({ key := "await-" ++ toString index, source := state, action := awaitAction
       results := (completedOutcomes.drop index ++ completedOutcomes.take index).map fun outcome =>
         { state := completedState, outcome := outcome, facts := [] } } :
      FiniteTransitionRow ModelValue ModelValue ModelValue ModelValue)
  let table : FiniteTable Unit ModelValue ModelValue ModelValue ModelValue := {
    setups := [⟨(), "operation"⟩]
    states := ⟨pendingState, pendingState.value⟩ ::
      (scheduledStates.zipIdx.map fun (state, index) => ⟨state, "scheduled-" ++ toString index⟩) ++
      [⟨completedState, completedState.value⟩]
    actions := (commands.zipIdx.map fun (command, index) =>
        ⟨scheduleAction command, "schedule-" ++ toString index⟩) ++
      [⟨awaitAction, awaitAction.value⟩, ⟨pollAction, pollAction.value⟩]
    outcomes := (scheduledOutcomes.zipIdx.map fun (outcome, index) =>
        ⟨outcome, "scheduled-" ++ toString index⟩) ++
      (completedOutcomes.zipIdx.map fun (outcome, index) =>
        ⟨outcome, "completed-" ++ toString index⟩) ++
      [⟨noProgressOutcome, noProgressOutcome.value⟩]
    facts := []
    initial := [⟨(), [pendingState]⟩]
    transitions := scheduleRows ++ pollRows ++ awaitRows
    terminalConditions := [[completedState]]
  }
  let target ← (table.checkTypedModel modelSpec
    (Providers.empty.provide provider)).mapError AdmissionError.target
  let context := { PropertyCheckContext.ofTarget target with fieldBindings }
  let fieldProperty ← (CheckedFieldProperty.check context fieldDeclaration).mapError
    AdmissionError.property
  let link ← (Property.check context (linkDeclaration)).mapError AdmissionError.property
  let compiled ← (Property.Scoped.compile target link [runFieldId] operationFieldId
    runLimits).mapError AdmissionError.scoped
  let declaration ← projectionDeclaration
  let plan ← (Observation.Projection.check target declaration () pendingState).mapError
    AdmissionError.projection
  let coverage ← (Observation.Projection.Coverage.check plan valueLimits coverageMappings).mapError
    AdmissionError.coverage
  pure ⟨target, fieldProperty, link, compiled, plan, coverage⟩


/-! ### The Testpilot Case

The workflow schedules both Nexus operations on one endpoint and awaits both. Each operation has
its own asynchronous handler, so the controller holds one completion authority per operation and
completes them independently; the history the controller then reads is the evidence both the Link
and the authored requirement are established from.
-/

/-- The second generated reference the Program needs: the workflow the two operations run in is
started through `StartWorkflowExecution`, whose transport path is derived from its own admitted
full name rather than copied beside it. -/
abbrev startMethod := Temporal.Api.Workflowservice.V1.WorkflowService.startWorkflowExecution

def startReference : Temporal.API.MethodReference startMethod := by constructor

def startWitness : Temporal.API.rpcOwner.Witness
    Temporal.Api.Workflowservice.V1.StartWorkflowExecutionRequest
    Temporal.Api.Workflowservice.V1.StartWorkflowExecutionResponse := ⟨startMethod, startReference⟩

def startBinding : Except Operation.Error (CheckedRpc Temporal.API.rpcOwner startWitness) :=
  Temporal.API.bindUnary startMethod startReference

def workflowServiceRole := "temporal.workflow-service"
def workerRole := "temporal.worker"
def taskQueueRole := "temporal.task-queue"
def nexusEndpointRole := "temporal.nexus-endpoint"
def namespaceBindingId := "temporal.typed-nexus.namespace"
def taskQueueBindingId := "temporal.typed-nexus.task-queue"
def nexusEndpointBindingId := "temporal.typed-nexus.nexus-endpoint"
def controllerId := "controller"
def workflowEntrypointId := "workflow"
def observationId := "history-event"
def scopedObservationId := "scoped-evidence"

/-- The protobuf oneof members of `HistoryEvent.attributes` this Case lifts. -/
def scheduledAttributesField := "nexus_operation_scheduled_event_attributes"
def completedAttributesField := "nexus_operation_completed_event_attributes"
/-- The one Run coordinate the recorded history does not carry: every event this Case lifts belongs
to the single Run the Case executes, so the Case declares that scope rather than reading it. -/
def runScopeValue := "typed-nexus"
def workflowType := "umpire-typed-nexus-workflow"

/-- Per-operation Program identities, so the two operations never share an instruction, a Slot or
a handler entrypoint. -/
def OperationCase.slotId (entry : OperationCase) : String :=
  "completion-authority-" ++ entry.operation
def OperationCase.handlerId (entry : OperationCase) : String := "handler-" ++ entry.operation
def OperationCase.startInstructionId (entry : OperationCase) : String :=
  "start-nexus-" ++ entry.operation
def OperationCase.awaitInstructionId (entry : OperationCase) : String :=
  "await-nexus-" ++ entry.operation
def OperationCase.authorityInstructionId (entry : OperationCase) : String :=
  "await-authority-" ++ entry.operation
def OperationCase.completeInstructionId (entry : OperationCase) : String :=
  "complete-nexus-" ++ entry.operation

/-- Two operations make this Case larger than the shared single-operation Program ceilings in two
places: the history that records both is bigger than one response budget, and two Nexus handler
entrypoints take longer to stop than one. Every other bound is the shared ceiling. -/
def typedNexusProgramLimits : ProgramLimits :=
  { programLimits with max_response_bytes := 8192, max_cleanup_duration_milliseconds := 20000 }

/-- The history read carries both operations' events, so it declares that larger response budget. -/
private def historyLimits : InstructionLimits := Program.instructionLimits 10000 1 128 8192

private def textOutcome : InstructionOutcomeDefinition :=
  Program.outcome #[
    Program.outcomeField .INSTRUCTION_OUTCOME_FIELD_STATUS statusType,
    Program.outcomeField .INSTRUCTION_OUTCOME_FIELD_VALUE textType]

private def controllerInstructions (entry : OperationCase) : Array InstructionDefinition := #[
  Program.node entry.authorityInstructionId (Program.awaitSlot entry.slotId)
    (bounds 10000) #[Ref.instruction controllerId "start-workflow"]
    (some (succeeded controllerId "start-workflow")) (some statusOutcome),
  Program.node entry.completeInstructionId
    (Program.completeNexusOperation entry.slotId (text "completed"))
    (bounds 10000) #[Ref.instruction controllerId entry.authorityInstructionId]
    (some (succeeded controllerId entry.authorityInstructionId)) (some statusOutcome)]

private def workflowInstructions (entry : OperationCase) : Array InstructionDefinition := #[
  Program.node entry.startInstructionId
    (Program.startNexusOperation nexusEndpointRole nexusService entry.operation (text "request"))
    (bounds 10000) #[] none (some statusOutcome),
  Program.node entry.awaitInstructionId
    (Program.awaitOutcome (Ref.instruction workflowEntrypointId entry.startInstructionId))
    (bounds 10000) #[Ref.instruction workflowEntrypointId entry.startInstructionId]
    none (some textOutcome)]

/-- The Program-declared source of the scoped capability's evidence. Each rule reads only the
history event it guards: a scheduled event by the operation identity it records, a completion by its
own attributes member. The operation key is the scheduled event's own id on one side and the
scheduled event a completion references on the other, so both land under the same key. -/
private def evidenceTarget : ProjectionTarget :=
  Program.scopedEvidenceTarget scopedObservationId
    ((operationCases.map fun entry =>
        Program.scopedEvidenceRule
          (guard := historyAttribute scheduledAttributesField "operation")
          (source := evidenceSourceId.value)
          (kind := entry.scheduledEvidenceKindId.value)
          (operation := field "event_id")
          (scope := #[Program.scopedEvidenceLiteral runFieldId.value runScopeValue])
          (fields := #[Program.scopedEvidenceBinding operationIdentityFieldId.value
            (historyAttribute scheduledAttributesField "operation")])
          (guardEqualsText := entry.operation)) ++
      [Program.scopedEvidenceRule
        (guard := Path.make #[Path.oneofSelector attributesGroup completedAttributesField])
        (source := evidenceSourceId.value)
        (kind := completedEvidenceKindId.value)
        (operation := historyAttribute completedAttributesField "scheduled_event_id")
        (scope := #[Program.scopedEvidenceLiteral runFieldId.value runScopeValue])]).toArray

private def program (startPath historyPath : String) : Program :=
  Program.make "temporal.case.typed-nexus.program"
    #[Program.role workflowServiceRole .ROLE_KIND_ENDPOINT,
      Program.role workerRole .ROLE_KIND_WORKER (namespaceBindingId := namespaceBindingId),
      Program.role taskQueueRole .ROLE_KIND_TASK_QUEUE
        (namespaceBindingId := namespaceBindingId) (resourceBindingId := taskQueueBindingId),
      Program.role nexusEndpointRole .ROLE_KIND_ENDPOINT
        (resourceBindingId := nexusEndpointBindingId)]
    (operationCases.map (fun entry => Program.capabilitySlot entry.slotId)).toArray
    #[Program.observation observationId (Types.singular (Types.messageType historyEventNode)),
      Program.observation scopedObservationId (Types.singular
        (Types.messageType "temporal.server.api.testpilot.v1.ScopedEvidence"))]
    (#[Program.controller controllerId (
        #[Program.node "start-workflow"
            (Program.invokeRPC workflowServiceRole startPath #[
              Program.environmentAssignment (field "namespace") namespaceBindingId,
              assign (field "workflow_id") runId,
              assign (nested ["workflow_type", "name"]) (text workflowType),
              Program.environmentAssignment (nested ["task_queue", "name"]) taskQueueBindingId,
              assign (field "request_id") runId])
            (bounds 10000) #[] none (some statusOutcome)
            ((Program.reservation workflowEntrypointId 1) ::
              operationCases.map fun entry => Program.reservation entry.handlerId 1).toArray] ++
          (operationCases.flatMap fun entry => (controllerInstructions entry).toList).toArray ++
          #[Program.node "history"
            (Program.invokeRPC workflowServiceRole historyPath #[
              Program.environmentAssignment (field "namespace") namespaceBindingId,
              assign (nested ["execution", "workflow_id"]) runId,
              assign (field "maximum_page_size") (signedInteger 64),
              assign (field "wait_new_event") (boolean true)]
              #[Program.responseProjection historyEvents .PROJECTION_KIND_EMIT_EACH
                  #[Program.observationTarget observationId, evidenceTarget]])
            historyLimits
            (operationCases.map fun entry =>
              Ref.instruction controllerId entry.completeInstructionId).toArray
            (some (ProgramExpr.all (operationCases.map fun entry =>
              succeeded controllerId entry.completeInstructionId).toArray))
            (some statusOutcome)]),
      Program.workflow workflowEntrypointId workflowType workerRole taskQueueRole (
        (operationCases.flatMap fun entry => (workflowInstructions entry).toList).toArray ++
          #[Program.node "finish-workflow" (Program.finish (text "completed"))
            bounds (operationCases.map fun entry =>
              Ref.instruction workflowEntrypointId entry.awaitInstructionId).toArray
            (some (ProgramExpr.all (operationCases.map fun entry =>
              succeeded workflowEntrypointId entry.awaitInstructionId).toArray))
            (some statusOutcome)])] ++
      (operationCases.map fun entry =>
        Program.nexusHandler entry.handlerId nexusService entry.operation workerRole taskQueueRole
          #[Program.node ("respond-async-" ++ entry.operation)
            (Program.respondNexus .NEXUS_RESPONSE_KIND_ASYNCHRONOUS (text "accepted") entry.slotId)
            bounds #[] none (some statusOutcome)]).toArray)
    (Program.cleanup "cleanup" #[])
    typedNexusProgramLimits
    (environment := #[Program.environment namespaceBindingId,
      Program.environment taskQueueBindingId, Program.environment nexusEndpointBindingId])

/-! ### The derived Contract

Every field the runtime reads is `Umpire.Case.Observed.pathOf` applied to the same
`PropertyFieldPath` the model Property compares, so a Property edit that moves a field moves the
Contract's read with it and a field the Property stops naming stops being read. -/

/-- The runtime read path of one modeled operand, from the declared Observation's own message. -/
def readPathOf (path : PropertyFieldPath) : Except String FieldPath :=
  Umpire.Case.Observed.pathOf path historyEventNode

/-- The Contract rule for one operation. Its first transition is the Link: an event belongs to this
operation only when the operation identity it records is this operation's declared one, and the
event it retains is the scheduled event the model captures. Its second transition is the authored
product requirement: the completion references exactly that scheduled event. -/
def operationRule (entry : OperationCase) (property : CheckedProperty) :
    Except String ContractRuleDefinition := do
  let operationPath ← readPathOf scheduledOperationPath
  let eventIdPath ← readPathOf scheduledEventIdPath
  let referencedPath ← readPathOf completedScheduledEventIdPath
  let capture := "scheduled-" ++ entry.operation
  pure (Monitor.rule (property.id.value ++ "." ++ entry.operation)
    .CONTRACT_RULE_KIND_SAFETY "pending"
    #[Monitor.state "pending" .CONTRACT_STATE_STATUS_NONTERMINAL,
      Monitor.state "scheduled" .CONTRACT_STATE_STATUS_NONTERMINAL,
      Monitor.state "satisfied" .CONTRACT_STATE_STATUS_SATISFIED]
    #[Monitor.transition ("capture-scheduled-" ++ entry.operation) "pending" "scheduled"
        #[.RUN_EVENT_KIND_INSTRUCTION_COMPLETED]
        (ContractExpr.all #[
          ContractExpr.present (observed observationId),
          ContractExpr.present (projected (observed observationId) operationPath),
          ContractExpr.equals (projected (observed observationId) operationPath)
            (ContractExpr.literal (Value.text entry.operation))])
        .CONTRACT_SUPPORT_KIND_MATCHING_EVENT
        #[Monitor.captureAssignment capture observationId],
      Monitor.transition ("match-completion-" ++ entry.operation) "scheduled" "satisfied"
        #[.RUN_EVENT_KIND_INSTRUCTION_COMPLETED]
        (ContractExpr.all #[
          ContractExpr.present (captured capture),
          ContractExpr.present (projected (observed observationId) referencedPath),
          ContractExpr.equals (projected (captured capture) eventIdPath)
            (projected (observed observationId) referencedPath)])
        .CONTRACT_SUPPORT_KIND_MATCHING_EVENT]
    (captures := #[Monitor.capture capture (Monitor.messageCapture historyEventNode)]))

/-- This Case retains one history event per operation, so it declares its own capture-byte ceiling
rather than the shared single-capture one; every other bound is the shared Contract ceiling. -/
def typedNexusContractLimits : ContractLimits :=
  { contractLimits with
    max_capture_bytes := 65536, max_captures := 64, max_transitions := 64
    max_work_per_event := 4000000 }

/-- The bounded-response window now runs online: the history read lifts each recorded Nexus event
into the declared `ScopedEvidence` Observation the scoped capability decodes, so the clause is
answered from recorded evidence rather than in the model alone.

What the lift cannot supply is the operation identity of a completion, which no completed event
records. The key is therefore the scheduled event a completion references, and the projection
releases one representative completed step. The bounded-response clause reads the completed outcome
by its declared identity, so the released payload never changes its answer -- but a Case that wanted
to tell the two completions apart online still could not, which is exactly what the crossed
completion gap below already names. -/
private def completionIdentityIsUnrecorded (link : CheckedProperty) : Umpire.Case.CaseKnownGap :=
  { kind := .interpretation
    code := "temporal.nexus3.typed-nexus.completion-identity-is-unrecorded"
    subject := some link.id.value
    detail := some ("the bounded-response window runs online from lifted history evidence; a " ++
      "completed event records no operation identity, so the operation key is the scheduled " ++
      "event it references and the projection releases one representative completed step") }

/-- The model Property separates a crossed completion from missing evidence, and the rules here do
not. A completed history event records the scheduled event it references but not the operation
identity that was scheduled, so a reference to another scheduled event is indistinguishable from
the sibling operation's own completion. Separating them at runtime would need a correlation
condition the model never declared, which ACT-4 makes an Implementation Link obligation rather than
a rule this Case may add on its own; until one is declared, both readings close the rule
inconclusive. -/
private def crossedCompletionIsInconclusive (requirement : CheckedProperty) :
    Umpire.Case.CaseKnownGap :=
  { kind := .interpretation
    code := "temporal.nexus3.typed-nexus.crossed-completion-is-inconclusive"
    subject := some requirement.id.value
    detail := some ("a completed event carries no operation identity, so a completion referencing " ++
      "another scheduled event leaves the rule pending rather than violated; the model Property " ++
      "still distinguishes the two") }

private def loweringError (definitionId construct : String) : Umpire.Case.Compiler.LoweringError :=
  { sourceDefinitionId := definitionId, source, construct }

/-- The checked two-operation declaration lowered to the closed Case format. -/
def typedNexusCase : Except Umpire.Case.Compiler.LoweringError
    temporal.server.api.testpilot.v1.Case := do
  let model ← checked.mapError fun _ => loweringError fieldPropertyId.value "checked-typed-nexus"
  let history ← historyBinding.mapError fun _ =>
    loweringError historyMethod.fullName "checked-history-binding"
  let start ← startBinding.mapError fun _ =>
    loweringError startMethod.fullName "checked-start-binding"
  let requirement := model.fieldProperty.property
  let requirementBinding := binding requirement.id.value
    requirement.behaviorFingerprint.render .«property»
  let rules ← operationCases.mapM fun entry =>
    (operationRule entry requirement).mapError fun reason => loweringError clauseId.value reason
  let lowered ← Umpire.Case.Scoped.lower model.plan model.compiled scopedObservationId
    model.coverage
  Umpire.Case.Compiler.compile {
    version := { major := 1 }
    caseId := "temporal.case.typed-nexus"
    producerId := "temporal.nexus3.typed-nexus"
    producerVersion := "1"
    definitions := [
      binding model.target.id.value model.target.behaviorFingerprint.render .target,
      requirementBinding,
      binding model.link.id.value model.link.behaviorFingerprint.render .«property»]
    sources := [source]
    knownGaps := [completionIdentityIsUnrecorded model.link,
      crossedCompletionIsInconclusive requirement]
    program := program (methodPath start.schema) (methodPath history.schema)
    contractId := "temporal.case.typed-nexus.contract"
    properties := rules.map (.monitor requirementBinding) ++ [lowered.contractLowering]
    contractLimits := typedNexusContractLimits
  }

end Temporal.Feature.Nexus3.TypedNexus
