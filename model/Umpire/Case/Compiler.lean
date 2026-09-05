import Umpire.Case

/-!
Checked Case compilation keeps source-property specialization in Lean while producing only the
closed Program and Contract vocabulary consumed by runtime Hosts.
-/

namespace Umpire.Case.Compiler

open Umpire
open Umpire.Case

/-- A stable failure for a checked construct outside the closed Case monitor vocabulary. -/
structure LoweringError where
  sourceDefinitionId : String
  source : SourceLocation
  construct : String
  deriving BEq, DecidableEq, Repr

/-- One checked property either lowered to a complete monitor or rejected explicitly. -/
inductive ContractLowering where
  | monitor (sourceDefinition : CaseDefinitionBinding) (rule : ContractRule)
  | unsupported
      (sourceDefinition : CaseDefinitionBinding)
      (source : SourceLocation)
      (construct : String)
  deriving BEq, Repr

/-- Complete checked inputs shared by all Case producers. -/
structure Input where
  version : FormatVersion
  caseId : String
  producerId : String
  producerVersion : String := ""
  definitions : List CaseDefinitionBinding
  sources : List SourceLocation
  knownGaps : List CaseKnownGap
  program : Program
  contractId : String
  properties : List ContractLowering
  contractLimits : ContractLimits
  deriving BEq, Repr

private def lowerProperty : ContractLowering → Except LoweringError ContractRule
  | .monitor sourceDefinition rule =>
      if sourceDefinition.kind == .property then
        pure rule
      else
        .error {
          sourceDefinitionId := sourceDefinition.definitionId
          source := { path := "" }
          construct := "property.definition-kind"
        }
  | .unsupported sourceDefinition source construct =>
      .error { sourceDefinitionId := sourceDefinition.definitionId, source, construct }

/-- Compile declaration-ordered checked properties without weakening unsupported constructs. -/
def compile (input : Input) : Except LoweringError Case := do
  let rules ← input.properties.mapM lowerProperty
  pure {
    version := input.version
    caseId := input.caseId
    metadata := {
      producerId := input.producerId
      producerVersion := input.producerVersion
      definitions := input.definitions
      sources := input.sources
      knownGaps := input.knownGaps
    }
    program := input.program
    contract := { contractId := input.contractId, rules, limits := input.contractLimits }
  }

end Umpire.Case.Compiler
