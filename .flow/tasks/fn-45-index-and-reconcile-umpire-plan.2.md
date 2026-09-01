---
satisfies: [R1, R2, R6]
---
# fn-45-index-and-reconcile-umpire-plan.2 Author the complete plan authority registry

## Description
Populate the production v1 registry and focused command surface for R1/R2.

**Size:** M
**Files:** `.plans/index.json`, `tools/planindex/check.go`, `tools/planindex/index_test.go`, `tools/planindex/check_test.go`, `Makefile`
**Touches:** [.plans/index.json, tools/planindex/check.go, tools/planindex/*_test.go, Makefile]

### Approach
- Classify every current `.plans/*.md` exactly once using the approved authority precedence and lifecycle vocabulary.
- Classify every current `.flow/specs/*.json` exactly once as Umpire roadmap, Umpire support, or other; record the intended post-reconciliation disposition/phase/status/readiness/completion-review/dependencies. Fields match current committed Flow state except the fn-17/fn-33 dependency drift owned by task .5 and the explicit readiness drift owned by task .6.
- Extend the checker under focused RED/GREEN tests so any direct or transitive dependency from retained scope into deferred-only scope fails deterministically.
- Add production-root tests and `umpire-check-plan-index`; do not add tracker sync or CI workflow files.
- Keep intentional historical missing links explicit and minimal in `allowedMissingLinks`.
- The desired-state registry is expected to report Flow/document drift until tasks .3-.6 finish; do not weaken it to make this wave green.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE4_SPEC.md` — normative root.
- `.plans/UMPIRE4_ORDER.md` — delivery-order root and Flow mapping.
- `.plans/UMPIRE4_SPEC_COMPS.md:7-16` — architecture/descriptive boundary.
- `.plans/UMPIRE4_SPEC_MODEL_ARCH.md:1-65` — scoped model contract.
- `.flow/specs/fn-21-nexus-duplicate-observation-control.json` — completed-prerequisite roadmap work whose stale ready flag task .6 clears.
- `.flow/specs/fn-42-centralize-configuration-authoring-with.json` — completed-prerequisite support that remains open/unready with SHIP review.
- `.flow/specs/fn-48-canonicalize-known-gaps-as-a-checked-set.json` — retained support depending on retained fn-43 and fn-47.
- `Makefile:1283-1317` — PHONY and focused lint target style.

### Quick commands
`go test -count=1 -tags test_dep ./tools/planindex/...`
## Acceptance
- [ ] Every current plan Markdown file and Flow spec JSON has one canonical ordered registry row and no row remains unclassified.
- [ ] Flow rows encode reconciled scope, phase, disposition, status, readiness, completion-review status, and complete dependency sets; fn-21 is completed-prerequisite open-SHIP roadmap work, fn-42/fn-44/fn-50 are completed-prerequisite open-SHIP support, and fn-43 plus fn-45..49/fn-51 are retained support.
- [ ] Production-root tests detect an added, removed, renamed, or multiply registered plan/spec.
- [ ] Focused fixtures prove both direct and transitive retained-to-deferred dependency paths fail while valid retained and completed-prerequisite paths pass.
- [ ] Parser/schema/unit tests pass; the not-yet-reconciled production command fails only with the expected deterministic document/Flow drift, including fn-17/fn-33 dependencies until task .5 and fn-21 readiness until task .6.
- [ ] The Make target is focused/read-only and existing comments are preserved.
## Done summary
Added the complete v1 authority registry for all 25 plan documents and 51 Flow specs, production coverage, deterministic direct/transitive retained-to-deferred rejection, and the focused read-only Make target. Final focused tests and Flow validation pass; the production check reports only staged task .3/.5/.6 drift, while global lint retains the pre-existing `tools/umpire/runtime/errors.go:60` errortype finding.

The TDD RED is `361d85663`, approved contract reconciliation is `afbc62df1`, GREEN implementation is `fa52ae24d`, and Codex SHIP review provenance is recorded by `6fe54afb8` (session `01a05c24-091f-7201-8763-60c24bb72dcd`, zero findings). Flow validation remains valid with zero errors and the unchanged 203-warning inherited baseline.

stage: impl-review - ran [SHIP at 2026-09-01T08:47:40Z; session 01a05c24-091f-7201-8763-60c24bb72dcd]

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 361d856632134e506498cd42a233c0de515f8d6f, afbc62df1018fa302365f17209a09516057d651e, fa52ae24d12c90a11ece38ac39a156d1a70a0ce3, 6fe54afb84d0c1a279d154319fa431e2c016521a
- Tests: baseline: green (go test -count=1 -tags test_dep ./tools/planindex/... and flowctl validate --all --json); expected RED (make umpire-check-plan-index target absent pre-edit), TDD RED: go test -count=1 -tags test_dep ./tools/planindex -run '^TestCheckRepositoryRejectsRetainedDependenciesOnDeferredScope$' (exit 1 before checker implementation), go test -count=1 -tags test_dep ./tools/planindex/..., go vet -tags test_dep ./tools/planindex/..., .bin/errortype -style-check=false ./tools/planindex/..., flowctl validate --all --json (valid; 0 errors; unchanged inherited 203 warnings), make umpire-check-plan-index (expected staged drift only: 15 task .3 document findings, 2 task .5 dependency findings, 9 task .6 readiness findings; 0 unexpected findings), git diff --check, make lint-code GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=61eade94a3dd876faaaaac55c682504fb40e20f3 (golangci: 0 introduced issues; inherited tools/umpire/runtime/errors.go:60 errortype finding keeps wrapper red)
- PRs: