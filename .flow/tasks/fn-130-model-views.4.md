---
satisfies: [R1, R4, R5]
---
# fn-130-model-views.4 Expose checked state mapping and render refinements

## Description
Add the minimal public reader mapping seam and the refinement view. Preserve existing admission and checked semantics.

**Size:** M
**Files:** `tools/umpire/check/claims.go`, `tools/umpire/check/refinement_test.go`, `tools/umpire/render/refinement.go`, `tools/umpire/render/refinement_test.go`
**Touches:** [tools/umpire/check/claims.go, tools/umpire/check/refinement_test.go, tools/umpire/render/refinement*]

### Approach
- Expose the existing checked refinement's state mapping through Realizer, following its current Refinement error path. Reuse the validated map and target membership checks; export no private engine structure and introduce no evaluator.
- Draw all reachable System-state projections and their mapped Product projections with existing carrier/stutter rows. Preserve one projected System node mapping into multiple Product nodes.
- Pin rejected and hole/incomplete mapping behavior, absent/non-refining owners, invalid source/target states and metadata positions. Compare the new public seam against the existing checked rows and independent fixture expectations.

### Investigation targets
**Required:**
- `tools/umpire/check/claims.go:362` - public checked refinement error path.
- `tools/umpire/check/claims.go:840` - existing state-map admission logic.
- `tools/umpire/check/refinement_test.go` - reader/refinement equivalence pins.
- `tools/umpire/interp/machine.go:192` - state lookup and projections.

### Quick commands
```bash
mise exec -- go test -count=1 -tags test_dep ./tools/umpire/check -run 'Test(RealizerGivesTheRefinementCheckReads|ARejectedRefinementStaysWithItsMachine|RenderStateMap)'
mise exec -- go test -count=1 -tags test_dep ./tools/umpire/render -run 'TestRefinement'
```

## Acceptance
- [ ] Public mapping reuses the existing checked logic and returns the same admission/rejection/unknown behavior.
- [ ] Refinement diagrams preserve split mappings, carriers/stutters and truthful non-established states.
- [ ] Byte stability and old-versus-new checked-row/state-map pins pass without changing model/check receipts or identities.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
