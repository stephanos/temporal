---
satisfies: [R2, R6]
---
# fn-45-index-and-reconcile-umpire-plan.3 Repair plan links and authority descriptions

## Description
Synchronize active/reference documentation with the registry for R6 and make historical exceptions explicit.

**Size:** M
**Files:** `.plans/GOMAD3_CMPR.md`, `.plans/GOMAD3_NEXT.md`, `.plans/GOMAD3_OS.md`, `.plans/UMPIRE2.md`, `.plans/UMPIRE3.md`, `.plans/UMPIRE4_DSL.md`, `.plans/UMPIRE4_COMPONENTS.md`, `.plans/UMPIRE4_ORDER.md`, `.plans/index.json`, `tools/planindex/check_test.go`
**Touches:** [.plans/GOMAD3_CMPR.md, .plans/GOMAD3_NEXT.md, .plans/GOMAD3_OS.md, .plans/UMPIRE2.md, .plans/UMPIRE3.md, .plans/UMPIRE4_DSL.md, .plans/UMPIRE4_COMPONENTS.md, .plans/UMPIRE4_ORDER.md, .plans/index.json, tools/planindex/check_test.go]

### Approach
- Repair stale U4 links where the intended successor is unambiguous and preserve historical wording/comments.
- Repair the three moved GOMAD choice schema, toolchain patch, and trace links to their current repository paths.
- Repair the stale SIM-0 through SIM-5 anchors in active/reference Gomad plans to the current `-complete` heading anchors.
- Mark `UMPIRE4_COMPONENTS.md` descriptive, remove stale authority/status implications, and point to the current normative/order/architecture documents.
- Remove completed fn-42 and fn-50 delivery entries from `UMPIRE4_ORDER.md` while preserving its remaining-work-only intent, retained simplicity dependencies, and prototype decision gate.
- Record genuinely historical missing targets in `allowedMissingLinks` rather than inventing replacement content.
- Add `TestRepositoryPlanLinks` against the production root under an observed failing RED before repairing documents, so the focused Quick command cannot pass vacuously.
- Do not turn any hand-authored plan into generated output.
- Use the focused repository-link test in this wave; the all-state Make check remains intentionally red for only the dependency and readiness drift owned by tasks .5 and .6.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE2.md:15-17` — stale vision/DSL links.
- `.plans/UMPIRE3.md:7-9` — historical missing links.
- `.plans/UMPIRE4_DSL.md:12-13` — active stale links.
- `.plans/GOMAD3_CMPR.md:24-29` — moved choice schema, toolchain patch, and trace targets.
- `.plans/GOMAD3_NEXT.md:84-180` and `.plans/GOMAD3_OS.md:66` — stale anchors for completed SIM-0 through SIM-5 headings.
- `.plans/UMPIRE4_COMPONENTS.md:3-7,51-68,818` — authority/status drift and missing targets.
- `.plans/UMPIRE4_SPEC_COMPS.md:7-16` — current architecture authority statement.

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
Added a production-root repository-link regression, repaired the registered Umpire and Gomad links, made the historical missing synthesis explicit, and reduced `UMPIRE4_ORDER.md` to remaining delivery work while retaining completed prerequisites. R2/R6 are covered by the focused production-link test and the full registry check's zero document findings.

Baseline: the original task Quick exited 0 with `[no tests to run]`; the supported replan added `TestRepositoryPlanLinks`, whose test-only commit then observed the required production-link RED before document repair. Provenance from base `1590bf134d676c1f3cdce572e56b732a07d26f17`: conductor-owned contract reconciliation `cd72d4740`, RED test `771390283`, conductor-owned Gomad ownership reconciliation `be4941cc7`, GREEN implementation `60cb2be49`, and Codex SHIP review metadata `5478baf2b`.

Verification: the exact focused Quick, all `tools/planindex` tests, package vet, JSON parsing, diff checks, and `flowctl validate --all --json` pass. `make umpire-check-plan-index` retains only task .5 dependency and task .6 readiness drift, with zero document/link or unexpected findings. Task-scoped no-fix lint reports zero golangci issues before the unchanged inherited `tools/umpire/runtime/errors.go:60` errortype finding keeps the wrapper red.

stage: impl-review - ran [SHIP; session 01a05c39-673c-77a1-8e02-71fa4e79740d; zero findings]

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: cd72d474039cf9b553080ee20137ac24ec9d0e58, 7713902833ec83d89d1d9b1b8f6f75f92e706f58, be4941cc7bb2dfbe0cbc85daf3334969f3a82a70, 60cb2be4966088d7e56e1aa5188102eab3182d2c, 5478baf2ba9b8b53e07be2d77ac60b93ddc784e7
- Tests: baseline: invalid/vacuous — go test -count=1 -tags test_dep ./tools/planindex/... -run '^TestRepositoryPlanLinks$' exited 0 with [no tests to run] before the supported task replan, TDD RED: go test -count=1 -tags test_dep ./tools/planindex/... -run '^TestRepositoryPlanLinks$' (exit 1 after test-only commit 771390283; production link findings observed before document repair), go test -count=1 -tags test_dep ./tools/planindex/... -run '^TestRepositoryPlanLinks$', go test -count=1 -tags test_dep ./tools/planindex/..., go vet -tags test_dep ./tools/planindex/..., jq empty .plans/index.json, flowctl validate --all --json (valid: true), make umpire-check-plan-index (expected exit 2: only task .5 dependency and task .6 readiness drift; zero document/link and unexpected findings), git diff --check, make lint-code GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=1590bf134 (golangci: 0 issues; inherited unchanged tools/umpire/runtime/errors.go:60 errortype finding keeps wrapper red)
- PRs: