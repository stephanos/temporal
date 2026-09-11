---
satisfies: [R14]
---
# fn-83-author-a-live-case-from-a-model-file.14 Report authoring mistakes at their line while the Model file compiles

## Description
Report every authoring mistake in the Model file, at the line that makes it, while the file compiles (R14). Today only the `model` command checks while compiling. `property` and `scenario` turn member, role and Action names into strings, and `query` only defines `completion : Except AdmissionError … := Authoring.check …`, which nothing evaluates in the Model file. So a misspelled `require successState: state suceeded`, a `for worker` role, an Action order the model cannot take, or limits that are too small all leave `Model.lean` green. They surface later as a `#guard` failure in a test module or as a bare `checked-model` error from `umpire-case --render`, because `Authoring.produceCase` discards the underlying `AdmissionError`. The same strings are why those names get no hover and no go-to-definition in the editor.

**Size:** M
**Files:** the Umpire command module from .10 (`property`, `scenario` and `query` become elaborators that resolve names and evaluate admission; the `model` command records what they resolve against), the Scenario/Query registry from .10 (gains a Model entry), the Temporal `case` command from .10 and `Authoring.produceCase` (production failures keep their construct), `model/Temporal/Feature/Nexus/Success/Tests.lean` and `RaceSyntaxTests.lean` (`#guard_msgs` pins), `model/Umpire/ARCHITECTURE.md` if the elaboration contract changes
**Touches:** [model/Umpire/**, model/Temporal/Case/**, model/Temporal/Feature/Nexus/Success/**]

### Approach
**Resolve names while compiling (hover and go-to-definition follow)**
- The `model` command records, beside its declaration, its role spelling, start and end states, and the ordered constructor names of its State, Action, Outcome and Fact domains (a registry entry of names and plain data, never a closure, like the existing Scenario and Query entries).
- `property … on <model>` resolves `<model>` to a recorded Model (unknown → located error listing declared Models), `for <role>` against the Model's role, `when <action>` against its Actions, and each `require … : state|outcome|fact <member>` against the matching domain. `scenario … on <model> <role> starts <state>` resolves the role and a State that is one of the Model's `starts`; `actions exactly [label: action, …]` resolves each Action. Unknown spellings reject in place with the existing message shape: `unknown … '<spelling>'; declared: …`.
- Each resolved member reference records constant info on its original syntax (for example `Lean.Elab.addConstInfo` with the constructor name), so the editor shows hover and go-to-definition on `awaitSuccess`, `succeeded`, `scheduled` and on `on`/`find`/`in`/`limits` references. The emitted Umpire records keep the same spellings, so IDs and bytes do not move.
- `query` checks that its property and scenario name the same Model as the query (`on <model>`), in place.

**Evaluate admission while compiling**
- After emitting the `query` definition, evaluate it during elaboration and turn every `.error` into a located message. Precedent: `Umpire/Property/Elab.lean`, `Scenario/Elab.lean` and `Query/Elab.lean` evaluate authored values with `Lean.Meta.evalExpr` behind `@[implemented_by]` and report located errors; follow that shape rather than `native_decide`.
- Placement and wording per `AdmissionError` constructor: `invalidTarget` at the `on <model>` reference; `invalidProperty` / `invalidBehavior` at the property or scenario reference, naming the failing clause or constraint when the typed error carries it; `invalidQuery` at the query name with the `QueryErrorKind` name; `notSelected` at the `find`/`verify` keyword with outcome-specific text.
- `notSelected` wording must not mislead. A `find` whose search completed without a witness is reported by Search as `limitReached` because the `shortest` strategy never claims completeness (`Umpire/Search.lean:743-745`). Distinguish "no candidate within the limits satisfies the property" from "a bound stopped the search" using the run's explored counts against the declared limits, and only suggest raising a limit when a bound was actually hit. `unsatisfiable` names the scenario; `violatingCounterexample` prints the counterexample's Action sequence; `neverTriggered` names the `when` Action; `stillPending` names the open clause.
- The `case` command evaluates production the same way and reports the Producer's construct (for example `property.clause-early-response`, `witness.absent`, an unmapped hook) at the `case` block. `Authoring.produceCase` keeps the underlying admission error instead of replacing it with `checked-model`.
- Measure `lake build` time for the Nexus success and Race syntax modules before and after; record the delta. If evaluation cost is material, record it and propose a bound rather than silently skipping evaluation.
- Existing value-level tests that build misspelled records through `Authoring.authoredProperty` / `authoredScenario` directly stay; they pin the admission layer below the commands.

### Test cases (`#guard_msgs (error)`)
- misspelled `require` member (state, outcome, fact), unknown `when` Action, unknown `for` role
- scenario: unknown role, start state that is not a declared start, unknown Action in `actions exactly`
- property or scenario declared on a different Model than the query
- unsatisfiable scenario (reversed Action order)
- a bound that stops the search before a witness (limit actually hit)
- a `find` whose completed search finds no witness (no misleading limit advice)
- a `verify` counterexample
- one `case` production failure reported by construct at the `case` block
- a `#guard` or info-tree check that a resolved member reference carries constant info for its constructor

### Investigation targets
**Required:**
- the Umpire command module from .10 — `property`, `scenario`, `query`, `case` commands; `resolveMember`, `unknownMemberMessage`
- `model/Umpire/Property/Elab.lean` (and `Scenario/Elab.lean`, `Query/Elab.lean`) — `evalExpr` / `implemented_by` evaluation and located-error rendering
- `Authoring.check`, `AdmissionError`, `produceCase` (post-.10 location)
- `model/Umpire/Search.lean` — `PlanningOutcome`, `finalizePlanning`, explored counts in the planning metadata
- `model/Temporal/Case/Registry.lean` — the extension shape to extend with a Model entry
- `model/Temporal/Feature/Nexus/Success/Tests.lean:563-770` — current value-level misspelling guards and `#guard_msgs` conventions

### Key context
- Depends on .11 and .13 so the `query` gap lines and the optional-Facts grammar exist before diagnostics are written against them. .5 (fault lines on `scenario`), .6 (new Model) and .8 (the tutorial lists every located diagnostic) depend on this task.
- AUT-03 already asks for failures at the relevant source location; no spec amendment.

## Acceptance
- [ ] Every name in `property`, `scenario` and `query` resolves while the Model file compiles; an unknown Model, role, Action or member rejects at that name listing the declared spellings
- [ ] Resolved member and declaration references carry constant info, so hover and go-to-definition work on them in the editor, pinned by a test
- [ ] Compiling a Model file evaluates each `query`; every `AdmissionError` is reported as a located error at the reference that caused it, with outcome-specific text for `notSelected` that suggests raising a limit only when a bound was hit
- [ ] Compiling a `case` block evaluates production; a production failure is reported by construct at the block and `produceCase` no longer reduces it to `checked-model`
- [ ] All test cases listed in the description are pinned by `#guard_msgs`; the existing value-level guards still pass
- [ ] Every checked-in fixture, golden, Definition ID and fingerprint is byte-identical; the build-time delta for the Nexus success and Race syntax modules is recorded
- [ ] `cd model && lake build`, `make lint-model`, `make umpire-check-case-runtime-conformance` pass


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
