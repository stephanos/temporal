---
satisfies: [R1, R4, R5, R8]
---
# fn-62-make-ordinary-temporal-model-authoring.4 Simplify the ordinary Property Behavior Query journey

## Description
After task `.3` and `fn-58` freeze the reusable facade, satisfy R1, R4, R5, and R8 by centralizing repeated checked declaration composition and migrating the three ordinary Nexus operations.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Operations/Internal.lean`, `model/Temporal/Feature/Nexus/Operations/AsyncStart.lean`, `model/Temporal/Feature/Nexus/Operations/Cancellation.lean`, `model/Temporal/Feature/Nexus/Operations/SuccessfulCompletion.lean`, `model/Temporal/Feature/Nexus/OperationsTests.lean`
**Touches:** [model/Temporal/Feature/Nexus/Operations/Internal.lean, model/Temporal/Feature/Nexus/Operations/AsyncStart.lean, model/Temporal/Feature/Nexus/Operations/Cancellation.lean, model/Temporal/Feature/Nexus/Operations/SuccessfulCompletion.lean, model/Temporal/Feature/Nexus/OperationsTests.lean]

### Approach
- Consume only the public Property facade delivered by `fn-58-partition-the-property-language` and the primitives from task `.3`; do not couple helpers to checker internals.
- Reuse `BehaviorDeclaration.exactlyOneAction`, setup helpers, and Target context adapters in `model/Umpire/Behavior/Language.lean:76-205` rather than wrapping Behavior in a new representation.
- Deepen `model/Temporal/Feature/Nexus/Operations/Internal.lean:17-39` to centralize family IDs, author source, and the mechanical raw-check/result/query sequence while keeping raw results and explicit checker-success proofs available.
- Use the named Limits and transition-result constructors from task `.3`; keep every Action, state, outcome, observation, capability, and Limit explicit.
- Migrate the repeated shapes in AsyncStart, Cancellation, and SuccessfulCompletion; preserve wrong-Action/wrong-outcome examples and every existing comment.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus/Operations/Internal.lean:17-39` — shared operation source/query mechanics.
- `model/Umpire/Property/Check.lean:450-486` — sole raw checker and proof-taking extraction contract.
- `model/Umpire/Behavior/Language.lean:76-205` — existing ordinary Behavior helpers and Target context.
- `model/Umpire/Query/Language.lean:523-583` — raw and proof-taking Query boundary.
- `model/Temporal/Feature/Nexus/Operations/AsyncStart.lean:17-103` — representative repeated operation journey.

**Optional** (reference as needed):
- `model/Temporal/Feature/Nexus/Operations/Cancellation.lean:18-103` — second repeated operation.
- `model/Temporal/Feature/Nexus/Operations/SuccessfulCompletion.lean:17-103` — third repeated operation.

### Key context
AUT-07 prohibits an umbrella operation DSL. Helpers must construct the existing Property, Behavior, and Query languages and call their existing checkers. The author remains responsible for explicit checker-success evidence; the library must not choose `native_decide`.

### Acceptance
- [ ] The three operation modules use shared explicit Action→state/outcome/observation, family-ID, source, named-Limit, and raw-check/query mechanics.
- [ ] Each operation retains readable Property, Behavior, and Query declarations plus directly inspectable raw typed results.
- [ ] Checked extraction still requires explicit proof, and omitted proof remains an elaboration error.
- [ ] Invalid clauses, missing capabilities, unsatisfiable Behavior, Target mismatch, and invalid Limits retain exact diagnostics and precedence.
- [ ] Existing operation IDs, fingerprints, intended/wrong traces, deterministic plans, artifacts, public imports, and comments remain exact.

## Acceptance
- [ ] R1, R4, R5, and R8 are satisfied without creating a fourth authoring language.
- [ ] `cd model && mise exec -- lake build Temporal.Feature.Nexus.OperationsTests` passes.
- [ ] Raw invalid-declaration APIs remain directly testable and no hidden compiler-trust path is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
