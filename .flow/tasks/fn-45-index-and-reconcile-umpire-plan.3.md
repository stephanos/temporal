---
satisfies: [R6]
---
# fn-45-index-and-reconcile-umpire-plan.3 Repair plan links and authority descriptions

## Description
Synchronize active/reference documentation with the registry for R6 and make historical exceptions explicit.

**Size:** M
**Files:** `.plans/UMPIRE2.md`, `.plans/UMPIRE3.md`, `.plans/UMPIRE4_DSL.md`, `.plans/UMPIRE4_COMPONENTS.md`, `.plans/index.json`
**Touches:** [.plans/UMPIRE2.md, .plans/UMPIRE3.md, .plans/UMPIRE4_DSL.md, .plans/UMPIRE4_COMPONENTS.md, .plans/index.json]

### Approach
- Repair stale U4 links where the intended successor is unambiguous and preserve historical wording/comments.
- Mark `UMPIRE4_COMPONENTS.md` descriptive, remove stale authority/status implications, and point to the current normative/order/architecture documents.
- Record genuinely historical missing targets in `allowedMissingLinks` rather than inventing replacement content.
- Do not turn any hand-authored plan into generated output.
- Use the focused repository-link test in this wave; the all-state Make check remains intentionally red until task .6 reconciles Flow.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE2.md:15-17` — stale vision/DSL links.
- `.plans/UMPIRE3.md:7-9` — historical missing links.
- `.plans/UMPIRE4_DSL.md:12-13` — active stale links.
- `.plans/UMPIRE4_COMPONENTS.md:3-7,51-68,798` — authority/status drift and missing targets.
- `.plans/UMPIRE4_SPEC_COMPS.md:7-16` — current architecture authority statement.

### Quick commands
`go test -count=1 -tags test_dep ./tools/planindex/... -run '^TestRepositoryPlanLinks$'`
## Acceptance
- [ ] Active/reference local links and anchors resolve.
- [ ] Any intentionally missing historical target is explicitly allowlisted by exact source/target/anchor and reason.
- [ ] Descriptive components prose no longer presents stale readiness as authority.
- [ ] No historical design meaning or existing comment is silently rewritten.
- [ ] Focused repository-link tests pass; any remaining full-check failure is only the Flow drift owned by task .6.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
