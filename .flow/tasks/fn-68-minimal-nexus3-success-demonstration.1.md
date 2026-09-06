---
satisfies: [R1, R2]
---
# fn-68-minimal-nexus3-success-demonstration.1 Compile the compact Nexus3 success model

## Description
Build the early proof point for R1/R2 using existing typed authoring owners.

**Size:** M
**Files:** model/Temporal/Feature/Nexus3/Nexus.lean (new), model/Temporal/Feature/Nexus3/Authoring.lean (new, success-slice command syntax, elaboration, and construction helpers), model/Temporal/Feature/Nexus3/Tests.lean (new), model/TemporalModelTests.lean.
**Touches:** [model/Temporal/Feature/Nexus3/*.lean, model/TemporalModelTests.lean]

### Approach
- Preserve Nexus.md and Integration.md as design references. Add the success-only executable Nexus.lean alongside them, retaining near-verbatim `model lifecycle`, `property successfulResult`, `behavior successfulCompletion`, `limits shortTrace`, and `query completion` blocks plus the readable vocabulary order and relevant teaching comments. The feature file must not assemble Umpire records directly.
- Use three states, two wait Actions, distinct outcomes, a scheduled-only setup, and a two-row finite table. Reuse FiniteTable.checkModelTarget, PropertySpec.check, ExactSequenceSpec.check, QuerySpec.check, and IncrementalPlannerKernel.ofCheckedQuery. Keep successful Except branches explicit; no native extraction/default Target or trivial replacement capability law.
- In Authoring.lean, own the small success-slice command grammar, elaboration, provider metadata, finite admission, and ID/name handling. Each command expands to ordinary typed declarations backed by the existing Umpire owners. Accept only the demonstrated forms and report unsupported or inconsistent names at elaboration; do not build a reusable parser framework. Derive the declaration key from the block's declared identifier; lifecycle members/relations and Behavior setup/occurrences include their named owner, and an explicit per-declaration override is optional.
- Attach two checked `capability-contract` Known Gaps to the completion Query for unsupported cancellation and operation-scoped progress. Pin their codes, subjects, and details; prove they survive Query admission without altering the success Property or its fingerprint.
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
- [ ] R1's checked witness is exercised by a named test with exact states, Actions, and outcomes; Nexus.lean contains the five approachable success blocks, no direct Umpire record assembly, and stays within the spec's line bound.
- [ ] Command-level negative tests cover unsupported or inconsistent block spellings with source-local elaboration errors; the syntax expands into existing Umpire Target/Property/Behavior/Query types rather than defining parallel semantics.
- [ ] Negative tests cover invalid result state, outgoing terminal row, impossible/shortened success sequence, and no successful witness; no unsupported compiler shortcut is introduced.
- [ ] R2 tests pin `temporal.nexus3.target.lifecycle`; all lifecycle-owned role, state, Action, outcome, fact, and relation IDs; all successfulResult clause IDs; successfulCompletion setup and occurrence IDs; and `temporal.nexus3.property.successfulResult`, `temporal.nexus3.behavior.successfulCompletion`, and `temporal.nexus3.query.completion`. Rename/override behavior, a real command-layer collision, semantic-fingerprint sensitivity, and reorder/comment independence are checked without a parallel registry.
- [ ] The completion Query carries the exact checked cancellation and operation-scoped-progress Known Gaps, while its success Property and fingerprint remain unchanged.
- [ ] Focused and aggregate Lean targets pass; audit the new load-bearing definitions against the existing owner trust policy without adding axioms or bypassing checker success.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
