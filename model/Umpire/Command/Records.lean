import Umpire.Core
import Umpire.Model.Types

/-!
# What a Model declares about side effects

`Umpire.Command.Authoring` owns the checked planning records a Model elaborates into. This module
owns the declarations that sit in front of them: the entities a feature acts on, the actions parties
perform, the observations that confirm a step, and the per-machine declarations a step function
cannot carry (what it tracks, what ends it, its setup parameters, its timers, and what each outcome
is observed through).

These are plain data. Nothing here is checked, and nothing here names a platform concept: an RPC, a
Testpilot instruction, a recorded event kind and a dynamic config key all belong to a realization the
platform owns, which binds these declarations to them. That split is what lets one Model be read
without reading the platform it runs on, and it is why `MachineDeclaration.evidence` names an observation rather
than an event.

Values are `ModelValue`s, the same spelling Search, fingerprints and lowering already use, so a
declaration carries no second encoding of a member.
-/

namespace Umpire.Command

open Umpire

/-- One reference an entity declares: the field naming it, and the entity it points at. A reference
is compared, never interpreted, so it carries no shape beyond those two names. -/
structure EntityReference where
  field : String
  entity : DefinitionId
  deriving BEq, Repr

/-- A kind of thing with identity: an operation, a caller. It declares what it refers to and
the `key` recorded data uses to name one instance, and it declares no state -- a machine tracks
that. A Model holds several instances of each entity, bounded by Limits. -/
structure Entity where
  id : DefinitionId
  name : String
  refers : List EntityReference := []
  key : String
  source : SourceLocation
  deriving BEq, Repr

/-- One input field of an action, typed by a finite enum whose **members** are its classes.

A member, not a constructor: a constructor that carries finite fields contributes one class per
assignment of them, so `handlerError (retryable : Bool)` is one constructor and the two classes
`handlerError (retryable := false)` and `handlerError (retryable := true)`. That is the granularity
an author claims behavior at, and it is the granularity an example is written at.

Each class is a set of concrete values -- values of the realized payload, which the Model does not
enumerate -- claimed to behave alike. `classes` lists them in enumeration order, which is
constructors in declaration order and, within one, the first field varying slowest, so a class's
position is stable across a Model's lifetime. -/
structure InputField where
  name : String
  domain : DefinitionId
  classes : List ModelValue := []
  deriving BEq, Repr

/-- The concrete value a functional Case uses for one class of one action's field.

A class covers values the Model cannot count: `handlerError (retryable := false)` stands for
BadRequest, Unauthenticated, NotFound and more, and nothing in the Model says how many. So an example
is what marks a class as an abstraction -- a class an author wrote an example for is one they claim
several values behave alike in, and the example is the one a functional Case runs.

`pattern` is the class, canonical: the constructor's fields in declaration order, spelled as
`classes` spells them, so an example and the class it stands for compare as equal strings. -/
structure Example where
  action : DefinitionId
  field : String
  pattern : String
  member : ModelValue
  deriving BEq, Repr

/-- What an action does to the entity it names: acts on an existing instance, creates one, or
neither. An action with no subject is behavior that no entity records. -/
inductive ActionSubject where
  | acts (entity : DefinitionId)
  | creates (entity : DefinitionId)
  | free
  deriving BEq, Repr

/-- A side effect performed by a party. `party` is a name the feature chose (`caller`, `handler`,
`network`, `worker`); `system`, the implementation under test, is reserved and performs no declared
action.

An action has no kind. Whether it is realized as a call, a command or a reply is a realization's
decision, so two actions that differ only in how they reach the server are one declaration here.
`schema` names the protobuf message, or the alternatives, that types the action's input and results;
it is empty when the action's payload has no message, and a Model stores only the name, never the
descriptor, so a Case's identity never embeds a schema. -/
structure Action where
  id : DefinitionId
  name : String
  party : String
  subject : ActionSubject := .free
  schema : List String := []
  input : List InputField := []
  results : Option DefinitionId := none
  examples : List Example := []
  source : SourceLocation
  deriving BEq, Repr

/-- Recorded data that confirms a step. Only a *derived* observation is declared: one read back
through a call, such as an operation's attempt count. Every other evidence name resolves against the
realization's catalog, so a Model does not restate the platform's own record kinds.

`entity` is the entity whose `key` finds the instance, which may be another entity than the row's,
and `read` is the field the value comes out of, spelled as the realization's catalog understands it.
-/
structure Observation where
  id : DefinitionId
  name : String
  entity : DefinitionId
  read : String
  source : SourceLocation
  deriving BEq, Repr

/-- A timer is `system` behavior: a step function with no input, enabled while it returns a
successor. The declaration carries only its name, because when it fires is the step's decision and
what it means is the machine's evidence line. -/
structure Timer where
  id : DefinitionId
  name : String
  deriving BEq, Repr

/-- A finite parameter of a machine's setup, named after the behavior it controls
(`atConcurrencyLimit`, `recordCancelCompletion`) and bound by the Profile rather than by the Model.
`domain` names the finite enum, or the two-valued domain a `Bool` parameter ranges over. -/
structure SetupParameter where
  id : DefinitionId
  name : String
  domain : DefinitionId
  deriving BEq, Repr

/-- How a set binds one party: the Case's own Program performs the party's actions, using each
class's example, or a real deployment or the world performs them and the verifier reads which class
occurred and checks the machine allows it. -/
inductive PartyBinding where
  | driven
  | observed
  deriving BEq, Repr

def PartyBinding.name : PartyBinding → String
  | .driven => "driven"
  | .observed => "observed"

/-- What a set is for: functional sets compile each `find` Query to a checked-in Case, canary sets
are admitted for a deployment to run, and exploratory sets name a coverage goal and a budget. -/
inductive SetPurpose where
  | functional
  | canary
  | exploratory
  deriving BEq, Repr

def SetPurpose.name : SetPurpose → String
  | .functional => "functional"
  | .canary => "canary"
  | .exploratory => "exploratory"

/-- What an exploratory set covers: the machine's rows, its result values, or the members of each
claimed class. -/
inductive CoverageGoal where
  | rows
  | results
  | classMembers
  deriving BEq, Repr

def CoverageGoal.name : CoverageGoal → String
  | .rows => "rows"
  | .results => "results"
  | .classMembers => "classMembers"

/-- One thing an exploratory set sets out to reach, as an exploration reads it: a row of the
machine's table, a result value, or a member of a claimed class. Each carries the Definition IDs a
Run's evidence is compared against, so an exploration needs no second reading of the Model. -/
inductive CoverageTarget where
  | row (key : String) (state action : DefinitionId) (results : List DefinitionId)
  | result (outcome : DefinitionId)
  | classMember (member : DefinitionId) (action field className exampleValue : String)
  deriving BEq, Repr

def CoverageTarget.kind : CoverageTarget → String
  | .row .. => "row"
  | .result .. => "result"
  | .classMember .. => "classMember"

/-- One set of Queries grouped by purpose, with every party except `system` bound. `queries` are
the Queries a functional or canary set runs, `machine`, `cover` and `budget` an exploratory set's
machine, goal and limits, `targets` what that set enumerates under them, and `repeat` the switch a
functional set's Cases run once per value of. -/
structure SetDeclaration where
  id : DefinitionId
  name : String
  purpose : SetPurpose
  bindings : List (String × PartyBinding) := []
  «repeat» : Option String := none
  queries : List DefinitionId := []
  machine : Option DefinitionId := none
  cover : List CoverageGoal := []
  budget : Option String := none
  targets : List CoverageTarget := []
  source : SourceLocation
  deriving BEq, Repr

/-- How one outcome of one action class is confirmed: through a declared or catalogued observation,
optionally only when a guard over the state before the step holds, or not at all.

`unobservable` is not an omission. It becomes a Known Gap in every Case whose path uses that step, so
a behavior the platform does not record is visible in the Case rather than silently unchecked. -/
inductive EvidenceBinding where
  | observed (observation : DefinitionId) (guard : Option String := none)
  | unobservable
  deriving BEq, Repr

/-- One evidence line of a machine: the outcome of one action class, and what confirms it. `pattern`
is the class as the Model spells it, which is what a rejection quotes back. -/
structure EvidenceLine where
  action : DefinitionId
  pattern : String
  outcome : ModelValue
  binding : EvidenceBinding
  deriving BEq, Repr

/-- What a machine declares beyond its step functions. The name keeps `Declaration` because
`Umpire.Machine` is the checked transition relation this elaborates into, and both are in scope
wherever a command is elaborated.

A machine is the transition relation the glossary calls a Machine, and its logic is ordinary Lean: a
step function per action class group, enumerated at elaboration into the finite table Search,
fingerprints and lowering already read. This record carries what a step function cannot say -- which
entity it tracks, which `phase` values end an instance, what the Profile binds, which timers exist,
and what each outcome is observed through -- plus that enumerated `table`, so a consumer reads one
value and never a function. -/
structure MachineDeclaration where
  id : DefinitionId
  name : String
  entity : DefinitionId
  ends : List ModelValue := []
  setup : List SetupParameter := []
  timers : List Timer := []
  evidence : List EvidenceLine := []
  table : BehaviorTable
  source : SourceLocation
  deriving Repr

/-- The declarations one Model file contributes, in declaration order. A Model is read as a whole:
an action's `subject` names an entity declared here, an evidence line names an observation declared
here or catalogued by the realization, and a machine tracks one of these entities. -/
structure Declarations where
  entities : List Entity := []
  actions : List Action := []
  observations : List Observation := []
  machines : List MachineDeclaration := []
  deriving Repr

namespace Declarations

/-- The entity one Definition ID names, if this Model declares it. -/
def entity? (declarations : Declarations) (id : DefinitionId) : Option Entity :=
  declarations.entities.find? fun entity => entity.id == id

/-- The action one Definition ID names, if this Model declares it. -/
def action? (declarations : Declarations) (id : DefinitionId) : Option Action :=
  declarations.actions.find? fun action => action.id == id

/-- The observation one Definition ID names, if this Model declares it. An evidence name that
resolves to `none` here is a catalogued one, which the realization owns. -/
def observation? (declarations : Declarations) (id : DefinitionId) : Option Observation :=
  declarations.observations.find? fun observation => observation.id == id

/-- The machine that tracks one entity, if this Model declares one. -/
def machineFor? (declarations : Declarations) (entity : DefinitionId) : Option MachineDeclaration :=
  declarations.machines.find? fun machine => machine.entity == entity

end Declarations

/-- Every action a party performs, in declaration order. A realization binds action classes rather
than parties, so this is a reading convenience and not the binding key. -/
def Declarations.actionsOf (declarations : Declarations) (party : String) : List Action :=
  declarations.actions.filter fun action => action.party == party

/-- The class members one action's field declares, or `[]` when the action or field is not declared.
-/
def Action.classesOf (action : Action) (field : String) : List ModelValue :=
  match action.input.find? fun input => input.name == field with
  | some input => input.classes
  | none => []

/-- The example one class of one field resolves to, if the action declares one. A class the Model
cannot realize on its own -- one standing for several realized values, which is what writing an
example claims -- and no example cannot be produced, which is a Case-production rejection rather
than a declaration error: a Model is admissible whether or not every class is realized. -/
def Action.example? (action : Action) (field pattern : String) : Option Example :=
  action.examples.find? fun entry =>
    entry.action == action.id && entry.field == field && entry.pattern == pattern

/-- The evidence line for one outcome of one action class, if the machine declares one. -/
def MachineDeclaration.evidence? (machine : MachineDeclaration) (action : DefinitionId) (pattern : String)
    (outcome : ModelValue) : Option EvidenceLine :=
  machine.evidence.find? fun line =>
    line.action == action && line.pattern == pattern && line.outcome == outcome

/-- Whether a value ends an instance of the machine's entity. A step from an ending value needs no
successor, which is what keeps a terminal state from being reported as stuck. -/
def MachineDeclaration.ends? (machine : MachineDeclaration) (value : ModelValue) : Bool :=
  machine.ends.contains value

end Umpire.Command
