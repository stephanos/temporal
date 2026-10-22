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
| R1 | Reference content controllers show stable out/mixed/in ordering and changed attempt counts; public prefix cases preserve suffix-only behavior. Direct minimizer invalid-pattern probes below supplement the exploration parser probes. Scoped stripped-parent refusal remains covered by the shared-opener/source units; the scope driver additionally proves actual stripped exploration refusal. |
| R2 | Scope driver authenticates every retained pre-round context, copies an actual `target_failure`, deletes its producing campaign, then minimizes the detached parent with inherited `main`; explicit `runtime` override and malformed-present context status 2 are actual CLI cases. Confirm diagnostic names detached parent. |
| R3 | Controller scoped/unscoped predicate results and attempt counts; actual minimized failure replays reproduced. Do not require identical greedy content minima in general. |
| R4 | Actual completed resume agrees and changed explicit scope refuses 3; reference lane exercises bound/implementation/parent/predicate checkpoint corruption. Add interrupted scoped resume and changed valid context probes below. |
| R5 | Actual minimized inspection contains removed-decision symbols and final grouping totals. Reference lane detached source reporting authenticates source payloads. Unit fixtures retain stripped and missing-site unavailable reasons and paired-payload tamper negatives. |
| R6 | Actual JSON inherited/override scope and per-reduction inside+outside totals equal removed counts; controller reports independently cover classifications. |
| R7 | Inspect leaves retained inspection bytes unchanged; actual replay and frozen failure metadata stay fixed. Journal publication/reopen and qualified wrong-context refusal, same-record/different-context collision and source candidate commitments are the already passing meaningful unit regressions, not invented public publisher commands. |
| R8 | Frozen unscoped Go/Rust prefix and content controller comparisons, repeated records and reductions stay exact; compare existing golden fields before accepting scoped additions. |

After each scope driver, its real detached parent is exactly
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

The direct probes and stripped-parent unit coverage are stated separately because
the current two drivers do not themselves execute all these cases. A dedicated
actual stripped failing parent/minimize probe, if required for final public
coverage, needs the existing guest compiled with `-ldflags=-s -w` and ordinary
prepared provenance before discovery; reusing the symbol-bearing retained parent
with a substituted module would test identity corruption instead of R1. Do not
claim that actual case from the current stripped exploration probe.

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
