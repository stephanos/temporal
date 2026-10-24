## Quality Audit — Correctness axis: fn-237.3

### Summary

- Files changed: 38 · Critical 0 · Should Fix 0 · Consider 0 · Ship: ⚠️ Fix first (required final gates pending).
- No actionable correctness findings against base `88f3f2098e434cdcd3078f8afe792ae1e64b0fe7`. This audit does not replace primary review.

### Evidence

- **Walked:** deadline arming/reset/spanning/priority, pending cancellation races, failure payload scopes, exact fact replay, concrete-action fairness/lassos, total-Tick induction, old closure controls and retained 1/6/11 graph projection.
- **Executed:** read-only hash comparisons matched all 863 protected entries against both manifests. **Cited:** receipts record native 33 passed/1 inherited ignored, required proof 7 passed, corrected policy controls 5 passed, and passing Go consumers. Earlier harness failures remain disclosed.

### Test Gaps

- [ ] Complete final default/required gates, timing, live slice/gaps and primary review; these remain explicitly pending.

### Test Budget

- Roughly 1,200 added test-body lines versus 576 implementation lines before excluding inline setup. Extensive replay/mutant controls explain the advisory ratio; retained assertions still check exact behavior.

### What’s Good

- Live constructors retain atomic cancellation and absent timeouts; existing refusal remains.
- Compatibility erases only the exact empty-fact Heartbeat stutter and preserves terminal cancellation loops.

Report returned here because auditor instructions prohibit repository writes; `.flow/tmp/fn2373-correctness-audit.md` was not created.
