# Suite search follow-ups (model)

## Conversation Evidence

> Fresh re-review of commit df992fd8e8 (fn-213's complete waypoint search), 2026-10-20: verdict ship; no correctness defects across 6,000 randomized cases against an independent reference; three follow-ups.
> user (2026-10-20): "wrap up current tasks and then stop" — so the follow-ups are recorded here instead of fixed.

## Goal & Context

`find` and `Run::along` in `umpire-suite` search one product space of (Model state, partial waypoint matches) through a wrapper world explored by the checker (`model/umpire/suite/src/search.rs`). The search is complete and shortest overall, but three behaviours are wrong or surprising.

## Acceptance Criteria

- **R1:** A panic inside a waypoint predicate is caught around `Waypoint::judge` (in `arrive`, for starts and steps alike) and reported as a distinct refusal naming the waypoint index, never as a Model action panic. Fixture pins it.
- **R2:** The budget for `find`/`Run::along` counts Model states again: explore Model states once under the space's budget, then run the pattern search over the explored graph keeping the best progress per state; the cut rule carries over. `Run::along` gains a way to set the budget, or documents why not. `Space::budget` doc and README agree with the behaviour. A fixture or probe shows a large bounded pattern answered without blowing the budget.
- **R3:** A skipped step panic refuses an answer only when it can affect it (panicking state's depth below the answer's length); otherwise the answer stands. Fixture pins both sides.
- Cosmetic, same task: fix the "Index 0 always waits" comment, give completed product states no successors, deduplicate starts before counting them in refusals.

## Boundaries

- No change to `path!` grammar or reading rules; the search stays complete and shortest overall.

## Decision Context

2026-10-20, root agent: recorded at the owner's stop request; placed after fn-225 and before fn-218, since fn-218's `cover` builds on the same search.
