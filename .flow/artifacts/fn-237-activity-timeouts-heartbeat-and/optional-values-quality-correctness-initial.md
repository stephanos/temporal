AXIS: correctness

## Quality Audit — Correctness axis: e2f25dc292..58027c27b8

### Summary

- Files changed: 92 · Findings: 1 · Critical 0 · Should Fix 1 · Consider 0 · Ship: ⚠️ Fix first

### Should Fix

- **model/umpire/backends/tlc/src/module.rs:143** (Conf 75): Coordinated CFG mutations can certify another selected World. Change `CONSTANT __umpire_world_remaining = 0` to legal code `2` in main and every isolated CFG; cross-CFG equality passes, while `Bound::check_module` checks unchanged metadata only. The emitted countdown references this constant solely in its declaration, so states, verdicts and replay remain identical. Validate main CFG assignments against typed `metadata.binding.world` and parameter symbols before comparing families.

### Test Gaps

- Add a coordinated all-CFG legal-code mutation using `world_remaining`; require refusal despite unchanged exploration.

### Evidence

- **walked**: Bound metadata check → TLC schema/CFG checks → runner → state agreement/replay.
- **executed**: Read-only inspection confirmed `__umpire_world_remaining` occurs only in its emitted declaration. No builds/tests ran; full gates and primary review remain pending.

### Test Budget

- 1,166 test-file additions versus 1,683 other Rust additions, including inline tests and fixture setup. Existing test edits show no weakened assertions.

### What’s Good

- Exact optional-code validation preserves absence versus exhausted zero; all four new expression kinds have traversal and priming cases.
