---
satisfies: [R1]
---
# fn-68-minimal-nexus3-success-demonstration.1 Compile the compact Nexus3 success model

## Description
Build the early proof point for R1/R2 using existing typed authoring owners.

**Size:** M
**Files:** model/Temporal/Feature/Nexus3/Nexus.lean (new), model/Temporal/Feature/Nexus3/Syntax.lean (new, success-slice grammar and macro expansion only), model/Temporal/Feature/Nexus3/Authoring.lean (new, typed construction and admission only), model/Temporal/Feature/Nexus3/Tests.lean (new), model/TemporalModelTests.lean.
**Touches:** [model/Temporal/Feature/Nexus3/*.lean, model/TemporalModelTests.lean]

### Approach
- Preserve Nexus.md and Integration.md as design references. Add the success-only executable Nexus.lean alongside them, retaining near-verbatim `model lifecycle`, `property successfulResult`, `behavior successfulCompletion`, `limits shortTrace`, and `query completion` blocks plus the readable vocabulary order and relevant teaching comments. The feature file must not assemble Umpire records directly.
- Use three states, two wait Actions, distinct outcomes, a scheduled-only setup, and a two-row finite table. Reuse FiniteTable.checkModelTarget, PropertySpec.check, ExactSequenceSpec.check, QuerySpec.check, and IncrementalPlannerKernel.ofCheckedQuery. Keep successful Except branches explicit; no native extraction/default Target or trivial replacement capability law.
- In Syntax.lean, own only the small success-slice command grammar and macro expansion. Each command expands to ordinary typed declarations backed by `Nexus3.Authoring`; accept only the demonstrated semantic forms and report unsupported or inconsistent names at elaboration. In Authoring.lean, own provider metadata, finite admission, checked planning, and initial ID/name handling without declaring syntax. Do not build a reusable parser framework. Task 4 immediately follows to remove remaining handwritten metadata, pinned-ID tests, and override plumbing before lowering.
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
- .plans/lean/LEAN_GUIDELINES.md — authored proof/validation rules

### Quick commands
`cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests`
`cd model && mise exec -- lake build TemporalModelTests`

## Acceptance
- [ ] R1's checked witness is exercised by a named test with exact states, Actions, and outcomes; Nexus.lean contains the five approachable success blocks, imports `Nexus3.Syntax`, contains no direct Umpire record assembly, and stays within the spec's line bound. Syntax.lean contains the macros but no semantic record owners; Authoring.lean contains the semantic owners but no syntax declarations.
- [ ] Command-level negative tests cover unsupported or inconsistent block spellings with source-local elaboration errors; the syntax expands into existing Umpire Target/Property/Behavior/Query types rather than defining parallel semantics.
- [ ] Negative tests cover invalid result state, outgoing terminal row, impossible/shortened success sequence, and no successful witness; no unsupported compiler shortcut is introduced.
- [ ] Declaration/reference consistency is checked sufficiently to support the exact witness. Task 4 owns the final derived-metadata policy, rename tests, removal of overrides/pinned IDs, and semantic-fingerprint sensitivity.
- [ ] The completion Query carries the exact checked cancellation and operation-scoped-progress Known Gaps, while its success Property and fingerprint remain unchanged.
- [ ] Focused and aggregate Lean targets pass; audit the new load-bearing definitions against the existing owner trust policy without adding axioms or bypassing checker success.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: