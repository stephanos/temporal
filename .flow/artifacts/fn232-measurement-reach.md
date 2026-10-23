# Native measurement preservation source handover

Status: source implementation in progress. Every original test-wasm measurement root remains **UNEXECUTED** for the receiving candidate. Portable decoder/statistics checks do not qualify native execution, compiler preparation, performance, replay, or profile gates.

## Source lineage

- Retirement receiving base: `9d54dfc9dc`.
- Original executable predicate oracle: `1d695061b6382009d0659238c652a5a765b1d5f5`.
- `a09fa72491`: path-translated cumulative native profile source from `42a88eb607` → `de94aaa91b` → `cac5b294eb`, preserving receiving linked metrics and lifecycle fixes.
- `b272619e91`: shared strict JSON decoder accepts a tighter positive nesting bound, never exceeding the unchanged qualification bound128. Measurement unknown GMWT objects use original32.
- `0b537f1f9c`: native measurement roles, actual guest sampling cancellation, request-local external callback-attempt counter, actual terminal/return timing, and approved frozen helper cache subset.

The prior attribution checkout also contains preserved uncommitted source changes. Their exact patch and source hashes are bound by `/home/agent/evidence/fn232-test-wasm-reach/measurement-profile-wip-binding.json`, SHA256 `e4393eb87bda2db6ece7472e6f7ac378c0e6e95d443504706a78edcf8461e8b2`. Historical proofs bind the source described by their handover, including WIP; they must not be relabelled as proofs of HEAD `cac5b294eb` alone. The receiving manual resolution preserves the native linked collector, typed compiled role selection, boxed input and metrics fixes; final receiving diff supplies the additional source binding.

The compiled-module cache subset was received from `9d54dfc9dc..fa1b541d11`, specifically helper `src/cache.rs`, `src/linked.rs`, `src/linked_cluster.rs`, and `tests/linked_compile_cache.rs`. Owner confirms those files remain byte-identical through subsequent host-only review repairs. The linked.rs method conflict was resolved by retaining BOTH guest_profile setter and compile_cache setter. Cache algorithms were not rewritten. Root must integrate the independent host cache lifecycle and receipt repairs.

## Bound old predicate inventory

`/home/agent/evidence/fn232-test-wasm-reach/measurement-sweep-map.json` contains the original119 internal measurement root bodies and six CLI roots, their predicates and original source. The separately owned sweep roots are excluded. `/home/agent/evidence/fn232-test-wasm-reach/measurement-input-contract-map.json` records the exact25-name ordered input catalogue, CLI error precedence, measurement flag/rerun quoting and sqlite transformation predicates. The additional internal/measurement flags root and sqlitebusy TestPrepare are measurement-owned.

The native source retains the original private artifact, complete replay, malformed input, admission, current producer, upper median, strict spread, one-second warm preparation/teardown and per-seed75% namespace bounds. The old Go helper protocol is not restored. Native architecture translations use actual linked terminal Outcomes, canonical native evidence streams, helper projected full choice tapes, and separate host-only metrics.

## Diagnostic boundary translations

`linked::ExecutionTiming` records actual authorization start, actual finish_outcome classification, and return after Store/sampler/watchdog/engine cleanup. Authorization-to-terminal excludes postterminal work. Terminal-to-return retains that work separately. These observations are excluded from canonical Record/provenance/choice/ordinary streams. A native test injects real postterminal work and checks the separate intervals; it is UNEXECUTED.

The diagnostic external-owner counter counts **NoRemote::write callback-frame write attempts**, not completed filesystem calls. It is request-local and compiled out of release. Zero attempts must not be represented as complete ownership proof for every possible filesystem operation. The structural boundary is actual helper-owned ABI dispatch and its fallback callback sink; there is no Go host callback producer in this native path. Namespace acceptance additionally requires actual profile namespace counts and refuses missing counters.

Native retained timing uses its actual monotonic interval; there is no fabricated second clock observer. Typed warm observations retain independent real clock observations. Ordinary and diagnostic roles have separate admitted executable identities and receipts. Sampling and fuel-off variants require the actual compiled features.

## Execution restrictions and evidence

No Cargo, Go, guest build, native cache or compiler lease was used by this task. The only executed implementation tests were std-only rustc decoder/statistics programs: three operation tests and three statistic tests passed using rustc1.98.0; this is not the pinned native compiler. Exact argv/PID/exit/stdout/source/compiler/executable hashes are in `/home/agent/evidence/fn232-test-wasm-reach/measurement-portable-tests.json`. Only the task's own two temporary executables were removed after retaining these bindings. All Rust dependency-bearing tests remain UNEXECUTED. Formatting and git diff whitespace checks are source checks only.

## Receiving dependencies and pending work

Qualification owns actual compiler preparation controls and raw auxiliary observations. Measurement uses the agreed scoped `CompilerControls` seam (GOMAXPROCS=1, absolute GOTMPDIR, GOWASM empty/satconv,signext, parallelismSome1). Those source changes must be integrated before compiling this batch. The Wasmtime compiled-module cache is distinct from the qualification-owned Go compilation cache.

Host measurement implementation and named driver source are still being completed. Final checkpoint, exact root-to-native predicate mapping, receiving source hashes and remaining frozen candidate gates will be appended before handover. This document does not certify the subsystem as complete.

## Receiving integration checkpoint (2026-10-23)

The original dirty worktree was preserved in `fc5a12ccee`, after native source
HEAD `0b537f1f9c`. Merge `f109db10c0` receives root `d71c795a8a`;
`9e4f9b40da` connects the existing namespace/operation and cluster producers and
repairs stale typed callers; `0cc497e67f` retains native fixture and documentation
repairs. Merge `0523f4a475` receives preparation owner checkpoint `af826b8051`,
including the complete compiler controls/inventory/cache source, instead of a
second partial compiler seam. That dependency has its own pending review.

Validation now uses the assigned isolated Cargo target
`/home/agent/.cache/codex-build/fn232-measurement-new` from the workspace directory,
which selects pinned Rust 1.99.1. An initial command from the repository root
selected default Rust 1.98 and exited 101; it was not counted as a code result.
Original WIP compilation found stale native Role arguments, missing Execution
fields, an unimplemented promised compiler interface and orphaned producer
modules. Those errors were repaired; no failed command supplies acceptance.

Fresh native checks on the integrated source:

- `cargo test -p gomad measurement --lib --no-default-features`: **104 passed,
  exit 0**, `/home/agent/measurement-unit-integrated.log`. The suite covers actual
  filesystem retention/leases, bounded command subprocess capture, native input
  and evidence readers, original statistics/namespace bounds and source admission.
  It does not execute the full original 119-root producer workloads.
- `cargo test -p gomad --test measurement_profile_corpus --no-default-features`:
  **3 passed, exit 0**, `/home/agent/measurement-public-ordinary.log`. Actual public
  native subprocesses execute successful/trapping WASM and retained success then
  invalid-request failure; raw modules, terminal streams, private modes, rerun
  records and unchanged latest-success publication are read back and authenticated.
- `cargo test -p gomad-wasmhost --lib linked::`: **5 passed, exit 0**,
  `/home/agent/measurement-linked-unit.log`. The corresponding diagnostic-profile
  build also passed 5, `/home/agent/measurement-linked-diagnostic.log`. One earlier
  incorrectly narrow `linked::tests` selector executed zero tests and supplies no
  proof. The helper suite includes actual terminal/cleanup timing and streaming
  dispatch. The deprecated counter update call was subsequently renamed to the
  pinned compiler's equivalent `try_update`; affected checks will be repeated.

The terminal timing test does **not** replace the old seven external process
control cases or their GroupGone guarantee. Their typed-native preservation is
assigned separately to the cache/lifecycle owner. Generic WASI clock-folding and
poll/clock equivalence are core preservation obligations, not satisfied by the
measurement digest-comparison fixture.

The catalogue-style `stress` adapter remains pending. Existing `stress --spec`,
`--cell` and `--rerun` retain their separately documented native interface. The
original stress recipe uses runtime_probes (256MiB, 4e9 fuel, 60s) and Temporal
15m bounds; substituting the generic measurement runtime_cost/30m recipe is
invalid. The original flags/source inventory remain required. All other 24
catalogue names now reach their existing native producer implementations; this
source wiring is not proof that all actual producer workloads have run.

Full original callable/subvariant-to-native behavior coverage, complete actual
producer runs, fn-232.13 frozen runtime/workload/replay qualification and fn-185.2/.3
profiles remain pending. No inventory reference may be marked covered solely by
these root names, parser fixtures or this handover. Independent source review is
in progress in a fresh gpt-6.1-sol/high context (same GPT family as writer).

## Final composed source checkpoint (2026-10-23)

The source receives independently green root `e944295e58` in `95d4f1845d`.
`3380239852` copies only lifecycle owner's scoped `df0cc2fad8`: actual native
terminal observations, request-local fixture delays, exact timing admission and
real child supervisor kill/watchdog/GroupGone controls. It does not import the
older cache branch wholesale. The root integration must skip a second copy of
`df0cc2fad8` after receiving this measurement branch.

Independent review found and repaired private helper-cache admission, lost cluster
compile metrics, the cluster recovery oracle, authentic reference/candidate replay
and nested namespace child leases. Native cluster compilation now retains measured
prepare/compile/cache fields. Both crash and recovery prefixes retain their actual
World/storage/network evidence. Original recovery predicates include durable
images, distinct identities and converged histories; a nine-mutation fixture
rejects their removal. Cache-directory rules apply to private Wasmtime caches;
the separately owned Go compiler cache legitimately admits an owned non-writable
outer directory with a private subtree.

Final typed timing checks require the actual execution Instants, exact reported
interval equality, positive duration and signed-duration bounds. This closes both
an understated 1.0005-second interval and zero/overflow intervals. Diagnostic and
sampling comparisons preserve exact fuel remaining; only explicitly declared
fuel-off, compiler-variant and GC-variant masks retain their varying-fuel contract.
A fuel-disabled infinite-loop corpus requires watchdog termination and retains
its original 5000ms wall bound. Cluster fuel-off execution no longer attempts to
initialize disabled fuel instrumentation.

Executed on the composed source with pinned Rust 1.99.1 and the assigned target:

- `cargo test -p gomad --lib`: **461 passed, 2 existing ignored, exit 0**,
  `/home/agent/measurement-composed-full-lib-final.log`.
- `cargo test -p gomad --lib --features helper-determinism-fixture measurement::warm::`:
  **14 passed, exit 0**, `/home/agent/measurement-terminal-controls-final.log`.
  These include real native prompt/slow preparation/delayed cleanup and actual
  child death/watchdog/reaping, through the production observation admission.
- `cargo test -p gomad --features helper-diagnostic-fuel-off --test measurement_profile_corpus --test profile_corpus`:
  **6 public execution tests plus 1 profile corpus passed, exit 0**,
  `/home/agent/measurement-public-fuel-off-final.log`.
- Helper `diagnostic_helper_cannot_authorize_an_ordinary_role` executed once each
  under `diagnostic-profile`, `guest-profile`, and `diagnostic-fuel-off`:
  **1 passed in each build, all exit 0**. Guest-profile malformed-bound preservation
  also executed **1 passed, exit 0**. Exact commands and outputs are retained in
  `/home/agent/measurement-helper-role-matrix-final.log`.
- `cargo clippy -p gomad --release --all-targets -- -D warnings`: **exit 0**,
  `/home/agent/measurement-clippy-final.log`.

Additional actual public native corpus checks before the final timing/comparator
repair passed: diagnostic **5 public plus 1 profile**, guest **6 public plus 1
profile**, and ordinary **4 public**. These are bounded WASM fixtures and raw
reader checks, not full Temporal workload execution. Causal failures are retained
in `measurement-cluster-cache-red.log`, `measurement-cluster-oracle-red.log`,
`measurement-positive-timing-red.log`, and `measurement-diagnostic-fuel-red.log`
under `/home/agent`; each failed the corresponding original missing predicate.
No failed or zero-test command supplies positive acceptance.

Fresh independent source review in a separate gpt-6.1-sol/high context returned
READY for the source repairs, lint delta and final exact timing/fuel corrections.
This review shares the model family; it is not a cross-family review. It does not
certify pending end-to-end producer qualification.

Explicit remaining gaps: the catalogue stress adapter; full original 119-root and
6-CLI actual behavior mapping; complete actual producer matrix; runtime-cost's
original Go-owner/helper-owner comparison applicability (the current native rows
are two ordinary linked runs); terminal catalogue post-load authorization mutation
(the current endpoint records six successes, not the original success/mismatch
pair); final-profile raw counter/environment/terminal equivalence reconciliation;
and the frozen fn-232.13 qualification. Original wire-only private revision strings
have no mutable native revision seam and require an explicit disposition; typed
role/config checks above do not silently waive them. Fn-185.2/.3 wall/CPU profiles
remain sequenced after fn-232. No complete preservation or qualification claim is
made here. Durable source/lint worktrees and the assigned target are retained for
the root integrator; other workers' targets were never used.

Final feature checks subsequently executed actual public producer/profile corpora
on the repaired source: diagnostic **5+1 passed**, guest profile **6+1 passed**,
fuel-off **6+1 passed**, every command exit 0, retained in
`/home/agent/measurement-public-profile-matrix-final.log`. Strict separate
profile-feature all-target checks passed for diagnostic, guest and fuel-off
(`/home/agent/measurement-clippy-feature-matrix-final.log`). An attempted combined
fixture/profile build was explicitly refused by the existing compile-time
incompatibility guard and supplies no acceptance; the supported fixture feature
is checked independently. Feature lint repairs retain guard lifetimes and change
only annotations, equivalent Option composition, checked fixture subtraction and
an explicit error for an impossible collector JSON shape.

The final reader located native terminal bytes/calls/advances in actual
`Outcome.local_model` metadata; raw fuel remains an explicit Outcome field.
Independent reader source is owned by the final-profile worker, and its actual
endpoint execution remains pending. Root has authorized the next source work:
actual module-digest admission refusal before engine creation, actual loaded
module descriptor/admission refusal before instantiation, measured cleanup and
actual error-path phase metrics. This follow-up is separate from the coherent
source checkpoint above. The immutable Go reference invocation glue for the
runtime owner comparison is owned by the lifecycle worker. Preparation auxiliary
observation publication is also required by the final-profile reader and remains
a next source delta; the current measurement preparation only publishes module
and provenance, although the underlying preparation produces the raw observations.

The independently supported fixture strict all-target check completed with exit 0
(`/home/agent/measurement-clippy-terminal-final.log`), and its actual production
observation/supervisor corpus again passed 14 tests with exit 0. Fresh independent
review read those completed outputs and returned source READY after green for the
final feature lint repairs. Source hashes for the final repaired Rust files are
retained at `/home/agent/measurement-source-checkpoint-sha256.txt`.

## Terminal/publication and custody follow-up

The additive source and actual evidence are recorded in
[fn232-measurement-terminal-custody-2026-10-23.md](fn232-measurement-terminal-custody-2026-10-23.md).
This supersedes the earlier source TODOs for typed admission-refusal controls,
actual terminal-to-child-reap publication timing, and preparation observation
publication/reconstructed-consumer custody. It does not replace the separately
pending immutable transport proofs, full Temporal named endpoint execution,
runtime-owner oracle or frozen qualification. The follow-up passed 462 native
library tests (3 ignored), all five strict feature builds, all public profile
variants and the actual compiler/runtime custody probes, with fresh independent
source READY after green.
