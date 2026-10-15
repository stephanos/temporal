---
satisfies: [R6]
---
# fn-180-gomad-replay-time-instrumentation-for.5 Temporal dump and probe demonstrated on a retained failure artifact, docs

## Description
Register at least one Temporal dump (shard and mutable-state summaries) and one probe behind the gomad build seam (default production build unchanged), demonstrate stop-and-inspect, dump and probe output on a retained Temporal failure artifact, and document usage in the WASM README. Include explicit cluster instance/incarnation selectors, local choice ordinals, resolved time/failure boundaries, and bounded scan versus monotonicity-dependent bisect examples. State unsupported old-artifact metadata and per-incarnation search limits.
## Acceptance
- [ ] Temporal dump + probe exist; default server build unchanged
- [ ] Demonstration on a retained Temporal failure artifact recorded in a .flow artifact
- [ ] README documents target selectors, actual stop resolution, evidence refusals and bounded scan/bisect semantics without claiming unconditional binary-search correctness
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
