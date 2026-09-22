---
satisfies: [R5]
---
# fn-22-deterministic-replay-semantic.5 Implement the bounded monotonic minimizer over the bridge

## Description
Implement the Go client of the replay bridge (one request outstanding, exact echo, byte caps, as `tools/umpire/campaign.Bridge`) and the reducer: for each candidate the bridge hands out, two fresh reruns through task .3, retained only when both reproduce the subject's key, then reported to the bridge; fixed limits (eight edits, twelve Runs in all, one active Run, 25 minutes, Case, event and report bytes) checked before preparation or dispatch; cancellation stops new work, the active Run follows fn-64 semantics and a lost Run is named. Completion is the bridge's: `minimized`, `irreducible` or `incomplete` with the limit that ended it. Determinism: the same scripted classes twice give the same decisions and the same report bytes.

### Approach
- The reducer is a consumed-value state machine like `campaign.Session`; the bridge client shares `campaign`'s transport where fn-33 left it reusable.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/replay/`

**Size:** L
**Files:** `tools/umpire/replay/minimize.go`, `tools/umpire/replay/minimize_test.go`, `tools/umpire/replay/bridge.go`, `tools/umpire/replay/bridge_test.go`

### Re-plan note (2026-09-22)
Rewritten on fn-85, fn-86, fn-87 and fn-33 after the first plan's MAJOR_RETHINK; see the spec's **Re-plan** section. Start only after the spec's fresh plan review.
## Acceptance
- [ ] Edit, Run, wall-time, Case-byte, event and report limits are enforced before the work they bound, and `incomplete` names the limit.
- [ ] Compile or preparation rejection, not-reproduced, indeterminate, cancellation and limit exhaustion stay distinct in the report.
- [ ] The same inputs and classes give the same reduction decisions and report bytes; a retained candidate never reintroduces a dropped step.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
