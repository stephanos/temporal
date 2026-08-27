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
  let forbidden? := fun source destination =>
    if source == `Consumer.Root && destination == `Forbidden.Target then
      some "generic-dependency"
    else
      none
  let violations := check forbidden? #[
    moduleRecord `Consumer.Root #[`Bridge.Zed, `Bridge.Alpha, `External.Library],
    moduleRecord `Bridge.Alpha #[`Consumer.Root, `Forbidden.Target],
    moduleRecord `Bridge.Zed #[`Forbidden.Target],
    moduleRecord `Forbidden.Target
  ]
  requireEqual "generic stable transitive violation" violations #[{
    rule := "generic-dependency"
    source := `Consumer.Root
    destination := `Forbidden.Target
    path := #[`Consumer.Root, `Bridge.Alpha, `Forbidden.Target]
  }]

end Tools.LeanImportGraphTests
