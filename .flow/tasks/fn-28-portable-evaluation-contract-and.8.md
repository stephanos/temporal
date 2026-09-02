---
satisfies: [R5, R6]
---

# fn-28-portable-evaluation-contract-and.8 Expose the resident executor through bounded HTTP protobuf transport
## Description

Add a thin long-lived HTTP adapter that accepts one protobuf execution request and returns one protobuf result while delegating all behavior to the executor module.

**Size:** M
**Files:** `tools/umpire/executorhttp/**`
**Touches:** [`tools/umpire/executorhttp/**`]

### Approach
- Use bounded protobuf request/response bodies, fixed routes and methods, request contexts, and explicit deadlines; expose no checker, model, executable, endpoint, credential, or retry selectors.
- Keep transport errors distinct from operational and semantic statuses; a client disconnect cancels owned work and cannot publish partial success.
- Prove one handler instance serves multiple sequential requests and rejects overlapping requests through the executor's atomic pre-I/O `busy` path without starting Go, Lean, Make, or shell subprocesses.

### Investigation targets

**Required** (read before coding):
- Parent executor and generated protobuf messages.
- Repository HTTP handler limits, error encoding, graceful shutdown, and `httptest` patterns.
- Existing Umpire structured error and bounded-capture conventions.

## Acceptance
- [ ] Exact bounded protobuf requests reach the executor and return canonical protobuf results with distinct transport/tooling statuses.
- [ ] Wrong method/path/content, unknown fields, oversized bodies, cancellation, deadlines, and malformed protobuf fail closed without semantic success.
- [ ] Sequential requests reuse one handler/executor instance and focused race/lint tests pass.
- [ ] Concurrent HTTP requests cannot bypass single-flight admission; the loser receives typed `busy` plus local `inconclusive` before runtime I/O.

## Done summary
Finalized the previously implemented bounded executor HTTP transport after auditing implementation commits `99fb684a17c70e3c8a863da9157408a502364d44` and `7bae3db49e9fa28d88acd7072fac4e34ba49bc78` plus the authoritative Codex SHIP review over `122b02fa1cfe6d5e94e9d8c79cf8d121179208f0..7bae3db49e9fa28d88acd7072fac4e34ba49bc78`. The reviewed adapter remains unchanged: fixed method/path/content handling, canonical bounded protobuf request/result admission, unknown/malformed/oversized fail-closed behavior, client cancellation and transport-wide deadlines, no partial success, sequential reuse, and atomic overlap rejection as typed `busy`/`inconclusive` before runner I/O are covered and green.

Fresh late-finalization verification passed for the focused executor HTTP unit, resident-executor integration, race, temporary-overlay fuzz, vet, and package-lint gates with workspace-isolated temporary directories. The host `PATH` resolved the Lean-bundled clang, which lacks the macOS standard-header search path, so verification pinned `CC`, `CXX`, and `SDKROOT` to Xcode; no product code or task `.9` artifact changed.

baseline: green (focused executor HTTP unit/resident-executor integration/race/fuzz/vet/package lint; no task code changed during finalization)

GATE_CLASSIFY_FULL: empty finalization diff refuses tier-B; unrelated repository-wide ENOSPC gates were not rerun under the scoped continuation instruction

stage: impl-review - ran [2026-09-02T00:58:08Z..2026-09-02T01:11:07Z] (authoritative committed Codex SHIP receipt reused; no new review dispatched)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 99fb684a17c70e3c8a863da9157408a502364d44, 7bae3db49e9fa28d88acd7072fac4e34ba49bc78
- Tests: TMPDIR=$PWD/.flow/tmp/fn28_8_resume_tmp GOTMPDIR=$PWD/.flow/tmp/fn28_8_resume_tmp TEST_TELEMETRY_DIR=$PWD/.flow/tmp/fn28_8_resume_tmp CC=$(xcrun --find clang) CXX=$(xcrun --find clang++) SDKROOT=$(xcrun --show-sdk-path) go test -count=1 -tags test_dep ./tools/umpire/executorhttp/... ./tools/umpire/executor/... (pass), TMPDIR=$PWD/.flow/tmp/fn28_8_resume_tmp GOTMPDIR=$PWD/.flow/tmp/fn28_8_resume_tmp TEST_TELEMETRY_DIR=$PWD/.flow/tmp/fn28_8_resume_tmp CC=$(xcrun --find clang) CXX=$(xcrun --find clang++) SDKROOT=$(xcrun --show-sdk-path) go test -count=1 -race -tags test_dep ./tools/umpire/executorhttp/... (pass), TMPDIR=$PWD/.flow/tmp/fn28_8_resume_tmp GOTMPDIR=$PWD/.flow/tmp/fn28_8_resume_tmp TEST_TELEMETRY_DIR=$PWD/.flow/tmp/fn28_8_resume_tmp CC=$(xcrun --find clang) CXX=$(xcrun --find clang++) SDKROOT=$(xcrun --show-sdk-path) go test -count=1 -tags test_dep -overlay $PWD/.flow/tmp/fn28_8_resume_tmp/fuzz-overlay.json -run ^$ -fuzz ^FuzzHandlerWireSurfaceFailsClosed$ -fuzztime=100x ./tools/umpire/executorhttp (pass), TMPDIR=$PWD/.flow/tmp/fn28_8_resume_tmp GOTMPDIR=$PWD/.flow/tmp/fn28_8_resume_tmp TEST_TELEMETRY_DIR=$PWD/.flow/tmp/fn28_8_resume_tmp CC=$(xcrun --find clang) CXX=$(xcrun --find clang++) SDKROOT=$(xcrun --show-sdk-path) go vet -tags test_dep ./tools/umpire/executorhttp/... (pass), TMPDIR=$PWD/.flow/tmp/fn28_8_resume_tmp GOTMPDIR=$PWD/.flow/tmp/fn28_8_resume_tmp TEST_TELEMETRY_DIR=$PWD/.flow/tmp/fn28_8_resume_tmp CC=$(xcrun --find clang) CXX=$(xcrun --find clang++) SDKROOT=$(xcrun --show-sdk-path) .bin/golangci-lint-v2.13.1 run --build-tags disable_grpc_modules,test_dep --timeout 10m --fix=false --config=.github/.golangci.yml ./tools/umpire/executorhttp/... (pass: 0 issues), GATE_CLASSIFY_FULL: empty finalization diff - refusing tier-B; unrelated repository-wide ENOSPC gates not rerun under scoped continuation instruction, AUTHORITATIVE_REVIEW_SHIP: 122b02fa1cfe6d5e94e9d8c79cf8d121179208f0..7bae3db49e9fa28d88acd7072fac4e34ba49bc78
- PRs: