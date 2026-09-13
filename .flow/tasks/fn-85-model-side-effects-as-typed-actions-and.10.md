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
- The Model file uses only the commands (`enum`, `entity`, `action`, `observation`, `machine`, `property`, `scenario`, `limits`, `query`, `set`); mark the regions task .13's AUTHORING.md will quote with the drift markers now.
- Query 2 replaces the async-Nexus fixture; list the Program and Contract diff against the old fixture in the receipt (the spec requires it) and keep the two-environment byte-identity assertion on the new fixture.
- The existing `model/Temporal/Feature/Nexus/COVERAGE.md` is an unrelated evidence record fn-86 retires; do not overwrite it.
- `TestNexusOperationAsyncFailure` and `TestNexusSyncOperationErrorRehydration` assertions on error details map to Properties over the failure observation's fields where the schema exposes them, otherwise to Known Gaps named in `COVERAGE.md`.

### Investigation targets
**Required:**
- `model/Temporal/Feature/Nexus/DESIGN.md` section 3 — the specimen (cancel rows excluded)
- `tests/nexus_workflow_test.go:512,1024,1617,2242` — the four upstream tests
- `model/Temporal/Feature/Nexus/Success/Model.lean` — the current command Model to retire in task .11
- `tests/testpilot_async_nexus_case_test.go:33-152` — the live test shape to carry over
- `model/Temporal/Feature/Nexus/Success/Tests.lean` — specimen style for the Model's own tests

**Optional:**
- `tests/testcore/testpilot/README.md:13-17` — the async fixture paragraph to rewrite

### Key context
- Nine live identities is the baseline; this task changes the count (four Queries times two switch values replace one); record the new number for the order document.

- 2026-09-12: machines are step functions (see the spec's Planning decisions); rewrite DESIGN.md section 3 in that form, with the `steps:` rows of the specimen expressed as `match` arms and the evidence lines keyed by action class and outcome.
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
