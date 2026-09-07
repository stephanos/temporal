import Main

/-! A subsequent model-edit check. `run-edit.sh` compiles this file against a temporary
copy with one added transition; the shared model and adapter sources remain unchanged. -/
namespace DslExperiment.VeilProbe.EditProbe

private def completedStep : Step := ⟨0, .completed⟩
private def completedWorld : World := (.succeeded, .started)

/-- The broadened model accepts completion before cancellation confirmation. -/
theorem finite_accepts_edit : (completedStep, completedWorld) ∈ successors initial := by
  simp [successors, alphabet, applyStep, advance, initial, completedStep, completedWorld]

/-- The unchanged adapter admits the newly authored transition. -/
theorem adapter_accepts_edit : optional.tr () initial completedStep completedWorld := by
  exact (transition_equivalence initial completedWorld completedStep).mpr finite_accepts_edit

/-- The new one-step witness passes the same canonical replay function. -/
theorem edited_witness_replays : replay [completedStep] = some completedWorld := by
  decide

/-- The existing safety clause now has a one-step counterexample. -/
theorem edited_witness_violates_safety : safe completedWorld = false := by
  decide

#print axioms finite_accepts_edit
#print axioms adapter_accepts_edit
#print axioms edited_witness_replays
#print axioms edited_witness_violates_safety

#eval (do
  let start ← IO.monoNanosNow
  let finite := successors initial
  let adapted : List (Step × World) := (enumerable.tr () initial).filterMap fun (label, outcome) =>
    match outcome with
    | Veil.ExecutionOutcome.success after => some (label, after)
    | _ => none
  unless finite == adapted do throw (IO.userError "edited consumers disagree")
  unless finite.contains (completedStep, completedWorld) do
    throw (IO.userError "new completion transition absent")
  IO.println "model_edit=started+completed->succeeded authored_branches_added=1"
  IO.println "finite=true veil_adapter=true exact_replay=true safety_violation=true"
  IO.println "adapter_source_edits=0 runtime_projection=unsupported"
  IO.println s!"edit_query_ns={(← IO.monoNanosNow) - start}" : IO Unit)

end DslExperiment.VeilProbe.EditProbe
