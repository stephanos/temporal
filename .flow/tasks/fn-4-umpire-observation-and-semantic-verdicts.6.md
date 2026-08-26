---
satisfies: [R8]
---
# fn-4-umpire-observation-and-semantic-verdicts.6 Document the offline Observation API and live-runtime handoff

## Description
Document the public offline Observation lifecycle and the exact future adapter boundary for R8 after both reusable and Temporal fixture surfaces exist.

**Size:** M
**Files:** `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`, `model/README.md`, `common/testing/umpire/ABOUT.md`
**Touches:** [model/Umpire/ARCHITECTURE.md, model/ARCHITECTURE.md, model/README.md, common/testing/umpire/ABOUT.md]

### Approach

- Document public imports, lifecycle, statuses, strict aggregation, and synthetic example.
- Reconcile offline verdict vocabulary with existing runtime monitor terminology.
- Define future adapter responsibilities without claiming live behavior.
- Leave generated regression projections unchanged.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/ARCHITECTURE.md:7-44` — public lifecycle/import table.
- `model/ARCHITECTURE.md:147-162` — ownership boundary.
- `model/README.md:68-123` — learning path and commands.
- `model/README.md:149-162` — runtime/evidence handoff.
- `common/testing/umpire/ABOUT.md:34-47` and `common/testing/umpire/ABOUT.md:77-124` — existing vocabulary.

## Acceptance
- [ ] Documentation covers every R8 handoff field and responsibility.
- [ ] Offline unknown/conflict semantics remain distinct from runtime claim terminology.
- [ ] No text claims live execution, collection, persistence, conformance, promotion, or multiple profiles.
- [ ] `make umpire-check-regression` passes with documentation and import-boundary checks.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
