---
satisfies: [R1, R2, R3, R4, R5]
---
# fn-130-model-views.2 Build the pinned phase renderer and its live command foundation

## Description
Turn the proven vertical slice into the live render owner and phase-view foundation. Reader/library code and thin command/manifest ownership are one integration unit; later tasks extend this existing caller.

**Size:** M
**Files:** `tools/umpire/render/phase.go`, `tools/umpire/render/phase_test.go`, `tools/umpire/cmd/umpire-render/main.go`, `tools/umpire/cmd/umpire-render/main_test.go`; narrow edits to `tools/umpire/lint/holes.go`, `tools/umpire/lint/holes_test.go`, `go.mod`, `go.sum`, `tools/umpire/ir/ownership_test.go`, `Makefile`, `.plans/UMPIRE_MODULES.md`
**Touches:** [tools/umpire/render/phase*, tools/umpire/cmd/umpire-render/**, tools/umpire/lint/holes.go, tools/umpire/lint/holes_test.go, go.mod, go.sum, tools/umpire/ir/ownership_test.go, Makefile, .plans/UMPIRE_MODULES.md]

### Approach
- Adopt task .1's pinned D2 adapter and input-sharing contract. Keep public reader plus lint as the only Umpire imports; add positive/negative D2-owner checks and an actual Make caller so the package is live immediately.
- Add immutable typed branch explanations to Cell from its existing Why call before any enabled/disabled/hole return. Retain ordered guards/selected match conditions, branch, source position, state/input and nested markers, and called predicates using existing spelling helpers; preserve Cell.Guard, predicates and Rule grouping behavior. Handle decision-free rows, holes and unmatched cases explicitly. No second evaluation, mutable expression alias or invented per-result branch mapping.
- Derive phase edges from typed cells/results and those retained explanations, joining disabled findings and acceptance reasons with Accepted.Judge. Preserve aggregated partial cells and hidden-field self-loops; never parse Rule.Text. Label explanations as cell-wide when multiple results prevent unique attribution.
- Implement the R3 projection order over typed states: enum state/top-level phase, unique nested enum phase path (lostStartAnswer.record.phase), explicit queue custody/outstanding, close-policy caller/handler, then attributed deterministic full-state fallback. Detect invalid/recursive paths and show ambiguous candidates; never silently drop a valid machine or collapse it to an empty label. Disclose hidden fields and test lossAvailable/polled/delivered-only changes as annotated self-loops.
- Use the independently frozen full projection inventory from .1, fixed stable identities/order, truthful facts/Because/guards/positions and matching D2/SVG text. The initial command exercises this vertical slice; whole-tree management is completed in .7.
- Use small independently authored fixtures for branching results, holes, empty domains, malicious syntax, nil/empty collections and differing acceptance reasons. Reuse baseline behavior pins; no production IR or Case edits.

### Investigation targets
**Required:**
- `tools/umpire/lint/holes.go:33` - typed cells/results.
- `tools/umpire/lint/accept.go:113` - existing exact acceptance joins.
- `tools/umpire/ir/ownership_test.go:69` - reader and external dependency policy.
- `tools/umpire/cmd/umpire-lint/main.go:65` - command usage/exit conventions.
- `tools/umpire/lint/holes_test.go:257` - phase-table fixtures.
- `tools/umpire/interp/decisions.go:39` - ordered Why decisions, including nested and decision-free paths.
- `model/ir/activity-standalone-race.json` and `model/ir/activity-standalone-record.json` - nested phase and queue state declarations.

### Quick commands
```bash
mise exec -- go test -count=1 -tags test_dep ./tools/umpire/lint ./tools/umpire/render ./tools/umpire/cmd/umpire-render
mise exec -- go test -tags test_dep ./tools/umpire/ir -run '^(TestLiveModelDependencyGraph|TestModelDependencyGraphRejectsCrossedOwners|TestEveryToolingPackageHasALiveCaller)$'
```

## Acceptance
- [ ] Live phase-view command and pinned D2 owner satisfy dependency/caller checks without private/lowering/Testpilot imports.
- [ ] Full expected projection inventory and synthetic controls cover nested lostStartAnswer, custody/outstanding queues, close-policy labels, hidden-only changes and attributed fallback without excluding valid machines.
- [ ] Enabled guards whose values select different transitions, then/else and match conditions, multiple-result attribution, decision-free rows, holes and unmatched cases retain truthful typed explanations from the existing single Why call; existing lint table/rule behavior remains unchanged.
- [ ] Synthetic and actual projection examples preserve enabled/disabled result details and exact acceptance-reason joins.
- [ ] Repeated artifacts and escaping/font/empty-input negative cases pass; modeled behavior, IR/Case bytes and identities remain unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
