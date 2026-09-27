import Umpire.Search.Product.Monitor

/-!
# What the Property monitors guarantee on their own

The monitors' agreement with the evaluator is tested, not proved (see
`Umpire.Search.Product.Monitor`). What holds of the monitors alone is proved here:

* a closed answer is never `unresolved`, as the evaluator's closed answer is two-valued;
* a monitor's budget never exceeds its clause Limit, at a root or after any step, so every monitor
  state is drawn from a finite set and a product over a finite Model is finite;
* a clause the evaluator requests no trigger for never sets its fired bit.
-/

namespace Umpire.Search.Product

/-- A closed ending answers every clause `satisfied` or `violated`. -/
theorem ClauseState.answer_closed_ne_unresolved (state : ClauseState) :
    state.answer false ≠ .unresolved := by
  cases state <;> simp [ClauseState.answer] <;> split <;> simp

/-- The clause Limit a monitor's budget counts down from; zero for the unbounded kinds. -/
def Monitor.bound : Monitor → Nat
  | .eventuallyWithin _ _ bound _ | .neverWithin _ _ bound _ => bound
  | _ => 0

/-- The budget a monitor state holds: the earliest pending deadline of `eventuallyWithin`, the
latest open window of `neverWithin`. -/
def ClauseState.budget? : ClauseState → Option Nat
  | .eventually _ pending _ => pending
  | .never _ window _ => window
  | _ => none

/-- A state whose budget is within `bound`. -/
def ClauseState.Within (bound : Nat) (state : ClauseState) : Prop :=
  ∀ remaining ∈ state.budget?, remaining ≤ bound

namespace Monitor

private theorem earliest_within {bound : Nat} {left right : Option Nat}
    (leftWithin : ∀ remaining ∈ left, remaining ≤ bound)
    (rightWithin : ∀ remaining ∈ right, remaining ≤ bound) :
    ∀ remaining ∈ earliest left right, remaining ≤ bound := by
  cases left with
  | none => cases right <;> simp_all [earliest]
  | some first =>
      cases right with
      | none => simp_all [earliest]
      | some second =>
          simp only [earliest, Option.mem_def, Option.some.injEq, forall_eq']
          exact Nat.le_trans (Nat.min_le_left _ _) (leftWithin first rfl)

private theorem latest_within {bound : Nat} {left right : Option Nat}
    (leftWithin : ∀ remaining ∈ left, remaining ≤ bound)
    (rightWithin : ∀ remaining ∈ right, remaining ≤ bound) :
    ∀ remaining ∈ latest left right, remaining ≤ bound := by
  cases left with
  | none => cases right <;> simp_all [latest]
  | some first =>
      cases right with
      | none => simp_all [latest]
      | some second =>
          simp only [latest, Option.mem_def, Option.some.injEq, forall_eq']
          exact Nat.max_le.mpr ⟨leftWithin first rfl, rightWithin second rfl⟩

private theorem pendingAfter_within (remaining : Nat) (here next : Bool) :
    ∀ left ∈ (pendingAfter remaining here next).join, left ≤ remaining := by
  intro left member
  simp only [pendingAfter] at member
  split at member
  · simp at member
  · split at member <;> simp at member <;> omega

private theorem bound_within {bound : Nat} {condition : Prop} [Decidable condition] :
    ∀ remaining ∈ (if condition then some bound else none), remaining ≤ bound := by
  intro remaining member
  split at member <;> simp_all

private theorem eventually_within {bound : Nat} (condition : Bool) {pending : Option Nat}
    (here : Bool) (within : ∀ remaining ∈ pending, remaining ≤ bound) :
    (if condition then ClauseState.eventually true none false
      else .eventually false pending here).Within bound := by
  cases condition
  · exact within
  · intro remaining member
    simp [ClauseState.budget?] at member

private theorem never_within {bound : Nat} (condition : Bool) {window : Option Nat}
    (here : Bool) (within : ∀ remaining ∈ window, remaining ≤ bound) :
    (if condition then ClauseState.never true none false
      else .never false window here).Within bound := by
  cases condition
  · exact within
  · intro remaining member
    simp [ClauseState.budget?] at member

theorem eventuallyStep_within {bound : Nat} {pending : Option Nat} (seen : Bool)
    (trigger response : Bool × Bool) (within : ∀ remaining ∈ pending, remaining ≤ bound) :
    (eventuallyStep bound pending seen trigger response).Within bound := by
  obtain ⟨triggerHere, triggerNext⟩ := trigger
  obtain ⟨responseHere, responseNext⟩ := response
  have old : ∀ left ∈ (pending.bind fun remaining =>
      pendingAfter remaining responseHere responseNext).join, left ≤ bound := by
    intro left member
    cases pending with
    | none => simp at member
    | some previous =>
        have := pendingAfter_within previous responseHere responseNext left (by simpa using member)
        have := within previous rfl
        omega
  have current : ∀ left ∈ (if triggerHere then
      pendingAfter bound (seen || responseHere) responseNext else none).join, left ≤ bound := by
    intro left member
    split at member
    · exact pendingAfter_within bound _ _ left member
    · simp at member
  exact eventually_within _ _ (earliest_within old (earliest_within current bound_within))

theorem neverStep_within {bound : Nat} {window : Option Nat} (seen : Bool)
    (trigger forbidden : Bool × Bool) (within : ∀ remaining ∈ window, remaining ≤ bound) :
    (neverStep bound window seen trigger forbidden).Within bound := by
  obtain ⟨triggerHere, triggerNext⟩ := trigger
  obtain ⟨forbiddenHere, forbiddenNext⟩ := forbidden
  have old : ∀ left ∈ (window.bind fun remaining =>
      if remaining ≥ 1 then some (remaining - 1) else none), left ≤ bound := by
    intro left member
    cases window with
    | none => simp at member
    | some previous =>
        have := within previous rfl
        simp only [Option.bind_some] at member
        split at member <;> simp at member
        omega
  have current : ∀ left ∈ (if (triggerHere && decide (bound ≥ 1)) = true then
      some (bound - 1) else none), left ≤ bound := by
    intro left member
    split at member <;> simp at member
    omega
  exact never_within _ _ (latest_within old (latest_within current bound_within))

end Monitor

private theorem invariantAt_budget (seen failed : Bool) (pattern : PropertyPattern)
    (values : List ModelValue) : (Monitor.invariantAt seen failed pattern values).budget? = none :=
  rfl

private theorem orderedStart_budget (before : Bool) :
    (Monitor.orderedStart before).budget? = none := rfl

private theorem orderedStep_budget (decided : Bool) (mark : Mark) (before after : Bool × Bool) :
    (Monitor.orderedStep decided mark before after).budget? = none := by
  obtain ⟨_, _⟩ := before
  obtain ⟨_, _⟩ := after
  dsimp only [Monitor.orderedStep]
  split <;> rfl

private theorem unbudgeted {bound : Nat} {state : ClauseState} (none : state.budget? = none) :
    state.Within bound := by
  intro remaining member
  rw [none] at member
  cases member

/-- A monitor starts within its clause Limit. -/
theorem Monitor.start_within (monitor : Monitor) (initial : Observed) :
    (monitor.start initial).1.Within monitor.bound := by
  cases monitor with
  | stateInvariant => exact unbudgeted (invariantAt_budget ..)
  | ordered => exact unbudgeted (orderedStart_budget _)
  | eventuallyWithin => exact bound_within
  | neverWithin => exact never_within _ _ bound_within
  | _ => exact unbudgeted rfl

/-- A monitor stays within its clause Limit across a step. -/
theorem Monitor.advance_within (monitor : Monitor) (state : ClauseState) (step : Observed)
    (within : state.Within monitor.bound) :
    (monitor.advance state step).1.Within monitor.bound := by
  cases monitor <;> cases state <;> simp only [Monitor.advance]
  all_goals first
    | exact within
    | exact unbudgeted (invariantAt_budget ..)
    | exact unbudgeted (orderedStep_budget ..)
    | exact unbudgeted rfl
    | skip
  case eventuallyWithin.eventually dead _ _ =>
    cases dead
    · exact Monitor.eventuallyStep_within _ _ _ within
    · exact unbudgeted rfl
  case neverWithin.never violated _ _ =>
    cases violated
    · exact Monitor.neverStep_within _ _ _ within
    · exact unbudgeted rfl

/-- A monitor's fired bit at a step is whether its trigger holds there, so a clause the evaluator
requests no trigger for never fires. -/
theorem Monitor.advance_fired (monitor : Monitor) (state : ClauseState) (step : Observed) :
    (monitor.advance state step).2 = monitor.trigger?.any step.holds := rfl

theorem Monitor.never_fires (monitor : Monitor) (untriggered : monitor.trigger? = none)
    (state : ClauseState) (step : Observed) (initial : Observed) :
    (monitor.start initial).2 = false ∧ (monitor.advance state step).2 = false := by
  simp [Monitor.start, Monitor.advance, untriggered]

end Umpire.Search.Product
