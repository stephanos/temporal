import Lean.Elab.Command
import Lean.Elab.ElabRules
import Umpire.Command.Catalog
import Umpire.Command.Finite
import Umpire.Command.Records
import Umpire.Command.Schema
import Umpire.Command.Registry
import Umpire.Command.Predicate
import Umpire.Command.Instances
import Umpire.Command.Refinement
import Umpire.Command.Claims
import Umpire.Command.Coverage

/-!
# The Model command grammar

Five commands -- `machine`, `property`, `scenario`, `limits`, `query` -- and their expansion into
typed `Umpire.Command` declarations. Nothing here names a feature: a project says once, through
`model_conventions`, which Definition ID root its declarations hang off and which namespace prefix
is scaffolding.

The grammar admits whatever the declaring inductives declare: the ordered state, Action, Model
Outcome and Fact domains are the constructors of the named types, in constructor order, and every
identifier a command mentions is resolved against them. There is no admissible-spelling list.
-/

namespace Umpire.Command

open Umpire
open Lean Elab Command

/-! ### The shapes a command's value can take

One rule governs a Model file: a column-0 word is a declaration kind followed by the author's name,
an indented `word:` is a framework key introducing a value, and everything else is an author name, a
declared member, a number, or an operator. Nothing an author writes is a label the framework only
carries back to them. -/

/-- One line of the retired keyed `require:` block of a Property, kept so the form is rejected at
its key with a message naming `holds:` rather than failing to parse. -/
declare_syntax_cat modelRequirement

syntax "state:" ident : modelRequirement
syntax "outcome:" ident : modelRequirement
syntax "fact:" ident : modelRequirement

/-- One Known Gap a Query carries: what kind of thing is missing, the name its code derives from,
optionally the Property it limits, and why. -/
declare_syntax_cat modelGap

syntax "gap:" ident "code:" str "subject:" str "detail:" str : modelGap
syntax "gap:" ident "code:" str "detail:" str : modelGap

/-- The last component of a constructor name, which is the spelling an author writes. -/
private def shortName : Name → Name
  | .str _ spelling => .str .anonymous spelling
  | name => name

private def spellings (constructors : List Name) : String :=
  ", ".intercalate (constructors.map fun constructor => (shortName constructor).toString)

private def unknownMemberMessage (domain spelling : String) (constructors : List Name) : String :=
  if constructors.isEmpty then
    s!"this Model declares no {domain}s, so '{spelling}' names nothing"
  else
    s!"unknown Model {domain} '{spelling}'; declared: {spellings constructors}"

private def unsortedInitialMessage (earlier later : String) : String :=
  "Model start states must be declared in sorted order, because the planner admits " ++
    s!"only a canonically ordered start-state list; '{later}' precedes '{earlier}'"

/-! ### Where a declaration comes from

The semantic family is the enclosing namespace with the project's scaffolding prefix removed, and
the source is the module being elaborated. Two Models that name the same declaration in different
files therefore carry distinct Definition IDs and distinct Provenance sources, without either file
saying so. -/

private def decapitalize (segment : String) : String :=
  match segment.toList with
  | [] => segment
  | first :: rest => String.ofList (first.toLower :: rest)

/-- Drop the leading components the project declared as scaffolding. A namespace that does not start
with them keeps all of its own. -/
private def stripPrefix : List String → List String → List String
  | scaffolding :: remainingPrefix, owned :: remainingOwned =>
      if scaffolding == owned then stripPrefix remainingPrefix remainingOwned
      else owned :: remainingOwned
  | _, owned => owned

private def semanticFamilyOf (namespacePrefix enclosing : Name) : String :=
  let owned := stripPrefix (namespacePrefix.components.map (·.toString))
    (enclosing.components.map (·.toString))
  ".".intercalate (owned.map decapitalize)

/-- The module being elaborated, as its package-relative source path. Deriving it from the module
name rather than from the file name on disk keeps the recorded source independent of where the
checkout lives and of what the package directory is called. -/
private def modulePath (declaring : Name) : String :=
  "/".intercalate (declaring.components.map (·.toString)) ++ ".lean"

private def originTerm : CommandElabM Term := do
  let conventions := Registry.conventions (← getEnv)
  let family := semanticFamilyOf conventions.namespacePrefix (← getCurrNamespace)
  let path := modulePath (← getMainModule)
  `(term| Origin.of $(Lean.quote conventions.root) $(Lean.quote family) $(Lean.quote path))

/-- The Definition ID a declaration made under `enclosing` carries.

A reference computes the referenced declaration's id from the namespace that declared it, never from
the referring file's own: the two agree only while both are in one file, and a Model that refers
across files would otherwise point at a name in its own family that nothing declares. -/
private def definitionIdIn (enclosing : Name) (kind key : String) : CommandElabM String := do
  let conventions := Registry.conventions (← getEnv)
  let family := semanticFamilyOf conventions.namespacePrefix enclosing
  pure ((Origin.of conventions.root family "").family.id kind key).value

/-- The id of a declaration this command is elaborating, which is the same computation under the
current namespace. -/
private def definitionIdHere (kind key : String) : CommandElabM String := do
  definitionIdIn (← getCurrNamespace) kind key

/-! ### Declaring a domain

`enum` is the four vocabulary declarations a Model file makes, without the `deriving` clause the
`machine` command requires and the author has no reason to think about. It resolves nothing and
reorders nothing: the constructors in declaration order are the ordered domain, which is what AUT-09
means by author-provided. `enum` is shorthand for exactly the
one it would have written. -/

-- A member's own doc comment goes after its bar, not before it. Before the bar it would be
-- indistinguishable from the doc comment of whatever declaration follows the `enum`, and the
-- repetition would swallow it.
elab doc?:(docComment)? &"enum" name:ident
    constructors:("|" (docComment)? ident (bracketedBinder)*)+ : command => do
  let declared ← constructors.mapM fun constructor => do
    let parts := constructor.raw
    let constructorDoc : Option (TSyntax ``Lean.Parser.Command.docComment) :=
      if parts[1].isNone then none else some (TSyntax.mk parts[1][0])
    let constructorName : Ident := TSyntax.mk parts[2]
    let binders : Array (TSyntax ``Lean.Parser.Term.bracketedBinder) :=
      parts[3].getArgs.map TSyntax.mk
    `(Lean.Parser.Command.ctor|
      $[$constructorDoc:docComment]? | $constructorName:ident $binders*)
  elabCommand (← `(command| $[$doc?:docComment]? inductive $name where
      $declared:ctor*
      deriving BEq, DecidableEq, Repr, Umpire.Command.Finite))
  -- The id is recorded here, where the declaring file's conventions and namespace are the ones in
  -- scope. A field that named this domain and rebuilt its id would read its own file's conventions,
  -- and two Models sharing a domain would disagree about what it is called.
  --
  -- It is recorded only once the `Finite` instance exists. A failed `deriving` is logged rather than
  -- thrown, so without this a domain whose members do not enumerate would still be a domain a later
  -- command walks, and the error it already reported would be followed by a worse one.
  let declName := (← getCurrNamespace) ++ name.getId
  let enumerates ← liftTermElabM do
    match (← getEnv).find? declName with
    | some _ =>
        let domainType ← mkConstWithLevelParams declName
        pure (← Meta.synthInstance? (← Meta.mkAppM ``Umpire.Command.Finite #[domainType])).isSome
    | none => pure false
  if enumerates then
    liftCoreM (Registry.recordDomain {
      declName
      name := name.getId.toString
      id := ← definitionIdHere "enum" name.getId.toString })


/-! ### The `property` command

A Property names its machine and a Lean predicate over the machine's steps: `Step → Bool` for a
claim about the step one Action produces (`when:` names the Action), or `Step → Step → Bool` for a
claim about the step before and the step after. The predicate is enumerated over the machine's own
table into the clause records a Property has always carried -- `Umpire.Command.Predicate` says how
-- so Search, the Behavior Fingerprint and Contract lowering never see a function. -/

private def undeclaredModelMessage (spelling : Name) : String :=
  s!"'{spelling}' is not a Model declared by a `machine` command"

private def retiredRequireMessage : String :=
  "the keyed `require:` form is retired; a `property` names a `machine:` and a `holds:` \
predicate over its steps, `Step → Bool` for a same-step claim under `when:` or \
`Step → Step → Bool` for a transition claim"

private def retiredModelKeyMessage : String :=
  "`model:` is retired on `property`; the key is `machine:`"

private def otherMachineStepMessage (carried expected : Name) : String :=
  s!"the predicate reads steps of '{carried}', which is not this machine's state; a `holds:` \
predicate is over `Step {expected} _ _`"

private def notDecidableMessage : String :=
  "the predicate is not decidable: `holds:` is a `Bool`-valued function over the machine's steps, \
so a claim is written with `==`, `&&`, `||` and `!`, not as a proposition"

private def predicateShapeMessage (expected : Name) : String :=
  s!"a `holds:` predicate is `Step {expected} _ _ → Bool` for a same-step claim under `when:`, or \
`Step {expected} _ _ → Step {expected} _ _ → Bool` for a transition claim"

private def transitionWithWhenMessage : String :=
  "a transition claim reads the step before, so it names no `when:` Action; a same-step claim \
under `when:` is `Step → Bool`"

private def sameStepWithoutWhenMessage : String :=
  "a same-step claim names the Action it is about under `when:`; a claim over every step is a \
transition claim, `Step → Step → Bool`"

/-- Resolve one spelling against a Model's declared domain, reporting an unknown one in place. The
message shape is every command's, so an author sees one vocabulary wherever they are. -/
private def resolveDeclared (domain : String) (declared : Array String) (declaringType : Name)
    (member : Ident) : CommandElabM String := do
  let spelling := member.getId.eraseMacroScopes.toString
  unless declared.contains spelling do
    throwErrorAt member (unknownMemberMessage domain spelling
      (declared.toList.map Name.mkSimple))
  -- Give the spelling the constructor it names, so the editor hovers it and goes to its
  -- definition. A machine's state key names an assignment of the structure's fields rather than a
  -- constructor, so for one of those there is nothing to point at. The emitted records keep the
  -- spelling either way, so no Definition ID moves.
  let points := declaringType ++ Name.mkSimple spelling
  if (← getEnv).contains points then
    liftTermElabM (Lean.Elab.addConstInfo member points)
  pure spelling

/-- The Model a `machine:` key names, resolved to what the `machine` command recorded about it. -/
private def resolveDeclaredModel (modelRef : Ident) : CommandElabM Registry.ModelEntry := do
  let modelName ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo modelRef)
  match Registry.model? (← getEnv) modelName with
  | some declared => pure declared
  | none => throwErrorAt modelRef (undeclaredModelMessage modelName)

/-- The Action a `when:` line names, as the key its class carries: a bare action, or a classed
action with its inputs applied, `handlerReply (handlerError true)`. The key joins the constructor
and its arguments with `-`, the way the machine command keys its Action members, so the written
class is looked up rather than rebuilt. -/
private partial def actionKeyOf (stx : Term) : String :=
  match stx with
  | `(($inner)) => actionKeyOf inner
  | `($head:ident $arguments*) =>
      "-".intercalate ((head.getId.eraseMacroScopes.getString!) ::
        arguments.toList.map fun argument => actionKeyOf ⟨argument.raw⟩)
  | `($head:ident) => head.getId.eraseMacroScopes.getString!
  | `(true) => "true"
  | `(false) => "false"
  | `($number:num) => toString number.getNat
  | _ => (stx.raw.reprint.getD "").trimAscii.toString

/-- The arity a predicate's type has over the machine's `Step`, or the reason it has none. -/
private def predicateArity (predicateRef : Term) (stateDecl outcomeDecl factDecl : Name)
    (type : Expr) : CommandElabM Nat := liftTermElabM do
  let stepType ← Meta.mkAppM ``Umpire.Step
    #[← mkConstWithLevelParams stateDecl, ← mkConstWithLevelParams outcomeDecl,
      ← mkConstWithLevelParams factDecl]
  Meta.forallTelescopeReducing type fun arguments body => do
    let body ← Meta.whnf body
    if body.isSort then
      throwErrorAt predicateRef notDecidableMessage
    unless body.isConstOf ``Bool do
      throwErrorAt predicateRef (predicateShapeMessage stateDecl)
    if arguments.isEmpty || arguments.size > 2 then
      throwErrorAt predicateRef (predicateShapeMessage stateDecl)
    for argument in arguments do
      let carried ← Meta.whnf (← Meta.inferType argument)
      unless ← Meta.isDefEq carried stepType do
        -- A `Step` over another machine's state names that state; anything else is the shape.
        match carried.getAppFn.constName?, carried.getAppArgs[0]? with
        | some head, some state =>
            if head == ``Shared.SemanticData.Result || head == ``Umpire.Step then
              match (← Meta.whnf state).getAppFn.constName? with
              | some other => throwErrorAt predicateRef (otherMachineStepMessage other stateDecl)
              | none => throwErrorAt predicateRef (predicateShapeMessage stateDecl)
            else throwErrorAt predicateRef (predicateShapeMessage stateDecl)
        | _, _ => throwErrorAt predicateRef (predicateShapeMessage stateDecl)
    pure arguments.size

/-- One attempt at elaborating the predicate, against an expected type or on its own terms. What
Lean logs while trying is collected rather than reported, so an attempt that is only a diagnosis
leaves nothing behind, and the attempt that is the claim's own shape reports exactly what it logged.
-/
private def attemptPredicate (predicateRef : Term) (expected : Option Expr) :
    CommandElabM (Option Expr × MessageLog) := do
  let saved ← get
  modify fun state => { state with messages := {} }
  let type? ← try
      let type ← liftTermElabM do
        Term.withoutErrToSorry do
          let value ← match expected with
            | some expected => Term.elabTermEnsuringType predicateRef expected
            | none => Term.elabTerm predicateRef none
          Term.synthesizeSyntheticMVarsNoPostponing
          instantiateMVars (← Meta.inferType value)
      pure (some type)
    catch failure =>
      logException failure
      pure none
  let logged := (← get).messages
  modify fun state => { state with messages := saved.messages }
  pure (if logged.hasErrors then none else type?, logged)

/-- Elaborate the predicate against the shape its claim has -- `when:` makes it a same-step
claim, its absence a transition claim -- and, when that fails, say which part is wrong: a predicate
over another machine's steps, one that is not decidable, or one of the other shape. A predicate that
elaborates as nothing reports Lean's own error at the term, against the shape the claim has. -/
private def elabPredicate (predicateRef : Term) (arity : Nat)
    (stateDecl outcomeDecl factDecl : Name) : CommandElabM Unit := do
  let stepType ← liftTermElabM do
    Meta.mkAppM ``Umpire.Step
      #[← mkConstWithLevelParams stateDecl, ← mkConstWithLevelParams outcomeDecl,
        ← mkConstWithLevelParams factDecl]
  let boolType := mkConst ``Bool
  let expected := if arity == 1 then mkForall `step .default stepType boolType
    else mkForall `before .default stepType (mkForall `after .default stepType boolType)
  let (elaborated?, logged) ← attemptPredicate predicateRef (some expected)
  if elaborated?.isSome then
    modify fun state => { state with messages := state.messages ++ logged }
    return
  -- On its own terms, to name the part that is wrong; and Lean's own error otherwise.
  let (type?, _) ← attemptPredicate predicateRef none
  if let some type := type? then
    let found ← predicateArity predicateRef stateDecl outcomeDecl factDecl type
    if found != arity then
      if arity == 1 then throwErrorAt predicateRef transitionWithWhenMessage
      else throwErrorAt predicateRef sameStepWithoutWhenMessage
  modify fun state => { state with messages := state.messages ++ logged }
  throwAbortCommand

private unsafe def evalEnumeratedUnsafe (declName : Name) :
    Elab.Term.TermElabM EnumeratedProperty :=
  Meta.evalExpr EnumeratedProperty (.const ``Umpire.Command.EnumeratedProperty []) (.const declName [])

/-- A Property's enumerated groups, read off the definition the command just emitted. -/
@[implemented_by evalEnumeratedUnsafe]
private opaque evalEnumerated (declName : Name) : Elab.Term.TermElabM EnumeratedProperty

private unsafe def evalRefinementReportUnsafe (declName : Name) :
    Elab.Term.TermElabM RefinementReport :=
  Meta.evalExpr RefinementReport (.const ``Umpire.Command.RefinementReport []) (.const declName [])

/-- A machine's derived step mapping, read off the definition the command just emitted. -/
@[implemented_by evalRefinementReportUnsafe]
private opaque evalRefinementReport (declName : Name) : Elab.Term.TermElabM RefinementReport

private def requirementTerm : PropertyRequirement → CommandElabM Term
  | .stateClause label spelling =>
      `(Umpire.Command.PropertyRequirement.stateClause $(Lean.quote label) $(Lean.quote spelling))
  | .outcomeClause label spelling =>
      `(Umpire.Command.PropertyRequirement.outcomeClause $(Lean.quote label) $(Lean.quote spelling))
  | .factClause label spelling =>
      `(Umpire.Command.PropertyRequirement.factClause $(Lean.quote label) $(Lean.quote spelling))

private def groupTerm (group : PropertyGroup) : CommandElabM Term := do
  let trigger ← match group.trigger with
    | .action spelling => `(Umpire.Command.PropertyTrigger.action $(Lean.quote spelling))
    | .priorState spelling => `(Umpire.Command.PropertyTrigger.priorState $(Lean.quote spelling))
  let requirements ← group.requirements.toArray.mapM requirementTerm
  `(({ trigger := $trigger, requirements := [$requirements,*] } : Umpire.Command.PropertyGroup))

/-- The `when:` line of a same-step claim: the Action, bare or with its class applied. -/
syntax propertyWhen := "when:" term

/-- The definition that enumerates a same-step claim over the machine's table. -/
private def sameStepCommand (enumeratedName : Ident) (declaredModel : Registry.ModelEntry)
    (key : String) (predicateRef : Term) : CommandElabM (TSyntax `command) := do
  let machineName := declaredModel.declName
  let stateType := mkIdent declaredModel.stateType
  let outcomeType := mkIdent declaredModel.outcomeType
  let stateKeyFor := mkIdent (machineName ++ `stateKeyFor)
  let outcomeKeyFor := mkIdent (machineName ++ `outcomeKeyFor)
  let factKeyFor := mkIdent (machineName ++ `factKeyFor)
  let actionKeyFor := mkIdent (machineName ++ `actionKeyFor)
  let transitions := mkIdent (machineName ++ `transitions)
  let keyLiteral := Lean.quote key
  `(command|
    def $enumeratedName : Umpire.Command.EnumeratedProperty := Umpire.Command.enumerateSameStep
      (Umpire.Command.members (α := $stateType))
      (Umpire.Command.members (α := $outcomeType))
      { state := $stateKeyFor, outcome := $outcomeKeyFor, fact := $factKeyFor }
      $keyLiteral
      (($transitions).filter (fun row => $actionKeyFor row.action == $keyLiteral)
        |>.flatMap (·.results))
      ($predicateRef))

/-- The definition that enumerates a transition claim over the machine's table. -/
private def transitionCommand (enumeratedName : Ident) (declaredModel : Registry.ModelEntry)
    (predicateRef : Term) : CommandElabM (TSyntax `command) := do
  let machineName := declaredModel.declName
  let stateType := mkIdent declaredModel.stateType
  let outcomeType := mkIdent declaredModel.outcomeType
  let stateKeyFor := mkIdent (machineName ++ `stateKeyFor)
  let outcomeKeyFor := mkIdent (machineName ++ `outcomeKeyFor)
  let factKeyFor := mkIdent (machineName ++ `factKeyFor)
  let transitions := mkIdent (machineName ++ `transitions)
  `(command|
    def $enumeratedName : Umpire.Command.EnumeratedProperty := Umpire.Command.enumerateTransition
      (Umpire.Command.members (α := $stateType))
      (Umpire.Command.members (α := $outcomeType))
      { state := $stateKeyFor, outcome := $outcomeKeyFor, fact := $factKeyFor }
      ($transitions)
      ($predicateRef))

elab "property" name:ident
    "machine:" modelRef:ident
    trigger?:(propertyWhen)?
    "holds:" predicateRef:term : command => do
    let ownerKey := Lean.quote name.getId.toString
    let declaredModel ← resolveDeclaredModel modelRef
    let roleKey := Lean.quote declaredModel.role
    -- `when:` names the Action a same-step claim is about; a claim with no `when:` is over the
    -- step before and the step after.
    let arity := if trigger?.isSome then 1 else 2
    elabPredicate predicateRef arity declaredModel.stateType declaredModel.outcomeType
      declaredModel.factType
    let enumeratedName := mkIdentFrom name (name.getId ++ `enumerated)
    let enumerated ← match trigger? with
      | some trigger => do
          let `(propertyWhen| when: $actionRef:term) := trigger
            | throwErrorAt trigger "unsupported `when:` line"
          let key := actionKeyOf actionRef
          unless declaredModel.actions.contains key do
            throwErrorAt actionRef (unknownMemberMessage "action" key
              (declaredModel.actions.toList.map Name.mkSimple))
          -- Hover a bare action's constructor where there is one; a classed key has none.
          if let `($head:ident) := actionRef then
            let points := declaredModel.actionType ++ Name.mkSimple key
            if (← getEnv).contains points then
              liftTermElabM (Lean.Elab.addConstInfo head points)
          sameStepCommand enumeratedName declaredModel key predicateRef
      | none => transitionCommand enumeratedName declaredModel predicateRef
    elabCommand enumerated
    let result ← liftTermElabM (evalEnumerated ((← getCurrNamespace) ++ enumeratedName.getId))
    if let some refusal := result.refusal then
      throwErrorAt predicateRef refusal.message
    let groups ← result.groups.toArray.mapM groupTerm
    -- The names are declared beside the Property, so a Query over several instances can read the
    -- claim back and make it over each instance's slot.
    let namesName := mkIdentFrom name (name.getId ++ `names)
    elabCommand (← `(command|
      def $namesName : Umpire.Command.PropertyNames := {
        declaration := $ownerKey
        roleName := $roleKey
        groups := [$groups,*] }))
    elabCommand (← `(command|
      def $name (values : ModelVocabulary) : Property :=
        authoredProperty ($modelRef) values $namesName))
    -- What a Query over a refining machine has to find on that machine by name: the Actions the
    -- claims are about and the outcomes and facts they fix. A state is read through the map.
    let triggered := result.groups.filterMap fun group => match group.trigger with
      | .action spelling => some spelling
      | .priorState _ => none
    let fixed := fun (select : PropertyRequirement → Option String) =>
      result.groups.flatMap fun group => group.requirements.filterMap select
    liftCoreM (Registry.recordProperty {
      declName := (← getCurrNamespace) ++ name.getId
      «model» := declaredModel.declName
      «actions» := triggered.toArray
      «outcomes» := (fixed fun requirement => match requirement with
        | .outcomeClause _ spelling => some spelling
        | _ => none).toArray
      «facts» := (fixed fun requirement => match requirement with
        | .factClause _ spelling => some spelling
        | _ => none).toArray })

/- The keyed form is retired. It is rejected here, at the key that used to introduce it, rather
than gated by the vocabulary check: `require` is a bare word, and SEM-20 keeps bare words out of
the gate. -/
elab "property" ident "machine:" ident "when:" term
    requireKeyword:"require:" modelRequirement+ : command => do
  throwErrorAt requireKeyword retiredRequireMessage

elab "property" ident modelKeyword:"model:" ident "when:" term
    "require:" modelRequirement+ : command => do
  throwErrorAt modelKeyword retiredModelKeyMessage

elab "property" ident modelKeyword:"model:" ident (propertyWhen)? "holds:" term : command => do
  throwErrorAt modelKeyword retiredModelKeyMessage

/-! ### The `scenario` command

`actions:` is the exact sequence the operation selects. Each occurrence's key is its position in
that sequence, because that is what distinguishes two occurrences of the same Action.

A Scenario may run over several instances of the machine's entity: `instances:` says how many, and
each action then names the instance that takes it, numbered from one -- `awaitStart 2`. The Search
runs over the product of the instances, so their steps interleave and every interleaving is a path;
a Case follows each instance through one sequence, so every instance performs the same actions. -/

/-- The inputs a classed action is written with, `complete (succeeded)`, keyed the way `when:` keys
them. -/
syntax scenarioArguments := "(" term,* ")"

/-- One action of a Scenario -- bare, or classed with its inputs applied -- with the instance that
takes it when there are several. -/
syntax scenarioAction := ident (scenarioArguments)? (num)?

/-- How many instances of the machine's entity a Scenario runs over. -/
syntax scenarioInstances := "instances:" num

private def zeroInstancesMessage : String :=
  "an instance count of zero admits no instance to run the Scenario over; a Scenario runs over at \
least one"

private def tooManyInstancesMessage (count : Nat) : String :=
  s!"{count} instances is more than nine; the product's keys number instances by one digit, and a \
Scenario over more instances than that is a Search no bound would admit"

private def unnumberedActionMessage (spelling : String) (count : Nat) : String :=
  s!"'{spelling}' names no instance; a Scenario over {count} instances writes which instance takes \
each action, `{spelling} 1` to `{spelling} {count}`"

private def numberedActionMessage (spelling : String) : String :=
  s!"'{spelling}' names an instance, but this Scenario declares no `instances:`; a Scenario over one \
instance writes its actions bare"

private def strayInstanceMessage (number count : Nat) : String :=
  s!"instance {number} is not one of the {count} this Scenario runs over"

private def productTooLargeMessage (states actions count size bound : Nat) : String :=
  s!"{count} instances of a machine with {states} states and {actions} action classes multiply out \
to {size} steps to enumerate; the bound is {bound}, so declare fewer instances or a smaller machine"

elab "scenario" name:ident
    "model:" modelRef:ident
    instances?:(scenarioInstances)?
    "starts:" setupRef:ident
    "actions:" "[" selected:scenarioAction,+ "]" : command => do
    let ownerKey := Lean.quote name.getId.toString
    let declaredModel ← resolveDeclaredModel modelRef
    let roleKey := Lean.quote declaredModel.role
    -- The setup state must be one the Model can start in, not merely one it declares. A machine
    -- over a structured state keys a state by every field, and a `starts:` line names its phase:
    -- the one start state whose first field holds that spelling is the one meant.
    let startKey ← do
      let spelling := setupRef.getId.eraseMacroScopes.toString
      let byPhase := declaredModel.starts.filter fun key =>
        !declaredModel.starts.contains spelling &&
          ((key.splitOn "-").head?).getD key == spelling
      match byPhase.toList with
      | [key] => pure key
      | _ => resolveDeclared "start state" declaredModel.starts declaredModel.stateType setupRef
    let setupKey := Lean.quote startKey
    let count ← match instances? with
      | none => pure 1
      | some declared =>
          let `(scenarioInstances| instances: $countRef:num) := declared
            | throwErrorAt declared "unsupported `instances:` line"
          let count := countRef.getNat
          if count == 0 then throwErrorAt countRef zeroInstancesMessage
          if count > 9 then throwErrorAt countRef (tooManyInstancesMessage count)
          -- The product is walked before the Search runs, so its size is checked here, at the line
          -- that decides it, rather than discovered as an elaboration that never finishes.
          let size := instancesSize declaredModel.states.size declaredModel.actions.size count
          if size > enumerationBound then
            throwErrorAt countRef (productTooLargeMessage declaredModel.states.size
              declaredModel.actions.size count size enumerationBound)
          pure count
    let mut spellings : Array String := #[]
    let mut occurrences : Array (String × Nat) := #[]
    for entry in selected.getElems do
      let `(scenarioAction| $actionRef:ident $[$arguments?:scenarioArguments]? $[$number?:num]?) :=
          entry
        | throwErrorAt entry "unsupported action"
      let spelling ← match arguments? with
        | none =>
            resolveDeclared "action" declaredModel.actions declaredModel.actionType actionRef
        | some arguments =>
            let `(scenarioArguments| ($inputs:term,*)) := arguments
              | throwErrorAt arguments "unsupported action"
            let key := "-".intercalate (actionRef.getId.eraseMacroScopes.getString! ::
              inputs.getElems.toList.map fun input => actionKeyOf ⟨input.raw⟩)
            unless declaredModel.actions.contains key do
              throwErrorAt entry (unknownMemberMessage "action" key
                (declaredModel.actions.toList.map Name.mkSimple))
            pure key
      let taker ← match number?, instances? with
        | none, none => pure 1
        | none, some _ => throwErrorAt actionRef (unnumberedActionMessage spelling count)
        | some numberRef, none => throwErrorAt numberRef (numberedActionMessage spelling)
        | some numberRef, some _ =>
            let number := numberRef.getNat
            if number == 0 || number > count then
              throwErrorAt numberRef (strayInstanceMessage number count)
            pure number
      spellings := spellings.push spelling
      occurrences := occurrences.push (spelling, taker)
    let entries ← occurrences.mapIdxM fun position (spelling, taker) =>
      let label := Lean.quote (toString (position + 1))
      let action := Lean.quote spelling
      let number := Lean.quote taker
      `(term| Umpire.Command.ScenarioOccurrence.mk $label $action $number)
    let namesName := mkIdentFrom name (name.getId ++ `names)
    elabCommand (← `(command|
      def $namesName : Umpire.Command.ScenarioNames := {
        declaration := $ownerKey
        roleName := $roleKey
        setupState := $setupKey
        occurrences := [$entries,*] }))
    elabCommand (← `(command|
      def $name (values : ModelVocabulary) : Scenario :=
        authoredScenario ($modelRef) values $namesName))
    liftCoreM (Registry.recordScenario {
      declName := (← getCurrNamespace) ++ name.getId
      «model» := declaredModel.declName
      «actions» := spellings
      instances := count })

macro "limits" name:ident
    "steps:" stepCount:num
    "actions:" actionCount:num
    "search:" searchCount:num : command =>
    `(command| def $name : Limits :=
        Limits.bounded $stepCount $actionCount $searchCount)

private def unknownGapKindMessage (spelling : String) : String :=
  s!"unknown Known Gap kind '{spelling}'; declared: capability, input, interpretation, claim"

private def duplicateGapMessage (name : String) : String :=
  s!"Known Gap '{name}' is already declared by this Query"

private def gapKindTerm (kind : Ident) : CommandElabM Term :=
  match kind.getId.eraseMacroScopes.toString with
  | "capability" => `(term| Umpire.KnownGapKind.capability)
  | "input" => `(term| Umpire.KnownGapKind.input)
  | "interpretation" => `(term| Umpire.KnownGapKind.interpretation)
  | "claim" => `(term| Umpire.KnownGapKind.claim)
  | spelling => throwErrorAt kind (unknownGapKindMessage spelling)

/-- The Known Gaps this Query declared, in declaration order; admission checks them as one canonical
set. A Query that declares none carries none: nothing is attached on its behalf. -/
private def knownGapsTerm (origin : Term) (gaps : Array (TSyntax `modelGap)) :
    CommandElabM Term := do
  let mut declared : Array String := #[]
  let mut terms : Array Term := #[]
  for declared? in gaps do
    match declared? with
    | `(modelGap| gap: $kindRef:ident code: $codeRef:str subject: $subjectRef:str
        detail: $detailRef:str) =>
        if declared.contains codeRef.getString then
          throwErrorAt codeRef (duplicateGapMessage codeRef.getString)
        declared := declared.push codeRef.getString
        terms := terms.push (← `(term|
          Origin.knownGap $origin $(← gapKindTerm kindRef) $codeRef (some $subjectRef) $detailRef))
    | `(modelGap| gap: $kindRef:ident code: $codeRef:str detail: $detailRef:str) =>
        if declared.contains codeRef.getString then
          throwErrorAt codeRef (duplicateGapMessage codeRef.getString)
        declared := declared.push codeRef.getString
        terms := terms.push (← `(term|
          Origin.knownGap $origin $(← gapKindTerm kindRef) $codeRef none $detailRef))
    | _ => throwErrorAt declared? "unsupported Known Gap"
  `(term| [$terms,*])

/-- Record what a `case` block needs to know about a Query: whether it selects a witness, and the
Scenario whose Action order its evidence lines resolve against. -/
private def recordQueryDeclaration
    (name scenarioRef : Ident) (selectsWitness : Bool) : CommandElabM Unit := do
  let scenarioName ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo scenarioRef)
  liftCoreM (Registry.recordQuery {
    declName := (← getCurrNamespace) ++ name.getId
    selectsWitness
    «scenario» := scenarioName })

/-! ### The `query` command

A Query names no Model: its Property and its Scenario each name one, and they must be the same. -/

private def mismatchedModelMessage (declaredProperty declaredScenario : Name) : String :=
  s!"the Property runs on Model '{declaredProperty}' and the Scenario on " ++
    s!"'{declaredScenario}'; a Query asks one question of one Model"

private def undeclaredMessage (kind : String) (spelling : Name) : String :=
  s!"'{spelling}' is not a {kind} declared by a `{kind}` command"

private def unliftableMessage (kind spelling : String) (refined refining : Name) : String :=
  s!"the Property names the {kind} '{spelling}' of '{refined}', and '{refining}' has no {kind} of \
that name; a Property on the refined machine is read on the refining one through the values of \
the same name, and a state through its `map:`"

private def liftedInstancesMessage : String :=
  "a Property on the refined machine is read on one instance of the refining one; a Query over \
several instances names a Property on the machine its Scenario runs over"

/-- What a Query runs on: the Model, the number of instances its Scenario runs over, and -- when
the Property is declared on the machine the Scenario's machine refines -- that refined Model. -/
private structure QuerySubject where
  model : Name
  instances : Nat := 1
  lifted : Option Name := none

/-- The Model a Query runs on, resolved from its Property and its Scenario rather than named again.
The two name one Model, or the Scenario's machine `refines:` the Property's: then the Property is
read on the refining machine's paths through its `map:`, and every Action, outcome and fact the
Property names has to be one the refining machine names too. -/
private def queryModelName (propertyRef scenarioRef : Ident) : CommandElabM QuerySubject := do
  let propertyName ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo propertyRef)
  let scenarioName ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo scenarioRef)
  let environment ← getEnv
  let declaredProperty ← match Registry.property? environment propertyName with
    | some declared => pure declared
    | none => throwErrorAt propertyRef (undeclaredMessage "property" propertyName)
  let declaredScenario ← match Registry.scenario? environment scenarioName with
    | some declared => pure declared
    | none => throwErrorAt scenarioRef (undeclaredMessage "scenario" scenarioName)
  if declaredProperty.model == declaredScenario.model then
    return { model := declaredProperty.model, instances := declaredScenario.instances }
  let refines := (Registry.machine? environment declaredScenario.model).bind (·.refines)
  unless refines == some declaredProperty.model do
    throwErrorAt scenarioRef
      (mismatchedModelMessage declaredProperty.model declaredScenario.model)
  if declaredScenario.instances > 1 then throwErrorAt scenarioRef liftedInstancesMessage
  let some refining := Registry.model? environment declaredScenario.model
    | throwErrorAt scenarioRef (undeclaredModelMessage declaredScenario.model)
  let require := fun (kind : String) (declared : Array String) (named : Array String) =>
    match named.find? fun spelling => !declared.contains spelling with
    | some spelling => throwErrorAt propertyRef
        (unliftableMessage kind spelling declaredProperty.model declaredScenario.model)
    | none => pure ()
  require "action" refining.actions declaredProperty.actions
  require "outcome" refining.outcomes declaredProperty.outcomes
  require "fact" refining.facts declaredProperty.facts
  pure { model := declaredScenario.model, lifted := some declaredProperty.model }

/-- The admission a Query evaluates: over the machine itself, or over the product of the instances
its Scenario runs over, reading the Property and the Scenario back from their names. A Property on
the refined machine is read on the refining one through `Umpire.Command.refinedProperty`. -/
private def checkTerm (modelRef : Ident) (instances : Nat) (queryKey : Term)
    (limitsRef propertyRef scenarioRef : Ident) (knownGaps : Term)
    (form : Option Term) (lifted : Option Name := none) : CommandElabM Term := do
  let form ← match form with
    | some form => pure form
    | none => `(QueryFormKind.selectWitness)
  if let some refined := lifted then
    let propertyNames := mkIdent (propertyRef.getId ++ `names)
    let refinedRef := mkIdent refined
    return ← `(check ($modelRef) $queryKey ($limitsRef)
      (fun values => Umpire.Command.refinedProperty ($modelRef) ($refinedRef) values $propertyNames)
      ($scenarioRef) (knownGaps := $knownGaps) (form := $form))
  if instances == 1 then
    `(check ($modelRef) $queryKey ($limitsRef) ($propertyRef) ($scenarioRef)
      (knownGaps := $knownGaps) (form := $form))
  else
    let propertyNames := mkIdent (propertyRef.getId ++ `names)
    let scenarioNames := mkIdent (scenarioRef.getId ++ `names)
    let count := Lean.quote instances
    `(Umpire.Command.checkInstances ($modelRef) $count $queryKey ($limitsRef)
      ($propertyNames) ($scenarioNames) (knownGaps := $knownGaps) (form := $form))

/-! ### Admission runs while the Model file compiles

A `query` block defines a value nothing in the Model file evaluates, so every authoring mistake
below the command surface -- a Property no admitted trace satisfies, a Scenario the Model cannot
take, limits too small to reach the trace the author means -- used to leave the file green and
surface somewhere else. The command evaluates its own admission here and reports what comes back at
the part of the block it belongs to. -/

private unsafe def evalDiagnosticUnsafe (diagnosticName : Name) :
    Elab.Term.TermElabM (Option (String × String)) :=
  let pair := mkApp2 (.const ``Prod [levelZero, levelZero]) (.const ``String []) (.const ``String [])
  Meta.evalExpr (Option (String × String)) (.app (.const ``Option [levelZero]) pair)
    (.const diagnosticName [])

@[implemented_by evalDiagnosticUnsafe]
private opaque evalDiagnostic (diagnosticName : Name) :
    Elab.Term.TermElabM (Option (String × String))

private unsafe def evalStuckStateUnsafe (diagnosticName : Name) :
    Elab.Term.TermElabM (Option String) :=
  Meta.evalExpr (Option String) (.app (.const ``Option [levelZero]) (.const ``String []))
    (.const diagnosticName [])

private unsafe def evalStringListUnsafe (diagnosticName : Name) :
    Elab.Term.TermElabM (List String) :=
  Meta.evalExpr (List String) (.app (.const ``List [levelZero]) (.const ``String []))
    (.const diagnosticName [])

/-- A list of keys read off a table the command just emitted. -/
@[implemented_by evalStringListUnsafe]
private opaque evalStringList (diagnosticName : Name) : Elab.Term.TermElabM (List String)

/-- A machine's stuck-state witness, read off the table the command just emitted. -/
@[implemented_by evalStuckStateUnsafe]
private opaque evalStuckState (diagnosticName : Name) : Elab.Term.TermElabM (Option String)

/-- Report whatever admission said, on the part of the `query` block it belongs to. -/
private def reportAdmission
    (name propertyRef scenarioRef limitsRef modelRef formKeyword : Syntax)
    (diagnosticName : Name) : CommandElabM Unit := do
  match ← liftTermElabM (evalDiagnostic diagnosticName) with
  | none => pure ()
  | some (anchorName, reported) =>
      let reference :=
        if anchorName == Diagnostic.anchorModel then modelRef
        else if anchorName == Diagnostic.anchorProperty then propertyRef
        else if anchorName == Diagnostic.anchorScenario then scenarioRef
        else if anchorName == Diagnostic.anchorLimits then limitsRef
        else if anchorName == Diagnostic.anchorForm then formKeyword
        else name
      throwErrorAt reference reported

/-- Emit the Query's own diagnostic beside it, then evaluate and report it. -/
private def elabQueryAdmission
    (name propertyRef scenarioRef limitsRef modelRef formKeyword : Syntax)
    (queryName : Ident) : CommandElabM Unit := do
  let diagnosticName := mkIdentFrom queryName (queryName.getId ++ `diagnostic)
  elabCommand (← `(command|
    def $diagnosticName : Option (String × String) := Umpire.Command.diagnose $queryName))
  reportAdmission name propertyRef scenarioRef limitsRef modelRef formKeyword
    ((← getCurrNamespace) ++ diagnosticName.getId)

elab "query" name:ident
    findKeyword:"find:" propertyRef:ident
    "in:" scenarioRef:ident
    "limits:" limitsRef:ident gaps:modelGap* : command => do
    let target ← queryModelName propertyRef scenarioRef
    let modelRef := mkIdent target.model
    let queryKey := Lean.quote name.getId.toString
    let knownGaps ← knownGapsTerm (← originTerm) gaps
    let admission ← checkTerm modelRef target.instances queryKey limitsRef propertyRef scenarioRef
      knownGaps none target.lifted
    elabCommand (← `(command|
      def $name : Except AdmissionError (CheckedModel ($modelRef)) := $admission))
    recordQueryDeclaration name scenarioRef (selectsWitness := true)
    elabQueryAdmission name propertyRef scenarioRef limitsRef modelRef findKeyword name

elab "query" name:ident
    verifyKeyword:"verify:" propertyRef:ident
    "in:" scenarioRef:ident
    "limits:" limitsRef:ident gaps:modelGap* : command => do
    let target ← queryModelName propertyRef scenarioRef
    let modelRef := mkIdent target.model
    let queryKey := Lean.quote name.getId.toString
    let knownGaps ← knownGapsTerm (← originTerm) gaps
    let admission ← checkTerm modelRef target.instances queryKey limitsRef propertyRef scenarioRef
      knownGaps (some (← `(QueryFormKind.verifyClaim))) target.lifted
    elabCommand (← `(command|
      def $name : Except AdmissionError (CheckedModel ($modelRef)) := $admission))
    recordQueryDeclaration name scenarioRef (selectsWitness := false)
    elabQueryAdmission name propertyRef scenarioRef limitsRef modelRef verifyKeyword name

/-! ### The side-effect commands

`entity`, `action` and `observation` declare what a feature acts on, what its parties do, and what
confirms a step. They follow the same reading rule as every other command -- a column-0 word is the
declaration kind followed by the author's name, an indented `word:` is a framework key -- and each
elaborates into the plain record `Umpire.Command.Records` owns, with Definition IDs derived from the
file's `Origin`.

The keys are parsed as a syntax category rather than a fixed signature, because most of them are
optional and an optional group in a command signature does not bind. That also puts every key on its
own line for a rejection to point at. -/

/-- The leading word of a declaration command, spelled so that it stays an ordinary identifier.

`entity`, `action` and `observation` are the vocabulary a Model is written in, and they are also
field names inside the records the commands build -- `EntityReference.entity`, `Observation.entity`,
`FiniteTransitionRow.action`. Reserving them as tokens, the way `property` and `scenario` are
reserved,
would make every one of those fields unwritable. `&"entity"` alone does not work either: a
non-reserved symbol is indexed under its own token, and a command that begins with a bare identifier
is dispatched under `ident`, so the parser would only be reachable behind a doc comment.

`includeIdent` indexes the parser under both, which is what makes a column-0 `entity` a command
without taking the word away from the rest of the tree. -/
private def declarationKeyword (word : String) : Lean.Parser.Parser :=
  Lean.Parser.nonReservedSymbolNoAntiquot word (includeIdent := true)

@[run_parser_attribute_hooks] private def entityKeyword := declarationKeyword "entity"
@[run_parser_attribute_hooks] private def actionKeyword := declarationKeyword "action"
@[run_parser_attribute_hooks] private def observationKeyword := declarationKeyword "observation"

/-- One indented key of an `entity` declaration. -/
declare_syntax_cat entityKey
syntax "refer:" withPosition((colGe ident ":" ident)+) : entityKey
syntax "key:" ident : entityKey

/-- One `examples:` line: the class as the Model spells it, and the concrete member a functional
Case uses for it. -/
declare_syntax_cat exampleLine
syntax ident ("(" (ident " := " term),* ")")? " → " ident : exampleLine

/-- One indented key of an `action` declaration. -/
declare_syntax_cat actionKey
syntax "party:" ident : actionKey
syntax "on:" ident : actionKey
syntax "creates:" ident : actionKey
syntax "input:" withPosition((colGe ident ":" ident)+) : actionKey
syntax "results:" ident : actionKey
syntax "schema:" ident ("|" ident)* : actionKey
syntax "examples:" withPosition((colGe exampleLine)+) : actionKey

/-- One indented key of an `observation` declaration. -/
declare_syntax_cat observationKey
syntax "on:" ident : observationKey
syntax "read:" ident : observationKey

private def duplicateKeyMessage (kind key : String) : String :=
  s!"the {kind} declares '{key}' twice; each key is declared once"

private def missingKeyMessage (kind key : String) : String :=
  s!"the {kind} declares no '{key}'; it is required"

private def undeclaredEntityMessage (spelling : Name) : String :=
  s!"'{spelling}' is not an entity declared by an `entity` command"

private def undeclaredDomainMessage (spelling : Name) : String :=
  s!"'{spelling}' is not a finite domain; an input field ranges over an `enum` declaration, whose \
constructors are its classes"

private def unspellableFieldMessage (domain : Name) (field : String) (type : MessageData) :
    MessageData :=
  m!"the class '{domain}' carries a field '{field}' of type {type}, which is not a domain a class \
can be written over: a constructor field is another `enum` declaration, a `Bool`, or a count"

private def unboundedSetupMessage (spelling : Name) : String :=
  s!"'{spelling}' is not a finite domain; a `setup:` parameter is varied over its values, so it \
ranges over an `enum` declaration, a `Bool`, or a count"

private def notAnEnumMessage (spelling : Name) : String :=
  s!"'{spelling}' is not an `enum` declaration; an input field ranges over an `enum`, whose members \
are its classes and whose Definition ID they hang off"

private def recursiveDomainMessage (domain : Name) : String :=
  s!"'{domain}' carries itself, so its members cannot be written out; a class is a finite spelling, \
and a domain that contains itself has no finite one"

private def domainTooLargeMessage (domain : Name) (size bound : Nat) : String :=
  s!"'{domain}' has more than {bound} members ({size} and still counting); a class is written out \
one per member, and the elaboration bound is {bound}"

private def duplicateReferenceMessage (field : String) : String :=
  s!"the entity refers by '{field}' twice; a row names a reference by its field, and two would not \
say which one it means"

private def duplicateEntityKeyMessage (key : String) (owner : String) : String :=
  s!"key name '{key}' is already the key of entity '{owner}'; recorded data would not say which \
instance it names"

private def reservedPartyMessage : String :=
  "'system' is the implementation under test; it performs no declared action, so an action's \
`party:` names one of the feature's own parties"

private def unmatchedExampleMessage (spelling : String) : String :=
  s!"'{spelling}' matches no class of this action: an example names one of the classes of one of \
the action's own `input:` domains, written with the constructor's own name and its fields by name"

private def duplicateExampleMessage (spelling : String) : String :=
  s!"'{spelling}' already has an example; a class has one concrete member, or the Case it produces \
would not be one Case"

private def bothSubjectsMessage : String :=
  "an action declares `on:` or `creates:`, not both: it either acts on an instance that exists or \
brings one into existence"

/-- One class of an input domain, as the tree it is rather than as the text it renders to.

A class is a constructor with its fields assigned, and a field's value is itself a class. Comparing
the rendered text instead would let two classes whose names run together across a parenthesis --
`ab (c := false)` and `a (bc := false)` -- compare equal, and an example would be stored against a
class its author did not write. -/
private inductive ClassValue where
  /-- A constructor that carries nothing, `Bool`'s `false` and `true`, or a count's numeral. -/
  | atom (spelling : Name)
  /-- A constructor with every field it carries assigned. -/
  | applied (constructor : Name) (fields : Array (String × ClassValue))
  deriving Inhabited

/-- A class as an author writes it, which is what `InputField.classes` carries and what an
`examples:` line is quoted as. -/
private partial def ClassValue.render : ClassValue → String
  | .atom spelling => spelling.getString!
  | .applied constructor fields =>
      let written := fields.toList.map fun (field, value) => s!"{field} := {value.render}"
      s!"{constructor.getString!} ({", ".intercalate written})"

/-- A class as a Definition ID fragment: the same tree, joined by `-` rather than punctuated, so a
state or action carries a key a reader recognises and an id admits. A structure's `mk` contributes
nothing, because a reader reads its fields and not its constructor. -/
private partial def ClassValue.key : ClassValue → String
  | .atom spelling => spelling.getString!
  | .applied constructor fields =>
      let written := fields.toList.map fun (_, value) => value.key
      let joined := "-".intercalate written
      if constructor.getString! == "mk" then joined
      else constructor.getString! ++ "-" ++ joined

/-- A class as the Lean term that denotes it, which is also the pattern that matches it. The
enumeration knows every class as a tree, so the generated code that names one -- a key function's
match arm, a member list -- is written from the same tree rather than from a second rendering. -/
private partial def ClassValue.term : ClassValue → CommandElabM Term
  | .atom spelling =>
      -- A count's member is a numeral, not a name: `Fin`'s members are spelled `0`, `1`, `2`, and
      -- `mkIdent` would make each of them an identifier no scope declares.
      let written := spelling.getString!
      if !written.isEmpty && written.all Char.isDigit then
        pure (Syntax.mkNumLit written)
      else
        `($(mkIdent spelling))
  | .applied constructor fields => do
      -- Positional, not by name: the fields are in declaration order here, and a pattern written
      -- positionally is one Lean accepts everywhere a term is accepted.
      let arguments ← fields.mapM fun (_, value) => value.term
      `($(mkIdent constructor) $arguments*)

/-- Whether a written class names a declared one.

The fields are matched by name rather than by position, so an author who writes a constructor's
fields in another order writes the same class. Everything else is structural, so nothing merges
across a name boundary.

A constructor a line qualifies is checked against the qualification, not stripped of it: a declared
constructor's name is its whole one, and a written name has to be a suffix of it. Two domains may
each declare `handlerError` -- `DESIGN.md` section 3 declares exactly that, on `Reply` and
`CancelReply` -- so discarding the qualification would accept one domain's class on the other's
field. -/
private partial def ClassValue.matches : ClassValue → ClassValue → Bool
  | .atom written, .atom declared => written.isSuffixOf declared
  | .applied writtenName writtenFields, .applied declaredName declaredFields =>
      writtenName.isSuffixOf declaredName && writtenFields.size == declaredFields.size &&
        declaredFields.all fun (field, declared) =>
          match writtenFields.find? fun (written, _) => written == field with
          | some (_, written) => written.matches declared
          | none => false
  | _, _ => false

private partial def domainMembers (domainRef : Ident) (declName : Name)
    (visiting : List Name := []) : CommandElabM (List ClassValue) := do
  -- A domain that carries itself has no finite spelling, and walking into it would not terminate.
  -- The bound below is a width, so the depth needs its own answer.
  if visiting.contains declName then
    throwErrorAt domainRef (recursiveDomainMessage declName)
  let visiting := declName :: visiting
  match (← getEnv).find? declName with
  | some (.inductInfo info) =>
      let mut members := []
      for constructor in info.ctors do
        members := members ++ (← constructorMembers declName constructor visiting)
        if members.length > elaborationBound then
          throwErrorAt domainRef (domainTooLargeMessage declName members.length elaborationBound)
      pure members
  | _ => throwErrorAt domainRef (undeclaredDomainMessage domainRef.getId)
where
  /-- One constructor's classes: itself when it carries nothing, and every assignment of its fields
  otherwise. -/
  constructorMembers (declName constructor : Name) (visiting : List Name) :
      CommandElabM (List ClassValue) := do
    let declaration ← liftTermElabM (getConstInfoCtor constructor)
    if declaration.numFields == 0 then
      return [.atom constructor]
    let fields ← liftTermElabM do
      Meta.forallTelescopeReducing declaration.type fun arguments _ => do
        let carried := arguments.extract (arguments.size - declaration.numFields) arguments.size
        carried.mapM fun argument => do
          let field ← argument.fvarId!.getDecl
          pure (field.userName.getString!, field.type)
    -- The first field varies slowest, so the classes read in the order the constructor is written.
    let mut assignments : List (Array (String × ClassValue)) := [#[]]
    for (field, type) in fields do
      let values ← fieldValues declName field type visiting
      assignments := assignments.flatMap fun assigned =>
        values.map fun value => assigned.push (field, value)
      if assignments.length > elaborationBound then
        throwErrorAt domainRef (domainTooLargeMessage declName assignments.length elaborationBound)
    pure (assignments.map fun assigned => .applied constructor assigned)
  /-- The values one constructor field ranges over.

  Only three shapes are admitted, and the recursion is what makes that necessary: a hand-written
  `Finite` instance can satisfy the gate for a type whose constructors do not enumerate -- `Nat` with
  a two-member instance is the smallest -- and walking into it would not terminate. A field is an
  `enum` declaration, a `Bool`, or a count, and anything else is named here rather than found by
  running out of stack. -/
  fieldValues (declName : Name) (field : String) (type : Expr) (visiting : List Name) :
      CommandElabM (List ClassValue) := do
    let unspellable : CommandElabM (List ClassValue) := do
      throwErrorAt domainRef
        (unspellableFieldMessage declName field (← liftTermElabM (Meta.ppExpr type)))
    match type.getAppFn with
    | .const name _ =>
        if name == ``Bool then
          pure [.atom ``Bool.false, .atom ``Bool.true]
        else if name == ``Fin then
          match (← liftTermElabM (Meta.whnf type)).getAppArgs[0]? with
          | some bound =>
              -- The bound is reduced before it is read: a count's bound is written as a name once
              -- it comes from the Limits rather than from the field, and `evalNat` reads arithmetic
              -- and literals but does not unfold a constant to find them.
              match (← liftTermElabM do (Meta.evalNat (← Meta.whnf bound)).run) with
              | some size =>
                  pure ((List.range size).map fun count => .atom (Name.mkSimple (toString count)))
              | none => unspellable
          | none => unspellable
        else if (Registry.domain? (← getEnv) name).isSome then
          domainMembers domainRef name visiting
        else
          unspellable
    | _ => unspellable

/-- Resolve an entity reference against what an `entity` command recorded, reporting an unknown one
in place and giving the editor the declaration to hover. -/
private def resolveEntity (entityRef : Ident) : CommandElabM Registry.EntityEntry := do
  -- A name that resolves to nothing at all and a name that resolves to something other than an
  -- entity are the same mistake to the author, so the resolution failure is caught and reported as
  -- the one message, spelled as the line spells it.
  let declName? ← try
      some <$> liftTermElabM (realizeGlobalConstNoOverloadWithInfo entityRef)
    catch failure =>
      if failure.isInterrupt || failure.isMaxRecDepth then throw failure else pure none
  match declName?.bind (Registry.entity? (← getEnv)) with
  | some declared => pure declared
  | none => throwErrorAt entityRef (undeclaredEntityMessage entityRef.getId)

/-- Resolve the enum an `input:` or `results:` line names, requiring it to be finite. A domain that
is not finite cannot be enumerated into a Model's table, and saying so here names the line rather
than failing an instance search inside the machine command. -/
private def resolveDomain (domainRef : Ident) :
    CommandElabM (Name × String × List ClassValue) := do
  let declName ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo domainRef)
  liftTermElabM do
    let domainType ← mkConstWithLevelParams declName
    let finiteType ← Meta.mkAppM ``Umpire.Command.Finite #[domainType]
    unless (← Meta.synthInstance? finiteType).isSome do
      throwErrorAt domainRef (undeclaredDomainMessage domainRef.getId)
  -- The id comes from the `enum` that declared it, never from this file: see `Registry.DomainEntry`.
  let some declared := Registry.domain? (← getEnv) declName
    | throwErrorAt domainRef (notAnEnumMessage domainRef.getId)
  pure (declName, declared.id, ← domainMembers domainRef declName)

/-- Resolve the domain a `setup:` parameter ranges over.

A setup parameter configures the implementation under test; it is not an input an action carries.
No class of it is ever written out and no example is ever stored against one, so the rule that a
class's Definition ID hangs off the `enum` that declared it has nothing to attach to here. What a
setup parameter has to be is finite, because varying the table over its values is what task `.5`
does with it -- and `DESIGN.md` section 3's own parameters are `Bool`, which is finite without being
an `enum`. -/
private def resolveSetupDomain (domainRef : Ident) : CommandElabM Name := do
  let declName ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo domainRef)
  liftTermElabM do
    let domainType ← mkConstWithLevelParams declName
    let finiteType ← Meta.mkAppM ``Umpire.Command.Finite #[domainType]
    unless (← Meta.synthInstance? finiteType).isSome do
      throwErrorAt domainRef (unboundedSetupMessage domainRef.getId)
  pure declName

elab doc?:(docComment)? entityKeyword name:ident keys:entityKey* : command => do
  let mut refers : Array (String × String × Name) := #[]
  let mut key : Option Ident := none
  let mut seenRefer := false
  for entry in keys do
    match entry with
    | `(entityKey| refer: $[$fields:ident : $targets:ident]*) => do
        if seenRefer then throwErrorAt entry (duplicateKeyMessage "entity" "refer:")
        seenRefer := true
        for field in fields, target in targets do
          let spelling := field.getId.toString
          if refers.any fun (seen, _, _) => seen == spelling then
            throwErrorAt field (duplicateReferenceMessage spelling)
          let declared ← resolveEntity target
          refers := refers.push (spelling, declared.id, declared.declName)
    | `(entityKey| key: $keyRef:ident) => do
        if key.isSome then throwErrorAt entry (duplicateKeyMessage "entity" "key:")
        key := some keyRef
    | _ => throwErrorAt entry "unsupported entity key"
  -- An entity that declares no `key:` is named by itself: the common case is one instance whose
  -- recorded identifier is the entity's own, and saying so twice reads as though it could differ.
  let keyRef := key.getD name
  let keySpelling := keyRef.getId.toString
  -- One key name per Model: recorded data finds an instance through it, so two entities sharing one
  -- would leave a row unable to say which instance it is about.
  for declared in Registry.localEntities (← getEnv) do
    if declared.key == keySpelling then
      throwErrorAt keyRef (duplicateEntityKeyMessage keySpelling declared.name)
  let origin ← originTerm
  let nameKey := Lean.quote name.getId.toString
  let entityId ← definitionIdHere "entity" name.getId.toString
  let referTerms : Array Term ← refers.mapM fun (field, targetId, _) =>
    `(term| ({ field := $(Lean.quote field)
               entity := Umpire.DefinitionId.of $(Lean.quote targetId)
             } : Umpire.Command.EntityReference))
  elabCommand (← `(command|
    $[$doc?:docComment]? def $name : Umpire.Command.Entity := {
      id := Umpire.DefinitionId.of $(Lean.quote entityId)
      name := $nameKey
      refers := [$referTerms,*]
      key := $(Lean.quote keySpelling)
      source := ($origin).source }))
  liftCoreM (Registry.recordEntity {
    declName := (← getCurrNamespace) ++ name.getId
    name := name.getId.toString
    id := entityId
    key := keySpelling
    refers := refers.map fun (field, _, declName) => (field, declName) })

/-- A written value as the class tree it denotes, or `none` when it is not one.

A value is a constructor, optionally with its fields assigned by name; parentheses around the whole
of it are the author's and mean nothing. Reading it into the same shape the walk produces is what
makes the comparison about the class rather than about the text. -/
private partial def classValueOf (written : Term) : Option ClassValue :=
  match written with
  | `(($inner)) => classValueOf inner
  | `($constructor:ident) => some (.atom constructor.getId)
  | `($literal:num) => some (.atom (Name.mkSimple (toString literal.getNat)))
  | _ =>
      match written.raw with
      | .node _ ``Lean.Parser.Term.app #[function, arguments] => do
          let .ident _ _ name _ := function | none
          let mut fields := #[]
          for argument in arguments.getArgs do
            match argument with
            | `(Lean.Parser.Term.namedArgument| ($field:ident := $value)) =>
                fields := fields.push (field.getId.getString!, ← classValueOf value)
            | _ => none
          some (.applied name fields)
      | _ => none

/-- How one `examples:` line reads back. -/
private structure ExampleLine where
  /-- The line as the author wrote it, which is what a rejection quotes. -/
  written : String
  /-- The class it names, or `none` when the line is not a class at all. -/
  class? : Option ClassValue
  member : String
  ref : Syntax

/-- The line as the author wrote it, for a rejection to quote. -/
private def exampleWritten (constructor : Ident) (bindings : Array (Ident × Term)) : String :=
  if bindings.isEmpty then constructor.getId.toString
  else
    let written := bindings.toList.map fun (field, value) =>
      let rendered := (value.raw.reprint.getD "").trim
      s!"{field.getId} := {rendered}"
    let joined := ", ".intercalate written
    s!"{constructor.getId} ({joined})"

elab doc?:(docComment)? actionKeyword name:ident keys:actionKey+ : command => do
  let mut party : Option Ident := none
  let mut subject : Option (String × Name × Bool) := none
  let mut inputFields : Array (String × Name × String × List ClassValue) := #[]
  let mut seenInput := false
  let mut results : Option (Name × String) := none
  let mut schema : Array String := #[]
  let mut examples : Array ExampleLine := #[]
  let mut seenExamples := false
  for entry in keys do
    match entry with
    | `(actionKey| party: $partyRef:ident) => do
        if party.isSome then throwErrorAt entry (duplicateKeyMessage "action" "party:")
        if partyRef.getId.toString == "system" then throwErrorAt partyRef reservedPartyMessage
        party := some partyRef
    | `(actionKey| on: $entityRef:ident) => do
        if subject.isSome then throwErrorAt entry bothSubjectsMessage
        let declared ← resolveEntity entityRef
        subject := some (declared.id, declared.declName, false)
    | `(actionKey| creates: $entityRef:ident) => do
        if subject.isSome then throwErrorAt entry bothSubjectsMessage
        let declared ← resolveEntity entityRef
        subject := some (declared.id, declared.declName, true)
    | `(actionKey| input: $[$fields:ident : $domains:ident]*) => do
        if seenInput then throwErrorAt entry (duplicateKeyMessage "action" "input:")
        seenInput := true
        for field in fields, domain in domains do
          let (declName, domainId, classes) ← resolveDomain domain
          inputFields := inputFields.push (field.getId.toString, declName, domainId, classes)
    | `(actionKey| results: $domainRef:ident) => do
        if results.isSome then throwErrorAt entry (duplicateKeyMessage "action" "results:")
        let (declName, domainId, _) ← resolveDomain domainRef
        results := some (declName, domainId)
    | `(actionKey| schema: $first:ident $[| $alternatives:ident]*) => do
        if !schema.isEmpty then throwErrorAt entry (duplicateKeyMessage "action" "schema:")
        schema := #[first.getId.toString] ++ alternatives.map (·.getId.toString)
        match ← (Umpire.Command.checkSchema schema.toList : IO _) with
        | .ok () => pure ()
        | .error reason => throwErrorAt entry reason
    | `(actionKey| examples: $[$lines:exampleLine]*) => do
        if seenExamples then throwErrorAt entry (duplicateKeyMessage "action" "examples:")
        seenExamples := true
        for line in lines do
          match line with
          | `(exampleLine| $constructor:ident $[($[$fields:ident := $values:term],*)]? → $member:ident) =>
              let bindings := match fields, values with
                | some fields, some values => fields.zip values
                | _, _ => #[]
              let class? : Option ClassValue :=
                if bindings.isEmpty then some (.atom constructor.getId)
                else do
                  let mut fields := #[]
                  for (field, value) in bindings do
                    fields := fields.push (field.getId.getString!, ← classValueOf value)
                  some (.applied constructor.getId fields)
              examples := examples.push {
                written := exampleWritten constructor bindings
                class?
                member := member.getId.toString
                ref := line }
          | _ => throwErrorAt line "unsupported example line"
    | _ => throwErrorAt entry "unsupported action key"
  let some partyRef := party
    | throwErrorAt name (missingKeyMessage "action" "party:")
  -- Examples are resolved after every key has been read, so an `examples:` block above the
  -- `input:` block it names reads the same as one below it.
  let mut resolved : Array (String × String × String × ExampleLine) := #[]
  for line in examples do
    -- The whole class is matched, not its head: a class is a member of the domain, so
    -- `handlerError (retryable := false)` and `handlerError (retryable := true)` are two classes,
    -- and a binding of a field the constructor does not carry is a class that does not exist.
    let mut matched : Option (String × String × String) := none
    for (field, _, domainId, classes) in inputFields do
      if matched.isNone then
        if let some class? := line.class? then
          -- Matched as a tree, not as text: a class is what it denotes, so a different order of
          -- bindings or a parenthesis the author added is the same class, and two classes whose
          -- names would run together in one string stay two classes.
          if let some declared := classes.find? (class?.matches ·) then
            matched := some (field, domainId, declared.render)
    let some (field, domainId, spelling) := matched
      | throwErrorAt line.ref (unmatchedExampleMessage line.written)
    if resolved.any fun (_, _, seen, _) => seen == spelling then
      throwErrorAt line.ref (duplicateExampleMessage spelling)
    resolved := resolved.push (field, domainId, spelling, line)
  let origin ← originTerm
  let nameKey := Lean.quote name.getId.toString
  let subjectTerm : Term ← match subject with
    | none => `(term| Umpire.Command.ActionSubject.free)
    | some (entityId, _, creates) =>
        let entityId ← `(term| Umpire.DefinitionId.of $(Lean.quote entityId))
        if creates then `(term| Umpire.Command.ActionSubject.creates $entityId)
        else `(term| Umpire.Command.ActionSubject.acts $entityId)
  let actionId ← definitionIdHere "action" name.getId.toString
  let inputTerms : Array Term ← inputFields.mapM fun (field, _, domainId, classes) => do
    let classTerms : Array Term ← classes.toArray.mapM fun declared =>
      `(term| Umpire.ModelValue.named (Umpire.DefinitionId.of $(Lean.quote domainId))
          $(Lean.quote declared.render))
    `(term| ({ name := $(Lean.quote field)
               domain := Umpire.DefinitionId.of $(Lean.quote domainId)
               classes := [$classTerms,*] } : Umpire.Command.InputField))
  let exampleTerms : Array Term ← resolved.mapM fun (field, domainId, spelling, line) =>
    `(term| ({ action := Umpire.DefinitionId.of $(Lean.quote actionId)
               field := $(Lean.quote field)
               pattern := $(Lean.quote spelling)
               member := Umpire.ModelValue.named (Umpire.DefinitionId.of $(Lean.quote domainId))
                 $(Lean.quote line.member)
             } : Umpire.Command.Example))
  let resultsTerm : Term ← match results with
    | none => `(term| none)
    | some (_, domainId) => `(term| some (Umpire.DefinitionId.of $(Lean.quote domainId)))
  let schemaTerms : Array Term := schema.map Lean.quote
  elabCommand (← `(command|
    $[$doc?:docComment]? def $name : Umpire.Command.Action := {
      id := Umpire.DefinitionId.of $(Lean.quote actionId)
      name := $nameKey
      party := $(Lean.quote partyRef.getId.toString)
      subject := $subjectTerm
      schema := [$schemaTerms,*]
      input := [$inputTerms,*]
      results := $resultsTerm
      examples := [$exampleTerms,*]
      source := ($origin).source }))
  liftCoreM (Registry.recordAction {
    declName := (← getCurrNamespace) ++ name.getId
    name := name.getId.toString
    party := partyRef.getId.toString
    subject := subject.map fun (_, declName, creates) => (declName, creates)
    inputFields := inputFields.map fun (field, domain, _, _) => (field, domain)
    results := results.map Prod.fst })


/-! ### The machine command

A machine is the transition relation the glossary calls a Machine. It names the entity it tracks, the
structure it keeps per instance, the `phase` values that end an instance, and one step function per
action it steps on.

`DESIGN.md` writes a machine's logic as rows. The user's 2026-09-12 decision replaced rows with
ordinary Lean step functions, enumerated at elaboration into the same finite table the rows produced,
which task `.3` proved produces the same rows and therefore the same Behavior Fingerprint. What the
command does is name the pieces a reader cannot infer from Lean -- which function steps on which
action, which entity the machine tracks, which states end an instance -- and transcribe the functions
into the table. It defines no behavior of its own. -/

/-- One indented key of a `machine` declaration. -/
declare_syntax_cat machineKey
syntax "for:" ident : machineKey
syntax "state:" ident : machineKey
syntax "ends:" "[" ident,+ "]" : machineKey
syntax "starts:" "[" ident,+ "]" : machineKey
syntax "timers:" "[" ident,+ "]" : machineKey
syntax "unobservable:" "[" ident,+ "]" : machineKey
syntax "setup:" withPosition((colGe ident ":" ident)+) : machineKey
syntax "evidence:" withPosition((colGe ident ":" ident)+) : machineKey
syntax "steps:" withPosition((colGe ident ":" ident)+) : machineKey
syntax "refines:" ident : machineKey
syntax "map:" ident : machineKey

@[run_parser_attribute_hooks] private def machineKeyword := declarationKeyword "machine"

private def undeclaredMachineEntityMessage (spelling : Name) : String :=
  s!"'{spelling}' is not an entity declared by an `entity` command; a machine tracks one entity's \
instances, so `for:` names one"

private def undeclaredStepActionMessage (spelling : Name) : String :=
  s!"'{spelling}' is not an action declared by an `action` command; a `steps:` line names the action \
its function steps on"

private def alreadyDeclaredMessage (declared : Name) : String :=
  s!"'{declared}' is already declared; a machine takes the name it is given, and checking this one \
would be checking something else that happens to share it"

private def unprovenTableMessage (states actions : Nat) : String :=
  s!"this machine's canonical-table law did not check, so the Model it would declare carries \
`sorryAx` while reading as complete -- and every value derived from the table rests on that law. If \
the errors above are a timeout, {states} states over {actions} action classes is past what the proof \
elaborates and the state structure has to be smaller or bounded; otherwise they say what else went \
wrong"

private def machineTooLargeMessage (states actions bound : Nat) : String :=
  s!"enumerating {states} states over {actions} action classes is {states * actions} steps, and the \
bound is {bound}; a machine this size is bounded by its Limits or by symmetry, not walked"

private def stuckStateMessage (witness : String) : String :=
  s!"the machine reaches '{witness}', does not end there, and can take no step from it; either a \
step is missing or '{witness}' belongs under `ends:`"

private def unnamedTimerMessage (spelling : String) : String :=
  s!"no `steps:` line names the timer '{spelling}'; a timer is `system` behaviour written as a step \
function, and one that never fires is a timer the machine does not have"

private def silentTimerMessage (spelling : String) : String :=
  s!"the timer '{spelling}' fires and records nothing an `evidence:` line names, so no Contract can \
tell it fired; give it evidence, or declare it `unobservable:` and every Case whose path uses it \
carries a Known Gap"

private def idleTimerMessage (spelling : String) : String :=
  s!"the timer '{spelling}' is never enabled: its step function returns nothing in every state, so \
the timer never fires and the machine does not have it"

private def notATimerMessage (spelling : String) : String :=
  s!"'{spelling}' is not a timer of this machine; `unobservable:` names a timer whose firing the \
realization records nowhere, and an action a party takes is driven rather than observed"

private def observableTimerMessage (spelling : String) : String :=
  s!"the timer '{spelling}' records evidence, so a Contract can tell it fired; `unobservable:` is \
for a firing nothing records, and declaring an observable one would put a Known Gap in every Case \
that does not need it"

private def unreturnedEvidenceMessage (spelling : String) (returned : String) : String :=
  s!"no step of this machine returns the fact '{spelling}', so nothing it confirms ever happens; \
the steps return {returned}"

private def duplicateStepMessage (spelling : String) : String :=
  s!"the machine steps on '{spelling}' twice; one action has one step function, and two would not \
say which one applies"

private def notAStateStructureMessage (spelling : Name) : String :=
  s!"'{spelling}' is not a finite state structure; a machine's `state:` names a `structure` whose \
fields are all finite, so its members can be enumerated"

private def unknownEndMessage (spelling : String) (field : String) (admitted : String) : String :=
  s!"'{spelling}' is not a value of the state's '{field}' field; `ends:` names the values of one \
state field that end an instance, and that field admits {admitted}"

private def endsAcrossFieldsMessage (earlier later : String) : String :=
  s!"`ends:` names values of two different state fields, '{earlier}' and '{later}'; an instance ends \
on the values of one field"

private def stepStateMessage (declName : Name) (carried expected : Expr) : MessageData :=
  m!"'{declName}' takes {carried} where this machine's state is {expected}; a step function's first \
argument is the state it steps from"

private def stepArgumentMessage (declName : Name) (index : Nat) (expected : Name)
    (carried : Expr) : MessageData :=
  m!"'{declName}' takes {carried} where this action's input {index + 1} is '{expected}'; a step \
function's arguments after the state are the action's own input domains, in order"

private def stepSignatureMessage (declName : Name) : String :=
  s!"'{declName}' is not a step function; a `steps:` line names one of the shape \
`State -> <the action's input domains, curried> -> List (Step State Outcome Fact)`"

private def disagreeingDomainsMessage (outcome fact foundOutcome foundFact : Name) : String :=
  s!"this machine's steps return outcomes in '{outcome}' and facts in '{fact}', and this one returns \
'{foundOutcome}' and '{foundFact}'; one machine has one outcome domain and one fact domain"

private def ambiguousStateValueMessage (spelling : String) (fields : String) : String :=
  s!"'{spelling}' is a value of more than one state field ({fields}), so which field it names is \
undecided; a machine begins and ends on the values of one field, named unambiguously"

private def noEndsFieldMessage (spelling : String) : String :=
  s!"'{spelling}' is not a value of any field of the machine's state structure"


private def mapWithoutRefinesMessage : String :=
  "`map:` says how this machine's state reads as another machine's, so `refines:` names that \
machine; a `map:` without `refines:` maps to nothing"

private def refinesWithoutMapMessage : String :=
  "`refines:` names the machine this one refines, and `map:` names the function that reads this \
machine's state as its state; a refinement needs both"

private def undeclaredRefinedMessage (spelling : Name) : String :=
  s!"'{spelling}' is not a machine declared by a `machine` command; `refines:` names the product \
machine this one refines"

private def mapShapeMessage (declName state refined : Name) : String :=
  s!"'{declName}' is not a map from this machine's state to the refined machine's; `map:` names a \
function `{state} → {refined}`"

private def abstractFieldClashMessage (field : String) : String :=
  s!"the state structure has a field named '{field}', which is the name the state this machine \
reads as in the refined machine takes; rename the field"

private def undecidedRefinementMessage (rows : Nat) : String :=
  s!"the refinement did not decide over its {rows} rows, so no witness was synthesized; the \
derived step mapping checked, so this is the size of the machine rather than its rows"

/-- Elaborate one of a machine's generated declarations.

A machine's definitions are as long as its state space: `DESIGN.md` section 3's protocol machine has
224 states, so its key array, its terminal list and its start list are list literals of that length,
and a literal that long nests deeper than a Lean file's default recursion limit. The limit is raised
on the generated declaration and nowhere else, because the length is the machine's size rather than
anything an author wrote, and an author who hit the file's own limit should still hear about it. -/
private def elabGenerated (generated : TSyntax `command) : CommandElabM Unit := do
  -- Heartbeats for the same reason as depth: the canonical-table law is proved by `rfl` over a
  -- table as large as the machine, so its cost is the machine's size and not anything an author
  -- wrote. A machine still too large for these refuses, because the command reads its own axioms.
  elabCommand (← `(command|
    set_option maxRecDepth 65536 in
    set_option maxHeartbeats 1000000 in
    $generated:command))

/-- What one `steps:` line resolved to: the action it steps on, that action's input domains, and the
function the author wrote. -/
private structure ResolvedStep where
  action : Registry.ActionEntry
  /-- Each input field's domain type, in declaration order: the function's arguments after the
  state, curried in that order. -/
  domains : Array Name
  function : Ident

/-- The Outcome and Fact domains a step function returns, read off its own type.

They are not keys the author writes. A step function's result type is
`List (Step State Outcome Fact)`, so the machine's outcome and fact domains are already written down
in the function the author wrote; asking for them again would be asking twice and admitting the
answers to disagree. -/
private def stepResultDomains (stepRef : Ident) (declName stateDecl : Name)
    (domains : Array Name) : CommandElabM (Name × Name) := do
  let arity := domains.size
  let some info := (← getEnv).find? declName
    | throwErrorAt stepRef (stepSignatureMessage declName)
  liftTermElabM do
    Meta.forallBoundedTelescope info.type (some (arity + 1)) fun taken result => do
      -- `forallBoundedTelescope` takes *at most* the bound, so a function with fewer arguments than
      -- the action declares inputs would reach the unification below with a function type and fail
      -- somewhere inside the synthesized dispatcher instead of here, at the line that named it.
      unless taken.size == arity + 1 do
        throwErrorAt stepRef (stepSignatureMessage declName)
      -- The first argument is the state. Without this a function over another type reaches the
      -- dispatcher, fails there, and is then reported by the axiom guard as a machine too large to
      -- prove -- a verdict about size on what is a type error.
      let carriedState ← Meta.inferType taken[0]!
      let expectedState ← mkConstWithLevelParams stateDecl
      unless (← Meta.isDefEq carriedState expectedState) do
        throwErrorAt stepRef (stepStateMessage declName carriedState expectedState)
      -- Each argument after it is the action's own input domain, in order. Two actions of the same
      -- arity over different enums are an ordinary slip, and without this the mismatch surfaces
      -- inside the synthesized dispatcher rather than at the line that named the function.
      for index in [0:domains.size] do
        let carried ← Meta.inferType taken[index + 1]!
        let expected ← mkConstWithLevelParams domains[index]!
        unless (← Meta.isDefEq carried expected) do
          throwErrorAt stepRef (stepArgumentMessage declName index domains[index]! carried)
      -- Unified against the shape rather than matched on the head constant: `Umpire.Step` is an
      -- abbreviation, so a match on what it reduces to would name a type the author never wrote,
      -- and would break the day the abbreviation moves.
      let stateType ← mkConstWithLevelParams stateDecl
      -- A Model's domains are all `Type`, so the holes are `Type` holes; a `Sort ?u` hole unifies
      -- with the universe rather than with the domain and reports the mismatch in `Step`'s own
      -- application, where an author cannot see what went wrong.
      let anyType := Expr.sort (Level.succ Level.zero)
      let outcome ← Meta.mkFreshExprMVar anyType
      let fact ← Meta.mkFreshExprMVar anyType
      let expected ← Meta.mkAppM ``List #[← Meta.mkAppM ``Umpire.Step #[stateType, outcome, fact]]
      unless (← Meta.isDefEq result expected) do
        throwErrorAt stepRef (stepSignatureMessage declName)
      let some outcomeName := (← instantiateMVars outcome).getAppFn.constName?
        | throwErrorAt stepRef (stepSignatureMessage declName)
      let some factName := (← instantiateMVars fact).getAppFn.constName?
        | throwErrorAt stepRef (stepSignatureMessage declName)
      pure (outcomeName, factName)

elab doc?:(docComment)? machineKeyword name:ident keys:machineKey+ : command => do
  -- A machine's generated definitions are as long as its state space: `DESIGN.md` section 3's
  -- protocol machine has 224 states, so its key array, its terminal list and its enumerated table
  -- are lists of that length, and building a list term that long recurses deeper than a Lean file
  -- normally does. The depth is raised for what this command generates and nothing else, because
  -- the length is the machine's size rather than anything an author wrote.
  let mut entity : Option Registry.EntityEntry := none
  let mut stateRef : Option Ident := none
  let mut endRefs : Array Ident := #[]
  let mut startRefs : Array Ident := #[]
  let mut timerRefs : Array Ident := #[]
  let mut unobservableRefs : Array Ident := #[]
  let mut setupParameters : Array (String × Name) := #[]
  let mut evidenceRefs : Array (Ident × Ident) := #[]
  let mut stepRefs : Array (Ident × Ident) := #[]
  let mut refinesRef : Option Ident := none
  let mut mapRef : Option Ident := none
  for entry in keys do
    match entry with
    | `(machineKey| refines: $refinedRef:ident) => do
        if refinesRef.isSome then throwErrorAt entry (duplicateKeyMessage "machine" "refines:")
        refinesRef := some refinedRef
    | `(machineKey| map: $mapFn:ident) => do
        if mapRef.isSome then throwErrorAt entry (duplicateKeyMessage "machine" "map:")
        mapRef := some mapFn
    | `(machineKey| for: $entityRef:ident) => do
        if entity.isSome then throwErrorAt entry (duplicateKeyMessage "machine" "for:")
        let declName? ← try
            some <$> liftTermElabM (realizeGlobalConstNoOverloadWithInfo entityRef)
          catch failure =>
            if failure.isInterrupt || failure.isMaxRecDepth then throw failure else pure none
        match declName?.bind (Registry.entity? (← getEnv)) with
        | some declared => entity := some declared
        | none => throwErrorAt entityRef (undeclaredMachineEntityMessage entityRef.getId)
    | `(machineKey| state: $typeRef:ident) => do
        if stateRef.isSome then throwErrorAt entry (duplicateKeyMessage "machine" "state:")
        stateRef := some typeRef
    | `(machineKey| ends: [$members,*]) => do
        if !endRefs.isEmpty then throwErrorAt entry (duplicateKeyMessage "machine" "ends:")
        endRefs := members.getElems
    -- The antiquotation names avoid `actions`, because `actions:` is already a token and
    -- `$actions:ident` would tokenize as `$` and that token rather than as an antiquotation.
    | `(machineKey| timers: [$members,*]) => do
        if !timerRefs.isEmpty then throwErrorAt entry (duplicateKeyMessage "machine" "timers:")
        timerRefs := members.getElems
    | `(machineKey| unobservable: [$members,*]) => do
        if !unobservableRefs.isEmpty then
          throwErrorAt entry (duplicateKeyMessage "machine" "unobservable:")
        unobservableRefs := members.getElems
    | `(machineKey| setup: $[$parameter:ident : $domain:ident]*) => do
        if !setupParameters.isEmpty then throwErrorAt entry (duplicateKeyMessage "machine" "setup:")
        for named in parameter, ranged in domain do
          setupParameters := setupParameters.push
            (named.getId.toString, ← resolveSetupDomain ranged)
    -- `recorded`, not `fact`: `fact:` is already a token of the `property` command's `require:`
    -- block,
    -- so `$fact:ident` tokenizes as `$` and that token rather than as an antiquotation. The same
    -- trap as `$actions:ident`, one category over.
    | `(machineKey| evidence: $[$recorded:ident : $observed:ident]*) => do
        if !evidenceRefs.isEmpty then throwErrorAt entry (duplicateKeyMessage "machine" "evidence:")
        for named in recorded, seen in observed do
          evidenceRefs := evidenceRefs.push (named, seen)
    | `(machineKey| starts: [$members,*]) => do
        if !startRefs.isEmpty then throwErrorAt entry (duplicateKeyMessage "machine" "starts:")
        startRefs := members.getElems
    -- The antiquotation names avoid `actions`, because `actions:` is already a token and
    -- `$actions:ident` would tokenize as `$` and that token rather than as an antiquotation.
    | `(machineKey| steps: $[$stepped:ident : $written:ident]*) => do
        if !stepRefs.isEmpty then throwErrorAt entry (duplicateKeyMessage "machine" "steps:")
        for takenOn in stepped, function? in written do
          stepRefs := stepRefs.push (takenOn, function?)
    | _ => throwErrorAt entry "unsupported machine key"
  let some declaredEntity := entity
    | throwErrorAt name (missingKeyMessage "machine" "for:")
  let some stateType := stateRef
    | throwErrorAt name (missingKeyMessage "machine" "state:")
  if stepRefs.isEmpty then throwErrorAt name (missingKeyMessage "machine" "steps:")
  -- Without `starts:` a Model begins nowhere, so nothing is reachable, every Property holds
  -- vacuously and the stuck check passes by having nothing to check.
  if startRefs.isEmpty then throwErrorAt name (missingKeyMessage "machine" "starts:")
  -- Without `ends:` no state is terminal: `terminal` names nothing, a Search runs to its limit on
  -- every path, the stuck check has no state it is allowed to stop in, and a Property that requires
  -- an instance to finish holds by never being reached. A machine says where it ends.
  if endRefs.isEmpty then throwErrorAt name (missingKeyMessage "machine" "ends:")
  -- A refinement is a `refines:` and a `map:` together: the machine refined, and the function that
  -- reads this machine's state as its state. Neither says anything without the other.
  let refinement ← match refinesRef, mapRef with
    | none, none => pure none
    | some refinedRef, none => throwErrorAt refinedRef refinesWithoutMapMessage
    | none, some mapFn => throwErrorAt mapFn mapWithoutRefinesMessage
    | some refinedRef, some mapFn => do
        let declName? ← try
            some <$> liftTermElabM (realizeGlobalConstNoOverloadWithInfo refinedRef)
          catch failure =>
            if failure.isInterrupt || failure.isMaxRecDepth then throw failure else pure none
        let some refinedMachine := declName?.bind (Registry.machine? (← getEnv))
          | throwErrorAt refinedRef (undeclaredRefinedMessage refinedRef.getId)
        let some refinedModel := Registry.model? (← getEnv) refinedMachine.declName
          | throwErrorAt refinedRef (undeclaredRefinedMessage refinedRef.getId)
        pure (some (refinedRef, refinedMachine, refinedModel, mapFn))
  -- The state's members are the machine's states. A structure is an inductive of one constructor, so
  -- the same walk that writes an action's classes writes them, and the same refusals apply.
  let stateDecl ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo stateType)
  unless isStructure (← getEnv) stateDecl do
    throwErrorAt stateType (notAStateStructureMessage stateType.getId)
  let stateMembers ← domainMembers stateType stateDecl
  -- The map is a function from this machine's state to the refined machine's, checked here at the
  -- line that named it rather than inside the declarations generated from it.
  if let some (_, _, refinedModel, mapFn) := refinement then
    let mapDecl ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo mapFn)
    let some info := (← getEnv).find? mapDecl
      | throwErrorAt mapFn (mapShapeMessage mapDecl stateDecl refinedModel.stateType)
    liftTermElabM do
      Meta.forallBoundedTelescope info.type (some 1) fun taken result => do
        let shaped ← if taken.size != 1 then pure false else do
          let carried ← Meta.inferType taken[0]!
          let expectedState ← mkConstWithLevelParams stateDecl
          let expectedRefined ← mkConstWithLevelParams refinedModel.stateType
          pure ((← Meta.isDefEq carried expectedState) && (← Meta.isDefEq result expectedRefined))
        unless shaped do
          throwErrorAt mapFn (mapShapeMessage mapDecl stateDecl refinedModel.stateType)
  -- Every `steps:` line, resolved before anything is generated from any of them.
  let timerNames := timerRefs.map fun timerRef => timerRef.getId.getString!
  let mut steps : Array ResolvedStep := #[]
  let mut timersStepped : Array String := #[]
  for (actionRef, functionRef) in stepRefs do
    let spelling := actionRef.getId.getString!
    if timerNames.contains spelling then
      -- A timer is `system` behaviour: it takes no input, so its step function takes only the state,
      -- and it is an action of the machine's domain like any other. Nothing a realization can drive.
      if timersStepped.contains spelling then
        throwErrorAt actionRef (duplicateStepMessage spelling)
      timersStepped := timersStepped.push spelling
      steps := steps.push {
        action := { declName := .anonymous, name := spelling, party := "system"
                    subject := none, inputFields := #[], results := none }
        domains := #[]
        function := functionRef }
    else
      let declName? ← try
          some <$> liftTermElabM (realizeGlobalConstNoOverloadWithInfo actionRef)
        catch failure =>
          if failure.isInterrupt || failure.isMaxRecDepth then throw failure else pure none
      let some declared := declName?.bind (Registry.action? (← getEnv))
        | throwErrorAt actionRef (undeclaredStepActionMessage actionRef.getId)
      if steps.any fun seen => seen.action.name == declared.name then
        throwErrorAt actionRef (duplicateStepMessage declared.name)
      steps := steps.push {
        action := declared
        domains := declared.inputFields.map Prod.snd
        function := functionRef }
  -- A timer no step names never fires, so it is a timer the machine does not have.
  for timerRef in timerRefs do
    unless timersStepped.contains timerRef.getId.getString! do
      throwErrorAt timerRef (unnamedTimerMessage timerRef.getId.getString!)
  -- The machine's Action domain is synthesized, one constructor per action it steps on, carrying
  -- that action's input fields. The author writes one function per action over that action's own
  -- inputs; the enumerator walks one `State -> Action -> List (Step ...)`, and this is what makes
  -- the first into the second without the author writing a sum type by hand.
  let declared := (← getCurrNamespace) ++ name.getId
  let declaredBefore := (← getEnv).contains declared
  let actionType := mkIdentFrom name (name.getId ++ `Action)
  let constructors ← steps.mapM fun resolved => do
    let constructorName := mkIdent (Name.mkSimple resolved.action.name)
    let binders ← resolved.action.inputFields.mapM fun (field, domain) =>
      `(Lean.Parser.Term.bracketedBinderF|
        ($(mkIdent (Name.mkSimple field)) : $(mkIdent domain)))
    `(Lean.Parser.Command.ctor| | $constructorName:ident $binders*)
  elabGenerated (← `(command|
    inductive $actionType where
      $constructors:ctor*
      deriving BEq, DecidableEq, Repr, Umpire.Command.Finite))
  -- The Outcome and Fact domains are the ones the author's own functions return. Every step function
  -- of one machine returns into one pair, so the first is read and the rest are required to agree.
  let mut domains : Option (Name × Name) := none
  for resolved in steps do
    let declName ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo resolved.function)
    let found ← stepResultDomains resolved.function declName stateDecl resolved.domains
    match domains with
    | none => domains := some found
    | some expected =>
        unless found == expected do
          throwErrorAt resolved.function (disagreeingDomainsMessage expected.1 expected.2 found.1 found.2)
  let some (outcomeType, factType) := domains
    | throwErrorAt name (missingKeyMessage "machine" "steps:")
  -- One total step function over the synthesized domain, dispatching to the author's own. This is
  -- the only thing the enumerator sees; nothing downstream knows there was more than one function.
  let stateBinder := mkIdent `state
  let dispatchArms ← steps.mapM fun resolved => do
    let constructorName := mkIdent (Name.mkSimple resolved.action.name)
    let binders := (List.range resolved.domains.size).toArray.map fun index =>
      mkIdent (Name.mkSimple s!"carried{index}")
    let arguments : Array Term := #[stateBinder] ++ binders
    `(Lean.Parser.Term.matchAltExpr|
      | .$constructorName:ident $binders* => $(resolved.function) $arguments*)
  let stepName := mkIdentFrom name (name.getId ++ `step)
  elabGenerated (← `(command|
    def $stepName ($stateBinder : $stateType) :
        $actionType → List (Umpire.Step $stateType $(mkIdent outcomeType) $(mkIdent factType))
      $dispatchArms:matchAlt*))
  -- The Action domain's own members, read back now that it exists.
  let actionDecl := (← getCurrNamespace) ++ actionType.getId
  -- The Action catalog is emitted in canonical order rather than in the order the `steps:` lines
  -- happen to be written. A Search admits a Model whose catalog is sorted by the Definition ID of
  -- each member, and those ids differ only in the member key, so sorting by the key is that order.
  -- It cannot be left to the author: a classed action contributes one member per assignment of its
  -- inputs, in its domain's member order, so no arrangement of `steps:` lines can sort
  -- `complete-succeeded`, `complete-failed` and `complete-canceled`.
  let actionMembers := (← domainMembers name actionDecl).mergeSort fun left right =>
    left.key ≤ right.key
  -- A row's key is its state's key and its action's key, and each of those is the member's position
  -- in the enumeration rather than a second rendering of it. Indexing `members` is what keeps the
  -- keys and the walk in step: they are the same list, read the same way.
  let keyArray : List ClassValue → Array Term := fun values =>
    (values.map fun value => Lean.quote value.key).toArray
  let stateKeysName := mkIdentFrom name (name.getId ++ `stateKeys)
  let actionKeysName := mkIdentFrom name (name.getId ++ `actionKeys)
  let rowKeyName := mkIdentFrom name (name.getId ++ `rowKey)
  elabGenerated (← `(command|
    def $stateKeysName : Array String := #[$(keyArray stateMembers),*]))
  elabGenerated (← `(command|
    def $actionKeysName : Array String := #[$(keyArray actionMembers),*]))
  -- The catalog as a list of its own members, in that same order. Everything that reads an action
  -- by position -- the key function, the enumeration, the declared Model -- reads this one list, so
  -- there is no second order for them to disagree about.
  let actionsName := mkIdentFrom name (name.getId ++ `actions)
  let actionTerms ← actionMembers.toArray.mapM ClassValue.term
  elabGenerated (← `(command|
    def $actionsName : List $actionType := [$actionTerms,*]))
  let stateKeyForName := mkIdentFrom name (name.getId ++ `stateKeyFor)
  elabGenerated (← `(command|
    def $stateKeyForName (state : $stateType) : String :=
      match (Umpire.Command.members (α := $stateType)).idxOf? state with
      | some at? => ($stateKeysName)[at?]!
      | none => ""))
  let actionKeyForName := mkIdentFrom name (name.getId ++ `actionKeyFor)
  elabGenerated (← `(command|
    def $actionKeyForName (taken : $actionType) : String :=
      match ($actionsName).idxOf? taken with
      | some at? => ($actionKeysName)[at?]!
      | none => ""))
  elabGenerated (← `(command|
    def $rowKeyName (state : $stateType) (taken : $actionType) : String :=
      $stateKeyForName state ++ "-" ++ $actionKeyForName taken))
  -- `ends:` names the values of one state field. Which field is not a key the author writes: the
  -- values name it, and naming values of two fields is the mistake the message reports.
  let memberFields : ClassValue → Array (String × ClassValue) := fun value =>
    match value with
    | .applied _ fields => fields
    | .atom _ => #[]
  -- The structure's own field order, so a message reads as the structure is written.
  let orderedFields : List String := match stateMembers.head? with
    | some (.applied _ fields) => fields.toList.map Prod.fst
    | _ => []
  let mut endField : Option String := none
  for endRef in endRefs do
    let spelling := endRef.getId.getString!
    -- Every member is looked at, not the first: a field carries the value in the member that holds
    -- it, and the member a machine ends in is rarely the one it starts in.
    let carriers := (stateMembers.flatMap fun member =>
      (memberFields member).toList.filterMap fun (field, value) =>
        match value with
        | .atom declared => if declared.getString! == spelling then some field else none
        | .applied constructor _ =>
            if constructor.getString! == spelling then some field else none).eraseDups
    let some field := carriers.head?
      | throwErrorAt endRef (noEndsFieldMessage spelling)
    -- Two fields that can both hold this value leave the machine's end undecided, and picking the
    -- first would decide it silently. The author writes which field they mean.
    if carriers.length > 1 then
      throwErrorAt endRef (ambiguousStateValueMessage spelling
        (", ".intercalate (orderedFields.filter carriers.contains)))
    match endField with
    | none => endField := some field
    | some seen =>
        unless seen == field do throwErrorAt endRef (endsAcrossFieldsMessage seen field)
  let endSpellings := endRefs.map fun endRef => endRef.getId.getString!
  let isTerminal : ClassValue → Bool := fun value =>
    match endField with
    | none => false
    | some field =>
        (memberFields value).any fun (carried, held) =>
          carried == field && endSpellings.any fun spelling =>
            match held with
            | .atom declared => declared.getString! == spelling
            | .applied constructor _ => constructor.getString! == spelling
  let terminalTerms ← (stateMembers.filter isTerminal).toArray.mapM ClassValue.term
  -- The walk is bounded. A step function is evaluated once per (state, action) pair whether or not
  -- the pair is enabled, so a machine whose state structure multiplies out past the bound is
  -- refused with both factors rather than enumerated part-way into a table smaller than the Model.
  let walked := stateMembers.length * actionMembers.length
  if walked > enumerationBound then
    throwErrorAt stateType
      (machineTooLargeMessage stateMembers.length actionMembers.length enumerationBound)
  let transitionsName := mkIdentFrom name (name.getId ++ `transitions)
  elabGenerated (← `(command|
    def $transitionsName :
        List (Umpire.FiniteTransitionRow $stateType $actionType
          $(mkIdent outcomeType) $(mkIdent factType)) :=
      Umpire.Command.enumerateOver $actionsName $rowKeyName $stepName))
  -- `starts:` names values of one field, the way `ends:` does. A state of several fields is not
  -- determined by one of them, so every other field takes its own first enumerated value: zero for a
  -- count, `false` for a flag, the first constructor for an enum. That is what a machine begins at,
  -- and saying it once here is why a `starts:` line names a phase rather than a whole structure.
  let heldValue : ClassValue → String → Option String := fun member field =>
    (memberFields member).findSome? fun (named, held) =>
      if named == field then
        some (match held with
          | .atom declared => declared.getString!
          | .applied constructor _ => constructor.getString!)
      else none
  let factMembersEarly ← domainMembers name factType
  let mut startTerms : Array Term := #[]
  let mut startKeys : Array String := #[]
  for startRef in startRefs do
    let spelling := startRef.getId.getString!
    let carriers := (stateMembers.flatMap fun member =>
      (memberFields member).toList.filterMap fun (field, value) =>
        match value with
        | .atom declared => if declared.getString! == spelling then some field else none
        | .applied constructor _ =>
            if constructor.getString! == spelling then some field else none).eraseDups
    let some field := carriers.head?
      | throwErrorAt startRef (noEndsFieldMessage spelling)
    if carriers.length > 1 then
      throwErrorAt startRef (ambiguousStateValueMessage spelling
        (", ".intercalate (orderedFields.filter carriers.contains)))
    -- The one member holding this value with every other field at its first: the members are in
    -- enumeration order and the first field varies slowest, so the first match is that member.
    let some member := stateMembers.find? fun candidate =>
        heldValue candidate field == some spelling
      | throwErrorAt startRef (noEndsFieldMessage spelling)
    -- The start-state list is canonical or the planner refuses the Model, and the refusal comes
    -- from admission rather than from the line that wrote it. The Action catalog needs no such rule
    -- because the command sorts it; a `starts:` line is the author's, and its order is the order
    -- the states are emitted in.
    if let some earlier := startKeys.back? then
      unless earlier < member.key do
        throwErrorAt startRef (unsortedInitialMessage member.key earlier)
    startTerms := startTerms.push (← member.term)
    startKeys := startKeys.push member.key
  -- An evidence line names a fact the steps return. One that names a fact no step returns confirms
  -- something that never happens, which is a mistake about the machine and not about the evidence.
  let factNames := factMembersEarly.map fun member => member.key
  -- A fact that carries fields has one member per assignment of them, and each member's key spells
  -- that assignment out. The mapping an `evidence:` line writes is the constructor's, not the
  -- member's: `DESIGN.md` section 3 records every `nexusOperationTimedOut` under one catalogued
  -- event name whichever of the three timers fired. So a line may name the constructor and cover
  -- its members -- which is also the only spelling an identifier admits, a member key being
  -- punctuated.
  let factConstructors := factMembersEarly.filterMap fun member =>
    match member with
    | .applied constructor _ => some constructor.getString!
    | .atom _ => none
  for (recordedRef, _) in evidenceRefs do
    let spelling := recordedRef.getId.getString!
    unless factNames.contains spelling || factConstructors.contains spelling do
      throwErrorAt recordedRef
        (unreturnedEvidenceMessage spelling (", ".intercalate factNames))
  -- The other side of an evidence line is the recorded data that confirms the fact. Most names need
  -- no declaration because the realization already carries them -- for Temporal, the generated
  -- history event kinds and the Testpilot Run Event kinds -- so the registry is asked first and the
  -- platform about whatever nothing declared. A Model file that imports no platform catalog module
  -- gets no check, the way a `schema:` line does.
  let declaredObservations := (Registry.observations (← getEnv)).map fun entry => entry.name
  for (_, observedRef) in evidenceRefs do
    let spelling := observedRef.getId.getString!
    unless declaredObservations.contains spelling do
      match ← (Umpire.Command.checkCatalog spelling : IO _) with
      | .ok () => pure ()
      | .error reason => throwErrorAt observedRef reason
  -- Which facts a Contract can read: the members an evidence line names, either as the member's own
  -- key or as the constructor covering it.
  let evidenceSpellings := evidenceRefs.map fun (recordedRef, _) => recordedRef.getId.getString!
  let evidencedMembers := factMembersEarly.filter fun member =>
    match member with
    | .atom spelling => evidenceSpellings.contains spelling.getString!
    | .applied constructor _ =>
        evidenceSpellings.contains member.key || evidenceSpellings.contains constructor.getString!
  let evidencedTerms ← evidencedMembers.toArray.mapM ClassValue.term
  -- A timer is `system` behaviour: nothing drives it, so the only way a Case can tell it fired is
  -- the evidence its rows record. One that records none is a step a Contract cannot see, which is
  -- what `unobservable:` says out loud and turns into a Known Gap.
  let silentName := mkIdentFrom name (name.getId ++ `silent)
  let idleName := mkIdentFrom name (name.getId ++ `idle)
  elabGenerated (← `(command|
    def $silentName : List String :=
      ($actionsName).filterMap fun taken =>
        let rows := ($transitionsName).filter fun row => row.action == taken
        if rows.isEmpty then none
        else if rows.any (fun row => row.results.any fun step => step.facts.any fun recorded =>
            (([$evidencedTerms,*] : List $(mkIdent factType))).contains recorded) then none
        else some ($actionKeyForName taken)))
  elabGenerated (← `(command|
    def $idleName : List String :=
      ($actionsName).filterMap fun taken =>
        if ($transitionsName).any (fun row => row.action == taken) then none
        else some ($actionKeyForName taken)))
  let silent ← liftTermElabM (evalStringList ((← getCurrNamespace) ++ silentName.getId))
  let idle ← liftTermElabM (evalStringList ((← getCurrNamespace) ++ idleName.getId))
  let unobservableNames := unobservableRefs.map fun timerRef => timerRef.getId.getString!
  for timerRef in timerRefs do
    let spelling := timerRef.getId.getString!
    if idle.contains spelling then
      throwErrorAt timerRef (idleTimerMessage spelling)
    if silent.contains spelling && !unobservableNames.contains spelling then
      throwErrorAt timerRef (silentTimerMessage spelling)
  for unobservedRef in unobservableRefs do
    let spelling := unobservedRef.getId.getString!
    unless timerNames.contains spelling do
      throwErrorAt unobservedRef (notATimerMessage spelling)
    unless silent.contains spelling do
      throwErrorAt unobservedRef (observableTimerMessage spelling)
  let terminalName := mkIdentFrom name (name.getId ++ `ends)
  let startsName := mkIdentFrom name (name.getId ++ `starts)
  elabGenerated (← `(command|
    def $terminalName : List $stateType := [$terminalTerms,*]))
  elabGenerated (← `(command|
    def $startsName : List $stateType := [$startTerms,*]))
  -- The declared Model, on the enumerated rows. Everything downstream -- the Behavior Fingerprint,
  -- Search, Contract lowering, `umpire-inspect` -- reads this and never sees a step function.
  let setupType := mkIdentFrom name (name.getId ++ `Setup)
  -- The setup domain is the command's own: nothing else in a Model file mentions it, and varying a
  -- machine's table over its `setup:` parameters is task `.5`'s, so a machine carries one setup.
  -- Its constructor is named after the machine's first start state as the `starts:` line spells
  -- it -- the phase, not the key naming every field, which a structured state punctuates into
  -- something no catalog key admits -- which is the convention the `model` command set and what
  -- keeps a migrated Model's canonical setup key where it was.
  let setupConstructor := Name.mkSimple (startRefs[0]!.getId.getString!)
  let setupName := mkIdentFrom name (name.getId ++ `Setup ++ setupConstructor)
  -- The constructor is built rather than written: an identifier inside a quotation is hygienic, so a
  -- literal one would be declared under a macro scope and no name outside this command could
  -- reach it.
  elabGenerated (← `(command|
    inductive $setupType where
      | $(mkIdent setupConstructor):ident
      deriving BEq, DecidableEq, Repr))
  let outcomeMembers ← domainMembers name outcomeType
  let factMembers ← domainMembers name factType
  let keyList : List ClassValue → Array Term := fun values =>
    (values.map fun value => Lean.quote value.key).toArray
  -- An outcome and a fact are keyed the way a state is, so a `property` can name what a step
  -- produced in the same spelling its clauses carry.
  let outcomeKeysName := mkIdentFrom name (name.getId ++ `outcomeKeys)
  let factKeysName := mkIdentFrom name (name.getId ++ `factKeys)
  let outcomeKeyForName := mkIdentFrom name (name.getId ++ `outcomeKeyFor)
  let factKeyForName := mkIdentFrom name (name.getId ++ `factKeyFor)
  elabGenerated (← `(command|
    def $outcomeKeysName : Array String := #[$(keyList outcomeMembers),*]))
  elabGenerated (← `(command|
    def $factKeysName : Array String := #[$(keyList factMembers),*]))
  elabGenerated (← `(command|
    def $outcomeKeyForName (produced : $(mkIdent outcomeType)) : String :=
      match (Umpire.Command.members (α := $(mkIdent outcomeType))).idxOf? produced with
      | some at? => ($outcomeKeysName)[at?]!
      | none => ""))
  elabGenerated (← `(command|
    def $factKeyForName (recorded : $(mkIdent factType)) : String :=
      match (Umpire.Command.members (α := $(mkIdent factType))).idxOf? recorded with
      | some at? => ($factKeysName)[at?]!
      | none => ""))
  -- A refining machine reads each of its states as one of the refined machine's, and carries that
  -- state as a field named after the refined machine: a Property on the refined machine is a claim
  -- about that field here, read apart from the state the way any field is. The keys are computed
  -- by the map itself, over the members, so the field says what the map says and nothing else.
  let abstractKeys ← match refinement with
    | none => pure #[]
    | some (refinedRef, refinedMachine, _, mapFn) => do
        if orderedFields.contains refinedMachine.name then
          throwErrorAt refinedRef (abstractFieldClashMessage refinedMachine.name)
        let abstractKeysName := mkIdentFrom name (name.getId ++ `abstractKeys)
        let refinedStateKeyFor := mkIdent (refinedMachine.declName ++ `stateKeyFor)
        elabGenerated (← `(command|
          def $abstractKeysName : List String :=
            (Umpire.Command.members (α := $stateType)).map fun state =>
              $refinedStateKeyFor ($mapFn state)))
        pure (← liftTermElabM
          (evalStringList ((← getCurrNamespace) ++ abstractKeysName.getId))).toArray
  -- Each state's fields, in the structure's own field order: the field's name and the member this
  -- state holds it at. A Contract compares `attempts` as a number and `phase` as an enum, and
  -- reading them back out of the state key is the parsing the key exists to avoid.
  let stateFieldList ← stateMembers.toArray.mapIdxM fun index member => do
    let held := match member with
      | .applied _ fields => fields
      | .atom _ => #[]
    let held := match refinement, abstractKeys[index]? with
      | some (_, refinedMachine, _, _), some key =>
          held.push (refinedMachine.name, ClassValue.atom (Name.mkSimple key))
      | _, _ => held
    let pairs ← held.mapM fun (field, value) =>
      `(term| ($(Lean.quote field), $(Lean.quote value.key)))
    `(term| [$pairs,*])
  let setupParameterTerms : Array Term := setupParameters.map fun (parameter, _) =>
    Lean.quote parameter
  -- Each action member as its action's name and the class it assigns each input field, spelled
  -- the way an `examples:` line spells a class, so a Case can tell which claims its path makes.
  let actionClassTerms : Array Term ← actionMembers.toArray.mapM fun member => do
    let (actionName, held) := match member with
      | .atom spelling => (spelling.getString!, #[])
      | .applied constructor fields => (constructor.getString!, fields)
    let pairs ← held.mapM fun (field, value) =>
      `(term| ($(Lean.quote field), $(Lean.quote value.render)))
    `(term| ($(Lean.quote actionName), [$pairs,*]))
  let names ← `(term|
    { declaration := $(Lean.quote name.getId.toString)
      roleName := $(Lean.quote declaredEntity.name)
      setup := $(Lean.quote setupConstructor.toString)
      stateKeys := [$(keyList stateMembers),*]
      stateFields := [$stateFieldList,*]
      actionKeys := [$(keyList actionMembers),*]
      outcomeKeys := [$(keyList outcomeMembers),*]
      factKeys := [$(keyList factMembers),*]
      setupParameters := [$setupParameterTerms,*]
      actionClasses := [$actionClassTerms,*] })
  -- A state the Model reaches, does not end in, and can take no step from is where a Search stops
  -- without having finished. The table is what knows, so the check runs on the emitted table and is
  -- reported back at the `steps:` block that produced it.
  let stuckName := mkIdentFrom name (name.getId ++ `stuck)
  elabGenerated (← `(command|
    def $stuckName : Option String :=
      (Umpire.Command.stuckState $startsName $terminalName $transitionsName).map $stateKeyForName))
  match ← liftTermElabM (evalStuckState ((← getCurrNamespace) ++ stuckName.getId)) with
  | none => pure ()
  | some witness =>
      let anchor := (stepRefs[0]?.map Prod.fst).getD name
      throwErrorAt anchor (stuckStateMessage witness)

  let origin ← originTerm
  elabGenerated (← `(command|
    def $name := Umpire.Command.declareModel $origin $names ($setupName)
      (Umpire.Command.members (α := $stateType))
      ($actionsName)
      (Umpire.Command.members (α := $(mkIdent outcomeType)))
      (Umpire.Command.members (α := $(mkIdent factType)))
      ($startsName) ($terminalName) ($transitionsName)
      (by exact ⟨rfl, rfl, rfl⟩)))
  -- `elabCommand` logs a failure rather than throwing it, so a table whose canonical-table law did
  -- not check would be declared anyway, carrying `sorryAx` and looking complete. Everything reads
  -- that law -- the Behavior Fingerprint, Search, Contract lowering -- and only `#print axioms`
  -- would say otherwise, so the command says it here instead.
  -- A name the file already declared would be inspected instead of this machine's, and a clean
  -- axiom set on somebody else's constant says nothing about this one.
  if declaredBefore then
    throwErrorAt name (alreadyDeclaredMessage declared)
  if (← getEnv).contains declared then
    let axioms ← liftCoreM (Lean.collectAxioms declared)
    if axioms.contains ``sorryAx then
      throwErrorAt name (unprovenTableMessage stateMembers.length actionMembers.length)
  else
    throwErrorAt name (unprovenTableMessage stateMembers.length actionMembers.length)
  -- The refinement: the morphism this machine reads through, the step mapping derived from it and
  -- reported at the `map:` line, and the witness -- the simulation's obligations over the two
  -- tables, decided by the kernel over the rows rather than written by the author.
  if let some (refinedRef, refinedMachine, refinedModel, mapFn) := refinement then
    let refined := mkIdent refinedMachine.declName
    let pairTerms := fun (pairs : List (String × String)) =>
      pairs.toArray.mapM fun (protocol, product) =>
        `(term| ($(Lean.quote protocol), $(Lean.quote product)))
    let outcomePairs ← pairTerms
      (sameNamedPairs (outcomeMembers.map (·.key)) refinedModel.outcomes.toList)
    let factPairs ← pairTerms (sameNamedPairs (factMembers.map (·.key)) refinedModel.facts.toList)
    let refinedOutcomeType := mkIdent refinedModel.outcomeType
    let refinedFactType := mkIdent refinedModel.factType
    let abstractionName := mkIdentFrom name (name.getId ++ `abstraction)
    elabGenerated (← `(command|
      def $abstractionName : Umpire.RefinementMorphism $setupType $stateType
          $(mkIdent outcomeType) $(mkIdent factType)
          $(mkIdent (refinedMachine.declName ++ `Setup)) $(mkIdent refinedModel.stateType)
          $refinedOutcomeType $refinedFactType :=
        Umpire.Command.refinementMorphism ($refined).setupValue $mapFn
          $outcomeKeyForName (Umpire.Command.members (α := $refinedOutcomeType))
          $(mkIdent (refinedMachine.declName ++ `outcomeKeyFor)) [$outcomePairs,*]
          $factKeyForName (Umpire.Command.members (α := $refinedFactType))
          $(mkIdent (refinedMachine.declName ++ `factKeyFor)) [$factPairs,*]))
    let reportName := mkIdentFrom name (name.getId ++ `refinement)
    elabGenerated (← `(command|
      def $reportName : Umpire.Command.RefinementReport :=
        Umpire.Command.deriveRefinement ($name) ($refined) $abstractionName
          $(Lean.quote name.getId.toString) $(Lean.quote refinedMachine.name)))
    let report ← liftTermElabM (evalRefinementReport ((← getCurrNamespace) ++ reportName.getId))
    if let some rejected := report.rejected then
      throwErrorAt mapFn rejected
    let witnessName := mkIdentFrom name (name.getId ++ `refines)
    elabGenerated (← `(command|
      theorem $witnessName :
          Umpire.TableRefinement ($name).table ($refined).table $abstractionName :=
        Umpire.TableRefinement.ofChecked (by decide +kernel)))
    let witness := (← getCurrNamespace) ++ witnessName.getId
    let decided ← if (← getEnv).contains witness then
        pure !(← liftCoreM (Lean.collectAxioms witness)).contains ``sorryAx
      else pure false
    unless decided do
      throwErrorAt refinedRef (undecidedRefinementMessage report.rows.length)
  -- A machine declares a Model, so `property`, `scenario` and `query` have to see one. What they
  -- resolve members by is the key the table carries -- a state's key names every field of the
  -- structure -- because that is what the emitted rows, the Behavior Fingerprint and a Contract all
  -- read. A machine over a one-field state keeps the bare spellings a `model` had.
  liftCoreM (Registry.recordModel {
    declName := declared
    role := declaredEntity.name
    stateType := stateDecl
    actionType := actionDecl
    outcomeType
    factType
    «states» := (stateMembers.map (·.key)).toArray
    «actions» := (actionMembers.map (·.key)).toArray
    «outcomes» := (outcomeMembers.map (·.key)).toArray
    «facts» := (factMembers.map (·.key)).toArray
    «starts» := startKeys })
  liftCoreM (Registry.recordMachine {
    declName := (← getCurrNamespace) ++ name.getId
    name := name.getId.toString
    id := ← definitionIdHere "machine" name.getId.toString
    entity := declaredEntity.declName
    stateType := stateDecl
    steps := steps.map fun resolved => (resolved.action.name, resolved.function.getId)
    actionDecls := steps.filterMap fun resolved =>
      if resolved.action.declName.isAnonymous then none else some resolved.action.declName
    timers := timerNames
    unobservable := unobservableNames
    evidence := evidenceRefs.map fun (recordedRef, observedRef) =>
      (recordedRef.getId.getString!, observedRef.getId.getString!)
    refines := refinement.map fun (_, refinedMachine, _, _) => refinedMachine.declName
    abstraction := match refinement with
      | some (_, _, _, mapFn) => mapFn.getId
      | none => .anonymous })


elab doc?:(docComment)? observationKeyword name:ident keys:observationKey+ : command => do
  let mut entity : Option Registry.EntityEntry := none
  let mut read : Option Ident := none
  for entry in keys do
    match entry with
    | `(observationKey| on: $entityRef:ident) => do
        if entity.isSome then throwErrorAt entry (duplicateKeyMessage "observation" "on:")
        entity := some (← resolveEntity entityRef)
    | `(observationKey| read: $readRef:ident) => do
        if read.isSome then throwErrorAt entry (duplicateKeyMessage "observation" "read:")
        read := some readRef
    | _ => throwErrorAt entry "unsupported observation key"
  let some declaredEntity := entity
    | throwErrorAt name (missingKeyMessage "observation" "on:")
  let some readRef := read
    | throwErrorAt name (missingKeyMessage "observation" "read:")
  let origin ← originTerm
  let nameKey := Lean.quote name.getId.toString
  let observationId ← definitionIdHere "observation" name.getId.toString
  elabCommand (← `(command|
    $[$doc?:docComment]? def $name : Umpire.Command.Observation := {
      id := Umpire.DefinitionId.of $(Lean.quote observationId)
      name := $nameKey
      entity := Umpire.DefinitionId.of $(Lean.quote declaredEntity.id)
      read := $(Lean.quote readRef.getId.toString)
      source := ($origin).source }))
  liftCoreM (Registry.recordObservation {
    declName := (← getCurrNamespace) ++ name.getId
    name := name.getId.toString
    entity := declaredEntity.declName
    read := readRef.getId.toString })

/-! ### The `set` command

A set groups Queries by purpose and binds every party except `system`: `driven` when the Case's own
Program performs the party's actions, `observed` when the world does and the verifier reads which
class occurred. A functional set compiles each `find` Query to one Case; a canary set's Queries are
admitted for a deployment to run; an exploratory set names a coverage goal and a budget. The set is
Umpire's -- which Cases it produces, and through which realization, is the owning platform's
command to say -- so what it checks here is what the Model alone decides. -/

declare_syntax_cat setKey
syntax "purpose:" ident : setKey
syntax "bind:" withPosition((colGe ident ":" ident)+) : setKey
syntax "repeat:" ident : setKey
syntax "queries:" "[" ident,* "]" : setKey
syntax "machine:" ident : setKey
syntax "cover:" sepBy1(ident, "|") : setKey
syntax "budget:" ident : setKey

@[run_parser_attribute_hooks] private def setKeyword := declarationKeyword "set"

private def unknownPurposeMessage (spelling : String) : String :=
  s!"unknown purpose '{spelling}'; a set is functional, canary or exploratory"

private def unboundPartyMessage (party : String) : String :=
  s!"party '{party}' performs actions and this set does not bind it; a set binds every party \
except `system` to `driven` or `observed`"

private def systemBoundMessage : String :=
  "`system` is the implementation under test and performs no declared action; a set binds every \
other party and never `system`"

private def unknownPartyMessage (party : String) : String :=
  s!"no declared action is performed by party '{party}'; a set binds the parties the Model's \
actions name"

private def unknownBindingMessage (spelling : String) : String :=
  s!"'{spelling}' is not a binding; a party is `driven` (the Case performs its actions) or \
`observed` (the world does, and the verifier reads which class occurred)"

private def duplicateBindingMessage (party : String) : String :=
  s!"party '{party}' is bound twice"

private def repeatOutsideFunctionalMessage (purpose : String) : String :=
  s!"`repeat:` runs a functional set's Cases once per switch value, and a {purpose} set produces \
no Cases to repeat"

private def unknownSwitchMessage (spelling : String) (known : List String) : String :=
  if known.isEmpty then
    s!"'{spelling}' is not a switch any realization declares"
  else
    s!"'{spelling}' is not a switch the realization declares; declared: {", ".intercalate known}"

private def queriesOutsideMessage : String :=
  "an exploratory set covers rather than lists Queries; `queries:` belongs to a functional or \
canary set"

private def missingQueriesMessage (purpose : String) : String :=
  s!"a {purpose} set lists the Queries it runs under `queries:`"

private def verifyQueryInSetMessage (spelling purpose : String) : String :=
  s!"Query '{spelling}' verifies rather than finds; a {purpose} set's Queries each realize one \
selected trace, so each is a `find` form"

private def observedActionMessage (action party queryName : String) : String :=
  s!"the Case for Query '{queryName}' cannot perform '{action}': its party '{party}' is \
`observed`, so the world performs it and the Case only reads that it did; bind '{party}' `driven` \
or leave the Query out"

private def missingCoverMessage : String :=
  "an exploratory set names what it covers under `cover:`: rows, results or classMembers"

private def unknownCoverMessage (spelling : String) : String :=
  s!"unknown coverage goal '{spelling}'; declared: rows, results, classMembers"

private def coverOutsideMessage (purpose : String) : String :=
  s!"`cover:` names an exploratory set's goal; a {purpose} set lists Queries"

private def missingBudgetMessage : String :=
  "an exploratory set names the `limits` it explores within under `budget:`"

private def budgetOutsideMessage (purpose : String) : String :=
  s!"`budget:` bounds an exploratory set; a {purpose} set's Queries carry their own limits"

private def machineOutsideMessage (purpose : String) : String :=
  s!"`machine:` names the machine an exploratory set covers; a {purpose} set's Queries name theirs"

private def missingMachineMessage : String :=
  "an exploratory set names the machine it covers under `machine:`"

private def notAMachineMessage (spelling : String) : String :=
  s!"'{spelling}' is not a machine declared by a `machine` command"

private def budgetNotLimitsMessage (spelling : String) : String :=
  s!"'{spelling}' is not a `limits` declaration; `budget:` names the limits an exploratory set \
explores within"

private def duplicateSwitchMessage (name : String) : String :=
  s!"switch '{name}' is already registered"

/-- The action a Scenario's action spelling names: the key's first segment, which is the
constructor for a classed action and the whole key for a bare one. -/
private def actionNameOf (spelling : String) : String :=
  ((spelling.splitOn "-").head?).getD spelling

elab doc?:(docComment)? setKeyword name:ident keys:setKey+ : command => do
  let mut purpose : Option (Ident × String) := none
  let mut bindings : Array (Ident × String × Ident × String) := #[]
  let mut bindEntry : Option Syntax := none
  let mut repeat? : Option Ident := none
  let mut queryRefs : Option (Syntax × Array Ident) := none
  let mut machineRef : Option Ident := none
  let mut coverRefs : Option (Syntax × Array Ident) := none
  let mut budgetRef : Option Ident := none
  for entry in keys do
    match entry with
    | `(setKey| purpose: $purposeRef:ident) => do
        if purpose.isSome then throwErrorAt entry (duplicateKeyMessage "set" "purpose:")
        purpose := some (purposeRef, purposeRef.getId.eraseMacroScopes.toString)
    | `(setKey| bind: $[$parties:ident : $modes:ident]*) => do
        if bindEntry.isSome then throwErrorAt entry (duplicateKeyMessage "set" "bind:")
        bindEntry := some entry
        for party in parties, mode in modes do
          bindings := bindings.push (party, party.getId.eraseMacroScopes.toString, mode,
            mode.getId.eraseMacroScopes.toString)
    | `(setKey| repeat: $switchRef:ident) => do
        if repeat?.isSome then throwErrorAt entry (duplicateKeyMessage "set" "repeat:")
        repeat? := some switchRef
    | `(setKey| queries: [$listed:ident,*]) => do
        if queryRefs.isSome then throwErrorAt entry (duplicateKeyMessage "set" "queries:")
        queryRefs := some (entry, listed.getElems)
    | `(setKey| machine: $covered:ident) => do
        if machineRef.isSome then throwErrorAt entry (duplicateKeyMessage "set" "machine:")
        machineRef := some covered
    | `(setKey| cover: $goals|*) => do
        if coverRefs.isSome then throwErrorAt entry (duplicateKeyMessage "set" "cover:")
        coverRefs := some (entry, goals.getElems)
    | `(setKey| budget: $limitsRef:ident) => do
        if budgetRef.isSome then throwErrorAt entry (duplicateKeyMessage "set" "budget:")
        budgetRef := some limitsRef
    | _ => throwErrorAt entry "unsupported set key"
  let some (purposeRef, purposeSpelling) := purpose
    | throwErrorAt name (missingKeyMessage "set" "purpose:")
  let purposeTerm ← match purposeSpelling with
    | "functional" => `(term| Umpire.Command.SetPurpose.functional)
    | "canary" => `(term| Umpire.Command.SetPurpose.canary)
    | "exploratory" => `(term| Umpire.Command.SetPurpose.exploratory)
    | other => throwErrorAt purposeRef (unknownPurposeMessage other)
  -- Every party the Model's actions name is bound, `system` is not, and nothing else is. The
  -- actions are the ones the set's Queries' machines step on, or the ones the machine an
  -- exploratory set covers steps on, by the declarations the machines resolved; a set that names
  -- neither binds the parties of the actions declared beside it. Each Query and the machine are
  -- resolved to their constants once, here, so the party check and the checks below read the same
  -- declaration.
  let environment ← getEnv
  let currentNamespace ← getCurrNamespace
  let mut resolvedMachine : Option (Name × Registry.MachineEntry) := none
  if let some covered := machineRef then
    let machineName? ← try
        some <$> liftTermElabM (realizeGlobalConstNoOverloadWithInfo covered)
      catch failure =>
        if failure.isInterrupt || failure.isMaxRecDepth then throw failure else pure none
    let some declared := machineName?.bind (Registry.machine? environment)
      | throwErrorAt covered (notAMachineMessage covered.getId.toString)
    resolvedMachine := some (machineName?.getD .anonymous, declared)
  let mut resolvedQueries : Array (Syntax × Name × Registry.QueryEntry) := #[]
  let listedQueries : Array Ident := (queryRefs.map (·.2)).getD #[]
  for queryRef in listedQueries do
    let queryName? ← try
        some <$> liftTermElabM (realizeGlobalConstNoOverloadWithInfo queryRef)
      catch failure =>
        if failure.isInterrupt || failure.isMaxRecDepth then throw failure else pure none
    let some declared := queryName?.bind (Registry.query? environment)
      | throwErrorAt queryRef (undeclaredMessage "query" queryRef.getId)
    resolvedQueries := resolvedQueries.push (queryRef, queryName?.getD .anonymous, declared)
  let steppedOn : Array Name := (resolvedQueries.flatMap fun (_, _, declared) =>
    match Registry.scenario? environment declared.scenario with
    | some declaredScenario =>
        match Registry.machine? environment declaredScenario.model with
        | some declaredMachine => declaredMachine.actionDecls
        | none => #[]
    | none => #[]) ++ ((resolvedMachine.map (·.2.actionDecls)).getD #[])
  let declaredActions := (Registry.actions environment).filter fun declared =>
    if resolvedQueries.isEmpty && resolvedMachine.isNone then
      currentNamespace.isPrefixOf declared.declName
    else steppedOn.contains declared.declName
  let parties := (declaredActions.map (·.party)).toList.eraseDups
  let mut seenParties : Array String := #[]
  for (partyRef, party, modeRef, mode) in bindings do
    if party == "system" then throwErrorAt partyRef systemBoundMessage
    unless parties.contains party do throwErrorAt partyRef (unknownPartyMessage party)
    if seenParties.contains party then throwErrorAt partyRef (duplicateBindingMessage party)
    seenParties := seenParties.push party
    unless mode == "driven" || mode == "observed" do
      throwErrorAt modeRef (unknownBindingMessage mode)
  for party in parties do
    unless seenParties.contains party do
      throwErrorAt (bindEntry.getD name) (unboundPartyMessage party)
  let observedParties := bindings.filterMap fun (_, party, _, mode) =>
    if mode == "observed" then some party else none
  -- Which keys a purpose takes: Queries for functional and canary, a goal and a budget for
  -- exploratory.
  let exploratory := purposeSpelling == "exploratory"
  if exploratory then
    if let some (entry, _) := queryRefs then throwErrorAt entry queriesOutsideMessage
    if coverRefs.isNone then throwErrorAt name missingCoverMessage
    if budgetRef.isNone then throwErrorAt name missingBudgetMessage
    if machineRef.isNone then throwErrorAt name missingMachineMessage
  else
    if let some (entry, _) := coverRefs then throwErrorAt entry (coverOutsideMessage purposeSpelling)
    if let some limitsRef := budgetRef then throwErrorAt limitsRef (budgetOutsideMessage purposeSpelling)
    if let some covered := machineRef then throwErrorAt covered (machineOutsideMessage purposeSpelling)
    if queryRefs.isNone then throwErrorAt name (missingQueriesMessage purposeSpelling)
  -- A switch is the realization's, registered by name: `repeat:` names one of them.
  let repeatTerm ← match repeat? with
    | none => `(term| none)
    | some switchRef => do
        unless purposeSpelling == "functional" do
          throwErrorAt switchRef (repeatOutsideFunctionalMessage purposeSpelling)
        let spelling := switchRef.getId.eraseMacroScopes.toString
        unless (Registry.switch? environment spelling).isSome do
          throwErrorAt switchRef (unknownSwitchMessage spelling
            ((Registry.switches environment).map (·.name)).toList)
        `(term| some $(Lean.quote spelling))
  -- Each Query finds rather than verifies, and in a functional set its path performs no action of
  -- an `observed` party, because the Case would have to perform it.
  let origin ← originTerm
  let mut queryNames : Array Name := #[]
  let mut queryIdTerms : Array Term := #[]
  for (queryRef, queryName, declared) in resolvedQueries do
    unless declared.selectsWitness do
      throwErrorAt queryRef (verifyQueryInSetMessage queryName.toString purposeSpelling)
    if purposeSpelling == "functional" then
      let selected := ((Registry.scenario? environment declared.scenario).map (·.actions)).getD #[]
      for spelling in selected do
        if let some performer := declaredActions.find? (·.name == actionNameOf spelling) then
          if observedParties.contains performer.party then
            throwErrorAt queryRef
              (observedActionMessage spelling performer.party queryName.toString)
    queryNames := queryNames.push queryName
    queryIdTerms := queryIdTerms.push
      (← `(term| ($origin).family.id "query" $(Lean.quote queryName.getString!)))
  let coverTerms ← ((coverRefs.map (·.2)).getD #[]).mapM fun goalRef => do
    match goalRef.getId.eraseMacroScopes.toString with
    | "rows" => `(term| Umpire.Command.CoverageGoal.rows)
    | "results" => `(term| Umpire.Command.CoverageGoal.results)
    | "classMembers" => `(term| Umpire.Command.CoverageGoal.classMembers)
    | other => throwErrorAt goalRef (unknownCoverMessage other)
  -- The budget is a `limits` declaration, which is what an exploration is bounded by.
  let budgetTerm ← match budgetRef with
    | none => `(term| none)
    | some limitsRef => do
        let limitsName ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo limitsRef)
        unless (← getConstInfo limitsName).type.isConstOf ``Umpire.Limits do
          throwErrorAt limitsRef (budgetNotLimitsMessage limitsRef.getId.toString)
        `(term| some $(Lean.quote limitsRef.getId.eraseMacroScopes.toString))
  -- What an exploratory set enumerates: the targets its goals name on the machine it covers,
  -- under its budget, with the claims the machine's actions make for the class members.
  let (machineTerm, targetsTerm) ← match resolvedMachine, budgetRef with
    | some (machineName, declared), some limitsRef => do
        let actionRefs := declared.actionDecls.map mkIdent
        pure (← `(term| some (Umpire.DefinitionId.of $(Lean.quote declared.id))),
          ← `(term| Umpire.Command.coverageTargets ($(mkIdent machineName))
            (Umpire.Command.classClaims ($(mkIdent machineName)) [$actionRefs,*])
            [$coverTerms,*] ($limitsRef)))
    | _, _ => do pure (← `(term| none), ← `(term| []))
  let bindingTerms ← bindings.mapM fun (_, party, _, mode) => do
    let modeTerm ← if mode == "driven" then `(term| Umpire.Command.PartyBinding.driven)
      else `(term| Umpire.Command.PartyBinding.observed)
    `(term| ($(Lean.quote party), $modeTerm))
  let nameKey := Lean.quote name.getId.toString
  let declaration ← `(term|
    { id := ($origin).family.id "set" $nameKey
      name := $nameKey
      purpose := $purposeTerm
      bindings := [$bindingTerms,*]
      «repeat» := $repeatTerm
      queries := [$queryIdTerms,*]
      machine := $machineTerm
      cover := [$coverTerms,*]
      budget := $budgetTerm
      targets := $targetsTerm
      source := ($origin).source })
  elabCommand (← `(command|
    $[$doc?:docComment]? def $name : Umpire.Command.SetDeclaration := $declaration))
  liftCoreM (Registry.recordSet {
    declName := (← getCurrNamespace) ++ name.getId
    name := name.getId.toString
    purpose := purposeSpelling
    queries := queryNames
    «repeat» := repeat?.map fun switchRef => switchRef.getId.eraseMacroScopes.toString })

/-! ### Registering a switch

A switch is declared by a realization, which is the owning platform's, so the Umpire set command
learns of it by registration: the platform's module says once which switches exist, and a `repeat:`
resolves against them. -/

private unsafe def evalSwitchUnsafe (declName : Name) :
    Elab.Term.TermElabM Umpire.Case.Producer.SwitchBinding :=
  Meta.evalExpr Umpire.Case.Producer.SwitchBinding
    (.const ``Umpire.Case.Producer.SwitchBinding []) (.const declName [])

@[implemented_by evalSwitchUnsafe]
private opaque evalSwitch (declName : Name) : Elab.Term.TermElabM Umpire.Case.Producer.SwitchBinding

elab "register_switch" switchRef:ident : command => do
  let declName ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo switchRef)
  let declared ← liftTermElabM (evalSwitch declName)
  if (Registry.switch? (← getEnv) declared.name).isSome then
    throwErrorAt switchRef (duplicateSwitchMessage declared.name)
  liftCoreM (Registry.recordSwitch {
    name := declared.name
    values := (declared.values.map (·.name)).toArray })

end Umpire.Command
