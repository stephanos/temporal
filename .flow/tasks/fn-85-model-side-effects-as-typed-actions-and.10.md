---
satisfies: [R11]
---
# fn-85-model-side-effects-as-typed-actions-and.10 The Nexus Model, part one: product and protocol machines, Queries 1 to 4, COVERAGE.md

## Description
Re-author the Nexus caller-side operation as the product machine and the protocol machine that refines it, in a new Model directory, without the cancel actions and rows (R11): the functional set with `repeat` over the HSM and CHASM switch and Queries 1 to 4 (sync success; async reply then succeeded callback, which replaces the async-Nexus fixture; async reply then failed callback; non-retryable handler error), each with a generated fixture and a passing live test under both switch values. Start `COVERAGE.md` beside the Model mapping every assertion of the four upstream tests to a Property, an observation or a Known Gap. Rewrite DESIGN.md section 3 to the cancel-free specimen so the design and the Model agree.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Caller/Model.lean` (new; final path a task decision, beside `Success/`), `model/Temporal/Feature/Nexus/Caller/COVERAGE.md` (new), `model/Temporal/Feature/Nexus/Caller/Tests.lean` (new), `model/Temporal/Case/Realization/Nexus.lean`, `tests/testcore/testpilot/testdata/<set>-<query>-case.json` (four new fixtures; `async-nexus-case.json` removed), `tests/testpilot_nexus_caller_case_test.go` (new; one function per Query, both switch values), `tests/testpilot_async_nexus_case_test.go` (removed or re-pointed at Query 2), `tests/testcore/testpilot/async_nexus_fixture.go` and `profile.go`, `model/Temporal/Feature/Nexus/DESIGN.md` (section 3), `model/Temporal.lean` or `TemporalModelTests.lean` (aggregator imports)
**Touches:** [model/Temporal/Feature/Nexus/Caller/**, model/Temporal/Feature/Nexus/DESIGN.md, model/Temporal/Case/**, model/Temporal.lean, model/TemporalModelTests.lean, tests/testcore/testpilot/**, tests/testpilot_*_test.go]

### Approach
- The protocol-migration oracle is retired in `.1`, so no declared mapping step is needed here; the
  conformance `expected.json` pins are the Verdict net.
- The Model file uses only the commands (`enum`, `entity`, `action`, `observation`, `machine`, `property`, `scenario`, `limits`, `query`, `set`); mark the regions task .13's AUTHORING.md will quote with the drift markers now.
- Query 2 replaces the async-Nexus fixture; list the Program and Contract diff against the old fixture in the receipt (the spec requires it) and keep the two-environment byte-identity assertion on the new fixture.
- The existing `model/Temporal/Feature/Nexus/COVERAGE.md` is an unrelated evidence record fn-86 retires; do not overwrite it.
- `TestNexusOperationAsyncFailure` and `TestNexusSyncOperationErrorRehydration` assertions on error details map to Properties over the failure observation's fields where the schema exposes them, otherwise to Known Gaps named in `COVERAGE.md`.
- Adjusted 2026-09-19 after .14, .16, .15, .4, .6, .5 and .7 landed. The two machines already
  elaborate as test specimens: `Temporal.Feature.Nexus.Tests.Machines.nexusProduct`
  (`Tests/Machines.lean:105-123`, with the `timeout` timer .6 added out of `scheduled` and
  `started`) and `nexusProtocol refines: nexusProduct map: productOf` (`:343-372`; 192 states, 1152
  rows, `setup: atConcurrencyLimit: Bool`, timers `backoff, scheduleToClose, scheduleToStart,
  startToClose`; about forty seconds to elaborate, seventy-five with the refinement decided, one
  hundred fifty with two Queries) over the entities, enums and actions of `Tests/Commands.lean`.
  Promote them into the Caller Model rather than re-authoring from `DESIGN.md`, and move or
  re-point the pins those two test modules carry. The grammar as landed: `machine` takes `for:`,
  `state:`, `starts:`, `ends:` (required since .16; a step out of an end state is admitted since
  .6), `timers:`, `unobservable:` (timers only), `setup:`, `evidence:`, `refines:`, `map:` (a Lean
  function) and `steps:` (one function per action); `property` takes `machine:`, an optional
  `when:` (bare or classed, `handlerReply (handlerError true)`) and `holds:` with a `Step → Bool`
  or `Step → Step → Bool` predicate (`require:` and `model:` are rejected at the key, .15);
  `scenario` takes `model:`, `starts:` naming the start phase, `actions:` with classed actions
  (`complete (succeeded)`) and an optional `instances:`; `query` accepts a Property declared on
  the machine the Scenario's machine refines and rejects at `find:` a name the refining machine
  lacks (.6); `set` takes `purpose:`, `bind:`, `repeat: implementation`, `queries:`; and the Temporal
  `case <name> realizes <set> as <template> evidence <lines>` block (`Temporal/Case/Syntax.lean:183-260`)
  produces `temporal.case.<set>.<query>` and `<set>-<query>-case.json` with one realization for
  all Queries and the evidence lines filtered per path (.7). That block's template arm is the
  template's (`nexusOperation service … operation … responds async`) and
  `Temporal.Case.Realization.asyncNexus` is built from the template, so this task adds the arm
  (or a realization reference) the Caller set needs, with the sync-reply, handler-error and
  failed-completion bindings .8 typed. `atConcurrencyLimit` has no dynamic-config key, so every
  Case over `nexusProtocol` carries the `input` Known Gap `…atConcurrencyLimit.unbound` (.5):
  record it in `COVERAGE.md` and keep it in view for .12's canary admission. The live harness
  already runs one Case once per switch value under a dedicated environment
  (`tests/testcore/testpilot/switch.go`: `NexusImplementationSwitch`, `CheckSwitchAgreement`;
  `tests/testpilot_async_nexus_case_test.go`), so one function per Query reuses it. `DESIGN.md`
  section 3 carries dated amendments from .2, .4, .6 and .15 (`:65,115,240,265,548`); rewriting it
  keeps them. The `Success` Model also carries `set nexusSuccessTests` and `case nexusSuccessSet`
  (fixture `nexusSuccessTests-completion-case.json`), which .11 deletes with the Success Model, so
  Query 2's fixture replaces both `async-nexus-case.json` and that one, and
  `tests/testcore/testpilot/README.md:39-46` describes set-derived fixtures already.

### Investigation targets
**Required:**
- `model/Temporal/Feature/Nexus/DESIGN.md` section 3 — the specimen (cancel rows excluded)
- `tests/nexus_workflow_test.go:512,1024,1617,2242` — the four upstream tests
- `model/Temporal/Feature/Nexus/Tests/Machines.lean:1-130,329-372,506-580` and `Tests/Commands.lean` — the product and protocol machines, the map, the product-Property-on-protocol Queries; the declarations this task promotes
- `model/Temporal/Feature/Nexus/Success/Model.lean` — the current command Model (`lifecycle`, `nexusSuccessTests`, `nexusSuccessSet`) to retire in task .11
- `model/Temporal/Case/Realization/Nexus.lean` and `model/Temporal/Case/Syntax.lean:183-260` — the realization and the set-realizing `case` block
- `tests/testpilot_async_nexus_case_test.go:33-152` and `tests/testcore/testpilot/switch.go` — the per-switch-value live test shape to carry over
- `model/Temporal/Feature/Nexus/Success/Tests.lean` — specimen style for the Model's own tests

**Optional:**
- `tests/testcore/testpilot/README.md:13-17` — the async fixture paragraph to rewrite

### Key context
- Eleven live identities is the baseline since .5 (the async-Nexus Case under `hsm` and `chasm` replaced one of the nine); this task changes the count (four Queries times two switch values replace those two); record the new number for the order document.

- 2026-09-12: machines are step functions (see the spec's Planning decisions); rewrite DESIGN.md section 3 in that form, with the `steps:` rows of the specimen expressed as `match` arms and the evidence lines keyed by action class and outcome.
- 2026-09-12: Properties are predicates by the same rule (task .3 delivers the command): write each Nexus Property as a `Step → Bool` or `Step → Step → Bool` function, not as keyed `require:` lines; COVERAGE.md names the predicate per upstream assertion.
## Acceptance
- [ ] the Caller Model file compiles with `nexusProduct`, `nexusProtocol refines: nexusProduct`, the functional set with `repeat` and Queries 1 to 4; every Lean block later quoted by AUTHORING.md is marked
- [ ] four fixtures regenerate through `umpire-case`; `async-nexus-case.json` is gone and Query 2's fixture carries the same Program and Contract modulo the diff listed in the receipt
- [ ] four live tests pass under both switch values; `COVERAGE.md` maps every assertion of the four upstream tests
- [ ] DESIGN.md section 3 equals the Model's declarations (no cancel rows); `make umpire-check-regression` exit 0 with the identity count recorded
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
