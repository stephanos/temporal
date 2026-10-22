# fn231 Rust minimizer batch handover — 2026-10-22

Source checkpoint: `2ac50dcabf`, branch `fn231-rust-minimizer`, base
`e5ea438c90`. Both fn231 tasks have source implementations in one coherent batch.
Task completion and predecessor acceptance remain with the conductor. This is
provisional implementation against the integrated fn232.9/.11 and site APIs;
it does not accept fn232.11 or claim runtime/qualification results.

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
