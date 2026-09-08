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
    PropertyTests.deliveredOutcome] }
private def relation (left right : PropertyFieldPath) : PropertyPredicate := .atom {
  field := .selectedAction, reference := PropertyTests.requestCancel,
  constraint := .fields ⟨.equal, .field left source, .field right source, source⟩ }
private def checkedValue (path : PropertyFieldPath) (count : Int) : Except Field.Error PropertyFieldValue := do
  let value ← (Value.check owner (Request := Unit) (Response := Unit) () .request limits
    (message "M" [(1, literal (.integer .int32 count))])).mapError
    fun e => Field.Error.mk source e.path e.reason
  let ref ← Field.reference owner () .request "M" 1 source
  let cursor ← (Field.root value).field ref source
  let cursor ← cursor.refine (.integer .int32) .singular .available source
  PropertyFieldValue.ofCursor path.root path.reference cursor source
private def result (expression : PropertyPredicate) (kind : PropertyPredicateContext)
    (request prior response : Int) := do
  let a ← (checkedValue requestPath request).toOption
  let b ← (checkedValue priorPath prior).toOption
  let c ← (checkedValue resultPath response).toOption
  let predicate ← (checkPropertyPredicate context PropertyTests.portableProperty kind expression).toOption
  let input ← (checkPropertyPredicateInput predicate {
    context := kind
    selectedAction := some a.modelValue, priorState := some b.modelValue,
    modelOutcome := some c.modelValue, fieldValues := [a, b, c] }).toOption
  pure (evaluatePropertyPredicate predicate input)
#guard result (relation requestPath priorPath) .guard 1 1 9 == some true
#guard result (relation requestPath priorPath) .guard 1 2 9 == some false
#guard result (relation requestPath resultPath) .expectation 1 9 1 == some true
#guard result (relation requestPath resultPath) .expectation 1 9 2 == some false
#guard result (relation requestPath resultPath) .guard 1 9 1 == none

end Umpire.PropertyFieldsTests
