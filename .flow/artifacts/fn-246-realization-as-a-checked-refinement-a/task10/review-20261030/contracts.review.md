## Review: bounded compatible whole-prefix frontier (9212a1c..0fc1c59)

I read the task spec, the full diff, the current `frontier.rs`/`runtime.rs`/`search.rs`, the test support fixture, the guest control dispatcher, and the Go runner/host tests and helpers. I hand-traced the countdown fixture through every runtime test and both Go tests (RENEW→GLOBAL, RENEW→LOCAL, question at prefix, reduce after GLOBAL_POLL, the three mutants) and the traces agree with the asserted witnesses, cursor vectors, `continues` values, and refusal codes. No shell was available, so pass/fail claims rest on the committed logs plus trace verification.

### Findings

**1. Per-node whole-prefix revalidation is not charged to any declared bound**
- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/umpire/protocol/runtime/src/frontier.rs:111-118`
- **R-IDs**: [R1]
- **Problem**: Every popped traversal label runs `validate_prefix_data` over its entire prefix. That walk calls `Realize::observe`, `reach_evidence` once per edge-prefix (quadratic per Reach event), `answer` and `propose`/`bootstrap`, none of it charged via `traversal.charge()`. Cost per Judge is O(labels × prefix × evidence), outside the `work` ceiling the contract names as the resource bound. The nodes being validated were built from already-validated roots through `checked_step`/`finish_event`, so the check is redundant with construction; the committed prefix is already validated once in `judge`/`propose`/`answer`.
- **Suggestion**: Validate the committed prefix once (already done) and either drop the per-label call or validate only roots copied from `self.frontier` before expansion. If per-label validation is kept deliberately, charge it to `work` so the declared bound is honest.

**2. Preflight note references logs that are not in the committed artifact set**
- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-246-realization-as-a-checked-refinement-a/task10/review-preflight-20261030/runtime-guest-import-fix.md:7-9`
- **R-IDs**: []
- **Problem**: The note says the original failure is preserved in `tests-go-frontier-final.log` and the post-fix Go pass in `tests-go-frontier-deterministic.log`. Neither file exists in the directory; only `tests-go-frontier-published-native-final.log` was committed. The claimed proof trail for the `random_get` regression therefore points at missing evidence.
- **Suggestion**: Either add the two referenced logs or correct the references to the files actually retained.

### FYI (non-blocking observations)

- **Depth-cut nodes are permanent.** A node blocked by `scope.depth` is re-rooted on every search, never expanded, and makes every later non-first search (`resolved_frontier` for Propose/Answer, or any Judge while pending) refuse Budget until Reset (`frontier.rs:140-142`). This matches "resource bounds are Refused without branch loss", but it means a single depth cut on an irrelevant branch disables Propose for the rest of the Run. Worth stating in the `Scope.depth` doc.
- **`Projection::Candidates` semantics changed.** Previously the first replaying candidate won; now every replaying candidate becomes a retained branch (`frontier.rs:250-269`), so a realization declaring two viable candidate paths will later hit "disagree" on Propose/Answer if the branches diverge. In-tree consumers pass, but the `Projection::Candidates` doc still reads as an ordered preference list.
- **`latest` in `resolved_frontier`.** The last committed observation is treated as `latest`, so `finish_observation` does not prune branches whose recomputed `continues` differs from the committed value; agreement is only checked among completed branches. Unreachable with a deterministic realization, only observable after `replace_realization`.
- **`Traversal::new(scope.prefix, scope.work)`** uses the prefix bound as the traversal depth, so an overflow surfaces as "Reach search depth exceeds limit 128" rather than "prefix bound 128". Both are Budget; cosmetic.
- Fixture mutants named in the acceptance ("erased budgets", "weakened universal-law") are not pinned as explicit tests here; the Go mutants cover greedy past, latest-only and wrong earlier report, and `RejectRenewFact` covers changed fact.

### Requirements coverage

| R-ID | Status | Evidence |
|------|--------|----------|
| R1 | met | Frontier on `umpire_checker::search::Traversal` with retained pending/blocked work; whole-prefix Explanation from Reset; typed `Evidence.facts` retained and revalidated via `reach_evidence`; atomic `Transaction` carrying prospective frontier; branch-wide propose/answer/finality agreement; Unexplained only on empty traversal; Go question/reduce/replay/publication tests; guest/native transcript parity for the new `frontier` transcript. |
| R5 | deferred | Task owns only the runtime side; full-Model native fixtures and TLC agreement are assigned to .246.6 per the task's focus statement. |
| R2, R3, R4, R6, R7 | deferred | Owned by other tasks of the epic; not claimed by this task. |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 2 introduced, 0 pre_existing.
Protected-path filter: 0

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>