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
- DESIGN's DSL-gaps paragraph names the rejecting-outcome gap (a closed activity's cancel collapses to a disabled action because `#[machine]` fixes `Outcome = ()`) and points to fn-221, which closes it after this task.
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
Implemented R4/R5: a reusable lexical boundary scanner with system-owned policy/negative controls, direct macro-exported activity World, generated prost type URLs and shared Any/Refusal plumbing. Reconciled DESIGN/README/MIGRATE with the real SDK slice, extraction ownership, projection/DSL limits and retained downstream obligations. The binding is 171 lines versus 333; refusal codes, reply bytes, Model transitions and report/rows goldens remain unchanged.

baseline: green for model-check, model-time, dedicated slice and exact SDK command. The umbrella is separately inherited red: this task's before/after logs contain exactly the same seven failure identities and assertion causes as predecessor .3. Its final Go suite took 404.885s; no green umbrella receipt was written. Complete command/exit/log ledger: `.flow/tmp/fn-216-4/commands.md`; comparison: `.flow/tmp/fn-216-4/legacy-comparison.md`.

The scanner controls first failed against the no-op implementation, then the real scanner named precisely report.rs:60/:91 and kettle_progress/model.rs:57 before their authorized progress-wording cleanup. Its final two controls pin all four extensions, lowercase/uppercase identifier substrings, API source/import/module prefixes, deterministic multiple file/line diagnostics and the exact organization-path exception with near misses. `make model-check` invokes the controls and real-tree scan without source exclusions.

Clean verification used a disposable clone with no descriptor and a new Cargo target: pinned generation, real native/WASM API closure, stock/control/Temporal guests, old wire controls, both Go modules and freshness all passed. A final-source recheck passed too. Generated descriptor bytes differ only in Go-map file ordering: both independently generated sets equal the original's 57 files by name and message content. Original descriptor SHA remains d8b5530e9f75a9f624b6a7f6f43422f9ab3e379d3a4201b86491bdb9175b7fd3. Disposable checkout/target remain at `/tmp/umpire-fn2164-clean.OGxRrh` for conductor inspection; user's caches were preserved.

All six existing realization controls passed unchanged, including required malformed/unsupported inputs, callback policy, terminal ordering, impossible observations/freeze, Reset/isolation, publication rollback and report/rows parity. The owner-authorized test-only scope expansion replaces `assert!(actions == expected_actions)` with `assert_eq!` at activity/model.rs:166 because exposing Debug triggered Clippy; no Model rule changed. Macro snapshot changes deliberately pin exported World/Action and concrete public-state return.

The root correctness audit caught an introduced private-State E0446 before formal review. Generated World/Action now follow the state's visibility; non-public states retain the original opaque return. The same module-local control passes the original base macro, fails the initial checkpoint with E0446 and passes the fix. Logs are private-state-base.log, private-state-checkpoint-red.log and private-state-fixed.log. Earlier failed external-call harness attempts remain recorded, not labeled passing. Verbatim root quality reports and closure: `.flow/tmp/fn-216-4/quality-audits.md`.

Final Verify model-check and dedicated SDK slice pass; the final dedicated live run took 257.404 ms including cleanup. The measured warm gate was 11.856s full / 2.526s inner against 30s/10s soft targets with concurrent verification. Final umbrella Model runs took 502.668ms/296.593ms; Answer Complete/Fail/HostRefusal calls were 134.958/65.583/168.333µs and 103.834/57/32.958µs. Exact SDK command also passed. Timings exclude build time from live measurements and make no idle-host claim.

The repository-required default `make lint-code-fast` selected the historical broad Go diff and failed existing unrelated lint issues, auto-fixing 10 unrelated Go files. The complete patch/log is retained; every tool-authored Go edit was exactly restored (zero Go diff). Task-scoped non-fixing lint with GOLANGCI_LINT_BASE_REV=3cf4aeb4648959c3ccc43a935a389bd63ea42f4a passes with no changed Go packages. This broad lint observation remains red, distinct from the green Rust lint gate.

Three fresh read-only Codex contexts returned SHIP with zero findings at reviewed head 585c56e980301ad37bcbd610ee85219764d162a6. Backend and rollout turn_context evidence confirm gpt-6.1-sol at high; writer/reviewer share the Codex family. Fresh reviewer Cargo execution was blocked by read-only build locks; reviewers ran cached controls and inspected worker logs. Receipt: `/tmp/impl-review-receipt-8f37faba39e2-fn-216-mini-activity-vertical-slice-the-model.4.json`; execution/fanout evidence is linked from review-execution.json. The later 70de4ffc8b checkpoint carries only review lifecycle records and the conductor-requested DESIGN qualification, fn-221 ordering and final measured documentation; the conductor explicitly assigned those docs to whole-spec completion review.

stage: impl-review - ran [2026-10-19T20:35:08Z..2026-10-19T20:36:57Z] (model: gpt-6.1-sol at high)

Tier: session (jev-unavailable(no_key))

No remote extraction/publication, Testpilot dependency change, frozen Scala repair, new Model transition or coverage reduction occurred. R33's broader Case work and R34-R40 remain open; fn-213 must include the Temporal fixture root and fn-218 retains Cases(8). MILESTONES edits carried by task commits were root-authored; this worker did not edit it. No mapped user route changed.

Tracker sync: n/a (bridge inactive).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: bf5382deefe55d66047d2d9e5ea483200cd54a1e, 585c56e980301ad37bcbd610ee85219764d162a6, 70de4ffc8bb739c3d84da52b3b83a8836c32a609
- Tests: make model-check, make model-time, make umpire-check-model-slice, UMPIRE_TEMPORAL_WASM=/home/agent/.cache/codex-build/cargo-target/wasm32-wasip1/release/umpire_temporal_guest.wasm CGO_ENABLED=0 mise exec -- go test -tags 'test_dep integration' ./tests -run '^TestTestpilotModelSlice$' -count=1, cd model && mise exec -- cargo test -p umpire-temporal-tests --test boundary -p umpire-temporal --test realization, cd model && mise exec -- cargo test -p umpire-dsl-macros --test fixtures private_state, CARGO_TARGET_DIR=/tmp/umpire-fn2164-clean.OGxRrh/target make model-check (disposable clone; empty target and missing descriptor initially), CGO_ENABLED=0 mise exec -- go run .flow/tmp/fn-216-4/compare-descriptors.go proto/api.binpb /tmp/umpire-fn2164-clean.OGxRrh/repo/proto/api.binpb, make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=3cf4aeb4648959c3ccc43a935a389bd63ea42f4a lint-code-fast, make umpire-check-live-tests, INHERITED_RED: umbrella exit2; identical seven legacy failure identities and assertion causes before/after; no green receipt, UNRELATED_RED: default historical-base make lint-code-fast exit2; all10 tool-authored unrelated Go edits exactly restored; scoped nonfixing lint passes, Evidence: .flow/tmp/fn-216-4/commands.md and quality-audits.md
- PRs: