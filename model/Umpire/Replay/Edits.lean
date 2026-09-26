import Umpire.Command.Authoring

/-!
# The edits a reduction tries

A reduction shortens a violated Query's Scenario while the Property stays what it was. It tries one
kind of edit, `dropPrefixStep i`: occurrence `i` of the Scenario's exact action sequence is dropped,
every step before the target row being a candidate, last first. Every Scenario ends on its target
row and the Producer rejects a silent step at the end of a path, so a silent step is a prefix step
and no second kind of edit is needed to name it.

The edit is over the Scenario the Query's behavior author writes, as `Scenario.exactly` writes it:
the occurrences are kept or dropped by position, the schedule fields are rebuilt from the kept ones
by `Scenario.withSteps`, and an exact trace (what an exploration candidate's Scenario also sets) loses
the same step. Whether the Model still admits what is left is not decided here: the edited Query is
re-admitted, and one the Model does not admit is `inapplicable`.
-/

namespace Umpire.Replay

open Umpire.Command

/-- `dropPrefixStep index`: the occurrence at `index` of the subject's exact action sequence,
zero-based, which performs `action`. -/
structure Edit where
  index : Nat
  action : DefinitionId
  deriving BEq, DecidableEq, Repr

def Edit.name (edit : Edit) : String := s!"dropPrefixStep {edit.index}"

/-- The exact action sequence edits are defined over: the Scenario's `actionsExactly`, which its
required occurrences and any exact trace follow step for step. A Scenario of another shape has no
prefix to edit. -/
def editable (scenario : Scenario) : Except String (List DefinitionId) := do
  let some actions := scenario.actionsExactly
    | throw "the Scenario states no exact action sequence"
  unless scenario.requiredOccurrences.map (·.action) == actions do
    throw "the Scenario's required occurrences are not its exact action sequence"
  if scenario.traceExactly.any (·.steps.length != actions.length) then
    throw "the Scenario's exact trace is not its exact action sequence"
  pure actions

/-- One sweep's edits in the order they are tried: every step before the last, last first. -/
def sweepOf (actions : List DefinitionId) : List Edit :=
  (actions.dropLast.zipIdx.map fun (action, index) => { index, action }).reverse

/-- The Scenario with only the occurrences at `kept` positions, in their order. The schedule fields
are rebuilt from them, and an exact trace keeps the same steps. -/
def restrict (scenario : Scenario) (kept : List Nat) : Scenario :=
  let keep {α : Type} (items : List α) : List α :=
    (items.zipIdx.filter fun (_, index) => kept.contains index).map (·.1)
  let restricted := scenario.withSteps (keep scenario.requiredOccurrences)
  { restricted with
    traceExactly := scenario.traceExactly.map fun trace => { trace with steps := keep trace.steps } }

variable {Setup State Action Outcome Fact : Type}
variable [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]

/-- The Query with its Scenario restricted to `kept`; every other field is the block's own. -/
def restrictSource {model : DeclaredModel Setup State Action Outcome Fact}
    (source : QuerySource model) (kept : List Nat) : QuerySource model :=
  { source with behavior := fun values => restrict (source.behavior values) kept }

end Umpire.Replay
