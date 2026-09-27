import Umpire.Search

/-!
# The Scenario progress automaton

`CheckedScenario.admits` decides whether one whole `Scenario.Trace` belongs to a Scenario. A
state-space search never holds whole traces: it reaches each state once and keeps only what the
state records. `ScenarioAutomaton` is the Scenario's part of that state: a small `Progress` value,
advanced one `Transition` at a time, whose final value decides `admits`.

## What a `Progress` records

`ScenarioAutomaton.lower` reads the checked Scenario once and picks one of two shapes.

* **Pinned schedule.** When `actionsExactly` or `traceExactly` is present, only one action schedule
  can be admitted. Every constraint over the action sequence -- allowed and forbidden actions,
  occurrence bounds, required occurrences and their `ordering`, `sequences`, `adjacencies` -- then
  has one answer, which lowering computes by asking `admits` itself about that schedule. The
  progress is the position reached in the schedule: the exact-trace index. A step that leaves the
  schedule, or differs from the exact trace's step, has no admitted extension and is dropped.
* **Free schedule.** Otherwise the progress keeps one occurrence counter per action a bound or a
  required occurrence names, and one index per `sequences` entry: how much of it the trace has
  matched so far, greedily, which is how a subsequence is recognised. Required occurrences with no
  `ordering` are a minimum count of their action. A counter saturates at its action's declared
  maximum, or at its minimum when there is no maximum: a step past the maximum, like a step to a
  forbidden or unallowed action, has no admitted extension (the same prefixes
  `CheckedScenario.admitsPrefix` rejects) and is dropped, so no count above the maximum is ever
  stored. `ordering`, whose slot assignment depends on the set of occurrences still unassigned, and
  `adjacencies`, whose substring match needs the active partial matches, are not encoded:
  lowering returns `Unsupported` naming the construct.

The fixed setup -- the role bindings and setup constraints, and an exact trace's setup and initial
state -- is decided once, at the root, by `admits` on the root trace of a Scenario that keeps only
those fields. An unsatisfiable Scenario admits no root.

## Evidence that the automaton is `admits` (R14)

The evidence is **testing**, not a theorem: `CheckedScenario.admits` is built from
`Umpire.Scenario.Check`'s private helpers, which no other module can unfold.
`Umpire.Search.Tests.Product` compares `ScenarioAutomaton.admits` with `CheckedScenario.admits` on
every trace of every Scenario it holds: exhaustively over bounded action sequences for a table of
Scenarios that exercises every construct, and over every trace within the Query Limits of the
checked-in Scenarios the Umpire test roots reach (the Switch example and the Search fixtures). It
also keeps `CheckedScenario.admitsPrefix` as a pruning oracle: every prefix it rejects, the
automaton drops.
-/

namespace Umpire.Search.Product

/-- One product transition: the selected Model action and the Model-owned `Step` result it led to.
A product path is a list of these, which is exactly the step list of a `Scenario.Trace`. -/
structure Transition where
  action : ModelValue
  result : Step ModelValue ModelValue ModelValue
  deriving BEq, DecidableEq, Repr

/-- The trace step a transition records. -/
def Transition.toTraceStep (transition : Transition) :
    ModelTraceStep ModelValue ModelValue ModelValue ModelValue :=
  ModelTraceStep.result transition.action transition.result

/-- The transition a trace step records. -/
def Transition.ofTraceStep (step : ModelTraceStep ModelValue ModelValue ModelValue ModelValue) :
    Transition := {
  action := step.selectedAction
  result := { outcome := step.outcome, state := step.state, facts := step.facts }
}

@[simp] theorem Transition.ofTraceStep_toTraceStep (transition : Transition) :
    Transition.ofTraceStep transition.toTraceStep = transition := rfl

@[simp] theorem Transition.toTraceStep_ofTraceStep
    (step : ModelTraceStep ModelValue ModelValue ModelValue ModelValue) :
    (Transition.ofTraceStep step).toTraceStep = step := rfl

/-- A Scenario construct the version-one progress automaton does not encode. -/
inductive ScenarioConstruct where
  | ordering
  | adjacencies
  deriving BEq, DecidableEq, Repr

/-- The construct's name, as the planning receipt's `unsupported-scenario:<construct>` reason
spells it. -/
def ScenarioConstruct.name : ScenarioConstruct → String
  | .ordering => "ordering"
  | .adjacencies => "adjacencies"

/-- A Scenario the version-one automaton cannot lower, naming the Scenario and the construct. This
is the whole error surface of lowering. -/
structure Unsupported where
  scenarioId : DefinitionId
  source : SourceLocation
  construct : ScenarioConstruct
  deriving BEq, DecidableEq, Repr

/-- The progress a Scenario has made along a trace. Under a pinned schedule only `position` moves;
under a free one, `counts` and `sequences` do and `position` stays `0`, so two traces that agree on
everything the verdict reads share one value. -/
structure Progress where
  position : Nat := 0
  counts : List Nat := []
  sequences : List Nat := []
  deriving BEq, DecidableEq, Hashable, Repr

/-- The occurrence demand on one action: the largest minimum any bound or the required
occurrences ask for, and the smallest declared maximum. -/
private structure Counter where
  action : DefinitionId
  minimum : Nat
  maximum : Option Nat
  deriving BEq, DecidableEq, Repr

/-- The value a counter stops at: nothing above the maximum is ever stored, and above the minimum
an unbounded count no longer changes the verdict. -/
private def Counter.cap (counter : Counter) : Nat :=
  counter.maximum.getD counter.minimum

private inductive Schedule where
  | pinned
      (actions : Option (List DefinitionId))
      (steps : Option (List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue)))
      (admitted : Bool)
  | free
      (allowed forbidden : List DefinitionId)
      (counters : List Counter)
      (sequences : List (List DefinitionId))
  deriving BEq, DecidableEq, Repr

/-- A checked Scenario lowered to its progress automaton. Build one with
`ScenarioAutomaton.lower`. -/
structure ScenarioAutomaton where
  private mk ::
  /-- The Scenario that keeps only the root-deciding fields; its `admits` of a root trace decides
  whether the root is admitted. -/
  private rootScenario : CheckedScenario
  private schedule : Schedule

private def rootTrace (setup : List RoleBinding) (initialState : ModelValue) : Scenario.Trace := {
  setup
  trace := { initialState, steps := [] }
}

/-- The Scenario with every action-sequence constraint removed and the exact trace cut to its setup
and initial state: it admits a trace exactly when the Scenario's setup part does. -/
private def rootScenarioOf (behavior : CheckedScenario) : CheckedScenario := {
  behavior with
  allowedActions := []
  requiredOccurrences := []
  forbiddenActions := []
  occurrenceBounds := []
  ordering := []
  sequences := []
  adjacencies := []
  actionsExactly := none
  traceExactly := behavior.traceExactly.map fun exact =>
    { exact with trace := { exact.trace with steps := [] } }
}

/-- A trace whose only content is its action schedule, for asking the action-sequence constraints
about that schedule. The Scenario it is asked of has no roles, setup or exact trace, so nothing else
of the trace is read. -/
private def scheduleTrace (schedule : List DefinitionId) : Scenario.Trace :=
  let placeholder (id : DefinitionId) : ModelValue := ModelValue.named id ""
  {
    setup := []
    trace := {
      initialState := placeholder (DefinitionId.of "")
      steps := schedule.map fun action =>
        ModelTraceStep.result (placeholder action)
          { outcome := placeholder action, state := placeholder action, facts := [] }
    }
  }

private def pinnedSchedule (behavior : CheckedScenario) : Option Schedule :=
  let steps := behavior.traceExactly.map (·.trace.steps)
  let schedule := behavior.actionsExactly <|>
    steps.map (·.map fun step => step.selectedAction.definitionId)
  schedule.map fun schedule =>
    let actionScenario : CheckedScenario :=
      { behavior with roles := [], setup := [], traceExactly := none }
    .pinned behavior.actionsExactly steps (actionScenario.admits (scheduleTrace schedule))

private def counters (behavior : CheckedScenario) : List Counter :=
  let actions := DefinitionId.canonicalSet
    (behavior.occurrenceBounds.map (·.action) ++ behavior.requiredOccurrences.map (·.action))
  actions.map fun action =>
    let bounds := behavior.occurrenceBounds.filter (·.action == action)
    let required := (behavior.requiredOccurrences.filter (·.action == action)).length
    {
      action
      minimum := bounds.foldl (fun minimum bound => Nat.max minimum bound.minimum) required
      maximum := bounds.foldl (fun maximum bound =>
        match maximum, bound.maximum with
        | some current, some next => some (Nat.min current next)
        | current, next => current <|> next) none
    }

/-- Lower a checked Scenario to its progress automaton, or name the construct version one does not
encode. `ordering` and `adjacencies` are encoded only under a pinned schedule, where they are
decided once; over a free schedule they are `Unsupported`, `ordering` reported first. -/
def ScenarioAutomaton.lower (behavior : CheckedScenario) : Except Unsupported ScenarioAutomaton :=
  let unsupported (construct : ScenarioConstruct) : Except Unsupported ScenarioAutomaton :=
    .error { scenarioId := behavior.id, source := behavior.source, construct }
  match pinnedSchedule behavior with
  | some schedule => .ok ⟨rootScenarioOf behavior, schedule⟩
  | none =>
      if !behavior.ordering.isEmpty then
        unsupported .ordering
      else if !behavior.adjacencies.isEmpty then
        unsupported .adjacencies
      else
        .ok ⟨rootScenarioOf behavior, .free behavior.allowedActions behavior.forbiddenActions
          (counters behavior) behavior.sequences⟩

/-- The progress at a root with this setup and initial state, or `none` when the Scenario admits no
trace starting there. -/
def ScenarioAutomaton.start
    (automaton : ScenarioAutomaton)
    (setup : List RoleBinding)
    (initialState : ModelValue) : Option Progress :=
  if !automaton.rootScenario.admits (rootTrace setup initialState) then
    none
  else
    match automaton.schedule with
    | .pinned _ _ admitted => if admitted then some {} else none
    | .free _ _ counters sequences =>
        some { counts := counters.map fun _ => 0, sequences := sequences.map fun _ => 0 }

private def advanceCount (action : DefinitionId) (counter : Counter) (count : Nat) : Option Nat :=
  if counter.action != action then
    some count
  else if counter.maximum.any (count + 1 > ·) then
    none
  else
    some (Nat.min (count + 1) counter.cap)

private def advanceSequence (action : DefinitionId) (sequence : List DefinitionId) (matched : Nat) :
    Nat :=
  if sequence[matched]? == some action then matched + 1 else matched

/-- Advance the progress by one transition, or `none` when no admitted trace continues this way. -/
def ScenarioAutomaton.step
    (automaton : ScenarioAutomaton)
    (progress : Progress)
    (transition : Transition) : Option Progress :=
  let action := transition.action.definitionId
  match automaton.schedule with
  | .pinned actions steps _ =>
      let onSchedule := actions.all fun actions => actions[progress.position]? == some action
      let onTrace := steps.all fun steps => steps[progress.position]? == some transition.toTraceStep
      if onSchedule && onTrace then some { progress with position := progress.position + 1 }
      else none
  | .free allowed forbidden counters sequences =>
      if (!allowed.isEmpty && !allowed.contains action) || forbidden.contains action then
        none
      else do
        let counts ← (counters.zip progress.counts).mapM fun (counter, count) =>
          advanceCount action counter count
        pure { progress with
          counts
          sequences := (sequences.zip progress.sequences).map fun (sequence, matched) =>
            advanceSequence action sequence matched }

/-- Whether a trace that reached this progress is admitted. -/
def ScenarioAutomaton.accepts (automaton : ScenarioAutomaton) (progress : Progress) : Bool :=
  match automaton.schedule with
  | .pinned actions steps _ =>
      actions.all (·.length == progress.position) && steps.all (·.length == progress.position)
  | .free _ _ counters sequences =>
      (counters.zip progress.counts).all (fun (counter, count) => count ≥ counter.minimum) &&
        (sequences.zip progress.sequences).all fun (sequence, matched) =>
          matched == sequence.length

/-- Advance from `progress` along `path`, stopping at the first transition no admitted trace
takes. -/
def ScenarioAutomaton.runFrom
    (automaton : ScenarioAutomaton)
    (progress : Progress)
    (path : List Transition) : Option Progress :=
  path.foldlM automaton.step progress

/-- Run the automaton over a whole trace: its root, then every step. -/
def ScenarioAutomaton.run (automaton : ScenarioAutomaton) (trace : Scenario.Trace) :
    Option Progress :=
  (automaton.start trace.setup trace.trace.initialState).bind fun progress =>
    automaton.runFrom progress (trace.trace.steps.map Transition.ofTraceStep)

/-- The automaton's verdict on a whole trace. For every Scenario that lowers, this is
`CheckedScenario.admits` (see the module header for the evidence). -/
def ScenarioAutomaton.admits (automaton : ScenarioAutomaton) (trace : Scenario.Trace) : Bool :=
  (automaton.run trace).any automaton.accepts

end Umpire.Search.Product
