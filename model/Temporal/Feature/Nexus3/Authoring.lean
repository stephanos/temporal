import Temporal.Shared
import Umpire.Planning
import Lean.Data.Name

/-! Small construction helpers for the declaration-named Nexus3 demonstration. -/

namespace Temporal.Feature.Nexus3.Authoring

open Umpire

def family : DefinitionFamily := Temporal.Shared.definitionFamily "nexus3"

def source : SourceLocation :=
  Temporal.Shared.sourceLocation "Temporal/Feature/Nexus3/Nexus.lean"

/-- Use the final component of a Lean declaration name as its default stable key. -/
def declarationKey
    (override : Option String := none)
    (declarationName : Lean.Name := by exact decl_name%) : String :=
  override.getD (Lean.Name.getString! declarationName)

/-- A declaration override replaces only the declaration-local identity key. -/
def declarationId
    (kind : String)
    (override : Option String := none)
    (declarationName : Lean.Name := by exact decl_name%) : DefinitionId :=
  family.id kind (declarationKey override declarationName)

def memberId (kind : String) (name : Lean.Name) : DefinitionId :=
  family.id kind (declarationKey none name)

def metadata
    (id : DefinitionId)
    (kind : DefinitionKind)
    (canonicalBehavior : String) : DefinitionMetadata :=
  Temporal.Shared.definitionMetadata id kind source canonicalBehavior

inductive FiniteAdmissionError where
  | outgoingTerminalTransition
  | noncanonicalTable
  | finite (error : FiniteTargetAdmissionError)

/-- Admit the finite Target after enforcing that the demonstration's terminal state has no rows. -/
def checkFiniteTarget [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Fact]
    (table : FiniteTable Setup State Action Outcome Fact)
    (canonicalTable : FiniteTable Setup State Action Outcome Fact)
    (identity : FiniteModelIdentity Setup State Action Outcome Fact)
    (definition : FiniteTargetDefinition)
    (composition : TargetComposition LawStatement)
    (terminal : State → Bool) : Except FiniteAdmissionError (QueryTarget LawStatement) := do
  let _ ← table.validateModel identity
    |>.mapError (FiniteAdmissionError.finite ∘ FiniteTargetAdmissionError.invalidTable)
  if table.transitions.any fun row => terminal row.source then
    throw .outgoingTerminalTransition
  if table ≠ canonicalTable then
    throw .noncanonicalTable
  table.checkModelTarget identity definition composition |>.mapError .finite

end Temporal.Feature.Nexus3.Authoring
