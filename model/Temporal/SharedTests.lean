import Temporal.Shared
import Umpire.Query.Elab

/-! Temporal-owned identity and source contracts.

The 1×/10× specimen counts independent family construction. Its added work is one
`Temporal.Shared.definitionFamily` plus one `DefinitionFamily.id` call per declaration. Source and
metadata construction add one `Temporal.Shared.sourceLocation` and one
`Temporal.Shared.definitionMetadata` record assembly per declaration. None traverses a declaration
collection; the owning language checkers remain separate and their existing work is excluded from
this structural count. -/

namespace Temporal.SharedTests

open Umpire

def family : DefinitionFamily :=
  Temporal.Shared.definitionFamily "nexus.caller"

example : family.root = DefinitionId.of "temporal.nexus.caller" ∧
    family.id "query" "asyncCompletion" =
      DefinitionId.of "temporal.nexus.caller.query.asyncCompletion" := by
  exact ⟨rfl, rfl⟩

#guard_msgs (error, substring := true) in
def foreignRootFamily : DefinitionFamily :=
  Temporal.Shared.definitionFamily
    (root := DefinitionId.of "foreign.nexus.caller") "nexus.caller"

def alternateSource : SourceLocation :=
  Temporal.Shared.sourceLocation "Temporal/SharedTests/Alternate.lean"

/-- The authored defaults: line and column one, the `lean-model` provenance, version one and no
documentation. -/
example : alternateSource = {
      path := "Temporal/SharedTests/Alternate.lean"
      line := 1
      column := 1
      provenance := "lean-model"
    } ∧
    Temporal.Shared.definitionMetadata (family.id "query" "asyncCompletion") .target alternateSource
      "temporal-nexus-caller-query/v1" = {
      id := DefinitionId.of "temporal.nexus.caller.query.asyncCompletion"
      kind := .target
      source := alternateSource
      version := 1
      behaviorVersion := "temporal-nexus-caller-query/v1"
      documentation := ""
    } := by
  exact ⟨rfl, rfl⟩

def oneIndependentIdentity : List DefinitionId :=
  [Temporal.Shared.definitionFamily "nexus.family-0" |>.id "query" "operation"]

def tenIndependentIdentities : List DefinitionId :=
  (List.range 10).map fun index =>
    Temporal.Shared.definitionFamily ("nexus.family-" ++ toString index) |>.id
      "query" "operation"

example : oneIndependentIdentity.length = 1 ∧ tenIndependentIdentities.length = 10 := by
  native_decide

#print axioms Temporal.Shared.definitionFamily

end Temporal.SharedTests
