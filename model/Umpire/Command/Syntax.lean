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

/-- One `before + action → result` row of a declared model. -/
declare_syntax_cat successStep

syntax ident ":" ident "+" ident "→"
  "{" "state" ":=" ident "," "outcome" ":=" ident "," "facts" ":=" "[" ident,* "]" "}" :
  successStep

/-- One `require` clause of a declared Property. -/
declare_syntax_cat successRequire

syntax "require" ident ":" &"state" ident : successRequire
syntax "require" ident ":" "outcome" ident : successRequire
syntax "require" ident ":" "fact" ident : successRequire

/-- The retired `resultingState` spelling still parses, so the macro can reject it in place and
name its replacement instead of failing as an unexplained parse error. -/
syntax "require" ident ":" "resultingState" ident : successRequire

/-- One Known Gap a Query carries: what kind of thing is missing, the name its code derives from,
optionally the Property it limits, and why. -/
declare_syntax_cat modelGap

syntax "gap" ident str &"subject" str &"detail" str : modelGap
syntax "gap" ident str &"detail" str : modelGap

/-- One labelled occurrence of a declared Action in a Behavior sequence. -/
declare_syntax_cat successOccurrence

syntax ident ":" ident : successOccurrence

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
  s!"unknown Model {domain} '{spelling}'; declared: {spellings constructors}"

private def parameterizedConstructorMessage (domain spelling : String) : String :=
  s!"Model {domain} '{spelling}' takes arguments; a {domain} domain must be an enum-like inductive"

private def duplicateTransitionMessage (key priorKey source selected : String) : String :=
  s!"duplicate Model step '{key}': '{source} + {selected}' is already declared by " ++
    s!"'{priorKey}'"

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
  key : Ident
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

/-- The keyword one alternation position actually matched. A retired spelling parses alongside its
replacement so the elaborator can point at the retired token, rather than reporting a parse error
that names neither spelling. -/
private def keywordSpelling : Syntax → String
  | .atom _ value => value
  | keyword => (keyword.getArg 0).getAtomVal

private def retiredKeywordMessage (retired replacement : String) : String :=
  s!"the Model command keyword '{retired}' is retired; write '{replacement}'"

private def rejectRetiredKeyword (keyword : Syntax) (retired replacement : String) :
    CommandElabM Unit := do
  if keywordSpelling keyword == retired then
    throwErrorAt keyword (retiredKeywordMessage retired replacement)

private def rejectRetiredMacroKeyword (keyword : Syntax) (retired replacement : String) :
    MacroM Unit := do
  if keywordSpelling keyword == retired then
    Lean.Macro.throwErrorAt keyword (retiredKeywordMessage retired replacement)

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

elab "model" name:ident "role" role:ident
    "states" stateType:ident
    "actions" actionType:ident "outcomes" outcomeType:ident "facts" factType:ident
    startsKeyword:(&"starts" <|> "initial") "[" initialRefs:ident,+ "]"
    endsKeyword:(&"ends" <|> "terminal") "[" terminalRefs:ident,+ "]"
    stepsKeyword:(&"steps" <|> "transitions")
    rows:successStep+ : command => do
  rejectRetiredKeyword startsKeyword "initial" "starts"
  rejectRetiredKeyword endsKeyword "terminal" "ends"
  rejectRetiredKeyword stepsKeyword "transitions" "steps"
  let stateCtors ← domainConstructors "state" stateType
  let actionCtors ← domainConstructors "action" actionType
  let outcomeCtors ← domainConstructors "outcome" outcomeType
  let factCtors ← domainConstructors "fact" factType
  let actionSpellings := actionCtors.map fun constructor => (shortName constructor).toString
  for pair in actionSpellings.zip actionSpellings.tail do
    unless pair.1 < pair.2 do
      throwErrorAt actionType (unsortedActionsMessage pair.2 pair.1)
  let initialStates ← initialRefs.getElems.toList.mapM (resolveMember "state" stateCtors)
  let terminalStates ← terminalRefs.getElems.toList.mapM (resolveMember "state" stateCtors)
  let setupConstructorName : Name := match initialStates.head? with
    | some first => shortName first.getId
    | none => `setup
  let initialPairs := initialStates.zip initialRefs.getElems.toList
  for pair in initialPairs.zip initialPairs.tail do
    let earlier := (shortName pair.1.1.getId).toString
    let later := (shortName pair.2.1.getId).toString
    unless earlier < later do
      throwErrorAt pair.2.2 (unsortedInitialMessage later earlier)
  if rows.size > transitionBound then
    throwErrorAt rows[transitionBound]! (transitionBoundMessage rows.size)
  let resolvedRows ← rows.toList.mapM fun (row : TSyntax `successStep) => do
    match row with
    | `(successStep| $key:ident : $source:ident + $selected:ident →
        { state := $resulting:ident , outcome := $outcomeRef:ident ,
          facts := [$observed,*] }) => do
        let sourceState ← resolveMember "state" stateCtors source
        let selectedAction ← resolveMember "action" actionCtors selected
        let targetState ← resolveMember "state" stateCtors resulting
        let resolvedOutcome ← resolveMember "outcome" outcomeCtors outcomeRef
        let observedFacts ← observed.getElems.toList.mapM (resolveMember "fact" factCtors)
        let keyLiteral := Lean.quote key.getId.eraseMacroScopes.toString
        let rowTerm ← `(term|
          { key := $keyLiteral
            source := $sourceState
            action := $selectedAction
            results := [step $resolvedOutcome $targetState
              [$(observedFacts.toArray),*]] })
        pure ({ key, sourceState, selectedAction, targetState, rowTerm : ResolvedRow })
    | _ => throwErrorAt row "unsupported Model step"
  let mut declared : List ResolvedRow := []
  for resolved in resolvedRows do
    if let some prior := declared.find? fun candidate =>
        candidate.sourceState.getId == resolved.sourceState.getId &&
          candidate.selectedAction.getId == resolved.selectedAction.getId then
      throwErrorAt resolved.key
        (duplicateTransitionMessage resolved.key.getId.eraseMacroScopes.toString
          prior.key.getId.eraseMacroScopes.toString
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
  elabCommand (← `(command|
    def $name := declareModel $origin $names ($setupName)
      ([$(memberIdents stateCtors),*]) ([$(memberIdents actionCtors),*])
      ([$(memberIdents outcomeCtors),*]) ([$(memberIdents factCtors),*])
      ([$(initialStates.toArray),*]) ([$(terminalStates.toArray),*])
      ([$(transitionTerms.toArray),*])
      (by exact ⟨rfl, rfl, rfl⟩)))

macro "property" name:ident "on" modelRef:ident "for" roleRef:ident
    "when" actionKeyword:("action")? actionRef:ident
    requirements:successRequire+ : command => do
    if let some retired := actionKeyword then
      Lean.Macro.throwErrorAt retired (retiredKeywordMessage "when action" "when")
    let ownerKey := Lean.quote name.getId.toString
    let roleKey := Lean.quote roleRef.getId.toString
    let actionKey := Lean.quote actionRef.getId.toString
    let clauses ← requirements.mapM fun requirement => do
      match requirement with
      | `(successRequire| require $label:ident : state $member:ident) =>
          `(term| PropertyRequirement.stateClause
              $(Lean.quote label.getId.toString) $(Lean.quote member.getId.toString))
      | `(successRequire| require $_:ident : resultingState $_:ident) =>
          Lean.Macro.throwErrorAt requirement (retiredKeywordMessage "resultingState" "state")
      | `(successRequire| require $label:ident : outcome $member:ident) =>
          `(term| PropertyRequirement.outcomeClause
              $(Lean.quote label.getId.toString) $(Lean.quote member.getId.toString))
      | `(successRequire| require $label:ident : fact $member:ident) =>
          `(term| PropertyRequirement.factClause
              $(Lean.quote label.getId.toString) $(Lean.quote member.getId.toString))
      | _ => Lean.Macro.throwErrorAt requirement "unsupported require clause"
    `(command| def $name (values : ModelVocabulary) : Property :=
        authoredProperty ($modelRef) values {
          declaration := $ownerKey
          roleName := $roleKey
          actionSpelling := $actionKey
          requirements := [$clauses,*]
        })

elab scenarioKeyword:("scenario" <|> "behavior") name:ident "on" modelRef:ident roleRef:ident
    "starts" setupRef:ident
    "actions" "exactly" "[" occurrences:successOccurrence,+ "]" : command => do
    rejectRetiredKeyword scenarioKeyword "behavior" "scenario"
    let ownerKey := Lean.quote name.getId.toString
    let roleKey := Lean.quote roleRef.getId.toString
    let setupKey := Lean.quote setupRef.getId.toString
    let mut selectedSpellings : Array String := #[]
    let mut entries : Array Term := #[]
    for occurrence in occurrences.getElems do
      match occurrence with
      | `(successOccurrence| $label:ident : $selected:ident) =>
          selectedSpellings := selectedSpellings.push selected.getId.eraseMacroScopes.toString
          entries := entries.push (← `(term|
            ($(Lean.quote label.getId.toString), $(Lean.quote selected.getId.toString))))
      | _ => throwErrorAt occurrence "unsupported Scenario occurrence"
    elabCommand (← `(command|
      def $name (values : ModelVocabulary) : Scenario :=
        authoredScenario ($modelRef) values {
          declaration := $ownerKey
          roleName := $roleKey
          setupState := $setupKey
          occurrences := [$entries,*]
        }))
    -- A `case` block resolves its `evidence` lines against this list, so the Action order the
    -- Scenario fixes is recorded beside the declaration rather than re-derived from the term.
    liftCoreM (Registry.recordScenario {
      declName := (← getCurrNamespace) ++ name.getId, «actions» := selectedSpellings })

macro "limits" name:ident
    stepsKeyword:(&"steps" <|> "transitions") stepCount:num
    actionsKeyword:(&"actions" <|> "selected_actions") actionCount:num
    searchKeyword:(&"search" <|> "candidate_evaluations") searchCount:num : command => do
    rejectRetiredMacroKeyword stepsKeyword "transitions" "steps"
    rejectRetiredMacroKeyword actionsKeyword "selected_actions" "actions"
    rejectRetiredMacroKeyword searchKeyword "candidate_evaluations" "search"
    `(command| def $name : Limits :=
        Limits.bounded $stepCount $actionCount $searchCount)

/-- Record what a `case` block needs to know about a Query: whether it selects a witness, and the
Scenario whose Action order its evidence lines resolve against. -/
private def recordQueryDeclaration
    (name scenarioRef : Ident) (selectsWitness : Bool) : CommandElabM Unit := do
  let scenarioName ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo scenarioRef)
  liftCoreM (Registry.recordQuery {
    declName := (← getCurrNamespace) ++ name.getId
    selectsWitness
    «scenario» := scenarioName })

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
    | `(modelGap| gap $kind:ident $name:str subject $subject:str detail $detail:str) =>
        if declared.contains name.getString then
          throwErrorAt name (duplicateGapMessage name.getString)
        declared := declared.push name.getString
        terms := terms.push (← `(term|
          Origin.knownGap $origin $(← gapKindTerm kind) $name (some $subject) $detail))
    | `(modelGap| gap $kind:ident $name:str detail $detail:str) =>
        if declared.contains name.getString then
          throwErrorAt name (duplicateGapMessage name.getString)
        declared := declared.push name.getString
        terms := terms.push (← `(term|
          Origin.knownGap $origin $(← gapKindTerm kind) $name none $detail))
    | _ => throwErrorAt declared? "unsupported Known Gap"
  `(term| Umpire.KnownGapSet.checkCanonical [$terms,*])

elab "query" name:ident "on" modelRef:ident
    findKeyword:(&"find" <|> "witness") propertyRef:ident "in" scenarioRef:ident
    "limits" limitsRef:ident gaps:modelGap* : command => do
    rejectRetiredKeyword findKeyword "witness" "find"
    let queryKey := Lean.quote name.getId.toString
    let knownGaps ← knownGapsTerm (← originTerm) gaps
    elabCommand (← `(command|
      def $name : Except AdmissionError (CheckedModel ($modelRef)) :=
        check ($modelRef) $queryKey ($limitsRef) ($propertyRef) ($scenarioRef)
          (knownGaps := $knownGaps)))
    recordQueryDeclaration name scenarioRef (selectsWitness := true)

elab "query" name:ident "on" modelRef:ident
    verifyKeyword:(&"verify" <|> "all") propertyRef:ident "in" scenarioRef:ident
    "limits" limitsRef:ident gaps:modelGap* : command => do
    rejectRetiredKeyword verifyKeyword "all" "verify"
    let queryKey := Lean.quote name.getId.toString
    let knownGaps ← knownGapsTerm (← originTerm) gaps
    elabCommand (← `(command|
      def $name : Except AdmissionError (CheckedModel ($modelRef)) :=
        check ($modelRef) $queryKey ($limitsRef) ($propertyRef) ($scenarioRef)
          (knownGaps := $knownGaps) (form := QueryFormKind.verifyClaim)))
    recordQueryDeclaration name scenarioRef (selectsWitness := false)


end Umpire.Command
