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
Migrated choice exploration, Runner outcome, segment, and portable-plan identities to complete deterministic stdlib projections with strict typed plan decoding and explicit original-string validation. Preserved controller/outcome membership, map-order repeatability, ordered-round and resume semantics, stale/corrupt-plan refusal, and no-replace plan publication without entering fn152 storage or fn153.11 simulation scope. Corrected the lineage comment to match schema-less refusal by protocol-identity validation. Focused, race, hostfs/world, and architecture gates pass; the implementation re-review is SHIP. Aggregate validation reaches only fn153.17's known stale compatibility approval.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 2350fc49ad85009b5a3a1166580323bbbb495f0c, 802dce211ac93dda49bc11b354373b54c620afd0, a68425fa3137c650c00ad310bdbd55b9ede0f5c6
- Tests: go -C tools/gomad3 test -tags test_dep -count=1 ./runner/internal/exploration ./runner/internal/campaign: PASS, focused Runner choice/outcome/portable-plan/resume tests: PASS, go -C tools/gomad3 test -tags test_dep -race -count=1 ./runner/internal/exploration ./runner/internal/campaign: PASS, go -C tools/gomad3 test -tags test_dep -count=1 ./internal/hostfs ./world: PASS, go -C tools/gomad3 test -tags test_dep -count=1 ./internal/archcheck: PASS, make -C tools/gomad3 validate: EXPECTED FN153.17-OWNED DRIFT (compatibility-pack request golang-x-sys-v047-darwin-arm64 approval is stale), flowctl codex impl-review re-review: SHIP, prior comment-contract finding fixed and no new findings
- PRs:
## Revival trigger (2026-10-13)

Revived after fn153.1, fn153.3 and fn153.4 completed with SHIP implementation reviews and after fn153.2 stabilized portable-plan publication. This task owns only choice exploration and Runner identity projections; fn152 storage is retained and simulation identities remain with fn153.11. The forced local claim is required only because Flow chain admission still sees the completed fn152 parent on an unpushed remote branch; pushing remains outside the requested scope.
