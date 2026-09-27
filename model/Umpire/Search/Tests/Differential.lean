import Umpire.Command
import Umpire.Search.Admission
import Umpire.Search.Tests.Fixtures
import Umpire.Search.Tests.Monitor
import Umpire.Examples.Switch
import Umpire.Replay.Tests
import Umpire.Variations.Compiler
import Umpire.Variations.Tests.Fixtures

/-!
# The two backends against each other

The R8 differential: a Query runs on `reference` and on `veil` through `AdmittedQuery.searchWith`,
and the two results are compared exactly as the fn-88 API Contracts define. Where the reference
terminates, the outcomes, the witness `Scenario.Trace`, the Plan artifact bytes except `explored`,
and the planning receipt JSON except the exempt fields must be equal. The exempt receipt fields are
the four backend fields (`searchBackend`, `backendReason`, `searchUnit`, `veilCommit`), the
`SearchStats` counter the receipt carries (`enumeratorPulls`), the `triggers` evidence, the
`explored` counts, and, for a `verify` that found a counterexample, `searchComplete`,
`searchTermination` and `coverage`. The receipt's `explored` is the same `ExploredCounts` as the
Plan's exempt `explored`, counting product states under `veil` by design; the spec's API Contracts
were amended to say so, and each run's Plan `explored` must still equal its receipt's. Where
the reference reports `limit-reached`, `veil` must not be `invalid` (its witness passed the kernel
replay gate), and what it reports must not contradict what the reference examined: an admitted
trace, an exercised coverage requirement, a fired clause.

`sweep` runs the comparison over every `query` declaration the environment records, together with
the Scenario automaton and Property monitor checks of `agreement`, and reports one line per Query;
the Temporal feature Models, which these tests cannot import, run it from
`TemporalModelTests.SearchDifferential`. Here it covers the Umpire Models; the Switch's three
hand-admitted Queries, every point of its checked variation space, and the unsatisfiable-Scenario
Query Promotion's tests derive from it are compared as well.

The module also holds the R15 witness-order fixture, where two paths reach one product state at
the same depth, and the R9 three-instance fixture, whose paths outgrow the reference's search bound
while its states do not. Wall times are recorded in the task evidence, never asserted.
-/

namespace Umpire.SearchTests.Differential

open Umpire
open Umpire.Command

/-! ### The comparison -/

private def renderValue (value : ModelValue) : String :=
  value.definitionId.value ++ "=" ++ value.value

/-- A trace on one line: the setup, the initial state, then each step's action, outcome and
resulting state. -/
def renderTrace (trace : Scenario.Trace) : String :=
  "[" ++ ", ".intercalate (trace.setup.map fun binding =>
      binding.role.value ++ ":" ++ renderValue binding.value) ++
    "] " ++ renderValue trace.trace.initialState ++
    String.join (trace.trace.steps.map fun step =>
      " -> " ++ renderValue step.selectedAction ++ " / " ++ renderValue step.outcome ++ " / " ++
        renderValue step.state)

private def renderOutcome : PlanningOutcome → String
  | .found trace reason => "found (" ++ reason.name ++ ") " ++ renderTrace trace
  | .invalid error => "invalid (" ++ error.kind.name ++ ") " ++ error.offendingValue
  | outcome => outcome.name

/-- The Plan with its `explored` counts replaced, both checksums resealed over the new content. -/
def withExplored (plan : Plan) (explored : ExploredCounts) : Plan :=
  let steps := { plan.plan with explored }
  let steps := { steps with artifactChecksum := steps.expectedArtifactChecksum }
  let plan := { plan with plan := steps }
  { plan with artifactChecksum := plan.expectedArtifactChecksum }

/-- The receipt fields a comparison exempts. -/
def exemptReceiptFields (verifyCounterexample : Bool) : List String :=
  ["searchBackend", "backendReason", "searchUnit", "veilCommit", "enumeratorPulls", "triggers",
    "explored"] ++
  if verifyCounterexample then ["searchComplete", "searchTermination", "coverage"] else []

/-- The receipt JSON with every exempt field set to `null`. -/
def comparableReceipt (run : PlanResult) (verifyCounterexample : Bool) : Option Lean.Json :=
  (Lean.Json.parse (canonicalPlanningReceiptJson run)).toOption.map fun json =>
    (exemptReceiptFields verifyCounterexample).foldl (fun json key => json.setObjVal! key .null)
      json

private def receiptFields (json : Lean.Json) : List (String × Lean.Json) :=
  match json.getObj? with
  | .ok fields => fields.toList
  | .error _ => []

/-- The receipt keys whose values differ. -/
private def differingKeys (left right : Lean.Json) : List String :=
  let keys := ((receiptFields left).map (·.1) ++ (receiptFields right).map (·.1)).eraseDups
  keys.filter fun key => left.getObjValD key != right.getObjValD key

private def isVerifyCounterexample (form : Query.Form) (outcome : PlanningOutcome) : Bool :=
  match form, outcome with
  | .verify _, .found _ .violatingCounterexample => true
  | _, _ => false

/-- The comparison where the reference terminated: every field equal but the exempt ones. -/
private def terminatedDifference (form : Query.Form) (reference veil : PlanResult) :
    Option String :=
  let both := "reference " ++ renderOutcome reference.result.outcome ++ "; veil " ++
    renderOutcome veil.result.outcome
  if reference.result.outcome != veil.result.outcome then
    some ("outcome differs: " ++ both)
  else
    let plans := match reference.artifact, veil.artifact with
      | none, none => none
      | some left, some right =>
          if !(right.hasValidArtifactChecksum && right.plan.hasValidArtifactChecksum) then
            some "the veil Plan's checksums do not seal its content"
          else if canonicalPlanBytes (withExplored right left.plan.explored) !=
              canonicalPlanBytes left then
            some ("Plan bytes differ outside explored: " ++ both)
          else none
      | _, _ => some ("one backend selected a Plan and the other did not: " ++ both)
    let ownExplored := [reference, veil].all fun run =>
      run.artifact.all (·.plan.explored == run.result.metadata.explored)
    let plans := if ownExplored then plans else
      some ("a run's Plan explored counts are not its receipt's: " ++ both)
    plans.orElse fun _ =>
      let exempt := isVerifyCounterexample form reference.result.outcome
      match comparableReceipt reference exempt, comparableReceipt veil exempt with
      | some left, some right =>
          if left == right then none
          else some ("receipt differs in " ++ ", ".intercalate (differingKeys left right) ++
            ": " ++ both)
      | _, _ => some "a receipt is not JSON"

private def triggeredClauses (run : PlanResult) : List (DefinitionId × DefinitionId) :=
  run.result.metadata.validity.triggers.map fun evidence =>
    (evidence.trigger.propertyId, evidence.trigger.clauseId)

/-- The comparison where the reference reached its limit: `veil` is not `invalid`, and it reports
no less than the reference examined. -/
private def limitDifference (reference veil : PlanResult) : Option String :=
  let referenceValidity := reference.result.metadata.validity
  let veilValidity := veil.result.metadata.validity
  match veil.result.outcome with
  | .invalid _ => some ("veil is invalid where the reference reached its limit: " ++
      renderOutcome veil.result.outcome)
  | outcome =>
      if referenceValidity.satisfiability == .nonempty &&
          veilValidity.satisfiability != .nonempty then
        some ("veil reports no admitted trace, and the reference examined one: veil " ++
          outcome.name)
      else if referenceValidity.coverage == .exercised && veilValidity.coverage != .exercised then
        some ("veil reports coverage unexercised, and the reference exercised it: veil " ++
          outcome.name)
      else if !(triggeredClauses reference).all (triggeredClauses veil).contains then
        some ("a clause the reference saw fire has no veil trigger evidence: veil " ++ outcome.name)
      else none

/-- The first difference between the two backends' results for one Query, or `none` when they
agree as the API Contracts define. -/
def difference (form : Query.Form) (reference veil : Except KnownGapError PlanResult) :
    Option String :=
  match reference, veil with
  | .error left, .error right =>
      if left == right then none else some "the Known Gaps are rejected differently"
  | .error _, .ok _ | .ok _, .error _ => some "one backend's Known Gaps are rejected"
  | .ok reference, .ok veil =>
      match reference.result.outcome with
      | .limitReached => limitDifference reference veil
      | _ => terminatedDifference form reference veil

/-- One Query's differential, on one line: the backend `searchWith .veil` ran and why, both
outcomes with what each explored, or the difference. -/
def line (form : Query.Form) (reference veil : Except KnownGapError PlanResult) : String :=
  match difference form reference veil, reference, veil with
  | some difference, _, _ => "DIFFERS " ++ difference
  | none, .ok reference, .ok veil =>
      let backend := veil.instrumentation.searchBackend.name ++ " " ++
        veil.instrumentation.backendReason.name
      let referenceText := reference.result.outcome.name ++ " " ++
        toString reference.result.metadata.explored.traces ++ " paths"
      if veil.instrumentation.searchBackend == .reference then
        backend ++ ", " ++ referenceText
      else
        backend ++ ", " ++ referenceText ++ ", " ++ veil.result.outcome.name ++ " " ++
          toString veil.result.metadata.explored.traces ++ " states"
  | none, _, _ => "rejected Known Gaps on both"

/-- Each backend's outcome and what it explored -- candidate paths on `reference`, product states on
`veil` -- and the difference between them. -/
def compared (form : Query.Form) (reference veil : Except KnownGapError PlanResult) :
    Option (String × Nat × String × Nat × Option String) := do
  let referenceRun ← reference.toOption
  let veilRun ← veil.toOption
  pure (referenceRun.result.outcome.name, referenceRun.result.metadata.explored.traces,
    veilRun.result.outcome.name, veilRun.result.metadata.explored.traces,
    difference form reference veil)

/-- Both backends through an admitted Query's own view. -/
def runs {LawStatement : Law → Prop} {target : QueryModel LawStatement}
    (admitted : AdmittedQuery target) :
    Except KnownGapError PlanResult × Except KnownGapError PlanResult :=
  (admitted.searchWith .reference, admitted.searchWith .veil)

def admittedLine {LawStatement : Law → Prop} {target : QueryModel LawStatement}
    (admitted : AdmittedQuery target) : String :=
  let (reference, veil) := runs admitted
  line admitted.query.form reference veil

def admittedCompared {LawStatement : Law → Prop} {target : QueryModel LawStatement}
    (admitted : AdmittedQuery target) : Option (String × Nat × String × Nat × Option String) :=
  let (reference, veil) := runs admitted
  compared admitted.query.form reference veil

/-! ### The product and monitor checks on one Query -/

/-- Every trace of the view within `depth` whose action sequence the Scenario's prefix check admits,
and each one-step extension it rejects, in the reference key order. A Model's traces outgrow any
bound over several instances; the ones the Scenario prunes are rejected by `admitsPrefix` itself,
which `Product.agrees` checks the automaton drops. -/
def scenarioTraces {LawStatement : Law → Prop} {target : QueryModel LawStatement}
    (view : SearchView target) (behavior : CheckedScenario) (setups : List (List RoleBinding))
    (depth : Nat) : List Scenario.Trace :=
  let rec from' : Nat → Scenario.Trace → ModelValue → List Scenario.Trace
    | 0, trace, _ => [trace]
    | depth + 1, trace, state =>
        trace :: (List.range view.actionLimit).flatMap fun actionIndex =>
          match view.actionAt actionIndex with
          | none => []
          | some action =>
              (List.range (view.stepLimit state action)).flatMap fun outcomeIndex =>
                match view.stepAt state action outcomeIndex with
                | none => []
                | some result =>
                    let next := Product.appendStep trace action result
                    if behavior.admitsPrefix (Product.actionsOf next) then
                      from' depth next result.state
                    else [next]
  setups.flatMap fun setup =>
    (List.range (view.initialLimit setup)).flatMap fun index =>
      match view.initialAt setup index with
      | none => []
      | some initialState =>
          from' depth { setup, trace := { initialState, steps := [] } } initialState

/-- Every Model trace a monitor check reads within `depth`, from the Query's own setups. -/
private def modelTraces {LawStatement : Law → Prop} (query : CheckedQuery LawStatement)
    (view : SearchView query.target) (depth : Nat) :
    List (ModelTrace ModelValue ModelValue ModelValue ModelValue) :=
  let setups := match query.completeness with
    | some evidence => evidence.roleAssignments
    | none => query.target.resolvedSetups
  (Product.allTraces view setups depth).map (·.trace)

/-- The deepest depth up to `depth` whose cost, growing with the depth, stays within `budget`, and
at least zero. It stops at the first depth over budget: a deeper one costs more to measure. -/
private def deepestWithin (depth budget : Nat) (cost : Nat → Nat) : Nat :=
  let rec go : Nat → Nat → Nat
    | 0, reached => reached
    | remaining + 1, reached =>
        if cost (reached + 1) ≤ budget then go remaining (reached + 1) else reached
  go depth 0

/-- The Scenario automaton and the Property monitors against their oracles on one Query.

The automaton agrees with `admits` on every trace of `scenarioTraces` within the Limits, and the
product's accepted paths decode to exactly the admitted ones (`Product.agrees`,
`Product.pathsFrom`). The monitors agree with the evaluator under both endings
(`Monitor.monitorsAgree`) on every Model trace, admitted or not: for the Query's own Properties to
the deepest depth within the Limits with at most 5000 traces, and for the clause table
`Monitor.generatedProperties` builds from those traces -- every supported kind, both units -- to the
deepest depth where traces times clauses stay within 100000; and on every path of the Query's own
product within the Limits (`Monitor.productAgrees`). Several interleaved instances outgrow any
exhaustive bound, which is why the depths are chosen by cost and printed. -/
def agreement {LawStatement : Law → Prop} (query : CheckedQuery LawStatement) : String :=
  match SearchView.ofCheckedQuery query.target.id query with
  | .error _ => "no search view"
  | .ok view =>
      let depth := Nat.min query.limits.steps.value query.limits.actions.value
      let automaton := match Search.Product.StateSpace.build query view .empty with
        | .error _ => none
        | .ok space =>
            let traces := scenarioTraces view query.behavior space.setups depth
            let accepted := space.initialStates.flatMap fun root =>
              (Product.pathsFrom space depth root []).filterMap fun (state, reversed) =>
                if space.accepts state then some (Search.Product.decode root reversed.reverse)
                else none
            some (Product.agrees query.behavior traces &&
              accepted == traces.filter query.behavior.admits)
      let monitored := match Search.Product.MonitoredProduct.build query view with
        | .error _ => none
        | .ok _ =>
            let fields := query.target.stateFields
            let ownDepth := deepestWithin depth 5000 fun candidate =>
              (modelTraces query view candidate).length
            let clauseDepth := deepestWithin ownDepth 100000 fun candidate =>
              let traces := modelTraces query view candidate
              traces.length * (Monitor.generatedProperties fields traces).length
            let clauseTraces := modelTraces query view clauseDepth
            let agrees :=
              Monitor.monitorsAgree query.form.properties fields
                  (modelTraces query view ownDepth) &&
                (Monitor.generatedProperties fields clauseTraces).all (fun property =>
                  Monitor.monitorsAgree [property] fields clauseTraces) &&
                Monitor.productAgrees query view depth
            some (agrees, ownDepth, clauseDepth)
      let monitors := match monitored with
        | none => ""
        | some (true, ownDepth, clauseDepth) =>
            s!", monitors ok (Query depth {ownDepth}, clause table depth {clauseDepth})"
        | some (false, _, _) => ", monitors DISAGREE"
      match automaton with
      | some false => "automaton DISAGREES" ++ monitors
      | none => "no automaton" ++ monitors
      | some true => "automaton ok" ++ monitors

/-! ### Every declared Query -/

section Declared

variable {Setup State Action Outcome Fact : Type}
  [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
  [DecidableEq Setup] [DecidableEq State] [DecidableEq Action] [DecidableEq Outcome]
  [DecidableEq Fact]
  {model : DeclaredModel Setup State Action Outcome Fact}

/-- A Query's admission over a declared Model, as `checkAdmitted` makes it, stopping before the
search. `behavior := none` admits a Property-only Query, which constrains no trace. -/
def admitQuery (model : DeclaredModel Setup State Action Outcome Fact) (key : String)
    (property : ModelVocabulary → Property) (behavior : Option (ModelVocabulary → Scenario))
    (form : QueryFormKind) (limits : Limits) (knownGaps : List KnownGap := []) :
    Except AdmissionError ((target : QueryModel model.lawStatement) × AdmittedQuery target) := do
  let target ← checkFiniteTarget model.table model.table model.identity model.modelSpec
    model.composition |>.mapError .invalidTarget
  let vocabulary ← modelVocabulary model model.table |>.mapError .invalidVocabulary
  let admitted ← Search.admit target (property vocabulary) (behavior.map (· vocabulary)) {
    id := model.origin.family.id "query" key
    source := model.origin.source
    target := model.targetId
    form := match form with
      | .selectWitness => .find
      | .verifyClaim => .verify
    limits
    policy := match form with
      | .selectWitness => .shortest
      | .verifyClaim => .exhaustive
  } knownGaps |>.mapError .admission
  pure ⟨target, admitted⟩

/-- A one-instance `query` block's admission: an admission whose search selects nothing is
compared too. -/
def admitSource (source : QuerySource model) :
    Except AdmissionError ((target : QueryModel model.lawStatement) × AdmittedQuery target) :=
  admitQuery model source.key source.property (some source.behavior) source.form source.limits
    source.knownGaps

private def admissionFailure : AdmissionError → String
  | .invalidTarget _ => "the Model"
  | .invalidVocabulary _ => "the vocabulary"
  | .admission diagnostic => "admission at " ++ reprStr diagnostic.located.stage
  | .notSelected outcome _ _ => "not selected: " ++ outcome.name
  | .instances _ => "the instances"

/-- A one-instance `query` block's line. Its own admission ran `AdmittedQuery.search`; the
re-admission here must search to the same run. -/
def sourceLine (declared : Except AdmissionError (Umpire.Command.CheckedModel model))
    (source : QuerySource model) :
    String :=
  match admitSource source with
  | .error error => "not admitted: " ++ admissionFailure error
  | .ok ⟨_, admitted⟩ =>
      let rerun := match declared with
        | .ok checked => admitted.search.toOption == some checked.run
        | .error _ => true
      if !rerun then "DIFFERS the re-admission does not search to the declared run"
      else admittedLine admitted ++ "; " ++ agreement admitted.query

/-- A `query` block over several instances, from the arguments its declaration applies
`checkInstances` to. Its search ran on the product's admission, which `checkInstances` does not
keep; that admission is made again here, so a Query whose search selected nothing is compared too.
A Query `checkInstances` rejects before searching is listed with the reason. -/
def instancesLine (model : DeclaredModel Setup State Action Outcome Fact) (count : Nat)
    (queryKey : String) (limits : Limits) (propertyNames : PropertyNames)
    (scenarioNames : ScenarioNames) (knownGaps : List KnownGap := [])
    (form : QueryFormKind := .selectWitness) : String :=
  let declared := checkInstances model count queryKey limits propertyNames scenarioNames knownGaps
    form
  let searched := match declared with
    | .ok _ | .error (.notSelected _ _ _) => true
    | .error _ => false
  match declared, searched with
  | .error error, false => "not admitted: " ++ admissionFailure error
  | _, _ =>
      match admitQuery (model.instances count) queryKey
          (liftedProperty model count · propertyNames)
          (some (liftedScenario model count · scenarioNames)) form limits knownGaps with
      | .error error => "not admitted: " ++ admissionFailure error
      | .ok ⟨_, admitted⟩ =>
          let rerun := match declared with
            | .ok checked => admitted.search.toOption == some checked.run
            | .error _ => true
          if !rerun then "DIFFERS the re-admission does not search to the declared run"
          else admittedLine admitted ++ "; " ++ agreement admitted.query

end Declared

open Lean Elab Command in
/-- Every `query` declaration the environment records under `namespaces`, by name: one info line
each, from `sourceLine` for a one-instance Query and `instancesLine` for one over several
instances, which is applied to the very arguments the declaration applies `checkInstances` to. -/
def sweep (namespaces : List Name) : CommandElabM Unit := do
  let environment ← getEnv
  let declared := (Registry.queries environment).toList.map (·.declName) |>.filter fun name =>
    namespaces.any (·.isPrefixOf name) && environment.contains name
  for name in declared.mergeSort (fun left right => left.toString ≤ right.toString) do
    let label := Lean.quote (name.toString ++ ": ")
    let sourceName := name ++ `source
    if environment.contains sourceName then
      elabCommand (← `(#eval IO.println ($label ++
        Umpire.SearchTests.Differential.sourceLine $(mkIdent name) $(mkIdent sourceName))))
    else
      let some value := (environment.find? name).bind (·.value?)
        | throwError "{name} has no value"
      let value := value.consumeMData
      unless value.getAppFn.isConstOf ``Umpire.Command.checkInstances do
        throwError "{name} is neither a one-instance Query nor an application of checkInstances"
      let lineName := (← getCurrNamespace) ++ `differential ++ name
      liftCoreM <| addAndCompile <| .defnDecl {
        name := lineName
        levelParams := []
        type := mkConst ``String
        value := mkAppN (mkConst ``Umpire.SearchTests.Differential.instancesLine
          value.getAppFn.constLevels!) value.getAppArgs
        hints := .opaque
        safety := .safe }
      elabCommand (← `(#eval IO.println ($label ++ $(mkIdent lineName))))

/-! ### The Switch

Every `query` block of the Umpire Models this module imports -- the Switch's and the Replay tests'
-- and the three Queries admitted against the Switch. Their Scenarios pin their schedules, so each
Query is on `veil`. -/

/--
info: Umpire.Examples.Switch.exactAction: veil default, found 2 paths, found 2 states; automaton ok, monitors ok (Query depth 1, clause table depth 1)
---
info: Umpire.ReplayTests.softAfterHard: veil default, found 3 paths, found 3 states; automaton ok, monitors ok (Query depth 3, clause table depth 3)
-/
#guard_msgs in
run_cmd sweep [`Umpire]

open Umpire.Examples.Switch in
/--
info: ["veil default, found 2 paths, found 2 states", "veil default, found 2 paths, found 2 states",
  "veil default, found 2 paths, found 2 states"]
-/
#guard_msgs in
#eval [admittedLine exactActionAdmitted,
  admittedLine (exactActionAdmitted.withQuery exploratoryQuery exploratoryQuery_target),
  admittedLine (exactActionAdmitted.withQuery exactTraceQuery exactTraceQuery_target)]

/-! The Queries the Switch's other search callers derive from it: every point of the checked
variation space the Variations compiler plans (`searchWithIntent` over `withQuery`), and the
unsatisfiable-Scenario Query Promotion's tests search through `withQuery`. A derived Query shares
the Switch's search view, so each runs through `Search.Selection.searchWith` over the view
`Search.admit` builds, which is what `AdmittedQuery.searchWith` runs. -/

/-- Both backends on a checked Query through the view `Search.admit` would build for it. -/
def queryLine {LawStatement : Law → Prop} (query : CheckedQuery LawStatement) : String :=
  match SearchView.ofCheckedQuery query.target.id query with
  | .error _ => "no search view"
  | .ok view =>
      line query.form (Search.Selection.searchWith .reference query view)
        (Search.Selection.searchWith .veil query view)

private def spaceAssignments : List CheckedVariationAxis → List (List ModelValue)
  | [] => [[]]
  | axis :: rest => axis.choices.flatMap fun choice =>
      (spaceAssignments rest).map ({ definitionId := axis.id, value := choice.id.value } :: ·)

/--
info: ["veil default, found 2 paths, found 2 states", "veil default, found 2 paths, found 2 states",
  "veil default, found 2 paths, found 2 states", "veil default, found 2 paths, found 2 states"]
-/
#guard_msgs in
#eval (spaceAssignments VariationsTests.checked.axes).map fun assignment =>
  match lowerSpacePoint VariationsTests.checked assignment with
  | .ok point => queryLine point.query
  | .error error => "not lowered: " ++ error.kind.name

open Umpire.Examples.Switch in
/-- info: "veil default, unsatisfiable 0 paths, unsatisfiable 0 states" -/
#guard_msgs in
#eval admittedLine (exactActionAdmitted.withQuery
  { exactActionQuery with behavior := { exactActionBehavior with spaceStatus := .unsatisfiable } }
  exactActionQuery_target)

/-! The comparison is not vacuous: a result set against another Query's differs in its outcome, a
Plan with other `explored` counts differs from the same Plan in nothing else, and a receipt differs
in exactly the fields a changed backend writes. -/
#guard (Examples.Switch.exactActionRunResult.toOption.map fun run =>
    difference (.find Examples.Switch.flipProperty) (.ok run) (.ok { run with
      instrumentation := { run.instrumentation with
        searchBackend := .veil "commit", backendReason := .unsupportedStrategy .seeded
        searchUnit := .states, enumeratorPulls := 99 } })) == some none

#guard (Examples.Switch.artifact.map fun plan =>
    (canonicalPlanBytes (withExplored plan { traces := 99 }) != canonicalPlanBytes plan,
      canonicalPlanBytes (withExplored (withExplored plan { traces := 99 }) plan.plan.explored) ==
        canonicalPlanBytes plan)) == some (true, true)

#guard (do
    let found ← Examples.Switch.exactActionRunResult.toOption
    let other ← (run 0 (.find property) .exhaustive).toOption
    difference (.find property) (.ok found) (.ok other)).isSome

/-! ### Witness order (R15)

`a` and `b` both lead from `idle` to `mid`, and `c` from `mid` to `done`. A Query requiring `c`
stops at `done`, and the two paths there reach one product state at depth one. The reference
enumerates `a` first and stops on `a · c`; `veil` keeps the first-discovered parent, `a`, drops the
state `b` reaches again, and reports the same witness, the lexicographically least in the key
order. -/

private def sameDepthTarget : Option (QueryModel (fun _ => True)) :=
  tableTarget "same-depth" sameDepthTable

private def sameDepthAction (key : String) : DefinitionId :=
  id ("planner.same-depth.action." ++ key)

private def requiresC : CheckedScenario := {
  behavior with
  roles := []
  allowedActions := []
  requiredOccurrences :=
    [{ id := id "planner.same-depth.occurrence.c", action := sameDepthAction "c" }]
  actionsExactly := none
}

/-- The witness a Query over the same-depth fixture reports on each backend, with the product
states `veil` visited. -/
private def sameDepthWitnesses (form : Query.Form) (strategy : SearchStrategy) :
    Option (List String × Nat × Nat) := do
  let model ← sameDepthTarget
  let ⟨query, view⟩ ←
    tableQuery model "same-depth" form requiresC (Limits.bounded 2 2 100) strategy
  let reference ← (Search.Selection.searchWith .reference query view).toOption
  let veil ← (Search.Selection.searchWith .veil query view).toOption
  guard (veil.instrumentation.searchBackend != .reference)
  guard (difference form (.ok reference) (.ok veil)).isNone
  let witness := fun (run : PlanResult) => match run.result.outcome with
    | .found trace _ => trace.trace.steps.map (·.selectedAction.definitionId.value)
    | _ => []
  guard (witness reference == witness veil)
  pure (witness veil, reference.result.metadata.explored.traces,
    veil.result.metadata.explored.traces)

#guard [Query.Form.find property, .pick [property]].flatMap (fun form =>
    [SearchStrategy.shortest, .exhaustive].map (sameDepthWitnesses form)) ==
  List.replicate 4 (some (["planner.same-depth.action.a", "planner.same-depth.action.c"], 4, 3))

/-! ### Three instances past the reference's bound (R9)

Three instances of a two-state machine have eight states, and from each the three flips. Within
ten steps there are more than 32768 paths, so the reference reaches its search bound; `veil` visits
the eight states and completes. -/

private def threeInstances : Option ((query : CheckedQuery (fun _ => True)) ×
    SearchView query.target) := do
  let model ← tableTarget "three-instances" threeInstanceTable
  tableQuery model "three-instances" (.verify property)
    { behavior with
      roles := [], allowedActions := [], requiredOccurrences := [], actionsExactly := none }
    (Limits.bounded 10 10 32768) .exhaustive

/-- Each backend on the three-instance fixture. -/
private def threeInstanceRuns : Option (String × Nat × String × Nat × Option String) := do
  let ⟨query, view⟩ ← threeInstances
  compared query.form (Search.Selection.searchWith .reference query view)
    (Search.Selection.searchWith .veil query view)

/-- info: some ("limit-reached", 32768, "verified-within-limits", 8, none) -/
#guard_msgs in
#eval threeInstanceRuns

end Umpire.SearchTests.Differential
