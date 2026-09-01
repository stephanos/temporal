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
- [ ] No generated, artifact, checksum, fingerprint, or fn50-authored unrelated-file drift remains; separately authorized roadmap commits `428fc87d9` and `69ce0485f` are excluded from fn50 implementation and evidence.

## Done summary
Audited CallerClosure construction guidance and comment preservation without changing model or public documentation: FiniteMachine is already the ordinary route, the direct TransitionKernel route remains expert-only, and the pre-migration CallerClosure comment is byte-identical. Reconciled R5 to require literal full lint classification plus green task-scoped no-fix lint and trust/import checks; all task-owned checks satisfy that contract with no model, generated, Go, artifact, or roadmap drift.

stage: impl-review - ran [2026-09-01T05:17:29Z..2026-09-01T05:28:31Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4e9db4fd528005c64c6a415bda4a9b085f76ab73, deeaf771d3519fd6f988acd1e178089bebee3ef4, 6ccc89745bfbdedb39e2e5828f61ed3cefb64c16
- Tests: baseline: green (cd model && mise exec -- lake build Temporal.System.Nexus.Tests Temporal.System.Nexus.ImplementationLinkTests Temporal.ImplementationLinkTests.Nexus; 53 jobs), baseline: green (cd model && mise exec -- lake build TemporalModelTests TemporalExperimentalTests UmpireTests; 193 jobs), baseline: green (make umpire-check-regression; 243 jobs), baseline: green (make lint-model; 200 jobs and import-boundary checks), baseline: red (make lint-code; inherited 1373 findings — errcheck 220, exhaustive 6, forbidigo 211, govet 5, revive 792, staticcheck 135, testifylint 4; six auto-fix edits exact-inverse-restored), cd model && mise exec -- lake build Umpire.Target.ImportTests Umpire.Target.Tests.FiniteMachine Temporal.System.Nexus.Tests Temporal.System.Nexus.ImplementationLinkTests Temporal.ImplementationLinkTests.Nexus; 57 jobs, migration trust scan: no added sorry/admit/axiom and no CallerClosure production native_decide, GATE_SKIPPED:unittest:docs-only - cumulative diff classified tier-B (no executable paths touched), GATE_SKIPPED:unittest:docs-only - cumulative diff classified tier-B (no executable paths touched), GATE_SKIPPED:unittest:docs-only - cumulative diff classified tier-B (no executable paths touched), make lint-model; 200 jobs and import-boundary checks, make lint-code (classified inherited red: exact pre-edit 1373-finding inventory; no task Go path; six auto-fix edits exact-inverse-restored), .bin/golangci-lint-v2.13.1 run --build-tags disable_grpc_modules,test_dep --timeout 10m --fix=false --new-from-rev=8459df4d5c2220a6426583a3388491a829c9b6c6 --config=.github/.golangci.yml; 0 issues, flowctl validate --all; Valid=True, git diff --check, impl-review Codex SHIP receipt /tmp/impl-review-receipt-fn-50-migrate-system-callerclosure-to.3.json
- PRs:
