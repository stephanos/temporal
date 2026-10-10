---
satisfies: [R1, R4]
---
# fn-130-model-views.5 Render inferred derived-design comparisons

## Description
Add the inferred-base diff family with honest provenance and concrete result comparison.

**Size:** M
**Files:** `tools/umpire/render/derived.go`, `tools/umpire/render/derived_test.go`
**Touches:** [tools/umpire/render/derived*]

### Approach
- Restrict inference to compatible entity/state/outcome/fact/action catalogs and score actual shared action/function bindings. Retain tied candidates in fixed order and label the comparison inferred; identical siblings establish equivalence rather than directed inheritance.
- Compare full concrete cells/results before projection. Include added/removed classes, starts/reachable states, facts/outcomes/Because, enabled/disabled differences and binding provenance even when shared functions remain identical.
- Test close-policy ties, record/member identical siblings, absent overlap, mutual best matches, changed starts and changes masked by projection. Never create a recursive inferred hierarchy.

### Investigation targets
**Required:**
- `proto/internal/temporal/server/api/umpire/v1/machine.proto:50` - machine catalog and StepBinding.
- `tools/umpire/lint/holes.go:33` - concrete cells and results.
- `model/temporal/features/nexus/workflow/system/ClosePolicy.scala:1000` - real derived variants.
- `tools/umpire/interp/types.go` - public result and table types.

### Quick commands
```bash
mise exec -- go test -count=1 -tags test_dep ./tools/umpire/render -run 'TestDerived'
```

## Acceptance
- [ ] Inference and ties are deterministic, visibly inferred and free of fabricated ancestry/cycles.
- [ ] Independent expected diffs detect binding/result/reachability/disabled-cell changes before projection.
- [ ] Identical siblings, zero overlap and ambiguous bases have truthful output and stable bytes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
