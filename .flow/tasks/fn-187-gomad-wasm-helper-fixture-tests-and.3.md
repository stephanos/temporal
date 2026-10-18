---
satisfies: [R3]
---
# fn-187-gomad-wasm-helper-fixture-tests-and.3 Delete inline tests the corpora already cover

## Description
Delete corpus-covered inline tests across network, volume, namespace, World,
clock, broker and the helper only after recorded coverage and complete assertion
equivalence. Add hand-named corpus cases for every genuine gap, regenerating
Go-generated fixtures on the Go side. Apply the same pass to duplicate Go WASI
namespace, volume, broker and network_v2 tests. Record before/after production
coverage per module and each retained invariant with the reason a corpus cannot
state it. Coverage loss on a model path blocks deletion. R3 remains in progress.

### Current execution state

Reuse `/home/agent/wt/wasm-fn187-functional`, branch wasm-fn187-functional, HEAD
`95eed5c83a26eb1e93f2b1ba2f971e33bc1d51bc`. The accepted R1/R2 index is preserved
against tree `c0bf54df043249aaf092048cb094c2ba4f1f9ee3`; staged diff SHA256 is
`0a343ce3549c1b7ed42146caf23c7800f2cd48f9a60c78ebfd3954036833d299`.
R3 is an unstaged incremental delta. The primary owner advanced gomad to
`bcb72b2bc7bce461dfcb38344be7c2b54153ad81`; do not rebase, overwrite, integrate
or commit either worktree without the existing conductor gates and owner authority.
No push, PR or CI action is authorized. The previously stopped runtime owner is
resumed for the resource-check batch authorized below.

Current source-bound inventory is `.flow/tmp/fn187-r3/functional-inventory-manifest.json`,
SHA256 `a10f5b9f8e4f2d4c825ac4b5bcff5a3589bd00c820e6006453e9232d300ccd33`.
Rust has 198 functional roots, 28 corpus roots, 134 invariants (133 private and
one separately justified host-concurrency root), two retained public originals,
34 machinery roots, one held root and two ignored roots. Diagnostic corpus has
26 roots. Main/helper/network/cluster/engine rosters are 140/21/18/3/7.
Go has 63 functional roots, 27 corpus roots and nine private roots.
All 15 selector files, 15 tools, 53 source bindings and 142 private source bindings
are verified. No new private waiver or reason reclassification is admitted.

### Latest verified batch

The pre-extraction frozen default checkpoint passes all six permitted arms: Rust 196 functional,
28 corpus and 134 invariant roots; Go 63 functional, 27 corpus and nine private
roots. Canonical Rust functional inventory remains 198. The phase-only allowed196
selector explicitly excludes the two held public originals, without reclassifying
them. Source pin is `b2b6531ce942f06c358eeb55aeb671b3e1c0e02b65f89930b7b2f01d681949dd`.
Receipt `r3-checkpoint-default-correctness-allowed196-2026-10-18/receipt.json`
has SHA256 `8ee102cd98a98fb443c3435535ec6b57f1f1adf278981d02ea5d593be46514af`.
All 407 helper cases replay in both relevant Rust and Go arms. Current Rust
functional-minus-union and Go baseline/current coverage gaps are zero. The raw
Rust baseline report retains one unmapped cursor expression; the separate checked
default-cursor equivalence receipt has SHA256
`8ea93ecfb0e0c31fbac7d92c0792f6c2498ca4a69a1b693f494f02085b50037c`.
Mapped baseline losses are zero, not a full-configuration acceptance certificate.

Three unused test-only remnants were removed before this source freeze. All 142
retained test bodies and invariant reasons are unchanged. The earlier consolidated
attempt stopped on three compiler warnings before any test ran. Historical claims
of zero warnings based only on stderr were too broad; this fresh build has zero
warnings in both Cargo JSON and stderr. Strict coverage exports retain exact
registry-only diagnostic inventories of 256/233/101 records, with no admitted
in-crate or unexpected diagnostic. They are not zero-diagnostic exports.

The serialized Go volume transport corpus preserves the original actual
subprocess seam, exact initial reply and full observations for all 115 committed
cases/779 operations. Existing TestVolumeConformanceCorpus supplies the unchanged
independent live-Go reference. Original versus reference-plus-transport production
coverage preserves all 1158 positive blocks across identical 5758 block keys.
Only TestVolumeConformance and its sole volumeConformanceCheck helper were removed.
Post-removal reference, transport and guard roots pass with 235 children/no skips.
Canonical affected-package golangci-lint plus errortype pass after four scoped
corpus-code style corrections. Exact byte assertions remain exact; no JSONEq
normalization, nolint, baseline or gate changes were accepted. Three affected
roots/77 children and the one selected pending-peer helper case also pass.

Receipts under `/home/agent/wt/evidence/fn187/r3-current/` are
`volume-transport-migration-runtime-2026-10-18/evidence.json`,
`volume-transport-migration-inventory-2026-10-18/evidence.json` and
`volume-transport-affected-lint-2026-10-18/evidence.json`.
Earlier tiny-storage, six-cluster/engine, stall, epoch/ABI and transport removals
have source/assertion/runtime conservation receipts linked from the inventory.
Detailed chronological evidence, failed attempts and the 22 retained worktrees'
ownership reasons remain in `.flow/artifacts/fn187-lightweight-correctness-2026-10-18.md`.
The immutable last-gap ledger stays frozen. All 526 original JSON contracts,
215 original helper bytes, 407 current helper inputs, 183 network inputs and
CLOSED42 are preserved. Python's 44 tooling checks pass.

Fresh-context periodic audits found no confirmed correctness defect and one
standards Should Fix concerning the 3101-line helper loader. That cleanup is now
verified: 19 function bodies moved exactly into cohesive common modules; eight
thin wrappers retain root names, and 32 remaining root functions retain tokens
except reviewed module qualifications/formatting. The loader is now 1084 lines.
One fresh pinned build has zero compiler warnings; the exact unchanged 21-name
roster passes with no failures, ignores or filtering, including the unfiltered
407-case corpus. No production code, fixture bytes, selector, private binding,
reason, coverage tool or staged index changed. Evidence is
`helper-loader-extraction-2026-10-18/evidence.json` under the evidence root.
The earlier six-arm coverage/source freeze predates this extraction; no fresh
current-source coverage, feature matrix or full certificate is claimed. Reviewers
and runtime owner are terminal. Verbatim audit reports remain historical below
the chronological artifact's current checkpoint.

### Remaining acceptance and execution constraints

The owner now explicitly requests: "run the resource-senitive checks now".
This lifts the prior WASM resource-execution hold, including the oversized
shared-reader, large-context and RSS original tests, resource/capacity budgets
and the full verification needed by these tasks. Execute serially with one
runtime/compiler owner and unchanged thresholds. The two public originals stay
whole until exact replacement/conservation proof supports deletion. Native
execution, fn-185 cost profiling, ignored throughput/model-profiling roots and
publishing remain out of scope. No task completion is implied by this authority.

The first resumed batch passes all three exact original tests, one at a time:
oversized tagged-frame capacity, chunked pending context over one MiB and exec
RSS inheritance. Build and tests exit 0 with no compiler warnings or source/index
drift. Receipt `resource-originals-2026-10-18/receipt.json` SHA256
`854d662e6f1465c74e03b914482d22c2c55ff9c071e9a10fa3cdb24fb766c8db`.
This is not full coverage or qualification. The runtime owner now proceeds with
the primary candidate's canonical full WASM gate and runtime qualification,
retaining their evidence separately from this R3 worker candidate.

Final current-candidate full/corpus/invariant/union conservation across the
required feature configurations, complete retained-test reasons, original-baseline
coverage conservation, lint, fresh-context independent review and exact primary
integration remain required. Functional-only checkpoints cannot certify full
acceptance; preserve the existing fail-closed final verifiers. R4 module moves
follow completed R3, and R5 retains its original counts/test-wall-time/docs scope.
Fn-189 current-candidate runtime/workload, exact replay/identity, budgets and
remaining host gates are unproved; keep its milestone in progress.

Use pinned Rust1.99.1 and matching LLVM tools for fresh coverage, plus stock
Go1.27.1, serial builds/tests/merges and exact
selectors. Require matching actual object SHA/build IDs/modes, separate raw pools,
strict diagnostic validation and no unexpected skips. Final gates retain their
original diagnostic requirements. Sanitize inherited helper,
generator/filter/seed and Rust/Go flag overrides. Never copy onto a Cargo hardlink;
ordinary embedded helper restoration uses a fresh inode with before/after checks.
Do not rerun the historical hardcoded selector preparer. Batch related correctness
work, combine verification and coverage runs, and write one compact handover.

Current hygiene checkpoint: `.flow/artifacts/rust-hygiene-2026-10-18.md`.
Primary and R3 formatting and strict Clippy pass across all six configurations.
The complete ordinary R3 Rust suite passes after the final projection extraction:
199 passed, zero failures, two ignored manual benchmarks; all three exact resource
originals pass with Rust1.99.1. Independent correctness has zero findings at
confidence75+, and the standards loader-size finding is resolved at998lines.
These are scoped hygiene checkpoints, not full conservation or final acceptance. The primary full WASM gate failed on the measurement
pure/host boundary; focused architecture and measurement regressions now pass
after an ownership fix. A fresh full gate and qualification are still required.
All pre-hygiene source/hash/line/coverage bindings must be refreshed and verified;
previous receipts remain historical, not current-candidate certificates.

**Touches:** tools/gomad/internal/backends/wasm/wasmhost/src/**, tools/gomad/internal/backends/wasm/wasmhost/tests/**, tools/gomad/internal/backends/wasm/wasi/**, tools/gomad/internal/model/network/**, tools/gomad/internal/model/volume/**, task-owned coverage scripts/evidence and documentation

2026-10-18 continuation: current Quick commands now pin Rust1.99.1; mise moved at owner request to tools/gomad/mise.toml in both checkouts, with CI/docs/source pin updated and 48 Python tooling checks passing red-to-green. The 142 retained-test bindings were mechanically refreshed without changing names, reasons or classifications; original staged patch remains unchanged. Coverage runner and snapshot pin use matching Rust1.99.1/LLVM23.1.1 and sanitized inherited flags/filters. New coverage remains unproved.

The remaining two public originals will be replaced through the existing Rust component-corpus owners, not a new private waiver or production API. Four exact-original JSON/wire fixtures are prepared and independently byte/hash checked: oversized tagged frame plus fresh same-instance reader, and original 400-operation chunked context. Both originals remain whole. Negative-control tests and deliberately no-op adapter stubs are now present in R3 pending the RED run; do not treat this worker as green or issue a final review. The existing EOF single-JSON roster intentionally rejects the new proposed fixture until its adapter is admitted. The latest source-binding hashes for those two adapter files are now stale during this test-first batch and must refresh after implementation. Runtime/compiler work stays serialized behind primary full WASM gate session68363; its conformance/projection/architecture constituents pass, full WASI execution remains live. Primary qualification-source manifest has5422 absolute-path entries under /home/agent/wt/evidence/fn189/rust199-full-gate-2026-10-18/. No full conservation, resource qualification, final acceptance, integration or completion is claimed.

Latest continuation supersedes the pending-stub checkpoint above: intended RED101 observed for both no-op adapters (classification and maximum controls), then read-sequence adapters implemented in the existing component corpus owners. Complete ordinary Rust suite201PASS/0FAIL/2ignored/0filtered, all407 helper cases and both original tests retained. Six-mode strict Clippy/fmt and all48 Python tooling checks pass. New guards explicitly admitted as machinery2: functional200/corpus28/diagnostic26/invariants134/public2/machinery36, held1/ignored2 unchanged. No new private waiver or deletion. Independent scoped correctness review found zero confirmed defects>=75. The original transport.rs/storage_transport.rs files match the previous full-suite source hashes; staged patch remains0a343ce3 unchanged.

Primary full WASM gate terminated exit2 after10m39s; sole reported failure was stale compiled-model digest after the reviewed measurement/process ownership move. Expected digest rebound to9b087b968ee0054b2886b670934e37f0e4546939442d61fde69606ac056c4c99 and process.go mutation/missing-source control added. All identity regressions pass in primary/R3, not a fresh full gate or qualification. Runtime/workload gates still pending.

Fresh frozen Rust1.99.1/default coverage build is live under session57822; source pin aa4a9903b6e35673ab69338585d24416855b7b7ad52d60344c65ba21b8ff02fc, phase /home/agent/wt/evidence/fn187/r3-current/r3-rust199-sequence-conservation-2026-10-18/default/. Do not edit worker phase inputs. Retain strict matching objects/LLVM23 diagnostics/separate raw pools and unchanged final conservation gates. Primary MIGRATE.md now provides the user-requested private repository migration handoff; no remote changes were made. Current evidence and verbatim correctness report are in .flow/artifacts/rust-hygiene-2026-10-18.md.
## Acceptance
- [ ] Coverage report before and after per module retained; no production line loses coverage
- [ ] Every deleted test names the corpus cases that replace it
- [ ] Remaining unit tests listed as invariants with reasons
- [ ] Go-side duplicate tests handled the same way

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
