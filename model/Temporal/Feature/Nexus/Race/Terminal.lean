import Temporal.Feature.Nexus.Race.Race

/-! The existing cancellation race with separately identified explicit terminal semantics. -/

namespace Temporal.Feature.Nexus.Race.Terminal

open Umpire

private def source : SourceLocation :=
  Temporal.Shared.sourceLocation "Temporal/Feature/Nexus/Race/Terminal.lean"

/-- The already-started cancellation slice reuses Race authority with explicit terminal closure.
Completion before request confirmation and repeated requests remain outside this Target. -/
def table : FiniteTable Nexus.Race.Race.Setup Nexus.Race.Race.State Nexus.Race.Race.Action
    Nexus.Race.Race.Outcome Nexus.Race.Race.Fact := {
  Nexus.Race.Race.table with terminalConditions := [[.canceled, .succeeded]]
}

/-- Adding terminal semantics is a new Target identity, leaving both existing slices unchanged. -/
def modelSpec : TableModelSpec := {
  Nexus.Race.Race.modelSpec with
  id := .of "temporal.nexus.race.terminal.target"
  source := source
  metadata := { id := .of "temporal.nexus.race.terminal.kernel", source := source }
  definitions := Nexus.Race.Race.definitions.map fun definition =>
    if definition.id == Nexus.Race.Race.targetId then
      { definition with
        id := .of "temporal.nexus.race.terminal.target"
        source := source, behaviorVersion := "temporal-nexus-race-terminal-terminal/v1" }
    else if definition.id == Nexus.Race.Race.kernelId then
      { definition with
        id := .of "temporal.nexus.race.terminal.kernel"
        source := source, behaviorVersion := "temporal-nexus-race-terminal-terminal-kernel/v1" }
    else definition
}

/-- Check the reused authoritative rows and the new explicit terminal declaration together. -/
def targetResult : Except TableAdmissionError (QueryModel Nexus.Race.Race.LawStatement) :=
  table.checkModel Nexus.Race.Race.identity modelSpec Nexus.Race.Race.modelProviders

/-- Cancellation evidence observes exactly the existing Race transitions and their alternatives. -/
theorem transitions_eq_race : table.transitions = Nexus.Race.Race.table.transitions := rfl

end Temporal.Feature.Nexus.Race.Terminal
