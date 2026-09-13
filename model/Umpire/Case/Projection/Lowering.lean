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
  segments : FieldPath

private def Read.of (root : String) (path : PropertyFieldPath) : Except String Read := do
  let schema := if path.side == .request then path.schema.request else path.schema.response
  let segments ← readPath schema root path.steps
  pure ⟨path, Testpilot.Authoring.Path.make (segments.map fun segment => match segment with
    | .field name => Testpilot.Authoring.Path.field name
    | .oneof group member => Testpilot.Authoring.Path.oneofSelector group member).toArray⟩

/-- One realized literal and the exact wire value it constructs. -/
structure Literal where
  private mk ::
  scalar : Operation.Scalar
  wire : temporal.server.api.testpilot.v1.Value

private def Literal.of (scalar : Operation.Scalar) : Except String Literal := do
  pure ⟨scalar, ← Coverage.scalarValue scalar⟩

/-- The two rule shapes, before rendering. `negated` holds for a `notEqual` comparison. -/
inductive Shape where
  | safety (negated : Bool) (observed : Read) (literal : Literal)
  | capture (negated : Bool) (captured observed selector : Read) (literal : Literal)
      (state response : String)

/-- Every coordinate a shape reads. -/
def Shape.reads : Shape → List PropertyFieldPath
  | .safety _ observed _ => [observed.path]
  | .capture _ captured observed selector _ _ _ => [selector.path, captured.path, observed.path]

/-- Every literal a shape compares against. -/
def Shape.literals : Shape → List Operation.Scalar
  | .safety _ _ literal | .capture _ _ _ _ literal _ _ => [literal.scalar]

private def comparison (negated : Bool) (left right : ContractExpression) :
    ContractExpression :=
  let equal := Testpilot.Authoring.ContractExpr.equals left right
  if negated then Testpilot.Authoring.ContractExpr.negation equal else equal

/-- Render a shape as the one Contract rule it denotes. The rule distinguishes the answers the model
Property does: an event that establishes the observed field and disagrees is a violation, not an
absence, and an event that never establishes it leaves the rule pending, so a Run that produced no
such event still closes inconclusive. A capture rule has no violated state: the event it captured
decides which later event it waits for, and one that never arrives leaves it pending. -/
def Shape.render (shape : Shape) (ruleId suffix observation root : String) :
    ContractRule :=
  let observed := Testpilot.Authoring.ContractExpr.observation observation
  let projected := Testpilot.Authoring.ContractExpr.path
  let present := Testpilot.Authoring.ContractExpr.present
  let all := Testpilot.Authoring.ContractExpr.all
  let completed := #[RunEventKind.RUN_EVENT_KIND_INSTRUCTION_COMPLETED]
  match shape with
  | .safety negated read literal =>
      let value := Testpilot.Authoring.ContractExpr.literal literal.wire
      Testpilot.Authoring.Contract.rule ruleId .CONTRACT_RULE_KIND_SAFETY "pending"
        #[Testpilot.Authoring.Contract.state "pending" .CONTRACT_STATE_STATUS_PENDING,
          Testpilot.Authoring.Contract.state "satisfied" .CONTRACT_STATE_STATUS_SATISFIED,
          Testpilot.Authoring.Contract.state "violated" .CONTRACT_STATE_STATUS_VIOLATED]
        #[Testpilot.Authoring.Contract.transition ("match-" ++ suffix) "pending" "satisfied"
            completed
            (all #[present observed, present (projected observed read.segments),
              comparison negated (projected observed read.segments) value])
            .CONTRACT_SUPPORT_KIND_MATCHING_EVENT,
          Testpilot.Authoring.Contract.transition ("reject-" ++ suffix) "pending" "violated"
            completed
            (all #[present observed, present (projected observed read.segments),
              comparison (!negated) (projected observed read.segments) value])
            .CONTRACT_SUPPORT_KIND_MATCHING_EVENT]
  | .capture negated captured read selector literal state response =>
      let captureId := state ++ "-" ++ suffix
      let retained := Testpilot.Authoring.ContractExpr.capture captureId
      Testpilot.Authoring.Contract.rule ruleId .CONTRACT_RULE_KIND_SAFETY "pending"
        #[Testpilot.Authoring.Contract.state "pending" .CONTRACT_STATE_STATUS_PENDING,
          Testpilot.Authoring.Contract.state state .CONTRACT_STATE_STATUS_PENDING,
          Testpilot.Authoring.Contract.state "satisfied" .CONTRACT_STATE_STATUS_SATISFIED]
        #[Testpilot.Authoring.Contract.transition ("capture-" ++ captureId) "pending" state
            completed
            (all #[present observed, present (projected observed selector.segments),
              comparison false (projected observed selector.segments)
                (Testpilot.Authoring.ContractExpr.literal literal.wire)])
            .CONTRACT_SUPPORT_KIND_MATCHING_EVENT
            #[Testpilot.Authoring.Contract.captureAssignment captureId observation],
          Testpilot.Authoring.Contract.transition ("match-" ++ response ++ "-" ++ suffix) state
            "satisfied" completed
            (all #[present retained, present (projected observed read.segments),
              comparison negated (projected retained captured.segments)
                (projected observed read.segments)])
            .CONTRACT_SUPPORT_KIND_MATCHING_EVENT]
        (captures := #[Testpilot.Authoring.Contract.capture captureId
          (Testpilot.Authoring.Contract.messageCapture root)])

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

/-- The generated Contract rule this derivation denotes. -/
def DerivedRule.rule {property : CheckedFieldProperty} {realization : Realization}
    (derived : DerivedRule property realization) : ContractRule :=
  derived.shape.render (property.property.id.value ++ "." ++ realization.ruleSuffix)
    realization.ruleSuffix derived.observation derived.root

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
private def messageRoot (observation : ObservationDefinition) : Option String := do
  let .singular singular ← (← observation.type).shape | none
  let .message named ← singular.type | none
  pure named.protobuf_type

private def within (clause : CheckedPropertySameStepClause) (result : Except String α) :
    Except Compiler.Error α :=
  result.mapError (Compiler.Error.mk clause.id.value clause.source)

/-- Lower a checked field Property to the monitor rule and request coverage one Case needs. -/
def lower (property : CheckedFieldProperty) (observation : ObservationDefinition)
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
        pure (Shape.safety negated (← Read.of root path) (← Literal.of mapping.value), [mapping])
    | .crossEvent selector state response, .captured captured, .observed observed
    | .crossEvent selector state response, .observed observed, .captured captured => do
        if selector.path.capture.isSome || !(selector.path.root == .outcome ||
            selector.path.root == .event) then
          throw "rule.unimplied"
        pure (Shape.capture negated (← Read.of root captured) (← Read.of root observed)
          (← Read.of root selector.path) (← Literal.of selector.value) state response, [])
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
