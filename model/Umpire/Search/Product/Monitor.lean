import Umpire.Search.Product.Scenario

/-!
# Property monitors

A Query's verdict on a trace is `evaluatePropertyEndpoint` of each of its Properties: a
`PropertyEndpointAnswer` (`satisfied`, `violated` or `unresolved`) read off the whole trace, closed
(`final` or `terminal`) or `partial`. A state-space search never holds the whole trace, so each
clause is lowered to a `Monitor`: a small `ClauseState` advanced one product transition at a time,
from which the clause's answer under either ending is read. `QueryMonitors` is every clause of a
Query's Properties, and the product's `MonitorFamily` (`Umpire.Search.Product`).

## What each monitor records

Each shape is read off the evaluator (`Umpire.Property.Evaluate`), not the clause's name. A
monitor sees exactly the values the evaluator's view offers: the trace admitted through the
Property's capability view, with each admitted state's fields beside it.

* `stateInvariant` keeps a seen bit (some state or field under the pattern's definition was seen)
  and a failed bit (one of them broke the constraint). Closed it is satisfied only when seen and not
  failed; partial it is `unresolved` while nothing was seen.
* `transitionContract` and `inputOutput` are one-step implications: a failed bit. Their coverage
  seen bit is the clause's bit in the product's fired-clause bitset.
* `identityRelation` is existential over the trace: a monotone seen bit; partial it is `unresolved`
  until it fires.
* `ordered` keeps a decided bit and where the earliest `before` occurrence lies relative to the
  current position (`Mark`); partial it is `unresolved` until decided.
* `eventuallyWithin` keeps the remaining budget of its earliest pending trigger, a dead bit (a
  trigger's window closed without a response) and whether a response sits at the current position;
  `neverWithin` keeps the remaining budget of its latest open window, a violated bit and whether a
  forbidden occurrence sits at the current position. Both are `unresolved` before a deadline. A
  budget never exceeds the clause Limit.

Positions are the evaluator's: `LimitUnit.steps` and `LimitUnit.actions` both count product steps,
except that a prior-state occurrence counted in steps sits one position back. So after `n` steps
every occurrence seen lies at or before `n`, and every later one at or after `n`; that is why a
bit for "at the current position" and a budget relative to `n` carry everything the future reads,
and why only the earliest pending deadline of `eventuallyWithin` and the latest open window of
`neverWithin` matter.

A monitor also reports whether its clause's trigger fired: the precondition of a
`transitionContract`, the input of an `inputOutput`, or the trigger of a bounded clause, at the
initial state when the pattern reads `state` and at every step. These are the clauses the evaluator
requests triggers for, so the fired bits are what coverage reads.

## Unsupported

`branches`, `guardedEventuallyWithin`, `guardedNeverWithin`, correlated clauses, and Limits in
`logicalTime` (or any unit other than `steps` and `actions`) are not lowered: `QueryMonitors.lower`
returns `MonitorUnsupported` naming the Property, the clause and its kind. That is the whole error
surface; nothing is weakened silently.

## Evidence that a monitor is the evaluator (R6)

The evidence for every kind is **testing** (`MonitorKind.trustBasis`): the clause semantics
(`clauseEndpointAnswer`, the view construction) are private to `Umpire.Property.Evaluate` and
`Umpire.Property.Check`, which no other module can unfold. `Umpire.Search.Tests.Monitor` compares
every monitor's answers and fired bits with `evaluatePropertyEndpoint` under both endings,
exhaustively: over every trace of bounded length of a synthetic alphabet, for a table of clauses
covering every kind, trace field and unit, and over every trace within the Limits of the checked-in
Models the Umpire test roots reach. `Umpire.Search.Product.MonitorProofs` proves what holds of the
monitors alone: budgets stay within the Limit, closed answers are never `unresolved`, and clauses
without a trigger never fire.
-/

namespace Umpire.Search.Product

/-- A clause kind or Limit unit the version-one monitors do not encode. -/
inductive UnsupportedClause where
  | branches
  | guardedEventuallyWithin
  | guardedNeverWithin
  | correlated
  | limitUnit (unit : LimitUnit)
  deriving BEq, DecidableEq, Repr

/-- The kind's name, as the planning receipt's `unsupported-clause:<kind>` reason spells it. -/
def UnsupportedClause.name : UnsupportedClause → String
  | .branches => "branches"
  | .guardedEventuallyWithin => "guardedEventuallyWithin"
  | .guardedNeverWithin => "guardedNeverWithin"
  | .correlated => "correlated"
  | .limitUnit unit => unit.name ++ "-limit"

/-- A Property clause the version-one monitors cannot lower, naming the Property, the clause and
its kind. This is the whole error surface of lowering. -/
structure MonitorUnsupported where
  propertyId : DefinitionId
  clauseId : DefinitionId
  source : SourceLocation
  kind : UnsupportedClause
  deriving BEq, DecidableEq, Repr

/-- The clause kinds version one lowers. -/
inductive MonitorKind where
  | stateInvariant
  | transitionContract
  | identityRelation
  | inputOutput
  | ordered
  | eventuallyWithin
  | neverWithin
  deriving BEq, DecidableEq, Repr

def MonitorKind.name : MonitorKind → String
  | .stateInvariant => "stateInvariant"
  | .transitionContract => "transitionContract"
  | .identityRelation => "identityRelation"
  | .inputOutput => "inputOutput"
  | .ordered => "ordered"
  | .eventuallyWithin => "eventuallyWithin"
  | .neverWithin => "neverWithin"

def MonitorKind.all : List MonitorKind :=
  [.stateInvariant, .transitionContract, .identityRelation, .inputOutput, .ordered,
    .eventuallyWithin, .neverWithin]

/-- What a kind's agreement with the evaluator rests on: a kernel-checked theorem, or the
exhaustive differential test. -/
inductive TrustBasis where
  | kernel
  | testing
  deriving BEq, DecidableEq, Repr

def TrustBasis.name : TrustBasis → String
  | .kernel => "kernel"
  | .testing => "testing"

/-- The agreement evidence the receipt records per kind. Every kind is `testing`; see the module
header for why. -/
def MonitorKind.trustBasis : MonitorKind → TrustBasis
  | _ => .testing

/-- The unit a positional clause counts in. The two differ only in where a prior-state occurrence
sits: one position back in `steps`, at its step in `actions`. -/
inductive Counting where
  | steps
  | actions
  deriving BEq, DecidableEq, Repr

private def Counting.of : LimitUnit → Except UnsupportedClause Counting
  | .steps => .ok .steps
  | .actions => .ok .actions
  | unit => .error (.limitUnit unit)

/-- Whether an occurrence of this pattern at step `n + 1` sits at position `n` rather than
`n + 1`. -/
private def Counting.lags (counting : Counting) (pattern : PropertyPattern) : Bool :=
  counting == .steps && pattern.field == .priorState

/-- One lowered clause. -/
inductive Monitor where
  | stateInvariant (state : PropertyPattern)
  | transitionContract (precondition postcondition : PropertyPattern)
  | identityRelation (relation : PropertyPattern)
  | inputOutput (input output : PropertyPattern)
  | ordered (before after : PropertyPattern) (counting : Counting)
  | eventuallyWithin (trigger response : PropertyPattern) (bound : Nat) (counting : Counting)
  | neverWithin (trigger forbidden : PropertyPattern) (bound : Nat) (counting : Counting)
  deriving BEq, DecidableEq, Repr

/-- Lower one clause, or name the kind or Limit unit version one does not encode. -/
def Monitor.lower : CheckedPropertyClause → Except UnsupportedClause Monitor
  | .stateInvariant _ state => .ok (.stateInvariant state)
  | .transitionContract _ precondition postcondition =>
      .ok (.transitionContract precondition postcondition)
  | .identityRelation _ relation => .ok (.identityRelation relation)
  | .inputOutput _ input output => .ok (.inputOutput input output)
  | .ordered _ before after unit => (Counting.of unit).map (.ordered before after)
  | .eventuallyWithin _ trigger response limit =>
      (Counting.of limit.unit).map (.eventuallyWithin trigger response limit.value)
  | .neverWithin _ trigger forbidden limit =>
      (Counting.of limit.unit).map (.neverWithin trigger forbidden limit.value)
  | .branches _ => .error .branches
  | .guardedEventuallyWithin _ => .error .guardedEventuallyWithin
  | .guardedNeverWithin _ => .error .guardedNeverWithin

def Monitor.kind : Monitor → MonitorKind
  | .stateInvariant .. => .stateInvariant
  | .transitionContract .. => .transitionContract
  | .identityRelation .. => .identityRelation
  | .inputOutput .. => .inputOutput
  | .ordered .. => .ordered
  | .eventuallyWithin .. => .eventuallyWithin
  | .neverWithin .. => .neverWithin

/-- The pattern whose occurrences the evaluator reports as the clause's realized trigger. -/
def Monitor.trigger? : Monitor → Option PropertyPattern
  | .transitionContract trigger _ | .inputOutput trigger _
  | .eventuallyWithin trigger _ _ _ | .neverWithin trigger _ _ _ => some trigger
  | _ => none

/-- The values one position of the evaluator's view offers a pattern: an admitted step, or the
admitted initial state (only `resultingState` and `stateFields` set). -/
structure Observed where
  priorState : Option ModelValue := none
  priorStateFields : List ModelValue := []
  selectedAction : Option ModelValue := none
  outcome : Option ModelValue := none
  resultingState : Option ModelValue := none
  stateFields : List ModelValue := []
  observations : List ModelValue := []
  deriving BEq, Repr

/-- The values a trace field offers at this position: a state offers itself and every field it
holds. -/
def Observed.values (observed : Observed) : PropertyTraceField → List ModelValue
  | .state | .resultingState => observed.resultingState.toList ++ observed.stateFields
  | .priorState => observed.priorState.toList ++ observed.priorStateFields
  | .selectedAction => observed.selectedAction.toList
  | .outcome => observed.outcome.toList
  | .observation | .relation => observed.observations

def Observed.holds (observed : Observed) (pattern : PropertyPattern) : Bool :=
  (observed.values pattern.field).any pattern.evaluate

/-- Whether the pattern occurs at the initial state. The evaluator reads only the state itself
there, not its fields, and only for a pattern on `state`. -/
private def Observed.initialOccurs (initial : Observed) (pattern : PropertyPattern) : Bool :=
  pattern.field == .state && initial.resultingState.any pattern.evaluate

/-- Whether a pattern occurs at a step `n + 1` at position `n` and at position `n + 1`. -/
private def Observed.occursAt (step : Observed) (counting : Counting) (pattern : PropertyPattern) :
    Bool × Bool :=
  let holds := step.holds pattern
  if counting.lags pattern then (holds, false) else (false, holds)

/-- Where the earliest `before` occurrence of an `ordered` clause lies after `n` steps: none seen,
at position `n`, or before it. -/
inductive Mark where
  | none
  | current
  | earlier
  deriving BEq, DecidableEq, Hashable, Repr

/-- A monitor's state. See the module header for what each records. -/
inductive ClauseState where
  | stateInvariant (seen failed : Bool)
  | contract (failed : Bool)
  | relation (seen : Bool)
  | ordered (decided : Bool) (before : Mark)
  | eventually (dead : Bool) (pending : Option Nat) (responseHere : Bool)
  | never (violated : Bool) (window : Option Nat) (forbiddenHere : Bool)
  deriving BEq, DecidableEq, Hashable, Repr

namespace Monitor

/-! The step functions, public so `Umpire.Search.Product.MonitorProofs` can unfold them. -/

def matchingState (pattern : PropertyPattern) (values : List ModelValue) :
    List ModelValue :=
  values.filter fun value => value.definitionId == pattern.reference

def invariantAt (seen failed : Bool) (pattern : PropertyPattern)
    (values : List ModelValue) : ClauseState :=
  let matching := matchingState pattern values
  .stateInvariant (seen || !matching.isEmpty)
    (failed || matching.any fun value => !pattern.constraint.evaluate value.value)

def orderedStart (before : Bool) : ClauseState :=
  .ordered false (if before then .current else .none)

def orderedStep (decided : Bool) (mark : Mark) (before after : Bool × Bool) :
    ClauseState :=
  let (beforeHere, beforeNext) := before
  let (afterHere, afterNext) := after
  if decided || (afterHere && mark == .earlier) ||
      (afterNext && (mark != .none || beforeHere)) then
    .ordered true .none
  else
    .ordered false (if mark != .none || beforeHere then .earlier
      else if beforeNext then .current else .none)

def earliest : Option Nat → Option Nat → Option Nat
  | some left, some right => some (Nat.min left right)
  | left, right => left <|> right

def latest : Option Nat → Option Nat → Option Nat
  | some left, some right => some (Nat.max left right)
  | left, right => left <|> right

/-- A pending deadline with `remaining` positions left after the position a step starts at, once
the step's responses are seen: `none` when answered, `some none` when its window closed,
`some (some d)` still pending with `d` positions left after the next one. -/
def pendingAfter (remaining : Nat) (responseHere responseNext : Bool) : Option (Option Nat) :=
  if responseHere || (responseNext && remaining ≥ 1) then none
  else if remaining == 0 then some none
  else some (some (remaining - 1))

def eventuallyStep (bound : Nat) (pending : Option Nat) (responseSeen : Bool)
    (trigger response : Bool × Bool) : ClauseState :=
  let (triggerHere, triggerNext) := trigger
  let (responseHere, responseNext) := response
  let old := pending.bind fun remaining => pendingAfter remaining responseHere responseNext
  let current := if triggerHere then
    pendingAfter bound (responseSeen || responseHere) responseNext
  else none
  let next : Option Nat := if triggerNext && !responseNext then some bound else none
  if old == some none || current == some none then
    .eventually true none false
  else
    .eventually false (earliest old.join (earliest current.join next)) responseNext

def neverStep (bound : Nat) (window : Option Nat) (forbiddenSeen : Bool)
    (trigger forbidden : Bool × Bool) : ClauseState :=
  let (triggerHere, triggerNext) := trigger
  let (forbiddenHere, forbiddenNext) := forbidden
  let violated :=
    window.any (fun remaining => forbiddenHere || (forbiddenNext && remaining ≥ 1)) ||
      (triggerHere && (forbiddenSeen || forbiddenHere || (forbiddenNext && bound ≥ 1))) ||
      (triggerNext && forbiddenNext)
  if violated then
    .never true none false
  else
    let old := window.bind fun remaining => if remaining ≥ 1 then some (remaining - 1) else none
    let current := if triggerHere && bound ≥ 1 then some (bound - 1) else none
    let next := if triggerNext then some bound else none
    .never false (latest old (latest current next)) forbiddenNext

end Monitor

/-- The monitor's state at the initial state, and whether its trigger fired there. -/
def Monitor.start (monitor : Monitor) (initial : Observed) : ClauseState × Bool :=
  let fired := monitor.trigger?.any initial.initialOccurs
  let state := match monitor with
    | .stateInvariant pattern =>
        Monitor.invariantAt false false pattern (initial.values .state)
    | .transitionContract .. | .inputOutput .. => .contract false
    | .identityRelation relation =>
        .relation (relation.field == .state && (initial.values .state).any relation.evaluate)
    | .ordered before _ _ => Monitor.orderedStart (initial.initialOccurs before)
    | .eventuallyWithin trigger response bound _ =>
        let responded := initial.initialOccurs response
        .eventually false
          (if initial.initialOccurs trigger && !responded then some bound else none) responded
    | .neverWithin trigger forbidden bound _ =>
        let triggered := initial.initialOccurs trigger
        let seen := initial.initialOccurs forbidden
        if triggered && seen then .never true none false
        else .never false (if triggered then some bound else none) seen
  (state, fired)

/-- Advance the monitor over one admitted step, and whether its trigger fired there. A state of
another monitor's shape is returned unchanged; `QueryMonitors` never pairs them. -/
def Monitor.advance (monitor : Monitor) (state : ClauseState) (step : Observed) :
    ClauseState × Bool :=
  let fired := monitor.trigger?.any step.holds
  let next := match monitor, state with
    | .stateInvariant pattern, .stateInvariant seen failed =>
        Monitor.invariantAt seen failed pattern (step.values .state)
    | .transitionContract precondition postcondition, .contract failed
    | .inputOutput precondition postcondition, .contract failed =>
        .contract (failed || (step.holds precondition && !step.holds postcondition))
    | .identityRelation relation, .relation seen => .relation (seen || step.holds relation)
    | .ordered before after counting, .ordered decided mark =>
        Monitor.orderedStep decided mark (step.occursAt counting before)
          (step.occursAt counting after)
    | .eventuallyWithin trigger response bound counting, .eventually dead pending responseSeen =>
        if dead then .eventually true none false
        else Monitor.eventuallyStep bound pending responseSeen
          (step.occursAt counting trigger) (step.occursAt counting response)
    | .neverWithin trigger forbidden bound counting, .never violated window forbiddenSeen =>
        if violated then .never true none false
        else Monitor.neverStep bound window forbiddenSeen
          (step.occursAt counting trigger) (step.occursAt counting forbidden)
    | _, state => state
  (next, fired)

/-- The clause's answer on the trace that reached this state, closed or partial. -/
def ClauseState.answer (state : ClauseState) (partialTrace : Bool) : PropertyEndpointAnswer :=
  let verdict (satisfied : Bool) (open? : Bool) : PropertyEndpointAnswer :=
    if satisfied then .satisfied else if partialTrace && open? then .unresolved else .violated
  match state with
  | .stateInvariant seen failed => verdict (seen && !failed) !seen
  | .contract failed => verdict (!failed) false
  | .relation seen => verdict seen true
  | .ordered decided _ => verdict decided true
  | .eventually dead pending _ =>
      if partialTrace then
        if dead || pending == some 0 then .violated
        else if pending.isSome then .unresolved else .satisfied
      else verdict (!dead && pending.isNone) false
  | .never violated window _ =>
      if violated then .violated
      else if partialTrace && window.any (· ≥ 1) then .unresolved
      else .satisfied

/-- The evaluator's combination of clause answers into a Property's: any violation, else any
unresolved clause, else satisfied. -/
def combineAnswers (answers : List PropertyEndpointAnswer) : PropertyEndpointAnswer :=
  if answers.contains .violated then .violated
  else if answers.contains .unresolved then .unresolved
  else .satisfied

/-- One lowered clause of a Query, with the Property it belongs to and the capability view that
admits what it reads. -/
structure ClauseMonitor where
  /-- The Property's index in `QueryMonitors.propertyIds`. -/
  property : Nat
  propertyId : DefinitionId
  clauseId : DefinitionId
  access : PropertyCapabilityView
  monitor : Monitor
  deriving BEq, Repr

/-- Every clause of a Query's Properties, lowered. Clause `i` owns bit `i` of the fired-clause
bitset. Build one with `QueryMonitors.lower`. -/
structure QueryMonitors where
  /-- The Properties by Definition ID, the order the reference search evaluates them in. -/
  propertyIds : List DefinitionId
  clauses : List ClauseMonitor
  /-- The Model's fields of a state, as `checkPropertyEvaluationInput` reads them. -/
  stateFields : ModelValue → List ModelValue

/-- Lower every clause of these Properties, or name the first one version one cannot encode. -/
def QueryMonitors.lower (properties : List CheckedProperty)
    (stateFields : ModelValue → List ModelValue) : Except MonitorUnsupported QueryMonitors := do
  let sorted := properties.mergeSort fun left right => decide (left.id.value ≤ right.id.value)
  let mut clauses := #[]
  for (property, index) in sorted.zipIdx do
    if let some rule := property.correlatedRules.head? then
      throw ({
        propertyId := property.id
        clauseId := rule.declaration.id
        source := property.source
        kind := .correlated } : MonitorUnsupported)
    for clause in property.clauses do
      match Monitor.lower clause with
      | .error kind =>
          throw ({
            propertyId := property.id
            clauseId := clause.id
            source := property.source
            kind } : MonitorUnsupported)
      | .ok monitor =>
          clauses := clauses.push {
            property := index
            propertyId := property.id
            clauseId := clause.id
            access := property.access
            monitor }
  pure { propertyIds := sorted.map (·.id), clauses := clauses.toList, stateFields }

namespace QueryMonitors

variable (monitors : QueryMonitors)

private def admit (access : PropertyCapabilityView) (value : ModelValue) : Option ModelValue :=
  if access.allows value then some value else none

private def fieldsOf (access : PropertyCapabilityView) (state : Option ModelValue) :
    List ModelValue :=
  ((state.map monitors.stateFields).getD []).filter access.allows

/-- The initial state as a Property with this capability view sees it. -/
def observeInitial (access : PropertyCapabilityView) (initialState : ModelValue) : Observed :=
  let state := admit access initialState
  { resultingState := state, stateFields := monitors.fieldsOf access state }

/-- One step from `prior` as a Property with this capability view sees it. -/
def observeStep (access : PropertyCapabilityView) (prior : ModelValue) (transition : Transition) :
    Observed :=
  let priorState := admit access prior
  let resultingState := admit access transition.result.state
  {
    priorState
    priorStateFields := monitors.fieldsOf access priorState
    selectedAction := admit access transition.action
    outcome := admit access transition.result.outcome
    resultingState
    stateFields := monitors.fieldsOf access resultingState
    observations := transition.result.facts.filter access.allows
  }

private def bits (fired : List Bool) : Nat :=
  fired.zipIdx.foldl (fun bits (fired, index) => if fired then bits ||| (1 <<< index) else bits) 0

/-- Every monitor's state at a root, and the bits of the clauses whose trigger fired there. -/
def start (initialState : ModelValue) : List ClauseState × Nat :=
  let started := monitors.clauses.map fun clause =>
    clause.monitor.start (monitors.observeInitial clause.access initialState)
  (started.map (·.1), bits (started.map (·.2)))

/-- Every monitor's state one transition from `prior`, and the bits of the clauses whose trigger
fired there. -/
def advance (states : List ClauseState) (prior : ModelValue) (transition : Transition) :
    List ClauseState × Nat :=
  let advanced := (monitors.clauses.zip states).map fun (clause, state) =>
    clause.monitor.advance state (monitors.observeStep clause.access prior transition)
  (advanced.map (·.1), bits (advanced.map (·.2)))

/-- Each Property's answer, in `propertyIds` order, on the trace that reached these states. -/
def answers (states : List ClauseState) (partialTrace : Bool) : List PropertyEndpointAnswer :=
  let answered := (monitors.clauses.zip states).map fun (clause, state) =>
    (clause.property, state.answer partialTrace)
  monitors.propertyIds.zipIdx.map fun (_, index) =>
    combineAnswers ((answered.filter (·.1 == index)).map (·.2))

/-- The clauses the evaluator requests triggers for, as `(Property, clause)`. -/
def requested : List (DefinitionId × DefinitionId) :=
  (monitors.clauses.filter (·.monitor.trigger?.isSome)).map fun clause =>
    (clause.propertyId, clause.clauseId)

/-- The clauses whose bit is set in a fired-clause bitset, as `(Property, clause)`. -/
def firedClauses (fired : Nat) : List (DefinitionId × DefinitionId) :=
  (monitors.clauses.zipIdx.filter fun (_, index) => fired.testBit index).map fun (clause, _) =>
    (clause.propertyId, clause.clauseId)

end QueryMonitors

end Umpire.Search.Product
