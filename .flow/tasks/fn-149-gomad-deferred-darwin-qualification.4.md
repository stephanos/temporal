---
satisfies: [R4]
---
# fn-149-gomad-deferred-darwin-qualification.4 Retain Darwin scheduled soak and final qualification matrix

## Description
Implements R4. Consume fn-112.10 scheduled/dispatched soak, fn-109.21, fn-110.5, fn-113.4 and fn-114.14 native finalization. Original source preservation and non-native measurement acceptance stays with donors.

**Size:** M
**Files:** .flow/artifacts/fn-149-gomad-deferred-darwin-qualification/final/**, MILESTONES.md, tools/gomad3/*.md, tools/gomad3integration/README.md, AGENTS.md
**Touches:** [.flow/artifacts/fn-149-gomad-deferred-darwin-qualification/final/**, MILESTONES.md, tools/gomad3/*.md, tools/gomad3integration/README.md, AGENTS.md]

### Approach
Retain one actual completed scheduled/dispatched Darwin soak and its workflow artifacts. Preserve original cohort/cross-batch clean counts, diagnostics/load, overflow/infrastructure separation and measured bound. CI execution requires separate authority and remains deferred now. Reconcile tasks1-3 against one final candidate; refresh invalidated evidence. Publish only evidence-backed local docs and retain independent qualification review with every native ledger row accounted for.

### Investigation targets
**Required:**
- .flow/tasks/fn-112-gomad-determinism-assurance-and-test.10.md
- .flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.21.md
- .flow/artifacts/native-scope-transfer-2026-10-07.md
- .github/workflows/gomad3.yml
- MILESTONES.md

### Quick commands
Use the original scheduled/dispatched soak and final-matrix ledgers. make gomad3-soak alone is not the inherited CI-run requirement. Validate changed records and manually check guide links/command inventories.

### Revival and evidence
Keep deferred until the owner requests Darwin qualification and supplies native darwin/arm64 execution, a supported pinned toolchain/profile and a frozen source candidate. No PR, push or CI action is authorized by this handoff or its revival. Retain the original command/evidence ledgers at d28d67c40ce74dd8886cf11b36fe7d2ddaf23675 and the native transfer manifest. All tests retain -tags test_dep; use integration only for integration tests. Bind source closure, prepared targets, packs and shards to the same identity. Repairs return to their original source owner; refresh affected evidence after changes.

## Acceptance
- [ ] Actual scheduled/dispatched Darwin report and execution artifacts retain all R4 cohort, diagnostics, load, failure classification and bound requirements. No bound is claimed from stand-ins or local-only substitute runs.
- [ ] Final matrix accounts for every transferred native command and task on the same source/toolchain/profile/module closure with passing results, preserved dispositions, refreshed identities, independent review and checked documentation. Any missing required result keeps overall Darwin qualification open.

## Done summary
Blocked:
Deferred by the owner on 2026-10-07. Revival requires an explicit Darwin qualification request, native darwin/arm64 execution, the supported pinned toolchain/profile and a frozen reviewed candidate. Native evidence is unverified. No PR, push or CI action is authorized; scheduled/dispatched soak execution remains deferred pending separate authority. Downstream qualification also needs the reviewed shared D8-D10 candidate and actual consumer checkout. Portable/source requirements stay with the donor tasks; this handoff completes none of them.
## Evidence
- Commits:
- Tests:
- PRs:
