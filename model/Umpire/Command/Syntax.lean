import Lean.Elab.Command
import Lean.Elab.ElabRules
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

/-! ### Declaring a domain

`enum` is the four vocabulary declarations a Model file makes, without the `deriving` clause the
`model` command requires and the author has no reason to think about. It resolves nothing and
reorders nothing: the constructors in declaration order are the ordered domain, which is what AUT-09
means by author-provided. A plain `inductive` is still admitted; `enum` is shorthand for exactly the
one it would have written. -/

-- A member's own doc comment goes after its bar, not before it. Before the bar it would be
-- indistinguishable from the doc comment of whatever declaration follows the `enum`, and the
-- repetition would swallow it.
macro doc?:(docComment)? &"enum" name:ident
    constructors:("|" (docComment)? ident)+ : command => do
  let declared ← constructors.mapM fun constructor => do
    let parts := constructor.raw
    let constructorDoc : Option (TSyntax ``Lean.Parser.Command.docComment) :=
      if parts[1].isNone then none else some (TSyntax.mk parts[1][0])
    let constructorName : Ident := TSyntax.mk parts[2]
    `(Lean.Parser.Command.ctor| $[$constructorDoc:docComment]? | $constructorName:ident)
  `(command| $[$doc?:docComment]? inductive $name where
      $declared:ctor*
      deriving BEq, DecidableEq, Repr)

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
ceiling on how large a table the elaborator will build, not a modelling recommendation. -/
private def transitionBound : Nat := 256

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

/-- The Known Gaps this Query declared, as a checked set. A Query that declares none carries none:
nothing is attached on its behalf. -/
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
  `(term| Umpire.KnownGapSet.checkCanonical [$terms,*])

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

end Umpire.Command
