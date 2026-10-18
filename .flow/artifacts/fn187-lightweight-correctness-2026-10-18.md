# fn-187 lightweight correctness checkpoint

This checkpoint covers bounded functional correctness, preservation and documentation. It is not
a source freeze, coverage certificate, performance result or task completion.
The consolidated R3 source remains in `/home/agent/wt/wasm-fn187-functional`;
R3 source has not been integrated into the primary checkout.

Latest status: 405 helper inputs and 42 additions remain after the rejected
stale-reply experiment was archived. Following the diagnostic control correction,
two guarded default regenerations are byte-identical, all 18 Rust helper tests
pass, and the real Go consumer passes all 405 cases plus four guard roots with
zero skips. The public stale-refusal replacement is still unresolved; its
original test remains.

## Independently checked inputs

Baseline: `c0bf54df043249aaf092048cb094c2ba4f1f9ee3`.

- All 215 accepted helper fixtures remain byte-identical. The checker exited 0;
  its per-file SHA receipt is
  `/home/agent/wt/wasm-fn187-functional/.flow/tmp/fn187-r3/root-source-only-helper215-byte-check.json`.
- All 130 entries in the volume owner's
  `/home/agent/wt/wasm-fn187-r3-volume/.flow/tmp/r3-volume-precise-inputs-admitted.tsv`
  match the consolidated checkout (`sha256sum -c --quiet`, exit 0).
- The conductor independently reran strict preservation across 526 accepted
  JSON contracts: 0 changes, exit 0. Its distinct receipt is
  `/home/agent/wt/wasm-fn187-functional/.flow/tmp/fn187-r3/root-source-only-accepted526-current.json`;
  earlier receipts were not overwritten.
- At the pre-archive checkpoint, all 43 named additions in the closed manifest had corresponding helper JSON
  files, and the directory contains 406 JSON files. All three manifest-bound
  audit hashes match their current audit files. These checks exited 0 and prove
  input inventory, not 406 successful replays.
- The primary documentation describes corpus, regeneration, invariant-test
  and measurement ownership in `tools/gomad/ARCHITECTURE.md`, with a cross-link
  in `tools/gomad/README.md`. These additions were checked against the accepted
  loader, measurement command and Makefile. `git diff --check` exited 0;
  Flow validation reported 0 errors and 34 pre-existing warnings.

## Focused correctness check

The previous test executable predates the latest namespace RawValue comparator
fix, so reusing it would not verify current source. A fresh, single-job
incremental compile completed in 41.14 seconds using the existing private
cache. All four exact namespace/volume consumers passed with one test thread:
namespace conformance, binary transcript codec, image admission and volume
core. The test summary is 4 passed, 0 failed, 0 ignored, 141 filtered out,
finishing in 0.70 seconds. These are incidental command durations, not matched
performance measurements. Owner handle: `63449`.
Log: `/home/agent/wt/evidence/fn187/r3-current/source-only-checkpoint/four-consumers.log`.
The log was independently inspected by the conductor, and the owner confirmed
terminal exit 0. Incidental instrumentation output is not exported or used as
coverage or performance evidence.

The owner's pre-regeneration static chain exited 0: strict preservation of 526 accepted
contracts, 215 accepted helper bytes, all 18 legacy namespace files, and the
406-input/43-addition inventory with 24 negative and 19 nonnegative guard cases.
The conductor independently read the new receipts under the same
`source-only-checkpoint` directory. Nine inputs still have no expected block;
the inventory is not a successful replay.

## Guarded regeneration failure

The next default 406-input guarded regeneration (owner handle `53770`) exited
101. Its exact corpus root failed only `closed_cluster_stale_reply.json`:
scripted reply 3, waiting for instance `a`'s result, never triggered. The
existing 120-second watchdog killed the parked child. The failed log is
`/home/agent/wt/evidence/fn187/r3-current/closed-batch-replay/helper406-generate-1.log`.
No second regeneration or full replay pass is claimed. Successful newly
generated expectations in this partial run still require byte-stability and
consumer verification.

The original stale-incarnation test remains intact at
`wasmhost/src/cluster/transport.rs::tests::restarted_frames_reject_old_incarnations`;
the fixture's replacement metadata incorrectly names `tests/cluster.rs`.
The parent is tracing the request/reply construction before fixing it. The
guest performs a write followed by a clock call, while the new script waits
for a result immediately after replying to the write. This is a construction
hypothesis, not a verified repair. Original assertions stay until actual proof.
In particular, a raw incarnation-2 call prefix alone does not replace the
original raw incarnation-2 result-wrapper assertion.

The parent's fourth bounded external diagnostic reproduces the actual cursor-3
wait: authorization and start/run controls complete, the first `fd_write` call
receives its reply, and no further frame arrives before the diagnostic's
12-second kill. The child kill is diagnostic termination, not a passing case.
No clock callback was observed, so simply adding a clock reply is not yet
justified. Earlier external traces had wire-routing and Python text-buffer
setup defects and are inconclusive; they are not helper failures. The current
trace is `closed-batch-replay/stale-sequence-diagnosis-4.log` under the raw
evidence root. No fixture or production repair is claimed.

The rejected input is recoverably archived at
`closed-batch-replay/rejected-stale-reply/closed_cluster_stale_reply.json`,
SHA-256 `2910563ea1d3ae5f3c5d74e76b54b53bdfb3359523de16f753a2e17c1369ad97`.
Its expected block is still null. All four affected emitter/guard source
preimages are retained beside it. Removing this experimental input from the
corpus is not resolution of the public stale-refusal obligation: the full
original transport test remains, and that replacement remains explicitly
unresolved rather than classified as a private invariant or waived.

A separate positive-restart guard regression reproduced a missing check:
`manual_restart_raw_result_guards_reject_wrong_kind_and_missing_nodes` failed
with `wrong raw RESULT2 kind accepted for a`. The exact root was selected and
compiled successfully; the failure is not a setup error. Log:
`closed-batch-replay/manual-result-kind-red.log`. Strengthening that guard and
checking actual positive restart output is separate from resolving stale-input
refusal. Its subsequent exact-root GREEN exited 0 (1 passed, 0 failed,
17 filtered out), log `closed-batch-replay/manual-result-kind-green.log`.
The conductor checked all four author-bound source pre/post hashes and the
complete packet diff: only the rejected input's emitter/guard branches were
removed, and the positive raw-result guard was strengthened. No accepted
fixture or production behavior changed in that packet.

First default regeneration of the remaining 405 inputs, owner handle `16662`,
exited 0: 1 corpus root passed, 0 failed, 17 filtered out, 26.12 seconds.
Log: `closed-batch-replay/helper405-generate-1.log`. Both positive manual-turn
cases ran ten times with the stricter result-kind and both-node guard.
Second regeneration, owner handle `51881`, exited 0 in 26.11 seconds. Its log
is `helper405-generate-2.log`; the owner's full 405-file comparison exited 0.
Post-stability strict 526-contract, 215-helper-byte and 18-namespace-byte checks
also exited 0 in separate `*after405-stability*` receipts.

The first full Rust helper surface failed only its old derived-input selftest:
it incorrectly applied storage corruption to `cluster_packet` derivations.
The fix preserves the storage and inspection branches and separately checks
complete cluster-bound rebuilding, asset materialization, exact WAT byte
restoration and cleanup before assertions. The final Rust surface (owner
`6816`) exited 0: 18 passed, 0 failed, 0 ignored, 0 filtered out in 5.82 seconds.
Log: `helper405-rust-all-guards-final.log`.

The actual Go consumer plus four exact guard roots (owner `91466`) exited 1:
404 helper cases passed, `closed_cluster_diagnostic_refusal` failed, and all
four guard roots passed with no skips. The new diagnostic fixture caps memory
at 131072 bytes (two pages), while its guest grows from one page to three.
Its default outcome is therefore capacity rather than the intended normal
exit; the Go control scaffold also assumed one callback rather than the
actual write-then-clock pair. Only the new fixture's setup and its independent
default-mode guards are being corrected. All four nondefault diagnostic
refusals remain required. Log: `helper405-go-real-and-guards.jsonl`. This failed
consumer run is not a complete Rust/Go pass.

The same new constructor also supplies fuel 1000 to a guest with a 4000-step
spin loop. Its intended benign control therefore adopts the existing success
cases' memory 262144 and fuel 1000000 together, before the next run. These are
functional input settings, not changed production limits or qualified budgets.

## Final default functional proof

The corrected new diagnostic constructor produces normal exit, peak memory
196608 and exactly `fd_write` callback 1 followed by `clock_time_get` callback 2.
Both Rust and Go independently enforce that default stage. All four nondefault
diagnostic refusal arms remain required and have not been claimed passing here.

- First corrected regeneration `41912` exited 0; second `10519` exited 0.
  Logs: `helper405-diagnostic-fixed-generate-{1,2}.log`.
- The conductor independently ran `diff -qr` against
  `helper405-diagnostic-fixed-round1`: exit 0, all 405 files present.
- Final Rust surface `38952` exited 0: 18 passed, 0 failed, 0 ignored,
  0 filtered out. Log: `helper405-diagnostic-fixed-rust18.log`.
- Final Go selection `53313` exited 0: the exact five intended roots passed,
  including all 405 named helper cases, with 0 failures and 0 skips. The
  conductor independently parsed `helper405-diagnostic-fixed-go5.jsonl`;
  `helper405-go-selection.json` retains the owner's exact selection receipt.
- The owner's post-correction static chain `1406` exited 0 for strict
  526-contract preservation, 215 accepted helper bytes, the 405/42 inventory
  with zero pending expectations and the 23-negative/19-nonnegative guard
  roster, plus `git diff --check`.

This establishes the current default functional corpus, not final R3 acceptance.
The one-off authoring emitter and its three-line hook are now removed. Its
recoverable archive is `closed-batch-replay/retired-authoring/closed_cluster_inputs.rs`,
SHA-256 `4f7192bff62ec70726188a5366ec7089b89687f7b31f34ebc968d05aa9e1336b`.
The conductor verified the hash, absent source file and absent emitter,
`after_all` and temporary trace hooks. No original assertion was removed by
this cleanup.

Pinned formatting initially failed in seven R3-owned files. The formatter
changed only those paths; the conductor independently checked the complete
post-format SHA manifest, the exact seven-file change set, `cargo +1.94.1 fmt
--all -- --check` and `git diff --check`, all exit 0. Receipts are
`closed-batch-replay/pinned-fmt-{before,after}.sha256` and the before snapshots.
The default functional proof above predates this mechanical formatting; it
must not be relabeled as a final post-format frozen-object certificate.

The conservation checkpoint `r3-checkpoint-405-postfmt/default` is now terminal;
it is not the final `after` phase. Review fixes added explicit no-run build
capture, frozen source/input/selector/tool bindings, source/object drift
rejection and frozen-source comparisons. Eight provenance regression tests
pass. The frozen manifest binds 2041 inputs and 313 source files; its SHA-256
is `dc4a4b9ada7e321a406c6a44f4059c7a3bd5d51a41f3633d8859253db044c40c`.

The complete Rust checkpoint passed 213 tests, with 0 failures and 2 ignored,
across seven executables (owner `73549`, exit 0). The explicit corpus selection
also passed all 20 roots across six executables (owner `12997`, exit 0).
Strict profile merge/export and source/object pin verification passed. The
conductor independently inspected the corpus driver and module snapshot.
Evidence is retained beneath
`/home/agent/wt/evidence/fn187/r3-current/r3-checkpoint-405-postfmt/default`.

The snapshot reports 421 Rust full-minus-corpus source identities: helper 69,
broker 17, cache 214, model 52, namespace 35, network 10, volume 4 and world 20;
clock has none. This is a gap inventory, not proof of lost coverage or an
invariant waiver. No matching Go augmentation, private-test union or final
conservation is claimed. Private selectors still require current-body
eligibility, and the unresolved public stale test stays outside that union.
Original assertions remain until their replacement prerequisites are proven.

## Subsequent provisional source cleanup

After that checkpoint became terminal, the owner imported the exact conditional
model and namespace assertion-trim packets into the worker checkout only.
The conductor independently verified all six current file hashes against their
author-bound postimages, and `git diff --check` exited 0. Model packet
`af3da2e221f72c42fff99e7e9186cccc82812125eaea752bede060f5df73aca2`
covers three files and namespace packet
`ea7970f8b7929807d2a09b693afd966d8684d52dfca2dc790a6610263c3b3067`
covers three files. Exact pre/post receipts are
`/home/agent/wt/wasm-fn187-r3-model/.flow/tmp/r3-FINAL-source-preimages.tsv`
and
`/home/agent/wt/wasm-fn187-r3-namespace/.flow/tmp/r3-namespace-conditional-preimages.json`.
Import logs are `closed-batch-replay/{model,namespace}-mapped-trims-import.log`.
These imports invalidate reuse of the preceding checkpoint as current-source
acceptance. Post-trim consumers and conservation are still required; there is
no new runtime or final acceptance claim. No further deletions are authorized
before that proof. The original public stale-incarnation test remains.

The first focused codec guard command (`74532`) exited 1 at compilation:
the namespace trim left an unused `reflect` import in `namespace_test.go`.
No guard ran, so this is not intentional RED evidence. The raw log remains
`closed-batch-replay/cluster-codec-go-red.jsonl`. The owner removed only that
unused import after terminal exit, retaining the original author receipt;
the conductor confirmed no `reflect` references remain. The identical RED
selection is being retried separately (`9488`, `cluster-codec-go-red-2.jsonl`).
No passing replacement or generated codec fixture is claimed at this point.

That retry subsequently exited 1 for the intended guard failure: the exact
`TestClusterTransportCodecLiteralStageGuards` root ran and rejected the
deliberately broad `err != nil` classifier with `current-tag payload refusal
counted as stale attribution`. The conductor independently inspected the JSON
events; the owner confirmed terminal exit. The reviewed GREEN packet
`f17a3036872490f5733f13132e3c8d2e9ce4b43fd2125d594997062b7f13dc33`
restores typed, exact attribution-stage classification and adds only a test
loader and its four-line test-only hook. The original stale test is unchanged.
The same exact Go guard is now running (`20046`, `cluster-codec-go-green.jsonl`);
generation and the bounded three-root Rust check remain conditional on GREEN.

The restored Go guard (`20046`) subsequently exited 0 with the exact root
passing and zero failures/skips. The generator produced the one named case in
two separate external directories and the canonical directory. The conductor
independently compared both complete directory pairs (`diff -qr`, exit 0);
all three JSON copies hash to
`41063e8555f1145c6366f428557d5fd50ea64a36cf1ce8db7a16858e408ddaca`.
The canonical generator log contains the actual root PASS. Owner confirmation
of the combined generation command's terminal exit, the non-regeneration Go
JSON consumer, and the exact Rust loader/guard/original-test selection remain
required. Corpus helper count remains 405; this new codec case is separate.

The owner subsequently confirmed generation command `75906` terminal exit 0
and exactly one JSON in each directory. The non-regeneration Go canonical
selection also exited 0 with both generation environment variables unset.
The conductor independently inspected `cluster-codec-go-canonical-three.jsonl`:
`TestClusterTransportCodecCorpusIsCurrent`, `TestClusterTransportCodecCorpus`
and `TestClusterTransportCodecLiteralStageGuards` all passed, including all six
literal mutation subcases, with no failed/skipped tests. The Rust no-run build
and three exact invocations remain the next gate; this Go proof does not close
the original Rust replacement obligation by itself.

Fresh conductor-run static checks after codec generation both exited 0:
526 original JSON contracts are unchanged and all 215 accepted helper files
remain byte-identical. Distinct receipts are
`closed-batch-replay/root-accepted526-after-codec.json` and
`closed-batch-replay/root-helper215-after-codec.json`; historical receipts
were not overwritten.

## Focused transport-codec Rust proof

Owner handle `91576` subsequently exited 0. The serial cached no-run build
finished in 43.83 seconds (incidental duration, not performance evidence).
The frozen test executable is
`/home/agent/wt/evidence/fn187/r3-current/codec-focused/objects/gomad-wasmhost-test`,
SHA-256 `30ac47662b1a17d9b5a8f6061ba43fcfd7bd75d32e66826b9bb72d63a4335a1d`,
ELF build ID `3f2831a0eae69b0b9a41eda34fc6d89e806a4195`.
The conductor independently checked its hash, build-ID notes and all three
exact test logs beneath `codec-focused`:

- `corpus/test.log`: `cluster::transport::corpus::go_cluster_transport_codec_corpus`.
- `guard/test.log`: `cluster::transport::corpus::codec_literal_stage_guards_reject_target_kind_and_nonrouting_errors`.
- `original/test.log`: `cluster::transport::tests::restarted_frames_reject_old_incarnations`.

Each passed individually: 1 passed, 0 failed, 0 ignored, 141 filtered out.
The owner confirmed isolated raw-profile pools, no LLVM diagnostics and
unchanged source/fixture hashes after execution. This closes the focused
Go/Rust codec correctness gate, not final production-coverage conservation or
permission to delete the original test. The exact incarnation codec contracts
are compared; no opposite-direction full-protocol equivalence or Go equivalent
of Rust's private Broken marker is claimed.

## Next-run resource-scope audit

Source review found a resource-sensitive root inside the ordinary suite:
`metrics::usage::tests::exec_does_not_inherit_parent_peak` touches 128 MiB,
starts a child and compares actual inherited/own RSS bounds. Exclude this
unchanged test and explicitly report it as held, not a private waiver or pass.
Ignored throughput measurements remain ignored.

An initial conservative audit also held positive telemetry checks. Complete
body review corrected that overbroad interpretation: telemetry availability,
nonzero samples and accounting identities have no performance/resource budget,
load comparison or stress allocation. Thus `platform_layout_and_units_are_checked`,
`reports_compile_cache_execution_and_resource_metrics_separately`,
`cluster_instances_match_their_single_runs_and_isolate_failures` and the two
diagnostic cache/ABI accounting roots remain ordinary functional checks.
Their results cannot qualify performance or resource budgets.

No existing assertion is weakened, split or deleted to bypass the genuine
hold. Parser, arithmetic and logical-counter tests remain eligible; incidental
metrics from normal subprocesses are not performance/resource acceptance.
Consequently the upcoming run is a non-resource-functional subset, not a
complete full-suite or final conservation certificate. Raw coverage gaps and
the original complete-suite/qualification obligations remain outstanding.

## Functional-arm tooling checkpoint

The owner added a separately named `functional` arm, mandatory explicit
selectors, functional-only snapshot labels, and Rust/Go final-certificate
prechecks that reject missing actual full arms. A review found that the first
certificate regression could accept an unrelated baseline failure; it now
supplies a narrow valid baseline and requires the candidate-specific refusal.
The conductor independently ran `python3 -m unittest -v test_functional_arm
test_phase_snapshot`: all 12 tests passed (4 functional-arm and 8 provenance),
exit 0. The tests run only Python fixtures; they do not build or execute WASM.
Owner logs include `closed-batch-replay/functional-arm-green-3.log` and
`functional-provenance-green.log`; the original RED log is retained.

Draft inventories separate 209 functional Rust roots into 21 genuine corpus,
133 candidate private invariants, 1 explicit host-concurrency invariant,
30 retained-public and 24 harness/machinery roots, with 1 resource-held root
outside that decomposition. Go has 54 functional roots (53 WASI plus 1 network),
24 corpus roots and 9 candidate private roots; no old 65-root fallback is used.
These are reviewed input counts, not runtime results or final classifications.
The Go private-reason sidecar needs all nine explanations before the next
snapshot. No new build or coverage arm is admitted by this tooling test alone.

## Bounded default admission

The missing Go explanations were added. All 133 Rust and 9 Go private candidate
entries now have precise reasons and current function/source bindings; the
conductor independently checked every referenced source hash. The separately
classified host-concurrency invariant is explicitly combined with private133
in `invariants-rust-after.json` (134 entries), not relabeled private.

The conductor reviewed `default-codec-functional-plan.md`, SHA-256
`3c996b24f5e742000f34a27d73c0bd6f68f4a900dde0ede6640f74a25017b13a`,
and `functional-inventory-manifest.json`, SHA-256
`c982c7eea8401881463446e4ec26bf3ae1153020fb31880e8252c870c1dd1864`.
Independent checks matched all owner-ledger, changed-source, selector and tool
hashes. Approval covers fresh `r3-checkpoint-405-codec-functional/default`
capture, one cached serial no-run build and exact Rust functional209, corpus21
and invariant134 arms. Matching Go54/24/9 execution waits for verified Rust
terminal results. All evidence uses new pools and source/object pins.

The existing cache-publication invariant internally starts eight tiny helpers;
that bounded functional concurrency is explicit, not a claim of one helper
process at a time. No independent foreground builds/tests, benchmark, host
budget test, stress ladder or feature matrix is admitted. The 128 MiB/RSS-bound
test remains held. Any failure or provenance/profile diagnostic stops the
checkpoint for diagnosis; no source deletion or final certificate is authorized.

The fresh phase captured 2058 inputs and 329 frozen files under source pin
`bc9521778463c8775c9f93c62b9c63a9961ddee89e585f290b374c680213703a`;
the conductor independently verified it. No-run build `42808` exited 0.
All three Rust arms subsequently exited 0: functional209 (`55302`), corpus21
(`28598`) and invariant134 (`89835`), with zero failed/ignored selected tests.
The conductor independently read every executable summary and driver log,
checked pinned inventories and actual frozen object hashes, and reverified
current source. The invariant arm keeps the host-concurrency root explicitly
separate in its classification. The ordinary functional main binary reports
three filtered roots: the one held RSS test and two ignored measurements.

Matching Go54/24/9 arms, same-arm Rust re-exports and explicitly functional
Rust/Go snapshots are now released under the reviewed plan. They remain pending;
no Go result, union conservation or final certificate is claimed here.

The three matching Go arms subsequently exited 0: functional54 (`47387`),
corpus24 (`21546`) and private9 (`4348`). The conductor independently parsed
their actual test JSON: respectively 53 WASI plus 1 network, 23 WASI plus
1 network, and 9 WASI roots, with zero failure/skip events in every arm.
The owner confirmed source/object verification after each run. Same-arm
re-exports and explicit functional Rust/Go snapshots are now running (`95945`);
the source freeze remains in effect. No union-gap result is claimed yet.

Re-export/snapshot command `95945` subsequently exited 0. The conductor
independently rechecked source/object pins and frozen object hashes, all exit 0.
The durable worker receipt is
`r3-checkpoint-405-codec-functional/checkpoint-gaps.json` beneath the evidence root.
Current-functional minus corpus/invariant union has nine Rust identities:
`abi.rs:571`, `cluster/transport.rs:79,134–135`, and
`storage_transport.rs:303–306,488`. Go has ten current-functional union gaps
(12 statements, including one zero-statement block). No gap is waived.

Independent baseline diagnostics exposed gaps that the current-functional
comparison alone cannot reveal:

- Across all 2752 originally covered Go blocks, three remain outside the new
  corpus/invariant union: `namespace.go:247.4–248.1`, `runtime.go:334.2–334.24`
  and `runtime.go:335.3–336.1`. The runtime ranges are the **network-present
  delegation**, not the nil-network invalid-deadline fallback; numbered-source
  review corrected the initial prose attribution without changing the measured
  missing identities. The ten current-functional Go gaps were not originally
  covered, so they are distinct from these three losses.
- Comparing the prior frozen post-format full checkpoint against the current
  union at unchanged model source locations exposes seven further identities,
  `model/inspection.rs:532–538`: the default `Cut` insertion in
  `Recorder::boundary`. The old and new line text matches, and the author-bound
  model production prefix is byte-identical. Current functional and union counts
  are both zero there. This is a raw prior-checkpoint diagnostic, not a completed
  baseline/FN/source identity certificate.

All further deletions remain held. Existing owners are auditing exact original
assertion/removal maps. The conductor reviewed and approved only the tiny volume
source restoration packet
`7023fcbb3afcc4757e544de3e3596a3949d77dcdf5357610117081eb36cb9a62`:
restore the original PendingOperations1/path_open setup and complete typed-capacity
OR direct-lookup-null assertion under its original public test name, not a
private relabel. Its earlier snapshot/digest replacement claim is withdrawn.
Current source preimage and patch application check were independently verified;
no new build or generation is authorized by this restoration approval.

The unresolved public stale-refusal replacement, final frozen-source/all-feature
corpus replay and byte stability, final five-feature coverage conservation,
independent review and R4 restructuring remain unproved. Timing, profiling,
workload and resource qualification remain held. Tasks .3–.5 and fn-189 retain
their original requirements and are not complete.

### Subsequent bounded correctness corrections

The immutable Rust baseline diagnostic examined 15590 originally covered
identities: nine map to current-functional coverage outside the union, twelve
map to coverage missing from both current functional and union, and one changed
cursor expression remains unmapped. The twelve include the seven model fallback
lines above, four public conformance paths and one real broker command path.
Diagnostic report SHA256:
`6ee59c0350821af14d119ca6fae70dbf2f9283f4077082ce8a565d6b251fa9c2`.
The conductor independently ran its four Python regression tests, all passing;
this is diagnostic evidence, not a qualification certificate or waiver.

The exact public volume assertion restoration was imported after a header-only
patch-coordinate correction. Current volume test SHA256:
`5a6e47443f3c9d31ffca9da73899c78295a8872386549e50770d651db0060ac5`.
The original dormant-predecision model test and all three assertions were also
restored; current model source SHA256:
`e1709bf5a234d6f5ca438274c8cccac6234710cc245b3772f1fbadbb5887066a`.
Their runtime and coverage acceptance is still pending.

The network runtime-idle callback guard produced its intended RED (20794), then
passed against the routing correction (61248), with no failures or skips in the
GREEN run. The correction invokes the actual runtime callback rather than its
lower-level network delegate and checks the entire version/deadline frame.
Its 62-case adapter corpus is Go-only; neither Rust network corpus consumes that
schema. Two serial Go regeneration/stability checks and canonical replay are
authorized, with only the existing R3 stream-delay case correction permitted.
No Rust parity claim or new Rust consumer is implied.

Source review approved narrow test-harness corrections for public digest
dispatch, initial-record trailing-JSON validation and actual queued broker
command admission. Existing expected outputs remain intact; the initial-record
wire fixture is additive and must be verified through its real generator.
The broker worker is bounded functional concurrency, not a timing benchmark.
No successful runtime or coverage result for these packets is claimed here.

These source changes make the earlier six-arm checkpoint historical. A fresh
source/object freeze is required for subsequent acceptance. Performance,
profiling, resource-limit qualification and native execution remain held.

The Go-only adapter correction subsequently completed its bounded verification:
both generations passed; all 62 files are byte-identical between rounds and 61
remain unchanged against the pre-correction corpus. Only the existing R3
stream-delay case action and expected version word changed. Its final SHA256 is
`1dd45989355b73a7a35401095bb28bb7fe4774bc390b0008dcbc13aaf66e3812`.
All three actual canonical Go roots passed with no failures or skips. The
conductor independently inspected the canonical test summaries and round-to-round
directory comparison. Preservation receipts retain 526 original JSON contracts
and 215 original helper files with zero changes. Evidence lives under
`network-idle-fix-generation` beneath the evidence root. This completes the
focused adapter correction, not the overall coverage-conservation task.

The reviewed namespace and broker packets are now imported in the worker.
Independent postimage verification matches the approved files:
`conformance.rs` =
`209a405940333782e4d4f99df988970b63afe9773073d4ffc91aaee2a64c0fac`,
`namespace_wire_conformance_test.go` =
`189741521ff78d001935172925ae90af98a1ae5c3ab3c8cae98c0a07ccf45321`,
and `broker/corpus.rs` =
`f2dc43e54ca1399bdf4505ce6b5c999aea81daee4b28f8269879ebf97c277854`.
These are source-import receipts only. Fresh focused runtime/coverage evidence
and real wire generation remain pending; the old frozen checkpoint cannot
certify the changed source. The primary Flow task description was updated through
flowctl, preserving the original acceptance requirements and in-progress status.

The conductor reviewed the finite focused-restoration plan SHA256
`457f25211ba56d41808b56f0df9835ad6432666a273d3b9a16032e5f543cb6ba`
and exact Rust-four/Go-five root selectors. Namespace wire generation is the
first approved runtime step; a historical helper may validate the generator only
after exact production-prefix and executable identity checks, with profiling
isolated from measurement pools. Fresh-source acceptance needs a new frozen
phase and fresh objects. Current provenance, exact functional-selection and
baseline-diagnostic Python suites independently pass all 16 tests. Their checks
do not establish runtime coverage or full-suite acceptance.

### Fresh focused restoration result

Both actual namespace-wire generations passed the exact root and all 21
subcases. Their output trees are byte-identical: all 20 earlier wire files remain
unchanged and the one additive initial-record case has the approved SHA256
`1181cefe39b674b15b1ab8a245bf9ff9913e8da0ea94140eba07a2a9cc4ec2fb`.
Strict receipts again report 526 original contracts and 215 helper files with no
changes; the 18 legacy namespace files remain byte-identical.

Fresh phase `r3-focused-restorations/default` has source pin
`eecf6b04c37a3c5a2b07338e506dd813beecdce9f3a7beabb82363846c575c92`.
The serial cached build (48560), exact Rust-four run (19007), exact Go-five run
(77686) and same-pool Rust re-export (11733) all terminated with exit 0. No
failures or skips occurred. The conductor independently read all selected test
results, verified the nine frozen object hashes and receipt input hashes, and
reran source verification successfully.

The receipt `r3-focused-restorations/focused-identity-recovery.json` binds the
unchanged production source/context and function owners. Independent raw LCOV
inspection confirms all twelve previously hidden Rust losses recovered:
`model/inspection.rs:532–538` each count 1, `conformance.rs:438–440` each 13325,
`conformance.rs:969` count 2, and `broker.rs:339` count 2. Actual Go profile
blocks `namespace.go:247.4,248.1`, `runtime.go:334.2,334.24` and
`runtime.go:335.3,336.1` have counts 1, 3 and 3 respectively. This closes those
specific baseline-loss corrections without waiving any public assertion.

This selected functional phase is not a full/corpus/invariant-union acceptance
run. The earlier nine Rust union gaps, new Go harness gaps, changed cursor
identity, complete inventories, remaining features, lint and independent review
still require proof. No additional deletion, broad matrix, resource qualification
or task completion is authorized by this focused result.

### Remaining Go control-probe gap audit

Source inspection corrects the earlier shorthand "new Go harness gaps": the
ten block identities are actual production paths, not test-source lines. Their
functional-only coverage comes from two public control probes inside
`TestClusterTransportCodecLiteralStageGuards`. Its current-a2 tagged reply
passes attribution and reaches exact `unexpected instance frame`; malformed
`{\n` reaches the actual JSON validation error. The canonical shared codec case
contains only the stale-a1 reply and cannot cover those controls.

The production identities remain `cluster.go:782.3,783.1`,
`805.4,805.67`, `813.3,813.10`, `816.11,816.11` (zero statements),
`818.3,821.17`, `824.3,824.25`, `844.3,844.25`, `862.3,862.23`,
`863.4,864.1`, and `protocol.go:58.3,59.1`. These were not covered in the original
Go baseline, but they are public behaviors now exercised by the functional arm;
neither a private relabel nor an instrumentation-noise waiver is justified.

A prospective minimal correction is Go-only serialized current-tag and
malformed-frame stage controls, generated/replayed by real Go production calls,
while retaining independent literal/mutation guards. The existing shared codec
fixture, Rust schema and helper405 must remain unchanged. This is a source-only
recommendation, not implementation authorization or coverage proof.

The finite Rust residual audit also corrects the old storage-bound rationale.
Although the retained unit injects `maximum=1`, the production branch at
`storage_transport.rs:303–306` is publicly expressible: descriptor validation
admits a nonzero declared length with matching chunk count/hash syntax, then
`read_chunks` rejects a length above the engine-computed bound before reading or
allocating the payload. A tiny descriptor-only case can therefore cover this
admission behavior without a large allocation or resource measurement. The
earlier parent residual audit's broad private classification is superseded;
the original injected-bound assertion remains retained pending exact mapping.
This is source evidence only; no new fixture or runtime result is claimed.

Source-order correction for that proposed guard: engine.rs emits `started` at
815 and validates authorization at 824 before namespace/storage attachment and
`read_context` at 855. The earlier proposed "no started" expectation was wrong.
The source packet must require exactly one started frame followed by the exact
invalid/storage-context-bytes result, with no callback, guest instantiation or
execution and no context payload. This preserves the real admission stage; it
does not turn a later refusal into an earlier one.

The current cursor proof script is also explicitly unusable for acceptance: it
targets obsolete `candidate-checks-4`, reads mutable candidate source and has no
consumer in the final comparison. Replacement requires frozen source/function,
object and original-fixture bindings, actual before/after complete observations,
both production-line hits, and fail-closed consumption. No cursor equivalence
or line-conservation waiver has been accepted.

Root subsequently added an isolated `cursor_source_identity` prerequisite to
the worker's comparison tool, with seven targeted unit tests. The initial
failures explicitly identified the missing checked-binding API (not a production
behavior regression); after implementation all seven and the sixteen existing
tooling tests pass. The checker requires a unique empty-cursor branch owned by
the exact production Session::volume_step FN, serde(default) on the cursor_input
field inside EnumerateInput, and an unchanged complete named-cursor branch. An
actual read-only invocation on archived source/FN data binds baseline line 1757
(FN1735) to current line 1773 (FN1751).

Current worker tool SHA256:
`f1943409a3d0ddcfaa5d66ebe4bff5930540d3c9518cf6de5cfe57303fb1d95c`;
new `test_cursor_identity.py` SHA256:
`d22e7fd6a14a5568313b3db0633a60766d0d38de65e8d2ab8eaf447086e3a48b`.
The helper is deliberately not consumed by final comparison: source binding
alone supplies neither runtime equivalence nor permission to waive the line.
Old frozen phases retain their archived tool bytes; their pins are not reused
for the changed current tooling. The obsolete cursor runner is unchanged.

### Residual case guard packet

The reviewed seven-path packet is imported in the worker, including exactly two
new helper inputs with `expected:null` and a Go-only two-stage corpus producer/
consumer. Existing 405 helper inputs, the historical closed-42 manifest and the
shared codec fixture remain unchanged. Packet manifest SHA256:
`1e269a28f211843201aec5eff1a1388508d8532031cff882604d2f94cf8e56a9`.
Root independently read the complete new guard modules and loader hooks and
verified every author snapshot and imported GREEN file hash.

The exact two Go guard roots ran against disabled validation (19469 EXIT1):
six stage-control mutations and the first remote-yield setup mutation failed for
accepting the deliberately bad input, not for compilation or setup. Restored
validation (31708 EXIT0) passed both roots and all six stage mutation subcases,
with no failures or skips. This is guard TDD evidence only. The two helper cases
have not been generated or replayed, and Go-only stage JSONs do not yet exist.
No Rust compilation, corpus generation, coverage closure or task acceptance is
claimed from these guards. The next bounded runtime plan remains under review.

### Subsequent guarded generation and replay

The reviewed generation-only selector preserves ALL cases when unset and rejects
use outside regeneration, malformed/duplicate/unknown names and missing inputs.
Rust guard/selection tests produced their intended failures with validation
disabled, then each passed after restoration. Two authentic generations of only
the new cases passed ten-repeat stability checks, with identical round1/round2
and canonical bytes. The actual projections confirm remote sched_yield/call1/
input{} then exit0, and exactly started plus invalid/storage-context-bytes with
zero instantiation/execution/callback for storage. All 405 earlier helper hashes,
526 original contracts, 215 helper files, 18 namespace files and CLOSED42 remain
unchanged. The unfiltered 407-case Rust canonical root passed (27851 EXIT0).

The first complete Go replay (71258 EXIT1) passed 406/407 cases. Only the new
remote-yield guard failed: it globally decoded projected execute/instantiate
metrics into uint64, although ordinary executed results correctly project their
positive values as `"nonzero"`. Root independently read the exact decode error
and generated metric shape. This is a newly introduced test-guard parser defect,
not a helper or golden mismatch. A production-shaped positive regression and
branch-specific storage-only numeric decoding are authorized before rerunning
the complete Go corpus. No Go-only stage generation ran after this failure.
The new cases are not yet accepted across both consumers.

The cursor runner was separately replaced with an import-safe explicit CLI:
candidate frozen phase and fresh destination are mandatory; archive source/FN,
object ownership/hashes and original Git fixture are checked, with complete
observations, raw output-byte equality and both production-line hits required
by its probe path. Six new runner tests observed the old import-time side effect
then passed; all 29 tooling tests pass together. Both real archived nine-object
inventories pass read-only hash/ownership preflight. No cursor helper probe or
final comparison mapping has executed. Current runner SHA256 is
`103c4f80e177b5b053dd4ce77502ecc83763de45eedd08a7336b7e3951636079`;
runner test SHA256 is
`73e627f7857f6d640ef90235c601733a813ae0fcbcc70ad5eeb4847d314d8767`.

The projected-metric regression subsequently failed as intended on the old
guard (13171 EXIT1), then passed after storage-only numeric metric decoding
(2923 EXIT0). Storage nonzero/missing/null/string-metric rejection checks remain.
The corrected unfiltered Go replay (97731 EXIT0) passed all 407 named cases and
the residual guard, with zero failures/skips. Root independently counted the
407 successful JSON test events and verified the corrected guard postimage
`ca02ad4e8dd14aec33ccb22cf87a586e908b1308df10eef4038782e243618ba8`.
No golden or production code was changed for this fix.

Go-only stage generation and its actual nonregeneration consumer/currentness/
literal-guard roots also passed. Root independently compared round1, round2 and
canonical directories: exactly two files, all bytes identical. Their SHA256s are
`0b4dc72818f8c820bcec6ad6cacf81b809a1e4025b0fc03ff9a0042097d5c7ba`
(current-tag reaches payload) and
`85535545fdca79fea73032ced3d2b6c6b772fc24d9984d54f16428219142f2ef`
(malformed JSON before attribution).

The two new helper cases are now correctness/stability-verified in the default
407-case Rust AND Go canonical corpus. This does not establish fresh branch
coverage, nondefault feature behavior, final union conservation, independent
review or task completion. The genuine new Go stage consumer must enter the
explicit corpus selection before a later frozen coverage run. EOF and cursor
execution/receipt consumption remain open. Performance/resource qualification
and native execution stay held.

### Archived cursor execution and EOF corpus proof

Two new provenance regression checks first exposed omitted cursor dependency/
test bindings, then passed after explicit source-pin and proof-input hashing.
All 31 tooling tests pass. Current runner SHA256:
`6297fce9ece42498bc3f967f69dca5f15a79a37448e3696e79e27a569e1976ea`;
phase snapshot SHA256:
`ce7f7e65c683df802bd69f5e7aac74fc32b7db97c8839cdeca903885355f9d2b`.

Archived cursor probe 58969 exited 0 against before-guard/default and the
historical r3-focused-restorations/default objects. The complete original Go
fixture SHA256 is
`4384b9018577396ceea512d9bb1976bf8e9fe70fcaf59c1b4d406334b29501f2`.
Its uncursored operation index 2 hits baseline None1757/FN1735 and candidate
cursor_input1773/FN1751 once each. Complete before/after output bytes match
SHA256 `76b14cfea7b3620757b0931ff3dcd3bd7e3483a683b5a263d101f7a18f41b858`.
Helper stderr is empty; strict merge/export checks pass. Export logs contain
26 explicitly checked external serde_json/base64 hash-zero records each, not
unqualified zero diagnostics. Root independently checked both raw line hits,
output equality, helper hashes and every receipt input hash. Executed tool bytes
were copied into the probe's tool-sources directory and checked against the
unchanged receipt before future edits.

The receipt at
`cursor-equivalence-archived-default-2026-10-18/receipt.json` has SHA256
`5bd5f96f091eae4fafcf1e6318d5c4a896acb8f0a16a8064ce04c69baa2b01bc`.
It is historical semantic evidence only: no current-source/full certificate or
final comparison mapping is implied. A strict candidate-specific consumer is
the next source-only proposal; final full/union gates must remain mandatory.

The separate Rust-only EOF fixture and actual TaggedInput/read_frame loader
were reviewed and imported without touching shared codec/helper fixtures.
Intended disabled-validation guard RED followed by restored GREEN succeeded;
fresh cached build 89189 exited 0. Four exact runtime roots then passed: new EOF
guard, genuine EOF corpus, unchanged original stale-incarnation test and unchanged
original round-trip test. Root independently read all four logs and verified
frozen main SHA256
`a7c7bf40440ea348672ef4743dbcb611843f8c7e921bbf98615d73ee1856f969`.
Stripping only the two cfg(test) module hooks reproduces original transport SHA
`35226c5c9439f224efc56c55b431f6818ae81ae92016501c3fa68a30d74ffa9b`.
This proves default runtime behavior, not fresh coverage or feature acceptance.

Current explicit selectors include genuine EOF corpus roots: Rust corpus22
(diagnostic20), functional214, machinery27; Go corpus25 and functional60. New
public behavior has not been relabeled private, and original assertions remain.
All final coverage/conservation, source freeze, feature gates, lint, review and
integration requirements stay open. Performance/resource/native holds remain.

### Remaining mixed-root assertion audit

Read-only review of `cluster_tagging_round_trips_single_execution_frames`
found assertions not preserved by the minimal shared codec or EOF cases:
fragmented two-frame output (`call` id1 followed by opaque `x`), exact first
reply id1, sequential wrong-node rejection and sticky third-read failure.
More precisely, after the wrong-node frame is consumed, the third read reaches
EOF and returns the EOF error; `Broken` preserves the first routing message.
The original root requires only that third read to fail, not that both error
messages match. A replacement trace must not confuse those two observations.
Retain this complete original root. Its injected `[]` output rejection alone
may be a defensive invariant; that does not justify reclassifying the root or
deleting its public behavior assertions.

For `crash_image_failures_are_classified_with_their_message`, the existing
admission storage-image-capacity fixture and literal guards preserve the public
capacity tuple. Its injected unknown-operation error is narrower: descriptor
context validation requires selected IDs to be observed pending IDs, and
collection compares observed context with actual pending context before crash.
`volume::error_text` passes an Invalid message through unchanged. This supports
retaining the exact injected infrastructure/message assertion as a defensive
candidate, not declaring all infrastructure failures unreachable. No assertion
was removed, root reclassified, or coverage waiver accepted in this audit.

### Strict cursor consumer proposal: controlled unit checkpoint

External packet directory:
`/home/agent/wt/evidence/fn187/r3-current/cursor-consumer-source-packet-2026-10-18`.
The proposal adds explicit optional receipt consumption for only original
conformance None1757 and the source/FN-checked default-cursor expression. The
actual full-arm refusal remains first; a proof supplies identity, never covered
status. Controlled tests exercise absent proof and missing full/union coverage.

Root source review found two gaps: export diagnostics were hashed but not
validated, and the executable roster could omit test objects. Both intended
regressions failed with `RuntimeError not raised` in
`review-findings-red-2.log`. The corrected proposal shares the existing strict
LLVM diagnostic parser with the consumer and requires the exact nine object
roles. It also checks actual ELF build IDs, original Git fixture/observations,
source/FNs, positive isolated line hits, raw/profile/tool hashes and archived
executed-tool copies. Rehashed zero-hit, changed-FN and missing-role mutations
are rejected.

`unit-combined-2.log` records 43 passing Python tests: 31 existing and 12 new
controlled-reader tests. Root independently read the failure and passing logs
and reviewed the corrected reader/parser/hooks. Test objects are copies of
`/bin/true` and profile data is explicitly synthetic; this is unit verification,
not actual cursor replay or coverage evidence. No helper or LLVM execution,
build, current-tool import, integration or commit was performed for this packet.
Historical receipt/tool bytes remain unchanged.

Before any import, isolate the current-tool-mutation unit test: it deliberately
alters/restores its EXTERNAL packet sibling tool, and must not mutate active
tool sources when run from the live tool directory. Source-packet review is not
formal implementation acceptance. Task .3 and all final gates remain open;
performance/resource qualification and the broad feature matrix stay held.

### Cursor consumer imported after confinement RED/GREEN

Revision 2 resolves the import prerequisite above. Its confinement regression
first refused the old sibling-source write before mutation, then passed after
moving the probe into temporary copied Python tools imported by a child process.
The parent source-write guard and byte comparison confirm the imported tools
remain untouched by the mutation test.

Root reviewed the sandbox delta and verified revision manifest SHA256
`e946d58168a2fcc66092356dff58638f7731e39c6c65564f7d82795a03b4c19d`
and patch SHA256
`7db475ea103d4cb3a45663426b1be1f4dd30fbc6ffae400e8db912e88242310b`.
The exact seven tool/test files were then imported into the worker only through
the preimage-checked apply_patch importer. Root independently matched all seven
current postimage hashes and read the current-tool run: 44 tests, exit 0.

Read-only parser checks also reproduce both archived 26-record external LLVM
diagnostic inventories exactly. The historical schema-v1 receipt is explicitly
refused by the new schema-v2 consumer. All archived files, accepted index,
526 original JSON contracts and 215 original helper bytes remain unchanged.
No compiler, LLVM command or helper ran during this import/checkpoint.

Receipt:
`/home/agent/wt/evidence/fn187/r3-current/cursor-consumer-import-2026-10-18/evidence.json`,
SHA256 `84ae87cb90d542f89cc809da99419dfeecb00fd5d9e8bb9d8d4441e1d9b2bccb`.
Actual logs are `unit-current.log`, `archive-parser.json` and `postflight.json`
in that directory. No live handles or commits remain. Worker and baseline
worktrees remain retained for unintegrated source and historical comparison;
other owners' worktrees are untouched.

This supersedes the pending-import limitation only. There is still no actual
current schema-v2 cursor probe, common frozen phase or final full/union coverage
certificate. The original mixed roots remain intact; no new deletion or private
reclassification was authorized. Fn-187.3 remains in progress, and fn-189,
R4/R5, independent review, primary integration and held qualification gates
remain open.

### Complete original transport trace: eight focused Rust roots pass

One additive Rust-only trace now expresses all eight assertions in the original
`cluster_tagging_round_trips_single_execution_frames` root. It drives the same
TaggedOutput across the two original fragments and `[]` rejection, compares
exact cumulative bytes, then drives the same TaggedInput across reply id1,
wrong-node routing and EOF. The third error is EOF while Broken retains the
first routing message. The non-object injection is covered by this data case;
no private waiver is needed for it.

Root read the whole loader, hand-derived JSON, hook-only source diff and exact
eight-row assertion map before execution. Packet manifest SHA256:
`7a10b77fb106236d5fc8270d4643164294b5054913b5410cf238cdf976f39c2d`.
Assertion-map SHA256:
`a84a2bd19205bce22de7c716d268ac4c504b7462d5a947ec8e474878875834a0`.

Cached single-job default bin-only builds exited 0. The disabled literal guard
then failed exactly with `changed trace target accepted` (exit 101), before
the reviewed GREEN guard was restored. The new guard/corpus, unchanged original
roundtrip/stale roots, existing EOF pair and shared-codec pair each passed one
exact test: eight passed, zero failed/ignored, 146 filtered per invocation.
Root independently read all eight logs and the intended RED failure, and
verified frozen executable SHA256
`e8992f3046c809a0fa5b15ef78abf8747e21266ff3ac64904de96b8ce39bb58b`
and actual ELF build ID `444a7112f8f4d6e45e76a7e0d1c4234a7d8213c2`.

Receipt:
`/home/agent/wt/evidence/fn187/r3-current/transport-trace-proof-2026-10-18/evidence.json`,
SHA256 `4e3ce6a6919fe8006ddd6ca6d1ccd237c98c2a7a128dda6d32c4817c3bc822c7`.
The original complete root and every non-hook transport byte remain unchanged;
526 contracts, 215 helper bytes, the 407-case helper roster and accepted index
are preserved. No helper guest, oversized/resource test, LLVM processing or
feature matrix ran. Raw profiles remain isolated and unmerged.

This establishes the trace's default correctness, not coverage conservation or
permission to remove the original root.

Subsequent source-only refresh adds only the genuine trace consumer to the
default/diagnostic corpus selections and functional selection, and its guard to
functional/machinery. Actual frozen main listing has exactly 147 roots versus
145 previously: two additions, zero removals. Current counts are Rust
functional216, corpus23/diagnostic21, machinery28; Go remains60/25, private
Rust133/Go9, separate host-concurrency1 and retained-public31. No classification
changed. All 142 private source bindings, 526 original contracts, 215 helper
bytes and the accepted index remain preserved. Current Python44 passes again.

Root independently checked all 15 current tool hashes and 15 selector/binding
hashes against manifest
`a9cf96cedd264d9e06ec045ed8d91670ac9290c7508263b308d0860e87f41fae`,
and read the actual test log and exact set-delta receipt. The historical
`prepare_functional_selectors.py` helper still points at an older roster; it
was not rerun or treated as current authority. The manifest records that caveat.
Inventory receipt:
`/home/agent/wt/evidence/fn187/r3-current/transport-trace-inventory-2026-10-18/evidence.json`,
SHA256 `31ec1400e78dd9ca726e187b3ae82e6a25e211ccd8f03a64380f6aa7d3c1d98e`.

The whole current functional216/Go60 and common-source conservation have not
run. Fresh coverage, current schema-v2 cursor proof, all remaining assertion
dispositions, final review/integration and held qualification requirements
remain open. No live handles, commits or task completion are claimed.

## Default 407-case correctness checkpoint

The subsequent single-default-feature checkpoint completed all six selected
arms: Rust functional216/corpus23/invariants134 and Go functional60/corpus25/
invariants9. Root read the terminal driver and Rust summaries, and independently
counted the Go JSON events:94 top-level passes,814 helper-case passes (407 in
each of functional and corpus), zero failures or skips. All three post-Go raw
pool reexports and final preservation checks exited0. This is functional
evidence, not an actual full-arm or conservation certificate.

Phase: `/home/agent/wt/evidence/fn187/r3-current/r3-checkpoint-default-407-trace`.
Source pin: `33cff2023a04f256e891f2c1ea02ab2fc00c941d4e7385c2934b585e12bfa790`.
The phase captured2077 inputs and separately preserved1706 immutable JSON copies;
nine frozen object roles were admitted. Original526 contracts and215 helper
files have zero changes;407 helper cases,42 CLOSED additions and accepted index
`0a343ce3549c1b7ed42146caf23c7800f2cd48f9a60c78ebfd3954036833d299`
remain preserved.

Raw diagnostics deliberately retain eight Rust corpus/invariant-union gaps:
seven model default-cut lines and one storage infrastructure mapper line.
The historical conformance `None` identity remains unmapped pending a current
schema-v2 proof. Go restores every baseline-covered block in functional, but
the namespace lookup-nil block remains absent from corpus/invariant union.
No public assertion was deleted or reclassified to hide these gaps.

Diagnostic SHA256s: Rust baseline
`405af2453c9bbdc0e9477c50a96ef2f17f735d4daffaa8592009acb1bd8e7be2`;
Rust current union
`5b00e07288ab3896516a4a27629874077456f8a780526164f95889d5e1f75419`;
Go baseline
`4ab95fe4e3323bd272f716d3bc7b9c05850fd9c0da19d3ed47b7fd1021dbbab3`.

Performance/resource/native and broad-feature qualification remain held.
Remaining assertion dispositions, current cursor proof, final review and
primary integration are still open. No commit or task-DONE claim.

## Current default cursor identity proof

Separate bounded schema-v2 replay against the frozen default checkpoint passed.
The original accepted `enumeration-depth.json` uncursored operation2 executed
both frozen conformance helpers. Complete outputs match the original Go
observations and each other byte-for-byte, SHA256
`76b14cfea7b3620757b0931ff3dcd3bd7e3483a683b5a263d101f7a18f41b858`.
Exact Session::volume_step-owned old None1757 and current cursor_input1773
each have one positive hit. Profiles are isolated from checkpoint pools.

Receipt:
`/home/agent/wt/evidence/fn187/r3-current/current-default-cursor-proof-2026-10-18/receipt.json`,
SHA256 `84f60228f502cc03d2c369b4b310a2f43ec6f8d8b94a353361436f7e45466f5b`.
Root independently ran the strict reader against the actual receipt and
candidate: exit0, `checked-default-cursor-equivalence`. This rechecks exact
source/FN identities, original Git fixture, executed tool copies, all nine
object roles and actual hashes/build IDs, raw/profile artifacts and strict
LLVM diagnostics. The frozen checkpoint receipt and all eight referenced
reports also match their recorded hashes; its nine object hashes match.
Checkpoint receipt SHA256:
`8910bb01729f84790a97fb6568f036104c738eac4a14b0205d2e0d1d221b0c7b`.

Only this default-phase source identity is established. Historical raw
diagnostics remain unchanged; no current full arm or conservation certificate
was issued, and covered status is never supplied by the identity receipt.
Eight Rust union gaps and one Go union gap remain. No compiler, performance or
resource-sensitive test ran for this proof. Source-only corpus proposals are
the next action; original mixed roots remain intact.

## Storage-create namespace absence: named Go corpus recovery

The precise remaining Go block originated in the retained complete
`TestStorageCrashFaultAndModelLimits`: storageConfig, PendingOperations1,
FD4/create `db`/rightsAll, then capacity classification and direct
`lookup("/workspace/db") == nil`. An initial mistaken `/tmp/x` direction was
stopped before runtime; its hook-only delta was archived and the adapter was
restored exactly to SHA256
`ee53ee457a01a1dcdde35476e8cf0496d4edccaa40df31b35d2884e657cb7f53`.

The accepted correctness addition reuses `volumeConformanceRun` through one
optional test-only observer and records the complete existing observations plus
explicit path/absence in a separate Go-only vector. It does not add a shared
Rust observation field or a top-level shared volume fixture. The original root,
global shared-script roster and production source are unchanged.

Source hashes: volume_test.go
`fd8ad3626decd47f18a2627e1d422d5f8a8839b1018b12add7767457b494fa97`;
new volume_pending_absence_test.go
`a9ec831af1e2fd1c2190c0be855e02405da4df4d5193034cc1ffc439ef018a49`.
New `volume-conformance/vectors/pending-open-namespace-absence.json` SHA256
`0953f3bea55166b2721d064fc7ea4b8641187f49e8c08c8a541f4473327a9271`.

Source review caught a shared-directory Rust compatibility issue, raw JSON
indentation comparison and canonical-generation safety before replay. Revised
source uses the existing RawMessage-aware comparison, external output branch,
and strict unknown/trailing-input refusal. Intended false-absence RED58048
exited1; reviewed GREEN was restored byte-exactly and GREEN89914 passed all12
mutation subtests. The first generation command failed before test selection
because its custom flag preceded the package list. Pinned Go help confirmed the
argument boundary; changing only flag placement produced two actual passes and
byte-identical external outputs. The setup-failure log remains preserved.

Root reviewed the complete generated JSON before canonical import. Exact
canonical corpus/guard/original roots then passed (three roots,12 guard children,
zero failures/skips). A separate new profile selected only the new corpus root:
`namespace.go:247.4,248.1` has one statement and count1. Root independently read
the exact selection, profile row and command, verified all944 pinned Go/test/JSON
inputs from the correct module directory, and reran the preservation verifier
with exit0; its complete output reproduces the receipt hash.

Receipt:
`/home/agent/wt/evidence/fn187/r3-current/lookupnil-go-vector-runtime-2026-10-18/evidence.json`,
SHA256 `f07f05d37283f80c1120730f7317d8db301885e1bf9628314f587f2ee81f0852`.
Focused profile SHA256:
`5c413327f5ccf7dcd6ba0e5a86a76de1ec085285a9e71d1566164005ea481c63`.
All526 original contracts,215 helper bytes,1706 captured JSON files,407 helper
cases, CLOSED42 and the accepted index remain unchanged. No live handles remain.

This proves recovery of the precise Go block by the new named corpus case, not
a refreshed common-source union/full certificate. The earlier default phase
and selectors are now historical relative to this Go-test delta. Explicit
inventory refresh and fresh final coverage remain required, alongside eight
Rust union gaps, assertion dispositions, feature gates, lint/review and primary
integration. No original assertion was removed or reclassified; no Rust helper,
performance/resource-sensitive test, commit or task completion is claimed.

The subsequent source-only inventory refresh is accepted: Go functional62 adds
only the new corpus loader and its literal guard; Go corpus26 adds only the
loader; privateGo9 and Rust216/23/134 remain unchanged. Root independently
compared exact sets against the prior actual selections: zero removals and no
invariant changes. All15 tools,15 selectors/bindings and24 changed-source hashes
match manifest `f9af19b7ceb28785fe6401022c009a23a91b9a0f078130862c1463b72c918be8`.
Root read the complete Python44 passing log. All142 private source bindings,
526 contracts,215 helper bytes,407 cases, CLOSED42 and index are preserved.
Receipt: `lookupnil-inventory-refresh-2026-10-18/evidence.json`, SHA256
`7b7d2d95c2f4a53c9287720ae8e8b6beda8d61368bfb2382682a08878ba20085`
under the same R3 evidence root. This refresh supplies no complete62/26 runtime
arm or conservation certificate. The stale historical selector-authoring helper
was not rerun. A two-case Rust-only storage mapper source packet is approved for
authoring only; its original complete test remains intact.

## Rust typed storage-error mapper corpus

A separate Rust-only JSON now preserves both exact inputs and complete outputs
of `crash_image_failures_are_classified_with_their_message`: Capacity
image-bytes/required3/maximum2 returns capacity plus the original WASI message;
Invalid unknown-operation returns infrastructure plus the exact original text.
The genuine loader calls the actual parent `crash_failure`; checked literal
bindings construct the original static-string Error variants without leaks.
This does not claim that a valid helper request can inject the defensive error.

Only a cfg(test) child hook is added to storage_transport.rs. Root independently
verified that stripping that hook reproduces every original source byte,
SHA256 `29230dfe4b380e4dbb609f3464b141b0102a23b22bb5129dc05ce8c70820cd1a`.
The original complete test remains intact. Current parent SHA256 is
`b9fa7a0e1bcd3d969107cfa57c66fe8b1852186204d32d7d19afc2fe28183fcd`;
child source SHA256
`012f3a810fcbc5b6f657b039c6a8aa67461eec44cc7ed308c9d44b4e59769efa`;
JSON SHA256 `b997bea7b0ddb25ff11ac66b9aa725346a8f5e17ba49d4d13de7ec5f1e0f3d52`.
Root strengthened the source-only guard with two schema-valid variant swaps
and a complete-tuple assertion before execution. There are19 negative controls.

Cached serial default bin-only RED build79995 exited0. The exact guard then
failed as intended (exit101), accepting a changed capacity expected-kind when
that literal check was deliberately removed. The reviewed GREEN was immediately
restored byte-exactly. GREEN build35902 exited0; the genuine corpus, guard and
unchanged original each passed one exact test, zero failed/ignored and148
filtered. No helper guest, resource test or broader suite ran.

Root read all logs, verified the actual frozen executable SHA256
`750b5cd4d9b97f51b11107192a800b7cc6a51b1e13556c6885ead23c06032a8e`,
ELF build ID `1f436500d41355a84cdca7b59548cafab01e92df` and link count1,
and reran the pure preservation verifier with exit0. The526 original contracts,
215 helper bytes,407 helper cases, CLOSED42 and accepted index remain unchanged.
Receipt: `storage-crash-mapper-proof-2026-10-18/evidence.json` under the R3
evidence root, SHA256
`b02aa9555f72c70db8e1fa365236d8743c42788e8b4f8deaeb6b0ed42ac3c204`.

This establishes assertion-faithful default correctness, not recovery in a
fresh corpus/full/union coverage export. Raw profiles remain isolated and
unmerged. Source-only selector refresh is next; the model's seven default-cut
lines and all original final gates remain open. No private classification,
original-test deletion, feature matrix, performance/resource qualification,
primary integration, commit or task-DONE claim.

## No-snapshot model replay corpus

The storage-mapper inventory refresh records Rust218 functional/24 corpus,
with only the two new mapper roots added and no removals. Go62/26/9 and
Rust134 invariants are unchanged; Python44 checks pass. Receipt
`storage-mapper-inventory-refresh-2026-10-18/evidence.json` SHA256
`00cce167aa10245cbb6f1f98c0a3316bdbe35adee185251d8539a984184c5de1`.

A new Rust-only JSON case now reproduces the original dormant-predecision
test before any snapshot. It uses the existing model/debug/decision helpers,
the actual typed decision tape, and checks all four stage observations:
debug success, no divergence immediately after debug, runtime error, then
divergence. The original complete test remains intact. Stripping only the
nested test-module hook reproduces the original model.rs SHA256
`e1709bf5a234d6f5ca438274c8cccac6234710cc245b3772f1fbadbb5887066a`.

The 20-control literal guard failed as intended when its debug-success check
was removed (exit101); the exact reviewed GREEN source was restored. Both
bounded bin-only builds passed. The guard, genuine corpus and unchanged
original each passed one exact test, zero failed/ignored, 150 filtered out.
Root read the actual logs and reran the pure verifier with exit0, checking
the frozen executable SHA256
`d42f7668574bb77aea84fcd0575e4abbc7c88e269f5d06497ff665e10cd5f752`,
ELF build ID `690ac53fd46072daca116e66f2a73f3755b28be7`, link count1,
and exact source bytes. Receipt under the same R3 evidence root:
`model-default-cut-proof-2026-10-18/evidence.json`, SHA256
`be2302b296bef4870ee43b8d61b0799e94e2bcb1122fecc4b6fbe2cf9d589808`.
The preservation receipts retain526 contracts,215 helper bytes,407 helper
cases, CLOSED42 and the accepted index.

This is bounded correctness evidence only. Profiles remain isolated and
unmerged; recovery of the seven mapped model lines and mapper line is not
yet claimed. Post-model inventory and focused coverage are next. No original
assertion deletion, private reclassification, full/union certificate, broad
feature matrix, performance/resource test, commit or task completion.

Post-model source inventory is accepted: Rust220 functional/25 corpus,
23 diagnostic corpus/30 machinery, exactly two new functional roots and one
loader/guard in the corresponding sets, with zero removals. Invariants134,
Go62/26/9, retained-public31, held1 and ignored2 are unchanged. Root independently
checked all15 selector,15 tool and29 changed-source hashes, exact set deltas,
and the actual44-test passing log. Root separately reran accepted-corpus
preservation:526 original JSON files, zero changed contracts. Inventory receipt
`model-default-cut-inventory-refresh-2026-10-18/evidence.json` SHA256
`c5f6fc90e5e4072ed05826841cac2442152f1e3143e13930f220163943b8043a`;
manifest `e9c475c5c8e2a1366276cdbfea9ae753370d4a1818f2d184bc428e81c629bf0b`.
This is a source inventory, not a claim that the complete220/25 arms ran.

## Focused mapped-line recovery, no rebuild

Evidence directory:
`/home/agent/wt/evidence/fn187/r3-current/default-cut-mapper-corpus-lines-2026-10-18`.
The model arm copies only the already-passing genuine corpus profile. The
mapper arm runs only its genuine corpus once on the same frozen d42f executable:
one passed, zero failed/ignored,150 filtered. Each raw pool is separately merged
and exported through the unchanged strict diagnostic parser. No build, helper,
guest or broad suite runs, and no older mapper executable/profile is mixed in.

Root independently read the exact mapper result and source-qualified LCOV:
model/inspection.rs lines532–538 each have count1; storage_transport.rs line491
has count1. The mapper line corresponds to historical488 with only the admitted
three-line cfg(test) hook shifting the file. Both export logs are empty.

This is single-object corpus-only recovery of the eight mapped Rust gaps,
not a refreshed full/union coverage certificate or production denominator.
The separate Go corpus already proves its namespace lookup-absence block.
Original assertions remain retained. Full feature gates, independent review,
primary integration and performance/resource qualification remain open.

Terminal focused receipt SHA256
`cf7faab53ef2182a89209fdd6656001a32d5b0bf1eccbf6f1ed54b1b3cba13c8`
(`evidence.json` in that directory) records both successful merges/exports,
exact baseline/current line text and positive owning FN/FNDA records. Root
independently checked all artifact hashes, copied-model raw provenance and
positive function owners. No live handles remain. All existing worktrees are
retained for their owners' uncommitted changes or historical evidence; none
was removed. No commit, push, PR or task-DONE operation was performed.

## Fresh common-source default gap-recovery checkpoint

`r3-checkpoint-default-407-gap-recovery` is terminal0 (sole owner76437).
Source pin `f8f28ae32006093adf86f63aaf8efe6c8780e3488526bcac2f2a83f84930be6f`
captures2083 inputs plus1709 separately copied immutable JSON files. Root
independently verified the complete copy inventory and source/object pins,
all9 actual ELF roles/hashes/build IDs/link counts, and all selected results.

Rust functional220/corpus25/invariants134 and Go62/26/9 all pass. Both Go
functional and corpus execute407 distinct helper cases, with zero failures or
skips. All three post-Go same-pool re-exports pass the unchanged strict parser;
their explicit external-only mismatch inventories contain254/231/97 entries,
not zero diagnostics. Original526 contracts/215 helper bytes/407 cases/CLOSED42
and accepted index are preserved; all commands are terminal.

Raw current functional-minus-union has zero Rust gaps. The conservative
original15590-covered-identity baseline reports zero mapped Rust losses and
one still-unmapped historical conformance.rs1757 None expression. All three
Go diagnostic arrays are empty: baseline-functional missing, baseline-union
missing and current-functional-minus-union. Root independently read and checked
these reports and all baseline report input hashes.

This closes the eight mapped Rust and one Go gaps in a fresh common-source
default correctness checkpoint, not an actual-full/all-feature certificate.
The current cursor binding, original full/resource/feature gates, assertion
dispositions, review and primary integration remain open. Exact mappings for
the three newest original roots are recorded in
`.flow/artifacts/fn187-r3-last-gap-assertions-2026-10-18.md`; those originals
remain intact while a minimal external deletion patch is prepared for review.

Frozen receipt: `r3-checkpoint-default-407-gap-recovery/receipt.json` under the
R3 evidence root, SHA256
`2d2046b813453d0fdd77e1dbc10dbb02eb50f997144c07f51dcadefb79a51868`.

## Three assertion-mapped originals removed in worker candidate

Following the fresh default checkpoint and exact assertion ledger, reviewed
v2 patch `4279c15c9690e67e150fd49cc25fbbaadbb6368c77d72bd04ce50d69318674c9`
removes only dormant_predecision_leaves_tape_mismatch_to_ordinary_runtime,
crash_image_failures_are_classified_with_their_message and
TestStorageCrashFaultAndModelLimits, plus the unused errors import and one
adjacent separator. Root independently verified all2083 captured inputs:
exactly these three test-source files changed; both Rust production prefixes,
all helpers/consumers/fixtures/tools/selectors and accepted index were unchanged.

Post-source hashes: model.rs
`91d49537e6413b7fc936e9008feeef835781c089815781e5a1a3a65150d28a77`;
storage_transport.rs
`aa63159f5d3b96d99b8aea81439f745dd52946828d2763dcbab4f0de81cb6e3c`;
volume_test.go
`5fe8be6b5045b371894a192b876c730ed64987275faf740d63dbbf609d894048`.
Preimages are recoverable in the immutable gap-recovery phase's source/files
snapshot; the v1 and v2 proposals remain preserved.

Bounded post-removal build83961 exits0. Frozen main SHA256
`853652e8387a598df3d7471af65aa2fcd3ef48247081074029cc15a9c400c076`,
ELF build ID `d0b2ff14420b5cab2590f61c65f1833695c40813`. Root independently
read all four replacement Rust logs: each1 passed,0 failed/ignored,148 filtered.
Root independently parsed exact Go two-root/12-distinct-control PASS with zero
failures/skips and package PASS. No removed original supplies these results.
Proof directory: `final-three-root-removal-proof-2026-10-18` under R3 evidence.

Only the worker candidate changes; primary source integration and independent
review remain pending. The preceding six-arm phase is now historical relative
to the test-only removals, not a fresh post-removal complete coverage claim.
The separate assertion ledger stays byte-frozen as the pre-removal review;
its original-roots-intact note describes that historical snapshot, not the
current candidate. This section records the subsequent removal and replay.
Source-only selector/binding refresh follows. No fixture regeneration, helper
guest, LLVM export, broad suite, feature matrix, performance/resource run,
commit or task-DONE operation is included in this cleanup.

Post-removal build83961 and Go58639 are both authoritatively terminal0, with
no live handles. Frozen proof receipt SHA256
`3958c33f5501518a85af0b9686ded75c9919ae8589c111fabadf86e45e104779`
(`evidence.json` in that proof directory). Root independently matched its
executable and all8 log hashes to the actual results described above.

The post-removal source inventory is accepted: Rust218 functional/25 corpus/
134 invariants/public29/main149; Go61 functional/26 corpus/9 private. Only the
two deleted Rust names and one deleted Go name leave their functional/public
sets; no corpus/private/machinery/held/ignored changes occur. Root independently
verified all15 selector/15 tool/29 source hashes, exact set deltas and142 private
bindings. Only12 bindings change SHA/line metadata; all reasons/classifications
remain identical. The actual Python44 log passes. No further runtime was run.

Inventory receipt `final-three-root-removal-inventory-2026-10-18/evidence.json`
SHA256 `ff151b8a064e9676d2af64bf431a107f051f0215ddaaf4b6436c8d886e684f86`;
current manifest SHA256
`b555181b1c8a1bc9adbfd98586baa3f7a81cfc3f4ec59b9cda4aa263b58e791b`.
This refresh is source-only: complete post-removal218/25/134 and61/26/9 arms,
full/all-feature conservation, remaining public assertion dispositions, lint,
independent review, source integration and original qualification still remain.
All handles are reaped; no task status is promoted to done.

## Retained workspace inventory

Read-only `git worktree list --porcelain` confirms these22 worktrees; no new
worktree was created or removed in this batch. The following short names are
under `/home/agent/wt/`, except the absolute primary path. Historical/other-owner
trees are not cleanup targets: their ownership and uncommitted work must be
reconciled before removal.

- `/Users/stephan/Workspace/skunkworks/gomad/temporal`: primary owner changes,
  accepted staged R1/R2 and local Flow/artifact updates awaiting integration.
- `wasm-fn187-functional`: active uncommitted R3 worker candidate and private cache.
- `fn184`: inherited other-owner qualification work; retained pending owner review.
- `fn184-base`: inherited qualification baseline; retained with fn184 evidence.
- `fn185`: deferred host-cost owner branch; outside this batch.
- `fn187`: inherited helper-corpus owner branch; not yet reconciled for cleanup.
- `fn189`: inherited upstream-sync owner branch; final qualification still open.
- `fn189-cluster-baseline`: retained upstream cluster comparison baseline.
- `wasm-fn187-baseline`: retained before-R3 source/coverage reference.
- `wasm-fn187-coverage`: inherited coverage owner branch and evidence.
- `wasm-fn187-measure`: inherited measurement owner branch; performance work held.
- `wasm-fn187-model-corpus`: earlier stopped dirty model-corpus work; owner retained.
- `wasm-fn187-network-corpus`: earlier stopped dirty network-corpus work; owner retained.
- `wasm-fn187-volume-corpus`: earlier stopped dirty volume-corpus work; owner retained.
- `wasm-fn187-r3-before-guard`: retained R3 pre-guard source reference.
- `wasm-fn187-r3-model`: retained source/assertion ledgers and owner patches.
- `wasm-fn187-r3-namespace`: retained source/assertion ledgers and owner patches.
- `wasm-fn187-r3-network`: retained source/assertion ledgers and owner patches.
- `wasm-fn187-r3-volume`: retained source/assertion ledgers and owner patches.
- `wasm-milestones-fn185`: deferred profiling preparation; other owner.
- `wasm-milestones-fn187`: inherited fixture preparation; not reconciled for cleanup.
- `wasm-milestones-pins`: retained compatibility pin preparation and provenance.

`/home/agent/wt/evidence/fn187/r3-current` is durable evidence, including immutable
source/JSON/profile snapshots and rejected experiments, not disposable scratch.
No new temporary checkout or scratch root was created by the conductor.

## Remaining Go functional-only surface audit

Current functional61 minus corpus26 minus private9 leaves26 distinct Go WASI
roots, each resolved by the conductor to exactly one current declaration.
These are not missing private invariants to admit wholesale. They include
currentness/literal/wire/comparator guards, two reference-contract roots and
the live `TestVolumeConformance` cross-language parity harness.

The conductor read the complete two volume contract roots and their shared
loader call paths. `TestVolumeCoreVectors` calls the same volumeCoreAPIScripts
and compares the complete deserialized vectors/core.json object. Conversely,
`TestVolumeCoreAPIContracts` is a reference-script literal checker, not a
serialized corpus consumer. `TestVolumeConformanceMigrationContracts` includes
an explicit negative comparison control: append a newline to the complete
ImportTranscript and require volumeConformanceDifference to reject it. A
blanket deletion would lose that guard even if model production lines stay
covered. These new R3 roots must not be promoted to corpus/private entries.

For the pre-existing `TestVolumeConformance`, actual common-phase Go events
show115 distinct passing subcases, exactly equal to the115 selected by
TestVolumeConformanceCorpus. The latter compares current Go observations to
the JSON; Rust's volume_conformance_corpus dynamically consumes every top-level
volume JSON, comparing initial session outcome and every complete observation.
The old parity harness additionally exercises the actual helper transport,
whereas the Rust corpus root calls Session directly. Its removal therefore
needs an explicit transport-path equivalence/replacement argument, not merely
the matching115 names or zero production-line gaps. No Go test is removed or
reclassified by this audit.

The original volume-owner handover independently identifies MigrationContracts
and CoreAPIContracts as reference checks, not serialized loaders:
`/home/agent/wt/wasm-fn187-r3-volume/.flow/tmp/r3-volume-handover.md`.
Source anchors are current wasi/volume_test.go (loaders, comparison, scripts),
wasi/volume_conformance_cases_test.go (migration/negative comparator control),
wasi/volume_conformance_core_test.go (API literals), and
wasmhost/src/conformance.rs (direct Rust serialized session replay).

## Complete retained-public Rust audit and next three-root cleanup

Source-only audit covers all29 executable entries/21 distinct test bodies;
eight network bodies compile in both main and network_corpus. Packet:
`/home/agent/wt/evidence/fn187/r3-current/retained-public-audit-2026-10-18`.
Audit SHA256 `492d38b677bc0dc32c1c93fd2e9b402fc8be490a9f9705c3103bcef740cfb1b1`.
It records exact observed routes and unresolved predicates for every body,
without promoting public behavior to private invariants.

Three complete replacements are approved for a minimal deletion-only patch:
transport restarted_frames_reject_old_incarnations, transport
cluster_tagging_round_trips_single_execution_frames, and protocol
image_descriptor_requires_local_model_before_execution. Root independently
read the original bodies, full trace/codec consumers and JSONs, ownership
request and independent stage guard. The shared codec compares original raw
RESULT2 bytes and actual stale read/Broken; the trace preserves fragmented
writes, [] refusal, exact stripped reply, routing error/Broken and third EOF.
Ownership uses the original fuel1/memory65536/output32/namespace1/1/100/zeroSHA
input, with actual guest path/hash, and compares the exact pre-start refusal.

Patch SHA256 `332538e935c53eaa0141bb22faed68a710e3fe5d98f039572ae8ab6dbb09e2e3`
removes79 test-only lines in two files, including only the unused original
Capture helper. Each fixture consumer has its own writer. Protocol production
prefix AND suffix around the test module must remain identical. All other
18 unique public bodies remain intact, including the oversized reader's
Broken-None/drain-next-frame assertions.

The audit identifies real remaining gaps: the exact201 replacement network
deadline is not proved by the scaled16 fixture; stream-ABI decoder/reserved
field/payload/errno matrices are not exercised by model scripts; complete
replay-mismatch snapshots/tape setup and several cluster control/bound input
maps still need exact binding. These are not coverage waivers. Storage v1
envelope structural assertions and descriptor/context input maps likewise
remain explicit, not inferred from unrelated v3 streams.

Root caught and corrected the proposed main roster before execution:
149 minus THREE roots is146, each exact root145 filtered. The v1 plan remains
preserved. Approved v2 manifest SHA256
`b25e34b9a88a0aebcd59939481c446ebbbc69cc898953776e6147086cace392a`,
plan SHA256 `e21fbfbabfebe1ed5e455fdae46db850a66aa2096fdc1486a196cd387eb201e1`.
Only one cached default main+helper_corpus build, three frozen objects,
ordinary-mode check/new-inode restoration at compile-time embedded helper
path, four main replacement roots and two helper roots (including canonical
unfiltered407) are approved. No new fixture, Go, LLVM, conformance, actual-full,
nondefault matrix, performance/resource run, commit or task-DONE is included.
Approval alone is not post-removal runtime evidence.

The approved patch is now imported in the worker: transport.rs SHA256
`e6293b1d49f6423f0266641d86bdd30afecf91fc087fa356089a467447d1e543`,
protocol.rs `7b284f2678ba7e20622b7201f436a79a89b8f848991e2fc764eaa95d3744b766`.
Root independently checked exact posts, transport prefix and both protocol
boundaries, retained oversized root, removed declarations and absence of
unrelated source drift. The first evidence preflight omitted the required
root argument to phase_snapshot.inputs and stopped before compilation; its
failure remains in transport-ownership-removal-proof-2026-10-18. Only the
orchestration call was corrected; fresh retry build/raw paths were used.

Build28801, exact six-root run54932 and preservation52257 are authoritatively
terminal0. Main roster146 excludes all three originals. Four replacement main
roots each1 pass/0 fail/ignored/145 filtered, and the ownership literal guard
plus UNFILTERED canonical407 consumer each1 pass/0 fail/ignored/19 filtered.
Root read all outputs, independently checked all three actual SHA/buildID/
link-count1 objects, exact normal-mode probe, restored embedded helper bytes
before/after canonical replay, and all2083 current input hashes plus index.

Main SHA256 `e3e2e741357807f8146c67553f7fe17d8e9660c9a079eab4d10edaeba9d859bc`,
build ID `e247b6f6610e506fc8c65a36d789735599541a46`.
Coverage-instrumented ordinary helper SHA256 now
`5fae7e8678af6fb0093205f02fdd6200be1f87ead3042ac7f8dea759079c1862`,
build ID `aa963fa3c23b8acc8787c3259ebbe4265ae75c28`. This is focused correctness
object identity, not final production qualification/model/runtime rotation.
Receipt in transport-ownership-removal-proof-2026-10-18-retry/evidence.json
SHA256 `d07f41da0e26bc232cb0ca87a3611f888b1bd3c9ccbaf5ec41a0b6b1a8d5a81f`.
Preservation526/215/407/CLOSED42/index passes; no live handles remain.
Source-only inventory refresh follows; complete post-removal coverage, original
full/feature/resource gates, remaining assertions, review and integration stay
open. No corpus bytes or production behavior were changed by this cleanup.

Additional read-only deadline audit: parsed every network_v2 JSON recursively,
not substring matches inside digest strings. No fixture contains numeric201
as a deadline/time. The only numeric201 values are request IDs in the two
listener-history-key cases. The previously suggested two-pair stall alternative
uses stall_nanos10 and advance11, so it does not close the original literal
replacement-deadline201 assertion. A faithful exact-input case remains needed;
the old test remains intact. No network source or corpus is modified here.

Source-only post-transport/ownership inventory is accepted. Current manifest
SHA256 `e887ffda32cee93b55372d83aeb2145bb17d806b133c66c75fedb999e7dc85d3`;
receipt transport-ownership-removal-inventory-2026-10-18/evidence.json SHA256
`e4f343bc55117650829eede0a4fd7a7d2f4d29d1e5d31afccbadb1cdb6d3e284`.
Only the three deleted names leave Rust functional/public sets:215 functional,
25 corpus,134 invariants,26 public entries (18 unique), main146. Go61/26/9,
machinery30, held1 and ignored2 are unchanged. Protocol is now explicitly
included among30 changed-source bindings. All142 private bindings remain
byte-identical (SHA256 c292bf2f5a64f4587b2fc4a74906bff9fe4ee92f22b6d3894a664ab0bbe31bf5).

Root independently checked exact removal sets and all15 selector/15 tool/30
source hashes, actual142 function-file bindings, complete44-test passing log,
and every focused-proof/inventory artifact hash. The first read-only root
binding check requested an uncopied historical path; it was corrected to the
immutable before-inputs hash, which proves unchanged bindings. No test or source
fix was involved. Preservation526/215/407/CLOSED42/index remains green.

Across the last two cleanup packets, six duplicate original roots are now
removed from the worker candidate with exact corpus replacements and bounded
post-removal checks. No fresh complete215/25/134 or Go61/26/9 coverage phase,
full/nondefault matrix, final cursor binding, independent review, primary source
integration, resource/performance qualification or task completion is claimed.
All runtime handles are reaped. The22 worktree inventory above remains retained
unchanged; no worktree, scratch root or evidence was removed.

## Exact replacement-deadline201 corpus addition

Continue the approved non-performance/resource-sensitive R3 lane with one
additive Go-generated network case, `healed-stall-epoch-exact-100-101-201`.
The original Rust epoch root stays intact. Root reviewed the literal config,
12 concrete inputs, write-result4 and four original epoch assertions, the
eight mutation controls, and complete existing Go/Rust replay consumers.
No model/oracle behavior or existing JSON is changed.

Source packet: `/home/agent/wt/evidence/fn187/r3-current/network-exact-epoch-source-packet-2026-10-18`.
Green patch SHA256 `eb7c3ee74e45d93aa84f0bda5c0b1e59738a57457d9042d856a47d12a4a4fb02`.
Approved finite first step: guard RED/GREEN and newcase-only generation twice
to external directories. Canonical import and bounded default Rust/Go replay
follow only after inspecting the generated JSON. No new LLVM/full/nondefault
matrix, benchmark, stress/RSS, native execution, commit or task completion.
This paragraph records scope approval, not a passing runtime result.

The additive case is now verified in the worker candidate. Omitting only the
deadline201 predicate in a copied Go overlay produced the intended failing
mutation control (exit1); the reviewed guard passed all8 controls. Two external
newcase-only generations passed and are byte-identical, SHA256
`413af6679a8b3a5ef81707c1a4ccd987e086e7754b651e3998e2026717efeab6`.
Root read the complete JSON before canonical import: advance101 keeps resetfalse,
deadline201, pending4/reserved1; advance201 yields resettrue, pending0/reserved0
and the exact timeout reset transition with bytes4.

Pinned canonical Go currentness, serialized replay and guard passed:3 roots,
183 serialized scripts,8 controls,0fail/skip. One cached default Rust
network_corpus build passed; classification, binary-history identity, complete
183-script replay and the unchanged original epoch root each passed1/0fail/
0ignored/23filtered. Frozen executable SHA256
`c4cccf7623da036557220ef5c92ee06387a1262603befb4e6d9afe1d9b40a728`,
build ID `5eb8a4291b8cf33eefc34fcb3c5b1a5048fe7ca1`, link-count1, roster24.
Root independently checked actual logs, all receipt artifact hashes, source
postimages, executable identity, all182 old network JSON hashes and215 original
helper hashes. Original stall source remains
`b85e50b3f2c62152c48cddd992390fc2f073df0578c44cc718f3de63e42762c5`.
Preservation526/215/407/CLOSED42/index passes. All handles are reaped.

Runtime receipt `network-exact-epoch-source-packet-2026-10-18/runtime/final-evidence.json`
SHA256 `f30fdc0f5baac53a96c9e0672c718eb7a6d7e9cfbc039ab9a6834b4aa0f2695c`.
Source-only inventory receipt SHA256
`44cbb90fa6dcf56132d82a7da98ece996d98a047639c3404d7fb1dcce1cfbab7`;
current manifest `ca2943ec7570b590d9e25998d929e01bebcffb4f1cb2ebe411183ed8d42456e9`.
Only the new Go guard enters the functional selector (61→62); corpus26/private9
stay unchanged. Rust215/25/134, public26, machinery30, main146 stay unchanged.
Root verified all15 selector/15 tool/34 source bindings;142 private bindings
remain byte-identical. No Python suite was rerun for this metadata-only update.

This closes the exact201 fixture gap, not R3 acceptance or a coverage certificate.
The original test stays until coverage/deletion requirements are met. No new
LLVM processing, full/nondefault/resource/performance gate, independent review,
primary source integration, commit or task-DONE. Next bounded correctness gap
is the network stream-ABI decoder/payload/reserved-field/errno corpus, still
unimplemented. All22 previously listed worktrees and evidence remain retained
for the same reasons; the worktree list was rechecked, with no additions/removals.

## Network stream-wire corpus correctness

Added a Rust-only typed wire corpus for the decoder/payload/errno primitives,
not another Go model-script claim. Both original ABI test bodies remain intact.
All28 original vectors are preserved; three additions cover nonzero BE fields
and valid-version lengths65600(success)/65601(layout refusal). The original
all-zero oversized vector also has invalid version0, so these additions isolate
the actual upper-length predicate. Largest allocation is65601 bytes, serial;
this is correctness, not resource/stress/performance acceptance.

Root reviewed the complete31-vector JSON, typed consumer, original assertions,
actual production callsites and v2 diff/plan. Output mutations bypass literal
admission and reach the SAME real decode/payload/errno comparisons. V1 remains
preserved. V2 manifest SHA256
`1b23f09ccce92fa41a49be3aecc55e1076bc7e48cafe0104cd602462f10da395`,
patch `a384ee6ebc5851d70e919f569c42c7f85eb54b56c39961720b040f390311d18e`.
Root independently proved removing only the3-line cfg(test) hook reconstructs
the entire original parent SHA b5474183c532375f2a197fd2d2114764323ccd123bfac832de1e071ebcfc95ab.
No production definition, model oracle or existing fixture was changed.

Intended RED build57355 exited0; exact guard exited101 on `changed wire errno
accepted` after only timeout actual-output equality was omitted. Reviewed GREEN
child9dc199216fe2fce13fb7163819cefae3be0b137eaf143f3a49aecbe835ef16c1
was restored and verified. Combined cached default build52987 and all8 exact
runs exited0: corpus+guard+both originals on main and network_corpus, each
1PASS/0fail/ignored,147 or25 filtered respectively. Rosters148/26 contain only
the expected two additions per target. Each corpus consumer executes all31.

Root read every actual log and independently verified both frozen SHA/buildID/
link-count1 objects: main a85b8de23a8a5dee5cb0c357a5a4a5dba03862aec702b1330c702d0d73671b69,
ID d813a140d33b6221b1f072c79321f55041d6ba98; network
1b40111237aff502fff233d20ee07c46acf43a7b6a732546b42b9c58cee7f660,
ID d6fc308243bea5ebfb946dc86cea49146816118b. Receipt:
`stream-abi-source-packet-2026-10-18-v2/runtime/evidence.json`, SHA256
`f0e59348d81ec8273ba074fe46e610a417425d351f32ec1da352ec59b2dae947`.
All receipt artifacts,183 prior network JSON hashes,215 original helper hashes,
original stall source, CLOSED42 and accepted index were independently checked.
Preservation526/215/407/CLOSED42/index passes;64232 terminal0, no live handles.

Source-only inventory receipt SHA256
`5470526862d5a07576a47ddbc4c1fd07ef80f43b264ef1208ffce5ff2f49dbb8`,
current manifest `e969156a42d0706ca8becc23d122090693ccfc730b407c531a62f54b679d946a`.
Only four functional/two corpus/two machinery executable entries are added
across the two targets: Rust219/27/134, diagnostic25, machinery32, public26;
Go62/26/9 unchanged. All15 selector/15 tool/37 source bindings checked;142
private bindings remain byte-identical c292bf2f5a64f4587b2fc4a74906bff9fe4ee92f22b6d3894a664ab0bbe31bf5.
No Python suite was rerun for this metadata-only update.

No new LLVM processing, full/nondefault/resource/performance gate, original
deletion, review, primary source integration, commit or task-DONE. The exact201
and ABI fixture gaps are now filled; next is their coverage/assertion-safe
deletion proof alongside remaining public assertion dispositions. Fn187.3 and
fn189 stay in_progress. All22 listed worktrees remain retained unchanged; the
list was rechecked. Evidence directories remain durable, not disposable scratch.
