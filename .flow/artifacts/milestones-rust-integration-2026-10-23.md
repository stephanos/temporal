# Rust integration development checkpoint

Current source: `e89dc0a247`, retained by `gomad` head `e944295e58`.
Observed 2026-10-23T07:09:20.810374+00:00. **Development verification only.** fn-232.13/R12-R14,
fn-222.1/R1-R2 and final retirement acceptance remain open. No final frozen
qualification or per-test support claim is recorded.

The combined root includes the single Rust workspace/default host, authenticated
runtime receiving reader, transitive process cleanup, explicit Temporal policies,
original tag-token compatibility, onebox orchestration and exact raw Spec/scenario
witnesses, private alias inspection caches, support publication/validation,
installed-engine cache propagation, cluster aggregate facts, determinism guards,
and full compiler/preparation inventory/cache receiving controls.

Root verification of the current source:

- Release library: **351 passed, 2 opt-in probes ignored**, exit 0.
- Tracked Python driver: **172 passed**; workload tools: **20 passed**, both exit 0.
- Release all-target strict Clippy with `-D warnings`: exit 0.
- All commands used the isolated retirement integration target and two build jobs;
  the release library tests ran serially. Exact raw logs and hashes are in the
  companion JSON under checkpoint `e944295e58`.

Preparation's exact-source real Go 1.27.1 probe separately passed cold/warm
compilation, scoped GOMAXPROCS/GOTMPDIR/GOWASM/-p controls, changed-argument warm
admission and source invalidation. Both independently read private sidecars
preserved complete 9-compiler/430-source inventories. This is compiler/preparation
proof, not guest qualification. Evidence:
`/home/agent/wt/evidence/fn232-preparation-integration-20261023/files.json`.
Old Rust optional World fields remain omitted rather than reserializing historical
configurations; new runtime preparation and the independent reader require the
explicit default 64 MiB World capacity. Actual execution enforcement is still
being completed in the core branch and earns no pass here.

Earlier integrated checkpoints retain these exact scopes:

- `8c75c49970`: required `make -C tools/gomad lint-rust` passed all 13 supported
  host/helper feature configurations. Three preceding attempts failed with
  concrete test conversions, a conformance-only unused method and linked tests
  referencing APIs absent from conformance builds. Checked conversions and
  matching existing feature guards fixed them without suppressions. The four
  affected boundary tests passed; retained Go archcheck/protocol tests passed.
  Independent same-family review found no behavior or applicable coverage loss.
- `9a1807bdce`: 332 library tests passed, one ignored; default strict Clippy and
  release support example build passed. Onebox orchestration 14, inspection 22
  and driver 154 tests passed. Source review closed both the compiler-alias
  cache collision and exact inspection recording-envelope findings. These
  checks precede the latest preparation changes and are not relabeled.

## Remaining acceptance

Core World transition-byte enforcement and public artifact/replay coverage,
measurement/native producer preservation and its strict feature checks, actual
terminal/cleanup controls, interception acceptance, and the coherent source-effect
inventory append remain under their existing isolated owners. All 26 historical
Go inventory obligations still require exact receiving reconciliation; the
relocated Namespace conformance root remains an actual retained Go oracle.
Removed subprocess transport predicates need explicit native applicability
mapping, not blanket skips or aliases to generic success tests.

The final frozen 32-by-25 runtime matrix, required workload and retained-replay
cells, complete `test-wasm` behavioral reach, raw/golden lanes and second
regeneration, 3,600-second qualification cycle, support publication, suite sweep,
profiling and final size comparison remain pending. Earlier scoped executions
retain their original source/binary identities. They do not qualify this source.

Open-spec worktrees and their evidence remain retained for active source and
acceptance owners. User `tools/gomad/.turbo` files and the old generated helper
binary remain untouched. Root's isolated target is inactive after the checks,
retained for further integration. Latest disk observation: checkout 139 GiB
free; `/home/agent` 50 GiB free. No root cache/evidence deletion.

## Historical checkpoint observations


Snapshot: 2026-10-23 06:23:55 UTC. Integrated source: `gomad` `292b8f8848e91ec0dade547044bb7ef668b5fba6`, from starting integration `0fee7dcb2b`. This is a development checkpoint. fn-232.13/R12-R14 and fn-222.1/R1-R2 remain open; no frozen qualification or retirement acceptance is recorded.

## Integrated source

The prospective retirement layout (`8849a27d15`) is now integrated: the Rust default host and helper share one workspace/lockfile, the transport helper is reference-only, and retained Go models/generator/oracles compile. The union preserves all 128 resolved package tuples. Narrow architecture repairs preserve generic Check while Discover rejects retired owners. Subsequent driver cleanup repairs follow registered process ancestry and owned sessions, attempt group/session barriers independently, preserve active registrations on failure, and publish an incomplete outcome after failed cleanup.

The shared explicit Installation/Qualification/Inspection policy constructor, zero-execution preparation admission repairs, and original compiler tag-token compatibility fix are integrated. The policy merge preserved existing configuration-decoder error-precedence tests. Preparation rejects authenticated executed records and mismatched source/argv/tags/scenario/network bindings.

The exact release-linked runtime producer, ordered onebox runner, original onebox inspection capture, and raw receiving adapters are integrated as development code. Inspection omits the invalid explicit zero choice flag, retains the original effective 1 GiB scenario bound, preserves private controls, and captures authenticated overlay descriptors/origins before alias detachment. This code does not itself earn a support row or qualify the integrated candidate.

## Actual root verification

The companion JSON authenticates the retained raw logs and their numeric exit 0. Root commands:

- `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn232-retirement-integration CARGO_BUILD_JOBS=2 cargo +1.99.1 test --manifest-path tools/gomad/rust/Cargo.toml --release --locked --offline -p gomad --lib`: 261 passed, one pre-existing ignored, before the narrow tag-token correction and later runtime-module integration.
- The same command with final filter `preparation`: 9 passed after the tag-token correction. New runtime/support/cache source needs its own integrated rerun.
- `cargo +1.99.1 clippy --manifest-path tools/gomad/rust/Cargo.toml --release --locked --offline -p gomad --all-targets --no-deps -- -D warnings`, with that isolated target/jobs: exit 0 after preparation integration, before later runtime/support/cache changes.
- `python3 -B -m unittest discover -s tools/gomad/rust/driver/tests`: 150 passed after runtime/onebox and reviewed nested cleanup integration.
- `python3 -B tools/gomad/testdata/parity/onebox-gates-test.py`: 11 passed.
- `python3 -B -m unittest discover -s tools/gomad/rust/workload_tools -p 'test*.py'`: 20 passed.
- `python3 -B tools/gomad/testdata/parity/onebox-inspection-test.py`: 19 passed after descriptor capture.
- `onebox-inspection-cli-test.py` against retained immutable `a07dd6e2c404ce9120c448e8568c18906fa2f9aebd87d11ce8d53a7bac8a861b`: corrected argv reached the invalid-scenario sentinel; old explicit choice override refused. Both actual exits were 2, with distinct asserted reasons. This is parser evidence, not current-candidate execution.
- Direct `python3 /home/agent/.codex/scripts/flowctl.py validate --all --json`: exit 0, 68 specs/457 tasks, zero errors, 34 retained historical warnings.

Independent same-family `gpt-6.1-sol` high reviews accepted the preparation fixes, inspection capture, and final driver cleanup (`f0e8b834f4`). The onebox driver review accepted its source conditional on support alias admission and transitive cleanup. Runtime review identified missing independent provenance/choice-summary admission and a shard-deadline gap; fix `50a22fb6d9` is undergoing re-review and is not yet integrated in this snapshot. Determinism review found sourceguard and zero-choice diagnostic gaps; its owner is repairing them. No component review supplies whole-task acceptance.

## Scoped live evidence and remaining integration

Runtime worker retained a real exploratory seed-0 shard: 25 fresh executions plus diagnostics-off exact replay, all public inspections, and a strict independent retained reread. Evidence: `/home/agent/wt/evidence/fn232-runtime-probe-20261022/`. It is explicitly not the final 32-by-25 cycle and does not qualify this integrated source. The stricter reader receipt binds its own source separately from the prior execution candidate.

Interception work remains isolated on `fn233-interception`. Current handwritten Go -exec and unchanged Make positive guests passed, and 120 synthetic cases passed on its immutable candidate. Live testing exposed cached go-run admission outside GOTMPDIR; source repair waits for live immutable Make processes to terminate. No fn233 completion is recorded here.

Support producer/validator and complete compiler-alias projection are still owned in `fn222-support-v2`; cache/lifecycle propagation is being completed in `fn232-linked-cache` with unique lifecycle changes preserved. The determinism branch preserves the 276-record prefix and has exact source-site rows, but its unfiltered gate still fails on 26 references to deleted Go evidence. These are real pending obligations, not waived checks. Original core/measurement preservation, full stress matrix, mandatory workload/replay/inspection gates, full one-hour cycle, support publication, suite sweep, profiling, and final size comparison remain under their Flow owners.

## Resources and custody

Root's isolated build target is `/home/agent/.cache/codex-build/fn232-retirement-integration`; root compilation commands above are terminal. Other current workers own distinct fn222-support, fn232-runtime-qualification, fn232-determinism-new, fn232-cache-lifecycle-new and fn233-interception-new targets. Do not clean another owner's target while its build/runtime/review work remains active. Evidence under `/home/agent/wt/evidence` and `/home/agent/evidence` remains open-owner custody.

New driver/inspection/support/runtime worktrees are retained for receiving integration and review; preparation's clean worktree was removed/pruned after its commits were integrated. The superseded duplicate workspace experiment is preserved only in branch `fn232-workspace` and must not be integrated. Older registered worktrees remain open fn185/fn188/fn221/fn232 reference or source owners, as indexed in the existing continuation handover. Untracked user `tools/gomad/.turbo` files remain untouched. The former helper target binary at `tools/gomad/internal/backends/wasm/wasmhost/target/release/gomad-wasmhost` is retained unchanged; it is not used to prove the new candidate.

Latest disk observation: checkout 172 GiB free; `/home/agent` 83 GiB free. No shared cache/evidence removal was performed by root during this batch.


## Reviewed cache and determinism integration update

Observed 2026-10-23T06:48:53.137146+00:00. Source checkpoint `e325783f6c`, following clean merge
`81a66b54b3` and onebox Spec-witness commit `ea31ff1702`. Earlier sections
retain their original checkpoint meaning; this update supersedes their pending
source-integration descriptions only.

Reviewed runtime provenance/choice-summary/deadline fixes (`0fd7ae7b6e`),
determinism source guards and zero-choice diagnostics (`d71c795a8a`), and cache/
lifecycle source (`187c1774fe`, handover `0df8180f8c`) are integrated. Cache
propagation, authenticated replay error classification, cluster aggregate facts
and complete configured/retained/summary ownership passed same-family independent
source review. The determinism ledger and its original 276-record prefix remain
unchanged by the final stress lint delta. Earlier source-review SHA rows describe
their historical bytes; `e325783f6c` binds the subsequent four-file lint changes.

Actual root verification, authenticated in the companion JSON:

- Full release library tests at `81a66b54b3`: 310 passed, one existing ignored,
  exit 0, run serially in the isolated retirement integration target.
- Integrated driver: 154 passed; workload tools: 20 passed; both exit 0.
- Onebox orchestration: 14 passed; inspection: 19 passed, observed terminal
  exit 0. A subsequent attempted workload-test path was absent; the corrected
  unittest-discovery command above passed and is retained separately.
- Strict release all-target Clippy initially failed with nine stress diagnostics,
  exit 101. Fix `e325783f6c` borrows unconsumed values, preserves exact framed
  test bytes, and checks a test-only integer conversion; no lint suppression.
  Strict Clippy then exited 0; all 26 stress tests passed, exit 0. Independent
  read-only review accepted the source delta as behavior-preserving.

Support publication commit `137c932a01` remains outside root pending review
repairs. Review found an alias-overlay cache collision and incomplete inspection
recording argv/scenario admission; the onebox and support owners are repairing
those together. Review also found the parsed World-transition byte cap was not
consumed by execution. Core preservation owns actual enforcement, retention and
replay tests; authenticating a requested flag cannot prove its enforcement.
These findings block acceptance and are not waived by the passing source tests.

Core, measurement, preparation receiving and interception work continues in its
existing isolated worktrees. The 26 retired-Go inventory references still need
exact behavioral reconciliation, including actual runtime/fixture proof. Full
frozen runtime/workload/replay qualification, source-generation checks, final
raw/golden lanes and support claims remain pending under the original milestones.

Disk observed after this build wave: checkout 152 GiB free, `/home/agent` 61 GiB
free. No root cache/evidence deletion. Existing open-spec worktrees listed in the
prior continuation inventory remain retained for their active source/evidence
owners; root's retirement integration target is retained for integrated checks.


## Core, interception, journal and measurement integration checkpoint

Root integrated core `4fcf9fa06b`, stress final/parity readers `e4acc1d0b8`,
interception `8e74337901`, authenticated journal observations `2f52fbb724`,
and measurement `e3fa292033`; journal evidence is retained by `23f5cb47af`.
These are source-integration checkpoints, not accepted host retirement or final
qualification. Existing earlier checkpoints retain their original meaning.

The interception merge initially rejected raw non-UTF-8 native metadata environments.
Its meaningful existing passthrough test exposed the regression. Shared raw command
execution now preserves native environment bytes, while observed preparation retains
its exact controlled UTF-8 map. Fresh same-family independent source review found no
remaining integration blocker. The measurement merge preserves fixture state, World
and storage bounds, profile capture and actual timing together. A missing cfg-gated
synthetic Execution initializer was found by review and repaired. Small subsequent
lint fixes preserve startup guard order, status conversion and emitted fixture bytes.

Completed root checks (raw logs and numeric exits in the companion JSON):

- Core checkpoint: 369 native and 212 helper tests passed, existing opt-in tests ignored.
- Interception checkpoint: 462 native tests passed, two ignored; strict workspace
  all-target Clippy passed. Stress reader 16, all-verb reader 13, interception shim
  reader 20 and Make reader four tests passed.
- Composed measurement: 575 native and 213 helper tests passed; respectively two
  and twelve existing opt-in tests ignored. The first compile failed on a newly
  required timing field and is retained alongside the corrected successful run.
- Corrected Python driver discovery ran 172 tests, with five stale ignored-producer
  inventory errors and an exact `qualify/spec.rs#decode_first` anchor failure.
  The preliminary wrong discovery pattern ran zero tests and exited five; it earns
  no verification credit.

The expanded admitted lint matrix adds the three native measurement profiles to the
existing thirteen configurations. Its final sixteen-configuration run is pending at
this checkpoint. An accidentally included forbidden fixture-plus-diagnostic feature
combination correctly triggered its compile-time refusal; it was removed from the
admitted matrix, with the refusal log preserved. Other failed lint attempts remain
retained; no warnings were suppressed.

Actual counterfactual execution has proved the draw and clock pairs on its own
immutable candidate. Handoff exposed missing ordered drain steps during early cluster
failure; its owner is fixing the producer while retaining the existing strict prefix
validator. No four-pair completion or broad provider/functional/retained coverage is
claimed. Real missing original predicates remain under the core owner. Qualification
primary recipes, standalone inspection admission, stable original backend evidence,
terminal-to-reap observations, fresh original ringpop references and derived source
custody remain active implementation work. None of the 96-stage/final-section gates,
full runtime/workload/replay matrices or the complete milestones are marked done.

Root's target remains `fn232-retirement-integration`; source/evidence worktrees under
`/home/agent/wt` remain retained for their open owners. Latest resource observation:
checkout 105 GiB free, `/home/agent` 16 GiB free. No root cache cleanup was performed.
The two untracked user trees remain untouched.


## Accepted supported-role source checkpoint

`d21472e774` follows reference source merge `9dc62d4f6a`, retained readers
`b0be4185de`, profile readers `be131fbbf0`, and external workload source custody
`2db19d584e`. The entire expanded `make -C tools/gomad lint-rust` command now
exits zero across all sixteen admitted configurations, superseding the earlier
pending role check. Failed attempts remain authenticated in the companion JSON.
The final repairs preserve native socket decoding and the old conformance
Unsupported path, gate only unavailable retained-reader assertions, move two
unchanged diagnostic vocabulary functions before tests, and remove a last-use
clone. Fresh independent same-family review found no remaining source blocker.

Actual role checks also passed: conformance library 68 tests (seven existing
opt-in tests ignored), native measurement fixture 15 tests, reference reader nine,
retained reader thirteen, profile reader twelve, and full source-custody/workload
Python suite thirty-one. No ignored test is counted as runtime proof.

The original runtime-cost development pair now genuinely exits zero under both
Go-owned and helper-owned execution: 100,000 select polls, 200,037 calls, expected
stdout and identical original sameEvidence. Its immutable helper is explicitly a
development oracle; this does not replace the current native third leg or final
frozen qualification. The actual source snapshot probe also retained and verified
240,370,785 bytes across 8,419 files outside the frozen repository; it is source
custody evidence, not a workload execution pass.

Reviewed terminal/custody followup `51b566a2ba` and stable backend evidence
`fc0e9dbe9d` remain ready for root integration. Core implicit-drain repair
`132b1ceb15` has a causal actual-WAT regression and independent review, but the
fresh original four-pair proof remains incomplete. Driver primary composition,
standalone inspection admission and original wire-only terminal proofs remain in
flight. Inventory resolution and the complete original behavioral gates remain
open; no Flow task is completed by this checkpoint.

All root build/test handles are terminal here. The companion JSON records every
registered worktree with a retention reason and the open evidence/scratch roots.
Root performed no cache cleanup. Last observed free space: checkout 109 GiB,
`/home/agent` 21 GiB. Untracked user trees are unchanged.
