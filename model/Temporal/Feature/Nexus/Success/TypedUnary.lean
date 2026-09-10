import Temporal.API
import Temporal.Shared
import Temporal.Testpilot.CaseSupport
import Umpire.Case.Compiler
import Umpire.Case.Observed
import Umpire.Property.Elab
import Umpire.Property.Evaluate
import Umpire.Property.Correlated
import Umpire.Operation.Parameterized

/-!
# A generated unary operation qualified by independent field requirements

`StartWorkflowExecution` is referenced through its generated declaration, never through a copied
method name or a hand-written schema: `Temporal.API.bindUnary` checks the candidate against the
generator's own selection, and the checked binding is what the parameterized Action template
carries. The finite domain lists exactly two admitted request values, which differ only in the
nested `workflow_type.name` they submit.

The independent requirement relates two different generated operations. The submitted
`workflow_type.name` of the selected Action must equal the `workflow_type.name` the server later
records in the `WorkflowExecutionStarted` history event, read through the generated
`GetWorkflowExecutionHistory` response schema. That is an actual product requirement rather than a
fictitious Start response field: a Start acknowledgement carries a run id, not a workflow type, and
the started event is written when the execution begins.

The Target owns the pairing. Each admitted Action instance is paired with the started evidence for
its own workflow type, so crossing the pairing is a Property violation rather than a rejection, and
the mutation tests in `Tests/TypedUnary.lean` exercise exactly that distinction.
-/

namespace Temporal.Feature.Nexus.Success.TypedUnary

open Umpire
open Umpire.Operation
open Umpire.Value
open Temporal.Testpilot.CaseSupport
open Testpilot.Authoring
open temporal.server.api.testpilot.v1

/-! ### Generated references -/

/-- The generated unary method this example qualifies. -/
abbrev startMethod := Temporal.Api.Workflowservice.V1.WorkflowService.startWorkflowExecution

/-- The second generated method, whose response carries the correlated started evidence. -/
abbrev historyMethod := Temporal.Api.Workflowservice.V1.WorkflowService.getWorkflowExecutionHistory

def startReference : Temporal.API.MethodReference startMethod := by constructor
def historyReference : Temporal.API.MethodReference historyMethod := by constructor

def startWitness : Temporal.API.rpcOwner.Witness
    Temporal.Api.Workflowservice.V1.StartWorkflowExecutionRequest
    Temporal.Api.Workflowservice.V1.StartWorkflowExecutionResponse := ⟨startMethod, startReference⟩

def historyWitness : Temporal.API.rpcOwner.Witness
    Temporal.Api.Workflowservice.V1.GetWorkflowExecutionHistoryRequest
    Temporal.Api.Workflowservice.V1.GetWorkflowExecutionHistoryResponse :=
  ⟨historyMethod, historyReference⟩

def source : SourceLocation :=
  Temporal.Shared.sourceLocation "Temporal/Feature/Nexus/Success/TypedUnary.lean"

/-- Semantic value bounds for the admitted request and response payloads. -/
def valueLimits : Limits := ⟨16, 20000000, 262144, 512⟩

/-- The declared runtime admission scope. It is a separate claim from the finite sample: runtime
values outside the sample are admitted, but only inside these semantic bounds, which are tighter
than the resource ceilings the checker itself runs under. -/
def runtimeBounds : RuntimeBounds := ⟨12, 8192, 64⟩

/-! ### Structural coordinates, taken from the generated descriptors -/

def startRequestRoot := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest"
def historyResponseRoot := "temporal.api.workflowservice.v1.GetWorkflowExecutionHistoryResponse"
def workflowTypeNode := "temporal.api.common.v1.WorkflowType"
def taskQueueNode := "temporal.api.taskqueue.v1.TaskQueue"
def historyNode := "temporal.api.history.v1.History"
def historyEventNode := "temporal.api.history.v1.HistoryEvent"
def startedAttributesNode := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes"
def payloadsNode := "temporal.api.common.v1.Payloads"
def payloadNode := "temporal.api.common.v1.Payload"

/-- The workflow type the Program submits, and the one the crossed sample submits instead. -/
def submittedWorkflowType := "umpire-typed-unary-workflow"
def alternateWorkflowType := "umpire-typed-unary-alternate"

/-! ### Model identities -/

def startActionId : DefinitionId := .of "temporal.nexus.success.typed-unary.action.start-workflow"
def pendingStateId : DefinitionId := .of "temporal.nexus.success.typed-unary.state.pending"
def startedStateId : DefinitionId := .of "temporal.nexus.success.typed-unary.state.started"
def startedOutcomeId : DefinitionId := .of "temporal.nexus.success.typed-unary.outcome.started-evidence"
def startedFactId : DefinitionId := .of "temporal.nexus.success.typed-unary.fact.started-recorded"
def operationRoleId : DefinitionId := .of "temporal.nexus.success.typed-unary.role.operation"
def targetId : DefinitionId := .of "temporal.nexus.success.typed-unary.target"
def kernelId : DefinitionId := .of "temporal.nexus.success.typed-unary.kernel"
def capabilityId : DefinitionId := .of "temporal.nexus.success.typed-unary.capability"
def providerId : DefinitionId := .of "temporal.nexus.success.typed-unary.provider"
def propertyId : DefinitionId := .of "temporal.nexus.success.typed-unary.property.submitted-workflow-type"
def groupId : DefinitionId := .of "temporal.nexus.success.typed-unary.property.group"
def caseId : DefinitionId := .of "temporal.nexus.success.typed-unary.property.case"
def clauseId : DefinitionId := .of "temporal.nexus.success.typed-unary.property.clause"

def pendingState : ModelValue := .named pendingStateId "pending"
def startedState : ModelValue := .named startedStateId "started"
def startedFact : ModelValue := .named startedFactId "started-recorded"

/-! ### Checked generated bindings and the parameterized Action template -/

/-- Admit the generated Start declaration against the generator's own structural selection. -/
def startBinding : Except Operation.Error
    (CheckedRpc Temporal.API.rpcOwner startWitness) :=
  Temporal.API.bindUnary startMethod startReference

/-- Admit the generated history declaration; its response supplies the correlated evidence. -/
def historyBinding : Except Operation.Error
    (CheckedRpc Temporal.API.rpcOwner historyWitness) :=
  Temporal.API.bindUnary historyMethod historyReference

/-- The authored Action template over the checked generated Start declaration. -/
def startTemplate : Except Operation.Error (ActionTemplate Temporal.API.rpcOwner
    Temporal.Api.Workflowservice.V1.StartWorkflowExecutionRequest
    Temporal.Api.Workflowservice.V1.StartWorkflowExecutionResponse Empty) := do
  ActionTemplate.check (rpc Empty (← startBinding)) startActionId

/-! ### Exact request and evidence payloads -/

/-- One admitted Start request. Only the nested `workflow_type.name` varies across the domain. -/
def requestValue (workflowType : String) : Raw :=
  Value.message startRequestRoot [
    (2, Value.literal (.text "umpire-typed-unary")),
    (3, Value.message workflowTypeNode [(1, Value.literal (.text workflowType))]),
    (4, Value.message taskQueueNode [(1, Value.literal (.text "umpire-typed-unary-queue"))])]

/-- The correlated started evidence: the workflow type the server recorded for the execution. -/
def startedEvidenceValue (workflowType : String) : Raw :=
  Value.message historyResponseRoot [
    (1, Value.message historyNode [
      (1, Value.repeated [
        Value.message historyEventNode [
          (1, Value.literal (.integer .int64 1)),
          (6, Value.message startedAttributesNode [
            (1, Value.message workflowTypeNode [(1, Value.literal (.text workflowType))])])]])])]

/-! ### Structural paths -/

/-- The submitted nested request field: `workflow_type.name` of the selected Action. -/
def submittedTypePath : PropertyFieldPath := {
  root := .request, reference := startActionId
  schema := Temporal.API.rpcOwner.schema startWitness, side := .request
  steps := [.field startRequestRoot 3, .establish, .field workflowTypeNode 1], type := .text }

/-- Presence of the optional `workflow_type` submessage, established before the read above. -/
def submittedTypePresencePath : PropertyFieldPath :=
  { submittedTypePath with steps := [.field startRequestRoot 3, .present], type := .boolean }

private def startedSteps : List Field.Step :=
  [.field historyResponseRoot 1, .establish, .field historyNode 1, .index 0,
    .field historyEventNode 6, .select "attributes", .field startedAttributesNode 1, .establish,
    .field workflowTypeNode 1]

/-- The recorded workflow type of the started event, read through the generated history response. -/
def startedTypePath : PropertyFieldPath := {
  root := .outcome, reference := startedOutcomeId
  schema := Temporal.API.rpcOwner.schema historyWitness, side := .response
  steps := startedSteps, type := .text }

/-- The three presence facts the started read traverses: an optional `history`, the selected
`attributes` oneof member, and an optional `workflow_type`. -/
def historyPresencePath : PropertyFieldPath :=
  { startedTypePath with steps := [.field historyResponseRoot 1, .present], type := .boolean }

def attributesPresencePath : PropertyFieldPath :=
  { startedTypePath with
    steps := (startedSteps.take 5) ++ [.present], type := .boolean }

def startedTypePresencePath : PropertyFieldPath :=
  { startedTypePath with
    steps := (startedSteps.take 7) ++ [.present], type := .boolean }

/-! ### Checked projections

Each operand of the requirement resolves to exactly one admitted projection, so every path above
is built by a real cursor walk over a real admitted payload rather than declared alongside it. -/

private def fieldError (reason : String) : Field.Error := ⟨source, "typed-unary", reason⟩

private abbrev StartProjection :=
  PropertyFieldProjection Temporal.API.rpcOwner startWitness
private abbrev HistoryProjection :=
  PropertyFieldProjection Temporal.API.rpcOwner historyWitness

/-- The submitted `workflow_type` cursor and its presence read, over the selected Action's own
immutable arguments. -/
def submittedProjections
    {template : ActionTemplate Temporal.API.rpcOwner
      Temporal.Api.Workflowservice.V1.StartWorkflowExecutionRequest
      Temporal.Api.Workflowservice.V1.StartWorkflowExecutionResponse Empty}
    (action : ActionInstance template valueLimits) :
    Except Field.Error (List (PropertyFieldProjection Temporal.API.rpcOwner
      template.declaration.reference)) := do
  let typeReference ← Field.reference Temporal.API.rpcOwner template.declaration.reference .request
    startRequestRoot 3 source
  let nameReference ← Field.reference Temporal.API.rpcOwner template.declaration.reference .request
    workflowTypeNode 1 source
  let root ← (Field.root action.arguments).refine (.message startRequestRoot) .singular .available
    source
  let workflowType ← root.field typeReference source
  let workflowType ← workflowType.refine (.message workflowTypeNode) .singular .optional source
  let presence ← workflowType.present source
  let established ← workflowType.establish source
  let name ← established.field nameReference source
  let name ← name.refine .text .singular .available source
  if presenceOrigin : presence.origin.value = action.arguments.value then
    if nameOrigin : name.origin.value = action.arguments.value then
      pure [← PropertyFieldProjection.ofAction action presence presenceOrigin source,
        ← PropertyFieldProjection.ofAction action name nameOrigin source]
    else throw (fieldError "submitted read left the selected arguments")
  else throw (fieldError "submitted presence read left the selected arguments")

/-- The recorded `workflow_type` cursor and the three presence facts its read traverses. -/
def startedProjections (workflowType : String) :
    Except Field.Error (List HistoryProjection) := do
  let value ← (Value.check Temporal.API.rpcOwner historyWitness .response valueLimits
    (startedEvidenceValue workflowType)).mapError fun error =>
      Field.Error.mk source error.path error.reason
  let historyReference ← Field.reference Temporal.API.rpcOwner historyWitness .response
    historyResponseRoot 1 source
  let eventsReference ← Field.reference Temporal.API.rpcOwner historyWitness .response
    historyNode 1 source
  let attributesReference ← Field.reference Temporal.API.rpcOwner historyWitness .response
    historyEventNode 6 source
  let typeReference ← Field.reference Temporal.API.rpcOwner historyWitness .response
    startedAttributesNode 1 source
  let nameReference ← Field.reference Temporal.API.rpcOwner historyWitness .response
    workflowTypeNode 1 source
  let root ← (Field.root value).refine (.message historyResponseRoot) .singular .available source
  let history ← root.field historyReference source
  let history ← history.refine (.message historyNode) .singular .optional source
  let historyPresence ← history.present source
  let history ← history.establish source
  let events ← history.field eventsReference source
  let events ← events.refine (.message historyEventNode) .repeated .available source
  let event ← events.index 0 source
  let attributes ← event.field attributesReference source
  let attributes ← attributes.refine (.message startedAttributesNode) .singular
    (.oneof "attributes") source
  let attributesPresence ← attributes.present source
  let attributes ← attributes.select "attributes" source
  let recordedType ← attributes.field typeReference source
  let recordedType ← recordedType.refine (.message workflowTypeNode) .singular .optional source
  let typePresence ← recordedType.present source
  let recordedType ← recordedType.establish source
  let name ← recordedType.field nameReference source
  let name ← name.refine .text .singular .available source
  pure [
    ← PropertyFieldProjection.ofCursor .outcome startedOutcomeId historyPresence (by decide) source,
    ← PropertyFieldProjection.ofCursor .outcome startedOutcomeId attributesPresence (by decide)
      source,
    ← PropertyFieldProjection.ofCursor .outcome startedOutcomeId typePresence (by decide) source,
    ← PropertyFieldProjection.ofCursor .outcome startedOutcomeId name (by decide) source]

/-- The correlated model outcome of one admitted Action: the started evidence for its own type. -/
def startedOutcome (workflowType : String) : Except Field.Error ModelValue := do
  let projections ← startedProjections workflowType
  let some recorded := projections.getLast? | throw (fieldError "missing started projection")
  pure recorded.modelValue

/-! ### The checked finite parameterized domain and its authoritative Target -/

/-- The two exact workflow types the finite domain admits, in declaration order. -/
def sampledWorkflowTypes : List String := [submittedWorkflowType, alternateWorkflowType]

/-- The Target, its kernel, and the capability the Property requires. -/
private def structuralKinds : List (DefinitionId × DefinitionKind) := [
  (targetId, .target), (kernelId, .machine), (providerId, .provider), (capabilityId, .capability)]

/-- The modeled vocabulary a Property clause may name. The Action's own definition is contributed
by the parameterized domain, which attaches the domain's canonical meaning to it, so it belongs to
the provider's meanings but never to this list. -/
private def vocabularyKinds : List (DefinitionId × DefinitionKind) := [
  (pendingStateId, .state), (startedStateId, .state),
  (startedOutcomeId, .outcome), (startedFactId, .fact)]

private def definitions : List DefinitionMetadata :=
  (structuralKinds ++ vocabularyKinds).map fun (id, kind) =>
    Temporal.Shared.definitionMetadata id kind source id.value

private def provider : Provider (fun _ => True) := {
  id := providerId
  source
  contract := { id := capabilityId, behaviorVersion := "temporal-nexus-success-typed-unary/v1"
                requiredLaws := [] }
  meanings := ((startActionId, DefinitionKind.action) :: vocabularyKinds).map
    fun (id, kind) => { definitionId := id, kind, behaviorVersion := id.value ++ "/meaning-v1" }
  lawProofs := []
}

private def modelSpec : TableModelSpec := {
  id := targetId
  source
  definitions
  requiredCapabilities := [capabilityId]
  metadata := { id := kernelId, source }
}

/-- The complete checked model: the parameterized Action domain, the authoritative Target whose
rows pair each admitted request with its own correlated started evidence, and the checked
independent field Property. -/
structure Model where
  template : ActionTemplate Temporal.API.rpcOwner
    Temporal.Api.Workflowservice.V1.StartWorkflowExecutionRequest
    Temporal.Api.Workflowservice.V1.StartWorkflowExecutionResponse Empty
  domain : ParameterDomain template valueLimits
  target : CheckedModel (fun _ => True) (List RoleBinding) ModelValue ModelValue ModelValue
    ModelValue
  property : CheckedFieldProperty

/-- Every way this authored example can fail admission, named by its owner. -/
inductive AdmissionError where
  | operation (error : Operation.Error)
  | parameter (error : ParameterError)
  | field (error : Field.Error)
  | target (error : TableAdmissionError)
  | property (error : PropertyError)
  | inconsistent (reason : String)

/-! ### The independent field requirement -/

private def presenceHolds (path : PropertyFieldPath) : PropertyPredicate :=
  PropertyPredicate.compareFields .equal (.field path source) (.literal (.boolean true) source)
    source

/-- The submitted nested request field must equal the workflow type the started event recorded.
The four presence atoms establish exactly the optional and oneof steps the two reads traverse. -/
def submittedTypeMatchesStarted : PropertyPredicate := .all [
  presenceHolds submittedTypePresencePath,
  presenceHolds historyPresencePath,
  presenceHolds attributesPresencePath,
  presenceHolds startedTypePresencePath,
  PropertyPredicate.compareFields .equal (.field submittedTypePath source)
    (.field startedTypePath source) source]

private def alwaysApplies : PropertyPredicate :=
  PropertyPredicate.compareFields .equal (.literal (.boolean true) source)
    (.literal (.boolean true) source) source

/-- The authored Property declaration; its one clause is the independent field requirement. -/
def declaration : Property := {
  id := propertyId
  source
  version := 2
  requires := [capabilityId]
  clauses := [.branches {
    id := groupId, source, guard := alwaysApplies
    cases := [{
      id := caseId, source, guard := alwaysApplies
      clauses := [⟨clauseId, source, submittedTypeMatchesStarted⟩] }] }]
}

/-! ### Admission -/

private def fieldBindings (template : ActionTemplate Temporal.API.rpcOwner
    Temporal.Api.Workflowservice.V1.StartWorkflowExecutionRequest
    Temporal.Api.Workflowservice.V1.StartWorkflowExecutionResponse Empty) :
    List PropertyFieldBinding :=
  [PropertyFieldBinding.ofAction template,
    PropertyFieldBinding.ofWitness Temporal.API.rpcOwner historyWitness startedOutcomeId]

/-- Admit the whole authored example: the generated binding, the finite domain, the Target that
owns the request/evidence pairing, and the independent field Property. -/
def checked : Except AdmissionError Model := do
  let template ← startTemplate.mapError AdmissionError.operation
  let domain ← (ParameterDomain.check template valueLimits (sampledWorkflowTypes.map requestValue)
    .sampled (.schema runtimeBounds)).mapError AdmissionError.parameter
  let outcomes ← sampledWorkflowTypes.mapM fun workflowType =>
    (startedOutcome workflowType).mapError AdmissionError.field
  unless domain.actions.length == outcomes.length do
    throw (.inconsistent "domain and evidence lengths differ")
  let rows := (domain.actions.zip outcomes).zipIdx.map fun ((action, outcome), index) =>
    ({ key := "start-" ++ toString index, source := pendingState, action
       results := [{ state := startedState, outcome := outcome
                     facts := [startedFact] }] } :
      FiniteTransitionRow ModelValue (ActionInstance template valueLimits) ModelValue ModelValue)
  let table : FiniteTable Unit ModelValue (ActionInstance template valueLimits) ModelValue
      ModelValue := {
    setups := [⟨(), "operation"⟩]
    states := [⟨pendingState, pendingState.value⟩, ⟨startedState, startedState.value⟩]
    actions := domain.catalog
    outcomes := outcomes.zipIdx.map fun (outcome, index) =>
      ⟨outcome, "started-evidence-" ++ toString index⟩
    facts := [⟨startedFact, startedFact.value⟩]
    initial := [⟨(), [pendingState]⟩]
    transitions := rows
    terminalConditions := [[startedState]]
  }
  let identity : FiniteModelIdentity Unit ModelValue (ActionInstance template valueLimits)
      ModelValue ModelValue := {
    setupBindings := fun _ => [⟨operationRoleId, pendingState⟩]
    stateId := (·.definitionId)
    actionId := fun _ => template.identity
    outcomeId := (·.definitionId)
    factId := (·.definitionId)
  }
  let target ← (domain.checkModel table identity modelSpec
    (Providers.empty.provide provider)).mapError AdmissionError.target
  let context := { PropertyCheckContext.ofTarget target with
    fieldBindings := fieldBindings template }
  let property ← (CheckedFieldProperty.check context declaration).mapError AdmissionError.property
  pure ⟨template, domain, target, property⟩

/-! ### The Testpilot Case

The controller submits the request and then reads the execution's history, emitting each event as a
declared Observation. The workflow entrypoint carries the submitted workflow type, because the
shared Driver admits a `StartWorkflowExecution` only when the instruction reserves a workflow
entrypoint and binds its namespace and task queue symbolically — the same generic policy the
existing Nexus.Success Case runs under. The workflow itself does nothing but finish; the evidence the
requirement reads is the started event, which the server writes when the execution begins.
-/

def workflowServiceRole := "temporal.workflow-service"
def workerRole := "temporal.worker"
def taskQueueRole := "temporal.task-queue"
def namespaceBindingId := "temporal.typed-unary.namespace"
def taskQueueBindingId := "temporal.typed-unary.task-queue"
def controllerId := "controller"
def workflowEntrypointId := "workflow"
def startInstructionId := "start-workflow"
def historyInstructionId := "history"
def observationId := "history-event"

/-- The concrete `workflow_type.name` coordinates the Program constructs. The coordinates the
monitor rule reads back out of the started history event are derived from the Property itself,
below. -/
private def submittedTypeTarget : FieldPath := nested ["workflow_type", "name"]

private def program (startPath historyPath : String) : Program :=
  Program.make "temporal.case.typed-unary.program"
    #[Program.role workflowServiceRole .ROLE_KIND_ENDPOINT,
      Program.role workerRole .ROLE_KIND_WORKER (namespaceBindingId := namespaceBindingId),
      Program.role taskQueueRole .ROLE_KIND_TASK_QUEUE
        (namespaceBindingId := namespaceBindingId) (resourceBindingId := taskQueueBindingId)]
    #[]
    #[Program.observation observationId historyEventType]
    #[Program.controller controllerId #[
      Program.node startInstructionId
        (Program.invokeRPC workflowServiceRole startPath #[
          Program.environmentAssignment (field "namespace") namespaceBindingId,
          assign (field "workflow_id") runId,
          assign submittedTypeTarget (text submittedWorkflowType),
          Program.environmentAssignment (nested ["task_queue", "name"]) taskQueueBindingId,
          assign (field "request_id") runId])
        (bounds 10000) #[] none (some statusOutcome)
        #[Program.reservation workflowEntrypointId 1],
      Program.node historyInstructionId
        (Program.invokeRPC workflowServiceRole historyPath #[
          Program.environmentAssignment (field "namespace") namespaceBindingId,
          assign (nested ["execution", "workflow_id"]) runId,
          assign (field "maximum_page_size") (signedInteger 64)]
          #[project historyEvents observationId .PROJECTION_KIND_EMIT_EACH])
        (bounds 10000 128) #[Ref.instruction controllerId startInstructionId]
        (some (succeeded controllerId startInstructionId)) (some statusOutcome)],
      Program.workflow workflowEntrypointId submittedWorkflowType workerRole taskQueueRole #[
        Program.node "finish-workflow" (Program.finish (text "started")) bounds #[] none
          (some statusOutcome)]]
    (Program.cleanup "cleanup" #[])
    programLimits
    (environment := #[Program.environment namespaceBindingId,
      Program.environment taskQueueBindingId])

/-! ### The derived Contract

The field the runtime reads is `Umpire.Case.Observed.pathOf` applied to the same
`PropertyFieldPath` the model Property compares, so a Property edit that moves the recorded
coordinate moves the Contract's read with it. The rule structure around that path -- its states,
its presence checks and the submitted-type literal it matches -- is authored here. -/

/-- The runtime read path of one modeled operand, from the declared Observation's own message. -/
def readPathOf (path : PropertyFieldPath) : Except String FieldPath :=
  Umpire.Case.Observed.pathOf path historyEventNode

/-- The runtime reading of the one checked clause: the workflow type the started event recorded is
the one the Program submitted. The rule distinguishes the same three answers the model Property
does. An event that establishes the recorded type and disagrees with it is a violation, not an
absence; an event that never establishes the field leaves the rule pending, so a Run that produced
no started event still closes inconclusive. -/
private def startedRule (checkedProperty : CheckedProperty) :
    Except String ContractRuleDefinition := do
  let recordedTypePath ← readPathOf startedTypePath
  pure (Contract.rule (checkedProperty.id.value ++ ".recorded-workflow-type")
    .CONTRACT_RULE_KIND_SAFETY "pending"
    #[Contract.state "pending" .CONTRACT_STATE_STATUS_NONTERMINAL,
      Contract.state "satisfied" .CONTRACT_STATE_STATUS_SATISFIED,
      Contract.state "violated" .CONTRACT_STATE_STATUS_VIOLATED]
    #[Contract.transition "match-recorded-workflow-type" "pending" "satisfied"
      #[.RUN_EVENT_KIND_INSTRUCTION_COMPLETED]
      (ContractExpr.all #[
        ContractExpr.present (observed observationId),
        ContractExpr.present (projected (observed observationId) recordedTypePath),
        ContractExpr.equals (projected (observed observationId) recordedTypePath)
          (ContractExpr.literal (Value.text submittedWorkflowType))])
      .CONTRACT_SUPPORT_KIND_MATCHING_EVENT,
      Contract.transition "reject-recorded-workflow-type" "pending" "violated"
      #[.RUN_EVENT_KIND_INSTRUCTION_COMPLETED]
      (ContractExpr.all #[
        ContractExpr.present (observed observationId),
        ContractExpr.present (projected (observed observationId) recordedTypePath),
        ContractExpr.negation (ContractExpr.equals
          (projected (observed observationId) recordedTypePath)
          (ContractExpr.literal (Value.text submittedWorkflowType)))])
      .CONTRACT_SUPPORT_KIND_MATCHING_EVENT])

/-- The modeled input field this Case must construct, and the exact instruction that constructs it. -/
def coverage : Umpire.Case.Coverage.Request := {
  inputs := [{ path := submittedTypePath, value := .text submittedWorkflowType
               entrypointId := controllerId, instructionId := startInstructionId }] }

private def compilerError (definitionId construct : String) : Umpire.Case.Compiler.Error :=
  { sourceDefinitionId := definitionId, source, construct }

/-- The checked typed unary declaration lowered to the closed Case format. -/
def typedUnaryCase : Except Umpire.Case.Compiler.Error
    temporal.server.api.testpilot.v1.Case := do
  let model ← checked.mapError fun _ =>
    compilerError propertyId.value "checked-typed-unary"
  let history ← historyBinding.mapError fun _ =>
    compilerError historyMethod.fullName "checked-history-binding"
  let checkedProperty := model.property.property
  let propertyBinding := binding checkedProperty.id.value
    checkedProperty.behaviorFingerprint.render .«property»
  let rule ← (startedRule checkedProperty).mapError fun reason =>
    compilerError clauseId.value reason
  Umpire.Case.Compiler.compile {
    version := { major := 1 }
    caseId := "temporal.case.typed-unary"
    producerId := "temporal.nexus.success.typed-unary"
    producerVersion := "1"
    definitions := [
      binding model.target.id.value model.target.behaviorFingerprint.render .target,
      propertyBinding]
    sources := [source]
    knownGaps := []
    program := program (methodPath model.template.declaration.schema) (methodPath history.schema)
    contractId := "temporal.case.typed-unary.contract"
    properties := [.monitor propertyBinding rule]
    contractLimits
    coverage
  }

end Temporal.Feature.Nexus.Success.TypedUnary
