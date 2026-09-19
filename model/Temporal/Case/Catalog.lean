import Temporal.Case.EventKind
import Testpilot.Protocol
import Umpire.Command

/-!
# Resolving an `evidence:` name against the realization's catalog

Evidence names recorded data that confirms a step. `Umpire` stores the name and asks a platform
whether the realization already carries it; this module is Temporal's answer, and what it admits is
what a Temporal Run actually records.

Two catalogs, because a Run produces two kinds of recorded data.

The history event kinds are the `attributes` oneof of the generated `HistoryEvent`, read off the
schema by `Temporal.Case.EventKind` rather than listed anywhere. They are what a Case reads back
through `GetWorkflowExecutionHistory`.

The Run Event kinds are the Testpilot protocol's own `RunEventKind`, which records what the harness
did rather than what the server recorded: `faultInjected` is one, and `DESIGN.md` section 3 names it
as the evidence of an injected worker outage. Its names are read off the generated enum's
descriptor, which is an `IO` value -- so they are read once, here, in the `initialize` block that
installs the check, and the check closes over the result.

A name in neither catalog is not thereby wrong: it may be a derived observation, such as a value
read back through a call, and those are declared with an `observation` command. The command asks the
registry first and this module only about what nothing declared.
-/

namespace Temporal.Case.Catalog

open Protobuf.Reflection

/-- The prefix the generated enum gives every Run Event kind. -/
private def runEventPrefix : String := "RUN_EVENT_KIND_"

/-- The zero value names no event, so nothing may be evidence of it. -/
private def runEventUnspecified : String := runEventPrefix ++ "UNSPECIFIED"

private def capitalizeFirst (segment : String) : String :=
  match segment.toList with
  | [] => segment
  | first :: rest => String.ofList (first.toUpper :: rest)

/-- `RUN_EVENT_KIND_FAULT_INJECTED` reads `faultInjected`, the way a history event kind's field name
reads. A value that does not carry the prefix keeps its whole name, so an unexpected shape stays
visible rather than silently losing characters. -/
def spelling (value : String) : String :=
  let stem := if value.startsWith runEventPrefix
    then (value.drop runEventPrefix.length).toString
    else value
  match stem.toLower.splitOn "_" with
  | [] => stem
  | head :: rest => head ++ String.join (rest.map capitalizeFirst)

/-- Every Run Event kind an `evidence:` line may name, read off the generated enum. -/
def runEventKinds : IO (List String) := do
  let values ← (enumDescriptor temporal.server.api.testpilot.v1.RunEventKind).values
  values.toList.filterMapM fun value => do
    match ← value.name with
    | some named => pure (if named == runEventUnspecified then none else some (spelling named))
    | none => pure none

/-- The verdict on one evidence name, against a catalog already read. -/
def check (runEvents : List String) (observed : String) : Except String Unit :=
  if (Temporal.Case.EventKind.admitted.contains observed) || runEvents.contains observed then
    .ok ()
  else
    .error s!"'{observed}' is neither a recorded event kind the realization carries nor an \
observation this Model declares; evidence names recorded data, so it is a generated history event \
kind, a Testpilot Run Event kind, or a derived `observation`"

initialize do
  let runEvents ← runEventKinds
  Umpire.Command.installCatalogCheck (check runEvents)

end Temporal.Case.Catalog
