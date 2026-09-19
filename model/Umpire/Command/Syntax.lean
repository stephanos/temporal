import Lean.Elab.Command
import Lean.Elab.ElabRules
import Umpire.Command.Finite
import Umpire.Command.Records
import Umpire.Command.Schema
import Umpire.Command.Registry

/-!
# The Model command grammar

Five commands -- `model`, `property`, `scenario`, `limits`, `query` -- and their expansion into
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

/-- One `before + action → after` Step row. Its relation key is derived from the row itself. -/
declare_syntax_cat modelStep

syntax ident "+" ident "→" ident "," "outcome:" ident : modelStep
syntax ident "+" ident "→" ident "," "outcome:" ident "," "facts:" "[" ident,* "]" : modelStep

/-- One `require:` line of a declared Property. Its clause key is derived from the line itself. -/
declare_syntax_cat modelRequirement

syntax "state:" ident : modelRequirement
syntax "outcome:" ident : modelRequirement
syntax "fact:" ident : modelRequirement

/-- One Known Gap a Query carries: what kind of thing is missing, the name its code derives from,
optionally the Property it limits, and why. -/
declare_syntax_cat modelGap

syntax "gap:" ident "code:" str "subject:" str "detail:" str : modelGap
syntax "gap:" ident "code:" str "detail:" str : modelGap

/-- The elaboration bound on declared transition rows. The tested scale is far smaller; this is a
ceiling on how large a table the elaborator will build, not a modelling recommendation.

It is `Umpire.Command.elaborationBound`, the same number a step function's enumeration is bounded by:
both say "this Model is too big to elaborate", so they are one decision with one owner. -/
private def transitionBound : Nat := Umpire.Command.elaborationBound

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

private def parameterizedConstructorMessage (domain spelling : String) : String :=
  s!"Model {domain} '{spelling}' takes arguments; a {domain} domain must be an enum-like inductive"

private def duplicateTransitionMessage (source selected : String) : String :=
  s!"duplicate Model step: '{source} + {selected}' is already declared"

/-- A Step row's relation key. It is the row's own coordinates, because those are what make it
unique -- the command rejects a second row leaving the same state on the same Action. -/
private def relationKey (sourceState selectedAction : Ident) : String :=
  (shortName sourceState.getId).toString ++ "-" ++ (shortName selectedAction.getId).toString

private def unreachableTerminalMessage (spelling : String) : String :=
  s!"Model end state '{spelling}' is unreachable from every start state"

private def unsortedActionsMessage (earlier later : String) : String :=
  "Model action constructors must be declared in sorted order, because the planner admits " ++
    s!"only a canonically ordered Action catalog; '{later}' precedes '{earlier}'"

private def unsortedInitialMessage (earlier later : String) : String :=
  "Model start states must be declared in sorted order, because the planner admits " ++
    s!"only a canonically ordered start-state list; '{later}' precedes '{earlier}'"

private def transitionBoundMessage (declared : Nat) : String :=
  s!"the Model declares {declared} steps; the elaboration bound is {transitionBound}"

/-- The ordered constructors of a named enum-like inductive. A constructor that takes arguments is
not an enum-like member, so the domain is rejected at the type the model names. -/
private def domainConstructors (domain : String) (typeRef : Ident) : CommandElabM (List Name) := do
  let name ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo typeRef)
  let info ← getConstInfoInduct name
  for constructor in info.ctors do
    let declaration ← getConstInfoCtor constructor
    if declaration.numFields != 0 then
      throwErrorAt typeRef
        (parameterizedConstructorMessage domain (shortName constructor).toString)
  pure info.ctors

/-- Resolve one authored spelling against a declared domain, reporting an unknown one in place. -/
private def resolveMember (domain : String) (constructors : List Name) (member : Ident) :
    CommandElabM Ident := do
  let spelling := member.getId.eraseMacroScopes
  match constructors.find? fun constructor => shortName constructor == spelling with
  | some constructor => pure (mkIdentFrom member constructor)
  | none => throwErrorAt member (unknownMemberMessage domain spelling.toString constructors)

/-- One transition row with every member resolved once, before the table is built from it. -/
private structure ResolvedRow where
  key : TSyntax `modelStep
  sourceState : Ident
  selectedAction : Ident
  targetState : Ident
  rowTerm : Term

/-- The states reachable from `seen` over the declared `before → result` edges. -/
private def reachableStates (edges : List (Name × Name)) : Nat → List Name → List Name
  | 0, seen => seen
  | fuel + 1, seen =>
      let next := (edges.filterMap fun edge =>
        if seen.contains edge.1 && !seen.contains edge.2 then some edge.2 else none).eraseDups
      if next.isEmpty then seen else reachableStates edges fuel (seen ++ next)

private def memberKeys (constructors : List Name) : Array Term :=
  constructors.toArray.map fun constructor => Lean.quote (shortName constructor).toString

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
`model` command requires and the author has no reason to think about. It resolves nothing and
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

private def memberIdents (constructors : List Name) : Array Term :=
  constructors.toArray.map fun constructor => mkIdent constructor

/-- The `model` command's whole body. Two command spellings share it -- one that declares a Fact
domain and one that declares none -- because an optional group in a command signature does not
bind. -/
private def elabModel
    (name role stateType actionType outcomeType : Ident)
    (factDomain : Option Ident)
    (initialRefs terminalRefs : Array Ident)
    (rows : Array (TSyntax `modelStep)) : CommandElabM Unit := do
  let stateCtors ← domainConstructors "state" stateType
  let actionCtors ← domainConstructors "action" actionType
  let outcomeCtors ← domainConstructors "outcome" outcomeType
  -- A Model that declares no Fact domain gets the empty one: no row can name a Fact, and no
  -- `require ...: fact ...` clause can either, because there is nothing to name.
  let factType : Ident := match factDomain with
    | some declared => declared
    | none => mkIdentFrom name ``NoFact
  let factCtors ← match factDomain with
    | some _ => domainConstructors "fact" factType
    | none => pure []
  let actionSpellings := actionCtors.map fun constructor => (shortName constructor).toString
  for pair in actionSpellings.zip actionSpellings.tail do
    unless pair.1 < pair.2 do
      throwErrorAt actionType (unsortedActionsMessage pair.2 pair.1)
  let initialStates ← initialRefs.toList.mapM (resolveMember "state" stateCtors)
  let terminalStates ← terminalRefs.toList.mapM (resolveMember "state" stateCtors)
  let setupConstructorName : Name := match initialStates.head? with
    | some first => shortName first.getId
    | none => `setup
  let initialPairs := initialStates.zip initialRefs.toList
  for pair in initialPairs.zip initialPairs.tail do
    let earlier := (shortName pair.1.1.getId).toString
    let later := (shortName pair.2.1.getId).toString
    unless earlier < later do
      throwErrorAt pair.2.2 (unsortedInitialMessage later earlier)
  if rows.size > transitionBound then
    throwErrorAt rows[transitionBound]! (transitionBoundMessage rows.size)
  let resolvedRows ← rows.toList.mapM fun (row : TSyntax `modelStep) => do
    let resolve := fun (source selected resulting outcomeRef : Ident)
        (observed : List Ident) => do
      let sourceState ← resolveMember "state" stateCtors source
      let selectedAction ← resolveMember "action" actionCtors selected
      let targetState ← resolveMember "state" stateCtors resulting
      let resolvedOutcome ← resolveMember "outcome" outcomeCtors outcomeRef
      let observedFacts ← observed.mapM (resolveMember "fact" factCtors)
      -- The relation key is the row: which state it leaves and which Action it takes. The command
      -- already rejects two rows with that pair, so the key is unique without an author label.
      let keyLiteral := Lean.quote (relationKey sourceState selectedAction)
      let rowTerm ← `(term|
        { key := $keyLiteral
          source := $sourceState
          action := $selectedAction
          results := [step $resolvedOutcome $targetState
            [$(observedFacts.toArray),*]] })
      pure ({ key := row, sourceState, selectedAction, targetState, rowTerm : ResolvedRow })
    match row with
    | `(modelStep| $source:ident + $selected:ident → $resulting:ident ,
        outcome: $outcomeRef:ident) =>
        resolve source selected resulting outcomeRef []
    | `(modelStep| $source:ident + $selected:ident → $resulting:ident ,
        outcome: $outcomeRef:ident , facts: [$observed,*]) =>
        resolve source selected resulting outcomeRef observed.getElems.toList
    | _ => throwErrorAt row "unsupported Model step"
  let mut declared : List ResolvedRow := []
  for resolved in resolvedRows do
    if declared.any fun candidate =>
        candidate.sourceState.getId == resolved.sourceState.getId &&
          candidate.selectedAction.getId == resolved.selectedAction.getId then
      throwErrorAt resolved.key
        (duplicateTransitionMessage
          (shortName resolved.sourceState.getId).toString
          (shortName resolved.selectedAction.getId).toString)
    declared := declared ++ [resolved]
  let edges := resolvedRows.map fun resolved =>
    (resolved.sourceState.getId, resolved.targetState.getId)
  let reached := reachableStates edges (edges.length + 1)
    (initialStates.map fun entry => entry.getId)
  for terminalState in terminalStates do
    unless reached.contains terminalState.getId do
      throwErrorAt terminalState
        (unreachableTerminalMessage (shortName terminalState.getId).toString)
  let transitionTerms := resolvedRows.map fun resolved => resolved.rowTerm
  let declarationKey := Lean.quote name.getId.toString
  let roleKey := Lean.quote role.getId.toString
  let setupKey := Lean.quote setupConstructorName.toString
  let names ← `(term|
    { declaration := $declarationKey
      roleName := $roleKey
      setup := $setupKey
      stateKeys := [$(memberKeys stateCtors),*]
      actionKeys := [$(memberKeys actionCtors),*]
      outcomeKeys := [$(memberKeys outcomeCtors),*]
      factKeys := [$(memberKeys factCtors),*] })
  let origin ← originTerm
  -- The setup domain is the command's, not the author's: nothing else in a Model file mentions it,
  -- and a file that had to declare one would be declaring scaffolding. It is scoped under the
  -- Model's own name, so two Models in one namespace never collide, and its single constructor is
  -- named after the Model's first start state, which is what keeps the canonical setup key -- and
  -- every fingerprint built on it -- the value the author's own declaration produced.
  let setupType := mkIdentFrom name (name.getId ++ `Setup)
  let setupName := mkIdentFrom name (name.getId ++ `Setup ++ setupConstructorName)
  elabCommand (← `(command|
    inductive $setupType where
      | $(mkIdent setupConstructorName):ident
      deriving BEq, DecidableEq, Repr))
  let spellingsOf := fun (constructors : List Name) =>
    (constructors.map fun constructor => (shortName constructor).toString).toArray
  liftCoreM (Registry.recordModel {
    declName := (← getCurrNamespace) ++ name.getId
    role := role.getId.eraseMacroScopes.toString
    stateType := ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo stateType)
    actionType := ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo actionType)
    outcomeType := ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo outcomeType)
    factType := ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo factType)
    «states» := spellingsOf stateCtors
    «actions» := spellingsOf actionCtors
    «outcomes» := spellingsOf outcomeCtors
    «facts» := spellingsOf factCtors
    «starts» := (initialStates.map fun declared =>
      (shortName declared.getId).toString).toArray })
  elabCommand (← `(command|
    def $name := declareModel $origin $names ($setupName)
      ([$(memberIdents stateCtors),*]) ([$(memberIdents actionCtors),*])
      ([$(memberIdents outcomeCtors),*]) (([$(memberIdents factCtors),*] : List $factType))
      ([$(initialStates.toArray),*]) ([$(terminalStates.toArray),*])
      ([$(transitionTerms.toArray),*])
      (by exact ⟨rfl, rfl, rfl⟩)))

/-! ### The `model` command

Two spellings, one body: a Model that declares a Fact domain, and one that declares none. -/

elab "model" name:ident
    "role:" roleRef:ident
    "states:" stateType:ident
    "actions:" actionType:ident
    "outcomes:" outcomeType:ident
    "facts:" factType:ident
    "starts:" "[" initialRefs:ident,+ "]"
    "ends:" "[" terminalRefs:ident,+ "]"
    "steps:" rows:modelStep+ : command =>
  elabModel name roleRef stateType actionType outcomeType (some factType)
    initialRefs.getElems terminalRefs.getElems rows

elab "model" name:ident
    "role:" roleRef:ident
    "states:" stateType:ident
    "actions:" actionType:ident
    "outcomes:" outcomeType:ident
    "starts:" "[" initialRefs:ident,+ "]"
    "ends:" "[" terminalRefs:ident,+ "]"
    "steps:" rows:modelStep+ : command =>
  elabModel name roleRef stateType actionType outcomeType none
    initialRefs.getElems terminalRefs.getElems rows

/-! ### The `property` command

A Property names its Model and the Action every clause is about, then one `require:` line per
requirement. The clause key is the requirement -- its kind and the member it names -- so a duplicate
requirement is a duplicate key, and rejects on the line that repeats it. -/

private def duplicateRequirementMessage (key : String) : String :=
  s!"duplicate requirement '{key}': this Property already requires it"

private def undeclaredModelMessage (spelling : Name) : String :=
  s!"'{spelling}' is not a Model declared by a `model` command"

/-- Resolve one spelling against a Model's declared domain, reporting an unknown one in place. The
message shape is the `model` command's, so an author sees one vocabulary wherever they are. -/
private def resolveDeclared (domain : String) (declared : Array String) (declaringType : Name)
    (member : Ident) : CommandElabM String := do
  let spelling := member.getId.eraseMacroScopes.toString
  unless declared.contains spelling do
    throwErrorAt member (unknownMemberMessage domain spelling
      (declared.toList.map Name.mkSimple))
  -- Give the spelling the constructor it names, so the editor hovers it and goes to its
  -- definition. The emitted records keep the spelling, so no Definition ID moves.
  liftTermElabM (Lean.Elab.addConstInfo member (declaringType ++ Name.mkSimple spelling))
  pure spelling

/-- The Model a `model:` key names, resolved to what the `model` command recorded about it. -/
private def resolveDeclaredModel (modelRef : Ident) : CommandElabM Registry.ModelEntry := do
  let modelName ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo modelRef)
  match Registry.model? (← getEnv) modelName with
  | some declared => pure declared
  | none => throwErrorAt modelRef (undeclaredModelMessage modelName)

elab "property" name:ident
    "model:" modelRef:ident
    "when:" actionRef:ident
    "require:" requirements:modelRequirement+ : command => do
    let ownerKey := Lean.quote name.getId.toString
    let actionKey := Lean.quote actionRef.getId.toString
    let declaredModel ← resolveDeclaredModel modelRef
    let roleKey := Lean.quote declaredModel.role
    let _ ← resolveDeclared "action" declaredModel.actions declaredModel.actionType actionRef
    let mut keys : Array String := #[]
    let mut clauses : Array Term := #[]
    for requirement in requirements do
      -- Every member a requirement names is resolved here, against the Model's own domain, so a
      -- misspelling is a located error while the Model file compiles rather than a `#guard` failure
      -- in some other module.
      let (kind, member) ← match requirement with
        | `(modelRequirement| state: $member:ident) => do
            let _ ← resolveDeclared "state" declaredModel.states declaredModel.stateType member
            pure ("state", member)
        | `(modelRequirement| outcome: $member:ident) => do
            let _ ← resolveDeclared "outcome" declaredModel.outcomes declaredModel.outcomeType member
            pure ("outcome", member)
        | `(modelRequirement| fact: $member:ident) => do
            let _ ← resolveDeclared "fact" declaredModel.facts declaredModel.factType member
            pure ("fact", member)
        | _ => throwErrorAt requirement "unsupported requirement"
      let spelling := member.getId.eraseMacroScopes.toString
      let key := kind ++ "-" ++ spelling
      if keys.contains key then
        throwErrorAt requirement (duplicateRequirementMessage key)
      keys := keys.push key
      let constructor := match kind with
        | "state" => `stateClause
        | "outcome" => `outcomeClause
        | _ => `factClause
      clauses := clauses.push (← `(term|
        $(mkIdent (`Umpire.Command.PropertyRequirement ++ constructor))
          $(Lean.quote key) $(Lean.quote spelling)))
    elabCommand (← `(command|
      def $name (values : ModelVocabulary) : Property :=
        authoredProperty ($modelRef) values {
          declaration := $ownerKey
          roleName := $roleKey
          actionSpelling := $actionKey
          requirements := [$clauses,*]
        }))
    liftCoreM (Registry.recordProperty {
      declName := (← getCurrNamespace) ++ name.getId, «model» := declaredModel.declName })

/-! ### The `scenario` command

`actions:` is the exact sequence the operation selects. Each occurrence's key is its position in
that sequence, because that is what distinguishes two occurrences of the same Action. -/

elab "scenario" name:ident
    "model:" modelRef:ident
    "starts:" setupRef:ident
    "actions:" "[" selected:ident,+ "]" : command => do
    let ownerKey := Lean.quote name.getId.toString
    let setupKey := Lean.quote setupRef.getId.toString
    let declaredModel ← resolveDeclaredModel modelRef
    let roleKey := Lean.quote declaredModel.role
    -- The setup state must be one the Model can start in, not merely one it declares.
    let _ ← resolveDeclared "start state" declaredModel.starts declaredModel.stateType setupRef
    let spellings ← selected.getElems.mapM
      (resolveDeclared "action" declaredModel.actions declaredModel.actionType)
    let entries ← spellings.mapIdxM fun position spelling =>
      `(term| ($(Lean.quote (toString (position + 1))), $(Lean.quote spelling)))
    elabCommand (← `(command|
      def $name (values : ModelVocabulary) : Scenario :=
        authoredScenario ($modelRef) values {
          declaration := $ownerKey
          roleName := $roleKey
          setupState := $setupKey
          occurrences := [$entries,*]
        }))
    liftCoreM (Registry.recordScenario {
      declName := (← getCurrNamespace) ++ name.getId
      «model» := declaredModel.declName
      «actions» := spellings })

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

/-- The Model a Query runs on, resolved from its Property and its Scenario rather than named again.
-/
private def queryModelName (propertyRef scenarioRef : Ident) : CommandElabM Name := do
  let propertyName ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo propertyRef)
  let scenarioName ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo scenarioRef)
  let environment ← getEnv
  let declaredProperty ← match Registry.property? environment propertyName with
    | some declared => pure declared
    | none => throwErrorAt propertyRef (undeclaredMessage "property" propertyName)
  let declaredScenario ← match Registry.scenario? environment scenarioName with
    | some declared => pure declared
    | none => throwErrorAt scenarioRef (undeclaredMessage "scenario" scenarioName)
  unless declaredProperty.model == declaredScenario.model do
    throwErrorAt scenarioRef
      (mismatchedModelMessage declaredProperty.model declaredScenario.model)
  pure declaredProperty.model

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
    let modelRef := mkIdent (← queryModelName propertyRef scenarioRef)
    let queryKey := Lean.quote name.getId.toString
    let knownGaps ← knownGapsTerm (← originTerm) gaps
    elabCommand (← `(command|
      def $name : Except AdmissionError (CheckedModel ($modelRef)) :=
        check ($modelRef) $queryKey ($limitsRef) ($propertyRef) ($scenarioRef)
          (knownGaps := $knownGaps)))
    recordQueryDeclaration name scenarioRef (selectsWitness := true)
    elabQueryAdmission name propertyRef scenarioRef limitsRef modelRef findKeyword name

elab "query" name:ident
    verifyKeyword:"verify:" propertyRef:ident
    "in:" scenarioRef:ident
    "limits:" limitsRef:ident gaps:modelGap* : command => do
    let modelRef := mkIdent (← queryModelName propertyRef scenarioRef)
    let queryKey := Lean.quote name.getId.toString
    let knownGaps ← knownGapsTerm (← originTerm) gaps
    elabCommand (← `(command|
      def $name : Except AdmissionError (CheckedModel ($modelRef)) :=
        check ($modelRef) $queryKey ($limitsRef) ($propertyRef) ($scenarioRef)
          (knownGaps := $knownGaps) (form := QueryFormKind.verifyClaim)))
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
`FiniteTransitionRow.action`. Reserving them as tokens, the way `model` and `property` are reserved,
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
              match (← liftTermElabM (Meta.evalNat bound).run) with
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
syntax "setup:" withPosition((colGe ident ":" ident)+) : machineKey
syntax "evidence:" withPosition((colGe ident ":" ident)+) : machineKey
syntax "steps:" withPosition((colGe ident ":" ident)+) : machineKey

@[run_parser_attribute_hooks] private def machineKeyword := declarationKeyword "machine"

private def undeclaredMachineEntityMessage (spelling : Name) : String :=
  s!"'{spelling}' is not an entity declared by an `entity` command; a machine tracks one entity's \
instances, so `for:` names one"

private def undeclaredStepActionMessage (spelling : Name) : String :=
  s!"'{spelling}' is not an action declared by an `action` command; a `steps:` line names the action \
its function steps on"

private def unprovenTableMessage (states actions : Nat) : String :=
  s!"this machine's canonical-table law did not check: {states} states over {actions} action \
classes is past what the proof elaborates, and a Model declared anyway would carry `sorryAx` while \
reading as complete. Reduce the state structure, or bound it -- every value derived from the table \
rests on this law"

private def machineTooLargeMessage (states actions bound : Nat) : String :=
  s!"enumerating {states} states over {actions} action classes is {states * actions} steps, and the \
bound is {bound}; a machine this size is bounded by its Limits or by symmetry, not walked"

private def stuckStateMessage (witness : String) : String :=
  s!"the machine reaches '{witness}', does not end there, and can take no step from it; either a \
step is missing or '{witness}' belongs under `ends:`"

private def unnamedTimerMessage (spelling : String) : String :=
  s!"no `steps:` line names the timer '{spelling}'; a timer is `system` behaviour written as a step \
function, and one that never fires is a timer the machine does not have"

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


/-- Elaborate one of a machine's generated declarations.

A machine's definitions are as long as its state space: `DESIGN.md` section 3's protocol machine has
224 states, so its key array, its terminal list and its start list are list literals of that length,
and a literal that long nests deeper than a Lean file's default recursion limit. The limit is raised
on the generated declaration and nowhere else, because the length is the machine's size rather than
anything an author wrote, and an author who hit the file's own limit should still hear about it. -/
private def elabGenerated (generated : TSyntax `command) : CommandElabM Unit := do
  elabCommand (← `(command| set_option maxRecDepth 65536 in $generated:command))

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
      -- Each argument after the state is the action's own input domain, in order. Two actions of
      -- the same arity over different enums are an ordinary slip, and without this the mismatch
      -- surfaces inside the synthesized dispatcher rather than at the line that named the function.
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
  let mut setupParameters : Array (String × Name) := #[]
  let mut evidenceRefs : Array (Ident × Ident) := #[]
  let mut stepRefs : Array (Ident × Ident) := #[]
  for entry in keys do
    match entry with
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
    | `(machineKey| setup: $[$parameter:ident : $domain:ident]*) => do
        if !setupParameters.isEmpty then throwErrorAt entry (duplicateKeyMessage "machine" "setup:")
        for named in parameter, ranged in domain do
          let (declName, _, _) ← resolveDomain ranged
          setupParameters := setupParameters.push (named.getId.toString, declName)
    -- `recorded`, not `fact`: `fact:` is already a token of the `model` command's `require:` block,
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
  -- The state's members are the machine's states. A structure is an inductive of one constructor, so
  -- the same walk that writes an action's classes writes them, and the same refusals apply.
  let stateDecl ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo stateType)
  unless isStructure (← getEnv) stateDecl do
    throwErrorAt stateType (notAStateStructureMessage stateType.getId)
  let stateMembers ← domainMembers stateType stateDecl
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
  let actionMembers ← domainMembers name actionDecl
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
  let stateKeyForName := mkIdentFrom name (name.getId ++ `stateKeyFor)
  elabGenerated (← `(command|
    def $stateKeyForName (state : $stateType) : String :=
      match (Umpire.Command.members (α := $stateType)).idxOf? state with
      | some at? => ($stateKeysName)[at?]!
      | none => ""))
  elabGenerated (← `(command|
    def $rowKeyName (state : $stateType) (taken : $actionType) : String :=
      match (Umpire.Command.members (α := $actionType)).idxOf? taken with
      | some actionAt => $stateKeyForName state ++ "-" ++ ($actionKeysName)[actionAt]!
      | none => ""))
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
      Umpire.Command.enumerate $rowKeyName $stepName))
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
    startTerms := startTerms.push (← member.term)
  -- An evidence line names a fact the steps return. One that names a fact no step returns confirms
  -- something that never happens, which is a mistake about the machine and not about the evidence.
  let factNames := factMembersEarly.map fun member => member.key
  for (recordedRef, _) in evidenceRefs do
    let spelling := recordedRef.getId.getString!
    unless factNames.contains spelling do
      throwErrorAt recordedRef
        (unreturnedEvidenceMessage spelling (", ".intercalate factNames))
  let terminalName := mkIdentFrom name (name.getId ++ `ends)
  let startsName := mkIdentFrom name (name.getId ++ `starts)
  elabGenerated (← `(command|
    def $terminalName : List $stateType := [$terminalTerms,*]))
  elabGenerated (← `(command|
    def $startsName : List $stateType := [$startTerms,*]))
  -- The declared Model, on the enumerated rows. Everything downstream -- the Behavior Fingerprint,
  -- Search, Contract lowering, `umpire-inspect` -- reads this and never sees a step function.
  let setupType := mkIdentFrom name (name.getId ++ `Setup)
  let setupName := mkIdentFrom name (name.getId ++ `Setup ++ `only)
  -- The constructor is built rather than written: an identifier inside a quotation is hygienic, so a
  -- literal `| only` would be declared under a macro scope and no name outside this command could
  -- reach it.
  elabGenerated (← `(command|
    inductive $setupType where
      | $(mkIdent `only):ident
      deriving BEq, DecidableEq, Repr))
  let outcomeMembers ← domainMembers name outcomeType
  let factMembers ← domainMembers name factType
  let keyList : List ClassValue → Array Term := fun values =>
    (values.map fun value => Lean.quote value.key).toArray
  let names ← `(term|
    { declaration := $(Lean.quote name.getId.toString)
      roleName := $(Lean.quote declaredEntity.name)
      setup := "only"
      stateKeys := [$(keyList stateMembers),*]
      actionKeys := [$(keyList actionMembers),*]
      outcomeKeys := [$(keyList outcomeMembers),*]
      factKeys := [$(keyList factMembers),*] })
  let origin ← originTerm
  elabGenerated (← `(command|
    def $name := Umpire.Command.declareModel $origin $names ($setupName)
      (Umpire.Command.members (α := $stateType))
      (Umpire.Command.members (α := $actionType))
      (Umpire.Command.members (α := $(mkIdent outcomeType)))
      (Umpire.Command.members (α := $(mkIdent factType)))
      ($startsName) ($terminalName) ($transitionsName)
      (by exact ⟨rfl, rfl, rfl⟩)))
  -- `elabCommand` logs a failure rather than throwing it, so a table whose canonical-table law did
  -- not check would be declared anyway, carrying `sorryAx` and looking complete. Everything reads
  -- that law -- the Behavior Fingerprint, Search, Contract lowering -- and only `#print axioms`
  -- would say otherwise, so the command says it here instead.
  let declared := (← getCurrNamespace) ++ name.getId
  if (← getEnv).contains declared then
    let axioms ← liftCoreM (Lean.collectAxioms declared)
    if axioms.contains ``sorryAx then
      throwErrorAt name (unprovenTableMessage stateMembers.length actionMembers.length)
  else
    throwErrorAt name (unprovenTableMessage stateMembers.length actionMembers.length)
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
  liftCoreM (Registry.recordMachine {
    declName := (← getCurrNamespace) ++ name.getId
    name := name.getId.toString
    id := ← definitionIdHere "machine" name.getId.toString
    entity := declaredEntity.declName
    stateType := stateDecl
    steps := steps.map fun resolved => (resolved.action.name, resolved.function.getId)
    timers := timerNames
    evidence := evidenceRefs.map fun (recordedRef, observedRef) =>
      (recordedRef.getId.getString!, observedRef.getId.getString!) })


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

end Umpire.Command
