import Umpire.Command.Claims
import Umpire.Json

/-!
# What an exploratory set sets out to reach

An exploratory set covers a machine rather than listing Queries: its goals name the rows of the
machine's table, its result values or the members of the classes its actions claim, and its budget
is the `limits` an exploration runs within. The targets are enumerated here, once, from the
declared Model in the machine's own catalog order, so a golden pins them and an exploration reads
them rather than the Model. Running an exploration is fn-33's; what it is asked to reach is this.
-/

namespace Umpire.Command

variable {Setup State Action Outcome Fact : Type}
variable [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]

/-- The states an exploration reaches within `depth` steps of the start states, by the table's own
rows. A row taken at step `k` needs its source within `k - 1` steps, so an exploration of `steps`
steps takes rows whose source is within `steps - 1`. The walk follows terminal rows too, as Search
does. -/
private def within (rows : List (FiniteTransitionRow State Action Outcome Fact))
    (starts : List State) : Nat → List State
  | 0 => starts
  | depth + 1 =>
      let seen := within rows starts depth
      rows.foldl (init := seen) fun seen row =>
        if seen.contains row.source then
          row.results.foldl (init := seen) fun seen result =>
            if seen.contains result.state then seen else seen ++ [result.state]
        else seen

/-- The targets one exploratory set enumerates over a declared Model: for each goal in the order
the set names them, the rows an exploration within the budget's steps can take, in table order; the
result values those rows reach, in catalog order; and the claims of the classes those rows' actions
make, in claim order. The whole list is cut at the budget's search count, so an exploration is
never asked for more than it may try. -/
def coverageTargets (model : DeclaredModel Setup State Action Outcome Fact)
    (claims : List Umpire.Case.Producer.ClassClaim) (goals : List CoverageGoal)
    (budget : Limits) : List CoverageTarget :=
  let sources := within model.table.transitions model.initial (budget.steps.value - 1)
  let rows := model.table.transitions.filter fun row => sources.contains row.source
  let stateId := catalogId model.states model.stateIds
  let actionId := catalogId model.actions model.actionIds
  let outcomeId := catalogId model.outcomes model.outcomeIds
  let rowTargets := rows.map fun row =>
    CoverageTarget.row row.key (stateId row.source) (actionId row.action)
      (row.results.map fun result => outcomeId result.outcome)
  let resultTargets := (model.outcomes.filter fun outcome =>
    rows.any fun row => row.results.any (·.outcome == outcome)).map fun outcome =>
      CoverageTarget.result (outcomeId outcome)
  let memberTargets := (claims.filter fun claim =>
    rows.any fun row => actionId row.action == claim.member).map fun claim =>
      CoverageTarget.classMember claim.member claim.row.action claim.row.field
        claim.row.className claim.row.exampleValue
  let targets := goals.flatMap fun goal =>
    match goal with
    | .rows => rowTargets
    | .results => resultTargets
    | .classMembers => memberTargets
  targets.take budget.search.value

/-- One target as the golden and an exploration read it. -/
def CoverageTarget.json : CoverageTarget → CanonicalJson
  | .row key state action results =>
      .object [("kind", .string "row"), ("key", .string key), ("state", .string state.value),
        ("action", .string action.value),
        ("results", .array (results.map fun outcome => .string outcome.value))]
  | .result outcome => .object [("kind", .string "result"), ("outcome", .string outcome.value)]
  | .classMember member action field className exampleValue =>
      .object [("kind", .string "classMember"), ("member", .string member.value),
        ("action", .string action), ("field", .string field), ("class", .string className),
        ("example", .string exampleValue)]

/-- An exploratory set's coverage, as its golden pins it: the set, the machine it covers, its goals
and budget, and the targets in the order they are enumerated. -/
def coverageJson (declared : SetDeclaration) : CanonicalJson :=
  .object [
    ("set", .string declared.id.value),
    ("purpose", .string declared.purpose.name),
    ("machine", CanonicalJson.ofOption (fun machine => .string machine.value) declared.machine),
    ("cover", .array (declared.cover.map fun goal => .string goal.name)),
    ("budget", CanonicalJson.ofOption CanonicalJson.string declared.budget),
    ("targets", .array (declared.targets.map CoverageTarget.json))]

end Umpire.Command
