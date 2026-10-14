---
satisfies: [R2, R3, R4]
---
# fn-71-standalone-lean-testpilot-protocol.2 Adopt the generated standalone Testpilot protocol

## Description
Promote the successful prototype into the reproducible Testpilot protocol boundary. The Testpilot `.proto` closure becomes the sole structural source for Lean wire declarations, with a focused generation/load contract and stale-output detection where output is checked in.

**Size:** M
**Files:** `model/Testpilot/Protocol.lean`, generated files selected by task 1, `model/Testpilot.lean`, `model/TestpilotTests.lean`, `model/lakefile.toml`, `model/lake-manifest.json`, `Makefile`, `model/ModelLint/ImportGraph.lean`, `model/ModelLint/ImportGraphTests.lean`
**Touches:** [`model/Testpilot/Protocol*`, `model/Testpilot.lean`, `model/TestpilotTests*`, `model/lakefile.toml`, `model/lake-manifest.json`, `Makefile`, `model/ModelLint/ImportGraph*.lean`]

### Approach
- Adopt task 1's measured generation or compile-time loading mode for exactly the Testpilot closure; delete unused prototype-only scaffolding.
- Add a deterministic generation/check command when Lean source is checked in, or an equivalent build dependency that cannot compile against stale declarations when generation occurs during elaboration.
- Expose the generated messages and enums through `Testpilot.Protocol`; add first-class `Testpilot` and `TestpilotTests` Lake roots.
- Classify Testpilot in ModelLint and reject direct, neutral-bridge transitive, and unknown-module paths from Testpilot into Umpire or Temporal using the existing structured shortest-path diagnostic style.
- Prove the generated `ProgramExpression` and `ContractExpression` remain distinct through narrow-import construction and elaboration-failure fixtures; the ergonomic surface is completed in task 3.

### Investigation targets
**Required** (read before coding):
- task 1 prototype and evidence
- `model/ModelLint/ImportGraph.lean:21-240`
- `model/ModelLint/ImportGraphTests.lean:49-290`
- `model/Tools/LeanImportGraph.lean`
- `Makefile:1000-1105`

**Optional** (reference as needed):
- `.plans/UMPIRE4_SPEC.md` current rule map
## Acceptance
- [ ] The `.proto` import closure is the sole Testpilot wire-structure source, and its reproducible generation/load command covers no unrelated Temporal schemas.
- [ ] A schema or checked-in generated-output change cannot leave stale Lean declarations compiling.
- [ ] First-class Testpilot/TestpilotTests roots build without importing Umpire or Temporal.
- [ ] ModelLint rejects direct, transitive bridge, and unclassified forbidden dependency paths with deterministic diagnostics while existing legal paths pass.
- [ ] Generated Program and Contract expression types remain statically distinct before the convenience API is added.
## Done summary
Promoted the exact-schema `#load_proto_file` prototype into the standalone `Testpilot.Protocol` boundary with traced schema invalidation, colocated isolated tests, and closed direct/transitive/unclassified import-policy coverage. Added first-class Lake roots and compile-time checks that generated Program and Contract expressions remain distinct; the user retains ownership of all uncommitted changes.

stage: impl-review - ran (SHIP after stale-artifact fix and colocated-layout re-review)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: make umpire-check-testpilot-protocol, schema mutation proof: changing value.proto rebuilt testpilotProtocolSchemas and Testpilot.Protocol; restoration rebuilt both, warm schema build: make umpire-check-testpilot-protocol completed without rebuilding Testpilot.Protocol, (cd model && mise exec -- lake build Testpilot TestpilotTests UmpireTests TemporalModelTests temporal-testpilot), (cd model && mise exec -- lake exe modelLintTests), make umpire-check-case-runtime-conformance, CGO_ENABLED=0 TMPDIR=<physical-temp-root> mise exec -- go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/cmd/umpire-gen-case-runtime-conformance ./tests/testcore/testpilot/..., make lint-model, git diff --check -- <task paths>, baseline reused: make lint-code GOLANGCI_LINT_FIX=false inherited red (1361 findings) from task1; task2 changed no Go source
- PRs:
