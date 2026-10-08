---
satisfies: [R3, R4, R5, R6]
---
# fn-149-safety-and-liveness-groups-for-object.5 Document grouped authoring and close integrated gates

## Description
Document grouped authoring and close integrated gates. Advances R3, R4, R5, R6 of the parent spec.

**Size:** M
**Files:** `model/README.md`, `model/SEMANTICS.md`, `tools/umpire/README.md`, `model/irgen/testdata/layout/**`, `model/ir/**`, `model/cases/**`, `tests/testcore/testpilot/testdata/generated/**`, `tools/canary/casebinding/testdata/**`
**Touches:** [model/README.md, model/SEMANTICS.md, tools/umpire/README.md, model/irgen/testdata/layout/**, model/ir/**, model/cases/**, tests/testcore/testpilot/testdata/generated/**, tools/canary/casebinding/testdata/**]

### Approach
- Update the author layout/template and examples with safety postconditions, bounded progress and the distinction between verification and witness finding. Document supported composition safety and link composition progress work to fn-150.
- Integrate task 3's reports and task 4's migration. Refresh functional/canary pins only where changed source Cases require it; preserve historical recorded Run companions.
- Run required integrated gates once under the shared heavy-check lock using the existing skip-go-checks split. Reuse valid focused evidence; capture full Go JSON timings as MILESTONES requires.
- Close requirement coverage with evidence; do not add a broad generated-API drift gate, new CI workflow or live composition Case capability.

### Investigation targets
**Required:**
- `model/README.md` - authoring and gate instructions.
- `model/SEMANTICS.md:145` - claims versus bounded progress.
- `model/irgen/testdata/layout` - executable author layout.
- `MILESTONES.md` - verification and regeneration serialization.
- `tools/umpire/README.md` - current reporting documentation.

### Key context
Re-anchor paths and interfaces against completed fn-140/fn-141 and the approved schema/package moves before editing. Keep this work outside the activity batch and serialize shared regeneration with it and the schema chain; no new spec-close dependency is implied.

### Quick commands
```bash
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks
make umpire-check-cases umpire-check-fixtures canary-check-case
make lint-model
make lint-code-fast
go test -tags test_dep -p 2 -timeout 30m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/...
```

## Acceptance
- [ ] Author documentation and executable layout agree on grouping and all R6 examples; composition progress is not advertised as shipped.
- [ ] Integrated model, Case/fixture, applicable lint and Go gates pass against the explained migration diff.
- [ ] Record focused and full gate evidence, changed artifact disposition and any unresolved unsupported capability without weakening expected verdicts.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
