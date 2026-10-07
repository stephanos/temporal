---
satisfies: [R8, R9]
---
# fn-148-consolidate-testpilot-evidence-and.7 Complete the successor-format migration and close documentation

## Description
Regenerate the supported artifact surface, compare measurements and behavior, regenerate and replay checked-in companions, and update the rules of record for R8 and R9. Close only after every delta is categorized against this spec or its CEL and Duration predecessors.

**Size:** M
**Files:** generated Cases and fixtures, recorded Runs, `model/SEMANTICS.md`, `model/README.md`, `.plans/UMPIRE_MODULES.md`, `.plans/UMPIRE4_SPEC.md`, `MILESTONES.md`
**Touches:** [model/cases/**, tests/testcore/testpilot/testdata/generated/**, tools/canary/**/testdata/**, common/testing/testpilot/**/testdata/**, model/SEMANTICS.md, model/README.md, .plans/UMPIRE_MODULES.md, .plans/UMPIRE4_SPEC.md, MILESTONES.md]

### Approach
- Regenerate current-format artifacts and recorded companions together; retain no historical format branch.
- Compare online and offline evaluation, causal buffering, same-atom/different-fields states, ordinals, defaults, ceilings, verdicts and identities.
- Record compact-size and admission-cost results without claiming savings the measurements do not show.

### Investigation targets
**Required** (read before coding):
- `MILESTONES.md:19-49` - shared full-gate and evidence discipline
- `common/testing/testpilot/protocol_test.go:80-175` - complete protocol surface
- `common/testing/testpilot/recordedrun/recordedrun.go:40-90` - current pairing
- `model/README.md:117-165` - artifact pipeline
- `.plans/UMPIRE_MODULES.md:400-450` - runtime and artifact ownership
- `.plans/UMPIRE4_SPEC.md:330-365` - standalone Case and version rules


### Quick commands

```bash
make umpire-gen-model
make umpire-gen-fixtures
make canary-gen-case
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks
go test -tags test_dep -p 2 -timeout 30m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/...
make umpire-check-cases
make umpire-check-fixtures
make canary-check-case
make lint-model
make lint-code-fast
make umpire-check-live-tests
```

## Acceptance
- [ ] R8's behavior, measurement, replay and identity evidence is complete.
- [ ] R9's semantics, module, protocol and milestone updates are current.
- [ ] No uncategorized Case, Run, verdict, conformance or ceiling delta remains.
- [ ] Required full gates pass with reusable evidence recorded.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
