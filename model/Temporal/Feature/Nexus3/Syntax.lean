import Lean.Elab.Command
import Lean.Elab.ElabRules
import Temporal.Feature.Nexus3.Authoring

/-! The success-slice command grammar and its expansion into typed Authoring declarations.

The grammar admits whatever the declaring inductives declare: the ordered state, Action, Model
Outcome and Fact domains are the constructors of the named types, in constructor order, and every
identifier a command mentions is resolved against them. There is no admissible-spelling list.
-/

namespace Temporal.Feature.Nexus3

open Umpire
open Lean Elab Command

/-- One `before + action → result` row of a declared model. -/
declare_syntax_cat nexus3Transition

syntax ident ":" ident "+" ident "→"
  "{" "state" ":=" ident "," "outcome" ":=" ident "," "facts" ":=" "[" ident,* "]" "}" :
  nexus3Transition

/-- One labelled occurrence of a declared Action in a Behavior sequence. -/
declare_syntax_cat nexus3Occurrence

syntax ident ":" ident : nexus3Occurrence

/-- The last component of a constructor name, which is the spelling an author writes. -/
private def shortName : Name → Name
  | .str _ spelling => .str .anonymous spelling
  | name => name

/-- The ordered constructors of a named enum-like inductive. -/
private def domainConstructors (typeRef : Ident) : CommandElabM (List Name) := do
  let name ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo typeRef)
  let info ← getConstInfoInduct name
  pure info.ctors

/-- Resolve one authored spelling against a declared domain, reporting an unknown one in place. -/
private def resolveMember (domain : String) (constructors : List Name) (member : Ident) :
    CommandElabM Ident := do
  match constructors.find? fun constructor => shortName constructor == member.getId with
  | some constructor => pure (mkIdentFrom member constructor)
  | none => throwErrorAt member s!"unknown Nexus3 {domain} '{member.getId}'"

private def memberKeys (constructors : List Name) : Array Term :=
  constructors.toArray.map fun constructor => Lean.quote (shortName constructor).toString

private def memberIdents (constructors : List Name) : Array Term :=
  constructors.toArray.map fun constructor => mkIdent constructor

elab "model" name:ident "role" role:ident
    "states" stateType:ident
    "actions" actionType:ident "outcomes" outcomeType:ident "facts" factType:ident
    "initial" "[" initialRefs:ident,+ "]" "terminal" "[" terminalRefs:ident,+ "]" "transitions"
    rows:nexus3Transition+ : command => do
  let stateCtors ← domainConstructors stateType
  let actionCtors ← domainConstructors actionType
  let outcomeCtors ← domainConstructors outcomeType
  let factCtors ← domainConstructors factType
  let setupConstructors ← domainConstructors (mkIdentFrom name `Setup)
  let setupConstructor ← match setupConstructors with
    | [only] => pure (mkIdent only)
    | _ => throwErrorAt name "a Nexus3 model needs exactly one Setup constructor"
  let initialStates ← initialRefs.getElems.toList.mapM (resolveMember "state" stateCtors)
  let terminalStates ← terminalRefs.getElems.toList.mapM (resolveMember "state" stateCtors)
  let transitionTerms ← rows.toList.mapM fun (row : TSyntax `nexus3Transition) => do
    match row with
    | `(nexus3Transition| $key:ident : $source:ident + $selected:ident →
        { state := $resulting:ident , outcome := $outcomeRef:ident ,
          facts := [$observed,*] }) => do
        let sourceState ← resolveMember "state" stateCtors source
        let selectedAction ← resolveMember "action" actionCtors selected
        let resultingState ← resolveMember "state" stateCtors resulting
        let modelOutcome ← resolveMember "outcome" outcomeCtors outcomeRef
        let observedFacts ← observed.getElems.toList.mapM (resolveMember "fact" factCtors)
        let keyLiteral := Lean.quote key.getId.toString
        `(term|
          { key := $keyLiteral
            source := $sourceState
            action := $selectedAction
            results := [Authoring.transitionResult $modelOutcome $resultingState
              [$(observedFacts.toArray),*]] })
    | _ => throwErrorAt row "unsupported Nexus3 transition"
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
    "when" "action" actionRef:ident
    "require" stateClause:ident ":" "resultingState" stateRef:ident
    "require" outcomeClause:ident ":" "outcome" outcomeRef:ident
    "require" factClause:ident ":" "fact" factRef:ident : command => do
    let ownerKey := Lean.quote name.getId.toString
    let roleKey := Lean.quote roleRef.getId.toString
    let stateClauseKey := Lean.quote stateClause.getId.toString
    let outcomeClauseKey := Lean.quote outcomeClause.getId.toString
    let factClauseKey := Lean.quote factClause.getId.toString
    let actionKey := Lean.quote actionRef.getId.toString
    let stateKey := Lean.quote stateRef.getId.toString
    let outcomeKey := Lean.quote outcomeRef.getId.toString
    let factKey := Lean.quote factRef.getId.toString
    `(command| def $name (values : Authoring.ModelVocabulary) : PropertySpec :=
        Authoring.propertySpec ($modelRef) values {
          declaration := $ownerKey
          roleName := $roleKey
          stateClause := $stateClauseKey
          outcomeClause := $outcomeClauseKey
          factClause := $factClauseKey
          actionSpelling := $actionKey
          stateSpelling := $stateKey
          outcomeSpelling := $outcomeKey
          factSpelling := $factKey
        })

macro "behavior" name:ident "on" modelRef:ident roleRef:ident "starts" setupRef:ident
    "actions" "exactly" "[" occurrences:nexus3Occurrence,+ "]" : command => do
    let ownerKey := Lean.quote name.getId.toString
    let roleKey := Lean.quote roleRef.getId.toString
    let setupKey := Lean.quote setupRef.getId.toString
    let entries ← occurrences.getElems.mapM fun occurrence => do
      match occurrence with
      | `(nexus3Occurrence| $label:ident : $selected:ident) =>
          `(term| ($(Lean.quote label.getId.toString), $(Lean.quote selected.getId.toString)))
      | _ => Lean.Macro.throwErrorAt occurrence "unsupported Nexus3 Behavior occurrence"
    `(command| def $name (values : Authoring.ModelVocabulary) : ExactSequenceSpec :=
        Authoring.behaviorSpec ($modelRef) values {
          declaration := $ownerKey
          roleName := $roleKey
          setupState := $setupKey
          occurrences := [$entries,*]
        })

macro "limits" name:ident "transitions" transitionCount:num "selected_actions" actionCount:num
    "candidate_evaluations" candidateCount:num : command =>
    `(command| def $name : QueryLimitSpec :=
        QueryLimitSpec.mk $transitionCount $actionCount $candidateCount)

macro "query" name:ident "on" modelRef:ident "witness" propertyRef:ident "in" behaviorRef:ident
    "limits" limitsRef:ident : command => do
    let queryKey := Lean.quote name.getId.toString
    `(command| def $name : Except Authoring.AdmissionError (Authoring.CheckedModel ($modelRef)) :=
        Authoring.check ($modelRef) $queryKey ($limitsRef) ($propertyRef) ($behaviorRef))

macro "query" _name:ident "on" _modelRef:ident "all" _propertyRef:ident
    "in" _behaviorRef:ident "limits" _limitsRef:ident : command =>
  Lean.Macro.throwError "unsupported Nexus3 success Query spelling"

end Temporal.Feature.Nexus3
