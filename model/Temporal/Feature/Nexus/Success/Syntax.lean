import Lean.Elab.Command
import Lean.Elab.ElabRules
import Temporal.Feature.Nexus.Success.Authoring

/-! The success-slice command grammar and its expansion into typed Authoring declarations.

The grammar admits whatever the declaring inductives declare: the ordered state, Action, Model
Outcome and Fact domains are the constructors of the named types, in constructor order, and every
identifier a command mentions is resolved against them. There is no admissible-spelling list.
-/

namespace Temporal.Feature.Nexus.Success

open Umpire
open Lean Elab Command

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
  s!"unknown Nexus model {domain} '{spelling}'; declared: {spellings constructors}"

private def parameterizedConstructorMessage (domain spelling : String) : String :=
  s!"Nexus model {domain} '{spelling}' takes arguments; a {domain} domain must be an enum-like inductive"

private def duplicateTransitionMessage (key priorKey source selected : String) : String :=
  s!"duplicate Nexus model step '{key}': '{source} + {selected}' is already declared by " ++
    s!"'{priorKey}'"

private def unreachableTerminalMessage (spelling : String) : String :=
  s!"Nexus model end state '{spelling}' is unreachable from every start state"

private def unsortedActionsMessage (earlier later : String) : String :=
  "Nexus model action constructors must be declared in sorted order, because the planner admits " ++
    s!"only a canonically ordered Action catalog; '{later}' precedes '{earlier}'"

private def unsortedInitialMessage (earlier later : String) : String :=
  "Nexus model start states must be declared in sorted order, because the planner admits " ++
    s!"only a canonically ordered start-state list; '{later}' precedes '{earlier}'"

private def transitionBoundMessage (declared : Nat) : String :=
  s!"Nexus model declares {declared} steps; the elaboration bound is {transitionBound}"

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
  s!"the Nexus command keyword '{retired}' is retired; write '{replacement}'"

private def rejectRetiredKeyword (keyword : Syntax) (retired replacement : String) :
    CommandElabM Unit := do
  if keywordSpelling keyword == retired then
    throwErrorAt keyword (retiredKeywordMessage retired replacement)

private def rejectRetiredMacroKeyword (keyword : Syntax) (retired replacement : String) :
    MacroM Unit := do
  if keywordSpelling keyword == retired then
    Lean.Macro.throwErrorAt keyword (retiredKeywordMessage retired replacement)

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
  let setupConstructors ← domainConstructors "setup" (mkIdentFrom name `Setup)
  let setupConstructor ← match setupConstructors with
    | [only] => pure (mkIdent only)
    | _ => throwErrorAt name "a Nexus model needs exactly one Setup constructor"
  let initialStates ← initialRefs.getElems.toList.mapM (resolveMember "state" stateCtors)
  let terminalStates ← terminalRefs.getElems.toList.mapM (resolveMember "state" stateCtors)
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
            results := [Authoring.step $resolvedOutcome $targetState
              [$(observedFacts.toArray),*]] })
        pure ({ key, sourceState, selectedAction, targetState, rowTerm : ResolvedRow })
    | _ => throwErrorAt row "unsupported Nexus model step"
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
  let setupKey := Lean.quote (shortName setupConstructor.getId).toString
  let names ← `(term|
    { declaration := $declarationKey
      roleName := $roleKey
      setup := $setupKey
      stateKeys := [$(memberKeys stateCtors),*]
      actionKeys := [$(memberKeys actionCtors),*]
      outcomeKeys := [$(memberKeys outcomeCtors),*]
      factKeys := [$(memberKeys factCtors),*] })
  elabCommand (← `(command|
    def $name := Authoring.successModel $names ($setupConstructor)
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
          `(term| Authoring.PropertyRequirement.stateClause
              $(Lean.quote label.getId.toString) $(Lean.quote member.getId.toString))
      | `(successRequire| require $_:ident : resultingState $_:ident) =>
          Lean.Macro.throwErrorAt requirement (retiredKeywordMessage "resultingState" "state")
      | `(successRequire| require $label:ident : outcome $member:ident) =>
          `(term| Authoring.PropertyRequirement.outcomeClause
              $(Lean.quote label.getId.toString) $(Lean.quote member.getId.toString))
      | `(successRequire| require $label:ident : fact $member:ident) =>
          `(term| Authoring.PropertyRequirement.factClause
              $(Lean.quote label.getId.toString) $(Lean.quote member.getId.toString))
      | _ => Lean.Macro.throwErrorAt requirement "unsupported Nexus require clause"
    `(command| def $name (values : Authoring.ModelVocabulary) : Property :=
        Authoring.authoredProperty ($modelRef) values {
          declaration := $ownerKey
          roleName := $roleKey
          actionSpelling := $actionKey
          requirements := [$clauses,*]
        })

macro scenarioKeyword:("scenario" <|> "behavior") name:ident "on" modelRef:ident roleRef:ident
    "starts" setupRef:ident
    "actions" "exactly" "[" occurrences:successOccurrence,+ "]" : command => do
    rejectRetiredMacroKeyword scenarioKeyword "behavior" "scenario"
    let ownerKey := Lean.quote name.getId.toString
    let roleKey := Lean.quote roleRef.getId.toString
    let setupKey := Lean.quote setupRef.getId.toString
    let entries ← occurrences.getElems.mapM fun occurrence => do
      match occurrence with
      | `(successOccurrence| $label:ident : $selected:ident) =>
          `(term| ($(Lean.quote label.getId.toString), $(Lean.quote selected.getId.toString)))
      | _ => Lean.Macro.throwErrorAt occurrence "unsupported Nexus Scenario occurrence"
    `(command| def $name (values : Authoring.ModelVocabulary) : Scenario :=
        Authoring.authoredScenario ($modelRef) values {
          declaration := $ownerKey
          roleName := $roleKey
          setupState := $setupKey
          occurrences := [$entries,*]
        })

macro "limits" name:ident
    stepsKeyword:(&"steps" <|> "transitions") stepCount:num
    actionsKeyword:(&"actions" <|> "selected_actions") actionCount:num
    searchKeyword:(&"search" <|> "candidate_evaluations") searchCount:num : command => do
    rejectRetiredMacroKeyword stepsKeyword "transitions" "steps"
    rejectRetiredMacroKeyword actionsKeyword "selected_actions" "actions"
    rejectRetiredMacroKeyword searchKeyword "candidate_evaluations" "search"
    `(command| def $name : Limits :=
        Limits.bounded $stepCount $actionCount $searchCount)

macro "query" name:ident "on" modelRef:ident
    findKeyword:(&"find" <|> "witness") propertyRef:ident "in" scenarioRef:ident
    "limits" limitsRef:ident : command => do
    rejectRetiredMacroKeyword findKeyword "witness" "find"
    let queryKey := Lean.quote name.getId.toString
    `(command| def $name : Except Authoring.AdmissionError (Authoring.CheckedModel ($modelRef)) :=
        Authoring.check ($modelRef) $queryKey ($limitsRef) ($propertyRef) ($scenarioRef))

macro "query" name:ident "on" modelRef:ident
    verifyKeyword:(&"verify" <|> "all") propertyRef:ident "in" scenarioRef:ident
    "limits" limitsRef:ident : command => do
    rejectRetiredMacroKeyword verifyKeyword "all" "verify"
    let queryKey := Lean.quote name.getId.toString
    `(command| def $name : Except Authoring.AdmissionError (Authoring.CheckedModel ($modelRef)) :=
        Authoring.check ($modelRef) $queryKey ($limitsRef) ($propertyRef) ($scenarioRef)
          (form := Authoring.QueryFormKind.verifyClaim))

end Temporal.Feature.Nexus.Success
