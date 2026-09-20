import Umpire.Case.Projection.Lowering
import Umpire.Property.Evaluate

/-!
# Field relations

A field relation is the clause form a `property` writes as `relates:`: two typed fields of one
step -- an action's input or result, or the recorded event that confirms the step -- compared
through their schemas. It is the command-level form of a checked field Property
(`Umpire.CheckedFieldProperty`), and it lowers the same way: the Producer builds the Property
declaration the relation denotes, admits it against the Model with the field bindings the
operands' schemas give, and derives the monitor rule through `Umpire.Case.Projection.lower`, so the
Contract reads exactly the fields the relation compares.

The records here are what the command emits and the Producer reads. The operand's reference --
the Definition ID of the action or fact it belongs to -- is the Model's, resolved by the Producer
through the vocabulary, which is why an operand carries the member's spelling rather than its id.
-/

namespace Umpire.Case.Producer

open Umpire

/-- One operand of a relation: which payload it reads and where. `member` is the spelling of the
Model member the operand belongs to (the action for an input or result, the fact the recorded
event confirms for an observation), `spelling` the dotted path as the author wrote it, and
`presence` the presence reads the path traverses, each as the steps of a presence path. -/
structure FieldOperand where
  root : PropertyFieldRoot
  member : String
  spelling : String
  schema : Operation.RpcSchema
  side : Value.Side
  steps : List Value.Field.Step
  type : Operation.Singular
  presence : List (List Value.Field.Step)
  deriving BEq, Repr

/-- The structural coordinates the operand denotes once its member's Definition ID is known. -/
def FieldOperand.path (operand : FieldOperand) (reference : DefinitionId) : PropertyFieldPath :=
  { root := operand.root, reference, schema := operand.schema, side := operand.side,
    steps := operand.steps, type := operand.type }

/-- The presence facts the operand's read traverses, as the Boolean paths a comparison must
establish before the read. -/
def FieldOperand.presencePaths (operand : FieldOperand) (reference : DefinitionId) :
    List PropertyFieldPath :=
  let base := operand.path reference
  operand.presence.map fun steps => { base with steps, type := .boolean }

/-- The three relation forms: equality, inequality, and the presence of an optional field. -/
inductive FieldRelationOperator where
  | equal
  | notEqual
  | present
  deriving BEq, Repr

/-- One `relates:` line: the Property it declares, the action the step is about, and its operands.
A `present` relation has one operand. -/
structure FieldRelation where
  id : DefinitionId
  name : String
  action : String
  operator : FieldRelationOperator
  left : FieldOperand
  right : Option FieldOperand := none
  source : SourceLocation
  deriving BEq, Repr

/-- The suffix every derived rule of a relation carries: the rule is `<property>.relation`, and
its transitions `match-relation` and `reject-relation`. -/
def FieldRelation.ruleSuffix : String := "relation"

private def literalTrue (source : SourceLocation) : PropertyFieldOperand :=
  .literal (.boolean true) source

private def alwaysApplies (source : SourceLocation) : PropertyPredicate :=
  PropertyPredicate.compareFields .equal (literalTrue source) (literalTrue source) source

private def presenceHolds (path : PropertyFieldPath) (source : SourceLocation) :
    PropertyPredicate :=
  PropertyPredicate.compareFields .equal (.field path source) (literalTrue source) source

/-- The presence path of a `present` operand: the read that establishes it, made a presence read.
An operand whose path establishes nothing at its end has no presence to relate. -/
def FieldOperand.presentPath (operand : FieldOperand) (reference : DefinitionId) :
    Option PropertyFieldPath :=
  let base := operand.path reference
  match operand.steps.getLast? with
  | some .establish | some (.select _) =>
      some { base with
        steps := operand.steps.dropLast ++ [Value.Field.Step.present]
        type := .boolean }
  | _ => none

/-- The Property declaration one relation denotes: one branch whose clause establishes every
presence fact the operands traverse and compares the two reads (or, for `present`, holds the
operand's own presence). `requires` is the capability the Model's Properties require. -/
def FieldRelation.declaration (relation : FieldRelation) (requires : List DefinitionId)
    (leftReference rightReference : DefinitionId) : Except String Property := do
  let source := relation.source
  let ownedId (suffix : String) : DefinitionId := .of (relation.id.value ++ "." ++ suffix)
  let expectation ← match relation.operator, relation.right with
    | .present, _ =>
        let some presence := relation.left.presentPath leftReference
          | throw "relation.presence: the operand is always present"
        let base := relation.left.path leftReference
        let facts := relation.left.presence.dropLast.map fun steps =>
          { base with steps, type := .boolean }
        pure (PropertyPredicate.all (facts.map (presenceHolds · source) ++
          [presenceHolds presence source]))
    | operator, some right =>
        let leftPath := relation.left.path leftReference
        let rightPath := right.path rightReference
        let facts := relation.left.presencePaths leftReference ++ right.presencePaths rightReference
        let comparison := PropertyPredicate.compareFields
          (if operator == .equal then .equal else .notEqual)
          (.field leftPath source) (.field rightPath source) source
        pure (PropertyPredicate.all (facts.map (presenceHolds · source) ++ [comparison]))
    | _, none => throw "relation.operand: a comparison needs two operands"
  pure {
    id := relation.id
    source
    version := 2
    requires
    clauses := [.branches {
      id := ownedId "group", source, guard := alwaysApplies source
      cases := [{
        id := ownedId "case", source, guard := alwaysApplies source
        clauses := [⟨ownedId "clause", source, expectation⟩] }] }] }

/-- Admit one relation against a Model: the declaration it denotes, checked with the field bindings
its operands' schemas give under the references the Model resolved its members to. -/
def FieldRelation.check (relation : FieldRelation) (context : PropertyCheckContext)
    (requires : List DefinitionId) (leftReference rightReference : DefinitionId) :
    Except String CheckedFieldProperty := do
  let declaration ← relation.declaration requires leftReference rightReference
  let bindings := [PropertyFieldBinding.ofSchema leftReference relation.left.schema] ++
    (relation.right.toList.map fun right => PropertyFieldBinding.ofSchema rightReference right.schema)
  (CheckedFieldProperty.check { context with fieldBindings := bindings } declaration).mapError
    fun error => "relation.admission: " ++ error.kind.name ++ ": " ++ error.offendingValue

end Umpire.Case.Producer
