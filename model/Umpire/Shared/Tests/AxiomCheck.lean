import Umpire.Shared.Test

/-!
# Self-tests for `assert_axioms`

Each of the checker's three failure modes is proved by a planted violation: `#guard_msgs (error)`
only accepts the file if the wrapped `assert_axioms` genuinely fails with that message, so a
checker that silently accepted any of these would fail the build here, not just go unnoticed.
-/

namespace Umpire.Shared.Tests.AxiomCheck

-- Left unproven on purpose: its only axiom is `sorryAx`, so checking it against the ordinary
-- allowlist (which never lists `sorryAx`) must fail. The `sorry` is deliberate, not a placeholder
-- left behind, so its usual warning is silenced here rather than left for `--wfail` to trip on.
set_option warn.sorry false in
theorem seededSorry : True := by sorry

/-- outside the allowed set [propext, Classical.choice, Quot.sound] -/
#guard_msgs (error, substring := true) in
assert_axioms [seededSorry] allowing [propext, Classical.choice, Quot.sound]

/-- Proven through `Classical.em`, so its axiom inventory carries `Classical.choice`; checking it
against an allowlist that only names `propext` must fail. -/
theorem extraAxiomWitness : True ∨ ¬True :=
  Classical.em True

/-- outside the allowed set [propext] -/
#guard_msgs (error, substring := true) in
assert_axioms [extraAxiomWitness] allowing [propext]

/-- noSuchDeclarationEver -/
#guard_msgs (error, substring := true) in
assert_axioms [noSuchDeclarationEver] allowing [propext]

end Umpire.Shared.Tests.AxiomCheck
