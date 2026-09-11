import Lean.Elab.Command
import Lean.Elab.ElabRules
import Temporal.Case.Registry
import Temporal.Case.Template
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

/-! ### Where a declaration comes from

The semantic family is the enclosing namespace below `Temporal.Feature`, and the source is the file
being elaborated. Two Models that name the same declaration in different files therefore carry
distinct Definition IDs and distinct Provenance sources, without either file saying so. -/

private def decapitalize (segment : String) : String :=
  match segment.toList with
  | [] => segment
  | first :: rest => String.ofList (first.toLower :: rest)

private def semanticFamilyOf (enclosing : Name) : String :=
  let components := enclosing.components.map (·.toString)
  let owned := match components with
    | "Temporal" :: "Feature" :: rest => rest
    | "Temporal" :: rest => rest
    | rest => rest
  ".".intercalate (owned.map decapitalize)

/-- The elaborating file, relative to the Lean package root. Lake elaborates with package-relative
paths already; an absolute one is trimmed so the recorded source does not depend on the checkout. -/
private def packageRelativePath (raw : String) : String :=
  let normalized := raw.replace "\\" "/"
  -- The last segment, not the first: a checkout whose own path contains `/model/` must not shorten
  -- the recorded source to something outside the package.
  (normalized.splitOn "/model/").getLast!

private def originTerm : CommandElabM Term := do
  let family := semanticFamilyOf (← getCurrNamespace)
  let path := packageRelativePath (← getFileName)
  `(term| Authoring.Origin.of $(Lean.quote family) $(Lean.quote path))

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
  let origin ← originTerm
  elabCommand (← `(command|
    def $name := Authoring.successModel $origin $names ($setupConstructor)
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
      | _ => throwErrorAt occurrence "unsupported Nexus Scenario occurrence"
    elabCommand (← `(command|
      def $name (values : Authoring.ModelVocabulary) : Scenario :=
        Authoring.authoredScenario ($modelRef) values {
          declaration := $ownerKey
          roleName := $roleKey
          setupState := $setupKey
          occurrences := [$entries,*]
        }))
    -- A `case` block resolves its `evidence` lines against this list, so the Action order the
    -- Scenario fixes is recorded beside the declaration rather than re-derived from the term.
    liftCoreM (Temporal.Case.Registry.recordScenario {
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
  liftCoreM (Temporal.Case.Registry.recordQuery {
    declName := (← getCurrNamespace) ++ name.getId
    selectsWitness
    «scenario» := scenarioName })

elab "query" name:ident "on" modelRef:ident
    findKeyword:(&"find" <|> "witness") propertyRef:ident "in" scenarioRef:ident
    "limits" limitsRef:ident : command => do
    rejectRetiredKeyword findKeyword "witness" "find"
    let queryKey := Lean.quote name.getId.toString
    elabCommand (← `(command|
      def $name : Except Authoring.AdmissionError (Authoring.CheckedModel ($modelRef)) :=
        Authoring.check ($modelRef) $queryKey ($limitsRef) ($propertyRef) ($scenarioRef)))
    recordQueryDeclaration name scenarioRef (selectsWitness := true)

elab "query" name:ident "on" modelRef:ident
    verifyKeyword:(&"verify" <|> "all") propertyRef:ident "in" scenarioRef:ident
    "limits" limitsRef:ident : command => do
    rejectRetiredKeyword verifyKeyword "all" "verify"
    let queryKey := Lean.quote name.getId.toString
    elabCommand (← `(command|
      def $name : Except Authoring.AdmissionError (Authoring.CheckedModel ($modelRef)) :=
        Authoring.check ($modelRef) $queryKey ($limitsRef) ($propertyRef) ($scenarioRef)
          (form := Authoring.QueryFormKind.verifyClaim)))
    recordQueryDeclaration name scenarioRef (selectsWitness := false)


/-! ### The `case` command

A Model file's last block. It names the `find` Query whose selected trace the Case realizes, the
realization template that runs it, and the recorded history event that confirms each Action the
Scenario selects. `fixture` is the only identity slot: the Case ID is `temporal.case.<fixture>`,
the Program and Contract IDs derive from it, and the Run scope is the fixture name.

It is an elaborator rather than a macro because it resolves names -- the Query's form, the
Scenario's Action order, the admitted history event kinds -- and registers the Case it declares. -/

declare_syntax_cat caseTemplate

syntax ident &"service" str &"operation" str &"responds" ident : caseTemplate
syntax ident &"type" str : caseTemplate

/-- One `evidence` line: the Action the Scenario selects, and the recorded event that confirms it. -/
declare_syntax_cat caseEvidence

syntax ident "←" &"history" ident : caseEvidence

private def spellingList (spellings : Array String) : String :=
  ", ".intercalate spellings.toList

private def unregisteredQueryMessage (spelling : String) : String :=
  s!"'{spelling}' is not a Query declared by a `query` command; a `case` block realizes one"

private def verifyQueryMessage (spelling : String) : String :=
  s!"Query '{spelling}' verifies rather than finds; a Case realizes one selected trace, so its " ++
    "`realizes` Query must be a `find` form"

private def unselectedActionMessage (spelling : String) (selected : Array String) : String :=
  s!"the Scenario never selects Action '{spelling}'; it selects: {spellingList selected}"

private def unmappedActionMessage (spelling : String) : String :=
  s!"the Scenario selects Action '{spelling}' but no `evidence` line says which recorded event " ++
    "confirms it"

private def duplicateEvidenceMessage (spelling : String) : String :=
  s!"Action '{spelling}' already has an `evidence` line"

private def duplicateFixtureMessage (fixture priorCase : String) : String :=
  s!"fixture '{fixture}' is already registered by Case '{priorCase}'"

private def unknownEventKindMessage (spelling : String) : String :=
  match Temporal.Case.EventKind.resolve spelling with
  | .error reported => reported
  | .ok _ => ""

private def unknownTemplateMessage (spelling : String) : String :=
  s!"unknown realization template '{spelling}'; declared: nexusOperation, workflow"

private def unknownResponseMessage (spelling : String) : String :=
  s!"unknown Nexus response form '{spelling}'; declared: sync, async"

private def templateTerm : TSyntax `caseTemplate → CommandElabM Term
  | `(caseTemplate| $named:ident service $service:str operation $operation:str
      responds $responds:ident) => do
      unless named.getId.eraseMacroScopes.toString == "nexusOperation" do
        throwErrorAt named (unknownTemplateMessage named.getId.eraseMacroScopes.toString)
      match responds.getId.eraseMacroScopes.toString with
      | "sync" => `(term| Temporal.Case.Template.nexusOperation $service $operation .sync)
      | "async" => `(term| Temporal.Case.Template.nexusOperation $service $operation .async)
      | spelling => throwErrorAt responds (unknownResponseMessage spelling)
  | `(caseTemplate| $named:ident type $workflowType:str) => do
      unless named.getId.eraseMacroScopes.toString == "workflow" do
        throwErrorAt named (unknownTemplateMessage named.getId.eraseMacroScopes.toString)
      `(term| Temporal.Case.Template.workflow $workflowType)
  | template => throwErrorAt template "unsupported realization template"

elab "case" name:ident &"fixture" fixture:str
    &"realizes" queryRef:ident
    &"as" template:caseTemplate
    &"evidence" lines:caseEvidence+ : command => do
  let queryName ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo queryRef)
  let environment ← getEnv
  let declaredQuery ← match Temporal.Case.Registry.query? environment queryName with
    | some declared => pure declared
    | none => throwErrorAt queryRef (unregisteredQueryMessage queryName.toString)
  unless declaredQuery.selectsWitness do
    throwErrorAt queryRef (verifyQueryMessage queryName.toString)
  let selected := match Temporal.Case.Registry.scenario? environment declaredQuery.scenario with
    | some declared => declared.actions
    | none => #[]
  let mut mapped : Array (String × String) := #[]
  for line in lines do
    match line with
    | `(caseEvidence| $selectedAction:ident ← history $eventKind:ident) =>
        let spelling := selectedAction.getId.eraseMacroScopes.toString
        unless selected.contains spelling do
          throwErrorAt selectedAction (unselectedActionMessage spelling selected)
        if mapped.any (·.1 == spelling) then
          throwErrorAt selectedAction (duplicateEvidenceMessage spelling)
        let kind := eventKind.getId.eraseMacroScopes.toString
        if (Temporal.Case.EventKind.attributesField? kind).isNone then
          throwErrorAt eventKind (unknownEventKindMessage kind)
        mapped := mapped.push (spelling, kind)
    | _ => throwErrorAt line "unsupported Nexus evidence line"
  for spelling in selected do
    unless mapped.any (·.1 == spelling) do
      throwErrorAt name (unmappedActionMessage spelling)
  let fixtureName := fixture.getString
  if let some prior := (Temporal.Case.Registry.cases environment).find? (·.fixture == fixtureName)
    then throwErrorAt fixture (duplicateFixtureMessage fixtureName prior.caseId)
  let caseId := "temporal.case." ++ fixtureName
  let realization ← templateTerm template
  let mappings ← mapped.mapM fun entry => `(term|
    Umpire.Case.Producer.EvidenceMapping.mk
      (vocabulary.namedAction $(Lean.quote entry.1)) $(Lean.quote entry.2))
  let identityName := mkIdentFrom name (name.getId ++ `identity)
  let realizationName := mkIdentFrom name (name.getId ++ `realization)
  let evidenceName := mkIdentFrom name (name.getId ++ `evidence)
  elabCommand (← `(command|
    def $identityName : Umpire.Case.Producer.Identity :=
      { caseId := $(Lean.quote caseId), fixture := $(Lean.quote fixtureName) }))
  elabCommand (← `(command|
    def $realizationName : Umpire.Case.Producer.Realization := $realization))
  elabCommand (← `(command|
    def $evidenceName : Umpire.Case.Producer.Vocabulary →
        List Umpire.Case.Producer.EvidenceMapping :=
      fun vocabulary => [$mappings,*]))
  elabCommand (← `(command|
    def $name : Except Umpire.Case.Compiler.Error
        temporal.server.api.testpilot.v1.Case :=
      Authoring.produceCase $queryRef $identityName $realizationName $evidenceName))
  liftCoreM (Temporal.Case.Registry.recordCase {
    declName := (← getCurrNamespace) ++ name.getId, caseId, fixture := fixtureName })

end Temporal.Feature.Nexus.Success
