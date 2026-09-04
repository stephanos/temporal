---
satisfies: [R1, R3, R4, R5, R7]
---
# fn-55-migrate-and-separate-the-local-temporal.4 Reconcile ownership documentation and no-legacy gates

## Description
Reconcile runtime, architecture, generator, Makefile, and CI descriptions with TestEnv-owned cluster/client lifecycle and Umpire-owned run lifecycle. Add durable gates proving the deprecated authority cannot re-enter Umpire and run the complete focused and regression verification matrix.

**Size:** S
**Files:** `tools/umpire/runtime/README.md`, `tools/umpire/runevaluation/README.md`, `tools/umpire/portableevaluation/README.md`, `.plans/UMPIRE4_COMPONENTS.md`, `.plans/UMPIRE4_ORDER.md`, `Makefile`, `.github/workflows/umpire.yml`, `tools/umpire/regression/ci_workflow_test.go`
**Touches:** [tools/umpire/runtime/README.md, tools/umpire/runevaluation/README.md, tools/umpire/portableevaluation/README.md, .plans/UMPIRE4_COMPONENTS.md, .plans/UMPIRE4_ORDER.md, Makefile, .github/workflows/umpire.yml, tools/umpire/regression/ci_workflow_test.go]

### Approach
- Replace invocation-owned loopback/server/client wording with the exact ownership split: `tests/testcore.TestEnv` owns cluster/client; Umpire owns wrapper/worker/endpoints/workflows/run resources and cleans them before TestEnv cleanup.
- Document explicit factory composition and removal of the zero-option Nexus/local factory without describing TestEnv as a production runtime surface.
- Point generated/live Umpire commands only at `tests/` with `test_dep integration`; keep unit commands under `tools/umpire` on `test_dep` and preserve the existing aggregate regression job.
- Extend regression workflow tests to pin the relocated generated file, TestEnv live-test command, and absence of a legacy package import/reference or production TestEnv import.
- Regenerate checked source through the existing generator and verify byte equality; do not rewrite historical completed specs.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/runtime/README.md:3-40,52-68` — runtime authority ownership
- `tools/umpire/runevaluation/README.md:120-178` and `portableevaluation/README.md:200-207` — live and portable commands
- `.plans/UMPIRE4_COMPONENTS.md:347-355,571-584` — component ownership map
- `Makefile:1132-1134` and `.github/workflows/umpire.yml:34-35` — aggregate gate wiring
- `tools/umpire/regression/ci_workflow_test.go:15,47-105` — command and generated-source guardrails
- `.plans/UMPIRE4_ORDER.md` — cleanup ordering and fn-55 summary

### Key context
- The no-legacy gate targets Umpire production/test sources; it does not delete or modify the repository's deprecated package.
- All Go tests include `test_dep`; only TestEnv live tests add `integration`.

## Acceptance
- [ ] Runtime, Run Evaluation, portable evaluation, architecture, and ordering docs state the exact TestEnv-versus-Umpire ownership boundary and explicit factory requirement.
- [ ] Makefile/CI run package-local fake-backed tests with `test_dep` and relocated live Umpire tests under `tests/` with `test_dep integration`.
- [ ] Regression gates fail if Umpire regains a deprecated test-server import/reference, if production `tools/umpire` imports `tests/testcore`, or if generated output/destination drifts.
- [ ] Fresh generator output equals the checked-in relocated generated test bytes and exact Artifact bindings remain pinned.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/temporal/local/... ./tools/umpire/runner/... ./tools/umpire/temporal/nexus/... ./tools/umpire/runevaluation/... ./tools/umpire/cmd/umpire-gen-tests-go/...` passes.
- [ ] `go test -race -count=1 -tags test_dep ./tools/umpire/temporal/local/...` passes.
- [ ] `go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpire'` runs and either passes or matches the exact recorded pre-change failure set with no fn-55/Umpire4 failure.
- [ ] `make umpire-check-regression` and `make fmt-imports` pass; `make lint-code` runs and either passes or matches the exact recorded pre-change baseline with zero task-scoped findings.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
