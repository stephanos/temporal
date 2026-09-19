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
- [x] the Caller Model file compiles with `nexusProduct`, `nexusProtocol refines: nexusProduct`, the functional set with `repeat` and Queries 1 to 4; every Lean block later quoted by AUTHORING.md is marked
- [x] four fixtures regenerate through `umpire-case`; `async-nexus-case.json` is gone and Query 2's fixture carries the same Program and Contract modulo the diff listed in the receipt
- [x] four live tests pass under both switch values; `COVERAGE.md` maps every assertion of the four upstream tests
- [x] DESIGN.md section 3 equals the Model's declarations (no cancel rows); `make umpire-check-regression` exit 0 with the identity count recorded
## Done summary

### The Model

`model/Temporal/Feature/Nexus/Caller/Model.lean` (family `temporal.nexus.caller`) promotes the two
test-specimen machines into a Model: `nexusProduct` (the six phases and the `timeout` timer) and
`nexusProtocol refines: nexusProduct map: productOf` (192 states, 1152 rows, timers `backoff`,
`scheduleToClose`, `scheduleToStart`, `startToClose`), the vocabulary they share (`operation`,
`Phase`, `Fact`, `schedule`, `handlerReply`, `complete`, `attempt`, `timeout`), six predicate
Properties (`terminalIsFinal`, `syncSucceeds`, `asyncStarts`, `completionSucceeds`,
`completionFails`, `handlerErrorFails`), four Scenarios (`syncReplied`, `asyncThenSucceeded`,
`asyncThenFailed`, `nonRetryableError`), the limits `two` and `three`, Queries 1 to 4
(`syncCompletion`, `asyncCompletion`, `asyncFailure`, `handlerError`, each `find:` a Property of
the protocol machine in its Scenario) and the verifying `terminalHolds`, and the functional set
`nexusCallerTests` (caller, handler and worker driven, network observed, `repeat: implementation`
over the HSM and CHASM switch). `nexusProtocol` carries no `setup:` parameter: `atConcurrencyLimit`
has no dynamic-config key, so it is not an input a Case can bind, and the concurrency-limit rows are
authored as the `attempt` step's own branch instead (the DESIGN.md `.10` amendment names the two
keys and the three reasons). Every block AUTHORING.md will quote is bracketed by
`-- authoring: <name>` … `-- authoring: end` markers. `Caller/Tests.lean` carries the product and
protocol pins formerly in `Tests/Machines.lean` (158 reachable protocol states, the refinement row
lookups, stutter-invariance), the four Query outcome pins, the `#guard_msgs` for a Query that times
out, and the case pins (instruction ids, declared kinds, identities, no known gaps).
`Tests/Machines.lean` keeps the product machine as the specimen the `machine` command's rejections
are pinned against and points at the Model for the rest; the `Success` Model loses its `case asyncNexusSuccess fixture "async-nexus"` block and keeps
`nexusSuccessSet` for .11 to retire. `COVERAGE.md` maps every assertion of the four upstream tests
(`tests/nexus_workflow_test.go:512,1024,1617,2242`) to a Property, an observation of the Run or a
Known Gap; its Known Gaps table (`result-value`, `handler-links`, `callback-links`,
`mutable-state`, `completion-authorization`, `duplicate-completion`, `reset`) names what an
upstream assertion reads that is not an observation of this Model and where each would close, none
authored as a `gap:` line, for .13's closure to decide.

### Realizing the set

`case nexusCallerCases realizes nexusCallerTests as nexusOperation service "umpire.case.service"
operation "complete" realized by asyncNexus` is the new `caseTemplate` arm (`Temporal/Case/Syntax.lean`):
it names a realization rather than a template, and writes no `evidence` lines because the set's
machine declares its evidence (`evidence:` on `nexusProtocol`): the elaborator quotes the machine's
catalog and `produceCase (evidenceCatalog := …)` derives the mapping per Query from the facts each
path records (`Producer.derivedEvidence`: a fact whose value is the spelling, or starts with the
spelling and a hyphen, maps the Action that recorded it to the kind). An `evidence` block stays
optional for a machine without a catalog. Bindings are keyed: `ActionBinding.key` names the member
key a Scenario's classed action resolves to (`handlerReply-async`, `schedule-unset-unset-unset`,
`complete-succeeded`, …) through `vocabulary.namedAction`, with the stated `action` id as the
fallback the Success slice's `asyncPath` still uses; `EntrypointItem.whenOnPath keys node` places
the controller's `await-completion-authority` only on the paths that complete asynchronously.
`Realization/Nexus.lean` carries all eight bindings keyed (the two sync replies, the async reply,
the four completions, the handler error with its type and retry behavior) and the plan: controller
`start-workflow`, `await-completion-authority` when on a completion path, the completion actions,
a fixed `await-close` (a close-event history read) and `history`; workflow `schedule`,
`await-nexus-operation`, `finish-workflow`; handler the reply actions. Its sources are
`Template.NexusOperation.historySources` (scheduled keyed by `event_id`, started, completed,
failed, canceled, timedOut) plus `pendingAttemptsSource`, and its producer id is
`temporal.nexus.caller.testpilot`.

### Query 2 against the async-Nexus fixture

`nexusCallerTests-asyncCompletion-case.json` replaces `async-nexus-case.json`. Program: the same
three entrypoints and instruction ids, plus the controller's `await-close` node (a
`GetWorkflowExecutionHistory` read with the close-event filter, `wait_new_event`) and the workflow's
`finish-workflow` carrying the literal `"done"` and a `boolean true` guard where the old fixture
finished on the awaited value; three evidence declarations (`scheduled`, `started`, `completed`,
history sources scoped to the Run and keyed by `scheduled_event_id`, `scheduled` by `event_id`)
where the old carried two (`evidence.started`, `evidence.completed`). Contract: one correlated
rule `fact-nexusOperationCompleted` (the Property fixes the recorded fact) where the old carried
two (`outcome-completed`, `state-succeeded`); projection rules over the three declared kinds; five
transitions where the old carried two, because the machine reaches the completed phase through the
scheduled and started phases that the Success slice's `lifecycle` folded into its start state; a
new projection fingerprint. Provenance: producer `temporal.nexus.caller.testpilot`, no known gaps
where the old carried the two Success-slice gaps, definitions naming
`temporal.nexus.caller.target.nexusProtocol`, `.behavior.asyncThenSucceeded`,
`.query.asyncCompletion` and `.property.completionSucceeds`. The two-environment byte-identity
assertion is kept on the new fixture, under each switch value.

### The projection table a Contract carries

`Umpire.Case.Projection.check` now emits only the rows reachable from the initial state under the
actions the rules confirm or submit (`reachableStates`, fuel the state count) rather than the whole
table: the protocol machine's 1152 rows made a 2.6 MB fixture that tripped `MaxTransitions 64` in
the `umpire-run` live test, and a Contract needs only the transitions its rules can take. The
caller fixtures are 26 to 36 KB with two to five transitions; `Correlated.lean`'s `maximumFacts`
check became `≤` (message "portable work maximum fact count exceeds the checked description") for
the same reason. `typed-nexus-case.json`, `nexusSuccessTests-completion-case.json` and the
conformance `correlated.json` regenerated.

### The Driver

The handler-error Query is the first live Run through a `NexusHandlerReply` error: the worker's
Nexus start interceptor finished the handler activation with the instructed error as an activation
failure, so the scheduler stopped the Run as `effect_wait_failed` and the Run closed incomplete
before the caller's `nexus_operation_failed` event was read. `nexusResult.replied` now marks a reply
the entrypoint instructed (a typed handler error or failed start, or the untyped error kind) and
`Session.nexusActivationOutcome` finishes that activation as succeeded while any other failed start
still fails it; `TestSessionAnswersTypedReplies` pins both, and the worker README says so.

### Live and gates

`tests/testpilot_nexus_caller_case_test.go` runs one function per Query under each switch value
(`nexusCallerQueries`: the fixture, the supporting history events in order, the terminal event),
two bindings and two concurrent Runs per value, the switch-agreement check across values, and
`requireCorrelatedNexusHistoryEvidence` on every Run; `TestTestpilotNexusCallerCaseMissingRemoteEndpoint`
and `TestTestpilotNexusCallerCaseRunsFromItsFixtureNameAlone` carry over from the async-Nexus test,
and the `umpire-run` and worker-outage peer tests point at Query 2's fixture. `make
umpire-check-live-tests`: failure identities match the empty expected set across **20 passing
identities** (eleven before: the four Queries under two values replace the async-Nexus Case under
two). `make umpire-check-regression` exit 0. `LEAN_NUM_THREADS=1 make lint-model` reports the
baseline warnings plus two unused-binder warnings in `Caller/Model.lean` from the `enum` command's
generated binders (`retryable`, `timeoutType`), the pattern `Tests/Commands.lean` already carries;
`lint-code-fast` 0 issues.

## Evidence
- Commits: 2331e682a69032fdd490672e0253a4d08f72c8d8
- Tests: `lake build`; `make umpire-gen-case-runtime-conformance`; `make umpire-check-testpilot-protocol`; `make umpire-check-testpilot-authoring`; `make umpire-check-case-runtime-conformance`; `make umpire-check-goldens`; `make umpire-gen-inventory && make umpire-check-inventory`; `make umpire-check-retired-vocabulary`; `LEAN_NUM_THREADS=1 make lint-model`; `GOLANGCI_LINT_BASE_REV=1e6f9f6 make lint-code-fast`; `go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/...`; `go vet -tags 'test_dep integration' ./tests/`; `CC=/usr/bin/cc TMPDIR=$(cd /tmp && pwd -P) make umpire-check-live-tests` (20 passing identities); `make umpire-check-regression` (exit 0)
- PRs:
