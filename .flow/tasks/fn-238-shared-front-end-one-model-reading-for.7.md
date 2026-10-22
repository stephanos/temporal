---
satisfies: [R4]
---
# fn-238-shared-front-end-one-model-reading-for.7 Front end / back end boundary write-up and measurements

## Description
Design: `model/design/shared-front-end.md`, Migration row .7.

DESIGN.md describes the front end / back end boundary and retires "independent source admission" for new work; MILESTONES updated by the conductor. Record line counts per crate, `make model-time` output and the .5/.6 ratios (update after .6 if taken).

## Acceptance
- [ ] DESIGN.md boundary section with reproducible commands and numbers.
- [ ] Line counts per crate and timing recorded.

## Done summary
DESIGN.md "Front end and back ends (fn-238)" records the boundary as built (umpire-front reads once, umpire-ir embedded by the macro off wasm32, macros generate native code, lifter emits from the embedded Model, guest and typed replay compiled), the retirement of independent source admission for new work with typed replay and TLC agreement as cross-checks, reproducible per-crate line counts (core 8,190 -> 8,297, +107 against the -850 to -1,600 estimate, with causes), per-construct cost (halves only for declaration-level constructs, estimate not yet measured; body-level unchanged, fn-227.11 example), Option B deferred by the owner, fn-238.8's guest identity change with the fn-239 renewal caveat, and `make model-time` receipts across the series. Stale TLC lines rewritten; DESIGN.md held at its 1,000-line cap. Review: NEEDS_WORK for two figures on line 45, corrected by the conductor in 99e59a4090 (.flow/tmp/fn2387-review.md). Docs only; no gates required.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4e7d2763f7, 99e59a4090
- Tests:
- PRs: