# fn-228.2 independent audits

### Correctness axis

## Quality Audit — Correctness axis: fn-228.2

### Summary

- Reviewed `bbd870394c..0a91a7b99d`, 38 changed files.
- Critical 0 · Should Fix 0 · Consider 0 · Ship: ✅ Ship, subject to outstanding required gates.
- No correctness findings survived review. Evidence: **walked** production execution paths and test assertions; tests were not rerun during this read-only audit.

### What’s Good

- **walked**: Bootstrap correlation, per-Run receipt retention, configured Reset, pending requests, callback context and publication rollback follow the settled contract. Poll completion uses the typed pending action; Answer preserves pending requests.
- **cited**: Wire vectors preserve method 6, reserved response tag 7, Annotate tag 8 and Propose tag 9. Go rejects unknown/unspecified proposal kinds. Canonical activity Model and DSL have no changes.

### Test Budget

- Approximately 972 added test/vector lines against 734 implementation/schema lines, excluding generated code and shared fixture support; below 2:1 before subtracting test setup.
- Existing test edits preserve behavioral assertions. Replacing the reserved-Propose rejection test matches the newly supported feature.

### Outstanding Validation

- **cited**: `.flow/tmp/fn-228.2/native-reviewed.log` and `focused-suite.log` contain passing focused results.
- Required post-edit default/full gates and fresh live/privacy proof remain pending; this audit establishes no passing claim for them.

Reviewer and writer are from the same model family. Working tree was not modified.

### Standards axis

## Quality Audit — Standards axis: fn-228.2

### Summary

- Files changed: 38 · Should Fix 1 · Consider 0 · Blocking: none possible (standards axis).
- Scope: `bbd870394c..0a91a7b99d`. Same-family review: writer gpt-6.1-sol medium, reviewer gpt-6.1-sol high.

### Should Fix

- **model/temporal/realization/macros/src/snapshots/umpire_temporal_realization_macros__tests__activity-realization.snap:1001** (Conf 100): Generated snapshot crosses the mechanical size threshold, 817 → 1,033 lines (executed count). Split the harness at parsed `impl` boundaries: authored implementation starts at 5, metadata at 369, trait implementation at 871; regenerate separate snapshots and account for every parsed item.

### Evidence and limits

- Cited: `.flow/tmp/fn-228.2/` native-reviewed, focused-suite, go-parity, guest-build, generic-lint, temporal-lint and final-negative logs/receipts report rc0. This auditor did not rerun tests or generation checks.
- Full gates and fresh live proof remain pending; no conclusion about their results.

### What's Good

- Walked: generic Runtime and Host retain opaque payload boundaries; Temporal request selection, correlation and polling completion remain with the realization.
- Cited: additive defaults and configuration behavior are documented; controls reuse the shared Runtime and exercise publication failure, pending proposals and callbacks between polls.

No other concrete standards findings. The size finding concerns generated test output; no handwritten production file crosses 1,000 lines.

Correctness: 0 findings; worst tier none.

Standards: 1 finding; worst tier Should Fix.

## Conductor disposition

Accept the standards recommendation as a test-harness change: split the complete parsed expansion at its natural implementation boundaries, account for every item, regenerate snapshots and retain the same assertions. No production code change is requested by these audits. Final gate and live-proof claims remain pending.
