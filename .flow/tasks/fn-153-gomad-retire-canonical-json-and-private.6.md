---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.6 Migrate compatibility authoring without changing approvals' meaning

## Description
Implements R2 for this owner; see the parent Architecture and Delivery sections.

**Size:** M
**Files:** internal/compatibilitypack/schema.go and authoring/{request,generate,publication,refresh,working_directories}.go with tests (6 core callers). Retained request/report/pack outputs belong to task17.
**Touches:** [tools/gomad3/internal/compatibilitypack/schema.go, tools/gomad3/internal/compatibilitypack/*_test.go, tools/gomad3/internal/compatibilitypack/authoring/**]

### Approach

- Switch ordinary pack/request/report encoding and parsing to stdlib/strictjson, retaining bounds, exact rules/module/source-set scope, sorted uniqueness and denied capability policy.
- Keep the complete reviewed approval projection and stale-approval refusal. Record how stdlib encoding changes derived approval/request/pack/generation values; final retained output conversion belongs to task17.
- Route already-shared publication through the staged primitive wrappers without altering multi-output transaction ownership.
- Behavior pin: compare frozen request/pack semantic projections and mutate each governance/identity input independently; test invalid original strings, unknown fields, duplicates, trailing input, capacity and stale approval.
- Do not perform new platform discovery or reapprove changed facts. Existing external pack approval workflow and public commands remain unchanged.
- Migrate schema_admission_test.go, schema_timezone_test.go, policy_exhaustive_test.go, mutation_test.go and external_test.go here, preserving their policy, timezone and external-caller controls. Task17 owns mechanical final regeneration of packs_generated_test.go.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/internal/compatibilitypack/schema.go:117`
- `tools/gomad3/internal/compatibilitypack/authoring/request.go:194`
- `tools/gomad3/internal/compatibilitypack/authoring/generate.go:249`
- `tools/gomad3/internal/compatibilitypack/authoring/publication.go:14`
- `tools/gomad3/internal/compatibilitypack/authoring/refresh.go:179`
- `tools/gomad3/internal/compatibilitypack/authoring/working_directories.go:72`
- `tools/gomad3/internal/compatibilitypack/authoring/request_test.go`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./internal/compatibilitypack/authoring

Capture the frozen behavior pin named above, run focused negative controls and follow the parent Delivery and verification section. Declare scope growth to the conductor before implementation. Shared gates run serially on the integrated frozen candidate.
## Acceptance
- [ ] Approvals still bind every reviewed input and reject semantic changes or stale approval; valid current inputs round trip with stdlib encoding.
- [ ] Schema, bounded rules, allowed/denied facts, modules/sums, owner/date/workload/platform and exact admission semantics remain unchanged.
- [ ] Source tests separate spelling-only failure from policy/error behavior; all retained output updates are explicitly assigned to task17 rather than silently omitted.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
