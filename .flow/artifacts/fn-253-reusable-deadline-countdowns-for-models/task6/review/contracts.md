Everything checked out. Summary of verification before the verdict:

- Every row of the lifecycle table matches `model/temporal/feature/activity/model.rs`: `initial` (146-164), Poll (187-199), Heartbeat (215-222), `retry` (289-298), `clear_terminal` (314-320).
- Tick priority ScheduleToClose, StartToClose, Heartbeat, ScheduleToStart matches `deadline` (239-283); `decrement` saturates (323-355); `expire_attempt` (301-311) matches the "retries only when not pending and another attempt remains, otherwise TimedOut" rule, which is the fn-246.13 correction, not the superseded cancel-wins policy.
- Pending rules: Cancel in Started only sets the bit (171-173); retryable failure with pending closes Failed (205-207); RespondCanceled closes Canceled; RespondCompleted still wins.
- All 16 named owners exist with the stated kinds (7 helpers, 3 `#[invariant]`, 6 `#[postcondition]`).
- `None` vs raw defaults: all arming is guarded by `is_some()`; the `unwrap_or(Ticks(1))` is never reached on an absent budget, so "neither substitutes a timeout for `None` nor clamps" is accurate.
- Links: design anchor `#temporal-layer-and-independence` exists at `model/design/deadline-countdowns.md:96` and does choose the feature README as authority with `time_out`/`time_out_attempt` as later effects; protocol model, compatibility README, tasks .7/.10 exist; both in-file anchors slug correctly. go.mod line 76 matches the pinned version; the three URL line ranges match the existing fn-246.15/.13 citations.
- Scope: only the README and `.flow` evidence changed; nothing under `model/umpire`, no `.rs`, no protocol dependency.

## Findings

**Severity**: P3
**Confidence**: 100
**Classification**: introduced
**File:Line**: `.flow/artifacts/fn-253-reusable-deadline-countdowns-for-models/task6/handoff.md:7`
**R-IDs**: []
**Problem**: Handoff says task .6 "still has a stale second **Touches** list of provisional capabilities/front paths", but the refreshed task file (`.flow/tasks/...6.md:6,13`) now lists only the README, and `custody.json` in the same commit says both Touches signposts match. The two evidence files disagree.
**Suggestion**: Drop or update that sentence in handoff.md so the evidence set is self-consistent.

**Severity**: P3
**Confidence**: 75
**Classification**: introduced
**File:Line**: `model/temporal/feature/activity/README.md:134-141`
**R-IDs**: [R3]
**Problem**: R3 asks for "pinned API and docs references already used by fn-237/fn-246.15". The new section links the pinned API directly but reaches the public docs reference only indirectly, via the anchor to the fn-246.15 section. It is met, just one hop away.
**Suggestion**: Optional: add the existing `docs.temporal.io/encyclopedia/detecting-activity-failures#start-to-close-timeout` link in the authority paragraph.

## Pre-existing issues (not blocking this verdict)

- [P3, confidence 100, introduced=false] `model/temporal/feature/activity/README.md:151` — the fn-246.15 section cites `service_grpc.pb.go:266-271` while the fn-246.13 section, the new section and `api-authority.json` all place RequestCancelActivityExecution at 2771-2777. The old range looks like a stale citation.

FYI: the Heartbeat-before-ScheduleToStart ordering in the priority list is vacuous, since `deadline_arming` forbids both being armed at once. The statement is still true.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | deferred | Task .2, already closed; design verdict at `model/design/deadline-countdowns.md:96-102` |
| R2 | deferred | Tasks .3-.5; `countdown.rs` exists, not in this diff |
| R3 | met | One named section in the chosen README, authority links to the pinned API and the approved corrections, all rules match the current Model |
| R4 | deferred | Tasks .1/.7/.9; README states adoption remains .7 |
| R5 | deferred | Task .8 |
| R6 | deferred | Task .10; README defers flag removal there |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 2 introduced, 1 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":1},"unaddressed":[]}
```

<verdict>SHIP</verdict>