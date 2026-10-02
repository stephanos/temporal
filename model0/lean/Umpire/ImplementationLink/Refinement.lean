import Umpire.ImplementationLink.Language

/-!
# Refinement: a forward simulation that may stutter

A refinement relates two machines of one feature: a detailed one, whose state carries what the
implementation keeps, and a simpler one, whose state is what an operation means. The detailed
machine's state reads as the simpler one's through a map, and every step of the detailed machine is
either a step of the simpler one from the mapped state or a **stutter** -- a transition the simpler
machine does not see, because the mapped states before and after are equal.

That is the bounded forward simulation `StepPreservation` states, with two allowances an
Implementation Link does not make: a step may stutter, and the simpler machine may record fewer
observations than the detailed one. Actions, outcomes and observations map by name, so the map is
partial on them, and a step that stutters needs no counterpart at all. What the simulation buys is
the same: `traceForward` carries every admitted trace of the detailed machine to an admitted trace
of the simpler one, so a Property established on the simpler machine's paths holds on the detailed
machine's paths read through the map.

A refinement is not an Implementation Link. SEM-08 reserves that name for connecting a feature
model to the platform's system model; a refinement connects two machines of one feature, and it
carries no Link-owned declaration, coverage or fingerprint. What it reuses is the simulation.
-/

namespace Umpire

/-- How a refining machine's values read as the refined machine's: the setup and the state totally,
through the authored map, and each outcome and observation by name, `none` where the refined machine
has no value of that name. -/
structure RefinementMorphism
    (SourceSetup SourceState SourceOutcome SourceObservation : Type)
    (DestinationSetup DestinationState DestinationOutcome DestinationObservation : Type) where
  mapSetup : SourceSetup → DestinationSetup
  mapState : SourceState → DestinationState
  mapOutcome : SourceOutcome → Option DestinationOutcome
  mapObservation : SourceObservation → Option DestinationObservation

variable {SourceSetup SourceState SourceAction SourceOutcome SourceObservation : Type}
  {DestinationSetup DestinationState DestinationAction DestinationOutcome
    DestinationObservation : Type}

namespace RefinementMorphism

variable (morphism : RefinementMorphism SourceSetup SourceState SourceOutcome SourceObservation
  DestinationSetup DestinationState DestinationOutcome DestinationObservation)

/-- The observations of a source step the destination can see. -/
def mapFacts (facts : List SourceObservation) : List DestinationObservation :=
  facts.filterMap morphism.mapObservation

/-- One destination step carries one source result when it reaches the mapped state with the mapped
outcome and records nothing the source did not. The destination may record less: a protocol
machine that writes a Started event before a completion the product machine writes alone is still
the same completion. -/
def Carries
    (result : Step SourceState SourceOutcome SourceObservation)
    (carried : Step DestinationState DestinationOutcome DestinationObservation) : Prop :=
  carried.state = morphism.mapState result.state ∧
    morphism.mapOutcome result.outcome = some carried.outcome ∧
    ∀ fact ∈ carried.facts, fact ∈ morphism.mapFacts result.facts

/-- The destination states a source trace passes through, stutters dropped. -/
def visibleStates [DecidableEq DestinationState] :
    SourceState → List (ModelTraceStep SourceState SourceAction SourceOutcome SourceObservation) →
      List DestinationState
  | _, [] => []
  | state, step :: rest =>
      if morphism.mapState state = morphism.mapState step.state then
        visibleStates step.state rest
      else
        morphism.mapState step.state :: visibleStates step.state rest

theorem visibleStates_cons_of_eq [DecidableEq DestinationState] {state : SourceState}
    {step : ModelTraceStep SourceState SourceAction SourceOutcome SourceObservation}
    {rest : List (ModelTraceStep SourceState SourceAction SourceOutcome SourceObservation)}
    (equal : morphism.mapState state = morphism.mapState step.state) :
    morphism.visibleStates state (step :: rest) = morphism.visibleStates step.state rest := by
  rw [visibleStates, if_pos equal]

theorem visibleStates_cons_of_ne [DecidableEq DestinationState] {state : SourceState}
    {step : ModelTraceStep SourceState SourceAction SourceOutcome SourceObservation}
    {rest : List (ModelTraceStep SourceState SourceAction SourceOutcome SourceObservation)}
    (unequal : ¬ morphism.mapState state = morphism.mapState step.state) :
    morphism.visibleStates state (step :: rest) =
      morphism.mapState step.state :: morphism.visibleStates step.state rest := by
  rw [visibleStates, if_neg unequal]

end RefinementMorphism

/-- How the destination accounts for one source transition: as a destination step from the mapped
state carrying it, or as a stutter, which the destination does not see because the mapped states
are equal. -/
inductive RefinedStep
    (destination : Machine DestinationSetup DestinationState DestinationAction DestinationOutcome
      DestinationObservation)
    (morphism : RefinementMorphism SourceSetup SourceState SourceOutcome SourceObservation
      DestinationSetup DestinationState DestinationOutcome DestinationObservation)
    (state : SourceState)
    (result : Step SourceState SourceOutcome SourceObservation) : Prop where
  | step (action : DestinationAction)
      (carried : Step DestinationState DestinationOutcome DestinationObservation)
      (carries : morphism.Carries result carried)
      (admitted : destination.authoritativeStep (morphism.mapState state) action carried)
  | stutter (equal : morphism.mapState state = morphism.mapState result.state)

/-- Initial and step preservation for one pair of Umpire transition kernels, where a step may
stutter. The obligations are `StepPreservation`'s read through the partial morphism. -/
structure StutteringSimulation
    (source : Machine SourceSetup SourceState SourceAction SourceOutcome SourceObservation)
    (destination : Machine DestinationSetup DestinationState DestinationAction
      DestinationOutcome DestinationObservation) where
  morphism : RefinementMorphism SourceSetup SourceState SourceOutcome SourceObservation
    DestinationSetup DestinationState DestinationOutcome DestinationObservation
  initialForward : ∀ setup state,
    source.authoritativeInitial setup state →
      destination.authoritativeInitial (morphism.mapSetup setup) (morphism.mapState state)
  stepForward : ∀ state action result,
    source.authoritativeStep state action result → RefinedStep destination morphism state result

namespace StutteringSimulation

variable {source : Machine SourceSetup SourceState SourceAction SourceOutcome SourceObservation}
  {destination : Machine DestinationSetup DestinationState DestinationAction
    DestinationOutcome DestinationObservation}

/-- Every admitted run of source steps is carried to admitted destination steps through the states
the destination sees. A step the destination also sees as a stutter is dropped: what remains is the
mapped path with every repetition of a mapped state removed. -/
theorem stepsForward [DecidableEq DestinationState]
    (simulation : StutteringSimulation source destination)
    (state : SourceState)
    (steps : List (ModelTraceStep SourceState SourceAction SourceOutcome SourceObservation))
    (admitted : AuthoritativeTraceSteps source state steps) :
    ∃ carried : List (ModelTraceStep DestinationState DestinationAction DestinationOutcome
        DestinationObservation),
      AuthoritativeTraceSteps destination (simulation.morphism.mapState state) carried ∧
        carried.map (·.state) = simulation.morphism.visibleStates state steps := by
  induction steps generalizing state with
  | nil => exact ⟨[], trivial, rfl⟩
  | cons step rest induction =>
      obtain ⟨first, tail⟩ := admitted
      obtain ⟨carriedRest, restAdmitted, restStates⟩ := induction step.state tail
      by_cases equal : simulation.morphism.mapState state = simulation.morphism.mapState step.state
      · refine ⟨carriedRest, ?_, ?_⟩
        · rw [equal]
          exact restAdmitted
        · rw [simulation.morphism.visibleStates_cons_of_eq equal, restStates]
      · rcases simulation.stepForward state step.selectedAction _ first with
          ⟨action, carriedStep, carries, stepAdmitted⟩ | stuttered
        · let carriedFirst : ModelTraceStep DestinationState DestinationAction DestinationOutcome
              DestinationObservation :=
            { selectedAction := action, outcome := carriedStep.outcome,
              state := carriedStep.state, facts := carriedStep.facts }
          refine ⟨carriedFirst :: carriedRest, ⟨stepAdmitted, ?_⟩, ?_⟩
          · show AuthoritativeTraceSteps destination carriedStep.state carriedRest
            rw [carries.1]
            exact restAdmitted
          · rw [simulation.morphism.visibleStates_cons_of_ne equal, List.map_cons, restStates]
            show carriedStep.state :: _ = _
            rw [carries.1]
        · exact (equal stuttered).elim

/-- Trace preservation, derived from the simulation's initial and step laws: an admitted source
trace has an admitted destination trace through the states the destination sees. -/
theorem traceForward [DecidableEq DestinationState]
    (simulation : StutteringSimulation source destination)
    (setup : SourceSetup)
    (trace : ModelTrace SourceState SourceAction SourceOutcome SourceObservation)
    (admitted : AuthoritativeModelTrace source setup trace) :
    ∃ carried : List (ModelTraceStep DestinationState DestinationAction DestinationOutcome
        DestinationObservation),
      AuthoritativeModelTrace destination (simulation.morphism.mapSetup setup)
        { initialState := simulation.morphism.mapState trace.initialState, steps := carried } ∧
        carried.map (·.state) =
          simulation.morphism.visibleStates trace.initialState trace.steps := by
  obtain ⟨carried, steps, states⟩ :=
    simulation.stepsForward trace.initialState trace.steps admitted.steps
  exact ⟨carried,
    { initial := simulation.initialForward setup trace.initialState admitted.initial, steps },
    states⟩

end StutteringSimulation

/-- An exact simulation is a stuttering one that never stutters: every value maps, and every
step is carried by the step it maps to. -/
def StepPreservation.toStutteringSimulation
    {source : Machine SourceSetup SourceState SourceAction SourceOutcome SourceObservation}
    {destination : Machine DestinationSetup DestinationState DestinationAction
      DestinationOutcome DestinationObservation}
    (simulation : StepPreservation source destination) :
    StutteringSimulation source destination := {
  morphism := {
    mapSetup := simulation.morphism.mapSetup
    mapState := simulation.morphism.mapState
    mapOutcome := fun outcome => some (simulation.morphism.mapOutcome outcome)
    mapObservation := fun fact => some (simulation.morphism.mapObservation fact) }
  initialForward := simulation.initialForward
  stepForward := fun state action result admitted =>
    .step (simulation.morphism.mapAction action) (simulation.morphism.mapStep result)
      ⟨rfl, rfl, by
        intro fact member
        simpa [RefinementMorphism.mapFacts, ValueTranslation.mapStep, Step.map] using member⟩
      (simulation.stepForward state action result admitted)
}

/-! ### The same simulation over finite tables

A machine declared by rows is a `FiniteTable`, and its kernel is derived from the rows. The
refinement is decided over the rows -- `FiniteTable.refines` is a `Bool` the kernel evaluates --
and `TableRefinement.ofChecked` turns that `true` into the simulation's obligations over the
tables, which `TableRefinement.simulation` carries to the kernels the tables derive. A Model file
writes none of this: the command that declares a refining machine synthesizes the witness by
`decide`. -/

/-- One source result is accounted for by the destination table: a row from the mapped state has a
result carrying it, or the mapped states are equal. -/
def FiniteTable.carries [DecidableEq DestinationState] [DecidableEq DestinationOutcome]
    [DecidableEq DestinationObservation]
    (destination : FiniteTable DestinationSetup DestinationState DestinationAction
      DestinationOutcome DestinationObservation)
    (morphism : RefinementMorphism SourceSetup SourceState SourceOutcome SourceObservation
      DestinationSetup DestinationState DestinationOutcome DestinationObservation)
    (state : SourceState)
    (result : Step SourceState SourceOutcome SourceObservation) : Bool :=
  decide (morphism.mapState state = morphism.mapState result.state) ||
    destination.transitions.any fun row =>
      decide (row.source = morphism.mapState state) &&
        row.results.any fun carried =>
          decide (carried.state = morphism.mapState result.state) &&
            decide (morphism.mapOutcome result.outcome = some carried.outcome) &&
            carried.facts.all fun fact => decide (fact ∈ morphism.mapFacts result.facts)

/-- Whether the destination table refines the source table through the morphism: every source
initial state maps to a destination initial state of the mapped setup, and every source result is
carried or stutters. -/
def FiniteTable.refines [DecidableEq DestinationSetup] [DecidableEq DestinationState]
    [DecidableEq DestinationOutcome] [DecidableEq DestinationObservation]
    (source : FiniteTable SourceSetup SourceState SourceAction SourceOutcome SourceObservation)
    (destination : FiniteTable DestinationSetup DestinationState DestinationAction
      DestinationOutcome DestinationObservation)
    (morphism : RefinementMorphism SourceSetup SourceState SourceOutcome SourceObservation
      DestinationSetup DestinationState DestinationOutcome DestinationObservation) : Bool :=
  (source.initial.all fun row => row.states.all fun state =>
    destination.initial.any fun carried =>
      decide (carried.setup = morphism.mapSetup row.setup) &&
        decide (morphism.mapState state ∈ carried.states)) &&
  source.transitions.all fun row => row.results.all fun result =>
    destination.carries morphism row.source result

/-- The simulation's obligations, stated over the tables' rows. -/
structure TableRefinement
    (source : FiniteTable SourceSetup SourceState SourceAction SourceOutcome SourceObservation)
    (destination : FiniteTable DestinationSetup DestinationState DestinationAction
      DestinationOutcome DestinationObservation)
    (morphism : RefinementMorphism SourceSetup SourceState SourceOutcome SourceObservation
      DestinationSetup DestinationState DestinationOutcome DestinationObservation) : Prop where
  initialForward : ∀ row ∈ source.initial, ∀ state ∈ row.states,
    ∃ carried ∈ destination.initial,
      carried.setup = morphism.mapSetup row.setup ∧ morphism.mapState state ∈ carried.states
  stepForward : ∀ row ∈ source.transitions, ∀ result ∈ row.results,
    morphism.mapState row.source = morphism.mapState result.state ∨
      ∃ carrier ∈ destination.transitions, carrier.source = morphism.mapState row.source ∧
        ∃ carried ∈ carrier.results, morphism.Carries result carried

namespace TableRefinement

variable {source : FiniteTable SourceSetup SourceState SourceAction SourceOutcome SourceObservation}
  {destination : FiniteTable DestinationSetup DestinationState DestinationAction
    DestinationOutcome DestinationObservation}
  {morphism : RefinementMorphism SourceSetup SourceState SourceOutcome SourceObservation
    DestinationSetup DestinationState DestinationOutcome DestinationObservation}

/-- The decided check is the obligations: what `FiniteTable.refines` evaluates to `true` on holds. -/
theorem ofChecked [DecidableEq DestinationSetup] [DecidableEq DestinationState]
    [DecidableEq DestinationOutcome] [DecidableEq DestinationObservation]
    (checked : source.refines destination morphism = true) :
    TableRefinement source destination morphism := by
  simp only [FiniteTable.refines, FiniteTable.carries, Bool.and_eq_true, List.all_eq_true,
    List.any_eq_true, Bool.or_eq_true, decide_eq_true_eq] at checked
  obtain ⟨initial, steps⟩ := checked
  refine ⟨fun row rowMember state stateMember => ?_, fun row rowMember result resultMember => ?_⟩
  · obtain ⟨carried, carriedMember, setupEq, stateMember⟩ := initial row rowMember state stateMember
    exact ⟨carried, carriedMember, setupEq, stateMember⟩
  · rcases steps row rowMember result resultMember with equal |
      ⟨carrier, carrierMember, sourceEq, carried, carriedMember, ⟨stateEq, outcomeEq⟩, facts⟩
    · exact .inl equal
    · exact .inr ⟨carrier, carrierMember, sourceEq, carried, carriedMember, stateEq, outcomeEq, facts⟩

/-- The obligations over the rows are the simulation between the kernels the rows derive. -/
def simulation [DecidableEq SourceSetup] [DecidableEq SourceState] [DecidableEq SourceAction]
    [DecidableEq SourceOutcome] [DecidableEq SourceObservation]
    [DecidableEq DestinationSetup] [DecidableEq DestinationState] [DecidableEq DestinationAction]
    [DecidableEq DestinationOutcome] [DecidableEq DestinationObservation]
    (refinement : TableRefinement source destination morphism)
    (validatedSource : CheckedTable SourceSetup SourceState SourceAction SourceOutcome
      SourceObservation)
    (validatedDestination : CheckedTable DestinationSetup DestinationState DestinationAction
      DestinationOutcome DestinationObservation)
    (sourceTable : validatedSource.table = source)
    (destinationTable : validatedDestination.table = destination)
    (sourceMetadata destinationMetadata : MachineMetadata) :
    StutteringSimulation (validatedSource.machine sourceMetadata).kernel
      (validatedDestination.machine destinationMetadata).kernel := {
  morphism
  initialForward := fun setup state admitted => by
    rw [FiniteMachine.kernel_authoritativeInitial_iff, CheckedTable.machine_initialStates_mem,
      destinationTable]
    rw [FiniteMachine.kernel_authoritativeInitial_iff, CheckedTable.machine_initialStates_mem,
      sourceTable] at admitted
    obtain ⟨row, rowMember, setupEq, stateMember⟩ := admitted
    obtain ⟨carried, carriedMember, carriedSetup, carriedState⟩ :=
      refinement.initialForward row rowMember state stateMember
    exact ⟨carried, carriedMember, by rw [carriedSetup, setupEq], carriedState⟩
  stepForward := fun state action result admitted => by
    rw [FiniteMachine.kernel_authoritativeStep_iff, CheckedTable.machine_steps_mem,
      sourceTable] at admitted
    obtain ⟨row, rowMember, sourceEq, _, resultMember⟩ := admitted
    rcases refinement.stepForward row rowMember result resultMember with equal |
      ⟨carrier, carrierMember, carrierSource, carried, carriedMember, carries⟩
    · exact .stutter (by rw [← sourceEq]; exact equal)
    · refine .step carrier.action carried carries ?_
      rw [FiniteMachine.kernel_authoritativeStep_iff, CheckedTable.machine_steps_mem,
        destinationTable]
      exact ⟨carrier, carrierMember, by rw [carrierSource, sourceEq], rfl, carriedMember⟩
}

end TableRefinement

end Umpire
