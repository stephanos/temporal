/-! A two-operation cancellation model shared by the finite and Veil experiments.
Only confirmed semantic events advance the model; command submission belongs to the driver.
`advance` is the sole authored transition definition. -/

namespace DslExperiment

/-- An operation begins acknowledged and can resolve after cancellation confirmation. -/
inductive Phase where
  | started | requested | canceled | succeeded
  deriving Repr, DecidableEq, BEq

/-- `tick` is an admitted labeled self-loop and consumes an operation transition. -/
inductive Event where
  | requested | canceled | completed | tick
  deriving Repr, DecidableEq, BEq

/-- A correlated semantic event, independent of observation retries. -/
structure Step where
  operation : Fin 2
  event : Event
  deriving Repr, DecidableEq, BEq

/-- The two operation states have stable positional identities. -/
abbrev World := Phase × Phase

/-- Both operations have already started; start acknowledgement is outside this slice. -/
def initial : World := (.started, .started)

/-- The sole transition authority; `none` rejects an event at the current phase. -/
def advance (phase : Phase) (event : Event) : Option Phase :=
  match phase, event with
  | .started, .requested => some .requested
  | .requested, .canceled => some .canceled
  | .requested, .completed => some .succeeded
  | .requested, .tick => some .requested
  | _, _ => none

/-- Update only the operation named by a semantic step. -/
def applyStep (world : World) (step : Step) : Option World := do
  if step.operation == 0 then
    let next ← advance world.1 step.event
    pure (next, world.2)
  else
    let next ← advance world.2 step.event
    pure (world.1, next)

/-- Stable exploration order is separate from solver witness selection. -/
def alphabet : List Step :=
  [0, 1].flatMap fun operation =>
    [Event.requested, .canceled, .completed, .tick].map fun event => ⟨operation, event⟩

/-- Enumerate exactly the transitions admitted by `applyStep`. -/
def successors (world : World) : List (Step × World) :=
  alphabet.filterMap fun step => (applyStep world step).map fun next => (step, next)

/-- Exact replay rejects the first event outside the authoritative relation. -/
def replay (steps : List Step) : Option World := steps.foldlM applyStep initial

end DslExperiment
