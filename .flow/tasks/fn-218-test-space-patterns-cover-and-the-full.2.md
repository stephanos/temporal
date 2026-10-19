---
satisfies: [R2, R3, R5]
---
# fn-218-test-space-patterns-cover-and-the-full.2 cover and sample in umpire-suite with expansion and coverage goldens

## Description
`cover(&space, Cases(n), path)` and `sample` in `umpire-suite`: `Cases(n)` required, checked against the enumerated count before search, mismatch fails first listing the alternatives; then per alternative the shortest admitted matching path via fn-213's `find`, in alternative order; failure naming the un-witnessed alternative (index, concrete choices, last reached state); count printed first; `Refusal` on a count over budget and `Refusal::Cut` on a cut search; `Covered` with `runs`, `len`, `all`, `coverage()` against the Model's catalog, `expansion()`; `sample(n, seed)` reproducible; `Run::along` refuses alternatives. Expansion and coverage goldens written through the suite runner. Suite fixtures pin a `Cases` mismatch, a failing alternative, an over-budget count, a cut search and a sample by seed. After .1 and fn-213.3. **Touches:** `model/umpire/suite/**`, `model/umpire/testing/**` (golden kinds). **Feedback loop (R30):** paste `make model-time`.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
