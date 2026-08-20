import Umpire3.Executable
import Umpire3.Property
import Umpire3.Refinement
import Umpire3.Transition
import Umpire3Tests.TemporalProduct
import Umpire3Tests.TemporalSystem
import Umpire3Tests.TemporalAPI
import Umpire3Tests.TemporalUpdate

namespace Umpire3.Tests

inductive ToggleState where
  | off
  | on
  deriving DecidableEq, Repr

inductive ToggleAction where
  | enable
  | disable
  deriving DecidableEq, Repr

def toggleStep : ToggleState → ToggleAction → ToggleState → Prop
  | .off, .enable, .on => True
  | .on, .disable, .off => True
  | _, _, _ => False

abbrev toggle : TransitionSystem where
  State := ToggleState
  Action := ToggleAction
  Initial := fun state => state = .off
  Step := toggleStep

def toggleNext : ToggleState → ToggleAction → List ToggleState
  | .off, .enable => [.on]
  | .on, .disable => [.off]
  | _, _ => []

theorem toggleNextIff (state action next) :
    next ∈ toggleNext state action ↔ toggle.Step state action next := by
  cases state <;> cases action <;> cases next <;> simp [toggleNext, toggleStep]

def executableToggle : ExecutableModel toggle where
  next := toggleNext
  next_iff := toggleNextIff

def boundedToggle : BoundedModel toggle where
  toExecutableModel := executableToggle
  initials := [.off]
  initial_iff := by
    intro state
    cases state <;> decide
  actions := [.enable, .disable]
  action_complete := by
    intro state action next _
    cases action <;> simp

example : toggle.Reachable .on := by
  refine ⟨.off, [.enable], rfl, ?_⟩
  have step : toggle.Step ToggleState.off ToggleAction.enable ToggleState.on := by trivial
  exact Runs.cons step (Runs.nil (model := toggle) ToggleState.on)

example : Safety toggle (fun state => state = .off ∨ state = .on) := by
  intro state _
  cases state <;> simp

example : boundedToggle.frontier 1 = [([.enable], .on)] := by decide

example : ([.disable], .on) ∉ boundedToggle.frontier 1 := by decide

example {history state depth}
    (member : (history, state) ∈ boundedToggle.frontier depth) :
    ∃ initial, toggle.Initial initial ∧ Runs toggle initial history state :=
  boundedToggle.frontier_sound member

def smokeScope : ExplorationScope where
  bound := { maxDepth := 4, maxResults := 100 }
  assumptions := [{ identifier := "reliable-persistence", statementHash := "sha256:example" }]

example : smokeScope.assumptions.length = 1 := rfl

example : (Refinement.identity toggle).Relates .off .off := rfl

example : ¬(Refinement.identity toggle).Stutters .off .off .enable .on := by
  intro stutters
  have related := stutters.2.2
  change ToggleState.on = ToggleState.off at related
  cases related

end Umpire3.Tests
