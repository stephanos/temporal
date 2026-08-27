---
satisfies: [R3, R4, R5]
---

# fn-32-add-umpire-refinement-and-the-first.4 Compose accepted System Model Traces through the Nexus Implementation Link

## Description
Prove the Run Evaluation-facing System-trace to Feature-Property handoff and layer-specific outcomes for R3–R5.

**Size:** M
**Files:** `model/Temporal/System/Nexus/**`, `model/Umpire/ImplementationLink/Tests/**`, `model/Temporal/ImplementationLinkTests/Nexus.lean`
**Touches:** [model/Temporal/System/Nexus/**, model/Umpire/ImplementationLink/Tests/**, model/Temporal/ImplementationLinkTests/Nexus.lean]

### Approach
- Consume an already-accepted System semantic trace plus explicit source setup, and re-admit its initial state and every step through the checked System kernel before translation.
- Apply the checked Implementation Link before invoking the unchanged Feature Property evaluator.
- Build independent Observation, source-admission, Implementation Link, and Property mutations under the exact non-base-System composed-test root `Temporal.ImplementationLinkTests.Nexus`.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Evaluation.lean` — Evidence-backed Model Trace boundary
- `model/Umpire/Property/Language.lean:1162-1228` — pure evaluator
- `model/Temporal/Feature/Nexus/OperationsTests.lean` — current start, cancel, and successful-completion property fixtures

### Acceptance
- [ ] Only source-kernel-admitted accepted System Model Traces reach Feature properties through checked Implementation Link.
- [ ] Observation, Implementation Link, and Property failures retain distinct diagnostics and identities.
- [ ] No runtime or raw-evidence adapter enters this task.
## Acceptance
- [ ] Only source-kernel-admitted accepted System Model Traces reach Feature properties through checked Implementation Link.
- [ ] Observation, Implementation Link, and Property failures retain distinct diagnostics and identities.
- [ ] No runtime or raw-evidence adapter enters this task.
### Acceptance
- [ ] R3–R5 positive, source target/Behavior Fingerprint/setup/transition, and independent boundary mutation matrices pass.
- [ ] An Implementation Link failure never becomes unknown evidence or a property violation.
- [ ] Feature evaluation remains unchanged.
## Done summary
Added the Nexus System-to-Feature composition operation so upstream Observation failures, checked Implementation Link failures, and unchanged Feature Property evaluations remain disjoint. Added table-driven composed coverage for start, cancellation, successful completion, source admission, coordinate, Behavior Fingerprint, Observation, and Property mutations; all post-review model and regression gates pass with the inherited malformed Go toolchain isolated under `/tmp`.

stage: impl-review - ran [Codex SHIP; receipt /tmp/impl-review-receipt-fn-32-add-umpire-refinement-and-the-first.4.json]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 6ce12ede67bdaaa4b658df540fe97908011a42ba
- Tests: baseline: green (cd model && mise exec -- lake build Umpire.ImplementationLink.Tests), baseline: green (cd model && mise exec -- lake build Temporal.System.Nexus.ImplementationLinkTests), baseline: green (cd model && mise exec -- lake build UmpireTests TemporalModelTests), baseline: red inherited tooling failure (make umpire-check-regression: generated Lean views passed, extracted Go 1.27 module-cache runtime sources were incomplete), baseline workaround: PATH=/tmp/fn32-task4-go-toolchain.tlojNU/golang.org/toolchain@v0.0.1-go1.27.0.linux-arm64/bin:$PATH GOTOOLCHAIN=local make umpire-check-regression, cd model && mise exec -- lake build Temporal.ImplementationLinkTests.Nexus, cd model && mise exec -- lake build Umpire.ImplementationLink.Tests, cd model && mise exec -- lake build Temporal.System.Nexus.ImplementationLinkTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, PATH=/tmp/fn32-task4-go-toolchain.tlojNU/golang.org/toolchain@v0.0.1-go1.27.0.linux-arm64/bin:$PATH GOTOOLCHAIN=local make umpire-check-regression, GATE_RECEIPT_NOT_WRITTEN:unittest:cd model && mise exec -- lake build Umpire.ImplementationLink.Tests - known false config/development.yaml symlink status made worktree appear dirty, GATE_RECEIPT_NOT_WRITTEN:unittest:cd model && mise exec -- lake build Temporal.System.Nexus.ImplementationLinkTests - known false config/development.yaml symlink status made worktree appear dirty, GATE_RECEIPT_NOT_WRITTEN:unittest:cd model && mise exec -- lake build UmpireTests TemporalModelTests - known false config/development.yaml symlink status made worktree appear dirty, GATE_RECEIPT_NOT_WRITTEN:smoke:make umpire-check-regression - known false config/development.yaml symlink status made worktree appear dirty
- PRs:
