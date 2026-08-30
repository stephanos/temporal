---
satisfies: [R1, R2, R6]
---

# fn-20-local-execution-semantic-conformance.3 Implement the bounded Go checker process bridge

## Description
Implement the transport-only Go checker adapter and fixed sibling-process lifecycle (R1/R2/R6), independently testable with a fixture child.

**Size:** M
**Files:** `tools/umpire/runevaluation/checker.go`, `tools/umpire/runevaluation/protocol.go`, `tools/umpire/runevaluation/checker_test.go`, `tools/umpire/runevaluation/testdata/**`
**Touches:** [tools/umpire/runevaluation/checker.go, tools/umpire/runevaluation/protocol.go, tools/umpire/runevaluation/checker_test.go, tools/umpire/runevaluation/testdata/**]

### Approach
- Keep the checker transport interface and fixture injection package-private. The eventual exported production `Check(admittedSet)` resolves the fixed sibling internally and exposes no response-producing callback or implementation parameter.
- Encode exact non-path admitted bindings, separate Run/RawEvidence Known Gaps, and the remaining direct request preimage; strictly decode the exact response with bounded N+1 readers, canonical byte checks, closed enums, and complete echoed-binding validation.
- Resolve only a regular realpath-contained sibling checker, verify the handshake, and expose no path/env/PATH/plugin override.
- Run under one 30-second child context; on cancellation, timeout, malformed output, nonzero exit, or stderr bytes, terminate and reap before returning one sanitized error.
- Use a fixture child with independent literal responses to test every lifecycle/protocol failure without invoking semantic code.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-18-versioned-umpire-artifact-boundary.2.md` — bounded strict JSON kernel pattern
- `.flow/tasks/fn-18-versioned-umpire-artifact-boundary.3.md` — canonical mutation/limit fixtures
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md` §API Contracts — sanitized error and cancellation precedent
- `tools/umpire/regression/generated_view.go:201-225` — existing strict trailing/duplicate JSON precedent
- `tools/common/artifactio/set.go:475-645` — path/identity and interruption patterns
## Acceptance
- [ ] Valid fixture child round-trips exact request/response bytes, four artifact bindings, Known Gap arrays, and all Behavior Fingerprints.
- [ ] Missing, symlinked/misdirected/non-regular, wrong-handshake, timed-out, canceled, nonzero, stderr-writing, malformed, noncanonical, oversized, trailing, or stale-response children fail closed and are reaped.
- [ ] N and N+1 byte tests allocate only within the stated ceiling.
- [ ] No exported production API exposes a checker implementation, executable location, or semantic callback; injection remains package-private and test-only.
- [ ] Go race/fuzz tests cover protocol decoding and concurrent independent invocations.
## Done summary
Implemented the private 32-MiB-per-direction run-evaluation checker bridge with exact canonical v2 request/response bindings, fixed verified sibling resolution, sanitized lifecycle failures, cancellation/timeout termination, and deterministic child reaping. Review fixes now validate the nested Evidence/Result semantic projection and query/Property/Known-Gap closure, while the request writer streams canonical pretty JSON directly into the bounded sink with exact N/N+1 capacity and 32-KiB maximum downstream writes.

Focused, race, bounded fuzz, both Lean targets, and `make umpire-check-regression` pass. Scoped golangci and errortype pass; non-mutating repository `make lint-code` reports zero golangci findings for the task diff, then stops on the inherited `tools/umpire/runtime/errors.go:60:9 (et:unw+)`. The roadmap-wide Go command and Make target remain deferred to task `.6`, matching the pre-edit baseline. Codex review session `01a0506a-1b2e-7a23-bf23-6b1ebe8b04e9` reached SHIP after two NEEDS_WORK rounds; memory capture was attempted but the repository memory store is not initialized.

stage: impl-review - ran [2026-08-30T02:05:22Z..2026-08-30T02:33:16Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 38658e6cc86c3ee31b05389d1cc617f193e0d6cd, 1e66a2d467450467629c35ad5eb10af6daf0a50d, 7fadb0405c76207b087297913398e523e2880ab6
- Tests: baseline: green for task .3 checker/Lean/regression gates; roadmap command and Make target deferred to task .6, cd model && mise exec -- lake build Umpire.Observation.Tests.Check, cd model && mise exec -- lake build Temporal.Tool.RunEvaluationTests temporal-run-evaluation-checker, go test -tags test_dep -count=1 ./tools/umpire/runevaluation/..., go test -tags test_dep -race -count=1 ./tools/umpire/runevaluation/..., go test -tags test_dep -run '^$' -fuzz '^FuzzDecodeCheckerResponse$' -fuzztime=5s ./tools/umpire/runevaluation, .bin/golangci-lint-v2.13.1 run --verbose --build-tags disable_grpc_modules,,test_dep, --timeout 10m --fix=false --config=.github/.golangci.yml ./tools/umpire/runevaluation/..., go vet -tags disable_grpc_modules,,test_dep, -vettool=.bin/errortype -style-check=false ./tools/umpire/runevaluation/..., TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp PATH=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/bin:$PATH make umpire-check-regression, GOLANGCI_LINT_BASE_REV=1d65402ddd93ed1185fab41f2972389cba86705c GOLANGCI_LINT_FIX=false make lint-code (golangci: 0 task-diff issues; inherited tools/umpire/runtime/errors.go:60:9 et:unw+), DEFERRED(task .6): go test -count=1 ./tools/umpire/cmd/umpire-local-run-evaluation/..., DEFERRED(task .6): make umpire-check-local-run-evaluation SET=tools/umpire/temporal/nexus/testdata/caller-closure-run-set OUTPUT_ROOT=/tmp/umpire-local-results, impl-review codex session 01a0506a-1b2e-7a23-bf23-6b1ebe8b04e9: SHIP
- PRs:
