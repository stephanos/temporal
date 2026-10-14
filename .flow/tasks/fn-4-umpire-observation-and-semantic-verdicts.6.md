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
Documented the complete public offline Observation lifecycle, strict semantic verdict aggregation, and the sole synthetic Nexus BasicLifecycle profile across the four public architecture/learning documents. Defined `EvidenceBundle` as the only future adapter handoff while keeping offline unknown/conflict/unsupported results distinct from Monitor and qualified-claim terminology and making no live execution, collection, persistence, conformance, promotion, or additional-profile claim.

baseline: green
verification: all four Quick commands passed; `make lint-model` completed the 114-target graph without lint errors
gate receipt: not warrantable because concurrent `.plans/UMPIRE4_ORDER.md` changes remain dirty outside this task
stage: impl-review - ran [2026-08-27T08:50:27Z..2026-08-27T08:50:27Z] (triage-skip: docs-only)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3fa5bc7a5de7d890e152b0cded868c5799a8f080
- Tests: cd model && mise exec -- lake build Umpire.Observation.Tests.Compilation, cd model && mise exec -- lake build Umpire.Observation.Tests, cd model && mise exec -- lake build Temporal.Feature.Nexus.ObservationTests, make umpire-check-regression, make lint-model
- PRs:
