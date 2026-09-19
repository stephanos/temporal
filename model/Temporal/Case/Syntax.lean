import Temporal.Case.Conventions
import Temporal.Case.EventKind
import Temporal.Case.Registry
import Temporal.Case.Realization.Nexus
import Temporal.Case.Template

/-!
# The `case` command

A Model file's last block, and the one part of the command surface Temporal owns: it names the
realization template that runs the Model, and the recorded history event that confirms each Action
the Scenario selects. The five Model commands it sits beside are `Umpire.Command`'s.

The Case ID prefix is Temporal's too. `fixture` is the only identity slot the grammar has; every
other identity derives from it.
-/

namespace Temporal.Case

open Umpire
open Umpire.Command
open Lean Elab Command

/-- The Definition ID root every Temporal Case ID hangs off. -/
def caseIdRoot : String := "temporal.case"

/-! The switches the Temporal realizations declare, registered once so a Model file's `set` resolves
its `repeat:` against them. -/
register_switch Temporal.Case.Realization.Nexus.implementationSwitch

/-! The `case` block names the `find` Query whose selected trace the Case realizes, the realization
template that runs it, and the recorded history event that confirms each Action the Scenario
selects. `fixture` is the only identity slot: the Case ID is `<caseIdRoot>.<fixture>`, the Program
and Contract IDs derive from it, and the Run scope is the fixture name.

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
  match EventKind.resolve spelling with
  | .error reported => reported
  | .ok _ => ""

private def unknownTemplateMessage (spelling : String) : String :=
  s!"unknown realization template '{spelling}'; declared: nexusOperation, workflow"

private def unknownResponseMessage (spelling : String) : String :=
  s!"unknown Nexus response form '{spelling}'; declared: sync, async"

private def notASetMessage (spelling : String) : String :=
  s!"'{spelling}' is neither a Query declared by a `query` command nor a set declared by a `set` \
command; a `case` block realizes a `find` Query under a `fixture`, or every Query of a functional \
set under identities derived from the set's and each Query's name"

private def notFunctionalMessage (spelling purpose : String) : String :=
  s!"set '{spelling}' is {purpose}; only a functional set compiles to Cases, one per Query"

private def unselectedBySetMessage (spelling : String) (selected : Array String) : String :=
  s!"no Query of the set selects Action '{spelling}'; the set's Queries select: \
{spellingList selected}"

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
  let declaredQuery ← match Umpire.Command.Registry.query? environment queryName with
    | some declared => pure declared
    | none => throwErrorAt queryRef (unregisteredQueryMessage queryName.toString)
  unless declaredQuery.selectsWitness do
    throwErrorAt queryRef (verifyQueryMessage queryName.toString)
  let selected := match Umpire.Command.Registry.scenario? environment declaredQuery.scenario with
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
        if (EventKind.attributesField? kind).isNone then
          throwErrorAt eventKind (unknownEventKindMessage kind)
        mapped := mapped.push (spelling, kind)
    | _ => throwErrorAt line "unsupported Nexus evidence line"
  for spelling in selected do
    unless mapped.any (·.1 == spelling) do
      throwErrorAt name (unmappedActionMessage spelling)
  let fixtureName := fixture.getString
  if let some prior := (Registry.cases environment).find? (·.fixture == fixtureName)
    then throwErrorAt fixture (duplicateFixtureMessage fixtureName prior.caseId)
  let caseId := caseIdRoot ++ "." ++ fixtureName
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
      Umpire.Command.produceCase $queryRef $identityName $realizationName $evidenceName))
  liftCoreM (Registry.recordCase {
    declName := (← getCurrNamespace) ++ name.getId, caseId, fixture := fixtureName })

/-! ### A functional set's Cases

A `case` block over a set realizes every Query the set lists, each under an identity derived from
the set's name and the Query's: the Case ID is `<caseIdRoot>.<set>.<query>` and the fixture
`<set>-<query>`. One realization and one evidence map serve them all, because the Queries run on
one machine. Which claims each Case makes is its own path's: the Producer is handed every class
claim the machine's actions declare and records the ones the path performs. -/

/-- The declarations of the actions one machine steps on, as the machine resolved them, for the
claims their examples make. A timer is stepped on by name and has no `action` declaration, so it is
not among them. -/
private def machineActionDecls (declaredMachine : Umpire.Command.Registry.MachineEntry) :
    Array Name :=
  declaredMachine.actionDecls

elab "case" name:ident
    &"realizes" setRef:ident
    &"as" template:caseTemplate
    &"evidence" lines:caseEvidence+ : command => do
  let setName ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo setRef)
  let environment ← getEnv
  let some declaredSet := Umpire.Command.Registry.set? environment setName
    | throwErrorAt setRef (notASetMessage setName.toString)
  unless declaredSet.purpose == "functional" do
    throwErrorAt setRef (notFunctionalMessage setName.toString declaredSet.purpose)
  -- Every Query of the set, with the Actions its Scenario selects and the Model it runs on.
  let mut queries : Array (Name × Array String × Name) := #[]
  for queryName in declaredSet.queries do
    let some declaredQuery := Umpire.Command.Registry.query? environment queryName
      | throwErrorAt setRef (unregisteredQueryMessage queryName.toString)
    let some declaredScenario :=
        Umpire.Command.Registry.scenario? environment declaredQuery.scenario
      | throwErrorAt setRef (unregisteredQueryMessage queryName.toString)
    queries := queries.push (queryName, declaredScenario.actions, declaredScenario.model)
  let selected := (queries.flatMap (·.2.1)).toList.eraseDups.toArray
  let mut mapped : Array (String × String) := #[]
  for line in lines do
    match line with
    | `(caseEvidence| $selectedAction:ident ← history $eventKind:ident) =>
        let spelling := selectedAction.getId.eraseMacroScopes.toString
        unless selected.contains spelling do
          throwErrorAt selectedAction (unselectedBySetMessage spelling selected)
        if mapped.any (·.1 == spelling) then
          throwErrorAt selectedAction (duplicateEvidenceMessage spelling)
        let kind := eventKind.getId.eraseMacroScopes.toString
        if (EventKind.attributesField? kind).isNone then
          throwErrorAt eventKind (unknownEventKindMessage kind)
        mapped := mapped.push (spelling, kind)
    | _ => throwErrorAt line "unsupported Nexus evidence line"
  for spelling in selected do
    unless mapped.any (·.1 == spelling) do
      throwErrorAt name (unmappedActionMessage spelling)
  let realization ← templateTerm template
  let realizationName := mkIdentFrom name (name.getId ++ `realization)
  elabCommand (← `(command|
    def $realizationName : Umpire.Case.Producer.Realization := $realization))
  for (queryName, selectedByQuery, modelName) in queries do
    let short := queryName.getString!
    -- Each Case carries the evidence lines for the Actions its own path selects: a line for an
    -- Action the path never selects is a rejection at production, not a mapping to keep.
    let mappings ← (mapped.filter fun entry => selectedByQuery.contains entry.1).mapM
      fun entry => `(term|
        Umpire.Case.Producer.EvidenceMapping.mk
          (vocabulary.namedAction $(Lean.quote entry.1)) $(Lean.quote entry.2))
    let fixtureName := declaredSet.name ++ "-" ++ short
    if let some prior := (Registry.cases environment).find? (·.fixture == fixtureName) then
      throwErrorAt setRef (duplicateFixtureMessage fixtureName prior.caseId)
    let caseId := caseIdRoot ++ "." ++ declaredSet.name ++ "." ++ short
    -- The claims the machine's actions make, for the Producer to record the ones this path performs.
    let claims ← match Umpire.Command.Registry.machine? environment modelName with
      | some declaredMachine =>
          let actionRefs := (machineActionDecls declaredMachine).map mkIdent
          `(term| Umpire.Command.classClaims ($(mkIdent modelName)) [$actionRefs,*])
      | none => `(term| [])
    let caseName := mkIdentFrom name (name.getId ++ Name.mkSimple short)
    let identityName := mkIdentFrom name (caseName.getId ++ `identity)
    let evidenceName := mkIdentFrom name (caseName.getId ++ `evidence)
    elabCommand (← `(command|
      def $identityName : Umpire.Case.Producer.Identity :=
        { caseId := $(Lean.quote caseId), fixture := $(Lean.quote fixtureName) }))
    elabCommand (← `(command|
      def $evidenceName : Umpire.Case.Producer.Vocabulary →
          List Umpire.Case.Producer.EvidenceMapping :=
        fun vocabulary => [$mappings,*]))
    elabCommand (← `(command|
      def $caseName : Except Umpire.Case.Compiler.Error
          temporal.server.api.testpilot.v1.Case :=
        Umpire.Command.produceCase $(mkIdent queryName) $identityName $realizationName
          $evidenceName (claims := $claims)))
    liftCoreM (Registry.recordCase {
      declName := (← getCurrNamespace) ++ caseName.getId, caseId, fixture := fixtureName })

end Temporal.Case
