---
satisfies: [R1, R2]
---
# fn-68-minimal-nexus3-success-demonstration.1 Compile the compact Nexus3 success model

## Description
Build the early proof point for R1/R2 using existing typed authoring owners.

**Size:** M
**Files:** model/Temporal/Feature/Nexus3/Nexus.lean (new), model/Temporal/Feature/Nexus3/Authoring.lean (new, only necessary local construction helpers), model/Temporal/Feature/Nexus3/Tests.lean (new), model/TemporalModelTests.lean.
**Touches:** [model/Temporal/Feature/Nexus3/*.lean, model/TemporalModelTests.lean]

### Approach
- Preserve Nexus.md and Integration.md as design references. Add the success-only executable Nexus.lean alongside them, retaining the readable vocabulary/model/Property/Behavior/Query order and relevant teaching comments.
- Use three states, two wait Actions, distinct outcomes, a scheduled-only setup, and a two-row finite table. Reuse FiniteTable.checkModelTarget, PropertySpec.check, ExactSequenceSpec.check, QuerySpec.check, and IncrementalPlannerKernel.ofCheckedQuery. Keep successful Except branches explicit; no native extraction/default Target or trivial replacement capability law.
- In Authoring.lean, centralize only repetitive provider metadata, finite admission, and ID/name handling. Use Lean's built-in decl_name% at declaration construction rather than repeating a declaration name string. Relative member/occurrence keys derive from named constructors/occurrences; explicit per-declaration override is optional. The pinned Lean source uses the declaration-name default-argument pattern in Lean/EnvExtension.lean:92.
- Express successfulResult through the existing transition-contract vocabulary for awaitSuccess. Use the exact two-step Behavior to make its trigger exercised and terminal; verify that correspondence on the selected witness. No new finalState Property evaluator.
- Add the Tests root to TemporalModelTests and document in the executable module that the cancellation/scoped-progress sketches remain unsupported.

### Investigation targets
**Required:**
- model/Temporal/Feature/Nexus3/Nexus.md:14-39 — requested teaching surface and design-only status
- model/Temporal/Feature/Nexus2/Authoring.lean:21-105 — checked constructor composition
- model/Temporal/Feature/Nexus2/Lifecycle.lean:77-129 — finite table and identity adapter
- model/Umpire/Target/FiniteMachine.lean:472-493 — admission via checkTarget
- model/Umpire/Query/Authoring.lean:19-54 — existing Query owner
**Optional:**
- model/Temporal/Feature/Nexus/Operations/SuccessfulCompletion.lean:28-112 — Property and planning proof pattern
- .plans/LEAN_GUIDELINES.md — authored proof/validation rules

### Quick commands
`cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests`
`cd model && mise exec -- lake build TemporalModelTests`

## Acceptance
- [ ] R1's checked witness is exercised by a named test with exact states, Actions, and outcomes; the standalone Nexus.lean stays within the spec's line bound.
- [ ] Negative tests cover invalid result state, outgoing terminal row, impossible/shortened success sequence, and no successful witness; no unsupported compiler shortcut is introduced.
- [ ] R2 tests pin temporal.nexus3.property.successfulResult, temporal.nexus3.behavior.successfulCompletion, and temporal.nexus3.query.completion, including rename/override/collision behavior and semantic-fingerprint sensitivity. Name capture and reorder/comment independence are checked with matched declarations, not a parallel registry.
- [ ] Focused and aggregate Lean targets pass; audit the new load-bearing definitions against the existing owner trust policy without adding axioms or bypassing checker success.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
