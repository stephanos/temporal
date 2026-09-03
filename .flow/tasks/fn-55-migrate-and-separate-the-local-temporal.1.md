---
satisfies: [R2, R3, R5]
---

# fn-55-migrate-and-separate-the-local-temporal.1 Add explicit Nexus authority injection and relocate its live proof

## Description
Add a supplied environment factory at the Nexus adapter boundary, then move the generated and hand-written live Nexus execution slice to the repository `tests/` package where it can use the shared TestEnv-backed attached factory. Keep pure Nexus behavior tests in the package with fakes and no live server; task `.3` removes the temporary compatibility fallback after every live caller has migrated.

**Size:** M
**Files:** `tools/umpire/temporal/nexus/runner.go`, `tools/umpire/temporal/nexus/runner_test.go`, `tools/umpire/temporal/nexus/integration_test.go`, `tools/umpire/temporal/nexus/participant_test.go`, `tools/umpire/temporal/nexus/caller_closure_path_test.go`, `tools/umpire/temporal/nexus/caller_closure_runner_generated_test.go`, `tools/umpire/temporal/nexus/binding_test.go`, `tools/umpire/cmd/umpire-gen-tests-go/generate.go`, `tools/umpire/cmd/umpire-gen-tests-go/generate_test.go`, `tests/umpire4_testenv_test.go`, `tests/umpire4_caller_closure_test.go`, `tests/umpire4_caller_closure_generated_test.go`
**Touches:** [tools/umpire/temporal/nexus/runner.go, tools/umpire/temporal/nexus/runner_test.go, tools/umpire/temporal/nexus/integration_test.go, tools/umpire/temporal/nexus/participant_test.go, tools/umpire/temporal/nexus/caller_closure_path_test.go, tools/umpire/temporal/nexus/caller_closure_runner_generated_test.go, tools/umpire/temporal/nexus/binding_test.go, tools/umpire/cmd/umpire-gen-tests-go/generate.go, tools/umpire/cmd/umpire-gen-tests-go/generate_test.go, tests/umpire4_testenv_test.go, tests/umpire4_caller_closure_test.go, tests/umpire4_caller_closure_generated_test.go]

### Approach
- Add one explicit Nexus binding constructor that requires `umpireruntime.EnvironmentFactory`; preserve the existing model/program methods and validate nil input with a deterministic construction error before runtime I/O.
- During this migration task only, retain the legacy zero-value fallback for not-yet-migrated Run Evaluation callers; mark its deletion as task `.3` and add a focused test proving explicitly constructed bindings use only the supplied factory. Do not add a global registry or any TestEnv-aware production declaration.
- Add `tests/umpire4_testenv_test.go` as the sole test-only adaptation point from `TestEnv.SdkClient`, namespace, and frontend address to `local.AttachedAuthority`/`NewAttachedFactory`. Reuse one TestEnv only for sequential cases; create independent environments for parallel cases.
- Relocate live cases from Nexus integration, participant, and caller-closure path files into `tests/umpire4_caller_closure_test.go`. Leave admission, evidence, configuration, adapter, and participant unit cases in place and replace any live dependency with deterministic fakes.
- Retarget the caller-closure generator's checked output to `tests/umpire4_caller_closure_generated_test.go`. Keep the canonical Artifact fixtures single-source, load them through a tests-only repo-relative helper, and preserve exact binding literals, subject preflight, bounded runner execution, Run Evaluation, normal/faulted meaning, transport distinction, and deterministic fresh-output equality.
- Preserve all existing comments with moved code; update only ownership/default-factory comments made false by explicit injection.

### Investigation targets
**Required** (read before coding):
- `tests/testcore/test_env.go:268-382,492-565` — supported TestEnv lifecycle, client, namespace, context, and worker access
- `tests/umpire4_portable_executor_test.go:32-70,269-309,330-345` — established attached factory and adapter override
- `tools/umpire/temporal/nexus/runner.go:9-43` — zero-value binding and hidden factory
- `tools/umpire/runner/runner.go:78-161` — adapter preflight and incomplete-adapter classification
- `tools/umpire/temporal/nexus/integration_test.go:14-110` and `participant_test.go:19-110` — live cases to relocate
- `tools/umpire/temporal/nexus/caller_closure_path_test.go:410-540,697-780` — live proof helpers and explicit adapter override
- `tools/umpire/cmd/umpire-gen-tests-go/generate.go:147-205,379-445` and `generate_test.go:14-160` — generated destination, imports, factory use, and deterministic oracle
- `tools/umpire/temporal/nexus/binding_test.go:80-135` — canonical input loader and generated binding literals

### Key context
- `TestEnv` must remain wholly under `tests/`; production and package-local Umpire tests must not import `tests/testcore`.
- The crossed-input and subject-preflight cases must prove zero factory access and zero TestEnv/runtime I/O.
- Do not copy the canonical Artifact fixture tree merely to satisfy `go:embed`; use the established tests-package fixture loader and retain exact checksum assertions.

## Acceptance
- [ ] The explicit Nexus constructor rejects a nil factory deterministically, preserves `CheckRequest`, `NewParticipant`, and `ValidateOutput`, and explicitly constructed bindings consult only the supplied factory; task `.3` owns deletion of the temporary zero-value fallback.
- [ ] Normal, duplicate-delivery, limit, and generated portability live cases run under `tests/` with `testcore.TestEnv` and `local.NewAttachedFactory`; no live server is started by a Nexus package test.
- [ ] Crossed input and subject drift fail before factory access; normal/faulted operational closure, Evidence sources, semantic meaning, and transport-only differences remain exact.
- [ ] The generator emits the relocated checked test deterministically, retains exact Artifact/input/subject bindings, and fresh output equals the checked-in bytes.
- [ ] Canonical fixture data remains single-source; no production Umpire file imports `tests/testcore` and no global authority registry exists.
- [ ] Existing comments are preserved with moved code, except comments whose zero-option ownership statement becomes false.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-tests-go/... ./tools/umpire/runner/... ./tools/umpire/temporal/nexus/...` passes without starting a live server.
- [ ] `go test -count=1 -tags 'test_dep integration' ./tests -run '^(TestUmpire.*CallerClosure|TestUmpire.*Portability)'` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
