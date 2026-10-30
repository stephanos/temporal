## Review: bounded compatible whole-prefix frontier (9212a1c..0fc1c59)

I read the task, the full diff, the current `runtime.rs`, the shared `Traversal`, the Kettle realization, the Go runner helpers and the epic's acceptance criteria. I traced the countdown fixture by hand through `search_journal`, `reach_frontier` and `finish_observation` for the RENEW, GLOBAL, LOCAL and complete-exhaustion sequences, and the results match every expectation in `tests/main/frontier.rs` and `runner/frontier_test.go`. The main integration risk, retained alternatives exploding or disagreeing for existing singleton consumers, does not materialize: Kettle's `check_evidence` rejects every path that is not exactly one authorized edge, so its unexamined copies die on first expansion, and `Projection::Current`/`Candidates` realizations keep a one-node frontier. The `first = self.pending.is_none()` asymmetry is sound because with no pending request every branch computes `continues = false`, so no finality disagreement is possible on the early-return path.

### Findings

**Severity:** P2
**Confidence:** 75
**Classification:** introduced
**File:Line:** `model/umpire/protocol/runtime/src/runtime.rs:596-617`
**R-IDs:** [R1]
**Problem:** Quiescence is decided from the canonical state alone, before any branch check. `external` is computed only against `current`, and if `Realize::propose(current)` returns `Err(Quiescent)` the refusal is returned before `check_proposals` runs. Scenario: a frontier with two compatible completed nodes where the canonical node's state has no enabled external action (or its `propose` refuses Quiescent) while a hidden alternative is still Active; the runtime publishes Quiescent and the Go runner terminates `TERMINATION_QUIESCENT` on the strength of an arbitrary hidden budget. The countdown fixture cannot hit this because phase is pinned by the observation target, which is why no test catches it.
**Suggestion:** Make the quiescence decision branch-wide like the others: evaluate external enablement and the `propose` result on every node from `resolved_frontier()`, and return `disagreement()` when they differ, before reporting Quiescent.

**Severity:** P3
**Confidence:** 100
**Classification:** introduced
**File:Line:** `.flow/artifacts/fn-246-realization-as-a-checked-refinement-a/task10/review-preflight-20261030/runtime-guest-import-fix.md:38-40`
**R-IDs:** []
**Problem:** The note says the original `random_get` failure is preserved in `tests-go-frontier-final.log` and the post-fix pass in `tests-go-frontier-deterministic.log`. Neither file is in the committed directory; only `tests-go-frontier-published-native-final.log` is. `tests-guest-sha256.txt` likewise names `.flow/tmp/...` scratch paths instead of the committed `.wasm` files.
**Suggestion:** Commit the two cited logs or correct the references to the files that exist, and rewrite the sha256 manifest against the committed artifact paths.

**Severity:** P3
**Confidence:** 75
**Classification:** introduced
**File:Line:** `model/umpire/protocol/runtime/src/frontier.rs:518-521`
**R-IDs:** []
**Problem:** `bound_frontier` pre-reserves two hash sets with capacity `scope.prefix` on every search. `prefix` was previously only compared, so a realization declaring a very large bound would now turn every Judge into an Allocation refusal. No in-repo consumer exceeds the default 1024, so this is a latent footgun rather than a live failure.
**Suggestion:** Reserve lazily or cap the reservation at the number of items actually encountered; the bound check itself does not need pre-reservation.

### FYI (non-blocking)

- `validate_prefix_data` runs on every popped traversal label (`frontier.rs:112-119`), revalidating the entire prefix with uncharged realization calls. Cost is O(labels × prefix) per Judge; fine for the fixtures but worth noting for wide Reach frontiers.
- Root-level `EvidenceMatch::Stutter` with zero edges now yields an immediate zero-edge explanation (`frontier.rs:296-306`); the old `reach` only short-circuited on `Match`. `valid_observation` already accepted this shape, so it is consistent, just a behavior widening.
- Multiple matching `Projection::Candidates` paths are now retained as compatible branches instead of first-wins, so later Propose/Answer must agree across them. This follows the spec; existing consumers are unaffected.
- `SingletonTarget` is now a blanket impl over every `ReachTarget` (`runtime.rs:56-59`), so the `R::Target: SingletonTarget<M::State>` bounds on `Runtime` are vacuous. Not added by this change, only hollowed out; could be dropped later.
- Code-smell baseline: `reach_frontier` and `search_journal` are long methods with several nested early-exit paths, but each branch maps to a distinct spec rule and I would not split them for this change.

### Requirements coverage

| R-ID | Status | Evidence |
|------|--------|----------|
| R1 | partial | Generic bounded frontier on the shared `Traversal`, immutable earlier reports, typed Facts retained, atomic commit, named depth/work/history refusals, complete-exhaustion Unexplained, Kettle complete-state matching and native/guest/Go parity are all implemented and tested. The Temporal activity-specific pinning of the global=3/heartbeat=2 mapping and the joined Temporal guest publication are explicitly deferred by the task note and absent from this diff. |
| R5 | deferred | Task body only states R5 cannot compensate for a dropped history; the full-state live/TLC umbrella is assigned to .246.6 per the task's focus note. No R5 work is in scope here. |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 3 introduced, 0 pre_existing.
Protected-path filter: 0

```json
{"classification_counts":{"introduced":3,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>