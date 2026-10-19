---
satisfies: [R4, R5]
---
# fn-216-mini-activity-vertical-slice-the-model.4 Slice findings, generic boundary gate and final integration evidence

## Description
Close R4/R5 together after the real slice. Reconcile documentation with observed results,
enforce the ownership boundary and verify the complete clean-generation/gate path.

**Size:** M
**Files:** reusable boundary scanner, system policy/negative controls, final gates and current docs.
**Touches:** [model/temporal/features/activity_mini/temporal.rs, model/temporal/realization/**, model/temporal/guest/src/lib.rs, model/temporal/api/build.rs, model/umpire/dsl/macros/src/machine.rs, model/umpire/wasm/src/**, model/umpire/testing/**, model/umpire/checker/src/report.rs, model/umpire/tests/fixtures/kettle_progress/model.rs, model/temporal/testing/**, model/scripts/**, Makefile, model/DESIGN.md, model/README.md, model/umpire/proto/README.md, model/umpire/hosts/go/README.md, MIGRATE.md]

### Approach
- Implement a reusable lexical scan in umpire-testing and invoke the parent contract's deny policy from outside the generic tree. Keep forbidden system literals and negative fixtures outside model/umpire so the gate has no self-exclusion. Scan every promised extension and report each offending relative file/line deterministically.
- Selftests must cover temporal/Temporal identifiers, forbidden API prefixes in source/import/module positions, all four extensions, multiple offenses, exact line numbers, allowed temporalio organization segments and near-miss paths that must still fail. Wire the check into make model-check.
- The current generic report comments and supplementary Kettle fixture's error message use the word temporal for progress classification. Rename those occurrences to progress, preserving behavior and rendered outputs. Do not broadly exempt files. Ask root for exact scope expansion if a later source introduces another occurrence outside declared files.
- Reproduce generation with disposable output/target locations and the existing Make prerequisite, preserving the user's descriptor/cache files. Compile the real API closure and both guests; run pinned Go freshness, generic and extension module tests, old wire controls, full model gate and passing umpire-check-model-slice target. Attempt the unchanged-scope umbrella and compare with .3's pre-change legacy baseline. Capture all raw failures rather than retrying them out of the evidence; do not call an unchanged legacy Case failure a pass or revive frozen Scala to remove it.
- Write DESIGN's Mini activity slice section from .1/.3 evidence, including closure-proof limits, input/realization projections, DSL gaps, protocol findings, Answer latency and untouched R33-R40 obligations. Remove the obsolete Testpilot-schema-copy description that the owner superseded.
- Update README/protocol/host reproduction and MIGRATE ownership. Preserve the extraction destination/history plan and local Cargo/Go references, state that Temporal remains in the server repo, and keep extraction/publication unexecuted.
- Preserve the new fixture root for fn-213 and fn-218's eight-case coverage obligation. Record impossible alternatives as a downstream grammar decision, never manufacture Model transitions or silently reduce Cases(8).
- Root alone updates MILESTONES. Report completed task status to the conductor rather than editing it here.

### Investigation targets
**Required:**
- `model/umpire/testing/src/lib.rs` - reusable test tooling surface.
- `model/umpire/testing/src/runner.rs` - deterministic fixture/report conventions.
- `Makefile:1424` - model gate entry.
- `model/README.md:36` - old schema/guest description.
- `model/DESIGN.md:394` - superseded schema-copy guidance.
- `MIGRATE.md:31` - extraction ownership and local references.

### Quick commands
Run the scanner's negative/selftest suite, generation freshness and both Go-module tests. Run make model-check, make model-time and make umpire-check-model-slice with the built Temporal guest. Attempt umpire-check-live-tests and report its baseline comparison separately. Preserve exact commands and output locations in evidence.

### Realization cleanup (owner, 2026-10-19)
`features/activity_mini/temporal.rs` is 326 lines; the owner asked that it become shorter and clearer in this task, after the gates above. The behavior pinned by `realization/tests/realization.rs`, the Go host parity test and the real SDK slice test must not change; refusal codes, reply bytes and the report/rows goldens stay identical.
- `#[machine]` emits `pub struct World;` and `pub fn world() -> World` instead of a private `World` behind `impl Machine<...>` (`model/umpire/dsl/macros/src/machine.rs:636`, snapshot update). The hand-written `Action` carrier, its three `From` impls, `Spell`, and the delegating `World` (including the `claims[0]` postcondition re-wrap) are deleted; the realization is `impl Realize<activity::World> for Realization` with no generic bounds, `runtime()` is `Runtime::new(activity::World, Realization)`, and the guest and crate facade name `activity::World`.
- The api build script enables prost type names with the `type.googleapis.com` domain so `unpack::<T>` and `pack` derive URLs from `prost::Name`; only the Describe RPC method path remains a constant.
- Generic Any and refusal plumbing (`unpack`, `pack`, `required`, `malformed`, `unsupported`) moves into `umpire_wasm` as constructors on `protocol::Refusal` and helpers on `Any`, and `runtime.rs` uses the same ones instead of its own `required`/`failure`. Nothing Temporal enters `model/umpire`; the boundary gate still passes.
- `project` becomes one `match (status, run_state)` table with the same refusals.
- `answer` splits into a `Policy` parse (payload cardinality, metadata, encoding, bytes) and a reply constructor holding the two prost literals; the explicit Scheduled precheck is dropped because the runtime already refuses a disabled Poll with the `State` code.
- Target is roughly 150 lines with every remaining line Temporal-specific. Update `realization/README.md` (no delegating world), DESIGN's DSL-gaps paragraph and the macro expansion snapshot accordingly.
## Approach
- Implement a reusable lexical scan in umpire-testing and invoke the parent contract's deny policy from outside the generic tree. Keep forbidden system literals and negative fixtures outside model/umpire so the gate has no self-exclusion. Scan every promised extension and report each offending relative file/line deterministically.
- Selftests must cover temporal/Temporal identifiers, forbidden API prefixes in source/import/module positions, all four extensions, multiple offenses, exact line numbers, allowed temporalio organization segments and near-miss paths that must still fail. Wire the check into make model-check.
- The current generic report comments and supplementary Kettle fixture's error message use the word temporal for progress classification. Rename those occurrences to progress, preserving behavior and rendered outputs. Do not broadly exempt files. Ask root for exact scope expansion if a later source introduces another occurrence outside declared files.
- Reproduce generation with disposable output/target locations and the existing Make prerequisite, preserving the user's descriptor/cache files. Compile the real API closure and both guests; run pinned Go freshness, generic and extension module tests, old wire controls, full model gate and passing umpire-check-model-slice target. Attempt the unchanged-scope umbrella and compare with .3's pre-change legacy baseline. Capture all raw failures rather than retrying them out of the evidence; do not call an unchanged legacy Case failure a pass or revive frozen Scala to remove it.
- Write DESIGN's Mini activity slice section from .1/.3 evidence, including closure-proof limits, input/realization projections, DSL gaps, protocol findings, Answer latency and untouched R33-R40 obligations. Remove the obsolete Testpilot-schema-copy description that the owner superseded.
- Update README/protocol/host reproduction and MIGRATE ownership. Preserve the extraction destination/history plan and local Cargo/Go references, state that Temporal remains in the server repo, and keep extraction/publication unexecuted.
- Preserve the new fixture root for fn-213 and fn-218's eight-case coverage obligation. Record impossible alternatives as a downstream grammar decision, never manufacture Model transitions or silently reduce Cases(8).
- Root alone updates MILESTONES. Report completed task status to the conductor rather than editing it here.

## Investigation targets
**Required:**
- `model/umpire/testing/src/lib.rs` - reusable test tooling surface.
- `model/umpire/testing/src/runner.rs` - deterministic fixture/report conventions.
- `Makefile:1424` - model gate entry.
- `model/README.md:36` - old schema/guest description.
- `model/DESIGN.md:394` - superseded schema-copy guidance.
- `MIGRATE.md:31` - extraction ownership and local references.

## Quick commands
Run the scanner's negative/selftest suite, generation freshness and both Go-module tests. Run make model-check, make model-time and make umpire-check-model-slice with the built Temporal guest. Attempt umpire-check-live-tests and report its baseline comparison separately. Preserve exact commands and output locations in evidence.

## Acceptance
- [ ] The generic boundary gate runs in make model-check and all promised positive/negative cases pin accurate deterministic file/line diagnostics without exclusions.
- [ ] Clean generation, the real native/WASM closure, both guests, old wire/host parity and dedicated real SDK gate pass. The umbrella invokes that gate without dropping legacy checks; compare legacy failures with their pre-change baseline and block introduced regressions. Timing follows the 10 s / 30 s soft policy and live latency is reported separately.
- [ ] DESIGN/README/MIGRATE match the implemented boundaries and measured evidence, explicitly name DSL/projection limits and retain all untouched R33-R40 and downstream fixture/coverage obligations.
- [ ] No Model change was made to fit runtime evidence, no Testpilot dependency boundary was weakened, and no remote extraction/publication occurred.
- [ ] `temporal.rs` uses the macro-exported `activity::World` directly with no delegating machine, derives Any type URLs from `prost::Name`, and keeps only Temporal-specific code; the realization, parity and real SDK tests, the goldens and the boundary gate pass unchanged.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

