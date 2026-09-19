import DslExperiment.Property

/-! Correlate a small typed evidence envelope before evaluating properties. Evidence identity
is scoped to a run; parent identities establish per-operation causal order. Missing parents are
retained until more evidence arrives. Only emitted model steps advance obligation clocks.
This is an executable boundary experiment, not a Temporal history decoder or a production Case. -/

namespace DslExperiment

/-- Submission grants a pending driver effect; it is not cancellation confirmation. -/
inductive EvidenceKind where
  | submitCancel
  | observed (event : Event)
  | unrelated
  | gap
  deriving Repr, DecidableEq, BEq

/-- A normalized event identity with an explicit causal predecessor when required. -/
structure Evidence where
  run : Nat
  id : Nat
  operation : Fin 2
  kind : EvidenceKind
  parent : Option Nat := none
  deriving Repr, DecidableEq, BEq

/-- Unsupported or contradictory evidence never proves a product violation by itself. -/
inductive ProjectionError where
  | conflictingIdentity | causalConflict | invalidTransition | unauthorizedInput | evidenceGap | capacity
  deriving Repr, DecidableEq, BEq

/-- Emissions retain transitive causal support, including command support for confirmation. -/
structure Emission where
  step : Step
  support : List Nat
  deriving Repr, DecidableEq, BEq

/-- Projection state is run-local and bounded; no wall clock participates in model order. -/
structure Projection where
  run : Nat
  world : World := initial
  seen : List Evidence := []
  commands : List (Fin 2 × Nat) := []
  latest : List (Fin 2 × Nat) := []
  pending : List Evidence := []
  emitted : List Emission := []
  error : Option ProjectionError := none
  deriving Repr, DecidableEq, BEq

private def keyFor (values : List (Fin 2 × Nat)) (operation : Fin 2) : Option Nat :=
  (values.find? fun pair => pair.1 == operation).map Prod.snd

private def setKey (values : List (Fin 2 × Nat)) (operation : Fin 2) (id : Nat) :
    List (Fin 2 × Nat) := (operation, id) :: values.filter (fun pair => pair.1 != operation)

private def release (projection : Projection) (evidence : Evidence) : Projection × Bool := Id.run do
  if projection.error.isSome then return (projection, false)
  let .observed event := evidence.kind | return (projection, true)
  let expected := if event == .requested then keyFor projection.commands evidence.operation
    else keyFor projection.latest evidence.operation
  if evidence.parent.isNone then
    return ({ projection with error := some .causalConflict }, true)
  if expected.isNone || expected != evidence.parent then
    let parentStillPending := projection.pending.any fun item => some item.id == evidence.parent
    let parentMissing := !(projection.seen.any fun item => some item.id == evidence.parent)
    if parentMissing || parentStillPending then return (projection, false)
    return ({ projection with error := some .causalConflict }, true)
  let step : Step := ⟨evidence.operation, event⟩
  let some world := applyStep projection.world step |
    return ({ projection with error := some .invalidTransition }, true)
  let support := if event == .requested then evidence.parent.toList ++ [evidence.id]
    else match projection.emitted.find? (fun emission => emission.support.getLast? == expected) with
      | some predecessor => predecessor.support ++ [evidence.id]
      | none => []
  if support.isEmpty then return ({ projection with error := some .causalConflict }, true)
  return ({ projection with
    world
    latest := setKey projection.latest evidence.operation evidence.id
    emitted := projection.emitted ++ [⟨step, support⟩] }, true)

private def drain : Nat → Projection → Projection
  | 0, projection => projection
  | fuel + 1, projection => Id.run do
      let mut next := projection
      let mut waiting := []
      for item in projection.pending do
        let (updated, consumed) := release next item
        next := updated
        if consumed then
          next := { next with pending := next.pending.filter fun pending => pending.id != item.id }
        else waiting := waiting ++ [item]
      next := { next with pending := waiting }
      if waiting.length == projection.pending.length then return next
      return drain fuel next

private def hasCycle (seen : List Evidence) (start : Nat) : Nat → Option Nat → Bool
  | 0, _ => false
  | _, none => false
  | fuel + 1, some parent =>
      parent == start || hasCycle seen start fuel
        ((seen.find? fun evidence => evidence.id == parent).bind Evidence.parent)

/-- Add evidence, deduplicate it, and release causally supported semantic steps atomically.
Rejecting an append preserves its prior semantic state and emissions. The 128-record bound is an
experimental work limit; hitting it yields inconclusive evidence. -/
def Projection.push (projection : Projection) (evidence : Evidence) : Projection := Id.run do
  if projection.error.isSome || evidence.run != projection.run then return projection
  if let some old := projection.seen.find? (fun item => item.id == evidence.id) then
    if old == evidence then return projection
    return { projection with error := some .conflictingIdentity }
  if projection.seen.length ≥ 128 then return { projection with error := some .capacity }
  let mut next := { projection with seen := projection.seen ++ [evidence] }
  match evidence.kind with
  | .unrelated => pure ()
  | .gap => next := { next with error := some .evidenceGap }
  | .submitCancel =>
      let phase := if evidence.operation == 0 then next.world.1 else next.world.2
      if evidence.parent.isSome then
        next := { next with error := some .causalConflict }
      else if phase != .started || (keyFor next.commands evidence.operation).isSome then
        next := { next with error := some .unauthorizedInput }
      else
        next := { next with commands := setKey next.commands evidence.operation evidence.id }
  | .observed _ =>
      if hasCycle next.seen evidence.id next.seen.length evidence.parent then
        next := { next with error := some .causalConflict }
      else next := { next with pending := next.pending ++ [evidence] }
  let drained := drain next.pending.length next
  if drained.error.isSome then
    return { projection with seen := next.seen, error := drained.error }
  return drained

/-- A fresh monitor and projection belong to one execution. -/
structure RunState where
  projection : Projection
  monitor : Monitor := {}
  deriving Repr, DecidableEq, BEq

/-- Online monitoring consumes only emissions produced by this evidence append. -/
def RunState.push (clause : ResponseClause) (state : RunState) (evidence : Evidence) : RunState :=
  let projection := state.projection.push evidence
  let fresh := projection.emitted.drop state.projection.emitted.length
  { projection, monitor := fresh.foldl (fun monitor emission =>
      monitor.consume clause emission.step) state.monitor }

/-- Projection gaps remain inconclusive; later evidence failure cannot erase a proved violation.
Accepted envelopes are immutable: a conflicting later append is rejected, not treated as a revision. -/
def RunState.finish (state : RunState) : Verdict :=
  if state.monitor.violated then .violated
  else if state.projection.error.isSome || !state.projection.pending.isEmpty then .inconclusive
  else state.monitor.finish .runtimePrefix

/-- Replay immutable evidence through the same run-local entry point used online. -/
def evaluateEvidence (clause : ResponseClause) (run : Nat) (events : List Evidence) : RunState :=
  events.foldl (RunState.push clause) { projection := { run } }

/-- Whole-stream replay and incremental append share the same state, including partial evidence. -/
theorem evidence_append (clause : ResponseClause) (run : Nat) (earlier later : List Evidence) :
    evaluateEvidence clause run (earlier ++ later) =
      later.foldl (RunState.push clause) (evaluateEvidence clause run earlier) := by
  have append (state : RunState) :
      (earlier ++ later).foldl (RunState.push clause) state =
        later.foldl (RunState.push clause) (earlier.foldl (RunState.push clause) state) := by
    induction earlier generalizing state with
    | nil => rfl
    | cons evidence rest ih => exact ih (state.push clause evidence)
  exact append { projection := { run } }

#print axioms evidence_append

end DslExperiment
