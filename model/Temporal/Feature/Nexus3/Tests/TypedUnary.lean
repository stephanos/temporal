import Temporal.Feature.Nexus3.TypedUnary

/-!
Executable checks for the generated unary example.

Four things are inspected here, none of them through the live call: the generated binding admits
only its own method and schema; the finite domain and the separately declared runtime scope report
exactly what they explored; the independent field requirement separates a correlated pairing from a
crossed one; and the whole-Case coverage rejects a Program that does not construct the modeled
input field, before any Driver I/O could happen.
-/

namespace Temporal.Feature.Nexus3.Tests.TypedUnary

open Umpire
open Umpire.Operation
open Umpire.Value
open Temporal.Feature.Nexus3.TypedUnary
open temporal.server.api.testpilot.v1

/-! ### The generated reference is admitted, and only against its own selection -/

#guard startBinding.isOk
#guard historyBinding.isOk
#guard startTemplate.isOk

private def startSchema : RpcSchema := Temporal.API.rpcOwner.schema startWitness
private def historySchema : RpcSchema := Temporal.API.rpcOwner.schema historyWitness

-- The admitted schema is the generator's own selection for this method, not a copy beside it.
#guard startSchema.fullName ==
  "temporal.api.workflowservice.v1.WorkflowService.StartWorkflowExecution"
#guard !startSchema.clientStreaming && !startSchema.serverStreaming

-- The Program's transport path is derived from that admitted full name.
#guard methodPath startSchema ==
  "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
#guard methodPath historySchema ==
  "/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory"

private def rejects (candidate : RpcSchema) (expected : Operation.Error) : Bool :=
  match Temporal.API.bindUnary startMethod startReference candidate with
  | .error actual => actual == expected
  | .ok _ => false

-- Another generated method's schema is a wrong-method binding even though both are unary.
#guard rejects historySchema (.wrongMethod startSchema.fullName historySchema.fullName)

-- A forged closure keeps the method name but changes what the artifact was built against.
#guard rejects { startSchema with
    request := { startSchema.request with root := historySchema.request.root } }
  (.incompatibleRequest startSchema.fullName)
#guard rejects { startSchema with response := historySchema.response }
  (.incompatibleResponse startSchema.fullName)
#guard rejects { startSchema with serverStreaming := true }
  (.incompatibleStreaming startSchema.fullName)

/-! ### The finite sample, the fixed dimension and the runtime scope are separate and inspectable -/

#guard checked.isOk
#guard (checked.toOption.map fun model => model.domain.actions.length) == some 2
#guard (checked.toOption.map fun model => model.domain.coverage) == some .sampled
#guard (checked.toOption.map fun model => model.domain.runtimeScope) ==
  some (.schema runtimeBounds)

-- The recorded meaning names the explored dimension, its coverage claim, and every sample.
#guard (checked.toOption.map fun model =>
  (model.domain.canonical.splitOn "\"dimension\":\"whole-request\"").length) == some 2
#guard (checked.toOption.map fun model =>
  (model.domain.canonical.splitOn "\"coverage\":\"sampled\"").length) == some 2

/-- A request value the finite domain never listed. -/
private def outsideSample : Raw := requestValue "umpire-typed-unary-unexplored"

-- Finite membership is exactly the two authored samples.
#guard (do
  let model ← checked.toOption
  let value ← (Value.check Temporal.API.rpcOwner startWitness .request valueLimits
    outsideSample).toOption
  pure (model.domain.actions.any fun action => action.arguments.value == value.value)) ==
  some false

-- The separately declared runtime scope admits a value the finite sample never explored; that
-- admission is a runtime claim and does not enlarge the finite one above.
#guard (do
  let model ← checked.toOption
  pure (model.domain.admitRuntime valueLimits outsideSample).isOk) == some true

/-- A request the checker's own resource ceilings admit, but the declared semantic scope does not:
its payload bytes exceed the declared runtime bound. -/
private def oversizedSample : Raw :=
  Value.message startRequestRoot [
    (2, Value.literal (.text "umpire-typed-unary")),
    (5, Value.message payloadsNode [(1, Value.repeated [
      Value.message payloadNode [(2, Value.literal (.bytes (List.replicate 9000 0)))]])])]

-- A value outside the declared semantic bounds is reported out of scope, which is a different
-- answer from an exhausted resource ceiling.
#guard (do
  let model ← checked.toOption
  pure (match model.domain.admitRuntime valueLimits oversizedSample with
    | .error error => error == ParameterError.outOfScope
    | .ok _ => false)) == some true

/-- Re-admit the domain under a different coverage claim or sample list. -/
private def domainWith (coverage : ParameterCoverage) (values : List String) :
    Option ParameterError := do
  let template ← startTemplate.toOption
  match ParameterDomain.check template valueLimits (values.map requestValue) coverage
      (.schema runtimeBounds) with
  | .error error => some error
  | .ok _ => none

-- A fixed dimension claims one exact value, and an abstraction has no preservation evidence.
#guard domainWith .fixed sampledWorkflowTypes == some .invalidFixedDomain
#guard domainWith .fixed [submittedWorkflowType] == none
#guard domainWith (.abstracted "any-workflow-type") sampledWorkflowTypes ==
  some (.unsupportedAbstraction "any-workflow-type")
#guard domainWith .sampled [submittedWorkflowType, submittedWorkflowType] ==
  some .duplicateArgument

/-! ### The independent field requirement, over the actual correlated evidence -/

/-- Evaluate the checked Property over one modeled step: the Action at `actionIndex` paired with
the started evidence recorded for `evidenceIndex`. -/
private def evaluation (actionIndex evidenceIndex : Nat) (withEvidence : Bool := true) :
    Option Bool := do
  let model ← checked.toOption
  let action ← model.domain.actions[actionIndex]?
  let workflowType ← sampledWorkflowTypes[evidenceIndex]?
  let submitted ← (submittedProjections action).toOption
  let started ← (startedProjections workflowType).toOption
  let outcome ← started.getLast?
  let trace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
    initialState := pendingState
    steps := [{ selectedAction := action.modelValue, modelOutcome := outcome.modelValue
                resultingState := startedState, observations := [startedFact] }] }
  let evidence := submitted.map (·.evidence) ++
    (if withEvidence then started.map (·.evidence) else [])
  let input ← (model.property.checkInput trace [evidence]).toOption
  pure (evaluateProperty model.property.property input).satisfied

-- Each admitted request is paired with the workflow type its own execution recorded.
#guard evaluation 0 0 == some true
#guard evaluation 1 1 == some true

-- Crossing the pairing violates the declared clause: a product violation, not a rejection.
#guard evaluation 0 1 == some false
#guard evaluation 1 0 == some false

-- Missing evidence never satisfies the comparison; the input is rejected instead.
#guard evaluation 0 0 (withEvidence := false) == none

-- Every operand resolves to a projection built by a real cursor walk over a real admitted payload,
-- so the declared coordinates and the admitted ones are the same coordinates.
#guard ((startedProjections submittedWorkflowType).toOption.map fun projections =>
  projections.map (·.evidence.path)) ==
  some [historyPresencePath, attributesPresencePath, startedTypePresencePath, startedTypePath]

#guard (do
  let model ← checked.toOption
  let action ← model.domain.actions[0]?
  let submitted ← (submittedProjections action).toOption
  pure (submitted.map (·.evidence.path))) == some [submittedTypePresencePath, submittedTypePath]

-- The request operand is the selected Action's own immutable arguments, so its projection carries
-- exactly that Action's model payload identity.
#guard (do
  let model ← checked.toOption
  let action ← model.domain.actions[0]?
  let submitted ← (submittedProjections action).toOption
  let name ← submitted.getLast?
  pure (name.modelValue == action.modelValue)) == some true

/-! ### Concrete byte and optional fidelity, independent of the live call -/

/-- One admitted request carrying a concrete `Payload`: exact `data` bytes, one keyed `metadata`
entry, and an optionally absent `workflow_type`. -/
private def payloadRequest (metadataKey : String) (metadata data : List UInt8)
    (workflowType : Option String := some submittedWorkflowType) : Raw :=
  Value.message startRequestRoot
    ((workflowType.toList.map fun name =>
        (3, Value.message workflowTypeNode [(1, Value.literal (.text name))])) ++
      [(5, Value.message payloadsNode [(1, Value.repeated [
        Value.message payloadNode [
          (1, Value.map [(.text metadataKey, Value.literal (.bytes metadata))]),
          (2, Value.literal (.bytes data))]])])])

/-- Read one concrete `Payload` scalar through the checked cursors, with no Property evaluation:
`data` (field 2) directly, or the `metadata` entry (field 1) stored under `key`. -/
private def payloadScalar (raw : Raw) (number : Nat) (key : String := "encoding") :
    Option Scalar := do
  let value ← (Value.check Temporal.API.rpcOwner startWitness .request valueLimits raw).toOption
  let payloadsReference ← (Field.reference Temporal.API.rpcOwner startWitness .request
    startRequestRoot 5 source).toOption
  let listReference ← (Field.reference Temporal.API.rpcOwner startWitness .request
    payloadsNode 1 source).toOption
  let selected ← (Field.reference Temporal.API.rpcOwner startWitness .request
    payloadNode number source).toOption
  let root ← ((Field.root value).refine (.message startRequestRoot) .singular .available
    source).toOption
  let input ← (root.field payloadsReference source).toOption
  let input ← (input.refine (.message payloadsNode) .singular .optional source).toOption
  let input ← (input.establish source).toOption
  let list ← (input.field listReference source).toOption
  let list ← (list.refine (.message payloadNode) .repeated .available source).toOption
  let payload ← (list.index 0 source).toOption
  let field ← (payload.field selected source).toOption
  if number == 2 then
    let data ← (field.refine .bytes .singular .available source).toOption
    (data.scalar source).toOption
  else
    let metadata ← (field.refine .bytes (.map .text) .available source).toOption
    let entry ← (metadata.lookup (.text key) source).toOption
    let entry ← (entry.establish source).toOption
    (entry.scalar source).toOption

-- Concrete bytes stay concrete: two byte strings of equal length are distinguished, which a digest
-- or size summary could not do.
#guard payloadScalar (payloadRequest "encoding" [1, 2] [0, 255]) 2 == some (.bytes [0, 255])
#guard payloadScalar (payloadRequest "encoding" [1, 2] [255, 0]) 2 == some (.bytes [255, 0])
#guard PropertyFieldOperator.matches .equal (.bytes [0, 255]) (.bytes [255, 0]) == false
#guard PropertyFieldOperator.matches .equal (.bytes [0, 255]) (.bytes [0, 255]) == true

-- A keyed map lookup reaches the exact bytes stored under its own typed key.
#guard payloadScalar (payloadRequest "encoding" [106, 115, 111, 110] [0]) 1 ==
  some (.bytes [106, 115, 111, 110])

-- An absent key is absent, not an empty default: the lookup has no presence to establish.
#guard payloadScalar (payloadRequest "encoding" [1] [0]) 1 "absent" == none

-- An explicitly supplied empty byte string is still a present value.
#guard payloadScalar (payloadRequest "encoding" [] [] (workflowType := none)) 2 == some (.bytes [])

/-- Read the descriptor presence of one top-level request field. -/
private def presence (raw : Raw) (number : Nat) : Option Bool := do
  let value ← (Value.check Temporal.API.rpcOwner startWitness .request valueLimits raw).toOption
  let reference ← (Field.reference Temporal.API.rpcOwner startWitness .request
    startRequestRoot number source).toOption
  let root ← ((Field.root value).refine (.message startRequestRoot) .singular .available
    source).toOption
  let field ← (root.field reference source).toOption
  let present ← (field.present source).toOption
  match (present.scalar source).toOption with
  | some (.boolean flag) => some flag
  | _ => none

#guard presence (payloadRequest "encoding" [1] [0]) 3 == some true
#guard presence (payloadRequest "encoding" [1] [0] (workflowType := none)) 3 == some false

-- Presence follows the descriptor: an implicit-presence scalar has none to read.
#guard presence (payloadRequest "encoding" [1] [0]) 2 == none

/-! ### The Case, and the coverage admitted before any Driver I/O -/

#guard match typedUnaryCase with
  | .ok output =>
      output.case_id == "temporal.case.typed-unary" &&
      (match output.contract.map (fun contract => contract.rules.toList) with
        | some [rule] => rule.kind == .CONTRACT_RULE_KIND_SAFETY && rule.horizon.isNone
        | _ => false)
  | .error _ => false

private def producedProgram : Option Program :=
  typedUnaryCase.toOption.bind fun output => output.program

/-- Admit one requested coverage map against the Program this Case actually produced. -/
private def coverageResult (request : Umpire.Case.Coverage.Request) : Option String := do
  let program ← producedProgram
  match Umpire.Case.Coverage.check program [] request "temporal.case.typed-unary" with
  | .error failure => some failure.reason
  | .ok () => none

private def inputMapping : Umpire.Case.Coverage.InputMapping :=
  { path := submittedTypePath, value := .text submittedWorkflowType
    entrypointId := controllerId, instructionId := startInstructionId }

-- The Program constructs the covered nested input field with exactly the modeled value.
#guard coverageResult coverage == none

-- Mutating the modeled value the clause reads breaks the coverage, not the Program.
#guard coverageResult { coverage with
    inputs := [{ inputMapping with value := .text alternateWorkflowType }] } ==
  some "request assignment constructs a different value than the modeled field"

-- A field the Program never constructs from an exact value is a missing mapping.
#guard coverageResult { coverage with
    inputs := [{ inputMapping with
      path := { submittedTypePath with
        steps := [.field startRequestRoot 4, .establish, .field taskQueueNode 1] } }] } ==
  some "covered input field is not constructed from an exact value"

#guard coverageResult { coverage with
    inputs := [{ inputMapping with instructionId := "absent" }] } ==
  some "unknown instruction absent"

-- A presence read describes the payload rather than constructing it.
#guard coverageResult { coverage with
    inputs := [{ inputMapping with path := submittedTypePresencePath, value := .boolean true }] } ==
  some "a presence read is not a request assignment target"

-- Result coordinates are covered by declared Observations, never by a request assignment.
#guard coverageResult { coverage with inputs := [{ inputMapping with path := startedTypePath }] } ==
  some "input coverage names request coordinates; results and events are covered by Observations"

/-! ### Trust -/

/-- info: 'Umpire.Operation.CheckedRpc.schema_eq' does not depend on any axioms -/
#guard_msgs in
#print axioms Umpire.Operation.CheckedRpc.schema_eq

/-- info: 'Umpire.Operation.ParameterDomain.decode_encode' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Operation.ParameterDomain.decode_encode

/-- info: 'Umpire.Value.Field.Cursor.scalar_correspondence' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Value.Field.Cursor.scalar_correspondence

/-- info: 'Umpire.Operation.ParameterTable.actionDomain_iff' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Operation.ParameterTable.actionDomain_iff

end Temporal.Feature.Nexus3.Tests.TypedUnary
