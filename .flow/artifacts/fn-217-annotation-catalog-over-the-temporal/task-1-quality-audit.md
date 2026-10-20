### Correctness axis

## Quality Audit — Correctness axis: fn-217.1 at f6ba7ea43e

### Summary

- Files changed: 15 · Critical 0 · Should Fix 1 · Consider 0 · Ship: ⚠️ Fix first

### Should Fix (High priority)

- **model/umpire/annotations/src/value.rs:225** (Conf 75): Walked: Text with prefix `ab`, suffix `bc`, and min/max length 4 admits `abbc`, but seed 0 creates base `abc`, tries padding `a`, then appends the full suffix and returns `ImpossibleConstraints` for length 5 — construct valid overlap extensions independently of random padding and add this regression.

### Out-of-axis observations

> Out-of-axis observation: **model/umpire/annotations/src/catalog.rs:689** — Walked: 17 generated optional fields `a..q` with only `excludes(a,b)` exhaust the 65,536-node budget before clearing `b`; this returns the explicitly documented `SizeLimit`, not `ImpossibleConstraints`.

### Test Gaps

- [ ] Exact-length expansion of overlapping prefix/suffix constraints; existing controls cover suffix-supplied substrings.

### Test Budget

- Ratio: 771 test lines : 1,801 implementation lines, excluding snapshots/manifests.
- Modified existing tests: none.

### What's Good

- Walked: fill commits message and RNG together only after successful validation.
- Walked: violations omit field values; explicit secret redaction clears catalogued secret fields.

Executed: scoped `git diff --check` passed. No repository mutations or test gates run.

### Standards axis

## Quality Audit — Standards axis: fn-217.1

### Summary

- Files changed: 15 · Should Fix 0 · Consider 1 · Blocking: none possible (standards axis).

### Consider

- **model/umpire/annotations/src/value.rs:586** (Conf 100): `repeated.unique` scans every preceding item, requiring n(n−1)/2 comparisons for distinct inputs, even after `max_items` fails — use hash-based uniqueness for supported carriers or enforce an explicit validation-work ceiling. Evidence: walked.

### What's Good

- Typed accessors, private erased operations and explicit admission preserve the reflection-free API boundary.
- No system-specific imports, new third-party dependencies, legacy API paths or 1,000-line crossings.

Evidence: walked scoped source and tests. Gates were not executed during this read-only audit; the conductor reports `make model-check` passed.

Correctness axis: 1 finding; worst tier Should Fix.
Standards axis: 1 finding; worst tier Consider.

Conductor disposition: fix exact-length text generation before review/completion. Keep documented bounded-search and uniqueness-cost observations nonblocking for this task; neither changes the independent Model or annotation semantics.

Final conductor disposition: the exact-length text defect was reproduced in 665e8d5334 and corrected in dc071f05c1, including 6,174 independently enumerated cases. The bounded-search observation later became a formal review finding and was fixed by restricting presence decisions to rule-referenced fields (8afe887b9f red / 02bfd95d48 green). The formal review also found path-alias duplicate admission and restricted unique-string domains; those controls were committed red and fixed. All three formal findings are fixed in the resumed SHIP review at cdf7f3c9e6766aadeb7aab6cadb89b15b40616c6. The O(n²) uniqueness observation remains a nonblocking Consider; no false constant-time guarantee is made.
