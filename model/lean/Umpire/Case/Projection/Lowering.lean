import Umpire.Case.Compiler

/-!
# The derived monitor rule

A checked field Property compares modeled field coordinates, and a Case checks that comparison at
runtime with one monitor rule over a declared Observation. `lower` derives that rule from the
checked Property rather than letting a Producer restate it: the Property decides which coordinates
the rule reads, which comparison it makes, where each operand comes from and which request field
the Program must construct, so a Property edit that moves a coordinate moves the rule, and a field
the Property stops comparing is no longer read.

What the Property does not state is carried by a `Realization` (the rule-level one, distinct from
`Umpire.Case.Producer.Realization`, which is a whole Program and its read-back coordinates): the
request-side literal
assignments the Program makes, the rule's identity suffix, and a `CapturePolicy`. The policy picks
one of the two rule shapes:

* `.none` is the two-transition safety rule. The Property compares a request operand, which the
  Program constructs from one realized literal, with an operand the Observation carries; the rule
  matches the observed read against that literal and is violated when it disagrees.
* `.crossEvent` is the three-state capture rule. The Property compares a prior-state operand with
  one the Observation carries; the rule captures the earlier event the policy's selector identifies
  and matches the later event's read against the captured one.

Every operand root resolves the same way under either shape: a request operand is a realized
literal, a prior-state operand is captured, and an outcome or event operand is observed. Presence
atoms (an optional or oneof read compared with `true`) establish the steps a read traverses and are
consumed rather than compared; the rule guards its derived read with its own presence checks, the
Observation or capture it reads from and the read path itself.

The result is a `Lowered` value. Its `DerivedRule` is the correspondence certificate: every
coordinate the rule reads is one the Property compares or the selector the realization names, and
every literal the rule compares against is one the realization assigns. Its coverage request names
exactly the request literals the comparison consumed, so `Umpire.Case.Compiler.compile` checks the
Program constructs each of them. A Property with no field comparison lowers to no rule.

Each rejection names its construct: `rule.clause-shape` for a field atom no conjunction of
comparisons carries, `rule.comparisons` for more than one comparison, `rule.operator` for an
ordering, `rule.unimplied` for an operand pair or field guard the policy does not realize,
`rule.literal-unassigned` for a request operand the realization assigns no literal,
`rule.property-literal` for a literal the Property writes itself rather than the Program assigning
it, and
`rule.observation-type` for an Observation that carries no single message. A coordinate with no
read path rejects with the reason `Projection.readPath` gives. `rule.certificate` names a lowering
that failed its own correspondence check, which the derivation never produces.

A Case over several instances lowers a Property once per instance. `InstancedRule.fold` folds those
derivations into one Rule whose compared literal is an instance value and which carries one Rule
instance per placement; each placement keeps its `DerivedRule`, and placements that disagree once
their literal is erased reject as `relation.instance-shape`.
-/

namespace Umpire.Case.Projection

open Umpire
open Umpire.Case
open temporal.server.api.testpilot.v1

/-! ### Realization -/

/-- The earlier event a capture rule retains, named by the value its observed coordinates record. -/
structure Selector where
  path : PropertyFieldPath
  value : Operation.Scalar
  deriving BEq, DecidableEq, Repr

/-- How a derived rule relates events. `crossEvent` names the earlier event it captures, the state
the rule waits in once captured, and the response label of the transition that answers it. -/
inductive CapturePolicy where
  | none
  | crossEvent (selector : Selector) (state response : String)
  deriving BEq, DecidableEq, Repr

/-- What a Case realizes that its checked field Property does not state. -/
structure Realization where
  /-- Request-side literal assignments the Program makes, with the instruction making each. -/
  literals : List Coverage.InputMapping := []
  /-- Appended to the Property ID to name the rule and its transitions. -/
  ruleSuffix : String
  capture : CapturePolicy := .none
  deriving BEq, DecidableEq, Repr

/-- The observed coordinates a capture policy reads to select its earlier event. -/
def Realization.selectorPaths (realization : Realization) : List PropertyFieldPath :=
  match realization.capture with
  | .none => []
  | .crossEvent selector _ _ => [selector.path]

/-- Every literal value the realization assigns, the selector's included. -/
def Realization.assigned (realization : Realization) : List Operation.Scalar :=
  realization.literals.map (·.value) ++ match realization.capture with
    | .none => []
    | .crossEvent selector _ _ => [selector.value]

/-! ### What a checked field Property compares -/

private def clauseExpectations (property : CheckedFieldProperty) :
    List CheckedPropertySameStepClause :=
  property.property.clauses.flatMap fun clause => match clause with
    | .branches group => group.cases.flatMap (·.clauses)
    | _ => []

private def temporalGuards (clause : CheckedPropertyTemporalClause) : List PropertyPredicate :=
  [clause.guard.expression] ++ clause.exception.toList.map (·.condition.expression) ++
    clause.caseGuard.toList.map (·.expression) ++
    clause.caseException.toList.map (·.condition.expression)

/-- Every applicability condition of the Property. The derived rule does not read them, so a field
compared in one is a comparison the rule would silently drop. -/
private def guardExpressions (property : CheckedFieldProperty) : List PropertyPredicate :=
  property.property.clauses.flatMap fun clause => match clause with
    | .branches group =>
        [group.guard.expression] ++ group.exception.toList.map (·.condition.expression) ++
          group.cases.flatMap fun branch =>
            [branch.guard.expression] ++ branch.exception.toList.map (·.condition.expression) ++
              branch.temporalClauses.flatMap temporalGuards
    | .guardedEventuallyWithin temporal | .guardedNeverWithin temporal => temporalGuards temporal
    | _ => []

/-- The field coordinates a predicate compares, literal operands aside. -/
private def fieldPaths (predicate : PropertyPredicate) : List PropertyFieldPath :=
  predicate.fieldOperands.filterMap fun operand => match operand with
    | .field path _ => some path
    | .literal _ _ => none

/-- Every field coordinate the Property's same-step expectations compare, in declaration order. -/
def comparedFields (property : CheckedFieldProperty) : List PropertyFieldPath :=
  (clauseExpectations property).flatMap fun clause => fieldPaths clause.expectation.expression

/-- A presence atom: an optional or oneof read compared with `true`. -/
private def establishesPresence (comparison : PropertyFieldComparison) : Bool :=
  comparison.operator == .equal && match comparison.left, comparison.right with
    | .field path _, .literal (.boolean true) _ => path.steps.getLast? == some .present
    | _, _ => false

/-- The atoms of a conjunction, or none when a disjunction or negation is in the way. -/
private def conjuncts : PropertyPredicate → Option (List PropertyAtom)
  | .atom atom => some [atom]
  | .all items => (items.mapM conjuncts).map List.flatten
  | .any _ | .not _ => none

/-! ### The derived shape -/

/-- One coordinate and the runtime read path derived from it. Only `readPath` constructs one, so the
segments a rule reads are always the segments its coordinates derive. -/
structure Read where
  private mk ::
  path : PropertyFieldPath
  segments : String

/-- The schema side a modeled coordinate belongs to. -/
private def sideSchema (path : PropertyFieldPath) : Operation.Schema :=
  if path.side == .request then path.schema.request else path.schema.response

private def Read.of (root : String) (path : PropertyFieldPath) : Except String Read := do
  let segments ← readPath (sideSchema path) root path.steps
  pure ⟨path, Testpilot.Authoring.Path.make (segments.map fun segment => match segment with
    | .field name => Testpilot.Authoring.Path.field name
    | .oneof group member => Testpilot.Authoring.Path.oneofMember group member).toArray⟩

/-- One realized literal, the coordinate that supplied it, and the exact wire value it constructs. -/
structure Literal where
  private mk ::
  path : PropertyFieldPath
  scalar : Operation.Scalar
  wire : temporal.server.api.testpilot.v1.Value

/-- The literal `scalar` constructs, an enum value named by the schema of the coordinate `path` that
supplied it. -/
private def Literal.of (path : PropertyFieldPath) (scalar : Operation.Scalar) : Except String Literal := do
  pure ⟨path, scalar, ← Coverage.scalarValue (sideSchema path) scalar⟩

/-- The two rule shapes, before rendering. `negated` holds for a `notEqual` comparison. -/
inductive Shape where
  | safety (negated : Bool) (observed : Read) (literal : Literal)
  | capture (negated : Bool) (captured observed selector : Read) (literal : Literal)
      (state response : String)

/-- Every coordinate a shape reads. -/
def Shape.reads : Shape → List PropertyFieldPath
  | .safety _ observed _ => [observed.path]
  | .capture _ captured observed selector _ _ _ => [selector.path, captured.path, observed.path]

/-- The one literal a shape compares against. -/
def Shape.literal : Shape → Literal
  | .safety _ _ literal | .capture _ _ _ _ literal _ _ => literal

/-- Every literal a shape compares against. -/
def Shape.literals (shape : Shape) : List Operation.Scalar :=
  [shape.literal.scalar]

private def comparison (negated : Bool) (left right : Expression) : Expression :=
  let equal := Testpilot.Authoring.Expr.equal left right
  if negated then Testpilot.Authoring.Expr.negate equal else equal

/-- The conjuncts that compare `left` with `right` only on an event that establishes `read`. An
equality with an absent operand is false, so it needs no presence check; its negation is true there,
so a negated comparison keeps `present read`. -/
private def compared (negated : Bool) (read left right : Expression) : Array Expression :=
  if negated then #[Testpilot.Authoring.Expr.present read, comparison negated left right]
  else #[comparison negated left right]

/-- Render a shape as the one Contract rule it denotes, comparing against `value` where the shape
compares against its literal. The rule distinguishes the answers the model Property does: an event
that establishes the observed field and disagrees is a violation, not an absence, and an event that
never establishes it leaves the rule pending, so a Run that produced no such event still closes
inconclusive. A capture rule has no violated state: the event it captured decides which later event
it waits for, and one that never arrives leaves it pending. -/
private def Shape.renderComparing (shape : Shape) (value : Expression)
    (ruleId suffix observation root : String) : ContractRule :=
  let observed := Testpilot.Authoring.Expr.observation observation
  let projected := Testpilot.Authoring.Expr.path
  let present := Testpilot.Authoring.Expr.present
  let all := Testpilot.Authoring.Expr.all
  let completed := #[RunEventKind.RUN_EVENT_KIND_INSTRUCTION_COMPLETED]
  match shape with
  | .safety negated read _ =>
      Testpilot.Authoring.Contract.rule ruleId .CONTRACT_RULE_KIND_SAFETY "pending"
        #[Testpilot.Authoring.Contract.state "pending" .CONTRACT_STATE_STATUS_PENDING,
          Testpilot.Authoring.Contract.state "satisfied" .CONTRACT_STATE_STATUS_SATISFIED,
          Testpilot.Authoring.Contract.state "violated" .CONTRACT_STATE_STATUS_VIOLATED]
        #[Testpilot.Authoring.Contract.transition ("match-" ++ suffix) "pending" "satisfied"
            completed
            (all (#[present observed] ++ compared negated (projected observed read.segments)
              (projected observed read.segments) value))
            .CONTRACT_SUPPORT_KIND_MATCHING_EVENT,
          Testpilot.Authoring.Contract.transition ("reject-" ++ suffix) "pending" "violated"
            completed
            (all (#[present observed] ++ compared (!negated) (projected observed read.segments)
              (projected observed read.segments) value))
            .CONTRACT_SUPPORT_KIND_MATCHING_EVENT]
  | .capture negated captured read selector _ state response =>
      let captureId := state ++ "-" ++ suffix
      let retained := Testpilot.Authoring.Expr.capture captureId
      Testpilot.Authoring.Contract.rule ruleId .CONTRACT_RULE_KIND_SAFETY "pending"
        #[Testpilot.Authoring.Contract.state "pending" .CONTRACT_STATE_STATUS_PENDING,
          Testpilot.Authoring.Contract.state state .CONTRACT_STATE_STATUS_PENDING,
          Testpilot.Authoring.Contract.state "satisfied" .CONTRACT_STATE_STATUS_SATISFIED]
        #[Testpilot.Authoring.Contract.transition ("capture-" ++ captureId) "pending" state
            completed
            (all #[present observed, comparison false (projected observed selector.segments) value])
            .CONTRACT_SUPPORT_KIND_MATCHING_EVENT
            #[Testpilot.Authoring.Contract.captureAssignment captureId observation],
          Testpilot.Authoring.Contract.transition ("match-" ++ response ++ "-" ++ suffix) state
            "satisfied" completed
            (all (#[present retained] ++ compared negated (projected observed read.segments)
              (projected retained captured.segments) (projected observed read.segments)))
            .CONTRACT_SUPPORT_KIND_MATCHING_EVENT]
        (captures := #[Testpilot.Authoring.Contract.capture captureId
          (Testpilot.Authoring.Types.messageType root)])

/-- Render a shape as the one Contract rule it denotes, comparing against its own literal. -/
def Shape.render (shape : Shape) (ruleId suffix observation root : String) : ContractRule :=
  shape.renderComparing (Testpilot.Authoring.Expr.literal shape.literal.wire) ruleId suffix
    observation root

/-! ### The certificate and the lowering -/

/-- List membership decided through `DecidableEq`. The coordinate and mapping types also derive a
`BEq` that is not known to be lawful, so the library's `BEq`-based decision does not apply. -/
private def decideMem {α : Type} [DecidableEq α] (value : α) : (list : List α) → Decidable (value ∈ list)
  | [] => isFalse nofun
  | head :: rest =>
      if same : value = head then isTrue (same ▸ .head rest)
      else match decideMem value rest with
        | isTrue member => isTrue (.tail head member)
        | isFalse absent => isFalse fun
          | .head _ => same rfl
          | .tail _ member => absent member

private local instance {α : Type} [DecidableEq α] (value : α) (list : List α) :
    Decidable (value ∈ list) :=
  decideMem value list

/-- A monitor rule derived from a checked field Property, with the correspondence that justifies
it: every coordinate it reads is one `property` compares or the selector `realization` names, and
every literal it compares against is one `realization` assigns. -/
structure DerivedRule (property : CheckedFieldProperty) (realization : Realization) where
  observation : String
  root : String
  shape : Shape
  reads_compared : ∀ path ∈ shape.reads,
    path ∈ comparedFields property ∨ path ∈ realization.selectorPaths
  literals_assigned : ∀ value ∈ shape.literals, value ∈ realization.assigned

/-- The rule ID this derivation's rule concludes under: the Property's ID and the realization's
suffix. -/
def DerivedRule.ruleId {property : CheckedFieldProperty} {realization : Realization}
    (_ : DerivedRule property realization) : String :=
  property.property.id.value ++ "." ++ realization.ruleSuffix

/-- The generated Contract rule this derivation denotes. -/
def DerivedRule.rule {property : CheckedFieldProperty} {realization : Realization}
    (derived : DerivedRule property realization) : ContractRule :=
  derived.shape.render derived.ruleId realization.ruleSuffix derived.observation derived.root

/-- A checked field Property lowered for one Case: its derived rule, if the Property compares any
field, and the request coverage that rule's literals require. -/
structure Lowered (property : CheckedFieldProperty) (realization : Realization) where
  rule : Option (DerivedRule property realization)
  coverage : Coverage.Request
  coverage_assigned : ∀ mapping ∈ coverage.inputs,
    mapping ∈ realization.literals ∧ mapping.path ∈ comparedFields property

/-- The derived rule as the Compiler admits it, bound to the checked Property it came from. -/
def Lowered.contractLowering {property : CheckedFieldProperty} {realization : Realization}
    (lowered : Lowered property realization) : Option Compiler.ContractLowering :=
  lowered.rule.map fun derived =>
    .monitor ⟨property.property.id.value, property.property.behaviorFingerprint.render, .property⟩
      derived.rule

/-! ### One Rule over several placements

A Case over several instances of an entity lowers a relation once per instance, and the derivations
differ only in the literal each compares against and in their suffixes. `InstancedRule.fold` lets one
Rule stand for all of them: the compared literal becomes the Rule's one instance value, and each
derivation contributes a Rule instance that concludes under the derivation's own rule ID and assigns
the derivation's own literal. Every derivation keeps its `DerivedRule`, so the fold restates no
certificate; what it adds is the check that the derivations agree once their literals are erased,
which is what lets the Rule rendered from the first stand for the rest. -/

/-- A literal's value erased to its type: the constructor and the integer kind or enum name. -/
private def erasedScalar : Operation.Scalar → Operation.Scalar
  | .boolean _ => .boolean false
  | .text _ => .text ""
  | .bytes _ => .bytes []
  | .integer kind _ => .integer kind 0
  | .enumeration name _ => .enumeration name 0
  | .floating double _ => .floating double 0

/-- Everything a derivation renders except its compared literal's value: the Observation it reads
and its message, the shape kind with its capture state names, the negation, every read with its
segments, and the compared literal's coordinate and type. -/
structure Erased where
  observation : String
  root : String
  capture : Option (String × String)
  negated : Bool
  reads : List (PropertyFieldPath × String)
  literal : PropertyFieldPath
  literalType : Operation.Scalar
  deriving DecidableEq

/-- A derivation with its compared literal's value erased. -/
def DerivedRule.erased {property : CheckedFieldProperty} {realization : Realization}
    (derived : DerivedRule property realization) : Erased :=
  let (capture, negated, reads) := match derived.shape with
    | .safety negated observed _ => (none, negated, [observed])
    | .capture negated captured observed selector _ state response =>
        (some (state, response), negated, [captured, observed, selector])
  { observation := derived.observation, root := derived.root, capture, negated
    reads := reads.map fun read => (read.path, read.segments)
    literal := derived.shape.literal.path
    literalType := erasedScalar derived.shape.literal.scalar }

/-- The instance value type of a compared field, by its schema type. Only text, integer and enum
fields have one: preparation admits no other instance value type, and a boolean in particular is
excluded because capture analysis prunes on boolean literals, which a Rule analyzed once for all of
its instances cannot do per instance. -/
private def instanceType : Operation.Singular → Option SingularType
  | .text => some (Testpilot.Authoring.Types.scalar .SCALAR_KIND_TEXT)
  | .integer kind => some (Testpilot.Authoring.Types.scalar (match kind with
      | .int32 => .SCALAR_KIND_INT32 | .int64 => .SCALAR_KIND_INT64
      | .uint32 => .SCALAR_KIND_UINT32 | .uint64 => .SCALAR_KIND_UINT64
      | .sint32 => .SCALAR_KIND_SINT32 | .sint64 => .SCALAR_KIND_SINT64
      | .fixed32 => .SCALAR_KIND_FIXED32 | .fixed64 => .SCALAR_KIND_FIXED64
      | .sfixed32 => .SCALAR_KIND_SFIXED32 | .sfixed64 => .SCALAR_KIND_SFIXED64))
  | .enumeration name => some (Testpilot.Authoring.Types.enumeration name)
  | .boolean | .bytes | .message _ | .floating _ | .unsupported _ => none

/-- The name of the last field a coordinate steps through, from its side's schema. -/
private def lastFieldName (path : PropertyFieldPath) : Option String := do
  let .field containing number ← (path.steps.filter fun step => match step with
      | .field _ _ => true
      | _ => false).getLast?
    | none
  let field ← (Value.Field.schemaFields (sideSchema path)).find? fun item =>
    item.1 == containing && item.2.number == number
  pure field.2.name

/-- One placement's derivation of a checked field Property, under that placement's realization. -/
structure Placed (property : CheckedFieldProperty) where
  realization : Realization
  derived : DerivedRule property realization

/-- One Rule standing for every placement of a checked field Property. The certificate is each
placement's own `DerivedRule`, so the value each Rule instance assigns is a literal its placement's
realization assigns, and the agreement of every placement's erased derivation with the first's, so
the Rule rendered from the first reads what every placement reads and compares at the same type.
Only `fold` constructs one. -/
structure InstancedRule (property : CheckedFieldProperty) where
  private mk ::
  /-- Appended to the Property ID to name the Rule, its capture and its transitions. -/
  ruleSuffix : String
  /-- The one value every Rule instance assigns: the compared literal. -/
  instanceValue : ContractInstanceValue
  first : Placed property
  rest : List (Placed property)
  shapes_agree : ∀ placed ∈ rest, placed.derived.erased = first.derived.erased

/-- Fold the placements of one checked field Property into one Rule named by `ruleSuffix`. `none`
leaves each placement its own plain rule: over at most one placement there is nothing to share, and
a compared literal whose type has no instance value type cannot be shared. Placements that disagree
once their literal is erased reject as `relation.instance-shape`, as does a compared coordinate that
names no field. -/
def InstancedRule.fold (property : CheckedFieldProperty) (ruleSuffix : String) :
    List (Placed property) → Except String (Option (InstancedRule property))
  | [] | [_] => pure none
  | first :: rest =>
      if agree : ∀ placed ∈ rest, placed.derived.erased = first.derived.erased then
        let literal := first.derived.shape.literal
        match instanceType literal.path.type with
        | none => pure none
        | some type => do
            let some name := lastFieldName literal.path | throw "relation.instance-shape"
            pure (some ⟨ruleSuffix, Testpilot.Authoring.Contract.instanceValue name type, first,
              rest, agree⟩)
      else throw "relation.instance-shape"

/-- The Rule the fold denotes: rendered once from the first placement under the unsuffixed
`ruleSuffix`, comparing against the instance value where each placement compared against its
literal, with one Rule instance per placement in placement order, each concluding under that
placement's rule ID and assigning that placement's literal. -/
def InstancedRule.rule {property : CheckedFieldProperty} (instanced : InstancedRule property) :
    ContractRule :=
  let first := instanced.first.derived
  let valueId := instanced.instanceValue.instance_value_id
  let rendered := first.shape.renderComparing (Testpilot.Authoring.Expr.instanceValue valueId)
    (property.property.id.value ++ "." ++ instanced.ruleSuffix) instanced.ruleSuffix
    first.observation first.root
  { rendered with
    instance_values := #[instanced.instanceValue]
    instances := ((instanced.first :: instanced.rest).map fun placed =>
      Testpilot.Authoring.Contract.ruleInstance placed.derived.ruleId
        #[Testpilot.Authoring.Contract.instanceAssignment valueId
          placed.derived.shape.literal.wire]).toArray }

/-- The folded Rule as the Compiler admits it, bound to the checked Property it came from. -/
def InstancedRule.contractLowering {property : CheckedFieldProperty}
    (instanced : InstancedRule property) : Compiler.ContractLowering :=
  .monitor ⟨property.property.id.value, property.property.behaviorFingerprint.render, .property⟩
    instanced.rule

/-- Where one comparison operand comes from at runtime. -/
private inductive Source where
  | literal (mapping : Coverage.InputMapping)
  | captured (path : PropertyFieldPath)
  | observed (path : PropertyFieldPath)

private def Source.of (realization : Realization) : PropertyFieldOperand → Except String Source
  | .literal _ _ => throw "rule.property-literal"
  | .field path _ =>
      if path.capture.isSome then throw "rule.unimplied"
      else match path.root with
        | .request =>
            match realization.literals.find? (·.path == path) with
            | some mapping => pure (.literal mapping)
            | none => throw "rule.literal-unassigned"
        | .priorState => pure (.captured path)
        | .outcome | .event => pure (.observed path)
        | .resultingState => throw "rule.unimplied"

/-- The single protobuf message a declared Observation carries. -/
private def messageRoot (observation : Observation) : Option String := do
  let .singular singular ← (← observation.type).shape | none
  let .message named ← singular.type | none
  pure named.protobuf_type

private def within (clause : CheckedPropertySameStepClause) (result : Except String α) :
    Except Compiler.Error α :=
  result.mapError (Compiler.Error.mk clause.id.value clause.source)

/-- Lower a checked field Property to the monitor rule and request coverage one Case needs. -/
def lower (property : CheckedFieldProperty) (observation : Observation)
    (realization : Realization) : Except Compiler.Error (Lowered property realization) := do
  let checked := property.property
  let rejects := fun (definitionId : DefinitionId) (source : SourceLocation) (construct : String) =>
    Compiler.Error.mk definitionId.value source construct
  if (guardExpressions property).any fun guard => !(fieldPaths guard).isEmpty then
    throw (rejects checked.id checked.source "rule.unimplied")
  let comparisons ← (clauseExpectations property).flatMapM fun clause => do
    let expectation := clause.expectation.expression
    if (fieldPaths expectation).isEmpty then
      pure []
    else
      let some atoms := conjuncts expectation
        | throw (rejects clause.id clause.source "rule.clause-shape")
      atoms.filterMapM fun atom => match atom.fieldComparison with
        | none => throw (rejects clause.id clause.source "rule.clause-shape")
        | some comparison =>
            pure (if establishesPresence comparison then none else some (clause, comparison))
  let (clause, comparison) ← match comparisons with
    | [] => return ⟨none, {}, fun _ member => nomatch member⟩
    | [single] => pure single
    | _ => throw (rejects checked.id checked.source "rule.comparisons")
  let negated ← match comparison.operator with
    | .equal => pure false
    | .notEqual => pure true
    | _ => throw (rejects clause.id clause.source "rule.operator")
  let some root := messageRoot observation
    | throw (rejects clause.id clause.source "rule.observation-type")
  let left ← within clause (Source.of realization comparison.left)
  let right ← within clause (Source.of realization comparison.right)
  let (shape, inputs) ← within clause (match realization.capture, left, right with
    | .none, .literal mapping, .observed path | .none, .observed path, .literal mapping => do
        pure (Shape.safety negated (← Read.of root path) (← Literal.of mapping.path mapping.value), [mapping])
    | .crossEvent selector state response, .captured captured, .observed observed
    | .crossEvent selector state response, .observed observed, .captured captured => do
        if selector.path.capture.isSome || !(selector.path.root == .outcome ||
            selector.path.root == .event) then
          throw "rule.unimplied"
        pure (Shape.capture negated (← Read.of root captured) (← Read.of root observed)
          (← Read.of root selector.path) (← Literal.of selector.path selector.value) state response, [])
    | _, _, _ => throw "rule.unimplied")
  let coverage : Coverage.Request := { inputs }
  -- The derivation above only reads compared or selected coordinates and only realized literals, so
  -- the certificate is decided here rather than trusted; a failure would be a lowering defect.
  if certified : (∀ path ∈ shape.reads,
        path ∈ comparedFields property ∨ path ∈ realization.selectorPaths) ∧
      (∀ value ∈ shape.literals, value ∈ realization.assigned) ∧
      (∀ mapping ∈ coverage.inputs,
        mapping ∈ realization.literals ∧ mapping.path ∈ comparedFields property) then
    pure ⟨some ⟨observation.observation_id, root, shape, certified.1, certified.2.1⟩, coverage,
      certified.2.2⟩
  else throw (rejects clause.id clause.source "rule.certificate")

end Umpire.Case.Projection
