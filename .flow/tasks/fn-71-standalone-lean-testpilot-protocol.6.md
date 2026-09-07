---
satisfies: [R7]
---
# fn-71-standalone-lean-testpilot-protocol.6 Prove a Testpilot-only Producer through Go admission

## Description
Add the independent cross-language proof that a Producer can import only Testpilot, author a bounded generated Case ergonomically, serialize it through the library, and pass Go semantic admission without target I/O.

**Size:** M
**Files:** `model/Testpilot/Examples/Synthetic.lean`, `model/TestpilotTests/Synthetic.lean`, `model/Temporal/Tool/Testpilot.lean`, `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go`, `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate_test.go`, `tests/testcore/testpilot/testdata/*.json`, `tests/testcore/testpilot/artifact_test.go`, `common/testing/testpilot/case_test.go`
**Touches:** [`model/Testpilot/Examples/Synthetic.lean`, `model/TestpilotTests/Synthetic.lean`, `model/Temporal/Tool/Testpilot.lean`, `tools/umpire/cmd/umpire-gen-case-runtime-conformance/**`, `tests/testcore/testpilot/testdata/*.json`, `tests/testcore/testpilot/artifact_test.go`, `common/testing/testpilot/case_test.go`]

### Approach
- Construct one representative Case in a module whose transitive imports are only Testpilot/neutral dependencies and whose provenance is arbitrary, non-Umpire bytes.
- Render it repeatedly through `Testpilot.ProtoJSON.canonical` and add the result to the separately owned functional fixture manifest.
- Strictly Decode and Prepare it under an existing local Profile without target I/O. Avoid Contract Run ID in this prepared Case if current Prepare still rejects it; task 3 covers its codec/decode representation.
- Round-trip empty and non-UTF-8 producer bytes and a resolved `Any`. Create malformed or context-invalid negatives only at the raw JSON/Go admission boundary after valid serialization.
- Verify descriptor, bounds, identity, and scope failures remain Go admission concerns, and ordinary Go unit tests never invoke Lean.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go:220-330`
- `tests/testcore/testpilot/artifact_test.go`
- `common/testing/testpilot/case.go`
- `common/testing/testpilot/internal/verification/prepare.go`
- `.flow/memory/bug/integration/moved-conformance-tests-must-not-import-2026-09-06.md`
## Acceptance
- [ ] The synthetic Producer has no Umpire or Temporal dependency path and uses only public Testpilot authoring and codec APIs.
- [ ] Repeated Lean rendering is deterministic and transactionally owns the checked fixture.
- [ ] Strict Go Decode plus Prepare succeeds under a local no-I/O Profile.
- [ ] Empty/non-UTF-8 provenance and resolved `Any` bytes round-trip exactly without Umpire interpretation.
- [ ] Malformed wire data and invalid descriptor/bounds/identity/scope inputs retain explicit Go rejection coverage.
## Done summary
Added a producer-neutral synthetic Testpilot Case using only the public Authoring and ProtoJSON APIs, with deterministic transactional fixture ownership and strict no-I/O Go admission. Focused coverage preserves empty and non-UTF-8 provenance plus resolved Any bytes exactly, and rejects malformed wire, unknown descriptors, invalid bounds, invalid identities, and unbound scope at the Go boundary; ordinary Go tests only consume the checked fixture.

The pre-edit and post-edit Lean, import-graph, transactional conformance, scoped Go, and model-lint gates passed. The accepted `make lint-code GOLANGCI_LINT_FIX=false` baseline from task `.5` was reused because it contains 1361 unrelated inherited findings.

stage: impl-review - ran (codex:gpt-5.6-sol:medium, SHIP)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: cd model && mise exec -- lake build Testpilot TestpilotTests UmpireTests TemporalModelTests temporal-testpilot, cd model && mise exec -- lake exe modelLintTests, TMPDIR=/private/tmp CGO_ENABLED=0 make umpire-check-case-runtime-conformance, TMPDIR=/private/tmp CGO_ENABLED=0 mise exec -- go test -p=1 -count=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/cmd/umpire-gen-case-runtime-conformance ./tests/testcore/testpilot/..., make lint-model, BASELINE_REUSED: make lint-code GOLANGCI_LINT_FIX=false - accepted inherited 1361 findings from fn-71-standalone-lean-testpilot-protocol.5, IMPL_REVIEW: codex:gpt-5.6-sol:medium SHIP (/tmp/fn71-task6-impl-review.json)
- PRs:
