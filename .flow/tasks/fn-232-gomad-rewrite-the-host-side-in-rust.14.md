# fn-232-gomad-rewrite-the-host-side-in-rust.14 Close the remaining test-wasm reach-map gaps after closure

## Description
Owner decision 2026-10-24 (MILESTONES "Opportunistic merge"): fn-232.13 closes on one green full cycle on current `gomad`, without a formal freeze. Remaining test-wasm reach-map gaps move here and are worked after closure.

Scope:
- the `gap:` and `partial:` entries in tools/gomad/testdata/parity/test-wasm-reach-map.json;
- the weak and partial findings in /home/agent/wt/evidence/fn232-reach-review/review-2026-10-23.md and review-2-remaining.md;
- worker E's branch fn232-reach-fixes-2.

Port each behavioral gap with its original assertions and bounds. Reclassify an entry only where the missing part tested Go internals, with an explicit reason. Real behavior bugs found here (e.g. an unknown determinism fault name silently ignored) are fixed with a test.

## Acceptance
- [ ] No reach-map entry has a `gap:` or `partial:` basis.
- [ ] The reach reader refuses `gap:` and `partial:` entries again. Until this task is done they are reported, not refused.
- [ ] Every evidence ID in the map runs in some cycle stage or gate, including feature-gated ones.
- [ ] Merge gate green, with the touched stages exercised in a cycle run on `gomad`.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
