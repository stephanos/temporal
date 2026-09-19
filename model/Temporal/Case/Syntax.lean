import Temporal.Case.Conventions
import Temporal.Case.EventKind
import Temporal.Case.Registry
import Temporal.Case.Realization.Nexus

/-!
# The `case` block

A Model file's last block, and the one part of the command surface Temporal owns: it names the set
whose Queries it realizes and the realization that runs them, and, for a machine that declares no
`evidence:` catalog, the recorded history event that confirms each Action the set's Queries select.
The commands it sits beside are `Umpire.Command`'s.

The Case ID prefix is Temporal's too. Every identity derives from the set's name and each Query's:
the Case ID is `<caseIdRoot>.<set>.<query>` and the fixture `<set>-<query>`, so a Case is named by
what it is rather than by a slot of its own.
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

/-- One `evidence` line: the Action the Scenario selects, and the recorded event that confirms it. -/
declare_syntax_cat caseEvidence

syntax ident "←" &"history" ident : caseEvidence

/-- The `evidence` block of a `case` over a set. A set whose machine carries `evidence:` lines
writes none: each fact a witness step records is confirmed by the observation the machine's line
maps it to, read off the witness at production. -/
syntax caseEvidenceBlock := &"evidence" caseEvidence+

private def spellingList (spellings : Array String) : String :=
  ", ".intercalate spellings.toList

private def unregisteredQueryMessage (spelling : String) : String :=
  s!"'{spelling}' is not a Query declared by a `query` command"

private def unmappedActionMessage (spelling : String) : String :=
  s!"the set's Queries select Action '{spelling}' but no `evidence` line says which recorded event " ++
    "confirms it"

private def duplicateEvidenceMessage (spelling : String) : String :=
  s!"Action '{spelling}' already has an `evidence` line"

private def duplicateFixtureMessage (fixture priorCase : String) : String :=
  s!"fixture '{fixture}' is already registered by Case '{priorCase}'"

private def unknownEventKindMessage (spelling : String) : String :=
  match EventKind.resolve spelling with
  | .error reported => reported
  | .ok _ => ""

private def notASetMessage (spelling : String) : String :=
  s!"'{spelling}' is not a set declared by a `set` command; a `case` block realizes every Query of \
a functional set under identities derived from the set's and each Query's name"

private def notFunctionalMessage (spelling purpose : String) : String :=
  s!"set '{spelling}' is {purpose}; only a functional set compiles to Cases, one per Query"

private def unselectedBySetMessage (spelling : String) (selected : Array String) : String :=
  s!"no Query of the set selects Action '{spelling}'; the set's Queries select: \
{spellingList selected}"

/-! ### A functional set's Cases

A `case` block over a set realizes every Query the set lists, each under an identity derived from
the set's name and the Query's. One realization and one evidence map serve them all, because the
Queries run on one machine. Which claims each Case makes is its own path's: the Producer is handed
every class claim the machine's actions declare and records the ones the path performs.

The realization is a `Umpire.Case.Producer.Realization` value, named or written in parentheses:
`as (Temporal.Case.Realization.asyncNexus "umpire.case.service" "complete")`.

It is an elaborator rather than a macro because it resolves names -- the set, its Queries, their
Scenarios' Action orders, the machine's `evidence:` catalog -- and registers the Cases it
declares. -/

/-- The declarations of the actions one machine steps on, as the machine resolved them, for the
claims their examples make. A timer is stepped on by name and has no `action` declaration, so it is
not among them. -/
private def machineActionDecls (declaredMachine : Umpire.Command.Registry.MachineEntry) :
    Array Name :=
  declaredMachine.actionDecls

elab "case" name:ident
    &"realizes" setRef:ident
    &"as" realization:term:max
    block?:(caseEvidenceBlock)? : command => do
  let lines : Array (TSyntax `caseEvidence) := match block? with
    | some block => block.raw[1].getArgs.map (⟨·⟩)
    | none => #[]
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
  -- Lines written at the Case must cover every selected Action; a Case that writes none reads the
  -- machine's own `evidence:` lines along each witness at production.
  if block?.isSome then
    for spelling in selected do
      unless mapped.any (·.1 == spelling) do
        throwErrorAt name (unmappedActionMessage spelling)
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
    -- The claims the machine's actions make, for the Producer to record the ones this path performs,
    -- and the machine's own `evidence:` lines, which a Case with no lines of its own reads.
    let (claims, catalog) ← match Umpire.Command.Registry.machine? environment modelName with
      | some declaredMachine => do
          let actionRefs := (machineActionDecls declaredMachine).map mkIdent
          let pairs ← declaredMachine.evidence.mapM fun (fact, observed) =>
            `(term| ($(Lean.quote fact), $(Lean.quote observed)))
          pure (← `(term| Umpire.Command.classClaims ($(mkIdent modelName)) [$actionRefs,*]),
            ← `(term| ([$pairs,*] : List (String × String))))
      | none => do pure (← `(term| []), ← `(term| ([] : List (String × String))))
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
          $evidenceName (claims := $claims) (evidenceCatalog := $catalog)))
    liftCoreM (Registry.recordCase {
      declName := (← getCurrNamespace) ++ caseName.getId, caseId, fixture := fixtureName })

end Temporal.Case
