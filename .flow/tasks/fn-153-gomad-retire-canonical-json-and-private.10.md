---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.10 Migrate choice exploration and Runner identity projections

## Description
Implements R2 for this owner; see the parent Architecture and Delivery sections.

**Size:** M
**Files:** runner/internal/exploration/choice/engine.go and runner/{choice_exploration_campaign,resume,portable_plan}.go with associated tests (4 core callers).
**Touches:** [tools/gomad3/runner/internal/exploration/choice/**, tools/gomad3/runner/choice_exploration_campaign*.go, tools/gomad3/runner/resume.go, tools/gomad3/runner/resume_test.go, tools/gomad3/runner/portable_plan*.go]

### Approach

- Re-read fn152's landed Runner survivors; migrate only pure exploration and coordinator identities it preserves, not removed journals or new log machinery.
- Use stdlib typed projections/strictjson while retaining controller/segment identity membership, regen equality meaning, round ownership, ordered completion and resume behavior.
- Behavior pin: recorded logical choice/controller/segment outcomes, current-build repeatability under map insertion variation, independently varied identity inputs, resumed vs uninterrupted semantic outcomes and corruption/stale plan rejection.
- Consume current record/preparation identities from predecessor tasks; keep portable plan/artifact ownership and no-replace directory publication.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/runner/internal/exploration/choice/engine.go:373`
- `tools/gomad3/runner/internal/exploration/choice/engine_test.go`
- `tools/gomad3/runner/choice_exploration_campaign.go:478`
- `tools/gomad3/runner/resume.go:141`
- `tools/gomad3/runner/portable_plan.go:183`
- `tools/gomad3/runner/seed_completion_characterization_test.go`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./runner/internal/exploration/choice

Capture the frozen behavior pin named above, run focused negative controls and follow the parent Delivery and verification section. Declare scope growth to the conductor before implementation. Shared gates run serially on the integrated frozen candidate.

## Acceptance
- [ ] Choice/controller/segment identities retain complete projection membership and same-build repeatability; identity-input changes remain observable.
- [ ] Resume, ordered seed/round commit semantics and artifact ownership match the frozen behavior pin; fn152-owned storage is not remigrated.
- [ ] Malformed/duplicate/unknown/trailing inputs, invalid typed strings, corrupt identities and stale/regenerated segments retain refusal and error ordering.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
