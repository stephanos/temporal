---
satisfies: [R3, R7, R10]
---
# fn-52-caller-neutral-grpc-portable-test-plans.5 Compile Lean tests into portable plans and prove parity

## Description
Make Lean a deterministic producer of the same typed plan accepted from external clients, retain exact existing model identities, and prove cross-language evaluation parity for R3 and R7.

**Size:** M
**Files:** `model/Umpire/Artifact/PortableEvaluationContract.lean`, `model/Temporal/Tool/PortableEvaluationContract.lean`, related Lean tests, generated portable fixtures, Go parity tests
**Touches:** [model/Umpire/Artifact/PortableEvaluationContract.lean, model/Temporal/Tool/PortableEvaluationContract.lean, model/Temporal/Tool/*PortableEvaluationContract*Tests.lean, tools/umpire/portableevaluation/testdata/**, tools/umpire/portableevaluation/*test.go]

### Approach
- Extend the existing closed Lean lowering rather than create another Property, Behavior, Query, or execution language.
- Emit the complete typed execution and verification programs plus exact ExperimentSpec/model bindings and compiler provenance.
- Lower every supported check to the shared finite vocabulary; emit explicit required/advisory external obligations for unsupported checks.
- Preserve existing ExperimentSpec checksums and fn-28 fixtures while adding separately versioned plan fixtures.
- Compare stable detailed Lean outcomes with Go results for every operator, branch, scope, obligation, and N/N+1 boundary.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Artifact/PortableEvaluationContract.lean:265-360` — reusable Lean contract data
- `model/Temporal/Tool/PortableEvaluationContract.lean:609-760` — current checked-Test lowering
- `model/Temporal/Tool/PortableEvaluationContractTests.lean` — lowering and failure fixtures
- `tools/umpire/portableevaluation/parity_test.go` — cross-language parity harness
- `tools/umpire/portableevaluation/testdata/**` — current generated contract/evidence fixtures

### Acceptance
- [ ] Lean emits deterministic typed plans accepted by the same path as external plans.
- [ ] Existing ExperimentSpec identities and fn-28 fixtures remain unchanged.
- [ ] Supported execution/verification operators and every result branch have Lean/Go parity.
- [ ] Unsupported checks are explicit obligations and cannot silently affect portable or model-bound success.
- [ ] Model bindings, compiler provenance, source locations, checksums, and obligation mutations reject deterministically.
- [ ] Focused Lean builds and Go parity tests pass.

## Acceptance
- [ ] R3 and R7 Lean production, identity retention, obligations, and parity are complete.
- [ ] Existing model/artifact identities remain stable.
- [ ] Focused Lean and Go tests pass.

## Done summary
Lean now compiles the existing caller-closure Test and duplicate-delivery control into the shared admitted PortableTestPlan protobuf without changing their ExperimentSpec or fn-28 identities. The generated plans preserve exact artifact projections and model provenance, retain supported checks, convert unsupported checked Property and Observation semantics into deterministic required obligations, and match Go admission, preparation, evaluation, mutation, scope, and N/N+1 behavior.

Review fixes added exact reconstruction of identity-bearing ExperimentSpec/RuntimeConfiguration fields and replaced synthetic obligation construction with actual checked Property/Observation inputs. Checked semantic failures are required because the Lean source has no advisory annotation; externally authored advisory obligations remain covered by the shared Go path.

baseline: red (the canonical Go commands were already blocked by the not-yet-landed executorgrpc package and the local Darwin cgo `stddef.h` failure; literal `make lint-code` already reported the same 1379-issue repository backlog)

Verification: `make proto`, focused Lean, generated fixture staleness, CGO-disabled scoped Go parity/preparation, `make umpire-check-regression` (270 jobs), `make lint-model` (236 jobs), and scoped non-mutating lint passed. Literal `make lint-code` reproduced exactly errcheck=220, exhaustive=5, forbidigo=211, govet=5, revive=798, staticcheck=136, testifylint=4; its known `tools/umpire1/monitor_test.go` auto-edit was restored.

stage: impl-review - ran [2026-09-03T21:47:32Z..2026-09-03T23:05:26Z] (model: gpt-5.6-sol)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 0cdd6df884cf044c5a945ed018562311ea62fb01, 7f4304a8d4168f591be1329ec28dd90158e5e08b, 85172a7076d12034f0ee88fa683f19854f0410cd
- Tests: make proto, cd model && mise exec -- lake build Temporal.Tool.PortableEvaluationContractTests, CGO_ENABLED=0 make umpire-gen-portable-evaluation-fixtures, CGO_ENABLED=0 make umpire-check-portable-evaluation-fixtures, CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/umpire/testplan/... ./tools/umpire/executor/... ./tools/umpire/portableevaluation/..., CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/umpire/portableevaluation ./tools/umpire/executor -run 'TestLeanGeneratedPortablePlansUseSharedAdmissionAndRetainExactBindings|TestLeanGeneratedPortablePlanRejectsChecksumBindingSourceAndLimitMutations|TestPrepareLeanGeneratedModelPlansRetainExactArtifactBindings', CGO_ENABLED=0 ./.bin/golangci-lint-v2.13.1 run --build-tags test_dep ./tools/umpire/testplan/... ./tools/umpire/executor/... ./tools/umpire/portableevaluation/..., make umpire-check-regression, make lint-model, git diff --exit-code ef58bdd6b2f5c1095eaa066f7063543a10cf507f -- model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean tools/umpire/temporal/nexus/testdata/caller-closure-input-set tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-input-set tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-run-set, baseline: red (go test -count=1 -tags test_dep ./tools/umpire/testplan/... ./tools/umpire/executor/... ./tools/umpire/executorgrpc/... ./tools/umpire/portableevaluation/...: inherited missing future executorgrpc package and Darwin cgo stddef.h), baseline: red (go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpirePortableGRPCExecutor$': inherited Darwin cgo stddef.h and future integration test dependency), make lint-code (inherited red reproduced exactly: 1379 issues; errcheck=220 exhaustive=5 forbidigo=211 govet=5 revive=798 staticcheck=136 testifylint=4)
- PRs: