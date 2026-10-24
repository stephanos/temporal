---
satisfies: [R11, R12]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.15 Align static Go inventory with the executed selector

## Description
Size: S

**Touches:** `tools/gomad/rust/driver/static_stages.py`, `tools/gomad/rust/driver/tests/test_static_stages.py`

### Files / Touches
- tools/gomad/rust/driver/static_stages.py
- tools/gomad/rust/driver/tests/test_static_stages.py

### Approach
Make Execution.go and authenticate_result reconstruct the same Go-owned listing selector as the actual action. The replay-regeneration action selects ^TestReplayPlanConformance$, but both listing paths currently inject -list .; the actual C4 run lists 36 roots and correctly executes one, then strict verification rejects the other 35. Preserve platform-aware Go enumeration, package/build-tag scope, repetitions, exclusions, command custody and independent raw verification. Do not weaken verify_go or infer selection by filtering observed execution after the fact.

### Bounded required reads
Read this spec R11/R12, task .13 owner acceptance, static_stages.py recipe/replay action, parse_go_inventory, go_scope_exclusions, verify_go, Execution.go and authenticate_result Go branch, and their existing tests; consult only the platform-aware Go enumeration memory relevant to these functions.

### Focused tests
Run the existing test_static_stages.py suite with its established driver test invocation, including producer and independent authenticator regressions for restricted and unrestricted selection, missing selected execution and duplicate roots. No full cycle or build is required in this fix task.
## Acceptance
- [ ] Replay regeneration lists only its Go-selected TestReplayPlanConformance inventory and accepts the authentic one-root pass without demanding 35 unrelated roots.
- [ ] Unrestricted Go actions retain complete platform-aware inventory, package/tag scope, repetitions and existing exclusions.
- [ ] Producer and independent reader agree on exact listing/run argv and still reject missing selected execution, duplicate listed/executed roots and changed command receipts.
- [ ] Focused static-stage regressions pass; no corpus expectations or .13 cycle/evidence acceptance are weakened.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
