---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.17 Regenerate final-input approvals, packs, goldens and identity outputs

## Description
Implements R2 for this owner; see the parent Architecture and Delivery sections.

**Size:** M
**Files:** Bounded generator/output owner. Core orchestration/tests are existing authoring/generated checks; wider mechanical outputs include 12 request/report/pack triples, generation manifest, six capability-review goldens and affected protocol/manifest values. No semantic policy edits.
**Touches:** [tools/gomad3/internal/compatibilitypack/requests/**, tools/gomad3/internal/compatibilitypack/reports/**, tools/gomad3/internal/compatibilitypack/packs/**, tools/gomad3/internal/compatibilitypack/generation.json, tools/gomad3/internal/compatibilitypack/packs_generated_test.go, tools/gomad3/target/testdata/capability-review/**, tools/gomad3/target/internal/livecap/protocol_generated.go, tools/gomad3/toolchain/runtime/overlay/src/cmd/internal/gomadcap/protocol_generated.go, tools/gomad3/qualification/*.json, tools/gomad3/qualification/soak/*.json]

### Approach

- Freeze all final sources after generic-package deletion and retain preimages of 12 requests/reports/packs. Stage a disposable complete batch with the reviewed working-directory/source resolution intact. Obtain every new review digest using RenderReview/PublishReview, compare each reviewed semantic projection with its preimage, then prepare all 12 requests with matching ApprovalSHA256 fields using PublishRequest before invoking Generate or Regenerate. A one-request-at-a-time generation loop cannot pass while other requests retain stale approvals.
- Generate and Check the entire prepared staged batch, including requests/reports/packs/generated tests/generation manifest, before publishing validated outputs through the existing publication workflow. Never clear live approval fields, bypass stale-approval checks or approve changed facts. A mismatched request or failed staged check leaves the live batch untouched; retain the existing ordered publication and error behavior if actual output publication fails.
- Compare reviewed semantic projections and source/module pins before and after conversion. Keep 12 IDs, owners/dates, workload/platforms, allowed/denied facts and exact scope. Do not replace retained platform discovery with Linux/arm64 live facts.
- Regenerate the six capability goldens and any inventoried committed manifest outputs from the actual final source closure; retain raw source/archive/patch/overlay pins unless their actual listed input changed.
- Check the task4 protocol output against all final twelve inputs and both source sets; recompute only actually stale derived identities. Independent choice/binary codecs remain unchanged.
- Behavior pin: old/new reviewed semantic projection diff, final source/output SHA bindings and regeneration followed by clean check. See the profile/pin and derived-summary memory entries from the research reports.
- Scope is a bounded mechanical regeneration gate; any newly admitted fact or changed review projection returns to its owning task for explicit review instead of silently expanding this task.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/internal/compatibilitypack/authoring/generate.go:249`
- `tools/gomad3/internal/compatibilitypack/authoring/request.go:194`
- `tools/gomad3/internal/compatibilitypack/packs_generated_test.go`
- `tools/gomad3/internal/compatibilitypack/generation.json`
- `tools/gomad3/target/capability_golden_test.go:24`
- `tools/gomad3/internal/gomadtool/generation/protocol/protocol.go:564`
- `tools/gomad3/Makefile:166`

### Verification

Focused command: make -C tools/gomad3 validate

Capture the frozen behavior pin named above, run focused negative controls and follow the parent Delivery and verification section. Declare scope growth to the conductor before implementation. Shared gates run serially on the integrated frozen candidate.
## Acceptance
- [ ] All 12 retained triples and affected goldens/manifests/producer identities bind final sources; generation/check commands report no stale outputs.
- [ ] Reviewed semantic projections, identity inputs, exact pins/admission scope and retained platform evidence match frozen preimages; approval encoding changes are explicit.
- [ ] A 12-approved-request batch succeeds only after every staged approval matches its new review digest; one stale/mismatched request prevents live publication, and original live preimages remain unchanged on preparation/check failure.
- [ ] Both source sets validate; no native discovery/qualification/CI is claimed, and no malformed/stale approval can be accepted through regeneration.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
