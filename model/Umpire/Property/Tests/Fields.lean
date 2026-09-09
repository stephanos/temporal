import Umpire.Property
import Umpire.Property.Tests.Fixtures
import Umpire.Operation.Action
import Umpire.Value.Field

/-! Independent same-step field requirements use the checked Property owner. -/
namespace Umpire.PropertyFieldsTests
open Umpire Operation Value

private def source : SourceLocation := ⟨"fields.lean", 7, 3, "authored"⟩

#guard (PropertyFieldComparison.check .equal (.literal (.bytes [0, 255]) source)
  (.literal (.bytes [255, 0]) source) source).toOption.isSome
#guard (PropertyFieldComparison.check .equal (.literal (.integer .int32 1) source)
  (.literal (.integer .int64 1) source) source).toOption.isNone
#guard (PropertyFieldComparison.check .equal (.literal (.enumeration "A" 1) source)
  (.literal (.enumeration "B" 1) source) source).toOption.isNone
#guard (PropertyFieldComparison.check .less (.literal (.bytes [0]) source)
  (.literal (.bytes [1]) source) source).toOption.isNone
#guard (PropertyFieldComparison.check .equal (.literal (.floating true 0) source)
  (.literal (.floating true 0) source) source).toOption.isNone

private def compareBytes : PropertyPredicate := .atom {
  field := .selectedAction
  reference := PropertyTests.requestCancel
  constraint := .fields ⟨.equal, .literal (.bytes [0, 255]) source,
    .literal (.bytes [255, 0]) source, source⟩ }
private def evaluateBytes := do
  let predicate ← checkPropertyPredicate PropertyTests.context PropertyTests.portableProperty
    .guard compareBytes
  let input ← checkPropertyPredicateInput predicate { context := .guard }
  pure (evaluatePropertyPredicate predicate input)
#guard evaluateBytes.toOption == some false

private def schema : Schema := ⟨"M", [{
  name := "M", protoSyntax := "proto3", descriptor := "m", fileContext := "", references := [],
  valueShape := some (.message [
    ⟨1, "count", .integer .int32, .singular, .implicit (.integer .int32 0), none⟩,
    ⟨2, "data", .bytes, .singular, .optional, none⟩,
    ⟨3, "selected", .text, .singular, .oneof "choice", none⟩,
    ⟨4, "alternate", .text, .singular, .oneof "choice", none⟩]) }]⟩
private def owner : RpcOwner where
  Witness _ _ := Unit
  schema _ := ⟨"example.Call", schema, schema, [], false, false⟩
private def limits : Limits := ⟨8, 10000, 1024, 100⟩
private def template := do
  let binding ← checkRpc owner (Request := Unit) (Response := Unit) ()
    (owner.schema (Request := Unit) (Response := Unit) ())
  ActionTemplate.check (rpc Empty binding) PropertyTests.requestCancel
private def rootPath (root : PropertyFieldRoot) (reference : DefinitionId) : PropertyFieldPath :=
  ⟨root, reference, owner.schema (Request := Unit) (Response := Unit) (), .request,
    [.field "M" 1], .integer .int32⟩
private def requestPath := rootPath .request PropertyTests.requestCancel
private def priorPath := rootPath .priorState PropertyTests.pendingCount
private def resultPath := rootPath .outcome PropertyTests.deliveredOutcome
private def context : PropertyCheckContext := { PropertyTests.context with
  fieldBindings := [PropertyFieldBinding.ofWitness owner (Request := Unit) (Response := Unit) ()
    PropertyTests.requestCancel, PropertyFieldBinding.ofWitness owner (Request := Unit) (Response := Unit) ()
    PropertyTests.pendingCount, PropertyFieldBinding.ofWitness owner (Request := Unit) (Response := Unit) ()
    PropertyTests.deliveredOutcome, PropertyFieldBinding.ofWitness owner (Request := Unit) (Response := Unit) ()
    PropertyTests.cancelDelivered] }
private def relation (left right : PropertyFieldPath) : PropertyPredicate := .atom {
  field := .selectedAction, reference := PropertyTests.requestCancel,
  constraint := .fields ⟨.equal, .field left source, .field right source, source⟩ }
private def requestProjection (value : Checked owner (Request := Unit) (Response := Unit) () .request limits)
    (cursor : Field.Cursor owner (Request := Unit) (Response := Unit) () .request limits
      valueType .singular .available) :
    Except Field.Error (PropertyFieldProjection owner (Request := Unit) (Response := Unit) ()) := do
  if same : cursor.origin.value = value.value then
    let t ← template.mapError fun _ => Field.Error.mk source "M" "template"
    have witness : t.declaration.reference = () := by
      change (t.declaration.reference : Unit) = ()
      exact Subsingleton.elim (α := Unit) t.declaration.reference ()
    let arguments : Checked owner t.declaration.reference .request limits := witness.symm ▸ value
    let selected : Field.Cursor owner t.declaration.reference .request limits
        valueType .singular .available := witness.symm ▸ cursor
    let projected ← PropertyFieldProjection.ofAction (ActionInstance.mk (template := t) arguments)
      selected (by simpa [selected, arguments] using same) source
    pure (witness ▸ projected)
  else throw ⟨source, "M", "wrong arguments"⟩
private def checkedValue (path : PropertyFieldPath) (count : Int) : Except Field.Error (PropertyFieldProjection owner (Request := Unit) (Response := Unit) ()) := do
  let value ← (Value.check owner (Request := Unit) (Response := Unit) () .request limits
    (message "M" [(1, literal (.integer .int32 count))])).mapError
    fun e => Field.Error.mk source e.path e.reason
  let ref ← Field.reference owner () .request "M" 1 source
  let cursor ← (Field.root value).field ref source
  let cursor ← cursor.refine (.integer .int32) .singular .available source
  if request : path.root = .request then requestProjection value cursor
  else PropertyFieldProjection.ofCursor path.root path.reference cursor request source
private def result (expression : PropertyPredicate) (kind : PropertyPredicateContext)
    (request prior response : Int) := do
  let a ← (checkedValue requestPath request).toOption
  let b ← (checkedValue priorPath prior).toOption
  let c ← (checkedValue resultPath response).toOption
  let predicate ← (CheckedFieldPredicate.check owner (Request := Unit) (Response := Unit) ()
    context PropertyTests.portableProperty kind expression).toOption
  let input ← (predicate.checkInput {
    context := kind
    selectedAction := some a.modelValue, priorState := some b.modelValue,
    modelOutcome := some c.modelValue } [a, b, c]).toOption
  pure (evaluatePropertyPredicate predicate.predicate input)
#guard result (relation requestPath priorPath) .guard 1 1 9 == some true
#guard result (relation requestPath priorPath) .guard 1 2 9 == some false
#guard result (relation requestPath resultPath) .expectation 1 9 1 == some true
#guard result (relation requestPath resultPath) .expectation 1 9 2 == some false
#guard result (relation requestPath resultPath) .guard 1 9 1 == none

private def compare (left right : PropertyFieldOperand) (operator : PropertyFieldOperator := .equal) :
    PropertyPredicate := PropertyPredicate.compareFields operator left right source
private def presencePath (number : Nat := 2) : PropertyFieldPath :=
  { requestPath with steps := [.field "M" number, .present], type := .boolean }
private def bytesPath : PropertyFieldPath :=
  { requestPath with steps := [.field "M" 2, .establish], type := .bytes }
private def isPresent (number : Nat := 2) :=
  compare (.field (presencePath number) source) (.literal (.boolean true) source)
private def bytesEqual := compare (.field bytesPath source) (.literal (.bytes [0, 255]) source)
private def admission (expression : PropertyPredicate) :=
  (CheckedFieldPredicate.check owner (Request := Unit) (Response := Unit) () context
    PropertyTests.portableProperty .guard expression).toOption.isSome
#guard !admission bytesEqual
#guard admission (.all [isPresent, bytesEqual])
#guard !admission (.any [isPresent, bytesEqual])
#guard !admission (.all [.any [isPresent, isPresent], bytesEqual])
#guard !admission (.all [.not isPresent, bytesEqual])
#guard admission (.any [.all [isPresent, bytesEqual], isPresent])
private def selectedPath : PropertyFieldPath :=
  { requestPath with steps := [.field "M" 3, .select "choice"], type := .text }
#guard admission (.all [isPresent 3,
  compare (.field selectedPath source) (.literal (.text "") source)])
#guard !admission (.all [isPresent 3,
  compare (.field { selectedPath with steps := [.field "M" 3, .select "wrong"] } source)
    (.literal (.text "") source)])
#guard !admission (.all [isPresent 4,
  compare (.field selectedPath source) (.literal (.text "") source)])
#guard !admission (compare (.field { requestPath with type := .integer .int64 } source)
  (.literal (.integer .int64 0) source))
#guard !admission (compare (.field { requestPath with schema := { requestPath.schema with fullName := "other.Call" } } source)
  (.literal (.integer .int32 0) source))
#guard !admission (compare (.field { requestPath with side := .response } source)
  (.literal (.integer .int32 0) source))
#guard !admission (compare (.field { requestPath with steps := [.field "Wrong" 1] } source)
  (.literal (.integer .int32 0) source))
#guard (PropertyFieldComparison.check .equal (.literal (.integer .uint32 (-1)) source)
  (.literal (.integer .uint32 0) source) source).toOption.isNone
#guard (PropertyFieldComparison.check .equal (.literal (.integer .int32 2147483648) source)
  (.literal (.integer .int32 0) source) source).toOption.isNone
#guard (PropertyFieldComparison.check .atMost (.literal (.integer .uint64 18446744073709551615) source)
  (.literal (.integer .uint64 18446744073709551615) source) source).toOption.isSome

#guard (field_compare% (.literal (.bytes [0, 255]) source) with .equal
  (.literal (.bytes [255, 0]) source) at source) ==
  compare (.literal (.bytes [0, 255]) source) (.literal (.bytes [255, 0]) source)

private def payloadProjections (raw : Raw) (number : Nat) (valueType : Singular)
    (selection : Option String := none) := do
  let value ← (Value.check owner (Request := Unit) (Response := Unit) () .request limits raw).toOption
  let ref ← (Field.reference owner () .request "M" number source).toOption
  let cursor ← ((Field.root value).field ref source).toOption
  let present ← (cursor.present source).toOption
  let presence ← (requestProjection value present).toOption
  let selected := match selection with
    | none => do
      let optional ← (cursor.refine valueType .singular .optional source).toOption
      let ready ← (optional.establish source).toOption
      (requestProjection value ready).toOption
    | some group => do
      let oneof ← (cursor.refine valueType .singular (.oneof group) source).toOption
      let ready ← (oneof.select group source).toOption
      (requestProjection value ready).toOption
  pure (presence.modelValue, presence :: selected.toList)
private def payloadResult (raw : Raw) (expression : PropertyPredicate)
    (number : Nat := 2) (valueType : Singular := .bytes) (selection : Option String := none) := do
  let (modelValue, projections) ← payloadProjections raw number valueType selection
  let predicate ← (CheckedFieldPredicate.check owner (Request := Unit) (Response := Unit) () context
    PropertyTests.portableProperty .guard expression).toOption
  let input ← (predicate.checkInput { context := .guard, selectedAction := some modelValue } projections).toOption
  pure (evaluatePropertyPredicate predicate.predicate input)
private def payloadRequirement := PropertyPredicate.all [isPresent, bytesEqual]
#guard payloadResult (message "M" [(2, literal (.bytes [0, 255]))]) payloadRequirement == some true
#guard payloadResult (message "M" [(2, literal (.bytes [255, 0]))]) payloadRequirement == some false
#guard payloadResult (message "M" [(2, literal (.bytes []))]) isPresent == some true
#guard payloadResult (message "M" []) isPresent == some false
#guard payloadResult (message "M" []) payloadRequirement == some false
#guard payloadResult (message "M" []) (.not isPresent) == some true
private def selectedRequirement := PropertyPredicate.all [isPresent 3,
  compare (.field selectedPath source) (.literal (.text "") source)]
#guard payloadResult (message "M" [(3, literal (.text ""))]) selectedRequirement 3 .text (some "choice") == some true
#guard payloadResult (message "M" [(4, literal (.text ""))]) selectedRequirement 3 .text (some "choice") == some false

private def inputError (expression : PropertyPredicate) (raw : PropertyPredicateInput)
    (projections : List (PropertyFieldProjection owner (Request := Unit) (Response := Unit) ())) :=
  match CheckedFieldPredicate.check owner (Request := Unit) (Response := Unit) () context
      PropertyTests.portableProperty .guard expression with
  | .error error => some error
  | .ok predicate => match predicate.checkInput raw projections with
    | .error error => some error
    | .ok _ => none
private def countIsZero := compare (.field requestPath { source with line := 91, column := 8 })
  (.literal (.integer .int32 0) source)
#guard (inputError countIsZero { context := .guard } []).map (·.sourceLocation) ==
  some (some { source with line := 91, column := 8 })
#guard (inputError (.not countIsZero) { context := .guard } []).map (·.kind) == some .missingPredicateInput
#guard (inputError countIsZero { context := .expectation } []).map (·.kind) == some .invalidPredicateContext
#guard (do
  let a ← (checkedValue requestPath 0).toOption
  let b ← (checkedValue requestPath 1).toOption
  pure ((inputError countIsZero { context := .guard, selectedAction := some a.modelValue } [b]).isSome)) == some true
#guard (do
  let a ← (checkedValue requestPath 0).toOption
  pure ((inputError countIsZero { context := .guard, selectedAction := some a.modelValue } [a, a]).isSome)) == some true

private def otherOwner : RpcOwner where
  Witness _ _ := Bool
  schema _ := owner.schema (Request := Unit) (Response := Unit) ()
#guard_msgs (error, substring := true) in
example (predicate : CheckedFieldPredicate owner (Request := Unit) (Response := Unit) () .guard)
    (projection : PropertyFieldProjection otherOwner (Request := Unit) (Response := Unit) true) :=
  predicate.checkInput { context := .guard } [projection]
#guard_msgs (error, substring := true) in
#check PropertyFieldValue.ofCursor
#guard_msgs (error, substring := true) in
example (value : ModelValue) : PropertyFieldProjection owner (Request := Unit) (Response := Unit) () := ⟨value⟩

private def truePredicate := compare (.literal (.boolean true) source) (.literal (.boolean true) source)
private def declaration : PropertyDeclaration := { PropertyTests.portableProperty with
  version := 2
  clauses := [.sameStepCases {
    id := .of "test.property.fields.group", source, guard := truePredicate,
    cases := [{
      id := .of "test.property.fields.case", source, guard := truePredicate,
      clauses := [⟨.of "test.property.fields.state", source, relation requestPath priorPath⟩,
        ⟨.of "test.property.fields.result", source, relation requestPath resultPath⟩] }] }] }
private def wholeProperty (prior response : Int) := do
  let a ← (checkedValue requestPath 1).toOption
  let b ← (checkedValue priorPath prior).toOption
  let c ← (checkedValue resultPath response).toOption
  let checked ← (CheckedFieldProperty.check owner (Request := Unit) (Response := Unit) () context declaration).toOption
  let trace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
    initialState := b.modelValue
    steps := [{
      selectedAction := a.modelValue, modelOutcome := c.modelValue,
      resultingState := b.modelValue, observations := [] }] }
  let input ← (checked.checkInput trace [[a, b, c]]).toOption
  pure (evaluateProperty checked.property input).satisfied
#guard wholeProperty 1 1 == some true
#guard wholeProperty 2 1 == some false
#guard wholeProperty 1 2 == some false
#guard (do
  let a ← (checkedValue requestPath 0).toOption
  let predicate ← (CheckedFieldPredicate.check owner (Request := Unit) (Response := Unit) () context
    PropertyTests.portableProperty .guard countIsZero).toOption
  let input ← (predicate.checkInput { context := .guard, selectedAction := some a.modelValue } [a]).toOption
  pure (input.operandValue (.field requestPath { source with line := 91, column := 8 }))) == some (some (.integer .int32 0))

#guard (do
  let predicate ← (CheckedFieldPredicate.check owner (Request := Unit) (Response := Unit) () context
    PropertyTests.portableProperty .guard truePredicate).toOption
  let input ← (predicate.checkInput { context := .guard } []).toOption
  pure (input.operandValue (.literal (.bytes [99]) source))) == some none

example (predicate : CheckedPropertyPredicate kind) (input : CheckedPropertyPredicateInput predicate)
    (operand : PropertyFieldOperand) (value : Scalar) (h : input.operandValue operand = some value) :
    input.denotesOperand operand value := input.operandValue_denotes operand value h

private def literalResult (operator : PropertyFieldOperator) (left right : Scalar) := do
  let predicate ← (checkPropertyPredicate context PropertyTests.portableProperty .guard
    (compare (.literal left source) (.literal right source) operator)).toOption
  let input ← (checkPropertyPredicateInput predicate { context := .guard }).toOption
  pure (evaluatePropertyPredicate predicate input)
#guard literalResult .equal (.bytes [0, 255]) (.bytes [0, 255]) == some true
#guard literalResult .notEqual (.bytes [0, 255]) (.bytes [255, 0]) == some true
#guard literalResult .equal (.enumeration "E" 1) (.enumeration "E" 1) == some true
#guard literalResult .equal (.enumeration "E" 1) (.enumeration "E" 2) == some false
#guard literalResult .less (.integer .int32 (-1)) (.integer .int32 0) == some true
#guard literalResult .atMost (.integer .int32 0) (.integer .int32 0) == some true
#guard literalResult .greater (.integer .uint64 18446744073709551615) (.integer .uint64 0) == some true
#guard literalResult .atLeast (.integer .int32 (-1)) (.integer .int32 0) == some false
#guard literalResult .equal (.integer .int32 1) (.integer .sint32 1) == none
#guard literalResult .less (.enumeration "E" 1) (.enumeration "E" 2) == none
private def resultingEvent (state event : Int) := do
  let statePath := { priorPath with root := .resultingState }
  let eventPath := rootPath .event PropertyTests.cancelDelivered
  let a ← (checkedValue statePath state).toOption
  let b ← (checkedValue eventPath event).toOption
  let checked ← (CheckedFieldPredicate.check owner (Request := Unit) (Response := Unit) () context
    PropertyTests.portableProperty .expectation (relation statePath eventPath)).toOption
  let input ← (checked.checkInput {
    context := .expectation
    resultingState := some a.modelValue, facts := some [b.modelValue] } [a, b]).toOption
  pure (evaluatePropertyPredicate checked.predicate input)
#guard resultingEvent 1 1 == some true
#guard resultingEvent 1 2 == some false
#guard_msgs (error, substring := true) in
example (predicate : CheckedFieldPredicate owner (Request := Unit) (Response := Unit) () .guard)
    (projection : PropertyFieldProjection owner (Request := Bool) (Response := Unit) ()) :=
  predicate.checkInput { context := .guard } [projection]

#guard payloadResult (message "M" [(1, literal (.integer .int32 99)),
  (2, literal (.bytes [0, 255])), (4, literal (.text "unconstrained"))]) payloadRequirement == some true
#guard (match CheckedFieldPredicate.check owner (Request := Unit) (Response := Unit) () context
    PropertyTests.portableProperty .guard
    (compare (.field { requestPath with type := .integer .int64 } { source with line := 81, column := 6 })
      (.literal (.integer .int64 0) source)) with
  | .error e => e.sourceLocation == some { source with line := 81, column := 6 }
  | _ => false)
#guard (PropertyFieldComparison.check .equal (.literal (.integer .int32 1) source)
  (.literal (.integer .int64 1) { source with line := 82, column := 9 }) source).toOption.isNone
example (operator : PropertyFieldOperator) (left right : PropertyFieldOperand) (at : SourceLocation) :
    (field_compare% left with operator right at at) =
      PropertyPredicate.compareFields operator left right at := rfl
private def identityComparison : PropertyFieldComparison :=
  ⟨.equal, .field requestPath source, .field priorPath source, source⟩
#guard identityComparison.canonical == { identityComparison with source := { source with line := 100 } }.canonical
#guard identityComparison.canonical != { identityComparison with operator := .notEqual }.canonical
#guard identityComparison.canonical != { identityComparison with
  right := .field { priorPath with steps := [.field "M" 2, .establish], type := .bytes } source }.canonical

end Umpire.PropertyFieldsTests
