## Current public acceptance: a07 candidate

Task .1 is verified DONE in Flow; task .2 is in progress for the explicit positive display golden and final spec review. Earlier pending runtime notes below are historical.

Immutable candidate `/home/agent/wt/evidence/fn232-integrated-provenancefix-01/gomad-a07dd6e2c404ce9120c448e8568c18906fa2f9aebd87d11ce8d53a7bac8a861b` is built from `8536a84a8595b0f5e27295b93885908bd65f07bb`. All 4,570 tools inputs match before/after build. Default 156 and fixture 158 tests pass, with one existing ignored test each; both strict all-target Clippy checks and formatting exit 0. The narrow authenticated-absence correction has fresh independent SHIP review (same GPT family), meaningful genuine-parent RED/GREEN and actual one-attempt minimization with exact final replay. The full predicate remains unchanged.

Public acceptance roots are `/home/agent/wt/evidence/fn231-public-provenancefix-01` and `.../fn231-public-provenancefix-02`. Scope01 and scope02 each pass 55 command receipts; semantic01 and semantic02 each pass 119 receipts against frozen Go, all terminal 0 with unchanged source/input/binary bindings and original limits. Both newly generated semantic goldens are byte-identical: SHA256 `3095bc851f9fa9f821dac792e585f16228975a0ebd0d8dd0a0cafeaedf05f61e`, 38,237 bytes. The checked semantic golden remains unchanged pending task .2. The second driver is `9808f8807c44024d31042630c8136b2f26321069`, differing only by the unrelated `.plans/GOMAD_RACES.md` commit; all compiled tools inputs were independently rehashed identical. An earlier scope02 HEAD-guard refusal launched no child and remains preserved. The two-run comparison receipt is `fn231-public-provenancefix-02/two-run-comparison.json`.

| R-ID | Accepted evidence |
| --- | --- |
| R1 | Both scope runs refuse four invalid patterns and genuine stripped-symbol parents before output/attempts. Semantic scoped controllers preserve global out/mixed/in stability and the suffix-only prefix exception. |
| R2 | Both scope runs authenticate eight producing-round snapshots, remove the campaign, inherit scope from a detached failure, accept explicit override and refuse malformed present context naming its parent. Legacy semantic parents remain unscoped. |
| R3 | Content controllers show 2 unscoped, 3 same-minimum scoped and 4 distinct-valid-minimum scoped attempts under the same predicate. Actual minimized artifacts reproduce exact retained failures. |
| R4 | Unchanged complete and interrupted resumes agree; changed scope/context, implementation, parent, predicate and bounds refuse. |
| R5 | Actual minimum inspection resolves both removed decisions to main.choose/main.go:6 with inline main.main; the final group contains one main.choose decision. Detached semantic inspection agrees, including unavailable reasons. The task .2 positive golden additionally snapshots these exact new symbol objects. |
| R6 | Actual inherited main scope resolves two sites and reports two removals in scope, zero out. Zero-match reductions remain nonvacuous. Six semantic scoped reports snapshot scope and counts. |
| R7 | Removing report-only symbol/group fields reconstructs the authenticated minimization block byte-for-byte. Schema, execution/failure identities and full predicate remain unchanged; context-qualified collisions and journal reopening are tested. |
| R8 | Frozen unscoped proposal fingerprints, choice-prefix proposal bytes and existing semantic behavior fields remain unchanged; both fresh differential lanes agree with Go. |

Task .2's narrow Python-only display checker/golden checkpoint `6e299e88ba` is awaiting fresh independent review. Both already-authenticated positive scope runs yield the same 938-byte projection, SHA256 `4d50ea6bf4def5cee39e09755672f23d99dc52e2f62642b0e2090fc8a365549d`; raw evidence is unchanged. No final Rust qualification, Go retirement, or whole-milestone completion is claimed.

# fn231 Rust minimizer batch handover — 2026-10-22

Source checkpoint: `2ac50dcabf`, branch `fn231-rust-minimizer`, base
`e5ea438c90`. Both fn231 tasks have source implementations in one coherent batch.
Task completion remains with the conductor. The initial checkpoint was provisional
against the integrated fn232.9/.11 and site APIs. The conductor subsequently
accepted fn232.11 and fn230.1 and started fn231.1; this batch does not imply its
own acceptance or claim runtime/qualification results.

## Implemented surfaces

- Exploration retains manifest-only `provenance/search-context.json`, with
  normalized patterns, authenticated module/choice identity, canonical producing
  plan digest, zero-based round and immutable pre-round resolution. Journal
  replay authenticates each context against its pre-event state. Base resolution
  is checked against the plan; later discoveries enter the next snapshot.
- The shared opener admits optional context strictly. Absent legacy context
  remains absent; malformed present context refuses. Context-qualified store
  publication, reuse, live references and retirement preserve record/failure
  identities and authenticate both digest components.
- Minimize accepts the existing scope grammar, inherits valid detached context,
  resolves the parent once and pins selection, source snapshot, parent identities,
  parent resolution and decision classifications in checkpoint binding and state.
  The simulation/controller generator uses global stable out/mixed/in passes;
  the public prefix branch remains unchanged. Both controller identities rotate.
- New minimized artifacts retain manifest-only original source tape and canonical
  descriptor. The opener checks the pair, bounds, identity and tape commitments,
  plus exact removed-decision joins. Existing original-record commitment,
  minimization evidence types and acceptance/replay predicate remain unchanged.
- Report wrappers add scope/counts to minimize JSON and exact symbol joins and
  final package/function/reason grouping to inspect. Legacy missing joins and
  stripped symbols remain explicit. Inspect never writes retained evidence.
- Existing choice-live and minimizer reference drivers now exercise inherited
  detached failures, scoped controller content predicates, explicit prefix scope,
  changed-scope resume and symbol/count reports. They have syntax checks only;
  runtime execution and reviewed golden regeneration remain outstanding.

## Executed verification

All Cargo invocations below used
`CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn232-retained-validation` and
`CARGO_BUILD_JOBS=2`. The conductor serialized ownership of this existing target;
no new heavy target or guest lane was started.

| Command | Numeric result | Raw evidence |
| --- | --- | --- |
| `cargo +1.99.1 test --manifest-path tools/gomad/rust/Cargo.toml --locked --release -p gomad --lib minimize` before production edits | 101; three existing tests passed, four feature tests failed, legacy fingerprint capture failed intentionally | `/home/agent/evidence/fn231-minimize-red.log` |
| Same Cargo test without the `minimize` filter, final source matching `2ac50dcabf` | 0; 101 passed, zero failed, 0.46 seconds test runtime | `/home/agent/evidence/fn231-lib-final.log` |
| `cargo +1.99.1 fmt --manifest-path tools/gomad/rust/Cargo.toml --all -- --check` | 0 | final source command output |
| Python `ast.parse` on `choice-live-check.py` and `minimize-reference-lane.py` | 0; two drivers parsed | final source command output |
| `git diff --check` | 0 | final source command output |
| `cargo +1.99.1 clippy --manifest-path tools/gomad/rust/Cargo.toml --locked --release -p gomad --all-targets -- -D warnings` | 101; task-owned diagnostics fixed, twelve unchanged baseline diagnostics remain | `/home/agent/evidence/fn231-clippy-owned-fixed.log` |

Tool versions: Cargo `1.99.1 (5f94df478 2026-08-27)`, rustc
`1.99.1 (58cf8f9c3 2026-10-13)`, Python `3.14.4`.

The first source/report test run had fixture mistakes: it expected the publisher
rather than its shared opener to defer malformed-payload refusal, and it renamed
a detached fixture inside a managed store. Those were repaired without relaxing
naming or validation. Another direct opener test restored a removed payload
with the wrong mode; restoring mode 0600 repaired that fixture. Final tests
include deliberately resealed invalid retained descriptors and missing pairs,
so early producer refusal does not substitute for opener admission coverage.
Intermediate logs remain under `/home/agent/evidence/fn231-*`.

Strict Clippy remains failing on unchanged base surfaces:
`campaign/ledger.rs:1190-1191` (`unreadable_literal`, five occurrences),
`campaign/execution.rs:439` (`collapsible_else_if`), `campaign/mod.rs:633`
(`assigning_clones`), `replay/cluster/oracles.rs:55` (`struct_field_names`),
`:68` (`struct_excessive_bools`), `:345` (`collapsible_if`), `:429`
(`needless_pass_by_value`), and `replay/cluster.rs:633` (`missing_errors_doc`).
The conductor owns integrated fixes and the final strict gate. No allowances
were added.

## Task and requirement coverage

| Task / requirements | Executed proof | Remaining integrated proof |
| --- | --- | --- |
| .1 / R1, R3, R8 | Shared grammar test; global category stability; content-predicate 2/3 and 2/4 attempts with valid distinct minima; exact frozen no-scope and public-prefix proposal fingerprints | Public prefix runs, scoped input refusals and unchanged simulation/cluster refusal lane |
| .1 / R2, R7 | Typed malformed-present opener tests, package/fingerprint validation, same-record/different-pattern context publication/reuse/retirement with identical record and failure identities | Real producing rounds, evolving snapshots, parallel completion order, detached failure inheritance and overrides |
| .1 / R4 | State key/classification/hash/history and source-match refusal tests | Interrupted/complete public resumes; changed scope/context/parent/map/bounds and old implementation admission |
| .2 / R5, R6 | Detached report and raw-evidence byte preservation; exact identity mismatch refusal; legacy missing source, stripped and resolved groups; inherited/explicit source binding | Runtime JSON/golden counts and symbol reports; second fresh regeneration |

Run the affected existing lanes twice from fresh outputs after integration and
review. `tools/gomad/scripts/behavioral-parity --help` was consulted. The minimize
entry point requires `--verb minimize --corpus ... --minimize-binary ...
--minimize-baseline ... --minimize-source ... --minimize-go-compiler ...
--output-root ...`, with the existing cache/preparation and golden flags. The
scoped producer/consumer path is in `choice-live-check.py --site-scope-only`
with its explicit binary/compiler/input/work arguments. Authenticate raw records
and payloads before reviewing semantic projections. New `scoped_controllers`
and `scoped_reports` are included in the minimizer semantic golden; do not remove
scope digests or source identities to obtain agreement. Keep the existing
unscoped Go/Rust controller and runtime acceptance comparisons.

## Ownership and retained workspaces

Two disjoint implementation delegates returned patches; this worker was the sole
integration committer. The delegates ran no Cargo builds. Fresh independent batch
review and integrated runtime/golden checks remain with the conductor. No Flow
transitions, push, PR, CI changes, dependency changes or Go-host feature work ran.

| Path | Reason retained |
| --- | --- |
| `/home/agent/wt/fn231-rust-minimizer` | Clean committed batch; integration/review and subsequent fixes |
| `/home/agent/wt/fn231-proposals` | Dirty delegate source snapshot, already incorporated; retained pending parent review and owned cleanup |
| `/home/agent/wt/fn231-provenance` | Dirty delegate source snapshot, already incorporated; retained pending parent review and owned cleanup |
| `/home/agent/evidence/fn231-*` | Open-spec raw RED/GREEN/Clippy logs and returned source patches |

Four failed unit-fixture directories under the shared `tmp` root were removed
after their exact exited test PIDs and private directory ownership were checked.
No other worker's target, worktree or scratch root was removed. Other registered
worktrees remain with their existing owners. Latest observed free space was
18 GiB on the shared filesystem; no Cargo process or target grant remains owned
by this worker at handover.

## Independent-review corrections — source `83f69b33b0`

All three confirmed findings against the initial checkpoint are corrected:

- Campaign artifact references now reuse the store's exact qualified-name grammar
  and manifest/name binding. A real prepared-artifact publication and scoped
  execution publication go through `RoundJournal::append` and `open`. The same
  regression refuses an otherwise valid wrong-round context before writing the
  journal, and refuses a qualified suffix that differs from its manifest digest.
- Source admission reconstructs original and final prefix candidate commitments
  using the exact existing candidate hash helper shared with the controller.
  A detached artifact with resealed source bytes/descriptor/manifest, unchanged
  selected identities and unchanged canonical record/minimization bytes now
  refuses a substituted removed-decision site. A separate negative covers the
  final prefix commitment with the original commitment intact. Test fixtures
  now use real controller-derived candidate hashes rather than arbitrary labels.
- Preflight errors now name the parent before ordering admission. A malformed
  present context reaches the actual opener path, exits 2, names its parent,
  and creates no minimizer output directory even with an explicit scope flag.

The scoped live driver also keeps the existing valid `all` failure policy's
budget of one; increasing that budget was an invalid input for this controller,
not permission to execute more rounds. The execution bound still selects eight.
No acceptance predicate, retained evidence schema or candidate bytes changed.

Review REDs ran before production fixes on root's exclusively assigned warmed
`fn232-live-corpus` target, release/locked/jobs2, with no guest executions:

| Focused filter | RED exit / cause | GREEN exit / result | Raw logs |
| --- | --- | --- | --- |
| `source_choice_resealed_site_substitution` | 101; resealed site substitution accepted | 0 within minimize suite | `/home/agent/evidence/fn231-review-source-red.log` |
| `minimize_malformed_present_context_names_parent` | 101; actual context error omitted the parent | 0 within minimize suite | `/home/agent/evidence/fn231-review-diagnostic-red.log` |
| `scoped_context_publication_appends_reopens` | 101; unsafe qualified campaign reference | 0; publication/append/reopen and wrong-context negatives pass | `/home/agent/evidence/fn231-review-journal-red-2.log`, `/home/agent/evidence/fn231-review-journal-green.log` |

The journal fixture's first RED stopped on a stale empty-tape digest after its
module identity changed. Updating that digest reached the intended reference
failure. The first GREEN compile exposed ownership of the prefix plan; validation
now consumes full then final prefixes while retaining the original identity map.
That compile failure remains in `fn231-review-minimize-green.log`.

Final corrected source verification:
`cargo +1.99.1 test --manifest-path tools/gomad/rust/Cargo.toml --locked --release
-p gomad --lib minimize` exited 0 with 18 tests;
its raw log is `/home/agent/evidence/fn231-review-minimize-green-2.log`.
The conductor then authorized one full default host unit run with the same flags
and no filter: exit 0, **105 passed**, zero failed, 0.52 seconds test runtime,
`/home/agent/evidence/fn231-review-lib-final.log`. This includes context, store,
scope and bounded/paired metadata regressions. Format, diff and both Python
syntax checks also exited 0. Strict Clippy has not been rerun on this correction;
the earlier baseline failure remains explicitly outstanding until the conductor's
integrated strict gate. No lint allowances or weakened assertions were added.

Fresh review of the corrected checkpoint and two fresh affected public runtime
lanes/golden generations remain outstanding. Root's source advances were not
merged into this isolated branch; no integration conflicts were introduced here.
The target lease is released, and five private failed-review fixture roots were
cleaned after their exact test PIDs were verified exited. The three own worktrees
listed above remain for integration/review; latest free space was 15 GiB.

## Integrated strict-lint follow-up

Root's integrated source passed 117 unit tests and format checking. Its strict
Clippy run found two feature-owned test diagnostics: the minimized fixture helper
crossed 100 lines after the real candidate commitments were added, and the actual
malformed-context CLI test used `result.err().expect()`. The candidate sequence
calculation is now a small fixture helper using the same controller calls, and
the CLI refusal uses `expect_err` (its success type is `i32`). No assertions,
production behavior, fixture inputs, bounds or lint allowances changed.

This follow-up ran format, both parity-driver Python syntax checks and diff
checking only, all exit 0. No Cargo build, guest or qualification command ran.
The conductor owns the new integrated strict Clippy and runtime gates; the
remaining integrated baseline diagnostics remain with their existing owners.


## Prepared integrated public validation plan (not executed)

Run only after the conductor supplies the new integrated fixture path and its
source/binary/helper receipts. The fixture must enable the host
`determinism-fixture` commands while embedding the ordinary helper role. The scope
driver's `fixture-replay-input` defaults to ordinary; a diagnostic, stress or
helper-determinism role is incompatible. Do not use the earlier immutable 374c
fixture. Run the integrated scripts from the same new source checkout as that
fixture, not this feature worktree's old checkout. No Cargo invocation is needed
for these commands once that fixture is supplied.

Exact bindings below use the existing immutable Go/reference inputs and warm
preparation caches. `FN231_SOURCE`, `FN231_FIXTURE` and `FN231_OUTPUT` are the three
conductor-supplied absolute paths: integrated checkout, freshly built fixture and
new private evidence parent. Require all four child output roots and both golden
candidate paths to be absent before the first command. Record binary/helper,
compiler, driver and input SHA-256 values with the source revision before running.

```bash
CORPUS="$FN231_SOURCE/tools/gomad/testdata/parity"
COMPILER=/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-stock-go1.27.1/bin/go
REFERENCE=/home/agent/wt/evidence/fn232/go-baseline-1d7b37f00a
REFERENCE_SOURCE=/home/agent/wt/fn232-go-source/temporal-1d7b37f00a
REPLAY_INPUT=/home/agent/wt/evidence/fn232-replay/stable-reference2/replay-input.json
MIN_CACHE=/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-refresh-minimize-go-cache
MIN_PREP=/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-refresh-minimize-preparation
```

Execute this pair first with `FN231_RUN=01`, then with `FN231_RUN=02`, sequentially;
these are two fresh full generations, not a rerun against already populated roots.
The reference driver builds the frozen Go wrapper from authenticated archived
source; it does not use the current Rust host as its reference.

```bash
python3 "$CORPUS/choice-live-check.py" \
  --binary "$FN231_FIXTURE" --go-compiler "$COMPILER" \
  --input "$REPLAY_INPUT" --go-cache /home/agent/.cache/go-build-local \
  --site-scope-only --work "$FN231_OUTPUT/scope-$FN231_RUN"

GOMAD_GOLDEN_REGENERATE=1 python3 "$FN231_SOURCE/tools/gomad/scripts/behavioral-parity" \
  --corpus "$CORPUS" --verb minimize --minimize-binary "$FN231_FIXTURE" \
  --minimize-baseline "$REFERENCE" --minimize-source "$REFERENCE_SOURCE" \
  --minimize-go-compiler "$COMPILER" --minimize-go-cache "$MIN_CACHE" \
  --minimize-preparation-root "$MIN_PREP" \
  --minimize-golden "$FN231_OUTPUT/semantic-$FN231_RUN.json" \
  --output-root "$FN231_OUTPUT/minimize-$FN231_RUN"
```

Input commitments already inspected (recheck before running): compiler
`1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64`;
reference inventory `0621a68952a5561c2e820f8cf753c43df12d2902e44f3747fec388d0d1172d94`;
reference archive `d1b96fd47f9e165793555040ff2d223a44bdfc922124fcdbb66cc309498aacd4`;
replay input `7293780ca18b7c24feaad6cc731de6127f31dba5d417891f892d1b2728b9fc32`.
Replay input module paths already exist: exact-seed7/8 share module hash
`e2b37ae878119478496e75bc60fc11b68dc2ab612c1a89724dc32d318a720ffb`,
observed-seed7/8 share
`c776ba5a4d7c82501b05eaeffbf17d88de1bc3dfd8052317eac4d69139ec0bf5`.
The scope driver builds the existing `site_scope` and
`live_backend/minimization` Go guests from the integrated tree using the current
runtime overlay. It does not introduce a new Go-host feature.

Coverage and required review of each pair:

| Requirement | Actual driver evidence and bounded supplements |
| --- | --- |
| R1 | Reference content controllers show stable out/mixed/in ordering and changed attempt counts; public prefix cases preserve suffix-only behavior. Direct minimizer invalid-pattern probes below supplement the exploration parser probes. The scope driver now also retains an authentic unscoped stripped failure and requires scoped minimize refusal before output/attempts. |
| R2 | Scope driver authenticates every retained pre-round context, copies an actual `target_failure`, deletes its producing campaign, then minimizes the detached parent with inherited `main`; explicit `runtime` override and malformed-present context status 2 are actual CLI cases. Confirm diagnostic names detached parent. |
| R3 | Controller scoped/unscoped predicate results and attempt counts; actual minimized failure replays reproduced. Do not require identical greedy content minima in general. |
| R4 | Actual completed resume agrees and changed explicit scope refuses 3; reference lane exercises bound/implementation/parent/predicate checkpoint corruption. Automated interrupted scoped resume and changed valid context probes are described below. |
| R5 | Actual minimized inspection contains removed-decision symbols and final grouping totals. Reference lane detached source reporting authenticates source payloads. Unit fixtures retain stripped and missing-site unavailable reasons and paired-payload tamper negatives. |
| R6 | Actual JSON inherited/override scope and per-reduction inside+outside totals equal removed counts; controller reports independently cover classifications. |
| R7 | Inspect leaves retained inspection bytes unchanged; actual replay and frozen failure metadata stay fixed. Journal publication/reopen and qualified wrong-context refusal, same-record/different-context collision and source candidate commitments are the already passing meaningful unit regressions, not invented public publisher commands. |
| R8 | Frozen unscoped Go/Rust prefix and content controller comparisons, repeated records and reductions stay exact; compare existing golden fields before accepting scoped additions. |

The scope driver now automatically performs the following supplements. Its real detached parent is exactly
`$FN231_OUTPUT/scope-$FN231_RUN/scope-failure-detached`. With fresh artifacts roots,
run `"$FN231_FIXTURE" minimize --json --site-scope=PATTERN --artifacts OUT PARENT`
for patterns `''`, `'main,'`, `'main/ child'`, `'main/*'`: require status 2 and no
OUT directory. Run the same invocation with `gomad.invalid/no-match`: require
success, an empty resolution and all removed counts out of scope. These are
ordinary existing CLI arguments, not fixture controls.

For scoped recovery, run the inherited minimize command with fresh OUT and
`GOMAD_FIXTURE_MINIMIZE_CONTROLS=CONTROL`
plus `GOMAD_FIXTURE_MINIMIZE_INTERRUPT='after accepted'`, then separately
`'after publication'`. Require status 3 and the corresponding existing
`interrupted ...` diagnostic. Resume each with the interrupt variable absent,
`--resume`, the same parent and CONTROL: compare final record hash, accepted
reductions and attempts with the driver's completed inherited result; after
accepted, require no repeated committed attempt lengths in CONTROL. These are
existing controls used by `live-backend-minimize.py`; no new control is proposed.
For changed-context resume, copy the detached parent outside managed stores,
increment its valid context round and reseal that payload's bytes/SHA-256 in
`manifest.json` only. Keep record and source unchanged. Resume against the
completed inherited output with the changed copy: require changed-bound status 3.
Retain both raw contexts and manifests to show the source binding changed. No
negative may be replaced by publisher early refusal.

The driver also compiles the unchanged existing failing guest with
`-ldflags=-s -w`, using its original ordinary preparation provenance and the
existing stripped-build route. It retains a target failure through normal
unscoped choice exploration (eight executions, depth 1024), authenticates the
journal and exact retained module bytes, and detaches that failure. Explicit
scoped minimize must exit 2 naming the detached parent and `no_symbols`, create
neither output nor attempt-control file, and preserve the parent snapshot. This
case never replaces a module in an already retained artifact.

Golden review: the existing checked
`goldens-minimize/semantic.json` SHA-256 is
`d8f7795d5bb4c4d9a54b38e361dc8b5fe4e0b0065336d8d0671127650dbcc070`.
Keep it unchanged during both generations. Compare the two complete candidate
files byte-for-byte, then compare the existing `comparisons`, `controllers`,
`failure_groups`, `cluster_refusals` fields exactly to the checked file and retain
all existing named refusal/status pairs. Review the new complete
`scoped_controllers`, `scoped_reports` and additional refusal rows deliberately
before updating the checked golden. Do not discard source/context commitments
or acceptance fields to obtain agreement. Scope observation files contain fresh
absolute paths: compare their semantic values with paths resolved to and
authenticated against their respective raw artifacts; whole-file equality across
fresh roots is not appropriate.

Scheduling estimates, not measured performance: serial guest execution, two pairs
plus probes, budget 20–40 minutes per enhanced reference generation and 3–10
minutes per scope generation on warm caches. Historical reference outputs were
388 MiB per generation (94 top-level commands); old scope output was 79 MiB
before the new failure cases. Reserve 2–4 GiB additional disk for both enhanced
pairs/probes and allow 2–4 GiB host RSS pending measurement; configured guest
linear memory is 256 MiB, which does not bound Wasmtime host memory. Check free
space before starting and keep compile/Cargo activity stopped during the guest
wave. No new target or duplicated dependency build is needed. Record each actual
session/log/terminal exit; these estimates and source inspection are not runtime
acceptance. Integrated strict Clippy remains the conductor's gate.


## Automated public acceptance supplements

The scope driver now executes all supplements described above, rather than
leaving manual follow-up cases. It records every command/status/output digest
through its existing receipts, stores zero-match and interrupted-resume results
in its existing observations, and reuses the original exploration budgets and
existing fixture controls. The malformed-context opener now explicitly checks
that the diagnostic names the parent. The stripped failing guest uses the same
unchanged Go source as the scoped failure; only the existing stripped compiler
flags differ. The normal unscoped producer/journal authenticates its retained
module before the scoped minimizer negative. No new CLI, schema, Go guest, golden,
build target or runtime control was introduced. Python AST and diff checks pass;
these added actual cases await execution with the conductor's new integrated
fixture. No compiler, Cargo or guest command ran for this source-only checkpoint.


## First actual integrated scope attempt and module-root correction

Immutable fixture `e9d7b6715480f7afc7059a9924ad7210dfefc97c5860883291e0bf9407aa3439`,
source `00a97c42ee371656e9b3655a3056b521bac8d4ce`, ran scope01 under the
conductor's explicit resource grant. Exec session 97675, launcher PID 3247012,
driver PID 3247054, exited **1** after 26.78 seconds;
source/binary/input bindings were unchanged. Raw stdout/stderr and terminal
receipt remain in `/home/agent/wt/evidence/fn231-public-00a97c42/scope01`;
per-command outputs remain in sibling `scope-01`. No retry, cleanup or semantic
generation followed. The failed child command was `scope-failure`, status 3:
`--working-dir .../tools/gomad go-run ./testdata/parity/live_backend/minimization`.
That target is a nested module (`gomad.wasm.exploration.fixture`), so the outer
module correctly refused to list it.

The driver now uses the target's own working directory and `go-run .`. The
stripped compiler helper takes both directory and package directly from the
authenticated producing provenance, preserving the correct source coordinates
for both the outer-module site-scope guest and the nested failing guest. The
unchanged guest, budgets and runtime flags are retained.

A source-only regression extracts and executes the actual command-assembly AST
from the old and corrected drivers, checks the real nested `go.mod` boundary,
and executes the actual stripped-build helper against a recording compiler stub
to assert its cwd/package and original-provenance immutability. Old source
exits 1 on the wrong module root; corrected source exits 0 on both paths.
Evidence: `/home/agent/evidence/fn231-module-command-{red,green}.log` and
`fn231-module-command-regression.py`. Both driver AST checks and diff check
also pass. This causal source correction did not run Go, Cargo or a guest; a
fresh authorized public attempt remains necessary. The first failed receipts
remain immutable.


## Scope resolution cache correction

The fn230 completion review identified repeated symbol resolution of cached
offsets in `Resolution::observe`. Observation now filters missing and cached
offsets before calling the resolver, deduplicates new offsets, and skips the
resolver altogether when none remain. The shared trace observer still records
each trace's matching fingerprints; unavailable package observations remain
cached as `None`. A private injectable resolver seam enables a meaningful call
regression without global instrumentation or changes to persisted formats.

Focused RED on the former behavior exited 101: the third batch incorrectly
contained new 5384 plus cached 5383. GREEN exited 0, proving only unique new
offsets reach the resolver, unavailable offsets are cached, cached/missing-only
traces make zero resolver calls, and fingerprint updates still happen. The
affected site_scope suite and workspace format/diff checks also exited 0. Logs
are `/home/agent/evidence/fn230-site-cache-focused-{red,green}.log` and
`fn230-site-cache-suite-green.log`. Commands used release/locked/jobs2 on the
authorized warmed `fn232-retained-validation` target. No Go or guests ran.

The first attempted focused invocation mistakenly named a new `retained-validation`
target. It was stopped with exit 143 before tests (not counted as RED), and its
190134716 bytes were removed after its cargo/compiler processes exited and exact
private path/ownership were verified. The existing warmed target was preserved.
Correct target RED session 56991 exited 101; GREEN session 11814 exited 0.
The affected suite ran to immediate exit 0. No target process remains owned.


## Actual scoped guest decoder budget correction

The corrected nested-module scope attempt used immutable fixture `b22999a9...`,
binary source ece8d8c089 and driver/checkpoint HEAD f694270661. Scope01 session
70239 (launcher 3265681/driver 3265724) exited 1 after 29.16 seconds, bindings
unchanged; no retries or semantic generation followed. The real scoped base
completed successfully but stopped `site_scope_empty`: its site 5750 had cached
package `None`. Read-only retained inspection identified
`symbol_table_too_large`, rather than a wrong package pattern. Original receipts
remain in `/home/agent/wt/evidence/fn231-public-modulefix`.

The genuine retained failing-guest module is 2533702 bytes, SHA-256
`88414b72a129853c6f475bc1325ca658a8a2de69ee3f3da8bddfc1ac6ca1f5a2`,
with 27830 data segments and 1655 functions. A diagnostic checkout at exact
f694270661 reproduced the decoder failure without Go or guest execution:
`Module::read` passed with 1375531 steps; `Tables::read` exhausted the unchanged
16777216-step cap at 16777219 steps, 11417515 bytes and 117907 entries. The
preserved diagnostic stack identified `Memory::byte -> Memory::u32 ->
Tables::read`: each fixed-width integer searched the sparse segment index again
for every byte. No symbol-site resolution was reached.

`Memory::uint` now looks up each contiguous segment or zero-filled gap once and
copies that bounded run. It charges each material byte and the comparisons
actually performed. All numeric byte/entry/step caps are unchanged. Original
zero-length reads, zero-filled holes, adjacent segments, end/range/overflow
errors and fixed 16-byte metadata framing remain covered. Portable regressions
prove one lookup per contiguous read, exact charges and exhaustion, as well as
the sparse-boundary cases. There is no unbounded segment walk, budget increase,
pattern change, Go guest translation or acceptance-predicate change.

Causal RED session 14457 exited 101 on the actual authenticated module/table
cap. GREEN session 35013 exited 0 and resolved site 5750 to package `main`,
function `choose`, file `gomad.wasm.exploration.fixture/main.go`, line 6, with
inline caller `main.main`. Table parsing finished at 15436416 steps; complete
site resolution at 15445910 steps, 12309786 bytes and 129537 entries, within all
original caps. The first RED attempt (48128) had a test-only empty-array type
ambiguity; after typing `[0_u8; 0]`, the intended causal RED was reached.

Raw logs: `/home/agent/evidence/fn231-uint-retained-red-2.log`,
`fn231-uint-retained-green.log`, `fn231-uint-symbolize-suite.log` and
`fn231-uint-metadata-suite.log`. Entire symbolizer filter exited 0 with 50 tests,
including all six build-info framing/uniqueness/overlap/budget cases and the
forensic actual-module test. Metadata filter exited 0 with seven passed and one
existing ignored test. Workspace format and diff checks exited 0. The absolute
path diagnostic and all temporary budget tracing were removed from final source;
the exact forensic test is retained externally as
`/home/agent/evidence/fn231-uint-retained-diagnostic-test.rs`. Production source
and portable regressions await independent review and integrated strict gates.
No Go, guest or new fixture ran for this fix, and no global instrumentation was
committed.


## Standalone helper strict-lint cleanup

The conductor authorized fixing the standalone helper's 17 baseline Clippy
diagnostics in this isolated checkout after the reviewed uint fix. Single-store
execution now names its existing start and terminal-finishing blocks; cluster
execution names its existing initial state/sink construction and borrows its
Activity owner. Validation, watchdog, storage/model finishing, evidence and
metrics operations retain their original order. Streaming unit setup names the
existing model/namespace fixture. The remaining changes add numeric separators,
use equally strict nonempty assertions with useful failure values, remove a
redundant clone, spell imports explicitly and move retained-world imports/test
module mechanically. No minimizer or runtime controller changed.

After those library issues cleared, all-target checking exposed four preexisting
`similar_names` warnings in `tests/linked_streaming.rs`. The conductor separately
authorized renaming only the four receiver bindings; their channel actions,
assertions and order are unchanged. No lint allowances, dependencies, feature
changes, policy changes, guest changes or assertion weakening were introduced.

All commands used exclusive `fn232-retained-validation`, jobs2, +1.99.1, locked
release and standalone helper `--no-default-features`. Full helper library tests
exited 0: 185 passed, ten existing ignored. The seven existing linked-streaming
integration tests exited 0. Strict Clippy `--all-targets -- -D warnings` exited 0,
as did standalone format and diff checks. Logs are
`/home/agent/evidence/fn232-helper-cleanup-lib.log`,
`fn232-helper-cleanup-streaming.log` and `fn232-helper-cleanup-clippy-4.log`.
Earlier Clippy logs preserve the original baseline and intermediate extraction
diagnostics, including one private type path and one needless borrow corrected
before the passing gate. Library session 50748 and streaming session 96697 both
completed with numeric exit 0; no owned target process remains. No Go, public
guest, runtime controller or new fixture ran. Independent source review and
integrated checks remain with the conductor, including its separate pending
broker aggregate fix.


## Genuine legacy-parent provenance presence correction

The fresh semantic01 generation under immutable host 8336643f failed its first
shared-output reduction assertion: the CLI returned 0, unchanged, two rejected
attempts and `minimal`, at the original 6406161-byte bound. The bound was not
reached: publication-cap checks happen after a reduction passes the predicate.
Original raw generation remains in
`/home/agent/wt/evidence/fn231-public-storagewait-01/{semantic01,minimize-01}`.
Its successful 55-command scope01 remains separate evidence, not acceptance of
the failed semantic generation. No checked golden changed.

Focused frozen Go minimization of the exact original seed7 parent exited 0,
changed true in one attempt, accepting cutoff0. Existing host833 executions of
the same module/configuration/seed under cutoff0 and cutoff1 reproduced the
original Rust parent record byte-for-byte (28c00513...) and the identical raw
choice trace. Evidence is `/home/agent/wt/evidence/fn231-minimize-focused-01`;
Go session10956 and prefix capture53363 both exited 0. The module SHA-256 is
`1f89ca1f54fbcc46ff01f4793c83e979d84e1af3bc486233e235e99676382e7c`.

Controlled detached copies investigated only `{}` preparation-payload presence.
Two early construction errors (missing typed file `module` flag and private file
mode) were preserved rather than treated as causal evidence. The final correctly
typed/private copy, under `fn231-provenance-controlled-01`, passed actual833
`fixture-open` with full payload authentication, then minimize refused the
present `{}` payload with status2/missing schema before any attempt. That strict
admission refusal remains intact; no schema was forged to reach execution.

The decisive test produced a fresh genuine parent using the existing fixture
publisher, the unchanged original module/config values and the same freshly
compiled host as its candidate. RED38038 exited101 at strict `Parent::matches`.
Full canonical field comparison found exactly one difference: candidate streams
included `target/provenance.json` with bytes2, records1 and SHA-256
`44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a`.
Parent and candidate had identical outcomes, identities, environment, arguments,
all other streams and raw choice bytes. The Parent fallback `{}` was being
accidentally promoted into a new retained stream by ordinary publication.
Raw full records/diff are in `fn231-provenance-host-red-02`. Its predecessor test
run picked a target-pool entry rather than the manifest-bearing artifact; that
lookup failure was corrected before the meaningful RED and remains recorded.

The narrow Parent-local correction preserves authenticated absence when building
a trial. It removes only a generated payload exactly equal to canonical `{}`,
only when the authenticated original manifest has no preparation payload, and
only after checking the complete corresponding stream descriptor plus consistency
of publication record, ledger record and record.json. Unexpected payloads or
descriptors refuse. All three record representations are updated together.
Authenticated present provenance stays byte-identical, and strict full-record
`Parent::matches` has no exemptions. No minimization evidence schema, acceptance
predicate, byte cap, replay requirement, guest source or dependency changed.

Genuine same-build GREEN53027 exited0. Parent and cutoff0 candidate both hashed
to 458c254d... with strict match true and no extra streams. The same test ran the
actual full minimizer at attempts64/bytes6406161: changed true in one attempt,
then exact replay of its final artifact passed. Raw records, traces and result
are in `fn231-provenance-host-green-01`. The absolute-path forensic test was
removed from shipped source and saved externally as
`/home/agent/evidence/fn231-provenance-genuine-green-test.rs`.

Five portable regressions cover authenticated absent/present provenance, valid
typed ordinary provenance, preserved invalid-schema refusal, unexpected generated
payload/descriptor/coherence failures, unchanged strict binding of other metadata
and retained payload tampering. Final affected minimize tests exited0 (23 passed);
replay tests exited0 (7 passed). Strict fixture all-target Clippy with warnings
denied, workspace format and diff checks exited0. One test-only useless String
conversion was fixed before the passing Clippy run, without allowances. Logs:
`/home/agent/evidence/fn231-provenance-host-red-2.log`,
`fn231-provenance-host-green.log`, `fn231-provenance-minimize-tests-final.log`,
`fn231-provenance-replay-tests.log`, `fn231-provenance-clippy-final.log`.
Commands used +1.99.1 release/locked/offline/jobs2, exclusive fn232-live-corpus;
its removed library intermediates were regenerated under the explicit lease.
No Go compilation, full generation retry or golden overwrite ran. Independent
review, integrated gates and two fresh complete public generations remain with
the conductor.


## Positive display golden from the two completed scope runs

The literal task2 golden coverage now includes inspect's complete per-removed
`symbol` objects, alongside package/function final groups and minimize scope and
removal counts. `site-minimizer-display.py` reuses the existing campaign artifact
authenticator/canonical encoding, authenticates the actual record and all bound
payloads, checks the complete inspect minimization block against retained raw
metadata, and verifies CLI candidate/removed identities exactly before selecting
only explicitly declared presentation fields. No record, evidence, choice or
identity value is rewritten to make two runs match.

The two inputs were the already completed and authenticated
`/home/agent/wt/evidence/fn231-public-provenancefix-01/scope-01` and
`/home/agent/wt/evidence/fn231-public-provenancefix-02/scope-02`. Their projections
were checked byte-identical before the new checked write. Both are 938 bytes,
SHA-256 `4d50ea6bf4def5cee39e09755672f23d99dc52e2f62642b0e2090fc8a365549d`.
The new checked path is `goldens-minimize/scoped-display.json`. The checker also
enforces repeated equality before regeneration under the existing
`GOMAD_GOLDEN_REGENERATE=1` convention, requiring distinct authenticated scope
outputs. The live scope path strictly compares the golden after existing report
and raw-byte checks; it writes its observed display view to the work directory.
No new host CLI or runtime control is involved.

The portable copied input fixture retains actual full candidate identities and
raw minimization metadata; it omits only the external artifact path. Fourteen
offline tests cover positive main.choose/main.go:6/inline main.main symbols,
missing and changed symbols/groups, counts (including a balanced but wrong
classification), ordinals, exact identities, raw predicate metadata, scope and
the two-run regeneration gate. These compare copied actual output to the frozen
reviewed golden, rather than deriving expected symbols from checker logic.

Verification commands (all exit0):
`python3 tools/gomad/testdata/parity/site-minimizer-display-tests.py`;
`python3 tools/gomad/testdata/parity/site-minimizer-display.py --scope-output
/home/agent/wt/evidence/fn231-public-provenancefix-01/scope-01 --scope-output
/home/agent/wt/evidence/fn231-public-provenancefix-02/scope-02`; three Python AST
checks and `git diff --check`. Logs are
`/home/agent/evidence/fn231-scoped-display-portable-tests.log` and
`fn231-scoped-display-actual-check.log`. Independent before/after hashes of each
actual record, manifest, inspection block, choice and retained source tape are
unchanged, recorded in `fn231-scoped-display-proof.json`. Checker SHA-256 is
`d744cee766cdf1fa77215414e6bb12a83057c2d8aad6e2ef677e5ca509df1efb`;
portable tests SHA-256 is
`624f5ff9d7efe3cefa24ed6430f4bef29e271f5906b019591e57fc813f8be49b`.
The original `semantic.json` remains byte-identical in this isolated checkout
(SHA-256 d8f7795d...). No Rust, Cargo, Go, guest, complete generation, lifecycle
transition or push ran for this Python-only coverage addition.

### Display review followup: JSON type-sensitive raw-block authentication

The complete retained minimization block comparison now compares canonical JSON
bytes before selecting display fields. This closes Python equality accepting
`true == 1` and `64 == 64.0` in omitted predicate/budget fields. Two portable
regressions mutate the actual copied inspect predicate `replay_match` to integer
1 and `attempt_budget` to float 64.0 while leaving retained metadata unchanged.
Both failed with “ValueError not raised” before the fix (16 tests, 2 failures,
exit1), then all 16 passed after the one-line canonical comparison (exit0).
Logs: `/home/agent/evidence/fn231-display-types-red.log` and
`fn231-display-types-green.log`.

The offline authenticated two-scope checker passed (exit0), as did Python AST
parsing for checker/tests/live hook and `git diff --check`. The explicit
canonical-type proof and projection hashes are in
`/home/agent/evidence/fn231-display-types-proof.json`; actual checker log is
`fn231-display-types-actual.log`. Both actual projections remain 938 bytes with
SHA-256 `4d50ea6bf4def5cee39e09755672f23d99dc52e2f62642b0e2090fc8a365549d`.
Checked scoped golden and semantic golden are unchanged. Checker SHA-256:
`fd78be2e9e73eaf6fd5a77b4262e8f409a9fe691436bf869600ee5bf49e5eea2`;
tests SHA-256:
`f884829d0e6ad448d0cade2bf83176bb86cbb48cdcfa1c3a1cc472bfa89f84bc`.
This followup ran no Go, Cargo, Rust, guest or runtime generation commands.

### Integrated reporting closeout

Integrated display checkpoints a4cfdf5ab7/c63cfc2999 passed all16 portable tests, authenticated both retained actual scope outputs, AST3 and diff checks (all exit0). Independent same-family reviewer returned SHIP for6506164bda after inspecting actual RED/GREEN logs and unchanged938B projections. Flow task2 was completed and its done status independently read back before this closeout commit. The checked semantic golden is the actual repeated3095bc851f9fa9f821dac792e585f16228975a0ebd0d8dd0a0cafeaedf05f61e output (38237B); previous fields and refusal entries remain unchanged, with only scoped cases/report blocks added. No Rust/Go inputs changed after the four acceptance phases. Formal full-spec completion review remains pending.

### Final spec completion review and closure

Formal flowctl codex completion-review returned SHIP for all R1–R8 at source5fc1290c3dcc913bbe3f0b88074dc801eacd09a4, base0be13696d3450558bdc094c53777615bc81598d8 (immediately before scoped feature integration). Reviewer gpt-6.1-sol/high, same family, independently ran16 display tests and authenticated both retained full outputs/goldens; its attempt to rerun Cargo was refused by read-only build locking, so Rust verification remains the already retained integrated156/158 passing suites and strict checks, not a new review-run claim. No gaps or unaddressed R-IDs. Receipt artifactb5b50dd6f922c4bb3928a832322e1edfaf9af31cafa301ed4eed2bda30d7aa68; outputSHA477ab259dbd1079c7a50dca0abb37c96660e1fd288a7f886ae62df63231436e7. The first pre-dispatch attempt on repository-wide base951 refused a diff over64MiB; no verdict or round was consumed, and the complete feature base was used without truncation. Flow completion_review_status=ship and spec/task statuses=done were read back after spec close. Remaining full rewritten-host qualification belongs to openfn232.13 and was not marked passed.
