import Tools.LeanImportGraph

/-! Executable regressions for the reusable Lean import-graph interface. -/

namespace Tools.LeanImportGraphTests

open Tools.LeanImportGraph

private def moduleRecord (name : Lean.Name) (imports : Array Lean.Name := #[]) : ModuleRecord :=
  { name, imports }

private def requireEqual [BEq α] [Repr α] (label : String) (actual expected : α) : IO Unit :=
  unless actual == expected do
    throw <| IO.userError s!"{label}: expected {repr expected}, got {repr actual}"

/-- Exercise deterministic transitive checking through the generic module interface. -/
def run : IO Unit := do
  let ruleSet : RuleSet String := {
    forbidden? := fun source destination =>
      if source == `Consumer.Root && destination == `Forbidden.Target then
        some "generic-dependency"
      else
        none
    ruleKey := id
  }
  let violations := check ruleSet #[
    moduleRecord `Consumer.Root #[`Bridge.Zed, `Bridge.Alpha],
    moduleRecord `Bridge.Alpha #[`Forbidden.Target],
    moduleRecord `Bridge.Zed #[`Forbidden.Target],
    moduleRecord `Forbidden.Target
  ]
  requireEqual "generic violation count" violations.size 1
  let some violation := violations[0]?
    | throw <| IO.userError "generic dependency linter returned no violation"
  requireEqual "generic rule" violation.rule "generic-dependency"
  requireEqual "generic stable shortest path" violation.path
    #[`Consumer.Root, `Bridge.Alpha, `Forbidden.Target]

end Tools.LeanImportGraphTests
