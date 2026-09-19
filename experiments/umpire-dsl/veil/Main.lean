import DslExperiment.Model
import Veil.Core.Tools.ModelChecker.TransitionSystem

/-! The adapter uses the unchanged pinned Veil semantic API. Both consumers derive from
`DslExperiment.successors`; there is no second cancellation transition definition. -/
namespace DslExperiment.VeilProbe

/-- Veil execution success is distinct from the operation's completed/canceled label. -/
def enumerable : Veil.EnumerableTransitionSystem Unit (List Unit) World (List World)
    Int Step (List (Step × Veil.ExecutionOutcome Int World)) () where
  initStates := [initial]
  tr := fun _ world => (successors world).map fun (label, next) => (label, .success next)

/-- Optional-checker view retains each correlated semantic event in its label. -/
def optional : Veil.RelationalTransitionSystem Unit World Step := enumerable.toRelational

/-- Shared-core view derives the relation directly from the same finite transition authority. -/
def shared : Veil.RelationalTransitionSystem Unit World Step where
  assumptions := fun _ => True
  init := fun _ world => world = initial
  tr := fun _ before label after => (label, after) ∈ successors before

/-- Every optional adapter edge denotes exactly a shared-core edge, in both directions. -/
theorem transition_equivalence (before after : World) (label : Step) :
    optional.tr () before label after ↔ shared.tr () before label after := by
  simp [optional, shared, enumerable, Veil.EnumerableTransitionSystem.toRelational]

/-- Both interpretations admit exactly the same initial state. -/
theorem initial_equivalence (world : World) : optional.init () world ↔ shared.init () world := by
  simp [optional, shared, enumerable, Veil.EnumerableTransitionSystem.toRelational]

/-- Finite list evaluation and the adapted safety proposition have identical meaning. -/
def safe (world : World) : Bool := decide (world.1 ≠ .succeeded)

/-- Boolean safety is used unchanged by either backend. -/
theorem safety_equivalence (world : World) : safe world = true ↔ world.1 ≠ .succeeded := by
  simp [safe]

#print axioms transition_equivalence
#print axioms initial_equivalence
#print axioms safety_equivalence

private def importedSuccessors (world : World) : List (Step × World) :=
  (enumerable.tr () world).filterMap fun (label, outcome) =>
    match outcome with
    | .success next => some (label, next)
    | _ => none

private def layers (next : World → List (Step × World)) : Nat → List (List Step × World)
  | 0 => [([], initial)]
  | n + 1 => (layers next n).flatMap fun (trace, world) =>
    (next world).map fun (label, after) => (trace ++ [label], after)

/-- Compare all bounded labeled paths, including self-loops, without state merging. -/
def runCoreChecks : IO Unit := do
  let start ← IO.monoNanosNow
  for depth in List.range 6 do
    let reference := layers successors depth
    let adapted := layers importedSuccessors depth
    unless reference == adapted do throw (IO.userError s!"path mismatch at depth {depth}")
    unless adapted.all (fun (trace, finalState) => replay trace == some finalState) do
      throw (IO.userError "adapted path failed replay")
    IO.println s!"depth={depth} paths={reference.length} exact_replay=true"
  unless (replay [⟨0, .completed⟩]).isNone do throw (IO.userError "wrong request accepted")
  IO.println s!"adapter_path_comparison_ns={(← IO.monoNanosNow) - start}"

end DslExperiment.VeilProbe

def main : IO Unit := DslExperiment.VeilProbe.runCoreChecks
