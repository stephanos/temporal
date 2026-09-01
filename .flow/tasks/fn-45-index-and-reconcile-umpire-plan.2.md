---
satisfies: [R1, R2]
---
# fn-45-index-and-reconcile-umpire-plan.2 Author the complete plan authority registry

## Description
Populate the production v1 registry and focused command surface for R1/R2.

**Size:** M
**Files:** `.plans/index.json`, `tools/planindex/index_test.go`, `tools/planindex/check_test.go`, `Makefile`
**Touches:** [.plans/index.json, tools/planindex/*_test.go, Makefile]

### Approach
- Classify every current `.plans/*.md` exactly once using the approved authority precedence and lifecycle vocabulary.
- Classify every current `.flow/specs/*.json` exactly once as Umpire roadmap, Umpire support, or other; record exact disposition/phase/status/readiness/completion-review/dependencies from current committed Flow state, including completed open-SHIP support and retained non-gating follow-ups.
- Add production-root tests and `umpire-check-plan-index`; do not add tracker sync or CI workflow files.
- Keep intentional historical missing links explicit and minimal in `allowedMissingLinks`.
- The desired-state registry is expected to report Flow/document drift until tasks .3-.6 finish; do not weaken it to make this wave green.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE4_SPEC.md` — normative root.
- `.plans/UMPIRE4_ORDER.md` — delivery-order root and Flow mapping.
- `.plans/UMPIRE4_SPEC_COMPS.md:7-16` — architecture/descriptive boundary.
- `.plans/UMPIRE4_SPEC_MODEL_ARCH.md:1-65` — scoped model contract.
- `.flow/specs/fn-42-centralize-configuration-authoring-with.json` — completed-prerequisite support that remains open/unready with SHIP review.
- `.flow/specs/fn-48-canonicalize-known-gaps-as-a-checked-set.json` — retained support depending on retained fn-43 and fn-47.
- `Makefile:1283-1317` — PHONY and focused lint target style.

### Quick commands
`go test -count=1 -tags test_dep ./tools/planindex/...`
## Acceptance
- [ ] Every current plan Markdown file and Flow spec JSON has one canonical ordered registry row and no row remains unclassified.
- [ ] Flow rows encode exact scope, phase, disposition, status, readiness, completion-review status, and complete dependency sets; fn-42/fn-44/fn-50 are completed-prerequisite open-SHIP support, and fn-43 plus fn-45..49/fn-51 are retained support at their actual readiness.
- [ ] Production-root tests detect an added, removed, renamed, or multiply registered plan/spec.
- [ ] Parser/schema/unit tests pass; the not-yet-reconciled production command fails only with the expected deterministic document/Flow drift.
- [ ] The Make target is focused/read-only and existing comments are preserved.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
