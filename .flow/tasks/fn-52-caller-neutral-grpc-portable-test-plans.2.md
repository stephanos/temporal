---
satisfies: [R2, R3, R4, R8, R10]
---
# fn-52-caller-neutral-grpc-portable-test-plans.2 Define and admit the typed portable plan protocol

## Description
Define the self-contained protobuf plan, unary gRPC method, deterministic semantic identity, and structural admission surface for R2-R4. Keep the fn-28 messages and fixture bytes unchanged.

**Size:** M
**Files:** `proto/internal/temporal/server/api/umpire/v1/portable_test_plan.proto`, `proto/internal/temporal/server/api/umpire/v1/service.proto`, generated `api/umpire/v1/portable_test_plan*.go`, generated `api/umpire/v1/service*.go`, `tools/umpire/testplan/**`
**Touches:** [proto/internal/temporal/server/api/umpire/v1/portable_test_plan.proto, proto/internal/temporal/server/api/umpire/v1/service.proto, api/umpire/v1/portable_test_plan*.go, api/umpire/v1/service*.go, tools/umpire/testplan/**]

### Approach
- Put all successor messages/enums in `portable_test_plan.proto` and import it from the separate `service.proto`; leave fn-28's `message.proto` and generated descriptor untouched.
- Model the complete execution, verification, provenance, obligation, Limit, and result vocabularies as closed protobuf messages and enums; do not embed JSON or opaque sub-artifact bytes.
- Compute identity from deterministic serialization of the admitted decoded value with checksum/attestation fields excluded.
- Recursively reject unknown fields/enums, incomplete oneofs, duplicates, crossed bindings, invalid order, unsupported versions/operators, and hard-limit overflow.
- Construct and size the mandatory non-success result envelope during admission, including plan-derived Known Gaps/obligations and a result-byte-limit diagnostic; reject a plan whose result Limit cannot contain it.
- Retain the existing EvaluationContract/ExecuteRequest HTTP surface without aliases or reinterpretation.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/umpire/v1/message.proto:1-600` — existing portable vocabulary and HTTP envelope
- `proto/internal/temporal/server/api/testservice/v1/service.proto:1-13` — separate service schema convention
- `tools/umpire/evaluationcontract/contract.go:17-110` — deterministic admission and hard maxima
- `tools/umpire/internal/artifactv2/artifact.go:16-150` — typed ExperimentSpec/DrivePlan source meanings

**Optional** (reference as needed):
- `tools/umpire/evaluationcontract/contract_test.go` — mutation and canonicalization tests

### Acceptance
- [ ] `PortableTestPlan`, `ExecutionResult`, and generated unary UmpireExecutor client/server bindings compile from the conventional schemas.
- [ ] External and model-compiled fixtures express the same closed plan shape without opaque execution or verification documents.
- [ ] Equivalent valid protobuf encodings yield one plan checksum; unknown or mutated behavior-affecting input rejects before I/O.
- [ ] Exact N is admitted and N+1 rejects independently for every structural and byte Limit, including an undersized mandatory result envelope.
- [ ] Existing fn-28 `message.proto` descriptor, generated types, HTTP request/response, and fixtures remain byte-identical.
- [ ] `make proto` and focused testplan tests pass.
## Acceptance
- [ ] R2-R4 typed schema and admission behavior are implemented.
- [ ] R8 legacy message descriptor, generated types, HTTP surface, and compatibility fixtures remain unchanged.
- [ ] Proto generation and focused tests pass.
## Done summary
Implemented the caller-neutral typed PortableTestPlan protocol, deterministic admission/checksum surface, unary gRPC bindings, independent limits, source-faithful model/runtime semantics, and mandatory result reservation while preserving the fn-28 protocol byte-for-byte. Focused admission, generation, Lean model, scoped lint, and aggregate Umpire regression checks are green; the spec-wide Go/integration commands remain inherited-unavailable until downstream executor tasks create their packages/tests.

baseline: red (spec-wide Go/integration Quick commands referenced downstream packages/tests not yet implemented; repository-wide lint also inherited local cgo/toolchain failures)

stage: impl-review - ran [2026-09-03T18:26:38Z..2026-09-03T18:46:42Z] (model: gpt-5.6-sol)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ef8dd8c739b26605d838481431b2ce01c0b890fb, 8a5a29ba415f062e35c9e148fc76d308ea8ff5d4
- Tests: make proto, cd model && mise exec -- lake build Temporal.Tool.PortableEvaluationContractTests, go test -count=1 -tags test_dep ./tools/umpire/testplan/..., .bin/golangci-lint-v2.13.1 run --build-tags 'disable_grpc_modules test_dep' --timeout 10m --fix=false --new-from-rev=865f397ca2cf23115117a1b2ea5c591f6ebc5133 --config=.github/.golangci.yml ./tools/umpire/testplan/..., make lint-model, make umpire-check-regression, git diff --exit-code 865f397ca2cf23115117a1b2ea5c591f6ebc5133 -- proto/internal/temporal/server/api/umpire/v1/message.proto api/umpire/v1/message.pb.go api/umpire/v1/message.go-helpers.pb.go tools/umpire/evaluationcontract tools/umpire/executor/testdata
- PRs: