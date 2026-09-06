---
satisfies: [R4, R5]
---
# fn-68-minimal-nexus3-success-demonstration.3 Prove the generated Case through the existing runtime test

## Description
Finish the demonstration with the existing fixture renderer, transactional fixture owner, and real local Temporal test (R4/R5). This task combines fixture ownership, integration assertions, and minimal command documentation.

**Size:** M
**Files:** Makefile, tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go and generate_test.go, tests/testcore/testpilot/testdata/async-nexus-case.json, tests/testcore/testpilot/artifact_test.go, tests/testpilot_async_nexus_case_test.go, model/README.md. The existing get-system-info-case.json is included in the adopted examples root but its bytes should stay unchanged.
**Touches:** [Makefile, tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go, tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate_test.go, tests/testcore/testpilot/testdata/*.json, tests/testcore/testpilot/artifact_test.go, tests/testpilot_async_nexus_case_test.go, model/README.md]

### Approach
- Extend the existing fixture tool with one separate Temporal-examples manifest/root for its two existing examples, reusing artifactio and the existing temporary-output-root publication pattern. Keep the default six conformance classes and their root unchanged; no new executable, artifact schema, or general registry. Extend the existing Makefile generation/check targets to publish and compare both independently owned roots. The check renders to a temporary root and diffs both complete trees, detecting stale example bytes without invoking Lean from Go tests. Record exact invocation syntax in model/README.md.
- Render the complete examples tree under a temporary root, decode/validate it, verify repeat determinism, then compare or publish through the existing transactional publisher. Regenerate async-nexus-case.json from the new renderer path.
- Add focused preparation/provenance assertions for that exact fixture and exercise the produced monitor with missing completion, foreign correlation IDs, and duplicate/unrelated history events. A duplicate cannot substitute for the required completed event. Reuse existing artifact test helpers; do not alter production runtime interpretation.
- Strengthen TestTestpilotAsyncNexusCase to assert cleanup succeeded and inspect its three supporting history Observations for scheduled/started/completed event types and matching scheduled-event/request references. Continue using the actual Profile, Testpilot Prepare, composite Temporal Driver, and existing local test environment.
- Update only the existing model README's Case-production paragraph/commands to point at the executable success slice and qualify the broader Markdown sketches. Run focused gates, then the required repository model/lint gates once. Use the documented physical TMPDIR workaround if the test environment requires it.

### Investigation targets
**Required:**
- tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go:124-179 — staged validation and transactional publisher
- tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go:213-216 — fixed conformance-class contract
- tests/testcore/testpilot/artifact_test.go:28-143 — fixture preparation and monitor helpers
- tests/testpilot_async_nexus_case_test.go:26-146 — existing real runtime demonstration
- model/README.md:34-38 — narrow documentation update
**Optional:**
- Makefile:1098-1129 — physical temporary path and live gate conventions
- .flow/memory/declined/generated-api-drift-verification.md — no broad drift/CI expansion

### Quick commands
`cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests temporal-case-runtime`
`mise exec -- go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-case-runtime-conformance ./tests/testcore/testpilot/...`
`mise exec -- go test -count=1 -tags 'test_dep integration' ./tests -run '^TestTestpilotAsyncNexusCase$'`
Fixture drift gate: `make umpire-check-case-runtime-conformance`.
Final once: `make umpire-build-model`, `make lint-model`, `make lint-code GOLANGCI_LINT_FIX=false`. Report unrelated baseline failures separately; the focused demo itself must pass.
## Acceptance
- [ ] R4's example fixtures render/check/publish through the existing tool; tests cover deterministic output, stale bytes, failed/incomplete rendering without partial publication, and unchanged six-class conformance behavior. The existing Makefile check compares both roots and fails on a deliberately stale async fixture; `make umpire-check-case-runtime-conformance` passes after regeneration. Ordinary tests do not invoke Lean.
- [ ] The checked-in async fixture prepares with Nexus3 bindings, and offline missing/foreign/duplicate-only evidence cannot satisfy its generated monitor.
- [ ] TestTestpilotAsyncNexusCase passes against the existing real local Temporal environment with completed disposition, satisfied Verdict, successful cleanup, and exactly correlated scheduled/started/completed supporting history.
- [ ] README records the smallest reproducible render/check/live commands and success-only scope; focused tests pass and required final model/lint results are recorded without adding CI or unrelated fixes.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
