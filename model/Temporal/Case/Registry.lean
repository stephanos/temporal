import Lean
import Umpire.Case.Compiler

/-!
# What the authoring commands know about each other

Three environment extensions, written by the commands and read by the `case` command and by the
renderer. They hold names and plain data only -- never a closure -- because they are written to the
`.olean` and read back in another module's elaboration.

* A Scenario records the Action spellings it selects, so a `case` block can say which Action an
  `evidence` line names that the Scenario never selects, on the line that names it.
* A Query records whether it selects a witness and which Scenario it runs in, so a `case` block can
  reject a `verify` Query -- a Case realizes one selected trace -- and reach the Scenario's Actions.
* A `case` block records the declaration that produces its Case, so the renderer enumerates every
  checked-in Case without a table anyone maintains.
-/

namespace Temporal.Case.Registry

open Lean

/-- One Scenario declaration and the Action spellings it selects, in declaration order. -/
structure ScenarioEntry where
  declName : Name
  actions : Array String
  deriving Inhabited, Repr, BEq

/-- One Query declaration: whether it selects a witness, and the Scenario it runs in. -/
structure QueryEntry where
  declName : Name
  selectsWitness : Bool
  scenario : Name
  deriving Inhabited, Repr, BEq

/-- One registered Case: the declaration that produces it, and the identities it carries. -/
structure CaseEntry where
  declName : Name
  caseId : String
  fixture : String
  deriving Inhabited, Repr, BEq

private def collect {α : Type} (imported : Array (Array α)) : Array α :=
  imported.foldl (init := #[]) (· ++ ·)

initialize scenarioExtension :
    SimplePersistentEnvExtension ScenarioEntry (Array ScenarioEntry) ←
  registerSimplePersistentEnvExtension {
    addEntryFn := Array.push
    addImportedFn := collect
  }

initialize queryExtension : SimplePersistentEnvExtension QueryEntry (Array QueryEntry) ←
  registerSimplePersistentEnvExtension {
    addEntryFn := Array.push
    addImportedFn := collect
  }

initialize caseExtension : SimplePersistentEnvExtension CaseEntry (Array CaseEntry) ←
  registerSimplePersistentEnvExtension {
    addEntryFn := Array.push
    addImportedFn := collect
  }

def recordScenario (entry : ScenarioEntry) : CoreM Unit :=
  modifyEnv fun env => scenarioExtension.addEntry env entry

def recordQuery (entry : QueryEntry) : CoreM Unit :=
  modifyEnv fun env => queryExtension.addEntry env entry

def recordCase (entry : CaseEntry) : CoreM Unit :=
  modifyEnv fun env => caseExtension.addEntry env entry

def scenarios (env : Environment) : Array ScenarioEntry := scenarioExtension.getState env
def queries (env : Environment) : Array QueryEntry := queryExtension.getState env
def cases (env : Environment) : Array CaseEntry := caseExtension.getState env

def scenario? (env : Environment) (declName : Name) : Option ScenarioEntry :=
  (scenarios env).find? (·.declName == declName)

def query? (env : Environment) (declName : Name) : Option QueryEntry :=
  (queries env).find? (·.declName == declName)

/-- One registered Case as the renderer consumes it: its identities beside the compiled bytes. -/
structure Materialized where
  caseId : String
  fixture : String
  value : Except Umpire.Case.Compiler.Error temporal.server.api.testpilot.v1.Case


/-- Register a Case value no `case` block declares. The two typed examples and the two
realization-only Cases carry their identities in Lean rather than in a Model, so they say so here
and `--list` still covers every checked-in functional fixture. -/
elab "register_case" declRef:ident &"id" caseId:str &"fixture" fixture:str : command => do
  let declName ← Elab.Command.liftTermElabM
    (Lean.Elab.realizeGlobalConstNoOverloadWithInfo declRef)
  Elab.Command.liftCoreM (recordCase {
    declName, caseId := caseId.getString, fixture := fixture.getString })

private def duplicateFixtureMessage (fixture first second : String) : String :=
  s!"fixture '{fixture}' is registered twice: by '{first}' and by '{second}'"

private def duplicateCaseMessage (caseId first second : String) : String :=
  s!"Case ID '{caseId}' is registered twice: by '{first}' and by '{second}'"

open Elab Term in
/-- Materialize every registered Case, sorted by Case ID. The registry holds names only, so this is
where those names become the values the renderer enumerates -- and where a fixture or a Case ID
registered twice is rejected, because nothing downstream could tell the two apart. -/
elab "registeredCases%" : term => do
  let registered := (cases (← getEnv)).qsort (·.caseId < ·.caseId)
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
  elabTerm (← `([$materialized,*])) none

end Temporal.Case.Registry
