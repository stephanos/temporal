---
satisfies: [R2, R3, R5, R6]
---
# fn-61-simplify-the-umpire-go-execution-surface.3 Internalize the generated binding handoff

## Description
Move generated identity and runtime-slot verification behind the facade and eliminate `runner` as a public execution package (R2-R3). Keep its pre-I/O correctness checks as one stage of the internal execution pipeline.

**Size:** M
**Files:** `tools/umpire/runner/**`, `tools/umpire/internal/execution/**`, `tools/umpire/executor/portable_projection.go`, `tools/umpire/executor/portable_executor.go`
**Touches:** [tools/umpire/runner/**, tools/umpire/internal/execution/**, tools/umpire/executor/portable_projection.go, tools/umpire/executor/portable_executor.go]

### Approach
- Relocate input identity, checksum/fingerprint, capability, and runtime-slot verification to a private execution admission stage owned by the resident executor pipeline.
- Replace the four-method public `runner.Adapter` with the smallest internal domain binding needed to enter checked execution; keep fault-injection seams package-private.
- Remove the public `runner.Run`, `InputBinding`, `RuntimeBindingResolver`, and pass-through `runChecked` surface after every caller is migrated.
- Preserve error kind/phase/code, exact admission ordering, execution-occurred classification, and all existing comments while moving code.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/runner/runner.go:19-45,102-202` — public concepts and handoff to collapse
- `tools/umpire/runner/runner.go:205-330` — exact binding/capability/slot checks
- `tools/umpire/executor/portable_projection.go` — plan-to-binding projection
- `tools/umpire/executor/portable_executor.go:62-108,142-175` — binding consumer and error mapping
- `tools/umpire/runner/runner_test.go:40-291` — pre-I/O and failure-classification matrix

**Optional** (reference as needed):
- `.flow/memory/bug/integration/portable-execution-boundaries-must-2026-09-03.md` — required slot and dispatch boundaries

### Key context
This is a behavior-neutral ownership move after fn-59 centralizes copies. Do not normalize, validate, or clone new values while relocating the checks.

### Acceptance
- [ ] The public runner package and its exported adapter/binding/run surface are removed.
- [ ] One internal stage retains exact artifact, generated identity, authority capability, runtime-slot type, and pre-dispatch validation.
- [ ] Every former runner negative row retains its phase/code and whether execution occurred.
- [ ] No compatibility wrapper or alias keeps the old runner API alive.
- [ ] Focused executor/internal execution tests pass with `-tags test_dep`.

## Acceptance
- [ ] Generated binding verification is private and preserves every admission classification.
- [ ] The shallow public runner package is deleted without aliases.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
