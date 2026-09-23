import Lean
import Umpire.Case.Compiler

/-!
# What the renderer knows about the Cases that exist

Two environment extensions: a `case` block records the declaration that produces each of its Cases,
a functional set's in one and a canary set's in the other, so the renderer enumerates every
checked-in Case, and every admitted canary Case, without a table anyone maintains. The two are kept
apart because a canary Case is rendered only on request, for the canary that pins it, and never as
a fixture of the functional suites. Each holds names and plain data only -- never a closure --
because it is written to the `.olean` and read back in another module's elaboration.

What a `case` block needs to know about the Scenario and Query it names lives in
`Umpire.Command.Registry`, because those are the Model commands' own declarations.
-/

namespace Temporal.Case.Registry

open Lean

/-- One registered Case: the declaration that produces it, and the identities it carries. -/
structure CaseEntry where
  declName : Name
  caseId : String
  fixture : String
  deriving Inhabited, Repr, BEq

private def collect (imported : Array (Array CaseEntry)) : Array CaseEntry :=
  imported.foldl (init := #[]) (· ++ ·)

initialize caseExtension : SimplePersistentEnvExtension CaseEntry (Array CaseEntry) ←
  registerSimplePersistentEnvExtension {
    addEntryFn := Array.push
    addImportedFn := collect
  }

def recordCase (entry : CaseEntry) : CoreM Unit :=
  modifyEnv fun env => caseExtension.addEntry env entry

def cases (env : Environment) : Array CaseEntry := caseExtension.getState env

initialize canaryExtension : SimplePersistentEnvExtension CaseEntry (Array CaseEntry) ←
  registerSimplePersistentEnvExtension {
    addEntryFn := Array.push
    addImportedFn := collect
  }

/-- Record an admitted canary Case: produced with no white-box gap, rendered only on request. -/
def recordCanary (entry : CaseEntry) : CoreM Unit :=
  modifyEnv fun env => canaryExtension.addEntry env entry

def canaries (env : Environment) : Array CaseEntry := canaryExtension.getState env

/-- One registered Case as the renderer consumes it: its identities beside the compiled bytes. -/
structure Materialized where
  caseId : String
  fixture : String
  value : Except Umpire.Case.Compiler.Error temporal.server.api.testpilot.v1.Case


private def duplicateFixtureMessage (fixture first second : String) : String :=
  s!"fixture '{fixture}' is registered twice: by '{first}' and by '{second}'"

private def duplicateCaseMessage (caseId first second : String) : String :=
  s!"Case ID '{caseId}' is registered twice: by '{first}' and by '{second}'"

open Elab Term in
/-- Materialize registered Cases, sorted by Case ID. The registry holds names only, so this is where
those names become the values the renderer enumerates -- and where a fixture or a Case ID
registered twice is rejected, because nothing downstream could tell the two apart. -/
private def materialize (entries : Array CaseEntry) : TermElabM Expr := do
  let registered := entries.qsort (·.caseId < ·.caseId)
  for index in [0 : registered.size] do
    for later in [index + 1 : registered.size] do
      let one := registered[index]!
      let other := registered[later]!
      if one.fixture == other.fixture then
        throwError (duplicateFixtureMessage one.fixture one.declName.toString
          other.declName.toString)
      if one.caseId == other.caseId then
        throwError (duplicateCaseMessage one.caseId one.declName.toString other.declName.toString)
  let materialized ← registered.mapM fun entry => `(term|
    ({ caseId := $(quote entry.caseId), fixture := $(quote entry.fixture),
       value := $(mkIdent entry.declName) } : Temporal.Case.Registry.Materialized))
  elabTerm (← `(([$materialized,*] : List Temporal.Case.Registry.Materialized))) none

open Elab Term in
/-- Every registered functional Case, materialized. -/
elab "registeredCases%" : term => do
  materialize (cases (← getEnv))

open Elab Term in
/-- Every admitted canary Case, materialized. -/
elab "registeredCanaryCases%" : term => do
  materialize (canaries (← getEnv))

end Temporal.Case.Registry
