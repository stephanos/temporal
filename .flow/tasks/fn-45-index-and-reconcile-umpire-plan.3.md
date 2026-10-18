---
satisfies: [R2, R6]
---
# fn-45-index-and-reconcile-umpire-plan.3 Repair plan links and authority descriptions

## Description
Synchronize active/reference documentation with the registry for R6 and make historical exceptions explicit.

**Size:** M
**Files:** `.plans/UMPIRE2.md`, `.plans/lean/UMPIRE3.md`, `.plans/lean/UMPIRE4_DSL.md`,
`.plans/UMPIRE4_COMPONENTS.md`, `.plans/UMPIRE4_ORDER.md`, `.plans/index.json`,
`tools/planindex/check_test.go`
**Touches:** [.plans/UMPIRE2.md, .plans/lean/UMPIRE3.md, .plans/lean/UMPIRE4_DSL.md,
.plans/UMPIRE4_COMPONENTS.md, .plans/UMPIRE4_ORDER.md, .plans/index.json,
tools/planindex/check_test.go]

### Approach
- Repair stale U4 links where the intended successor is unambiguous and preserve historical wording/comments.
- Mark `UMPIRE4_COMPONENTS.md` descriptive, remove stale authority/status implications, and point to the current normative/order/architecture documents.
- Remove completed fn-42 and fn-50 delivery entries from `UMPIRE4_ORDER.md` while preserving its remaining-work-only intent, retained simplicity dependencies, and prototype decision gate.
- Record genuinely historical missing targets in `allowedMissingLinks` rather than inventing replacement content.
- Add `TestRepositoryPlanLinks` against the production root under an observed failing RED before repairing documents, so the focused Quick command cannot pass vacuously.
- Do not turn any hand-authored plan into generated output.
- Use the focused repository-link test in this wave; the all-state Make check remains intentionally red for only the dependency and readiness drift owned by tasks .5 and .6.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE2.md:15-17` — stale vision/DSL links.
- `.plans/lean/UMPIRE3.md:7-9` — historical missing links.
- `.plans/lean/UMPIRE4_DSL.md:12-13` — active stale links.
- `.plans/UMPIRE4_COMPONENTS.md:3-7,51-68,818` — authority/status drift and missing targets.
- `.plans/lean/UMPIRE4_SPEC_COMPS.md:7-16` — current architecture authority statement.

### Quick commands
`go test -count=1 -tags test_dep ./tools/planindex/... -run '^TestRepositoryPlanLinks$'`
## Acceptance
- [ ] Active/reference local links and anchors resolve.
- [ ] Any intentionally missing historical target is explicitly allowlisted by exact source/target/anchor and reason.
- [ ] Descriptive components prose no longer presents stale readiness as authority.
- [ ] The delivery-order queue contains only remaining work and does not reintroduce completed fn-42 or fn-50 entries.
- [ ] No historical design meaning or existing comment is silently rewritten.
- [ ] `TestRepositoryPlanLinks` executes against the production root, observes the pre-repair link findings as RED, and cannot pass via `[no tests to run]`.
- [ ] Focused repository-link tests pass; the full check has zero document/link findings and zero unexpected Flow findings, with any remaining failure limited to the dependency and readiness drift owned by tasks .5 and .6.
## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: