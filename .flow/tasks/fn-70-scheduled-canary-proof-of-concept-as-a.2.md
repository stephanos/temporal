---
satisfies: [R2, R4, R8]
---
# fn-70-scheduled-canary-proof-of-concept-as-a.2 Give the checked Nexus success Case a finite target Workflow lifetime

## Description
Give the checked Nexus success Case a finite target Workflow lifetime.

**Size:** M
**Files:** model/Temporal/Feature/Nexus/Success/Producer.lean; model/Temporal/Feature/Nexus/Success/TestpilotTests.lean; model/Temporal/Feature/Nexus/Success/Tests.lean; tests/testcore/testpilot/testdata/async-nexus-case.json; tests/testcore/testpilot/artifact_test.go; tools/umpire/cmd/umpire-gen-case-runtime-conformance/**
**Touches:** [model/Temporal/Feature/Nexus/Success/Producer.lean, model/Temporal/Feature/Nexus/Success/TestpilotTests.lean, model/Temporal/Feature/Nexus/Success/Tests.lean, tests/testcore/testpilot/testdata/async-nexus-case.json, tests/testcore/testpilot/artifact_test.go, tools/umpire/cmd/umpire-gen-case-runtime-conformance/**]

### Approach
- After fn77 source re-anchor, capture exact original Case bytes/provenance and complete raw transitive axiom inventories for changed lowering/proof declarations before edits. Reuse actual fn77 typed request constructors if changed; do not recapture baseline afterward.
- Add Producer-owned StartWorkflowExecution workflow_execution_timeout of 60 seconds using the established typed duration/assignment surface. Preserve checked success Property/Contract and correlation; no canary request patch, no new monitor, no broad CaseSupport budget change.
- Verify target lifetime exceeds the existing 30s execution plus three separate 5s settle/cleanup/Close phases; if fn77 changed those values, reconcile explicitly before editing rather than silently reduce Case meaning.
- Regenerate canonical fixture only through owner tooling, verify deterministic bytes and intentional new digest, preserve checked provenance/Known Gaps and prove unchanged Contract. Add focused lowering and decoded-fixture timeout assertions and run the actual owner test modules.

### Investigation targets
**Required:**
- model/Temporal/Feature/Nexus/Success/Producer.lean:105,142,251 — request, cleanup, checked lowering.
- model/Temporal/Testpilot/CaseSupport.lean:42 — current execution/cleanup budgets.
- common/testing/testpilot/internal/execution/runtime.go:54,64,75 — separate bounded phases.
- tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go:367 — owner fixture manifest.

### Quick commands
`cd model && mise exec -- lake build Temporal.Feature.Nexus.Success.ProducerTests Temporal.Feature.Nexus.Success.Tests`
`mise exec -- go test -count=1 -tags test_dep ./tests/testcore/testpilot ./tools/umpire/cmd/umpire-gen-case-runtime-conformance`
`mise exec -- make umpire-gen-case-runtime-conformance` (intentional owner publication after baseline)
`mise exec -- make umpire-check-case-runtime-conformance`

### Execution constraints
Read native fn70 R1–R10 and repository guides. Preserve comments and unrelated dirty source. Do not stage, commit, or push; the user owns commits. Fn77 Producer edits finish first for source serialization only: re-anchor exact delivered source/artifact before fn70 edits, without introducing a semantic prerequisite. No fn79 operation cancellation, replacement Driver/evaluator, runtime Lean invocation, general recovery/lease framework, production deployment/config mutation, or fn29 machinery. Proposed new file owners may reuse an established equivalent; record actual paths. Run Lean jobs serially. Baseline existing focused tests before edits; new named tests apply after creation and must be wired into the actual package/module roots. No silent skips, unmatched test regexes, fixture expectation weakening, or inferred passing gates.

Create proposed Temporal.Feature.Nexus.Success.ProducerTests for the timeout/lowering qualification and import it from the existing Temporal.Feature.Nexus.Success.Tests aggregate. Build the new module explicitly after creation; baseline only the existing aggregate beforehand.

## Acceptance
- [ ] Producer declares a finite 60s server Workflow execution lifetime; functional/canary consume the same resulting artifact.
- [ ] Checked success Contract, correlation and source bindings remain intact; only intended Program/artifact identity changes are recorded.
- [ ] Original-to-final raw axiom comparison adds no assumptions; deterministic owner generation and decoded timeout checks pass.
- [ ] No request rewriting, monitor duplication or unrelated generated fixture changes enter the delivery.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
