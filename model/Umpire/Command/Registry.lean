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
its Definition IDs off, and which namespace prefix is scaffolding rather than semantic family. A
project declares it once with `model_conventions`; a file that declares none gets an empty root and
the whole namespace as its family.
-/

namespace Umpire.Command.Registry

open Lean

/-- One Model declaration and the member spellings it declares, in declaration order. A later
command resolves a spelling against these rather than re-reading the inductives. -/
structure ModelEntry where
  declName : Name
  role : String
  /-- The declaring types, so a resolved member reference can be given the constructor it names and
  the editor can hover it and go to its definition. -/
  stateType : Name
  actionType : Name
  outcomeType : Name
  factType : Name
  states : Array String
  actions : Array String
  outcomes : Array String
  facts : Array String
  /-- The states the Model may start in, a subset of `states`. -/
  starts : Array String
  deriving Inhabited, Repr, BEq

/-- One Property declaration and the Model it runs on, so a Query need not name the Model again. -/
structure PropertyEntry where
  declName : Name
  model : Name
  deriving Inhabited, Repr, BEq

/-- One Scenario declaration and the Action spellings it selects, in declaration order. -/
structure ScenarioEntry where
  declName : Name
  model : Name
  actions : Array String
  deriving Inhabited, Repr, BEq

/-- One Query declaration: whether it selects a witness, and the Scenario it runs in. -/
structure QueryEntry where
  declName : Name
  selectsWitness : Bool
  scenario : Name
  deriving Inhabited, Repr, BEq

/-- One declared domain: an `enum`'s Definition ID, recorded where it is declared.

A domain's id hangs off the family of the namespace that declared it *and* off the conventions that
file could see, so a later command reads it from here rather than recomputing it. Recomputing it at
the reference would read the referring file's conventions, and two Models that share a domain would
disagree about its id. -/
structure DomainEntry where
  declName : Name
  /-- The author's spelling, which is what a rejection quotes. -/
  name : String
  /-- The Definition ID the declaration carries. -/
  id : String
  deriving Inhabited, Repr, BEq

/-- One declared entity: the key recorded data names an instance by, and the references it declares.
A later command resolves an entity reference against these rather than re-reading the declaration. -/
structure EntityEntry where
  declName : Name
  /-- The author's spelling, which is what a rejection quotes. -/
  name : String
  /-- The Definition ID the declaration carries. A reference reads it from here rather than deriving
  one from the referring file's own `Origin`, which would point at a name in the wrong family. -/
  id : String
  key : String
  /-- Each `refer:` line as (field, the referenced entity's declaration name). -/
  refers : Array (String × Name)
  deriving Inhabited, Repr, BEq

/-- One declared action: the party that performs it, the entity it acts on, and its input fields with
the enum each ranges over. The enum type is kept so a later command can resolve a class pattern
against the constructors the author actually declared. -/
structure ActionEntry where
  declName : Name
  name : String
  party : String
  /-- The entity the action acts on or creates, and which of the two it is. -/
  subject : Option (Name × Bool)
  /-- Each `input:` line as (field, the enum type it ranges over). -/
  inputFields : Array (String × Name)
  /-- The enum a `results:` line names, if any. -/
  results : Option Name
  deriving Inhabited, Repr, BEq

/-- One declared observation: the entity whose key finds its instance, and the field it reads. -/
structure ObservationEntry where
  declName : Name
  name : String
  entity : Name
  read : String
  deriving Inhabited, Repr, BEq

/-- What a project decided once, rather than per declaration. -/
structure Conventions where
  /-- The Definition ID root every declared family hangs off; empty for none. -/
  root : String := ""
  /-- The leading namespace components that are scaffolding, not semantic family. -/
  namespacePrefix : Name := .anonymous
  deriving Inhabited, Repr, BEq

def collect {α : Type} (imported : Array (Array α)) : Array α :=
  imported.foldl (init := #[]) (· ++ ·)

initialize modelExtension : SimplePersistentEnvExtension ModelEntry (Array ModelEntry) ←
  registerSimplePersistentEnvExtension {
    addEntryFn := Array.push
    addImportedFn := collect
  }

initialize propertyExtension : SimplePersistentEnvExtension PropertyEntry (Array PropertyEntry) ←
  registerSimplePersistentEnvExtension {
    addEntryFn := Array.push
    addImportedFn := collect
  }

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

initialize domainExtension : SimplePersistentEnvExtension DomainEntry (Array DomainEntry) ←
  registerSimplePersistentEnvExtension {
    addEntryFn := Array.push
    addImportedFn := collect
  }

initialize entityExtension : SimplePersistentEnvExtension EntityEntry (Array EntityEntry) ←
  registerSimplePersistentEnvExtension {
    addEntryFn := Array.push
    addImportedFn := collect
  }

initialize actionExtension : SimplePersistentEnvExtension ActionEntry (Array ActionEntry) ←
  registerSimplePersistentEnvExtension {
    addEntryFn := Array.push
    addImportedFn := collect
  }

initialize observationExtension :
    SimplePersistentEnvExtension ObservationEntry (Array ObservationEntry) ←
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

def recordModel (entry : ModelEntry) : CoreM Unit :=
  modifyEnv fun env => modelExtension.addEntry env entry

def recordProperty (entry : PropertyEntry) : CoreM Unit :=
  modifyEnv fun env => propertyExtension.addEntry env entry

def recordScenario (entry : ScenarioEntry) : CoreM Unit :=
  modifyEnv fun env => scenarioExtension.addEntry env entry

def recordQuery (entry : QueryEntry) : CoreM Unit :=
  modifyEnv fun env => queryExtension.addEntry env entry

def recordEntity (entry : EntityEntry) : CoreM Unit :=
  modifyEnv fun env => entityExtension.addEntry env entry

def recordDomain (entry : DomainEntry) : CoreM Unit :=
  modifyEnv fun env => domainExtension.addEntry env entry

def recordAction (entry : ActionEntry) : CoreM Unit :=
  modifyEnv fun env => actionExtension.addEntry env entry

def recordObservation (entry : ObservationEntry) : CoreM Unit :=
  modifyEnv fun env => observationExtension.addEntry env entry

def models (env : Environment) : Array ModelEntry := modelExtension.getState env
def properties (env : Environment) : Array PropertyEntry := propertyExtension.getState env
def scenarios (env : Environment) : Array ScenarioEntry := scenarioExtension.getState env
def queries (env : Environment) : Array QueryEntry := queryExtension.getState env

def domains (env : Environment) : Array DomainEntry := domainExtension.getState env

def domain? (env : Environment) (declName : Name) : Option DomainEntry :=
  (domains env).find? (·.declName == declName)

def entities (env : Environment) : Array EntityEntry := entityExtension.getState env

/-- The entities *this file* declared. A Model's rules about the names it uses -- one key per Model --
are about the file, while `entities` is the whole import closure, so two feature Models may each
declare their own `operation`. -/
def localEntities (env : Environment) : List EntityEntry := entityExtension.getEntries env
def actions (env : Environment) : Array ActionEntry := actionExtension.getState env
def observations (env : Environment) : Array ObservationEntry := observationExtension.getState env

def entity? (env : Environment) (declName : Name) : Option EntityEntry :=
  (entities env).find? (·.declName == declName)

def action? (env : Environment) (declName : Name) : Option ActionEntry :=
  (actions env).find? (·.declName == declName)

def observation? (env : Environment) (declName : Name) : Option ObservationEntry :=
  (observations env).find? (·.declName == declName)

def model? (env : Environment) (declName : Name) : Option ModelEntry :=
  (models env).find? (·.declName == declName)

def property? (env : Environment) (declName : Name) : Option PropertyEntry :=
  (properties env).find? (·.declName == declName)

def scenario? (env : Environment) (declName : Name) : Option ScenarioEntry :=
  (scenarios env).find? (·.declName == declName)

def query? (env : Environment) (declName : Name) : Option QueryEntry :=
  (queries env).find? (·.declName == declName)

/-- The conventions in force, or the defaults when none were declared. The last declaration wins, so
a project that declares them once gets them everywhere. -/
def conventions (env : Environment) : Conventions :=
  ((conventionsExtension.getState env).back?).getD {}

/-- Declare a project's conventions once. -/
elab "model_conventions" &"root" root:str &"under" namespacePrefix:ident : command =>
  Elab.Command.liftCoreM (modifyEnv fun env =>
    conventionsExtension.addEntry env {
      root := root.getString, namespacePrefix := namespacePrefix.getId })

end Umpire.Command.Registry
