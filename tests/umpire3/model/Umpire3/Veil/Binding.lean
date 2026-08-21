import Lean
import Umpire3.FirstOrderView
import Umpire3.Veil.Semantics

namespace Umpire3.Veil

open Lean Elab Term Meta

structure ResolvedEnumBinding where
  identifier : String
  veilIdentifier : String
  values : List (String × String)
  deriving DecidableEq, Repr

def canonicalEnumIdentifiers (artifact : FirstOrderArtifact) : List (String × List String) :=
  artifact.sorts.filterMap fun sort =>
    if sort.kind = .enumeration then some (sort.identifier, sort.values) else none

structure ResolvedSemanticBinding (artifact : FirstOrderArtifact) where
  binding : SemanticBinding artifact
  declaration : String
  axioms : List String

structure ResolvedBinding where
  view : ResolvedFirstOrderView
  semantic : ResolvedSemanticBinding view.artifact
  moduleName : String
  concreteModuleName : String
  trustMode : String
  actionLabels : List (String × String)
  actionCoverage : actionLabels.map Prod.fst = view.artifact.actionIdentifiers
  fieldLabels : List (String × String)
  fieldCoverage : fieldLabels.map Prod.fst = view.artifact.stateFields.map FirstOrderField.identifier
  enumLabels : List ResolvedEnumBinding
  enumCoverage : enumLabels.map (fun binding =>
    (binding.identifier, binding.values.map Prod.fst)) = canonicalEnumIdentifiers view.artifact
  propertyIdentifier : String
  propertyLabel : String
  propertyCoverage : propertyIdentifier = view.artifact.property

structure BindingExport where
  binding : ResolvedBinding
  firstOrder : FirstOrderExport
  artifact_eq : binding.view.artifact = firstOrder.view.artifact

def BindingExport.of (binding : ResolvedBinding)
    (firstOrder : FirstOrderExport) : Option BindingExport :=
  if artifact_eq : binding.view.artifact = firstOrder.view.artifact then
    some { binding, firstOrder, artifact_eq }
  else none

private def nameBindingJson (binding : String × String) : Lean.Json := Lean.Json.mkObj [
  ("identifier", binding.fst),
  ("backendIdentifier", binding.snd),
]

private def actionBindingJson (binding : String × String) : Lean.Json := Lean.Json.mkObj [
  ("action", binding.fst),
  ("backendAction", binding.snd),
]

private def enumBindingJson (binding : ResolvedEnumBinding) : Lean.Json := Lean.Json.mkObj [
  ("identifier", binding.identifier),
  ("backendIdentifier", binding.veilIdentifier),
  ("values", Lean.Json.arr (binding.values.map nameBindingJson).toArray),
]

private def semanticBindingJson {artifact : FirstOrderArtifact}
    (binding : ResolvedSemanticBinding artifact) : Lean.Json := Lean.Json.mkObj [
  ("declaration", binding.declaration),
  ("axioms", Lean.Json.arr (binding.axioms.map Lean.toJson).toArray),
  ("trustBadge", if binding.axioms.isEmpty then "kernel" else "kernel-with-declared-axioms"),
]

def BindingExport.compiledJson (exported : BindingExport)
    (semanticHash : String) : Lean.Json := Lean.Json.mkObj [
  ("view", exported.firstOrder.toJson semanticHash),
  ("semanticBinding", semanticBindingJson exported.binding.semantic),
  ("moduleName", exported.binding.moduleName),
  ("concreteModuleName", exported.binding.concreteModuleName),
  ("trustMode", exported.binding.trustMode),
  ("actionLabels", Lean.Json.arr
    (exported.binding.actionLabels.map actionBindingJson).toArray),
  ("fieldLabels", Lean.Json.arr
    (exported.binding.fieldLabels.map nameBindingJson).toArray),
  ("enumLabels", Lean.Json.arr
    (exported.binding.enumLabels.map enumBindingJson).toArray),
  ("propertyLabel", exported.binding.propertyLabel),
]

def BindingExport.toJson (exported : BindingExport)
    (semanticHash sourceDigest : String) : Lean.Json := Lean.Json.mkObj [
  ("formatVersion", "umpire3/veil-binding/v1"),
  ("backendRevision", "300c305e945750ab3fb62de4a79c23161b24da39"),
  ("sourceDigest", sourceDigest),
  ("artifactDigest", "derived"),
  ("binding", exported.compiledJson semanticHash),
]

def BindingExport.json (exported : BindingExport)
    (semanticHash sourceDigest : String) : String :=
  (exported.toJson semanticHash sourceDigest).compress

private def requireConstant (reference : Syntax) (name : Name) : TermElabM Unit := do
  unless (← getEnv).contains name do
    throwErrorAt reference "Veil binding cannot resolve declaration {name}"

private def requireConstructors (reference : Syntax) (name : Name)
    (expected : List String) : TermElabM Unit := do
  let some (.inductInfo information) := (← getEnv).find? name
    | throwErrorAt reference "Veil binding cannot resolve inductive declaration {name}"
  let actual := information.ctors.map (·.getString!) |>.mergeSort (· < ·)
  let expected := expected.mergeSort (· < ·)
  unless actual = expected do
    throwErrorAt reference "Veil declaration {name} has constructors {actual}; expected {expected}"

private def checkModule (reference : Syntax) (moduleName : Name)
    (actions fields : List (String × String)) (enums : List ResolvedEnumBinding)
    (propertyIdentifier : String) (concrete : Bool) : TermElabM Unit := do
  requireConstant reference (moduleName ++ `State)
  requireConstant reference (Name.str moduleName propertyIdentifier)
  let actionLabels := actions.map Prod.snd
  requireConstructors reference (moduleName ++ `Label) actionLabels
  for actionLabel in actionLabels do
    requireConstant reference (Name.str moduleName actionLabel)
  let fieldLabels := fields.map Prod.snd
  requireConstructors reference (moduleName ++ `State ++ `Label) fieldLabels
  for enumBinding in enums do
    let className := Name.str moduleName (enumBinding.veilIdentifier ++ "_EnumClass")
    requireConstant reference className
    for value in enumBinding.values do
      requireConstant reference (Name.str className value.snd)
  if concrete then
    requireConstant reference (moduleName ++ `modelCheckerResult)

declare_syntax_cat veilNameBinding
syntax str " => " ident : veilNameBinding

declare_syntax_cat veilEnumBinding
syntax str " => " ident " [" veilNameBinding,* "]" : veilEnumBinding

private def parseNameBinding (binding : Syntax) : TermElabM (String × String) := do
  let `(veilNameBinding| $identifier:str => $veilIdentifier:ident) := binding
    | throwUnsupportedSyntax
  let some identifier := identifier.raw.isStrLit?
    | throwErrorAt identifier "Veil binding identifier must be a string literal"
  return (identifier, veilIdentifier.getId.toString)

private def parseEnumBinding (binding : Syntax) : TermElabM ResolvedEnumBinding := do
  let `(veilEnumBinding| $identifier:str => $veilIdentifier:ident [$[$values:veilNameBinding],*]) := binding
    | throwUnsupportedSyntax
  let some identifier := identifier.raw.isStrLit?
    | throwErrorAt identifier "Veil enum identifier must be a string literal"
  let values ← values.toList.mapM fun value => parseNameBinding value.raw
  return { identifier, veilIdentifier := veilIdentifier.getId.toString, values }

syntax (name := resolvedVeilBinding)
  "resolved_veil_binding% " ident ident ident ident
  "veil_actions" "[" veilNameBinding,* "]"
  "veil_fields" "[" veilNameBinding,* "]"
  "veil_enums" "[" veilEnumBinding,* "]"
  "veil_property" str "=>" ident
  "veil_trust" ident : term

@[term_elab resolvedVeilBinding] meta def elabResolvedVeilBinding : TermElab :=
    fun stx expectedType? => do
  let `(term| resolved_veil_binding% $viewName:ident $moduleName:ident $concreteModuleName:ident
      $semanticBindingName:ident
      veil_actions [$[$actionBindings],*]
      veil_fields [$[$fieldBindings],*]
      veil_enums [$[$enumBindings],*]
      veil_property $propertyIdentifier:str => $propertyName:ident
      veil_trust $trustModeSyntax:ident) := stx
    | throwUnsupportedSyntax
  let declarationName ← realizeGlobalConstNoOverloadWithInfo viewName
  let declaration ← getConstInfo declarationName
  unless declaration.type.getAppFn.constName? == some ``FirstOrderView do
    throwErrorAt viewName "resolved Veil declaration must have type FirstOrderView"
  let semanticDeclarationName ← realizeGlobalConstNoOverloadWithInfo semanticBindingName
  let semanticDeclaration ← getConstInfo semanticDeclarationName
  unless semanticDeclaration.type.getAppFn.constName? == some ``SemanticBinding do
    throwErrorAt semanticBindingName "resolved Veil semantic declaration must have type SemanticBinding"
  let semanticAxioms ← collectAxioms semanticDeclarationName
  rejectForbiddenAxioms semanticBindingName semanticAxioms
  let semanticAxioms := semanticAxioms.qsort Name.lt |>.map (Syntax.mkStrLit ∘ toString)
  let actionPairs ← actionBindings.toList.mapM fun action => parseNameBinding action.raw
  let fieldPairs ← fieldBindings.toList.mapM fun field => parseNameBinding field.raw
  let enumPairs ← enumBindings.toList.mapM fun enumBinding => parseEnumBinding enumBinding.raw
  let some propertyIdentifier := propertyIdentifier.raw.isStrLit?
    | throwErrorAt propertyIdentifier "Veil property identifier must be a string literal"
  let trustMode := trustModeSyntax.getId.toString
  unless trustMode = "reconstructed" || trustMode = "trusted" do
    throwErrorAt trustModeSyntax "Veil trust mode must be reconstructed or trusted"
  checkModule moduleName moduleName.getId actionPairs fieldPairs enumPairs propertyName.getId.toString false
  checkModule concreteModuleName concreteModuleName.getId actionPairs fieldPairs enumPairs propertyName.getId.toString true
  let actionTerms ← actionPairs.toArray.mapM fun (identifier, veilIdentifier) =>
    `(( $(Syntax.mkStrLit identifier), $(Syntax.mkStrLit veilIdentifier) ))
  let fieldTerms ← fieldPairs.toArray.mapM fun (identifier, veilIdentifier) =>
    `(( $(Syntax.mkStrLit identifier), $(Syntax.mkStrLit veilIdentifier) ))
  let enumTerms ← enumPairs.toArray.mapM fun enumBinding => do
    let valueTerms ← enumBinding.values.toArray.mapM fun (identifier, veilIdentifier) =>
      `(( $(Syntax.mkStrLit identifier), $(Syntax.mkStrLit veilIdentifier) ))
    `(ResolvedEnumBinding.mk
      $(Syntax.mkStrLit enumBinding.identifier)
      $(Syntax.mkStrLit enumBinding.veilIdentifier)
      [$[$valueTerms],*])
  let expanded ← `(ResolvedBinding.mk
    (resolved_first_order% $(mkIdent declarationName))
    (ResolvedSemanticBinding.mk
      $(mkIdent semanticDeclarationName)
      $(Syntax.mkStrLit semanticDeclarationName.toString)
      [$[$semanticAxioms],*])
    $(Syntax.mkStrLit moduleName.getId.toString)
    $(Syntax.mkStrLit concreteModuleName.getId.toString)
    $(Syntax.mkStrLit trustMode)
    [$[$actionTerms],*]
    (by decide)
    [$[$fieldTerms],*]
    (by decide)
    [$[$enumTerms],*]
    (by decide)
    $(Syntax.mkStrLit propertyIdentifier)
    $(Syntax.mkStrLit propertyName.getId.toString)
    (by decide))
  withMacroExpansion stx expanded <| elabTerm expanded expectedType?

end Umpire3.Veil
