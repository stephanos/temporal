# fn-180 integrated WASM inspection batch

Work in progress; this is not task completion or final-candidate acceptance.
Native execution remains deferred. No push, pull request or CI dispatch occurred.

## Runtime qualification checkpoint

The first integrated runtime candidate passed mandatory qualification with a
frozen maintained-source manifest. The root checked the same manifest after
the terminal result (`sha256sum -c ... --quiet`, exit 0).

- Command (repository root):
  `GOMAD_WASM_TEST_HELPER=/tmp/gomad-fn180-release-helper-final GOMAD_WASM_QUALIFICATION_ROOT=/tmp/gomad-fn180-qualification-weqhi9/runtime GOMAD_WASM_QUALIFICATION_MANIFEST=/tmp/gomad-fn180-qualification-weqhi9/source.sha256 CARGO_TARGET_DIR=/tmp/gomad-wasm-current-target make -C tools/gomad/internal/backends/wasm qualify-runtime GOMAD_WASM_STOCK_GO=/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go`
- Exit 0: 32 seeds, 25 fresh recordings per seed (800 total), 32 exact replays,
  aggregate pass, 8,150 recorded trace records.
- Diversity: 30 distinct choice hashes, 32 output hashes; map 32, run queue 18,
  timer callbacks 2, canary 2; exit outcomes 0 and 17.
- Linux/arm64, Go 1.27.1, `nogreenteagc`, Wasmtime 47.0.3,
  `cranelift-fuel-nan-canonical-nounwind-v2`.
- Source manifest: `41488fedbb8d732c8d49fe2312747ee083c24c36d6584b6972aa24fcef1660b4`.
- Helper: `86335660d8a7f7225d41c4cbbcccb66935d1b0a682a15000add076f34b732eb5`.
- Guest: `97a745597433330e23ebca2090925df9c4fc20355a6bfc6f24288c37a74f75a2`.
- Runtime overlay: `7c220e2c89e83713598a0549b82d731b01e560d711dea865c5d62c301d14bc12`.
- Runtime implementation: `sha256:6940d4218be85ff27527c83573d5f00474382960964b1b37f56f06b0ececb079`.
- Model: `sha256:34d8ab4cccbe85fd848c716e5671efd3cd2f06edc7118c38bf1f418a4f92c5b9`.
- Compiler: `1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64`.

Raw evidence: `/tmp/gomad-fn180-qualification-weqhi9/runtime/seed-00` through
`seed-31`; command log `runtime-qualification.log` and machine receipt
`runtime-acceptance.json` in the parent directory. These results belong only to
the identities above; subsequent fixes are not silently covered by this run.

## Integrated diagnostic run and remaining work

The first full `make -C tools/gomad test-wasm` ended with exit 2; raw output is
`/tmp/gomad-fn180-test-wasm.log`. It exposed architecture gzip admission,
in-flight qualification test declarations, the initially ungated Temporal
integration test's reserved `TZ` environment, stale captured-entropy WAT bytes,
ordinary replay divergence misclassification, Go-owned debug initialization
refusal, and a storage-history watchdog under concurrent qualification load.
Record focused repairs and final reruns below; this diagnostic run is not green.

Final-helper retained shard and mutable-state lock-held inspection cells passed
in `/tmp/fn180-temporal-final.log`. The one-box cell failed before recording
because its standalone network route uses Go ownership and debug initialization
required a helper-local model. A one-node typed scenario demonstration and a
separate ordinary Go-owner compatibility regression are in progress. The two
lock-held cells alone are not being substituted for the complete R6 workflow
demonstration.

The storage-history failure reproduced in the focused test (exit 1, 38.051 s;
`/tmp/fn180-storage-history-focused.log`), so it is not dismissed as concurrent
load. The noninstrumented WAT fixture invokes 100,000 writes; the helper's new
unconditional inspection snapshot path serializes growing storage state after
each call. The helper repair must remove inspection work for guests without
the debug ABI rather than increase this test's watchdog.

`make -C tools/gomad validate` subsequently passed with exit 0; raw output is
`/tmp/fn180-validate-current.log`. This is generated/source validation, not a
native runtime qualification pass. Low-level Go-owned inspection now refuses
before launching a helper; its regression test and implementation identity
checks passed (exit 0). This intentionally rotates the compiled Go model digest
to `sha256:d3aacafd7ee76fd70cbe6f6623d06238950f6ac4355c8d0fa2d0f76c211bd078`.

Final integrated tests, source-bound helper rebuild qualification, independent
review, complete task/R-ID mapping and local commits remain pending.

## Compatibility repair checkpoint

Private production helper
`/tmp/gomad-fn180-release-helper-regressions` has SHA-256
`2b5e1095308be07201c3b0cf548ad5797afa7a4e41d3b944dbf2576b3a023f7b`.
It repairs Go-owned dormant debug hooks, ordinary replay divergence ownership
and unnecessary per-filesystem-call inspection snapshots. Active inspection on
Go-owned paths remains explicitly unsupported, not silently enabled.

- Full Rust suite: exit 0, 373 passed / 2 ignored;
  `/tmp/gomad-fn180-rust-regression-suite-final.log`.
- Default and conformance clippy, all targets, warnings denied: exit 0;
  `/tmp/gomad-fn180-rust-regression-clippy-final.log` and
  `/tmp/gomad-fn180-rust-conformance-clippy-regressions.log`.
- Conformance rebuild: exit 0, unchanged digest
  `26bbdfa2b72a23a497aa9687179d633d009a90bcaa882f3b7cba545463df5091`;
  `/tmp/gomad-fn180-rust-conformance-release-regressions.log`.
- Actual forced exact replay, five tape tamper controls, captured-input parity
  and inspection refusal guards: exit 0, 5.643 s;
  `/tmp/gomad-fn180-forced-replay-regressions.log`.
- Actual source-free retained standalone and restarted-cluster inspection:
  exit 0, 44.248 s; `/tmp/gomad-fn180-retained-inspection-ticker.log`.
- Go-owner evidence, filesystem parity, interrupted-prefix recovery and large
  storage history: exit 0, 13.587 s;
  `/tmp/fn180-ownership-regressions-configured.log`. The storage test itself
  completed in 6.26 s without changing its watchdog or expected evidence.
  An earlier attempt omitted required `GOMAD_STOCK_GO`; its non-green output
  remains `/tmp/fn180-ownership-regressions.log`.
- Architecture/source-set check: exit 0, 33.163 s;
  `/tmp/fn180-archcheck-current.log`.
- Full affected backend lint: exit 0 / no issues;
  `/tmp/gomad-fn180-bridge-targeted-lint.log`.

The subsequent real Temporal scenario recording exposed a separate operator
choice-trace cap (`capacity`, `WASI capacity: runtime-choice-bytes`, helper
reaped, no cancellation/watchdog). Its raw diagnostic is
`/tmp/fn180-temporal-scenario-final2.log`, exit 1 / 170.456 s. It reached
workflow startup, not workflow completion or the canary. The scenario default
was 1,000,000 bytes although the reused whole-workflow qualification uses
`choice.MaximumTraceBytes`; the bounded default and error precedence are being
repaired under the existing operator owner.

R6 also exposed a predictable inspection footprint problem: each boundary
repeats the 51,730-byte captured schema, growing namespace/recovery/network
state and full World snapshots. The bounded metadata repair keeps the existing
64 MiB cap and all boundaries/raw prefix streams; it replaces repeated opaque
state snapshots with explicitly versioned, domain-separated commitments made
from recording state and independently recomputed during replay. Final helper
rebuild, tamper tests and qualification must cover that representation before
acceptance. Earlier helper results above are not relabeled as final-v2 proof.

## Frozen compact-v3 candidate

Metadata v3 keeps every boundary, lifecycle action, prefix offset and raw
128-byte witness. Versioned binary boundary/lifecycle frames remove repeated
JSON fields; recording-authenticated state commitments are recomputed from
actual owner state during replay. All six raw prefix streams remain retained.
The hard limits remain 64 MiB for inspection metadata and 512 MiB for the
outer retained bundle; the controller's separate scenario-evidence bound is
not the retained-bundle bound. Old metadata v1/v2 is explicitly unsupported.

The realistic sizing regression retains 51,000 boundaries and 376,000 model
effects, with 4,896,000 choice bytes and 6,016,008 synthetic WASI bytes. Go's
actual encoders produce 54,360,725 metadata bytes, 67,418,763 ClusterRecord
bytes and 89,892,781 retainedExecution bytes. The WASI stress stream is
synthetic, not a recovered historical raw workload. Actual R6 remains required.

- Maintained-source manifest: 1,994 files,
  `/tmp/gomad-fn180-qualification-compact-v8lRoB/source.sha256`, SHA-256
  `3c990db2097f9490edc5b9df3977d6a8512c89b287237b14247a3f88bf3aa5f1`.
  Root precheck and post-runtime-qualification check both exited 0.
- Private single-link production helper:
  `/tmp/gomad-fn180-release-helper-compact`, SHA-256
  `a8dfd7fe41013d41e7206a6029759087b25fc39920d98688d44c8f5e20bdba97`.
- Private single-link conformance helper:
  `/tmp/gomad-fn180-conformance-helper-compact`, unchanged SHA-256
  `26bbdfa2b72a23a497aa9687179d633d009a90bcaa882f3b7cba545463df5091`.
- Rust suite: exit 0, 377 passed / 2 existing ignored;
  `/tmp/gomad-fn180-compact-full.log`. Default/conformance clippy with warnings
  denied, pinned release builds and formatting also passed.
- Strict Go codec/compatibility/tamper/source-inventory and realistic-size
  tests: exit 0, `/tmp/gomad-fn180-compact-focused.log`.
- Actual retained standalone/restarted-cluster inspection: exit 0, 36.593 s,
  `/tmp/gomad-fn180-retained-inspection-compact.log`. Empty folded-clock
  frames remain canonical nonnil empty bytes, preserving the helper's base64
  request ABI through cloning; its regression is included.
- Full affected targeted lint: exit 0 / no issues,
  `/tmp/gomad-fn180-compact-targeted-lint.log`.

The remaining logs in this section live under
`/tmp/gomad-fn180-qualification-compact-v8lRoB`:

- Focused live inspection at all 35 decisions: exit 0, 13.032 s,
  `live-inspection.log`.
- Mandatory documented `qualify-runtime`: exit 0, 800 recordings (32 seeds
  times 25 fresh instances), 32 exact replays, aggregate pass, 8,150 trace
  records; `runtime-qualification.log`, `runtime-acceptance.json` and raw
  `runtime/seed-00` through `seed-31`. Diversity remains 30 choice hashes,
  32 outputs, map 32 / run queue 18 / timers 2 / canary 2, outcomes 0 and 17.
  All reports bind the manifest/helper above, Go model `d3aacafd…`, guest
  `97a74559…` and runtime overlay `7c220e2c…` under Go 1.27.1 Linux/arm64.
- `make -C tools/gomad validate`: exit 0, `validate.log` (portable generated
  source validation, not native runtime qualification).
- Full `internal/archcheck`: exit 0, 57.420 s, `archcheck.log`.
- Full `internal/model/cluster`: exit 0, 0.067 s, `model-cluster.log`.
- Canonical `make lint-code-gomad` with Linux pinned linter/vettool overrides:
  exit 0 / no issues, `lint-gomad.log`.
- Public debug library ordinary and native-Gomad no-op source tests: exit 0
  each (`go test -tags test_dep ./common/gomaddebug -count=1` and the same
  command with `-tags 'gomad test_dep'`); no native execution pass is implied.

Final installed fn-181 matrix: exit 0, 66.122 s,
`/tmp/fn181-compact-operator-20261016-RsPr9s/matrix.log`; manifest checks in
`source-before.log` and `source-after.log` exited 0. SIGKILL after two committed
outcomes preserved resumability and resumed to the 24-outcome baseline.
Detached durable success/canary/network replay and all 18 tamper controls
passed. Installed helper/model match this candidate. Installation hashes:
kill/resume `21e0dc4cadfdbee538aee7fd662584a788e176051ad107f7b378aab6ebcb329c`,
detached `7db358ddc54fa5d33ba350edd3019ece2a2dd41c83293b859bdd45eff6cb378c`.

Full `make -C tools/gomad test-wasm` passed with exit 0 at 04:22:24 UTC;
raw output is `test-wasm.log` under the final qualification root above. The
final R6 all-three demonstration failed on the one-box watchdog. Independent batch review,
final task/R-ID mapping, Flow completion and commits remain pending.
The R6 log/root are `/tmp/fn180-temporal-compact-final3.log`
and `/tmp/fn180-temporal-compact-final3`. Both lock-held cells passed; no
one-box workflow success is claimed. fn-182 preparation is refreshed in
its existing handover; measured cells have not started.

Independent review must cover the full previously unqualified implementation,
not just this session's repairs. The correct integrated source base is the
completed fn-178 checkpoint `d087db1e48723d1164c19f559f78d75fc5731eb9`:
fn-181 originated at its child `fe3fa4debe`, and fn-180's initial commits
precede the later `6a73d6dfb2` stop checkpoint. The review scope is fn-180
R1–R6, fn-181 and fn-183, including new codec/source inventories; fn-182
profile-preparation code can be reviewed without claiming its pending measured
R4/R5 acceptance. The intervening naming migration is already closed and must
remain preserved, not be undone.

## Integrated task and requirement mapping

The following mapping describes evidence, not Flow completion. Independent
review remains pending; fn-180 task .5 additionally requires a passing real
one-box demonstration. Opt-in qualification skips in the broad suite are not
substitute passes. All current results bind the candidate above.

| Task / requirements | Evidence |
| --- | --- |
| fn-180.1 / R1,R4 | Strict preflight, authenticated cuts/registry, v3 codec/tamper/capacity/source inventories; actual source-free standalone and restarted-cluster retained inspection. |
| fn-180.2 / R1,R4 | All 35 real decision stops; allocating dumps/GC, readable pprof, witness-only perturbation, cancellation/reaping; 800 recordings and 32 exact replays. |
| fn-180.3 / R2,R3,R4 | Registered dumps, panic/truncation, allocation-free probes, allocating-probe divergence with diagnostics off, fuel pressure and nonyielding timeout; ordinary/native-no-op source tests. Historical compiler/assembly receipts remain historical source-shape evidence, not a new overhead measurement. |
| fn-180.4 / R5 | Scan/nonmonotonic recovery, conditional bisect/reversal, budgets, cleanup and preflight; actual fresh standalone/restarted-incarnation evaluations including panic and timeout. |
| fn-180.5 / R6 | Shard and mutable-state lock-held demonstrations passed. Real one-box hit its original watchdog, so this requirement is open. |
| fn-181.1–.2 / R1,R2 | Full-suite preparation/admission/classification controls and installed Rust matrix; clone, standalone refusal and watchdog/cancellation/protocol precedence regressions. |
| fn-181.3 / R3 | Final source-free durable success/canary/network replay and 18 missing/tampered-payload controls. |
| fn-181.4 / R4 | Final SIGKILL after two committed outcomes, resumability/refusal controls and resume equality with 24-outcome baseline. |
| fn-181.5 / R5,R6 | Final installed matrix (66.122 s), runnable command-shape tests and full WASM/lint/validate; named fixtures only, not arbitrary Temporal support. |
| fn-183.1 / R1,R3 | Preserved caller audit: production routing already isolated, no unsupported adapter deletion. Current installation/reference-selection refusal and lifecycle/reference tests pass. |
| fn-183.2 / R2 | Preserved discarded-reconstruction removal; current Rust/conformance coverage for malformed images, counters, independent bounds and rename recovery. |
| fn-183.3 / R4,R5 | Current namespace/volume conformance, lifecycle replay, callback refusal and all three named storage-evidence regressions; validate/architecture/lint and ownership documentation. |
| fn-182.1 / R1,R4 (preparation only) | Historical closure/init analysis and disposable reductions preserved; current closure test passes. Required matched baseline measurements remain open. |
| fn-182.2 / R2,R3 (preparation only) | Guest-only seams, current closure/cache/language-version/replay-preflight tests and refreshed driver/config/helper/dry run. No current module-size/performance/workload-adoption acceptance is claimed. |

fn-182 measured R4/R5 cells have not started. Native execution remains with
fn-128/fn-149. Historical implementation commits and earlier handovers are
preserved; final local task commits will reference this one integrated record.

## Independent review, round 1

Fresh read-only Codex `gpt-6.1-sol:high` correctness/contracts/integration
draws all returned NEEDS_WORK, with no failed draws. This is the same model
family as the writer. Sidecars and merge plan are under
`.flow/review-fanout/cfaf90a3ecc5240c3e6c22232e2b41ee/`; merged receipt is
`/tmp/impl-review-receipt-657da2bc4466-branch-8af828021da1.json`.

The wrapper pinned committed `3b3d9b9ccd9f83d3ccc27f423e34967fa160f59d`,
excluding the staged repairs described above. The review is therefore not a
final-candidate SHIP: its twelve merged findings are retained honestly.
Provider wiring, cluster metadata transfer, dormant Go-owner compatibility,
compact metadata, cancellation provenance/capture, scenario clone, ordinary
divergence, choice-capacity projection and scenario default-limit findings
correspond to the already-tested staged repairs. The distinct missed-crash-
marker terminal-precedence case is being checked against the current code.
The measurement summarizer's host-sidecar glob defect is verified and being
fixed with a regression outside the frozen runtime manifest.

After checkpointing the actual tested source and the narrow script repair,
the required single-dispatch re-review will use the same receipt and base.
No source acceptance, Flow completion or final verdict is inferred from this
first round. The integration draw surfaced the distinct missed-boundary
case; the correctness draw surfaced the summarizer defect; duplicate findings
from all three draws were collapsed in the merge plan, not silently discarded.

## One-box watchdog and review fixes

The tested integrated source and script repair were checkpointed locally as
`86f25ef560`; no push or Flow completion occurred. The source manifest stayed
unchanged through the R6 gate's terminal result.

R6 exited 1 after 2191.505 s: shard passed in 146.44 s, mutable-state passed in
157.18 s, one-box failed in 1887.87 s. Its execution ran from
04:16:22.880583212Z to 04:46:23.027853320Z (1800.147270108 s), ending as
`watchdog_timeout` / `execution_timeout` under the original 30-minute bound.
Cold engine compilation finished at 04:17:39Z; the remaining roughly
28 minutes 44 seconds were execution, not source compilation. Both output
streams were empty and no execution/inspection metadata was captured. This
is not a valid R6 demonstration or a measured performance pass.

The timeout artifact is
`/tmp/fn180-temporal-compact-final3/onebox/artifacts/v1/campaign-20261016T041457.361326630Z-352a9b0aa719b342a3b1a2482ac2dae3/failures/sha256-c9917c1b3f64d2a65949d52e06bbec63`,
record `sha256:55638e6a82f27a65faf65f73dcce47dae2af7c0962ced9167087b31c4a51178e`.
All staging was cleaned and processes reaped; no retry or bound change was
made. Both successful lock-held cells' heap profiles opened with pprof.
Repeated full-state snapshot work is a source-level hypothesis; isolated
host-side timing/count diagnosis is required before choosing a repair.

The summarizer repair's three regression tests pass, including rejection of
a genuinely unsealed cell directory. The separate selected-marker capacity
defect was reproduced RED in an isolated worktree with actual helper crash
and stop cases, then fixed without scanning unrelated terminals or masking
prior protocol/replay errors. It is now integrated; focused tests pass in
`/tmp/gomad-fn181-marker-integrated.log`. Only `cluster/provider.go` and two
new test files changed. Its already-embedded cluster implementation identity
rotates; the helper, runtime and WASI model do not. The prior manifest is
historical for this new Go candidate, and affected evidence/review will be
refreshed rather than relabeled.

Integrated marker regressions passed (cluster 0.009 s, backend 10.890 s).
Canonical `make lint-code-fast GOLANGCI_LINT_BASE_REV=HEAD
GOLANGCI_LINT=/tmp/gomad-host-tools/golangci-lint
ERRORTYPE=/tmp/gomad-host-tools/errortype GOLANGCI_LINT_FIX=false` exited 0,
with no issues; log `/tmp/gomad-fn181-marker-lint-fast-linux.log`. An initial
attempt using environment-only overrides selected the repository's Mach-O
binary in recursive Make and failed before lint; it supplies no lint result.

The selected-marker repair is committed as `46ff86c0e3`. Its fresh maintained
manifest is `/tmp/fn181-marker-operator-20261016-txi40z/source.sha256`, SHA-256
`025f8d1579d7e48b5c8f3f149f1522beb06282cbbbb3b4fc6df04593c082a4e9`;
all 1,994 entries passed before/after checks. The final installed operator
matrix passed, exit 0 / 43.075 s (`matrix.log` in that same root): actual
SIGKILL/resume matched the 24-outcome baseline (18.92 s), and detached durable,
canary and network replay plus 18 tamper controls passed (24.15 s).
Installation identities are
`bdce8cb0f51265333be7da0b13e699b3be51450f321ee232d62768f97b388ccf`
and `dd48129ac2aa1312ca7e000a143b073b567357daf742448d6c36cbca1a00cbeb`.
The same helper/model remain unchanged. Fresh real source-free standalone
and restarted-cluster inspection passed, exit 0 / 40.944 s:
`/tmp/gomad-fn181-marker-retained-inspection.log`. Full cluster-package
coverage and the required single-dispatch re-review are still pending.

Full cluster-package coverage subsequently passed, exit 0 / 139.782 s:
`/tmp/gomad-fn181-marker-full-cluster.log`. The single-dispatch re-review of
committed `46ff86c0e3` returned NEEDS_WORK: findings 1–9 and 11–12 are fixed;
finding 10 remains for a selected `unsupported` terminal before a lifecycle
marker (the first repair preserved capacity only). The review confirmed all
fn-180 source R1–R5 and fn-183 source R1–R3, while explicitly preserving R6,
fn-182 empirical and changed-candidate qualification gaps. No new blocking
source findings were introduced. The same receipt/session is retained; the
selected unsupported case and its scenario error mapping are being repaired
and regression-tested in isolation before the next committed re-review.

Isolated snapshot diagnosis first measured 10,000 actual 25-file schema
namespace/recovery serializations and hashes in 0.328 s (55,641-byte canonical
state). That no-volume synthetic case does not explain the real timeout and
is not workload performance acceptance. A short host-instrumented run using
the preserved real module and original inputs will distinguish growing
storage/network/World costs before any production repair is selected.

The exact retained real module/original-input diagnostic ended at its separate
three-minute bound and reaped cleanly. Host-only counters recorded 12,180
snapshots / 81.252 s, of which World serialization consumed 71.043 s (87.4%)
and processed 24.939 GB cumulatively, with a 5.166 MB largest materialization.
Network evidence took 9.614 s / 7.957 GB; namespace canonicalization took
0.482 s, with no attached-volume/recovery/log cost. Snapshot cost grew from
roughly 0.44 ms early to 10.72 ms in the last interval. Among 4,773 decision
boundaries, 4,240 combined World commitments were unchanged (88.8%).
Evidence is `/tmp/gomad-snapshot-profile-704750.jsonl` and
`/tmp/gomad-fn180-snapshot-diagnostic-once.log`; actual startup was observed,
but no completion or acceptance pass is claimed. The isolated diagnostic
helper's timing feature and temporary diagnostic-cluster admission are not
production changes. The first proposed repair removes World materialization
at handoffs where it is overwritten before any consumer, while preserving
all authenticated namespace/network/peer cuts. An unchanged-commitment cache
requires separate authoritative-mutation and exact-byte/hash proof before use.

The remaining classification repair now preserves all selected reaped
capacity/unsupported/divergence/invalid/infrastructure terminals before a
missing lifecycle marker. Prior node errors stay authoritative; ordinary
exit/trap, unknown, unrelated and unreaped cases retain refusal. Scenario
projection also preserves unsupported/divergence boundary kinds and typed
runtime replay divergence. A genuine WASI CPU-clock import reproduced the
unsupported crash/stop defect RED (7.601 s); complete isolated controls passed
in 32.895 s. Integrated main-tree controls passed, exit 0 / backend 18.591 s,
in `/tmp/gomad-fn181-classified-marker-integrated.log`.

The new WASM-only fixture has an explicit host-lint inventory entry, with
near-miss sibling refusal and routing tests. Its absence was caught by the
canonical lint preflight rather than bypassed. Full integrated lint-tool
tests passed in 5.859 s (`/tmp/gomad-fn181-classified-marker-linttool.log`), and
canonical Linux-overridden fast lint exited 0 / no issues
(`/tmp/gomad-fn181-classified-marker-lint-fast-final.log`). Source and fixture
are ready for the next committed single-dispatch review; no task closure is
claimed yet.

The first isolated snapshot repair passed its RED/GREEN exact-byte/hash
regression and the second three-minute diagnostic. World materializations
now match decision boundaries (6,367 materializations / 6,366 boundaries),
rather than all 19,612 handoff snapshots. At nearly matched progress around
4,375 boundaries, cumulative World materialization fell from about 16.0 GB
to 3.9 GB. The bounded run reached 6,366 boundaries versus 4,773 previously;
host load/cold compilation prevent treating that as accepted workload speedup.
It still hit its separate diagnostic bound, with World serialization
40.127 s / 17.247 GB and combined hashing 10.439 s / 19.915 GB. Unchanged
combined commitments accounted for 5,674 / 6,366 boundaries (89.1%). Logs are
`/tmp/gomad-fn180-snapshot-defer-diagnostic.log` and
`/tmp/gomad-snapshot-profile-726614.jsonl`.

The next isolated repair is limited to caching the combined 32-byte World
commitment under the authoritative World mutation revision and a freshly
computed complete network commitment. No network evidence cache, wire-format
change, relaxed bound or weaker authentication is authorized. It requires
cached/uncached differential and invalidation coverage before integration.

## Reviewed classification candidate

Committed `18882a5f2a` received SHIP from the same read-only Codex reviewer
session/receipt after all twelve prior findings were fixed. No introduced,
pre-existing or unaddressed findings remain. This is source acceptance only;
the reviewer explicitly left R6, current broad qualification reconciliation
and empirical fn-182 acceptance open.

The final classification-candidate installed matrix passed, exit 0 / 42.063 s,
under `/tmp/fn181-classification-operator-20261016-7jBRX1`: SIGKILL/resume
matched 24 outcomes (18.47 s), and detached durable/canary/network replay plus
18 tamper controls passed (23.58 s). The 2,001-file manifest, including the
new fixture and lint-tool sources, passed before/after checks; SHA-256
`b69b9ef9d03c337a6ee47f97cc7bafc6332dd22f4ee9e2f64f17d8f080aedb4c`.
Installation identities are
`3b9c87ade6bb0226da7ea727633a0ac9983c95ddfe28fcef46c3ad72241c7727`
and `26f5ca57ad4fca545dffa347fe406d83cf16e28447459f4ece23cd22fb3883d4`.
Helper/model remain `a8dfd7fe…` / `d3aacafd…`. Root go.mod and go.sum retain
their original qualification digests `1c33f45a…` and `0b474c83…`.

All five explicitly retained fn-183 lifecycle/callback/storage-evidence
regressions passed again on this candidate:
`/tmp/gomad-fn183-current-named-regressions.log` (cluster 14.116 s; see the
same log for WASI). Broad affected backend/cluster/command/installation
packages and a focused portable CLI/runner/campaign acceptance reconciliation
are running before Flow task completion. The original full WASM, Rust and
runtime-qualification records remain bound to their own manifest/helper;
unchanged lower-level inputs are distinguished from changed scenario routing.

The owner moved the final queue to fn-184, then fn-187, then fn-185. Commit
`99998d6673` updates MILESTONES and Flow coordination/dependencies; profiling
will extend the post-fn-187 measurement command. This documentation change
alters no helper, runtime, qualification threshold or model semantics.

The broad affected Go packages now pass on the classification candidate:
backend 417.769 s, cluster 176.977 s, command 91.621 s and installation
0.046 s, exit 0; `/tmp/gomad-fn181-classified-full-affected.log`.
The current 2,001-file manifest passed again afterward. Only Flow/milestone
documentation changed during this gate; no maintained production/test source
or helper changed. The focused portable CLI/runner/campaign/target gate also
passed all 16 selected top-level tests with no skips:
`/tmp/fn181-portable-closure-20261016-305VN1/portable.log` (CLI 0.013 s,
runner 0.486 s, campaign 0.202 s, target 0.002 s). It includes typed scenario
admission, cloning, retained provenance, ordered commit/failure policy and
frozen-plan/journal recovery controls, and executes no deferred native runtime.

The lower-level Rust, WASI, model/oracle, generator and runtime inputs match
the earlier full qualification manifest; changed scenario routing and terminal
projection have the current broad/focused gates above. The final independent
source review at `18882a5f2a` is SHIP. Flow fn-181.1 through .4 and fn-183.1
through .2 are done in separate local task commits. Final spec qualification
tasks and completion reviews are next. Full R6 and fn-182 measurements remain
open; the new six-file Rust snapshot repair is still isolated and has not
replaced this qualified helper or main-tree source.

Both final spec-completion reviews returned SHIP and Flow now reports fn-181
and fn-183 done, with every task done. Reviews used fresh read-only
`gpt-6.1-sol:high` contexts, the same GPT family as the writer. The fn-181
review covers R1–R6 with no gaps (`/tmp/gomad-fn181-completion-review.json`,
session `01a14325-90bb-7ae1-a52d-0e205426055d`); fn-183 covers R1–R5 with no
gaps (`/tmp/gomad-fn183-completion-review.json`, session
`01a14325-b2c9-7513-a3bf-2dfe2f3f75c9`). Sandbox write restrictions prevented
their fresh Go/Cargo test attempts; they relied on inspected source and
committed execution evidence, with manifest/helper verification. No new
runtime test pass is attributed to those reviews. Historical and future
helper candidates remain distinct.

The six-file World snapshot repair is committed separately as `e57d1364e2`
in `/tmp/gomad-fn180-snapshot-production-20261016`, ready for main integration.
Its independent correctness/contracts/integration draws all returned SHIP,
with no findings or failed draws. Final receipt is
`/tmp/gomad-fn180-snapshot-impl-review.json`; sidecars and zero-finding merge
plan are in that worktree's `.flow/review-fanout/1b5b30359c8d194f743958dbfaffb897/`.
The clean patch excludes diagnostic timers, temporary feature admission and
bench/reproduction code. It passed 383 Rust tests / two existing ignores,
both clippy configurations/builds and source-free standalone/restarted-cluster
inspection in 41.359 s. New production helper is
`/tmp/gomad-fn180-snapshot-production-helper`, SHA-256
`2379a19f65fb397c61b303f773cca4f3b7665e2417fecb167a6706fd1eb868b9`.
Its actual R6 and broad qualification remain pending; earlier `a8df` results
do not qualify this new helper. Fresh network evidence remains uncached.

The reviewed six-file repair is now integrated on main as `90c1230140`.
The complete `wasmhost` tree matches reviewed isolated commit `e57d1364e2`
byte for byte. The new qualification freezes 2,003 tracked source files,
including root module files, under
`/tmp/gomad-fn180-snapshot-qualification-agrcID/source.sha256` (SHA-256
`974d426e38ec887a7dfcb4ee2f924320549100c81edc3d03ede15d43bd9bcd52`).
Root re-entry verification passed all entries. Production helper is `2379a19f…`;
the conformance helper remains `26bbdfa2…`. Maintained source stays frozen
through the new full WASM, runtime qualification and actual Temporal gates.

The new focused all-35-decision inspection gate passed, exit 0, 12.745 s:
`/tmp/gomad-fn180-snapshot-qualification-agrcID/live-inspection.log`.
Mandatory runtime qualification first refused the relative-path source manifest
before recording (make exit 2); that setup failure is retained as
`runtime-qualification-relative-manifest.log` in the same directory. The
absolute-path manifest has identical paths/digests and passed all 2,003 checks:
`source-absolute.sha256`, SHA-256
`d06e8dc26bb2df17f8556678629a9c13cf60230d058d28f199809a3af518e0e6`.
The unchanged mandatory 32×25-recording / 32-exact-replay recipe is running
against that manifest. Full `make -C tools/gomad test-wasm` also runs with the
private production and conformance helpers, `GOFLAGS=-p=2` and pinned stock
Go 1.27.1. Its log is `test-wasm.log` in that directory. Portable generated
validation passed (make exit 0), recorded in `validate.log`; this supplies no
native runtime qualification.

Actual Temporal acceptance is running unchanged at
`/tmp/fn180-temporal-snapshot-final4.log`. The two lock-held production snapshot
cells have retained their failures and reached source-free inspection. The
unchanged one-box workflow remains live; no success is claimed before both
its workflow-completion and intentional-canary witnesses, exact replay and
inspection pass within the original bounds. These parallel correctness gates
are not a performance measurement window. No fn-180 task is completed yet.

New-helper mandatory runtime qualification is now terminal PASS, make exit 0:
32 seeds × 25 fresh recordings (800), 32 fresh exact replays, aggregate exit 0
(0.867 s), 8,150 trace records. Diversity remains 30 choice hashes, 32 output
hashes, map 32, run queue 18, timer callbacks 2, canary 2, outcomes 0 and 17.
Every report binds `2379a19f…`, source manifest `d06e8dc2…`, guest module
`97a74559…`, runtime `7c220e2c…` and model `d3aacafd…`. The absolute manifest
and helper hashes were unchanged afterward; root independently reran the
2,003-file source verification, exit 0. Exact commands and numeric results
are in `runtime-acceptance.json` beside `runtime-qualification.log` in the
new qualification directory. The original relative-manifest setup failure
remains preserved. Full WASM and actual one-box R6 are still live, not passes.

The new full `make -C tools/gomad test-wasm` is now terminal PASS, exit 0
(root session `99426`). Its projection/conformance and complete WASM Go package
gates passed with the new private helpers; backend 304.792 s, cluster
146.375 s, command 68.436 s, qualification 3.204 s and toolchain 0.847 s.
The dedicated opt-in Temporal demonstration is separate and remains live.
Root reran the frozen 2,003-file absolute source check after the full suite,
exit 0. Raw output remains `test-wasm.log` in the new qualification directory.
These parallel gate durations are not accepted performance comparisons.

## Task closure on the snapshot candidate

Current full-WASM output verifies all 35 choice stops, dormant/active probes,
registered dump panic/truncation/GC handling, witness-only allocation divergence
with diagnostics off, fuel pressure, nonyielding callback cleanup, source-free
standalone/restarted-incarnation inspection, strict prefix/registry/tamper
controls and bounded scan/conditional-bisect controls. The real retained pair
passed in the broad suite (31.99 s and 16.14 s); heap pprof parsing passed.
The mandatory new-helper runtime gate above is complete. Ordinary and
`test_dep,gomad` no-op `common/gomaddebug` tests each passed again, exit 0,
without native runtime execution. Only the six reviewed Rust files differ
from the prior Go/lint-qualified `18882a5f2a` candidate; their complete tree
matches reviewed `e57d1364e2` and has passing release/clippy/fmt evidence.

This proves fn-180.1/R1,R4; .2/R1,R4; .3/R2,R3,R4; .4/R5 independently of
the still-running .5/R6 one-box demonstration. Their task receipts reference
this handover, current full-suite/runtime logs and the two SHIP source-review
receipts. Reviews were fresh read-only `gpt-6.1-sol:high`, the same GPT family
as the writer. No fn-180 spec-completion verdict is claimed. Parent completion
review and .5 remain gated on genuine R6. Native execution, push, PR and CI
remain deferred or unauthorized as before.

## Actual Temporal final4 failure and next repair

Final4 is terminal exit 1, total 1,323.753 s. Both lock-held production
cells passed (shard 106.26 s, mutable-state 109.50 s), including source-free
replay/inspection, probes and readable pprof. One-box failed after 1,107.97 s
at `runner.Explore`: `target_supervision: backend infrastructure: cluster
network listener snapshot is invalid or unordered`. This is not the previous
watchdog. No workflow-completion or canary witness is proven: its result was
not retained before validation failed. The prepared module/provenance and
zero-byte partial output heads remain under
`/tmp/fn180-temporal-snapshot-final4/onebox/artifacts/v1/campaign-20261016T052842.136330156Z-282a05d0ba13bfa8a7da108576721ebb`.
Raw log remains `/tmp/fn180-temporal-snapshot-final4.log`. All staging was
cleaned, helpers reaped and the 2,003-file source postcheck passed, exit 0.

Read-only diagnosis found a definite consumer comparator mismatch: native,
generated Go and Rust network snapshots order listener endpoints by node,
incarnation, address, then port; `model/cluster/record.go` omits address.
A canonical earlier address with a larger port, or equal ports on distinct
addresses, can therefore be rejected by the cluster validator. The actual
offending pair from final4 is unavailable and is not claimed to equal the
minimal reproducer. A narrowly scoped test-first consumer-order repair is
authorized under .5, preserving canonical bytes, digests and bounds. The
freeze is released only for that repair, not a helper rewrite or gate waiver.

The separate read-only network freshness audit rejects speculative caching
on existing counters: registration changes serialized incarnation state
without a history/readiness increment, reads change byte occupancy without
history, and readiness queries can execute pending operations. Complete
network evidence also depends on endpoint log and its separate World.
No new networking cache, measurement or optimization is accepted from that
audit. The current failure is addressed first; prior runtime/full-WASM results
remain tied to their original frozen inputs and helper.

The two-file consumer fix is now committed as `a0c6feb92e`: validation reuses
the already-authoritative `wire.NetworkEndpointKey` and removes the obsolete
private comparator. Authenticated regression RED is retained at
`/tmp/fn180-listener-order-red-authenticated.log`; GREEN covers fifteen
canonical/reversed/duplicate subcases plus existing invalid-port tests,
including real generated-Go admission and cluster encode/decode. Full ordinary
model/cluster tests passed (worker 0.082 s, root rerun 0.072 s). Canonical
Linux fast lint passed, exit 0 / zero issues, at
`/tmp/fn180-listener-order-lint-fast.log`. No wire version, digest formula,
helper, runtime or WASI-model input changed. Shared cluster and scenario
implementation identities do rotate, so old installations are not relabeled.

New frozen 2,003-file manifest is
`/tmp/gomad-fn180-listener-qualification-e11mId/source.sha256`, SHA-256
`d6ef0211f125f6fb69f5acdb183487b369037eeb71684fc933a4610d19a031ef`.
Root verified it before the new affected-package gate (session `42881`,
`affected.log` in that directory). The full unchanged all-three-cell Temporal
gate is running once as session `2325`, root/log
`/tmp/fn180-temporal-listener-final5{,.log}`, with private helper `2379a19f…`
and original execution/campaign/test bounds. Maintained source is frozen again;
no acceptance is claimed for this live R6 run. Independent narrow source
review follows the green affected gates; spec completion still awaits R6.

The full installed scenario matrix already passed on this listener candidate,
exit 0 / 43.028 s, at `/tmp/fn181-listener-order-operator-20261016-smufbE`.
SIGKILL after two committed outcomes resumed to the same 24-outcome baseline
(22.05 s). Detached durable success/canary/network replay and all eighteen
tamper controls passed (20.98 s). Source/helper checks passed before/after;
kill/resume installation is
`7e0060c6f310b2aa5ba3d04f59bb25b37bf0ddc781aba8fc573cf98382afac08`,
detached installation
`7438e5f740c6b54277f4b3cfb659c05b43e1de436c23fffea4e58e0335f08292`.
This rerun qualifies those named fixtures with the new cluster identities,
not the pending full Temporal workflow or native execution.

All five affected packages are now terminal PASS, exit 0: model/cluster
0.083 s, backend 321.862 s, WASM cluster 133.312 s, command 86.339 s and
installation 0.155 s (`affected.log`, root session `42881`). The source
manifest passed again afterward. The newly source-bound mandatory runtime
gate also passed: 800 fresh recordings, 32 exact replays, aggregate 1.021 s,
8,150 trace records, diversity 30 choice hashes / 32 outputs, map 32, run queue
18, timer callbacks 2, canary 2 and outcomes 0/17. All 32 reports bind the
new `d6ef0211…` manifest and unchanged helper `2379a19f…`. All-35-decision
inspection plus debug controls passed, exit 0 / 12.624 s. Exact commands and
postchecks are in `runtime-acceptance.json` in the listener qualification root.

The narrow listener-fix review is finalized SHIP, zero findings across
correctness, contracts and integration, with no failed draw. Receipt:
`/tmp/gomad-fn180-listener-order-impl-review.json`; sidecars and empty merge
plan: `.flow/review-fanout/5094cd41d69a60efb1e82f72db5b1c97/`. All three ran
fresh read-only `gpt-6.1-sol:high`, the writer's GPT family. They inspected
source and retained test results; no fresh reviewer test execution is claimed.
They explicitly keep full R6 open and do not claim the missing final4 pair
or workflow witness. The original final5 session `2325` is still live: both
lock-held cells passed again, and one-box recording helper PID `1148154`
started at 05:59:15Z, finished cold engine compilation at 06:00:30Z and has
not yet supplied a verified workflow/canary artifact. Original bounds remain.

Fn-182 preparation is refreshed for the rotated cluster/scenario identities
at `/tmp/fn182-listener-preparation-20261016-aH4Z8O`; the current handover has
the new driver hash and command. Its 16-cell dry run passed, but timed cells
remain unlaunched and `/tmp/fn182-listener-r4-20261016` remains absent.
