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
Reconciled Umpire ownership and execution-boundary documentation, pinned the relocated generated source with a deterministic byte-diff gate, and added AST/workflow guardrails against restoring legacy authority or hidden TestEnv coupling. Make and CI now run the complete tagged `^TestUmpire` live suite and accept only the exact recorded Umpire2/Umpire3 failure identities; the aggregate gate and 270 model jobs pass, global lint remains exact-baseline inherited red at 1,374 findings, and task-scoped lint reports zero findings.

stage: impl-review - ran [2026-09-04T07:43:22Z..2026-09-04T07:55:12Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 6d9df614713bdf3a530d1fd24ace9f314e1b36f1, 65a015b25b956cb7ce6b07f49dcecf03c36aeaf2, 77628e5564ec235bff6431828e53a928a6725f72
- Tests: baseline: focused and race green; INHERITED_RED: go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpire' reproduced the exact nine Umpire2/Umpire3 failure identities; make fmt-imports green; INHERITED_RED: make lint-code reported 1,374 repository findings, GATE_SKIPPED:unittest:green-receipt f546aefa, TMPDIR=/private/tmp CC=/usr/bin/clang CXX=/usr/bin/clang++ go test -count=1 -tags test_dep ./tools/umpire/temporal/local/... ./tools/umpire/runner/... ./tools/umpire/temporal/nexus/... ./tools/umpire/runevaluation/..., TMPDIR=/private/tmp CC=/usr/bin/clang CXX=/usr/bin/clang++ go test -race -count=1 -tags test_dep ./tools/umpire/temporal/local/..., make umpire-check-generated-go-test, go test -count=1 -tags test_dep ./tools/umpire/regression/..., go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpire' (run by umpire-check-live-tests; exact inherited Umpire2/Umpire3 failure set accepted, zero Umpire4 failures), make umpire-check-regression (run with physical TMPDIR and Apple clang; green receipt 77628e55; 270 model jobs green), make fmt-imports, INHERITED_RED: make lint-code (exact pre-change 1,374 repository findings; known unrelated formatter side effect restored), .bin/golangci-lint-v2.13.1 run --build-tags 'disable_grpc_modules,test_dep,integration' --timeout 10m --fix=false --new-from-rev=be2fe95a4358ecc49df86536856bbbf38b9e74dd --config=.github/.golangci.yml ./tools/umpire/regression/... (0 issues), Codex impl-review /tmp/impl-review-receipt-fn-55-migrate-and-separate-the-local-temporal.4.json: SHIP, 0 findings
- PRs: