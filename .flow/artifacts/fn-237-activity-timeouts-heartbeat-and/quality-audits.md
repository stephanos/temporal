### Correctness axis

## Quality Audit — Correctness axis: fn-237.1 integrated patch

- Files changed: 47, including new fixture artifacts · Critical 0 · Should Fix 1 · Consider 0 · Ship: ⚠️ Fix first

### Should Fix

- **model/umpire/front/src/body/effects.rs:344** (Conf 75): `fact_result` accepts `::Option::Some(Fact::A)` and `::Option::None`: stripping the leading `::` matches the whitelist, then skips shadow checks. These absolute paths do not identify standard Option constructors — allow absolute paths only through canonical `core::option::Option` or `std::option::Option`, and add both source refusal controls. Evidence: walked.

### Validation

- Evidence: walked native dispatch, SSA lowering, interpreter decoding/evaluation, both TLA paths, and exact replay controls; no additional correctness findings.
- Evidence: cited focused green log and required-mode optional-fact witness/mutation successes; no tests executed during this audit.

### Test Budget

- Existing test assertions were preserved. Added controls extensively cover enabledness, rejection, fact clearing, typed decoding, and independent source mutations; gross test additions include substantial fixture/setup code.

Raw report returned verbatim for persistence. No file written because the auditor charter prohibits repository writes. **Finding count: 1; worst tier: Should Fix.**

### Standards axis

## Quality Audit — Standards axis: fn-237.1

### Summary

- Files changed: 47, including new fixture/control artifacts · Should Fix 0 · Consider 0 · Blocking: none possible (standards axis).
- No reportable standards findings.

### What's Good

- **Walked:** declaration admission, SSA lowering, interpretation, and both emitters follow the approved ownership map and reuse existing typed sequences and enum decoding.
- **Walked:** scope remains bounded to handler facts; no new checker, TLS recorder, general Option IR, or Activity policy.
- **Executed:** line counts show no changed file crossing 1,000 lines; the tracked diff contains no changes to existing golden artifacts.
- **Cited:** focused success log read; no tests or builds run during this audit.

Report not saved: the auditor’s higher-priority read-only contract prohibits repository writes. Intended destination: `.flow/tmp/fn2371-quality-standards.md`. Findings: **0**; worst tier: **none**.

Correctness: 1 finding; worst tier Should Fix. Fixed with a red-to-green refusal control for absolute short Option constructors.

Standards: 0 findings; worst tier none.
