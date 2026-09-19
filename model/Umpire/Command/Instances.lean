import Umpire.Command.Authoring

/-!
# Several instances of one machine

A Model holds several instances of each entity, bounded by Limits, and a Search over them takes
every enabled step of every instance as a candidate: interleavings are paths. This module is how
that is done without a second Model: the product of `count` copies of one declared machine, built
at the data level from the checked declaration, with a state per slot and an action per
(slot, action) pair.

The product is a Search view, not a Model Definition. Its capability requires the machine's own
canonical-table law with the machine's own proof -- the product's rows are a pure function of the
machine's rows, so nothing is authorized that the machine did not authorize -- and its Definition
IDs hang off the machine's family under an `-instances-<count>` owner, so a Case's provenance still
names the machine.

Each slot is a **state field** of the product state, under its own definition, holding the
per-instance state's key. That is what lets a Property written over one instance be read over the
product: a claim about "the state after this action" becomes a claim about the acting slot's field,
which the evaluator reads apart from the product state the same way the Contract reads a machine's
`phase` apart from `phase-attempts`.

Instances are numbered from one where an author writes them and in every key; a slot index is the
same number less one. The product's catalogs are emitted in the canonical order Search admits, which
is the order of the lowered values' keys, not the order the slots happen to multiply out in.
-/

namespace Umpire.Command

open Umpire

/-- The state of `count` instances: one per-instance state per slot, in slot order. -/
abbrev Slots (State : Type) := List State

/-- One instance's action: the slot that takes it, numbered from zero, and the action itself. -/
abbrev Slotted (Action : Type) := Nat × Action

/-- Every `count`-tuple over `members`, first slot varying slowest. -/
def slotProducts (count : Nat) (members : List α) : List (List α) :=
  match count with
  | 0 => [[]]
  | count + 1 => members.flatMap fun head => (slotProducts count members).map (head :: ·)

/-- The name of one slot's field, numbered from one the way an author numbers instances. -/
def slotField (slot : Nat) : String := "instance-" ++ toString (slot + 1)

/-- The key of a slotted action: the instance number and the action's own key. -/
def slottedKey (slot : Nat) (actionKey : String) : String :=
  toString (slot + 1) ++ "_" ++ actionKey

/-- The key of a product state: every slot's key, in slot order. -/
def slotsKey (keys : List String) : String := "_".intercalate keys

/-- How many (state, action) evaluations the product of `count` instances walks, which is what the
enumeration bound is checked against before anything is built. -/
def instancesSize (states actions count : Nat) : Nat := states ^ count * (count * actions)

/-- The owner the product's Definition IDs hang off: the machine's, marked with its instance count. -/
def instancesOwner (key : String) (count : Nat) : String :=
  key ++ "-instances-" ++ toString count

private def orderedBy (key : α → String) (items : List α) : List α :=
  items.mergeSort fun left right => key left ≤ key right

/-- The product of `count` instances of a declared machine, as a declared Model of its own.

Every product state is a tuple of per-instance states and every product action is one instance's
action. A row steps one slot by the machine's own row and leaves the others where they are; a
product state ends when every slot ends; the product starts in every tuple of the machine's start
states. Outcomes and facts are the machine's own values under the machine's own definitions. -/
def DeclaredModel.instances [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (model : DeclaredModel Setup State Action Outcome Fact) (count : Nat) :
    DeclaredModel Setup (Slots State) (Slotted Action) Outcome Fact :=
  let origin := model.origin
  let ownerKey := instancesOwner model.key count
  let stateKeyOf : State → String := fun state =>
    ((model.table.states.find? fun entry => entry.value == state).map (·.key)).getD ""
  let actionKeyOf : Action → String := fun action =>
    ((model.table.actions.find? fun entry => entry.value == action).map (·.key)).getD ""
  let key : Slots State → String := fun slots => slotsKey (slots.map stateKeyOf)
  let slottedKeyOf : Slotted Action → String := fun taken =>
    slottedKey taken.1 (actionKeyOf taken.2)
  let stateId : Slots State → DefinitionId := fun slots =>
    origin.ownedId "state" ownerKey (key slots)
  let actionId : Slotted Action → DefinitionId := fun taken =>
    origin.ownedId "action" ownerKey (slottedKeyOf taken)
  let slotFieldId : Nat → DefinitionId := fun slot =>
    origin.ownedId "state-field" ownerKey (slotField slot)
  -- The catalogs in the order Search admits them: by the lowered value's order key.
  let stateOrder : Slots State → String := fun slots =>
    modelValueOrderKey (ModelValue.named (stateId slots) (key slots))
  let actionOrder : Slotted Action → String := fun taken =>
    modelValueOrderKey (ModelValue.named (actionId taken) (slottedKeyOf taken))
  let states := orderedBy stateOrder (slotProducts count model.states)
  let actions := orderedBy actionOrder
    ((List.range count).flatMap fun slot => model.actions.map fun action => (slot, action))
  let initial := orderedBy stateOrder (slotProducts count model.initial)
  let terminal := states.filter fun slots => slots.all model.terminal.contains
  let fieldValuesOf : Slots State → List ModelValue := fun slots =>
    slots.zipIdx.map fun (state, slot) => ModelValue.named (slotFieldId slot) (stateKeyOf state)
  let stepOrder : Step (Slots State) Outcome Fact → String := fun step =>
    stepOrderKey ⟨ModelValue.named (model.identity.outcomeId step.outcome) "",
      ModelValue.named (stateId step.state) (key step.state),
      step.facts.map fun fact => ModelValue.named (model.identity.factId fact) ""⟩
  -- One slot steps by the machine's own row; the others stay. The results are ordered by their
  -- lowered keys, which is the order Search admits.
  let row : Slots State → Slotted Action →
      Option (FiniteTransitionRow (Slots State) (Slotted Action) Outcome Fact) :=
    fun slots taken => do
      let current ← slots[taken.1]?
      let source ← model.table.transitions.find? fun candidate =>
        candidate.source == current && candidate.action == taken.2
      let results := source.results.map fun result =>
        ({ outcome := result.outcome
           state := slots.set taken.1 result.state
           facts := result.facts } : Step (Slots State) Outcome Fact)
      pure {
        key := key slots ++ "-" ++ slottedKeyOf taken
        source := slots
        action := taken
        results := orderedBy stepOrder results }
  let transitions := states.flatMap fun slots => actions.filterMap (row slots)
  let targetId := origin.family.id "target" ownerKey
  let kernelId := origin.ownedId "kernel" ownerKey "planner"
  let capabilityId := origin.ownedId "capability" ownerKey "transitions"
  let providerId := origin.ownedId "provider" ownerKey "finite-table"
  let operationRoleId := origin.ownedId "role" ownerKey model.roleName
  let stateIds := states.map stateId
  let stateFieldIds := (List.range count).map fun slot => (slotField slot, slotFieldId slot)
  let stateFieldValues := states.map fun slots =>
    slots.zipIdx.map fun (state, slot) => (slotFieldId slot, stateKeyOf state)
  let actionIds := actions.map actionId
  let relationIds := transitions.map fun row => origin.ownedId "relation" ownerKey row.key
  let table : FiniteTable Setup (Slots State) (Slotted Action) Outcome Fact := {
    setups := model.table.setups
    states := states.map fun slots => { value := slots, key := key slots }
    actions := actions.map fun taken => { value := taken, key := slottedKeyOf taken }
    outcomes := model.table.outcomes
    facts := model.table.facts
    initial := [{ setup := model.setupValue, states := initial }]
    transitions }
  let identity : FiniteModelIdentity Setup (Slots State) (Slotted Action) Outcome Fact := {
    setupBindings := fun _ => initial.map fun value => { roleId := operationRoleId, state := value }
    stateId
    actionId
    outcomeId := model.identity.outcomeId
    factId := model.identity.factId
    stateFields := fieldValuesOf }
  -- The product's capability is discharged by the machine's own law: its rows are a function of the
  -- machine's rows, so the machine's proof is the product's authority.
  let contract : Capability := {
    id := capabilityId
    behaviorVersion := capabilityId.value
    requiredLaws := [model.law] }
  let meanings :=
    table.states.map (fun entry => meaning (stateId entry.value) .state) ++
    stateFieldIds.map (fun (_, fieldId) => meaning fieldId .state) ++
    table.actions.map (fun entry => meaning (actionId entry.value) .action) ++
    table.outcomes.map (fun entry => meaning (model.identity.outcomeId entry.value) .outcome) ++
    table.facts.map (fun entry => meaning (model.identity.factId entry.value) .fact)
  let provider : Provider model.lawStatement := {
    id := providerId
    source := origin.source
    contract
    meanings
    lawProofs := [{ definition := model.law, proof := model.lawProof }] }
  let definitions :=
    [origin.metadata targetId .target, origin.metadata kernelId .machine,
      origin.metadata capabilityId .capability, origin.metadata providerId .provider,
      origin.metadata model.lawId .law] ++
    (meanings.map fun provided => origin.metadata provided.definitionId provided.kind) ++
    table.transitions.map fun row =>
      origin.metadata (origin.ownedId "relation" ownerKey row.key) .relation
  let modelSpec : TableModelSpec := {
    id := targetId
    source := origin.source
    definitions
    requiredCapabilities := [capabilityId]
    metadata := { id := kernelId, source := origin.source } }
  {
    origin, key := ownerKey, roleName := model.roleName, setupValue := model.setupValue,
    states, actions, outcomes := model.outcomes, facts := model.facts, initial, terminal,
    targetId, kernelId, capabilityId, providerId, lawId := model.lawId, operationRoleId,
    stateIds, stateFieldIds, stateFieldValues, actionIds, outcomeIds := model.outcomeIds,
    factIds := model.factIds, relationIds,
    table, identity, lawStatement := model.lawStatement, law := model.law,
    lawProof := model.lawProof,
    composition := Providers.empty |>.provide provider, modelSpec }

/-- The product is checked against the machine's own law, so a Query over it is a Query over the
machine's law statement: what `CheckedModel` types by. -/
theorem DeclaredModel.instances_lawStatement [BEq Setup] [BEq State] [BEq Action] [BEq Outcome]
    [BEq Fact] (model : DeclaredModel Setup State Action Outcome Fact) (count : Nat) :
    (model.instances count).lawStatement = model.lawStatement := rfl

/-! ### Reading one instance's claims over the product

A Property is written over one instance. Over the product it is the same claim for each instance:
triggered by that instance's action and answered on that instance's slot. -/

/-- The Property one instance's groups make over the product: one clause per instance per
requirement, each reading the acting slot's field. -/
def liftedProperty [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (model : DeclaredModel Setup State Action Outcome Fact) (count : Nat)
    (values : ModelVocabulary) (names : PropertyNames) : Property :=
  let product := model.instances count
  let slotFieldId := fun slot => product.origin.ownedId "state-field" product.key (slotField slot)
  {
    id := model.origin.family.id "property" names.declaration
    source := model.origin.source
    requires := [product.roleCapability names.roleName]
    clauses := (List.range count).flatMap fun slot => names.groups.flatMap fun group =>
      let trigger := match group.trigger with
        | .action spelling => PropertyPattern.selectedAction
            (values.namedAction (slottedKey slot spelling))
        | .priorState spelling => PropertyPattern.exact .priorState (slotFieldId slot) spelling
      let clauseId := fun (label : String) =>
        model.origin.ownedId "property" names.declaration (slotField slot ++ "-" ++ label)
      group.requirements.map fun requirement =>
        match requirement with
        | .stateClause label spelling =>
            .transitionContract (clauseId label) trigger
              (PropertyPattern.exact .resultingState (slotFieldId slot) spelling)
        | .outcomeClause label spelling =>
            .transitionContract (clauseId label) trigger (.outcome (values.namedOutcome spelling))
        | .factClause label spelling =>
            .inputOutput (clauseId label) trigger (.fact (values.namedFact spelling))
  }

/-- The Scenario one instance-qualified sequence makes over the product: every instance starts in
the named start state, and each occurrence is the named instance's action. -/
def liftedScenario [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (model : DeclaredModel Setup State Action Outcome Fact) (count : Nat)
    (values : ModelVocabulary) (names : ScenarioNames) : Scenario :=
  authoredScenario (model.instances count) values {
    declaration := names.declaration
    roleName := names.roleName
    setupState := slotsKey (List.replicate count names.setupState)
    occurrences := names.occurrences.map fun occurrence => {
      occurrence with
      action := slottedKey (occurrence.instanceNumber - 1) occurrence.action
      instanceNumber := 1 } }

/-- The instance number a product action's key names, and the action's own key. -/
def unslotKey (key : String) : Option (Nat × String) :=
  match key.splitOn "_" with
  | slot :: rest => if rest.isEmpty then none else
      slot.toNat?.map fun number => (number, "_".intercalate rest)
  | [] => none

/-! ### Checking a Query over several instances

The Search runs over the product; the Producer reads one instance. The one instance is the first,
and every instance performs the same sequence, because a Contract follows each operation through
the machine on its own and there is one Contract per Case: two instances that did different things
would need two sequences for one rule to be derived from. -/

/-- The per-instance step one product step stands for, read through the acting slot's field. -/
private def projectStep (values : ModelVocabulary) (slot : Nat)
    (fields : ModelValue → List ModelValue)
    (step : ModelTraceStep ModelValue ModelValue ModelValue ModelValue) :
    Option (ModelTraceStep ModelValue ModelValue ModelValue ModelValue) := do
  let (number, actionKey) ← unslotKey step.selectedAction.value
  if number != slot + 1 then none
  let slotState ← (fields step.state).find? fun field =>
    field.definitionId.value.endsWith ("." ++ slotField slot)
  pure {
    selectedAction := values.namedAction actionKey
    outcome := step.outcome
    state := values.namedState slotState.value
    facts := step.facts }

/-- Check a Query over `count` instances of one machine: the product is searched, and what a
Producer reads is the first instance's projection of what the Search selected, with every
instance's actions as the Program's path. -/
def checkInstances [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Fact]
    (model : DeclaredModel Setup State Action Outcome Fact)
    (count : Nat)
    (queryKey : String)
    (limits : Limits)
    (propertyNames : PropertyNames)
    (scenarioNames : ScenarioNames)
    (knownGaps : List KnownGap := [])
    (form : QueryFormKind := .selectWitness) :
    Except AdmissionError (CheckedModel model) := do
  if count == 0 then
    throw (.instances "an instance count of zero admits no instance to run the Scenario over")
  if let some stray := scenarioNames.occurrences.find? fun occurrence =>
      occurrence.instanceNumber == 0 || occurrence.instanceNumber > count then
    throw (.instances s!"the Scenario names instance {stray.instanceNumber} of {count}")
  let sequences := (List.range count).map fun slot =>
    (scenarioNames.occurrences.filter (·.instanceNumber == slot + 1)).map (·.action)
  match sequences.head? with
  | none => pure ()
  | some first =>
      if let some slot := (List.range count).find? fun slot => sequences[slot]? != some first then
        throw (.instances s!"instance {slot + 1} performs {", ".intercalate
          ((sequences[slot]?).getD [])} where instance 1 performs {", ".intercalate first}; a \
Case follows each operation through one sequence, so every instance performs the same actions")
  let product := model.instances count
  let checked ← check product queryKey limits (liftedProperty model count · propertyNames)
    (liftedScenario model count · scenarioNames) knownGaps form
  -- The one instance a Producer reads: the machine as declared, its Property and the first
  -- instance's own sequence, admitted on their own.
  let target ← checkFiniteTarget model.table model.table model.identity model.modelSpec
    model.composition |>.mapError .invalidTarget
  let vocabulary ← modelVocabulary model model.table |>.mapError .invalidVocabulary
  let own : ScenarioNames := { scenarioNames with
    occurrences := (scenarioNames.occurrences.filter (·.instanceNumber == 1)).map fun occurrence =>
      { occurrence with instanceNumber := 1 } }
  let admitted ← Search.admit target (authoredProperty model vocabulary propertyNames)
      (some (authoredScenario model vocabulary own)) {
    id := model.origin.family.id "query" queryKey
    source := model.origin.source
    target := model.targetId
    form := match form with
      | .selectWitness => .find
      | .verifyClaim => .verify
    limits
    policy := match form with
      | .selectWitness => .shortest
      | .verifyClaim => .exhaustive
  } knownGaps |>.mapError .admission
  let fields := checked.target.stateFields
  let witness := checked.witness.map fun selected =>
    let initial := ((fields selected.trace.initialState).find? fun field =>
      field.definitionId.value.endsWith ("." ++ slotField 0)).map fun field =>
        vocabulary.namedState field.value
    ({ setup := [{ «role» := model.operationRoleId, value := vocabulary.stateAt 0 }]
       trace := {
         initialState := initial.getD (vocabulary.stateAt 0)
         steps := selected.trace.steps.filterMap (projectStep vocabulary 0 fields) } } :
      Scenario.Trace)
  let program := checked.witness.map fun selected =>
    selected.trace.steps.filterMap fun step =>
      (unslotKey step.selectedAction.value).map fun (_, actionKey) =>
        (vocabulary.namedAction actionKey).definitionId
  pure {
    target := model.instances_lawStatement count ▸ checked.target
    vocabulary := checked.vocabulary
    property := checked.property
    behavior := checked.behavior
    query := model.instances_lawStatement count ▸ checked.query
    run := checked.run
    witness := checked.witness
    instances := count
    realizable := {
      target, vocabulary
      property := admitted.property
      behavior := admitted.scenario
      witness, program } }

end Umpire.Command
