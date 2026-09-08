import Temporal.Feature.Nexus2.Race

/-! The existing cancellation race with separately identified explicit terminal semantics. -/

namespace Temporal.Feature.Nexus3.Cancellation

open Umpire

private def source : SourceLocation :=
  Temporal.Shared.sourceLocation "Temporal/Feature/Nexus3/Cancellation.lean"

/-- The already-started cancellation slice reuses Race authority with explicit terminal closure.
Completion before request confirmation and repeated requests remain outside this Target. -/
def table : FiniteTable Nexus2.Race.Setup Nexus2.Race.State Nexus2.Race.Action
    Nexus2.Race.Outcome Nexus2.Race.Fact := {
  Nexus2.Race.table with terminalConditions := [[.canceled, .succeeded]]
}

/-- Adding terminal semantics is a new Target identity, leaving both existing slices unchanged. -/
def targetDefinition : FiniteTargetDefinition := {
  Nexus2.Race.targetDefinition with
  id := .of "temporal.nexus3.cancellation.target"
  source := source
  metadata := { id := .of "temporal.nexus3.cancellation.kernel", source := source }
  definitions := Nexus2.Race.definitions.map fun definition =>
    if definition.id == Nexus2.Race.targetId then
      { definition with
        id := .of "temporal.nexus3.cancellation.target"
        source := source, canonicalBehavior := "temporal-nexus3-cancellation-terminal/v1" }
    else if definition.id == Nexus2.Race.kernelId then
      { definition with
        id := .of "temporal.nexus3.cancellation.kernel"
        source := source, canonicalBehavior := "temporal-nexus3-cancellation-terminal-kernel/v1" }
    else definition
}

/-- Check the reused authoritative rows and the new explicit terminal declaration together. -/
def targetResult : Except FiniteTargetAdmissionError (QueryTarget Nexus2.Race.LawStatement) :=
  table.checkModelTarget Nexus2.Race.identity targetDefinition Nexus2.Race.targetComposition

/-- Cancellation evidence observes exactly the existing Race transitions and their alternatives. -/
theorem transitions_eq_race : table.transitions = Nexus2.Race.table.transitions := rfl

end Temporal.Feature.Nexus3.Cancellation
