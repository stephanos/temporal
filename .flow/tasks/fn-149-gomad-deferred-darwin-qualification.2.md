---
satisfies: [R2]
---
# fn-149-gomad-deferred-darwin-qualification.2 Retain Darwin model, pack and integration qualification

## Description
Implements R2. Native qualification for fn-105.3-.5, all open fn-109.2-.21/.23-.49, fn-112.9/.16, fn-113.1-.4 and fn-114.14.

**Size:** M
**Files:** .flow/artifacts/fn-149-gomad-deferred-darwin-qualification/integration/**, tools/gomad3/internal/compatibilitypack/{requests,reports,packs,generation.json}
**Touches:** [.flow/artifacts/fn-149-gomad-deferred-darwin-qualification/integration/**, tools/gomad3/internal/compatibilitypack/{requests,reports,packs,generation.json}]

### Approach
Run full native host and model/lifecycle/process/race controls, built-CLI, default/integration, core/smoke/representative and affected suites. Retain the approved native discover/review/generate/qualify sequence, pack identities, exact replay and original measurement controls where native execution is required. Authoring writes requests/reports/packs/generation.json in its declared root; bind that root before revival, freeze the resulting candidate and refresh affected runtime evidence after any identity change. Declare any consumer pack root in the actual checkout before writing it. This evidence task owns native authoring outputs, never implementation repairs. Keep ordinary source coverage under donors and preserve failure provenance by command/component.

### Investigation targets
**Required:**
- .flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.21.md
- .flow/tasks/fn-113-gomad-reduce-version-pin-maintenance.4.md
- .flow/tasks/fn-114-gomad-correct-search-path-defects-and.14.md
- tools/gomad3/Makefile:102-153

### Quick commands
Use inherited ledgers, including GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host, make -C tools/gomad3 core-qualification, and original root integration/smoke/affected commands.

### Revival and evidence
Keep deferred until the owner requests Darwin qualification and supplies native darwin/arm64 execution, a supported pinned toolchain/profile and a frozen source candidate. No PR, push or CI action is authorized by this handoff or its revival. Retain the original command/evidence ledgers at d28d67c40ce74dd8886cf11b36fe7d2ddaf23675 and the native transfer manifest. All tests retain -tags test_dep; use integration only for integration tests. Bind source closure, prepared targets, packs and shards to the same identity. Repairs return to their original source owner; refresh affected evidence after changes.
## Acceptance
- [ ] All R2 inherited native command/workload/pack and control results pass on one frozen supported Darwin candidate with unchanged dispositions and required exact replay.
- [ ] Source/profile/module/pack approvals and refusal/drift cases are retained; partial portable coverage does not count as native full-host qualification and portable failures remain source-owned.

## Done summary
Blocked:
Deferred by the owner on 2026-10-07. Revival requires an explicit Darwin qualification request, native darwin/arm64 execution, the supported pinned toolchain/profile and a frozen reviewed candidate. Native evidence is unverified. No PR, push or CI action is authorized; scheduled/dispatched soak execution remains deferred pending separate authority. Downstream qualification also needs the reviewed shared D8-D10 candidate and actual consumer checkout. Portable/source requirements stay with the donor tasks; this handoff completes none of them.
## Evidence
- Commits:
- Tests:
- PRs:
