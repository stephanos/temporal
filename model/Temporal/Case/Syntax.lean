import Temporal.Case.Conventions
import Temporal.Case.EventKind
import Temporal.Case.Registry
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

end Temporal.Case
