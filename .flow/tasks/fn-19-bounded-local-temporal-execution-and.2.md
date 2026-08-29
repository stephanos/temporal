---
satisfies: [R1, R2, R7]
---
# fn-19-bounded-local-temporal-execution-and.2 Build the domain-neutral checked runtime and participant contracts

## Description
Implement R1/R2's deep reusable Go boundary for checked requests, participant programs, commands, receipts, resources, and preflight.

**Size:** M
**Files:** `tools/umpire/runtime/runtime.go`, `tools/umpire/runtime/request.go`, `tools/umpire/runtime/participant.go`, `tools/umpire/runtime/errors.go`, `tools/umpire/runtime/request_test.go`
**Touches:** [tools/umpire/runtime/runtime.go, tools/umpire/runtime/request.go, tools/umpire/runtime/participant.go, tools/umpire/runtime/errors.go, tools/umpire/runtime/request_test.go]

### Approach
- Consume only fn-18 admitted typed sets; require exact two-member ExperimentSpec/RuntimeConfiguration input and never parse persisted bytes.
- Define immutable CheckedRunRequest, closed participant program/command/receipt/resource vocabulary, adapter/environment interfaces, stable error kinds, and bounded identity/value types.
- Validate profile/config/program/target/action/occurrence/participant/protocol/capability/budget/run/seed/attempt relations before invoking a factory.
- Use constructor/private-field discipline so callbacks, arbitrary maps, alternate semantic values, and unchecked receipts cannot enter the engine.
- Prove every preflight mutation returns no request and never calls an IO-counting fake factory.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/artifact` admitted set/types after fn-18
- Task `.1` exact profile values
- parent spec participant and preflight contracts

### Acceptance
- [ ] The package contains no Temporal/Nexus import or vocabulary.
- [ ] Every exact preflight failure is typed, deterministic, and side-effect free.
- [ ] One valid fixture produces an immutable checked request; reordered/duplicate/drifted inputs reject rather than normalize.
- [ ] No byte decoder, writer, mapping, evaluator, or general plugin surface exists.

## Acceptance
- [ ] R1/R2 checked runtime/participant boundary is deep, inert, and domain-neutral.
- [ ] Focused Go request/contract tests pass.
- [ ] Every public value enforces its documented Limits.

## Done summary
Implemented the deep domain-neutral checked runtime boundary and an artifact-owned exact typed executable-set projection, with immutable constructor-enforced requests, programs, commands, receipts, resources, correlations, limits, and typed side-effect-free preflight failures. Task-owned runtime/artifact tests, temporaltest, LocalProfileTests, race, vet, and diff checks are green; the later-task local/nexus/CLI packages, Nexus ExecutionTests, and make target remain absent exactly as recorded by the inherited red baseline. Memory capture was attempted after the non-trivial review fix but the enabled store is not initialized.

stage: impl-review - ran [round 1 NEEDS_WORK; round 2 SHIP at 2026-08-29T12:28:42.449936Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: cf04856da00d72e339acdde1ac15cf893b9a1b41, 985c5239c6008d4d1d9cf52fa3abd6e10ef1f347
- Tests: go test -count=1 ./tools/umpire/runtime/..., go test -count=1 ./tools/umpire/artifact/..., go test -race -count=1 ./tools/umpire/runtime/..., go vet ./tools/umpire/runtime/... ./tools/umpire/artifact/..., git diff --check, go test -count=1 ./temporaltest/..., cd model && mise exec -- lake build Temporal.System.Execution.LocalProfileTests, INHERITED_RED:go test -count=1 ./tools/umpire/temporal/local/... - package belongs to later task and is absent at baseline, INHERITED_RED:go test -count=1 ./tools/umpire/temporal/nexus/... - package belongs to later task and is absent at baseline, INHERITED_RED:go test -count=1 ./tools/umpire/cmd/umpire-local-run/... - package belongs to later task and is absent at baseline, INHERITED_RED:cd model && mise exec -- lake build Temporal.Feature.Nexus.ExecutionTests - target belongs to later task and is absent at baseline, INHERITED_RED:make umpire-run-local SET=tools/umpire/temporal/nexus/testdata/caller-closure-input-set OUTPUT_ROOT=/tmp/umpire-local-runs RUN_ID=umpire.local.caller-closure.run-1 - target belongs to later task and is absent at baseline
- PRs:
