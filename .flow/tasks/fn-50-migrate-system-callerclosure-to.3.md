---
satisfies: [R5]
---
# fn-50-migrate-system-callerclosure-to.3 Verify and document the CallerClosure migration

## Description
Review public guidance/comments and run final compatibility gates (R5).

**Size:** S
**Files:** `model/Temporal/System/Nexus/CallerClosure.lean`, `model/Umpire/ARCHITECTURE.md`, `model/README.md`, `model/ARCHITECTURE.md`
**Touches:** [model/Temporal/System/Nexus/CallerClosure.lean, model/Umpire/ARCHITECTURE.md, model/README.md, model/ARCHITECTURE.md]

### Approach
- Keep architecture docs unchanged where they already describe FiniteMachine as the ordinary route; revise only factually stale direct-kernel wording.
- Audit that every pre-existing CallerClosure comment remains or is minimally corrected for ownership.
- Run focused suites, aggregate builds, exact regressions, trust/import checks, and lint.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/System/Nexus/CallerClosure.lean:3-10` — module meaning that should remain stable
- `model/Umpire/ARCHITECTURE.md:82-105` — ordinary versus expert Target routes
- `model/README.md:140-155` — model authoring overview
- `model/ARCHITECTURE.md:160-175,290-305` — existing Target boundary

## Acceptance
- [ ] No public document or comment makes a stale claim about CallerClosure construction.
- [ ] Existing comments unrelated to the changed representation remain byte-for-byte present.
- [ ] Focused and aggregate Lean builds, exact regression, trust/import, and `make lint-model` pass.
- [ ] Literal `make lint-code` runs and reproduces only the pre-edit inherited 1,373-finding golangci baseline with no task Go path; the existing `tools/umpire/runtime/errors.go:60:9` `errortype` finding is unchanged, and a no-fix golangci run scoped from the task base passes.
- [ ] No generated, artifact, checksum, fingerprint, or unrelated-file drift remains.

## Done summary
Audited the CallerClosure module comment and public Target-authoring guidance: the pre-migration
comment remains byte-for-byte intact, and the documentation already presents `FiniteMachine` as
the ordinary route with direct `TransitionKernel` construction reserved for expert cases. No model
or documentation edit was required. Focused, aggregate, regression, import/trust, and model-lint
gates pass with no generated or unrelated drift. The literal `make lint-code` run reproduces its
pre-edit inherited 1,373-finding branch-wide golangci baseline with no task Go path, and the existing
`tools/umpire/runtime/errors.go:60:9` full-tree `errortype` finding is unchanged. The separately
required no-fix golangci run scoped from the task base reports zero issues.

## Evidence
- Commits: 4e9db4fd528005c64c6a415bda4a9b085f76ab73
- Tests: cd model && mise exec -- lake build Temporal.System.Nexus.Tests Temporal.System.Nexus.ImplementationLinkTests Temporal.ImplementationLinkTests.Nexus; cd model && mise exec -- lake build TemporalModelTests TemporalExperimentalTests UmpireTests; make umpire-check-regression; make lint-model; cd model && mise exec -- lake build Umpire.Target.ImportTests Umpire.Target.Tests.FiniteMachine Temporal.System.Nexus.Tests Temporal.System.Nexus.ImplementationLinkTests Temporal.ImplementationLinkTests.Nexus; migration trust scan (no added sorry/admit/axiom and no CallerClosure production native_decide); make lint-code (classified inherited red: identical pre-edit 1,373 findings, no task Go path, six auto-fix edits exact-inverse-restored); GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=8459df4d5c2220a6426583a3388491a829c9b6c6 make lint-code (golangci: 0 task-diff issues; unchanged inherited errortype: tools/umpire/runtime/errors.go:60:9); .bin/golangci-lint-v2.13.1 run --build-tags disable_grpc_modules,test_dep --timeout 10m --fix=false --new-from-rev=8459df4d5c2220a6426583a3388491a829c9b6c6 --config=.github/.golangci.yml
- PRs:
