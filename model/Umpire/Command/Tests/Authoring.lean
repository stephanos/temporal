import Umpire.Command.Authoring

/-!
Mutation helpers for tests of the Model commands. Every one of them takes a checked declaration
apart and puts it back differently, which is how a test asks what the admission layer rejects. They
are deliberately outside the production module: nothing a command emits calls them.
-/

namespace Umpire.Command.Tests

open Umpire
open Umpire.Command

def withStates
    (table : FiniteTable Setup State Action Outcome Fact)
    (states : List (FiniteCatalogEntry State)) : FiniteTable Setup State Action Outcome Fact :=
  { table with states }

def withTransitions
    (table : FiniteTable Setup State Action Outcome Fact)
    (transitions : List (FiniteTransitionRow State Action Outcome Fact)) :
    FiniteTable Setup State Action Outcome Fact :=
  { table with transitions }

def transitionRow
    (key : String)
    (source : State)
    (selectedAction : Action)
    (results : List (Step State Outcome Fact)) :
    FiniteTransitionRow State Action Outcome Fact :=
  { key, source, action := selectedAction, results }

def withOccurrences (spec : Scenario) (occurrences : List Scenario.Step) : Scenario :=
  spec.withSteps occurrences

def withClauses (spec : Property) (clauses : List PropertyClause) : Property :=
  { spec with clauses }

def reorderedAndDocumented (spec : Property) (documentation : String) : Property :=
  { spec with clauses := spec.clauses.reverse, documentation }

/-- One Scenario occurrence, addressed through the declaring file's own family. -/
def occurrence (origin : Origin) (key : String) (selectedAction : DefinitionId) : Scenario.Step :=
  { id := origin.family.id "occurrence" key, action := selectedAction }

end Umpire.Command.Tests
