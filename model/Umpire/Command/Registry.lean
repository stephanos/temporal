import Lean
import Umpire.Command.Authoring

/-!
# What the Model commands know about each other

Two environment extensions and one convention record, written by the commands and read by whatever
turns a checked Model into something runnable. They hold names and plain data only -- never a
closure -- because they are written to the `.olean` and read back in another module's elaboration.

* A Scenario records the Action spellings it selects, so a later command can say which Action a
  declaration names that the Scenario never selects, on the line that names it.
* A Query records whether it selects a witness and which Scenario it runs in, so a consumer that
  needs one selected trace can reject a `verify` Query and reach the Scenario's Actions.

`Conventions` is the one thing these commands cannot derive: which definition root a project hangs
its Definition IDs off, which namespace prefix is scaffolding rather than semantic family, and which
Known Gaps its Queries carry. A project declares it once with `model_conventions`; a file that
declares none gets an empty root, the whole namespace as its family, and no Known Gaps.
-/

namespace Umpire.Command.Registry

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

/-- What a project decided once, rather than per declaration. -/
structure Conventions where
  /-- The Definition ID root every declared family hangs off; empty for none. -/
  root : String := ""
  /-- The leading namespace components that are scaffolding, not semantic family. -/
  namespacePrefix : Name := .anonymous
  /-- A declaration of type `Except KnownGapError KnownGapSet` every Query carries; anonymous for
  none. -/
  knownGaps : Name := .anonymous
  deriving Inhabited, Repr, BEq

def collect {α : Type} (imported : Array (Array α)) : Array α :=
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

initialize conventionsExtension :
    SimplePersistentEnvExtension Conventions (Array Conventions) ←
  registerSimplePersistentEnvExtension {
    addEntryFn := Array.push
    addImportedFn := collect
  }

def recordScenario (entry : ScenarioEntry) : CoreM Unit :=
  modifyEnv fun env => scenarioExtension.addEntry env entry

def recordQuery (entry : QueryEntry) : CoreM Unit :=
  modifyEnv fun env => queryExtension.addEntry env entry

def scenarios (env : Environment) : Array ScenarioEntry := scenarioExtension.getState env
def queries (env : Environment) : Array QueryEntry := queryExtension.getState env

def scenario? (env : Environment) (declName : Name) : Option ScenarioEntry :=
  (scenarios env).find? (·.declName == declName)

def query? (env : Environment) (declName : Name) : Option QueryEntry :=
  (queries env).find? (·.declName == declName)

/-- The conventions in force, or the defaults when none were declared. The last declaration wins, so
a project that declares them once gets them everywhere. -/
def conventions (env : Environment) : Conventions :=
  ((conventionsExtension.getState env).back?).getD {}

/-- Record one project's conventions. -/
private def declareConventions (root : String) (namespacePrefix knownGaps : Name) :
    Elab.Command.CommandElabM Unit :=
  Elab.Command.liftCoreM (modifyEnv fun env =>
    conventionsExtension.addEntry env { root, namespacePrefix, knownGaps })

/-- Declare a project's conventions once, with the Known Gaps every Query carries. -/
elab "model_conventions" &"root" root:str &"under" namespacePrefix:ident
    &"gaps" gapsRef:ident : command => do
  let knownGaps ← Elab.Command.liftTermElabM
    (Lean.Elab.realizeGlobalConstNoOverloadWithInfo gapsRef)
  declareConventions root.getString namespacePrefix.getId knownGaps

/-- The same, for a project whose Queries carry no Known Gaps. -/
elab "model_conventions" &"root" root:str &"under" namespacePrefix:ident : command => do
  declareConventions root.getString namespacePrefix.getId .anonymous

end Umpire.Command.Registry
