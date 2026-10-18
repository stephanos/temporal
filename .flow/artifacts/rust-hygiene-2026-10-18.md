# Rust hygiene and WASM gate checkpoint

Neither fn-187 nor fn-189 is complete. R3 remains in its worker checkout and is
not integrated. All qualification budgets and final acceptance gates remain intact.

## Latest continuation checkpoint

### Approved namespace-readiness qualification — current execution

The owner approved register-namespace, verify real standby readiness, then start
workflow. Flow records the narrow exception; upstream validation, all history,
durability, canary and capacity/resource assertions remain unchanged. Both
fixture copies now implement that ordering. The new29-case predicate root and
existing replication-readiness root pass after intended RED, as recorded below.

Primary HEAD is now `60e4dc6ca1` (an externally created checkpoint, not a commit
by this continuation). Rehashing the previous5393-file source roster finds
exactly two source changes, the approved recovery/replication fixture files.
New manifest is
`/home/agent/wt/evidence/fn189/namespace-readiness-2026-10-18/qualification/source.sha256`,
SHA256 `4fb14d40ced978f5dbd046f17a18a6372bce83041990afeeee9b5e2795c2b661`.
Fresh actual independent-cluster success then canary qualification is live under
owned session5455, with `run-cluster.sh`, `cluster-success.log` and
`cluster-canary.log` in that directory. The success cell has now passed in783.99s,
with its child process terminal exit0 and `cluster-success.exit`0. Its retained
result is `cluster/wasm-qualifications/v1/result-f4065475a18ae1118fad022c3c0802492160365d36cdbca05531ec2e3c22ed98.json`.
Qualified/deterministic/target-success are true; both recorded executions have
attempted exact replays with match=true. All20 samples are present. Budget passes
with zero violations, helper peak3379494912 bytes and operator peak1143832576,
below its recommendation. Four pre/post-crash observations and history/convergence
oracles pass. Actual module is `sha256:b1ac4d230191fa3a52623f05aa165a2625816303d7bc46f76c8402790c5283ba`.
Canary is now live in a fresh process with the same module and budgets, warm
preparation2.682811418s. The launcher uses new staged fixture copies,
unchanged detached request/broker inputs and budgets, the stock standalone Go,
unchanged3700def4… helper and an8GiB pre-helper safety observation. It is the
sole compiler/runtime owner. Canary and the complete ten-cell corpus are not
claimed passed. The earlier failures and diagnostic captures remain intact.
`complete-workload-corpus.sh` is prepared and syntax checked but unexecuted. It
requires both zero-exit cluster cells, then runs the other eight exact Makefile
cells in separate processes on this same freeze and invokes the unchanged
mandatory ten-cell aggregate. No old-source partial results are imported.

The paired default Rust full-coverage owner61194 is terminal exit0, baseline352
passed/2ignored and candidate201passed/2ignored. Both strict exports pass with
inventoried external diagnostics. Remaining arms/comparisons and other feature
modes still gate R3; no conservation certificate or original deletion is claimed.
The staged R3 patch is still0a343ce3… and its sources remain unintegrated.

The initial two source-audit dispatches failed before reading a diff because
their shell ignored `workdir`; neither issued a clean verdict. Fresh-context
retries used explicit repository paths. Their scoped same-family GPT reports
follow verbatim; neither supplies final task acceptance or runtime qualification.

### Correctness axis — namespace readiness

## Quality Audit — Correctness axis: namespace readiness delta

- Files changed: 2 · Critical 0 · Should Fix 0 · Consider 1.
- No blocking correctness finding. Live qualification remains with the conductor.

### Consider

- **tools/gomad/internal/backends/wasm/wasi/integration_test.go:3517** (Conf 100): The recurring source gate selects only the older readiness test, omitting the new 29 namespace controls — extend its `-run` selector to include `TestReplicatedNamespaceReadiness`.

### Evidence

- **Walked:** registration → validated standby frontend `:7233` → same ID/REGISTERED/global/exact replicas → workflow creation. `DescribeNamespace` reads standby persistence without cross-cluster forwarding; RPC/polling contexts share the 90-second bound.
- **Executed:** scoped diff, copy comparisons and retained-log reads. Both copies match; retained logs show the undefined-predicate failure followed by both readiness tests passing. No builds, tests, runtime operations or writes performed.

### Preservation

- **Walked:** local recovery registration/start ordering and existing five-event, durable-result and observation assertions remain intact. Reopen, oracle, canary and resource-budget callers are unchanged.

### Test Budget

- Approximately 80 control-test lines excluding initial fixture setup versus 43 helper/caller/import lines; below 2:1. Existing assertions were preserved.

Correctness axis:1 Consider; no blocking finding. The recurring selector will be
expanded after the current source-frozen runtime owner terminates.

### Standards axis — namespace readiness

## Quality Audit — Standards axis: namespace readiness

### Summary

- Files changed: 2 · Should Fix 0 · Consider 0 · Blocking: none possible (standards axis).
- No actionable standards findings in the diff against `60e4dc6ca1539f78b822ea31ba0dd7c36bef1243`.

### What's Good

- **tools/gomad/internal/backends/wasm/testdata/temporal_broker/replication_test.go:117** — Walked: private predicate and bounded `await.Require` keep readiness checks local; named table cases reuse a cloned response.
- **tools/gomad/internal/backends/wasm/testdata/temporal_storage/recovery_test.go:142** — Walked: extracting registration creates the required ordering seam without widening public APIs or retaining a second workflow-start path.

### Scope

- Executed: scoped diff and line counts; neither file crosses 1,000 lines.
- No builds, tests, runtime qualification, or writes performed.

Standards axis:0 findings; no worst tier.

### Helper publication continuation — current source freeze

This is the current checkpoint; live-handle and paired-phase descriptions below
are historical where superseded here. Neither spec is complete. The ordinary
canonical full gate passed on helper `098add07…`, then runtime session73782
terminated exit2 before the first guest: Cargo's fresh default build published
a helper with link count2, correctly refused by the unchanged execution-input
ownership check. This is not a determinism failure or a runtime seed pass.
The failure remains in
`/home/agent/wt/evidence/fn189/rust199-full-gate-identity-fixed-2026-10-18/runtime.log`
and `runtime.exit`. The rebuilt helper identity differs from that earlier full
gate, so a fresh full gate is required, not a silent reuse of that pass.

The actual production Make recipe now always copies through a fresh temporary
inode, verifies bytes, sets0755 and atomically renames the result. Default target
and already-matching custom outputs no longer bypass normalization. No ownership
check, qualification threshold or budget changed. New
`tools/gomad/internal/backends/wasm/helper_build_test.go` executes the actual Make
recipe against a controlled external Cargo-output fixture. Intended RED catches
default count2 and matching-custom count3; GREEN passes default/custom/matching
cases through the real hostfs bounded read and verifies permissions. Logs are
`helper-publication-red.log` and `helper-publication-green.log` in the failed
qualification directory above. Primary/R3 Make and test copies match exactly.

Actual canonical helper build now passes; published helper is regular,0755,
link count1, SHA256
`3700def42eec69623251d87e97ba0ddde41906863569edc17bab93bb95344525`.
Fresh primary full gate PASSED, observed owned session60237 exit0, with log
`/home/agent/wt/evidence/fn189/rust199-helper-publication-fixed-2026-10-18/test-wasm.log`.
Log SHA256 is `3dd27f156d3cf14f7f546e29acc82783b9d2cf89387c432553bb69bae97a9fda`;
the WASI package passes in390.460s, including100 fresh-guest output/transcript
equalities and the unchanged ownership/identity/capacity controls. The actual
helper hash remains3700def4…; guest-test logs record that same identity.
The guarded fresh32-seed runtime run PASSED, observed owned session25163 exit0,
using `run-runtime.sh`/`runtime.log` in that same directory. All32 reports retain
25 recordings and one exact replay:800 recordings and32 replays. The aggregate
passes, with30 choice digests,32 outputs and both canary outcomes. The source
checks before and after pass for all5393 entries. Runtime log SHA256 is
`6cd1bde84dabd614e1e86a33b1dd5368b2105c15be958ff41545fe97ac25d7a2`.
The initial workload run, session60324, terminated exit2 after both controlled
cells passed. Cooperative admission correctly refused the module-cache compiler
distribution. No code or threshold was changed: the existing standalone stock
Go at `tools/gomad/.toolchain/stock/2198cf19890232fc/go/bin/go` is a regular,
single-link0755 executable outside the module cache, version1.27.1, with the
same authenticated executable SHA256 `1675694e…`. The failed partial attempt
and `workloads.exit` remain intact.

A fresh canonical ten-cell attempt terminated under owned session69652, exit2,
using `run-workloads-stock-distribution.sh`, `prepare-inputs-stock-distribution.log`
and `workloads-stock-distribution.log` in that same evidence directory. Results
go to a separate `workloads-stock-distribution/` tree, with new inputs and
byte-identical staged fixture copies. All four fixture cells, both durable-SQLite
cells and both onebox cells have passed:8of10. Onebox success/canary take274.81s/
126.15s. The first multi-instance cluster cell failed in369.80s with
`finish simulation: WASI capacity: network-v2-evidence-bytes`. Cluster canary and
the mandatory aggregate did not execute. No ten-cell/aggregate pass is claimed.
The actual failure is the128MiB reconstructed network-document limit; missing
accepted samples cause secondary resource-accounting errors, not proof of an RSS
budget overrun. Empty executions do not establish whether guests committed.
Read-only diagnosis confirmed each document combines per-instance operations and
expanded World with the cumulative shared network Record. Which component
dominates remains unmeasured. No source, fixture, oracle or budget was changed.
All previously owned runtime/compiler handles are terminal. A separately labeled
nonacceptance diagnostic reproduction is the next investigation step.
The first actual diagnostic terminated under owned session16579, exit1, with files in
`/home/agent/wt/evidence/fn189/cluster-network-capacity-diagnostic-2026-10-18/`.
The explicit Go build overlay changes only host-side capture before classification
and error-path component publication; production source and helper stay frozen.
`run.log` and `run2.log` retain two pre-execution launcher refusals (mandatory
intent without a manifest, then a missing relative-source `./` prefix).
`run3.log` records the same actual guest module SHA256 `5b26fdef…`, preparation
1m26.443787083s. The capture proves active reached bidirectional replication
readiness and emitted the correct five-event observation. Its terminal is
`capacity: network-v2-transcript-bytes`; the reconstructed-document capacity
masked that earlier failure. Operations occupy134217519 bytes, native Record
2212091 and expanded World4718357. Of471834 operations-log entries,460633 are
polls and456907 polls remain pending, through logical241.830977071s. Shared
Record duplication is not the dominant size cause. Exact original component
captures remain in `parts-run3/`, with raw network and stdout in
`terminal-3900236330/`. Pre/post source-manifest checks pass. A second provisional
run added only host-side stdout mirrors for both nodes to distinguish standby
progress from active waiting. That run, owned session93293, terminated exit1 in
187.51s, again with exactly the same component sizes and active observation.
Both nodes' stdout proves local and peer History readiness at logical32s;
standby emits no workflow observation before the capacity failure. Pre/post
source checks pass for all5393 entries, and the helper SHA remains3700def4….
No diagnostic timing or provisional result is qualification acceptance.

Offline HTTP/2/HPACK and generated-protobuf decoding of the original retained
operations provides a more specific upstream compatibility finding. All three
workflow tasks1048581/1048586/1048594 leave active at logical32.000000001s.
Standby acknowledges low watermark1048595 at32.100000001s. The namespace-create
message for the same namespace is not sent until32.863623988s. Exact filtered
evidence is `protocol-order.json`, SHA256
`fc0b589dc5d9f90196124c67e28ff972e61f7d7cbd05e3fef000a61173a55144`;
the full decoded packets/messages and diagnostic Go readers are retained beside
it. No malformed HTTP/2 or HPACK decode was observed; ordinary GOAWAY frames
remain represented rather than treated as parse errors.

Upstream ancestor `f2b571e2eee8a9143fbc4711f3db8b5a37146b3b` added default-true
`ValidateReplicationTaskSourceCluster`. At
`service/history/replication/executable_task.go:890`, a missing namespace now
returns apply=false without eager namespace synchronization. The versioned
transition executor treats apply=false as successful skip. This matches the
observed task acknowledgment before namespace propagation and subsequent absent
standby history; exact branch execution was not logged because the unchanged
fixture uses a no-op logger. Namespace/workflow ordering is the leading concrete
compatibility cause, not proven defective WASM routing or shared-record growth.
Independent read-only research verified the exact packet order, configuration
and executor branches, and the upstream
`TestGetNamespaceInfo_NamespaceNotFoundSkipsRefreshAndProcessing` explicitly
locks the new behavior. It found no namespace-validation override or eager
`GetNamespace` request in the retained traffic.
Disabling this upstream validation, increasing limits, weakening history oracles
or relabeling provisional output is not acceptable. A proper fixture change
would require namespace readiness on standby before creating workflow tasks, but
fn189's explicit no-test-rewriting/unchanged-fixture contract requires owner
direction before applying that change. No source or acceptance criterion has
been changed. All compiler/runtime handles are terminal.
Dedicated extra gates have not started. The selected-SQLite launcher now requires
the new attempt's zero-exit record and reads its result tree, not the failed
partial attempt. The launchers
(`run-runtime.sh`, `run-workloads.sh`, `run-extra.sh`) pass syntax validation and
require preceding zero-exit records; this is not qualification acceptance.
Its source manifest
has5393 absolute regular-file entries, including the untracked regression test
and module mise file, SHA256
`c3bcfd4b825e397e3c35dea1c112e933485f43dcd00d472c302ca08ae23812dd`.

All64 coverage-tooling tests still pass in `helper-publication-tooling-green.log`.
All17 tools,153 source bindings and15 selector records were checked. Fresh
paired source-only phases under `/home/agent/wt/evidence/fn187/r3-current/` are
`before-rust199-helper-publication-2026-10-18/default`, source pin
`b023222af555bce5c0643e4141ebd44c0efcdf089ec1a29c3be77209fd48b5ae`,
and `r3-rust199-helper-publication-2026-10-18/default`, source pin
`7c4b30874446661c0af86a871cc311a9ed3ff73dec6afd62523edd4a3b99fd50`.
Previous paired snapshots preserve their pre-fix Make bytes and are historical.
No new coverage object/profile/certificate exists. Staged R3 patch remains
`0a343ce3549c1b7ed42146caf23c7800f2cd48f9a60c78ebfd3954036833d299`.

After the namespace-ordering diagnosis, the frozen paired default Rust coverage
run is live under owned session61194. Both source-only phase verifiers passed
against their actual roots immediately before launch. The serial launcher is
`/home/agent/wt/evidence/fn187/r3-current/run-rust199-publication-pair.py`;
it builds baseline first, runs its complete original test roster and coverage
export, then builds/runs the candidate. Build JSON/stderr, command/exit records
and per-arm logs remain inside each phase above. Rust1.99.1, matching LLVM,
separate raw profiles and the private guarded Cargo target remain mandatory.
No coverage pass, deletion, integration, commit or task completion is claimed.
That paired run is now terminal exit0. The baseline full roster passes352 tests
with2 ignored; candidate passes201 with2 ignored. Both strict coverage exports
pass with their explicitly inventoried external hash-zero diagnostics. Corpus,
invariant, Go, cursor and final comparison arms remain required; no conservation
certificate is claimed. The staged R3 patch remains unchanged.

The owner subsequently approved the narrow namespace-readiness fixture change.
The exception is recorded through Flow in the fn189 spec. Intended RED is
retained in `/home/agent/wt/evidence/fn189/namespace-readiness-2026-10-18/readiness-red.log`
(owned session5080 terminal exit1, undefined readiness predicate only). The
scoped implementer prepared the registration/workflow split, bounded real
standby frontend readiness and29 predicate cases in both fixture copies.
Fresh staged GREEN verification passed under owned session31667, terminal exit0,
using `readiness-green.log` in that directory. Both the new readiness root and
existing replication-readiness root pass. No full changed-fixture qualification
pass is claimed. All owned compiler/runtime handles are terminal; the old cluster
gate remains red until a newly frozen candidate passes.

### Correctness axis — helper publication

No confirmed correctness findings in the bounded helper-publication fix.

**Walked:** fresh staging creates an independent inode; copy, chmod, comparison, and atomic rename are failure chained. The regression covers default, custom, and matching custom outputs through the production ownership check.

**Executed:** primary and R3 Makefile/test copies are byte-identical.

Tests and runtime gates were not rerun by this reviewer. No shipping verdict.

Correctness axis:0 findings; no worst tier.

### Standards axis — helper publication

## Quality Audit — Standards axis: helper publication

- Files reviewed: 2 · Should Fix 0 · Consider 0. No material standards findings.
- Evidence: cited. The Make recipe has one publication path; the Go test exercises default, custom, and matching-output cases through actual Make and the existing host filesystem API.
- Evidence: executed. Canonical and R3 copies match byte-for-byte.
- No builds, tests, runtime, LLVM, or writes performed.

Standards axis:0 findings; no worst tier. Scoped same-family Codex reviews,
not final task acceptance or an integration/shipping verdict.

### Paired Rust 1.99 tooling continuation

This subsection supersedes the export-policy and fresh-baseline tooling work
listed as pending below. Neither spec is complete; no original test was removed,
no R3 integration occurred, and its staged patch still hashes to `0a343ce3…`.

The exact external unwinder diagnostic disposition is implemented test-first.
`export-diagnostics-red.log` retains the original unknown-crate rejection;
`export-diagnostics-green.log` and `selection-pin-green.log` pass. Admission
requires the exact symbol, hash zero, unique registry mapping, package version
47.0.3, exact source path and source SHA. The existing five-crate policy is not
broadened. Negative controls reject other functions, versions, nonzero hashes,
local/absent/ambiguous mappings, changed/missing source, unexpected diagnostics
and incomplete summaries. The historical failed phase is not reclassified.
Its `full/unwinder-profdata-show.txt` and `unwinder-observation.json` retain the
nonzero hash/count683 observation, bound to six profile/object/export hashes.
Independent review checked those hashes and reproduced the complete297-record
inventory, with exactly one special disposition. Compiler causation is unproven.

Both comparison sides now require source/object pins, declared source roots and
their own LCOV normalization; neither a `before` prefix nor old fixed line numbers
provide an exemption. Cursor schema3 binds both phases and roots. Go baseline
full selection derives all65 original roots from the primary's actual four files.
Candidate full selects the reviewed union of functional/private/corpus roots,
requires every surviving original root, and rejects arbitrary full overrides.
Current candidate union is63 roots:61 WASI and2 network. Source, production
coverage and full/corpus-plus-invariant obligations remain independently checked.

Snapshots now bind the actual executed external tools/selectors, including their
frozen copies. A supplied selector must match its exact pinned path/hash. Intended
RED evidence is `external-tools-red.log` and `selection-pin-red.log`; final expanded
tooling suite is64PASS in `fresh-paired-tooling-final.log`. The worker's additional
red-green logs are `fresh-pair-red.log`, `fresh-cursor-red.log`,
`fresh-go-selection-red.log` and `fresh-pair-cursor-green-final.log`.
All17 inventory tools,152 source bindings and15 selector records were checked.

New paired phases under `/home/agent/wt/evidence/fn187/r3-current/` are
`before-rust199-paired-2026-10-18/default` (source pin
`b37a04a46cfb9e24ad9d888f42296099beff5b8744408fe933300d003993ba6b`)
and `r3-rust199-paired-2026-10-18/default` (source pin
`c1ffbcb20eb67691d7a922b378a630b2d4dfe4e62e99d3a1dcdd31c166d7da41`).
Both bind35 actual tool/selector files; baseline/candidate source inventories
contain856/2104 files. These phases have source pins only so far: no new object
or conservation certificate is claimed. Preserve all earlier phases unchanged.

The fresh primary canonical `test-wasm` gate PASSED, observed session12238 exit0;
log `/home/agent/wt/evidence/fn189/rust199-full-gate-identity-fixed-2026-10-18/test-wasm.log`,
SHA256 `e2a3ce7ed2e309bc9d83cb48cf3c6a024fab4e809f882b0c9a07cdc86f4d09f2`.
The WASI package passes in379.272s, including the corrected implementation identity.
Conformance, architecture/projection and all canonical packages pass. Opt-in
qualification/prototype skips in this ordinary gate remain separate obligations.
Its absolute regular-source qualification manifest contains5392 entries, SHA256
`f3814c8419889a0753acafe170f50a6b9b32cfb665868c2c57952455f1998340`,
including the new module-local mise file. All entries passed a fresh hash recheck.
The canonical32-seed runtime qualification is now live under owned session73782,
using `run-runtime.sh` and `runtime.log` beside that gate log. No runtime/compiler/
LLVM work may overlap this sole resource owner. Native,
profiling, ignored benchmarks, publishing and migration execution remain excluded.

Qualification launchers are retained beside that log as `run-runtime.sh`
and `run-workloads.sh`; both pass `bash -n`. Runtime has started; workloads have
not executed. They require
the preceding conductor-observed zero-exit record, fresh output roots and source
hash rechecks. Runtime keeps all32 seeds/25 recordings/one replay and enables the
existing Linux8GiB-available-memory admission guard. Workloads retain the canonical
ten cells and aggregate, staging byte-identical committed fixture files only.
The paired candidate phase's `go-production-source-tree-equality.json` independently
records exact equal hashes for all48 recursive production Go files: WASI24,
network6 and volume18, bound to both source pins. This proves source equality,
not coverage conservation. The narrower package-root-only receipt is retained too.

### Correctness axis

## Quality Audit — Correctness axis: paired fn-187 tooling checkpoint

- Reviewed 8 tooling modules and 8 test files directly; directory is Git-ignored.
- Critical 0 · Should Fix 0 · Consider 0. No confirmed correctness findings.

Evidence: **executed** all 64 Python tests, passing in 1.408s with bytecode writes disabled.

**Walked** source/tool/selector bindings, per-phase LCOV roots, full-only certification gates, exact Go statement identities, unchanged-context Rust mapping, cursor receipt validation, and the exact unwinder diagnostic exception.

The cursor proof supplies one identity mapping while full and corpus-plus-invariant coverage remain independently required.

Scope excludes live conservation execution, integration, task completion, and shipping. No source or retained-evidence changes.

Correctness axis:0 findings; no worst tier. Scoped checkpoint, not final impl-review.

### Standards axis

## Quality Audit — Standards axis: paired fn187 tooling

- Files reviewed: 12 · Should Fix 0 · Consider 0.
- No material standards findings. Reviewed the ignored files’ complete current contents; the scoped Git diff is empty because these files are untracked/ignored.
- Evidence: cited. No tests, builds, helpers, LLVM commands, or evidence mutations executed.

### What’s Good

- Source, selector, and diagnostic checks have shared owners in `phase_snapshot.py`, `compare.py`, and `coverage.py`.
- Mutation cases use named subtests, and tool mutation probes copy sources into temporary directories.
- All scoped files remain below 1,000 lines; largest is `coverage.py` at 329 lines.

Fresh-context reviewer used the inherited Codex session model; the conductor retains writer/reviewer model-family provenance.

Standards axis:0 findings; no worst tier. Writer/reviewer are same-family Codex;
these are source/tooling checkpoints, not a full task shipping verdict.

### Earlier sequence-adapter and full-gate checkpoint

This section supersedes the earlier live-handle and pending-adapter descriptions
below. The primary full WASM gate is terminal, exit2 after10m39s. Conformance,
projection and architecture checks passed; its sole reported failing root was
`TestImplementationIdentityBindsCompiledModelDigest`. The reviewed measurement
ownership move changed the primary's only two modified production WASI files,
`measurement.go` and `process.go`. Their contents and identity owner match R3.
The R3 baseline is older and also carries the already accepted addition of
measurement.go to identity inputs. Expected identity now records the actual
`sha256:9b087b968ee0054b2886b670934e37f0e4546939442d61fde69606ac056c4c99`;
the mutation/missing-source test now explicitly covers process.go. All four
identity roots pass in both checkouts. Logs are primary
`.flow/tmp/rust-hygiene-2026-10-18/identity-owner-after.log` and R3
`.flow/tmp/fn187-r3/identity-owner-after.log`. No fresh full-gate exit0,
runtime qualification or workload qualification is claimed.

The two read-sequence adapters are now implemented. With no-op adapters, both
negative-control roots failed for the intended reasons: changed classification
and changed storage maximum were accepted. See R3
`.flow/tmp/fn187-r3/sequence-adapters-red.log` (exit101). Actual adapters compare
full real observations, fresh cluster readers over a shared cursor/Broken, and
three storage reads on one descriptor including actual stored operation counts.
Existing EOF and storage-vector corpus roots replay the new cases as well.
Both original inline tests remain unchanged.

The first strict lint pass caught two test-code style issues, retained in
`sequence-adapters-clippy.log`. After correction, all six configurations and
formatting pass in `sequence-adapters-clippy-2.log`. Complete ordinary Rust
tests pass:201passed, zero failures, two ignored benchmarks, zero filtered,
with all407 helper cases and both originals retained. The seven target summaries
are140/4/3/7/21/18/8 passed. See `sequence-adapters-full-rust.log` in R3.
No warnings appear in that successful test log. All48 Python tooling checks pass
in `sequence-coverage-tooling.log`.

The two new negative controls are explicitly test machinery, not private
invariants. Rust selectors are now functional200/corpus28/diagnostic26/
invariants134/public2/machinery36, with held1/ignored2 unchanged. No new private
waiver was admitted. The staged patch remains unchanged.

A fresh Rust1.99.1 default coverage build completed successfully in4m59s with
zero warnings/errors in Cargo JSON and stderr. Its immutable phase is
`/home/agent/wt/evidence/fn187/r3-current/r3-rust199-sequence-conservation-2026-10-18/default/`.
The first package-clean attempt stopped because Cargo1.99 requires a root
CACHEDIR.TAG absent from this older private target. The directory was inspected:
it contains only Cargo release/platform output and rustc metadata, with an
existing platform-child cache tag. The owner added the standard tag to the exact
guarded target; no broader cleanup or protection bypass was used. The failed log
and command record are preserved as `clean-conformance-cachedir-missing.log` and
`commands-before-cache-tag.jsonl` before retry. The retry clean and fresh exact
conformance build pass, and ordinary helper restoration uses its frozen copy.

All seven frozen full-suite targets pass,201tests/2ignored, and profile merge
passes. Strict export then rejects one newly encountered external zero-hash
variant, among297 diagnostics: serde_json241/base6415/rustix10/wasmtime30 plus
one wasmtime_internal_unwinder. Functional/corpus/invariant arms did not start.
No conservation certificate is issued. Raw profiles, ordinary/conformance
objects, source/object pins and failed export remain intact. Owned sessions57822
and43422 are terminal; no compiler/test/LLVM handle remains live at this point.

The rejected symbol is
`_RNvNtNtCsbt0ASiPIg7r_26wasmtime_internal_unwinder4arch7aarch6417get_stack_pointer`.
It maps uniquely to registry package47.0.3 `src/arch/aarch64.rs:49`, source SHA256
`4daf28dcd10780235ff1216f82c6f21c1c8afd803c594a851277eca8f8e3eecd`.
The combined export reports FNDA683. llvm-profdata independently reports a real
record hash0x44b0d86e39910fc9/count683; the reported missing variant has hash0.
The existing five-crate admission remains unchanged and correctly rejects this
unrecognized crate. A filename-filter probe retained the same diagnostics and
was not substituted for the raw export. Any compatibility disposition must be
independently reviewed, exact to the function/version/source, tested against
local/ambiguous/nonzero/unknown variants, and recorded in a fresh tooling phase.

Read-only baseline research identified current primary after hygiene as the
faithful fresh pre-R3 baseline. It retains all256 original src test names and
all65 original Go roots; the sole absent integration/common name is an added
coverage-harness self-test. Cargo manifests, lock and toolchain are equal to R3.
Production Go bytes match across wasi24/network6/volume18 files, preserving exact
statement comparison. The known Rust conformance cursor seam remains the narrow
existing equivalence obligation. Do not reconstruct from the dirty old
before-guard checkout or reuse its1.94 profiles. Fresh paired phase roots,
executed-tool/selector binding, source normalization and cursor proof are needed;
the final before/after verifier must remain fail-closed. Old records stay
historical. Final full/functional/corpus/invariant/union and original-baseline
conservation remain required across supported configurations.

### Independent correctness followup

No confirmed correctness defects at confidence 75+ in this follow-up.

- Walked: `wasmhost/src/cluster/transport_input_corpus.rs:49` preserves fresh-reader lifecycle over one shared stream and `Broken`; compares actual classification, message, bytes and broken state.
- Walked: `wasmhost/src/storage_transport/vector_corpus.rs:63` reuses the descriptor across all three reads; compares actual failures and actual stored operation counts. Classification/maximum/count guards exercise execution, so they are not tautological.
- Walked: `wasi/identity_test.go:24` retains digest equality, adds mutation/missing-source controls, and updates the expected digest to `9b087b…`. Executed: `cmp` confirms matching primary/R3 measurement, process and identity sources/tests; fixture byte counts and wire hashes match their JSON declarations.

Baseline nuance: R3 also changes production `identity.go` versus its older HEAD, adding `measurement.go` to identity inputs; its current contents match primary.

Source-only checkpoint, same-family GPT review. No builds/tests/writes; no final coverage, conservation, R3 acceptance or shipping verdict.

## Tooling

Both checkouts now pin Rust 1.99.1 through `tools/gomad/mise.toml` and the crate's
`rust-toolchain.toml`, including rustfmt and Clippy. Live Makefile, CI and
measurement-admission pins agree. The old-toolchain admission regression still
uses Rust 1.94.1 as deliberately rejected evidence. Historical receipts retain
their actual compiler versions.

The Cargo policy enables Clippy all and pedantic, plus fallible_impl_from,
needless_collect, redundant_clone, dbg_macro, todo, unimplemented and
undocumented_unsafe_blocks. `lint-rust` checks formatting and all targets in six
feature configurations, with `-D warnings`; `lint-code` and WASM CI invoke it.
Exceptions are scoped and reasoned, not group-wide pedantic/restriction allowances.

Primary `make -C tools/gomad/internal/backends/wasm lint-rust` exits 0 across
default, conformance, diagnostic-profile, diagnostic-fuel-off, guest-profile and
guest-profile plus diagnostic-fuel-off. Evidence:
`.flow/tmp/rust-hygiene-2026-10-18/pedantic-primary-matrix-5.log`.
Earlier failed diagnostic passes are retained beside it. The separate R3 checkout
also passes all six configurations, including formatting and all targets:
`/home/agent/wt/wasm-fn187-functional/.flow/tmp/fn187-r3/pedantic-worker-matrix-4.log`.
That lane's failed diagnostic and compiler passes are retained alongside its log.
No post-hygiene Rust test, coverage,
resource or independent-review result is implied by Clippy.

Matching LLVM tools are installed: rustc 1.99.1 reports LLVM 23.1.1 and its
bundled llvm-cov reports `23.1.1-rust-1.99.1-stable`. The ordinary R3 full Rust
suite passes with one build job and one test thread, no coverage flags and
no ignored benchmarks executed: 199 passed, zero failures, two ignored manual
benchmarks. All three exact resource originals pass. Its first log is
`/home/agent/wt/wasm-fn187-functional/.flow/tmp/fn187-r3/rust-hygiene-default-tests-1.log`;
source/testdata hashes are retained in the adjacent `*-source.sha256` file.
Both corpus-regeneration variables, the residual-case filter and the RSS-child
marker were confirmed unset before tests began. Source hashes matched afterward.

The standards checkpoint found the R3 loader above 1,000 lines. Four unchanged
telemetry/profile projection functions and their prefix constant moved into
`tests/common/helper_corpus_projection.rs`; only visibility/import wiring changed.
The loader is now 998 lines. A byte comparison of the moved block, normalized
only for the two `pub(super)` visibility qualifiers, matched exactly. Test roots
and assertions were not changed. The complete ordinary Rust suite passed again
with the same 199/0/2 results after extraction:
`.flow/tmp/fn187-r3/rust-hygiene-default-tests-2.log` in R3. Its source hashes are
retained alongside the log. The six-configuration formatting/Clippy rerun also
exits 0 after extraction: `.flow/tmp/fn187-r3/pedantic-worker-matrix-5.log` in R3.
The standards followup below confirms the finding is resolved. Source/testdata
hash verification and both checkouts' `git diff --check` pass; the staged patch
hash remains unchanged. No final coverage or qualification is implied.

The World writer's signed time conversions were changed in its schema template,
then regenerated in both checkouts; generated Rust was not hand-edited. Existing
wire bits, admission order and capacity/refusal behavior are preservation targets.

## Architecture failure

The earlier primary full `test-wasm` run exited 2 because `wasi/measurement.go`
contained host supervision and reflection despite being a pure-model file.
Its retained gate log is
`/home/agent/wt/evidence/fn189/resource-resume-2026-10-18/test-wasm.log`.
Runtime and workload qualification did not follow that failed gate.

`MeasureTerminalExit` and its supervisor types now live in the existing
`wasi/process.go` owner. The pure historical converter uses the existing closed
output types and canonical JSON comparison instead of reflection, retaining
null-versus-empty distinctions and strict field admission. Collector provenance
now also binds `process.go`. No architecture exemption or import allowance was
added; the negative import fixture now explicitly includes `measurement.go`.

The focused architecture gate reproduced the failure on both required source
platforms, then passed after the fix. Focused measurement tests also pass,
including historical output shape/unknown-field controls, source existence and
current/old compiler admission. Logs:
`.flow/tmp/rust-hygiene-2026-10-18/architecture-before.log`,
`architecture-after.log` and `measure-owner-after.log` in the same directory.
This does not replace a fresh complete `test-wasm` or qualification run.

The same focused architecture/measurement selection also passes in R3:
`.flow/tmp/fn187-r3/go-hygiene-ownership-after.log` in that checkout.
Primary source-generation tests pass for Rust World schema/tag ordering and
generation/checking of every endpoint; see
`.flow/tmp/rust-hygiene-2026-10-18/generator-after-module.log`.
The first generator test invocation used a relative path across a nested Go
module and failed setup; it was corrected to the module's full import path.

## Preservation and remaining work

The R3 coverage runner now uses Rust 1.99.1 and matching LLVM 23.1.1. Its source
pin includes both toolchain files and the WASM Makefile, and those explicit inputs
are copied into the frozen snapshot. The runner clears inherited compiler flags,
wrappers, corpus filters, regeneration, seeds and the RSS-child marker. Regression
tests failed on the stale pin, omitted inputs and inherited overrides, then passed
after correction. All 48 tooling checks pass again; see R3's
`.flow/tmp/fn187-r3/coverage-toolchain-recheck.log`.

The inventory now binds 148 current source/configuration paths and 15 tools.
All 142 retained-test source hashes/lines are refreshed, with names, paths and
reasons unchanged. The separate host-concurrency classification is unchanged.
The 13 non-binding selector/reason files are unchanged; only the two source-binding
records' hashes rotate. All 178 manifest file hashes match their current files.
This is source inventory verification, not coverage conservation. Retained bodies
are not claimed byte-identical to their pre-hygiene versions. The spec's current
Quick commands now pin 1.99.1 through `flowctl spec set-plan`.

The R3 staged patch SHA256 remains
`0a343ce3549c1b7ed42146caf23c7800f2cd48f9a60c78ebfd3954036833d299`.
No staging, commits, integration, push or CI run was performed. The worker's
source/hash/line bindings and coverage freeze predate these edits and must be
rebound and reverified, not treated as current certificates. Original resource
test passes used the previous compiler and remain historical evidence.

The requested crate-reuse subagent's findings are recorded separately in
`rust-crate-reuse-review-2026-10-18.md`; no new dependency was adopted.
Native execution, fn-185 profiling and ignored benchmarks remain excluded.

The fresh primary helper rebuild/full WASM gate is running under owned host
session 68363. Logs and source hashes are retained under
`/home/agent/wt/evidence/fn189/rust199-full-gate-2026-10-18/`.
The first launch stopped before compilation because `/usr/bin/time` was absent;
its log remains. The corrected launch uses Bash timing, one Cargo build job and
one Rust test thread. The ordinary helper rebuilt successfully in 3m13s; the
full gate has proceeded to the separate conformance-helper build. No passing
full-gate result is claimed yet.

## Remaining inline-test replacement decision

The followup read-only research review recommends extending the existing
`cluster/transport_input_corpus.rs` and `storage_transport/vector_corpus.rs`
component corpus owners. These are component contracts, not helper end-to-end
tests. Neither remaining original is deleted or reclassified yet.

The cluster fixture must own the exact original oversized wire bytes and a
two-read sequence. Both reads construct fresh `TaggedInput` readers sharing
one cursor and one `Broken`. Compare capacity classification/error and absent
Broken, followed by exact stripped `next` frame bytes including newline. Input
must not be rebuilt from the production `FRAME_LIMIT`, which could move the
fixture boundary silently.

The storage fixture must own the original descriptor and framed context bytes,
including canonical key ordering, `/workspace`, capacity1024, limits1000/1000,
return_image=false, all400 original operations and their3000-character paths.
Validate one descriptor, then perform three reads with fresh cursors in the
original order: maximum1 refusal, missing-end refusal at18892800, and complete
success at18892800. Compare actual stored operation counts, not input counts;
retain the explicit context-size-greater-than1MiB assertion. The existing test
module can observe stored state without adding a production API.

Fixture loaders must decode, invoke existing production operations and compare
observations; case-name-specific scenario construction is not an acceptable
replacement. Preserve accepted fixture bytes and existing guards. Demonstrate
negative controls and assertion/coverage conservation with originals excluded
before removing either original. No new invariant waiver is needed.

The four proposed fixture files are now present in R3 under
`tests/testdata/cluster_transport_input/oversized-tagged-frame-followup.{json,wire}`
and `tests/testdata/storage_transport_vectors/pending-context-over-one-mib.{json,wire}`.
They are not loader-admitted or runtime-verified. The existing EOF corpus's
one-JSON roster is deliberately still unchanged; it will reject the proposed
second JSON until the read-sequence adapter and roster are implemented together.
Both inline originals remain unchanged. Do not claim the current worker suite
green based on the earlier 199-pass receipt.

Independent input diagnostics verified the cluster wire's 8388706 total bytes,
exact original padding and next-frame bytes. Its SHA256 is
`d0129c73ce964bf162e12b0e238b6c0f7425dce5d2708f87d346dee39bd0ce93`.
Storage wire SHA256 is
`615ecd1c6d532b01d7e0d6d0f7e23d612de1b33fef2c3f0228eefc2d7d6bbb76`;
decoding its20 frames yields1259101 bytes/400operations with the independently
researched original canonical payload SHA256
`75f2ecdbd473838ca2795841bf93c217a20286a0ac642f34bb57c9ca5e6fc2e8`.
The inventory binds all four additions, bringing source/configuration paths
to152. These diagnostics establish input equivalence only, not test replacement
or coverage conservation. Compiler/runtime work remains serialized behind the
live primary full gate.

The owner subsequently authorized completed-spec worktree cleanup. Flow confirms
fn-184 and all four tasks done; its two clean, merged, unused worktrees `fn184`
and `fn184-base` were removed with Git, followed by worktree pruning. Their ignored
helper binaries were copied and SHA256-verified under
`/home/agent/wt/evidence/fn184/retired-worktrees-2026-10-18/`; its README records
commits and recovery details. No branch or commit was deleted. Twenty worktrees
remain. The fn-189 baseline and all fn-185/fn-187 worktrees remain because their
specs are open; uncommitted evidence is preserved. Historical per-worktree reasons
remain in `fn187-lightweight-correctness-2026-10-18.md`, superseded only for those
two removed fn-184 checkouts.

## Periodic independent reviews

### Correctness axis

Correctness checkpoint: no findings at confidence 75+ in the scoped hygiene and measurement changes.

- Walked: `wasi/measurement.go:109` retains typed null/empty distinctions and strict unknown-key rejection; `internal/measure/regression_more_test.go:1591` covers those cases.
- Walked: `wasmhost/src/abi.rs:419` preserves signed wire bits. Checked narrowing remains behind buffer, file, port or fixed-count bounds, including corresponding R3 sites.
- Walked: request borrows and primary test predicate rewrites preserve behavior; `internal/measure/producers.go:4920` includes the relocated host owner in source provenance.

Scope limits: source-only checkpoint against `bcb72b2bc7`; R3 deletion conservation and full coverage remain unaudited here. No compiler, tests, guests, writes or repository mutations performed. Runtime results are claimed by the dispatcher. Same-family GPT review. No final acceptance or shipping verdict.

### Standards axis

## Quality Audit — Standards axis: Rust hygiene checkpoint

### Summary

- Files changed: 65 tracked primary files plus `mise.toml`; R3 counterpart inspected. Should Fix 1 · Consider 0 · Blocking: none possible (standards axis).
- Source-only, same-family GPT review. Compiler/test results are cited owner evidence; final acceptance, coverage conservation and qualification remain open.

### Should Fix

- **tools/gomad/internal/backends/wasm/wasmhost/tests/helper_corpus.rs:989** (Conf 100): R3 crosses the 1,000-line threshold, from 972 to 1,090 lines (executed: base numstat/current line-count calculation) — extract a cohesive loader responsibility into `tests/common`, preserving assertions and test registration.

### What’s Good

- Cited: `wasmhost/Cargo.toml:25` enables pedantic and selected additional lints; inspected exceptions name individual lints and reasons. No new blanket pedantic/restriction exemption or dependency change found.
- Executed: primary/R3 toolchain files and World template/output match. Cited: WASM `Makefile:11` checks six feature modes; CI invokes that gate before building.

### Limits

- No writes, staging, compiler, tests, Go, LLVM or guest execution performed. Toolchain pins agree at 1.99.1; “latest” availability was not independently checked.

Correctness axis: zero findings at confidence 75+; worst tier none.
Standards axis: one original finding; worst tier Should Fix; resolved by the
cohesive extraction and independently checked below. No final acceptance claimed.

## Quality Audit — Standards axis: extraction follow-up

### Summary

- Finding resolved · Should Fix 0 · Consider 0 · Blocking: none possible (standards axis).

Executed: R3 `tools/gomad/internal/backends/wasm/wasmhost/tests/helper_corpus.rs` is now 998 lines; the extracted module is 98 lines. The threshold crossing is resolved.

Walked: `helper_corpus.rs:54` wires the projection module and imports its two entry points. `tests/common/helper_corpus_projection.rs:3` owns the profile prefix and four related projection functions; only `project` and `split_profile` receive `pub(super)` visibility. No introduced standards issue found.

This remains a source-only, same-family GPT checkpoint. The ordinary-suite pass and unchanged registration count are cited owner evidence; the strict lint rerun and final acceptance remain separate gates.
