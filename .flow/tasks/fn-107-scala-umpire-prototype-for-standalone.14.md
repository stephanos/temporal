---
satisfies: [R3, R4, R5, R9]
---
# fn-107-scala-umpire-prototype-for-standalone.14 Extend generic finite checker hooks and progress semantics

## Description
**Touches:** [model/go/umpire/**]

Extract the generic checker work from task 3. This task uses existing finite Go tables and callbacks; the IR interpreter is a separate lane. Feature behavior remains authored in Scala.

**Size:** M
**Files:** generic search/claims/refinement/composition modules and focused generic tests.

### Approach
- Extend existing table/search owners with a small reusable boundary for passive observer state. Include that state in visited identity, preserving all machine transitions and existing deterministic witness order.
- Extend generic refinement to check initial correspondence and reject a visible event/result treated as stutter. Preserve invisible stutter, existing Definition IDs, and legacy behavior in the absence of the new declaration.
- Supply generic finite deadline/deadlock/fair-cycle checking hooks with explicit assumptions and resource limits. Distinguish unresolved prefixes and exhausted work from verified progress.
- Keep composition and scoped provider substitution generic; account for all admitted initial states. Never encode activity eligibility, Nexus ownership, or other feature policy in Go.
- Replay witnesses against the same finite transition relation. Preserve existing comments and baseline model behavior.

### Investigation targets
**Required:** model/go/umpire/search.go; model/go/umpire/claims.go; model/go/umpire/refine.go; model/go/umpire/compose.go; model/go/umpire/table.go; model/scalav2/SEMANTICS.md.
**Optional:** reviewed specimens and task 2's source declarations, read-only.

### Quick commands
`mise exec -- go test -tags test_dep ./model/go/...`; scoped generic Go lint. Record any inherited repository-wide lint failure separately.
## Acceptance
- [ ] Distinct passive-observer histories remain distinct explored states; observer state never suppresses a machine behavior.
- [ ] Initial-state and visible-output refinement mutation controls fail with replayable diagnostics; invisible stutters and existing baseline checks pass.
- [ ] Generic deadlock/deadline/fair-cycle controls distinguish proved failures, unresolved prefixes, and explicit work-limit exhaustion under named assumptions.
- [ ] Composition/substitution accounts for admitted initial states and preserves existing table/identity behavior; all generic and baseline Go model tests pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
