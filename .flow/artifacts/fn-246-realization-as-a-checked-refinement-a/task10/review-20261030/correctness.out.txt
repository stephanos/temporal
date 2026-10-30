Review complete. I traced the frontier search end to end (root seeding, 0/1 BFS ordering, cursor accounting, first-match retention vs. exhaustive resolution, blocked/depth handling, branch-wide proposal/answer/finality agreement, atomic commit via `Change::Run.frontier`) against the fixture and the Go runner expectations, and checked the proposal re-validation against the live Temporal path.

## Findings

**Severity:** P3
**Confidence:** 100
**Classification:** introduced
**File:Line:** `model/umpire/protocol/runtime/src/runtime.rs:56`
**R-IDs:** []
**Problem:** `SingletonTarget<S>` is now a blanket impl over every `ReachTarget<S>`, so the `R::Target: SingletonTarget<M::State>` bounds on `Runtime` and `Dispatch` are vacuous. The trait and the bounds are leftover machinery with no remaining role.
**Suggestion:** Either delete the trait and the bounds, or keep it only as a documented public alias. Not blocking.

**Severity:** P3
**Confidence:** 75
**Classification:** introduced
**File:Line:** `model/umpire/protocol/runtime/src/frontier.rs:174`
**R-IDs:** []
**Problem:** In `expand_event`'s Proposal arm and in `check_proposals`, a branch whose state makes `Realize::propose` return an `Err` (for example the fixture's `Quiescent` for a closed phase while the published state is still Active) propagates that realization error as the Judge/Propose refusal instead of the `disagreement()` State refusal. The run is still not committed, so semantics are safe, but the exposed finding is "countdown closed" rather than "compatible branches disagree", which hides the real cause from consumers.
**Suggestion:** Map a per-branch realization error inside branch-wide decision checks to `disagreement()` (or wrap its message) so the refusal names the divergence.

**Severity:** P3
**Confidence:** 75
**Classification:** introduced
**File:Line:** `model/umpire/protocol/runtime/src/frontier.rs:140`
**R-IDs:** []
**Problem:** A depth-blocked node can never be resolved (its partial path already exceeds `scope.depth`), yet first-mode Judge retains it forever. Every later `resolved_frontier` (Propose/Answer) and every later no-match Judge then refuses with Budget until Reset. This matches the spec's "unresolved work is refused without branch loss", so it is not wrong behaviour, but a single over-depth compatible path permanently disables proposals for the run.
**Suggestion:** Consider dropping blocked nodes whose block is `Depth` from `resolved_frontier` only when the realization explicitly declares depth as a hard bound, or document the behaviour. FYI only.

FYI (not findings): `search_journal` runs `validate_prefix_data` on every popped node, so each Judge is O(nodes × prefix) including realization calls; fine for current bounds. In `tests/support/control.rs`, a Reset with no configuration while the frontier runtime is installed is forwarded to that runtime rather than restoring the model runtime; test-support only.

Verified sound: retained typed facts/outcomes via shared `Rc<Evidence>` without Outcome/Fact Clone bounds; cursor vectors match `valid_observation`'s `edges+1` contract; `continues` agreement filter for non-latest events plus exhaustive agreement check when pending; frontier/prefix/pending only change on commit; `BuildHasherDefault<DefaultHasher>` is seedless so the WASM `random_get` import is gone; proposal re-validation is deterministic for the live Temporal path because `install_context` replaces the realization before any inner Proposal event exists.

## Requirements coverage

| R-ID | Status | Evidence |
|------|--------|----------|
| R1 | met | Bounded frontier on `umpire_checker::search::Traversal` (`frontier.rs`), retained alternatives and unexamined work, whole-prefix immutable reports, branch-wide decision checks, failed publication/allocation/reset/finite-bound controls in `tests/main/frontier.rs`, Go runner/host frontier parity and replay mutants in `runner/frontier_test.go`. Temporal guest publication explicitly not claimed by this task. |
| R5 | partial | This task only supplies the generic latent-history retention R5 relies on; the full protocol Model/TLC obligations are assigned to .246.6 and are out of this task's scope. |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 3 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":3,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>