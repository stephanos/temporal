import Umpire.Case.Compiler
import Umpire.Case.Correlated
import Umpire.Case.Projection.Coverage
import Umpire.Query
import Umpire.Case.Relation

/-!
# The generic Case Producer

One checked Model, one selected witness, and one named realization become one Case. Everything the
Contract says is derived from the checked values:

* Each `require` clause of the checked Property becomes one operation-correlated bounded-response
  clause. The clause the Model wrote decides the clause the Case carries, so editing a `require`
  line changes the Case bytes rather than any file here.
* The Contract carries no monitor rule at all. Its whole content is the correlated capability that
  `Umpire.Case.Correlated.lower` produced, whose `Lowered` value is the correspondence certificate
  between the checked Model and the wire bytes.
* The evidence those clauses read is lifted out of the recorded history by a declaration on the
  realization's history read: one rule per admitted evidence kind, keyed by whatever path the
  realization says names the operation.

Nothing here names a protocol, a history attribute, or a role. Those live in the realization the
caller supplies, which is why this module holds under SCP-02 while producing Temporal Cases.

What still rejects, and why: a witness the Query did not select (a Case realizes one selected
trace, so a `verify`-form Query has none), an evidence line naming a kind the realization does not
admit, a path ending on a step no evidence line confirms, an evidence line for an Action the
witness never selects, a clause whose shape no correlated predicate can carry, and a requested
clause the lowering did not produce. None of those is waivable by a Known Gap.
-/

namespace Umpire.Case.Producer

open Umpire
open Umpire.Case.Compiler
open temporal.server.api.testpilot.v1 hiding ModelValue

/-! ### The checked authoring bundle

The Producer never sees an authoring surface. A caller converts its own bundle into these
Umpire-owned values at the call site, because a `.umpire` module may not import a feature
namespace. -/

/-- The checked member values of one Model, in declaration order. -/
structure Vocabulary where
  states : List ModelValue
  actions : List ModelValue
  outcomes : List ModelValue
  facts : List ModelValue
  /-- Each state's fields as model values, parallel to `states`. A machine's state is a structure,
  so the Contract compares its fields apart -- `attempts` as a number, `phase` as an enum -- rather
  than reading one spelling back apart. Empty where a state carries no fields. -/
  stateFields : List (List ModelValue) := []
  deriving BEq, Repr

/-- The Model Value an out-of-catalog member resolves to; a declared member never reaches it. -/
def unknownValue : ModelValue := ModelValue.named (DefinitionId.of "") ""

namespace Vocabulary

def stateAt (values : Vocabulary) (index : Nat) : ModelValue :=
  (values.states[index]?).getD unknownValue

def stateFieldsAt (values : Vocabulary) (index : Nat) : List ModelValue :=
  (values.stateFields[index]?).getD []

/-- Each state paired with its own fields, which is what a lowering looks a state up in. -/
def statesWithFields (values : Vocabulary) : List (ModelValue × List ModelValue) :=
  values.states.zipIdx.map fun (value, index) => (value, values.stateFieldsAt index)

def actionAt (values : Vocabulary) (index : Nat) : ModelValue :=
  (values.actions[index]?).getD unknownValue

def outcomeAt (values : Vocabulary) (index : Nat) : ModelValue :=
  (values.outcomes[index]?).getD unknownValue

def factAt (values : Vocabulary) (index : Nat) : ModelValue :=
  (values.facts[index]?).getD unknownValue

/-! Selection by declared spelling. An unknown spelling resolves to the unknown Model Value, whose
Definition ID no Target provides, so the clause referencing it is rejected at admission. -/

def named (values : List ModelValue) (spelling : String) : ModelValue :=
  (values.find? (·.value == spelling)).getD unknownValue

def namedState (values : Vocabulary) (spelling : String) : ModelValue :=
  named values.states spelling
def namedAction (values : Vocabulary) (spelling : String) : ModelValue :=
  named values.actions spelling
def namedOutcome (values : Vocabulary) (spelling : String) : ModelValue :=
  named values.outcomes spelling
def namedFact (values : Vocabulary) (spelling : String) : ModelValue :=
  named values.facts spelling

end Vocabulary

/-- One class an `examples:` line claims, as the Producer looks for it on a path: the Definition ID
of the machine's action member that realizes the class, and the provenance row to record when the
path performs it. A class with no example is no claim, so it never reaches here. -/
structure ClassClaim where
  member : DefinitionId
  row : Provenance.AbstractionClaimRow
  deriving BEq, Repr

/-- Everything the Producer reads out of a checked authoring bundle. `witness` stays optional
because a Query that verifies rather than selects has none, and rejecting that at production is
what keeps the diagnostic on the Case rather than on the Query. -/
structure Input (LawStatement : Law → Prop) where
  target : QueryModel LawStatement
  vocabulary : Vocabulary
  property : CheckedProperty
  scenario : CheckedScenario
  witness : Option Scenario.Trace
  operationRole : DefinitionId
  queryId : DefinitionId
  querySource : SourceLocation
  queryFingerprint : String
  knownGaps : KnownGapSet
  /-- Where a rejection points when the construct it names is the Model's, not the Query's. -/
  source : SourceLocation
  /-- The actions the Program performs, in path order, when they are not the operation's own
  sequence: several instances of one entity interleave their actions on one path, and the Program
  performs all of them while the Contract follows each operation through its own. `none` when the
  operation's sequence is the Program's. -/
  program : Option (List (DefinitionId × Nat)) := none
  /-- How many instances of the entity the path runs over; the actions of `program` name theirs. -/
  instances : Nat := 1
  /-- The machine's setup parameters, each as its name and its Definition ID. A parameter is bound by
  the Profile through the realization's configuration key; one the realization binds to no key is a
  Known Gap of the Case. -/
  setupParameters : List (String × DefinitionId) := []
  /-- The abstraction claims the Model's actions make, by the member that realizes each. The Case
  records the ones its path performs. -/
  claims : List ClassClaim := []
  /-- The field relations the machine's Properties declare. The Case lowers the ones whose action
  its path performs into monitor rules over the fields they compare. -/
  relations : List FieldRelation := []

/-! ### Identity

`fixture` names the checked-in file, and every other identity derives from it. The defaults are the
derivation; a caller that must keep older bytes passes the field it needs explicitly. -/

structure Identity where
  caseId : String
  fixture : String
  programId : String := caseId ++ ".program"
  contractId : String := caseId ++ ".contract"
  /-- The one Run coordinate recorded history does not carry: every event a Case lifts belongs to
  the single Run it executes, so the Case declares that scope rather than reading it. -/
  runScope : String := fixture
  deriving BEq, Repr

/-- The identity a fixture name derives under one project's Case ID root. The root is the caller's,
because what a Case ID is rooted at is a project's convention, not Umpire's. -/
def Identity.ofFixture (root fixture : String) : Identity :=
  { caseId := root ++ "." ++ fixture, fixture }

/-! ### Realization

A realization is the Program a Case runs plus the coordinates a Contract needs to read it back. It
is a value, not syntax: adding a shape is one declaration in the owning feature namespace. -/

/-- A timer of the machine as the realization makes it fire: the duration the realized deadline is
set to. A timer the platform owns, such as a retry backoff, has no binding; the realization reads
its duration and does not set it. -/
structure TimerBinding where
  name : String
  milliseconds : Nat
  deriving BEq, Repr

/-- The recorded data one evidence kind is read from: one arm of the recorded event's attributes
oneof out of a history read, the payload of one Run Event kind the runtime records, or the elements
of a repeated field read back through a unary RPC an instruction polls. -/
inductive RecordedSource where
  | historyEvent (attributesField : String)
  | runEvent (kind : temporal.server.api.testpilot.v1.RunEventKind)
  | read (method path : String)

/-- What a realization knows about one admitted evidence kind. `eventKind` is the spelling an
author writes; everything else is how the realization reads that kind back out of recorded data,
and it is emitted once, as the Program's declaration of the kind. -/
structure EvidenceSource where
  eventKind : String
  recorded : RecordedSource
  operationKeyPath : String
  kindId : DefinitionId
  sourceId : DefinitionId
  /-- The fields the kind exposes, each with the path read out of the recorded value. -/
  fields : List (String × String) := []
  /-- The recorded field that names which instance's event this is, resolved against the kind's
  schema: a relation that captures an event of this kind selects the instance's own by this field
  read against the literal the confirming action's binding assigns under the same spelling. A kind
  with no selector cannot be captured. -/
  selector : Option FieldOperand := none

/-- One resolved (Action, source) pair: the Action a recorded event confirms, and how to read it. -/
structure EvidenceRule where
  action : ModelValue
  source : EvidenceSource

/-- One `evidence` line: the Action the Scenario selects, and the event kind that confirms it. -/
structure EvidenceMapping where
  action : ModelValue
  eventKind : String
  deriving BEq, Repr

/-! #### The Program as a plan

A realization no longer writes a Program. It declares the scaffolding every Case of the feature
carries and binds each action class to the instruction that performs it; the Producer walks a Query's
path and puts those instructions in the order the path took them.

That split is what keeps this module free of the feature: the Producer decides *where* an
instruction goes from the path alone, and the realization decides *what* it is. An entrypoint whose
sequence interleaves scaffolding and actions -- a controller that starts a workflow, waits, performs
an action, then reads history -- says so by ordering its items, so the assembly reproduces that
sequence without knowing what any of them mean. -/

/-- Where a node lands on a path over several instances of the entity: the Case it belongs to, the
instance that performs it, numbered from one the way a Scenario numbers them, and how many there
are. A Case over one instance is placement 1 of 1, and every id it derives is unchanged. -/
structure Placement where
  identity : Identity
  /-- The instance that performs the node, numbered from one. -/
  number : Nat := 1
  /-- How many instances the path runs over. -/
  count : Nat := 1
  deriving BEq, Repr

/-- The suffix an instance's own ids carry: none on a Case over one instance, `-<n>` on one over
several, so the two instances' nodes, slots and entrypoints never collide. -/
def Placement.suffix (placement : Placement) : String :=
  if placement.count ≤ 1 then "" else "-" ++ toString placement.number

/-- One item in an entrypoint's instruction sequence: a node the realization always emits, a node it
emits for each instance whose path performs one of the named classes, or the place where the path's
actions of the named classes land. -/
inductive EntrypointItem where
  /-- A node every Case carries once, built from the placement and the evidence rules it must lift.
  On an entrypoint emitted per instance, once per copy. -/
  | fixed (node : Placement → List EvidenceRule → InstructionNode)
  /-- A node a Case carries once for each instance whose path performs a class one of `keys` names:
  scaffolding one side effect needs, such as the wait for a handle another party publishes. -/
  | whenOnPath (keys : List String) (node : Placement → List EvidenceRule → InstructionNode)
  /-- The actions of these classes, in the order the path performs them. -/
  | actions (classes : List DefinitionId)
  /-- These items once per instance, in instance order, each copy carrying that instance's
  actions and the nodes it performs: the wait for an authority and the completion that consumes
  it, kept together per instance so each completion follows its own wait. -/
  | perInstance (items : List EntrypointItem)

/-- One entrypoint of a realization's Program: how it activates, and its items in order.

`activate` is the matching `Program` constructor with everything but its instructions applied, so an
entrypoint's activation stays the authoring surface's business. It takes the placement because a
workflow's type is derived from the Case's fixture name, and an entrypoint emitted per instance is
named after the instance. `perInstance` emits the entrypoint once per instance of the entity, each
copy carrying that instance's actions alone: a handler that answers one instance's operation. -/
structure EntrypointPlan where
  activate : Placement → Array InstructionNode → Entrypoint
  items : List EntrypointItem
  perInstance : Bool := false

/-- Everything a realization's Program carries besides its actions: the roles it declares, the
handle slots its instructions publish and consume -- the ones every Case declares, and the ones
declared once per instance -- the observations its reads write into, its entrypoints in order, and
its cleanup. -/
structure ProgramPlan where
  roles : Array Role
  slots : Array Slot := #[]
  instanceSlots : Placement → Array Slot := fun _ => #[]
  observations : Array Observation
  entrypoints : List EntrypointPlan
  cleanup : Cleanup

/-- What one action class is realized as. `node` builds the instruction the party performs; the
Producer supplies the node's id, so a class performed twice on one path cannot collide.

A binding names its class by `key`, the member key a Scenario's path spells it by
(`handlerReply-async`, `schedule-unset-unset-unset`), and the Producer resolves the key against the
Model's vocabulary, so one realization serves every Model whose actions carry those classes.
`action` is the Definition ID a realization states for a Model whose own actions realize nothing
and whose path the realization states instead; it is read only where the key resolves to no
member. -/
structure ActionBinding where
  action : DefinitionId
  instructionId : String
  node : Placement → String → InstructionNode
  key : String := ""
  /-- The literals the node assigns to fields of the action's own schema, by the dotted path the
  field is written at (`workflow_type.name`). A field relation over an input field reads the
  literal here, so the rule the Contract carries compares the value the Program constructs; a
  relation that captures an earlier event reads here the literal that selects the instance's own
  event, under the spelling of the source's selector. -/
  literals : Placement → List (String × Operation.Scalar) := fun _ => []

/-- The Definition ID a binding's class has in this Model: the vocabulary's action member of the
binding's key, or the id the binding states where the vocabulary has no such member. -/
def ActionBinding.resolve (binding : ActionBinding) (vocabulary : Option Vocabulary) :
    DefinitionId :=
  match vocabulary with
  | some values =>
      if binding.key.isEmpty then binding.action
      else
        let named := values.namedAction binding.key
        if named.definitionId.value.isEmpty then binding.action else named.definitionId
  | none => binding.action

/-- One setup parameter of a machine as the realization binds it: the parameter's Definition ID and
the configuration key the Profile sets it under. The value a Case's path assumed is not written
here: the Case bytes do not depend on it, and the Profile records the value it ran under. -/
structure SetupBinding where
  parameter : DefinitionId
  key : String
  deriving BEq, Repr

/-- One value of a switch: its name, and the configuration -- each key with the value it takes --
an environment running under it sets. -/
structure SwitchValue where
  name : String
  configuration : List (String × String) := []
  deriving BEq, Repr, Inhabited

/-- A switch as the realization declares it: a rollout flag between two implementations of one
behavior, with the configuration each value sets. It is not a Model parameter; a functional set's
`repeat` names it and each Query's Case runs once per value. -/
structure SwitchBinding where
  name : String
  values : List SwitchValue
  deriving BEq, Repr, Inhabited

structure Realization where
  /-- The scaffolding every Case of this feature carries, with the places its actions land. -/
  plan : ProgramPlan
  /-- What each action class is realized as. An action on a path with no binding rejects. -/
  actions : List ActionBinding := []
  /-- The configuration key each setup parameter of the machine is bound to. A parameter with no
  binding is one the Profile cannot set, so the Case carries a Known Gap naming it. -/
  setup : List SetupBinding := []
  /-- The switches this realization declares. A `repeat:` names one of these or rejects. -/
  switches : List SwitchBinding := []
  producerId : String
  producerVersion : String := "1"
  projectionId : DefinitionId
  /-- The Run coordinate the projection scopes evidence by. -/
  scopeField : DefinitionId
  /-- The coordinate that names one operation across every admitted evidence kind. -/
  operationKey : DefinitionId
  historyObservation : String
  correlatedObservation : String
  /-- The durations the machine's timers realize as. A timer with no binding is one the platform
  owns. -/
  timers : List TimerBinding := []
  sources : List EvidenceSource
  projectionLimits : Case.Projection.Limits
  /-- Evaluation ceilings for the correlated consumer, separate from the semantic window. -/
  runLimits : Property.Correlated.Limits

/-- The switch a `repeat:` names, or `none` for one this realization does not declare, which is
what rejects the `set`. -/
def Realization.switch? (realization : Realization) (name : String) : Option SwitchBinding :=
  realization.switches.find? (·.name == name)

/-- The duration a timer realizes as, or `none` for a timer the platform owns. -/
def Realization.timer? (realization : Realization) (name : String) : Option TimerBinding :=
  realization.timers.find? (·.name == name)

/-- The Known Gaps a Case carries for the setup parameters the realization binds to no
configuration key. The Profile sets a parameter through its key, so an unbound one runs under
whatever value the environment happens to have, and the Case says so rather than claim otherwise.
The gap is an `input` gap coded after the parameter, with the parameter as its subject. -/
def unboundSetupGaps
    (realization : Realization)
    (setupParameters : List (String × DefinitionId)) : List KnownGap :=
  setupParameters.filterMap fun (name, parameter) =>
    if realization.setup.any (·.parameter == parameter) then none
    else some {
      kind := .input
      code := DefinitionId.of (parameter.value ++ ".unbound")
      subject := some parameter
      detail := some s!"the setup parameter '{name}' is bound to no configuration key of the \
realization, so the Profile cannot set it and the Case runs under the environment's own value" }

/-! ### The derived correlated Property

A `require` clause says what must hold at the step that selects one Action. The checked Scenario
says where that Action sits in the operation's own sequence. Together they are a bounded response:
from the operation's first selected Action, the required value is due within exactly as many
semantic transitions as the Scenario places between them.

That is what makes the derived Contract discriminating rather than vacuous. A same-step clause
triggered on its own Action would answer satisfied for an operation that never reached the Action
at all, because nothing triggered; triggering on the operation's first Action instead leaves the
obligation open until the operation either reaches the required value or the window closes. -/

private def productionError (source : SourceLocation) (definitionId construct : String) : Error := {
  sourceDefinitionId := definitionId
  source
  construct
}

/-- The predicate field one modeled trace field names in a same-step predicate environment. A trace
field with no same-step predicate is a clause this lowering cannot express. -/
private def predicateField : PropertyTraceField → Option PropertyPredicateField
  | .priorState => some .priorState
  | .selectedAction => some .selectedAction
  | .resultingState => some .resultingState
  | .outcome => some .outcome
  | .observation => some .expectationFact
  | .state | .relation => none

private def predicateOf (pattern : PropertyPattern) : Option PropertyPredicate := do
  let field ← predicateField pattern.field
  let constraint ← match pattern.constraint with
    | .present => some PropertyAtomConstraint.present
    | .equals value => some (.equals (.text value))
    | _ => none
  pure (.atom { field, reference := pattern.reference, constraint })

/-- Whether one modeled step already carries a pattern's value. -/
private def patternHolds
    (pattern : PropertyPattern)
    (step : ModelTraceStep ModelValue ModelValue ModelValue ModelValue) : Bool :=
  let carries := fun (value : ModelValue) =>
    pattern.reference == value.definitionId &&
      match pattern.constraint with
      | .present => true
      | .equals expected => expected == value.value
      | _ => false
  match pattern.field with
  | .selectedAction => carries step.selectedAction
  | .resultingState => carries step.state
  | .outcome => carries step.outcome
  | .observation => step.facts.any carries
  | _ => false

/-- One `require` clause as an operation-correlated clause, placed by the checked Scenario. A
same-step clause is the only form with a trigger and a response to carry across; a value constraint
the portable predicate vocabulary has no spelling for, a trigger that is not an Action, and an
Action the Scenario never selects each reject by clause name rather than being narrowed or guessed.

The window is inclusive of its trigger step, so a required value that the selected trace already
carries somewhere before the Action the clause names would answer the clause without that Action
ever being observed. That is the vacuity this trigger choice exists to avoid, so a Property whose
response holds earlier rejects rather than lowering a clause a shorter trace could satisfy. -/
private def scopedClauseOf
    (source : SourceLocation)
    (scopeField operationKey : DefinitionId)
    (occurrences : List DefinitionId) (opening : ModelValue)
    (steps : List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue))
    (clause : CheckedPropertyClause) : Except Error PropertyCorrelatedClause :=
  let unexpressible := fun construct =>
    Except.error (productionError source clause.id.value construct)
  match clause with
  | .transitionContract id trigger response
  | .inputOutput id trigger response =>
      if trigger.field != .selectedAction then
        unexpressible "property.clause-shape"
      else
        match occurrences.idxOf? trigger.reference, predicateOf response with
        | some bound, some lowered =>
            if (steps.take bound).any (patternHolds response) then
              unexpressible "property.clause-early-response"
            else
              .ok {
                id, source
                trigger := .selectedActionIs opening, response := lowered
                scope := [scopeField], key := operationKey
                bound, ending := .«partial» }
        | none, _ => unexpressible "property.clause-occurrence"
        | _, none => unexpressible "property.clause-shape"
  | _ => unexpressible "property.clause-form"

/-! ### The declared projection

Which Step a recorded event confirms is read from the checked Machine along the witness trace, so
an author states the mapping once, in the `steps` block, rather than repeating the state, outcome
and facts beside every event kind. -/

private def stepOf
    (steps : List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue))
    (action : ModelValue) : Option (Step ModelValue ModelValue ModelValue) :=
  (steps.find? fun step => step.selectedAction == action).map fun step =>
    { «state» := step.state, «outcome» := step.outcome, «facts» := step.facts }

/-- One resolved evidence rule and the steps its evidence confirms: the silent steps the witness
took before the rule's own, then the rule's own. -/
abbrev ResolvedRule := EvidenceRule × List (ModelValue × Step ModelValue ModelValue ModelValue)

private def projectionDeclaration
    (realization : Realization)
    (rules : List ResolvedRule) :
    Case.Projection.Declaration ModelValue ModelValue ModelValue ModelValue := {
  id := realization.projectionId
  scopeFields := [realization.scopeField]
  operationField := realization.operationKey
  sources := (rules.map (·.1.source.sourceId)).eraseDups
  rules := rules.map fun entry =>
    { kind := entry.1.source.kindId
      meaning := .confirmed none entry.2 }
  «limits» := realization.projectionLimits }

/-! ### Evidence resolution

An `evidence` line names an event kind; the realization says which kinds it admits. A kind outside
that list, a line for an Action the witness never selects, and a mapped Action the witness performs
twice each reject by name.

A Model whose machine carries `evidence:` lines needs no `evidence` lines of its own: each fact a
witness step records is confirmed by the observation the machine's line maps it to, so the mapping
is read off the witness. A classed fact is covered by the line naming its constructor.

A step that records nothing an evidence line names is silent: an `unobservable:` timer, or an
action whose step keeps the operation where it was. The Contract cannot see it fire, so the rule of
the next observed step confirms the silent steps before it together with its own -- the machine has
no other way from the state before them to the state the evidence shows -- and the Case carries a
Known Gap naming each, because the step is inferred rather than observed. A silent step the path
ends on is confirmed by nothing and rejects. -/

/-- Whether an `evidence:` line's fact spelling covers a recorded fact: the member itself, or the
constructor whose members the fact is one of. -/
private def coversFact (spelling factValue : String) : Bool :=
  factValue == spelling || factValue.startsWith (spelling ++ "-")

/-- The evidence mappings a witness implies under the machine's own `evidence:` lines: one per fact
a step records that some line covers, naming the step's Action. -/
def derivedEvidence
    (catalog : List (String × String))
    (steps : List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue)) :
    List EvidenceMapping :=
  (steps.flatMap fun step => step.facts.filterMap fun fact =>
    (catalog.find? fun entry => coversFact entry.1 fact.value).map fun entry =>
      ({ action := step.selectedAction, eventKind := entry.2 } : EvidenceMapping)).eraseDups

/-- The Known Gap a silent step on the path records: the Contract infers the step from the evidence
of the step after it. -/
private def silentGap (action : ModelValue) : KnownGap := {
  kind := .capability
  code := DefinitionId.of (action.definitionId.value ++ ".unobserved")
  subject := some action.definitionId
  detail := some s!"the step '{action.value}' records nothing an evidence line names, so the \
Contract infers it from the evidence of the step after it rather than observing it" }

private def resolveEvidence
    (source : SourceLocation)
    (realization : Realization)
    (selected : List ModelValue)
    (steps : List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue))
    (evidence : List EvidenceMapping) :
    Except Error (List ResolvedRule × List KnownGap) := do
  let admitted ← evidence.mapM fun mapping => do
    let admitted ← match realization.sources.find? (·.eventKind == mapping.eventKind) with
      | some admitted => pure admitted
      | none => throw (productionError source mapping.eventKind "evidence.kind-unknown")
    unless selected.any (· == mapping.action) do
      throw (productionError source mapping.action.definitionId.value
        "evidence.action-unselected")
    unless steps.any (·.selectedAction == mapping.action) do
      throw (productionError source mapping.action.definitionId.value
        "evidence.action-unwitnessed")
    pure (mapping.action, admitted)
  let mut resolved : List ResolvedRule := []
  let mut silent : List (ModelValue × Step ModelValue ModelValue ModelValue) := []
  let mut gaps : List KnownGap := []
  for step in steps do
    let taken : Step ModelValue ModelValue ModelValue :=
      { «state» := step.state, «outcome» := step.outcome, «facts» := step.facts }
    match admitted.find? (·.1 == step.selectedAction) with
    | some (action, admitted) =>
        if resolved.any (·.1.action == action) then
          throw (productionError source action.definitionId.value "evidence.action-repeated")
        resolved := resolved ++ [({ action, source := admitted }, silent ++ [(action, taken)])]
        silent := []
    | none =>
        silent := silent ++ [(step.selectedAction, taken)]
        unless gaps.any (·.subject == some step.selectedAction.definitionId) do
          gaps := gaps ++ [silentGap step.selectedAction]
  if let some (action, _) := silent.head? then
    throw (productionError source action.definitionId.value "evidence.action-unmapped")
  pure (resolved, gaps)

/-! ### Program assembly

The path decides the order. Each action the path performs becomes the instruction its class binds,
appended where its entrypoint's `actions` item sits; every other node is the realization's and is
emitted as written. An action the path performs that no class binds rejects, because a Case that
silently dropped a side effect would still run.
-/

/-- The node one occurrence of an action class contributes. The first occurrence carries the
binding's own instruction id and each later one appends its 1-based ordinal, so a path that performs
a class twice produces two distinct nodes rather than a duplicate id preparation would reject; on a
path over several instances the id carries the instance first, and the ordinal counts that
instance's own occurrences. -/
private def boundNode
    (placement : Placement)
    (binding : ActionBinding)
    (ordinal : Nat) : InstructionNode :=
  let instructionId := binding.instructionId ++ placement.suffix ++
    (if ordinal == 0 then "" else "-" ++ toString (ordinal + 1))
  binding.node placement instructionId

/-- The placement of instance `instance` of `instances` under one identity. -/
private def placementOf (identity : Identity) (instances slot : Nat) : Placement :=
  { identity, number := slot, count := instances }

/-- The nodes the path contributes to one `actions` item: every occurrence whose class the item
names and whose instance `admitted` admits, in path order, each numbered by how many times its own
instance has performed its class. -/
private def actionNodes
    (source : SourceLocation)
    (identity : Identity)
    (instances : Nat)
    (bindings : List ActionBinding)
    (path : List (DefinitionId × Nat))
    (classes : List DefinitionId)
    (admitted : Nat → Bool := fun _ => true) : Except Error (Array InstructionNode) := do
  let mut nodes : Array InstructionNode := #[]
  let mut seen : List (DefinitionId × Nat) := []
  for (action, slot) in path do
    let ordinal := seen.count (action, slot)
    seen := seen ++ [(action, slot)]
    unless classes.contains action && admitted slot do
      continue
    match bindings.find? fun binding => binding.action == action with
    | some binding =>
        nodes := nodes.push (boundNode (placementOf identity instances slot) binding ordinal)
    | none => throw (productionError source action.value "realization.action-unbound")
  pure nodes

/-- The Program's evidence declarations: one per admitted kind the resolved rules read, in the
order the rules first name them, each scoped to this Case's Run. -/
private def evidenceDeclarations
    (scopeField : DefinitionId)
    (identity : Identity)
    (resolved : List EvidenceRule) :
    Array temporal.server.api.testpilot.v1.EvidenceDeclaration :=
  let sources := resolved.foldl (fun (admitted : List EvidenceSource) rule =>
    if admitted.any (·.kindId == rule.source.kindId) then admitted
    else admitted ++ [rule.source]) []
  sources.toArray.map fun admitted =>
    let scope := #[Testpilot.Authoring.Program.evidenceScope scopeField.value identity.runScope]
    let fields := admitted.fields.toArray.map fun (fieldId, path) =>
      Testpilot.Authoring.Program.evidenceField fieldId path
    match admitted.recorded with
    | .historyEvent attributesField =>
        Testpilot.Authoring.Program.historyEvidenceDeclaration admitted.kindId.value
          admitted.sourceId.value attributesField admitted.operationKeyPath scope fields
    | .runEvent kind =>
        Testpilot.Authoring.Program.runEventEvidenceDeclaration admitted.kindId.value
          admitted.sourceId.value kind admitted.operationKeyPath scope fields
    | .read method readPath =>
        Testpilot.Authoring.Program.readEvidenceDeclaration admitted.kindId.value
          admitted.sourceId.value method readPath admitted.operationKeyPath scope fields

/-- Assemble one Program from the realization's plan and the path's actions.

An action the realization binds must also be placed exactly once: a binding no entrypoint's `actions`
item names would leave a side effect out of the Program, and one two items name would emit its nodes
twice under the same ids, so the walk records what it placed and rejects either by name. An action with no binding is not an omission -- a party bound `observed` performs
nothing, and a Model whose actions are waits rather than side effects realizes none of them -- so the
Contract carries it and the Program does not. -/
def assembleProgram
    (source : SourceLocation)
    (identity : Identity)
    (resolved : List EvidenceRule)
    (realization : Realization)
    (path : List (DefinitionId × Nat))
    (vocabulary : Option Vocabulary := none)
    (instances : Nat := 1) : Except Error Program := do
  -- Each binding's class as this Model names it, so the path's ids and the bindings' agree.
  let bindings := realization.actions.map fun binding =>
    { binding with action := binding.resolve vocabulary }
  let instanceNumbers := (List.range instances).map (· + 1)
  -- The instances whose own path performs a class one of the keys names, in instance order.
  let performing (keys : List String) : List Nat :=
    instanceNumbers.filter fun slot => bindings.any fun binding =>
      keys.contains binding.key && path.contains (binding.action, slot)
  -- The classes an `actions` item names, read through the bindings: an item names classes by the
  -- ids the realization states; the ones its bindings resolved carry the Model's ids instead.
  let resolvedClasses (classes : List DefinitionId) : List DefinitionId :=
    classes.map fun stated =>
      (((realization.actions.zip bindings).find? (·.1.action == stated)).map
        (·.2.action)).getD stated
  -- The nodes one item contributes under one placement, admitting the instances `admitted`
  -- names, and the classes it placed. A `perInstance` group is walked by `emit` below, once per
  -- instance; a group inside a group is not a shape a realization writes.
  let emitItem (item : EntrypointItem) (placement : Placement) (admitted : Nat → Bool)
      (placed : List DefinitionId) :
      Except Error (Array InstructionNode × List DefinitionId) := do
    match item with
    | .fixed node => pure (#[node placement resolved], placed)
    | .whenOnPath keys node =>
        pure (((performing keys).filter admitted).toArray.map fun slot =>
          node (placementOf identity instances slot) resolved, placed)
    | .actions classes =>
        let classes := resolvedClasses classes
        for action in classes do
          if placed.contains action then
            throw (productionError source action.value "realization.action-placed-twice")
        pure (← actionNodes source identity instances bindings path classes admitted,
          placed ++ classes)
    | .perInstance _ => throw (productionError source "" "realization.item-nested")
  let emit (items : List EntrypointItem) (placement : Placement) (admitted : Nat → Bool)
      (placed : List DefinitionId) :
      Except Error (Array InstructionNode × List DefinitionId) := do
    let mut nodes : Array InstructionNode := #[]
    let mut placedHere := placed
    for item in items do
      match item with
      | .perInstance inner =>
          -- The inner items place their classes once for every instance, each copy checked
          -- against what was placed before the group; the classes are recorded once.
          let before := placedHere
          for slot in instanceNumbers.filter admitted do
            let mut placedInner := before
            for entry in inner do
              let (emitted, placedNext) ← emitItem entry (placementOf identity instances slot)
                (· == slot) placedInner
              nodes := nodes ++ emitted
              placedInner := placedNext
            placedHere := (placedHere ++ placedInner).eraseDups
      | entry =>
          let (emitted, placedNext) ← emitItem entry placement admitted placedHere
          nodes := nodes ++ emitted
          placedHere := placedNext
    pure (nodes, placedHere)
  let mut entrypoints : Array Entrypoint := #[]
  let mut placed : List DefinitionId := []
  for plan in realization.plan.entrypoints do
    -- An entrypoint emitted per instance is one copy per instance, each carrying that instance's
    -- actions; every other entrypoint is one copy carrying every instance's.
    let copies : List (Placement × (Nat → Bool)) :=
      if plan.perInstance then
        instanceNumbers.map fun slot => (placementOf identity instances slot, (· == slot))
      else [(placementOf identity instances 1, fun _ => true)]
    let mut placedHere : List DefinitionId := []
    for (placement, admitted) in copies do
      let (nodes, placedCopy) ← emit plan.items placement admitted placed
      placedHere := placedHere ++ placedCopy
      entrypoints := entrypoints.push (plan.activate placement nodes)
    placed := placedHere.eraseDups
  for binding in bindings do
    if path.any (·.1 == binding.action) && !placed.contains binding.action then
      throw (productionError source binding.action.value "realization.action-unplaced")
  let slots := realization.plan.slots ++ (instanceNumbers.toArray.flatMap fun slot =>
    realization.plan.instanceSlots (placementOf identity instances slot))
  pure (Testpilot.Authoring.Program.make identity.programId realization.plan.roles
    slots realization.plan.observations entrypoints realization.plan.cleanup
    (evidenceDeclarations realization.scopeField identity resolved))

/-- The Program one realization assembles for a path, for a caller that wants the Program alone:
`umpire-inspect` and the template tests read the shape a realization produces without producing a
Case. `produce` calls the same assembly, so what this returns is what a Case carries.

The default empty path emits no action node, which is what a realization that binds none produces
anyway; pass the path whenever the realization has bindings. -/
def Realization.program
    (realization : Realization)
    (identity : Identity)
    (resolved : List EvidenceRule := [])
    (path : List DefinitionId := [])
    (source : SourceLocation := { path := "" }) : Except Error Program :=
  assembleProgram source identity resolved realization (path.map (·, 1)) none

/-! ### Field relations

A relation whose action the path performs becomes one monitor rule: the declaration it denotes is
admitted against the Model with the operands' schemas as field bindings, and
`Umpire.Case.Projection.lower` derives the rule from it, reading the observation the realization
records history into and the literal the action's binding assigns to the input field. -/

/-- One relation lowered for a Case: the checked Property's definition binding, the monitor
lowering and the request literals the rule's coverage requires. -/
structure LoweredRelation where
  definition : Provenance.DefinitionBinding
  lowering : Compiler.ContractLowering
  inputs : List Coverage.InputMapping
  source : SourceLocation

private def operandReference (vocabulary : Vocabulary) (action : ModelValue)
    (operand : FieldOperand) : DefinitionId :=
  match operand.root with
  | .request => action.definitionId
  | .event => (vocabulary.namedFact operand.member).definitionId
  | .outcome => (vocabulary.namedOutcome operand.member).definitionId
  | .priorState | .resultingState => (vocabulary.namedState operand.member).definitionId

/-- The entrypoint a stated action class is placed on, by its plan's activation. -/
private def entrypointOf (placement : Placement) (realization : Realization)
    (stated : DefinitionId) : Option String :=
  (realization.plan.entrypoints.find? fun plan => plan.items.any fun item =>
    match item with
    | .actions classes => classes.contains stated
    | _ => false).map fun plan => (plan.activate placement #[]).entrypoint_id

/-- The selector a captured operand's rule retains the earlier event by: the field the source of
the captured kind names, read under the operand's own reference, against the literal the binding of
the action that kind confirms assigns for this instance under the selector's spelling. -/
private def captureSelector {LawStatement : Law → Prop}
    (input : Input LawStatement)
    (placement : Placement)
    (realization : Realization)
    (rules : List EvidenceRule)
    (relationId : String)
    (captured : FieldOperand)
    (reference : DefinitionId) : Except Error Case.Projection.Selector := do
  let rejects := productionError input.source relationId
  let some rule := rules.find? (·.source.eventKind == captured.observed)
    | throw (rejects "relation.capture-unrecorded")
  let some selector := rule.source.selector
    | throw (rejects "relation.capture-unselectable")
  let some binding := realization.actions.find? fun binding =>
      binding.resolve (some input.vocabulary) == rule.action.definitionId
    | throw (rejects "relation.capture-unbound")
  let some (_, value) := (binding.literals placement).find? (·.1 == selector.spelling)
    | throw (rejects "relation.capture-literal-unassigned")
  pure { path := { selector with member := captured.member }.path reference, value }

private def lowerRelation {LawStatement : Law → Prop}
    (input : Input LawStatement)
    (placement : Placement)
    (realization : Realization)
    (rules : List EvidenceRule)
    (relation : FieldRelation) : Except Error LoweredRelation := do
  let rejects := productionError input.source relation.id.value
  let action := input.vocabulary.namedAction relation.action
  let leftReference := operandReference input.vocabulary action relation.left
  let rightReference := match relation.right with
    | some right => operandReference input.vocabulary action right
    | none => leftReference
  let checked ← (relation.check (.ofTarget input.target) input.property.requires leftReference
    rightReference).mapError rejects
  let some observation := realization.plan.observations.find?
      (·.observation_id == realization.historyObservation)
    | throw (rejects "relation.observation-undeclared")
  let operands := [(relation.left, leftReference)] ++
    (relation.right.map fun right => (right, rightReference)).toList
  -- The literal the Program assigns to each input operand: the action's binding states it by the
  -- field's dotted path, and the node it builds is where the rule's coverage looks for it.
  let requestOperands := operands.filter (·.1.root == .request)
  let literals ← requestOperands.mapM fun (operand, reference) => do
    let some (stated, binding) := (realization.actions.map fun binding =>
        (binding.action, { binding with action := binding.resolve (some input.vocabulary) })).find?
        (·.2.action == action.definitionId)
      | throw (rejects "relation.action-unbound")
    let some (_, value) := (binding.literals placement).find? (·.1 == operand.spelling)
      | throw (rejects "relation.literal-unassigned")
    let some entrypointId := entrypointOf placement realization stated
      | throw (rejects "relation.action-unplaced")
    pure ({ path := operand.path reference
            value
            entrypointId
            instructionId := binding.instructionId ++ placement.suffix } : Coverage.InputMapping)
  -- An operand read at the state before the step is an earlier event the rule captures: the
  -- selector names the instance's own event, and the states the rule passes through are named
  -- after the kinds it reads.
  let capture ← match operands.filter (·.1.root == .priorState),
      operands.find? (·.1.root != .priorState) with
    | [], _ => pure Case.Projection.CapturePolicy.none
    | [(captured, reference)], some (observed, _) => do
        let selector ← captureSelector input placement realization rules relation.id.value
          captured reference
        pure (.crossEvent selector captured.observed observed.observed)
    | _, _ => throw (rejects "relation.capture-shape")
  let lowered ← Case.Projection.lower checked observation
    { literals, ruleSuffix := FieldRelation.ruleSuffix ++ placement.suffix, capture }
  let some lowering := lowered.contractLowering
    | throw (rejects "relation.no-rule")
  pure {
    definition := {
      definitionId := checked.property.id.value
      behaviorFingerprint := checked.property.behaviorFingerprint.render
      kind := .«property» }
    lowering
    inputs := lowered.coverage.inputs
    source := relation.source }

/-! ### Production -/

/-- Lower one checked Model into a Case through a named realization. The checked values are
carried, never compared against an expected Model: a different Machine, Scenario, Query or Property
produces different Case bytes. Only a claim this Producer cannot realize rejects.

`required` names clauses the caller requires the Case to carry, beyond the ones the checked
Property already names. Coverage is always requested explicitly, never left to a default. -/
def produce {LawStatement : Law → Prop}
    (input : Input LawStatement)
    (identity : Identity)
    (realization : Realization)
    (evidence : List EvidenceMapping)
    (required : List DefinitionId := [])
    (evidenceCatalog : List (String × String) := []) :
    Except Error temporal.server.api.testpilot.v1.Case := do
  let rejects := productionError input.source
  -- A Case realizes one selected trace, so a Query that verifies rather than selects has no
  -- witness to realize and rejects here. A Known Gap does not admit it.
  let selected ← match input.witness with
    | some selected => pure selected
    | none => throw (rejects input.queryId.value "witness.absent")
  -- The operation's own sequence in trace order: what it does first, and where each later Action
  -- sits after it. Only an `exactly` Scenario fixes that order, and the derivation needs it, so a
  -- Scenario that only bounds occurrences rejects rather than being read in canonical key order.
  let occurrences ← match input.scenario.actionsExactly with
    | some occurrences => pure occurrences
    | none => throw (rejects input.scenario.id.value "behavior.sequence.absent")
  let selectedValues := occurrences.filterMap fun occurrence =>
    input.vocabulary.actions.find? fun value => value.definitionId == occurrence
  let opening ← match occurrences.head? with
    | some first =>
        match input.vocabulary.actions.find? fun value => value.definitionId == first with
        | some opening => pure opening
        | none => throw (rejects first.value "behavior.action.undeclared")
    | none => throw (rejects input.scenario.id.value "behavior.sequence.absent")
  -- Evidence lines written at the Case, or, where it wrote none, the ones the machine's own
  -- `evidence:` lines imply along the witness.
  let evidence := if evidence.isEmpty then derivedEvidence evidenceCatalog selected.trace.steps
    else evidence
  let (evidenceRules, silentGaps) ← resolveEvidence input.source realization selectedValues
    selected.trace.steps evidence
  let correlatedRules ← input.property.clauses.mapM
    (scopedClauseOf input.source realization.scopeField realization.operationKey
      occurrences opening selected.trace.steps)
  if correlatedRules.isEmpty then
    throw (rejects input.property.id.value "property.clauses.absent")
  let correlatedProperty ← (Property.check (.ofTarget input.target) ({
      id := input.property.id
      source := input.property.source
      version := input.property.version
      requires := input.property.requires
      clauses := []
      correlatedRules })).mapError fun _ =>
    rejects input.property.id.value "property.correlated-admission"
  let compiled ← (Property.Correlated.compile input.target correlatedProperty
    [realization.scopeField] realization.operationKey realization.runLimits).mapError fun _ =>
    rejects input.property.id.value "property.correlated-compile"
  let setup : List RoleBinding :=
    [{ «role» := input.operationRole, value := input.vocabulary.stateAt 0 }]
  let plan ← (Case.Projection.check input.target
    (projectionDeclaration realization evidenceRules) setup
    (input.vocabulary.stateAt 0)).mapError fun _ =>
    rejects realization.projectionId.value "projection.admission"
  let lowered ← Umpire.Case.Correlated.lower plan compiled realization.correlatedObservation
    (Case.Projection.Coverage.empty plan) input.vocabulary.statesWithFields
  -- The Query's own Known Gaps, one per setup parameter the realization leaves unbound, and one
  -- per silent step the path takes.
  let knownGaps ← (KnownGapSet.ofUnordered
      (input.knownGaps.toList ++ unboundSetupGaps realization input.setupParameters ++
        silentGaps)).mapError
    fun _ => rejects input.queryId.value "known-gaps.setup"
  -- The relations whose action the path performs, each one monitor rule over the fields it
  -- compares.
  let program := input.program.getD (occurrences.map (·, 1))
  let performed := program.map (·.1)
  -- One rule per instance: the literal a rule compares against and the event it captures are the
  -- instance's own.
  let placements := (List.range input.instances).map fun slot =>
    ({ identity, number := slot + 1, count := input.instances } : Placement)
  let relations ← (input.relations.filter fun relation =>
    performed.contains (input.vocabulary.namedAction relation.action).definitionId).flatMapM
    fun relation => placements.mapM fun placement =>
      lowerRelation input placement realization (evidenceRules.map (·.1)) relation
  compile {
    version := { major := 1 }
    caseId := identity.caseId
    producerId := realization.producerId
    producerVersion := realization.producerVersion
    definitions := [
      { definitionId := input.target.id.value
        behaviorFingerprint := input.target.behaviorFingerprint.render, kind := .target },
      { definitionId := input.scenario.id.value
        behaviorFingerprint := input.scenario.behaviorFingerprint.render, kind := .«scenario» },
      { definitionId := input.queryId.value
        behaviorFingerprint := input.queryFingerprint, kind := .«query» },
      { definitionId := correlatedProperty.id.value
        behaviorFingerprint := correlatedProperty.behaviorFingerprint.render, kind := .«property» }]
      ++ (relations.map (·.definition)).eraseDups
    sources := [input.target.source, input.scenario.source, input.querySource,
      input.property.source] ++ (relations.map (·.source)).eraseDups
    knownGaps := knownGaps.toProvenanceGaps
    program := ← assembleProgram input.source identity (evidenceRules.map (·.1)) realization
      program (some input.vocabulary) input.instances
    -- A claim is recorded for each class the Program performs: every instance's actions, not only
    -- the operation the Contract follows, because each of them ran the class's example.
    abstractionClaims := (input.claims.filter fun claim =>
      performed.contains claim.member).map (·.row)
    contractId := identity.contractId
    properties := [lowered.contractLowering] ++ relations.map (·.lowering)
    -- Every clause the Model wrote must appear among the lowered ones, so a clause silently lost
    -- between the checked Property and the Contract rejects here, before any Driver I/O. A caller
    -- may name further clauses it requires; one this Case does not carry rejects the same way.
    -- A relation's rule requires the Program to construct the literal it compares against.
    coverage := { clauses := (correlatedRules.map (·.id) ++ required).eraseDups
                  inputs := relations.flatMap (·.inputs) }
  }

end Umpire.Case.Producer
