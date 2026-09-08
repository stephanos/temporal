---
satisfies: [R2, R6, R9]
---
# fn-78-typed-temporal-authoring-and-checked.3 Expose typed authoring for existing Behavior constraints

## Description
Implement D2 and R2 by exposing typed authoring forms for the Behavior constraints already enforced by the checked Behavior language. The new surface must elaborate to existing canonical declarations and must not authorize Target transitions.

**Size:** M
**Files:** `model/Umpire/Behavior/{Language,Authoring,Tests/**}.lean`
**Touches:** [model/Umpire/Behavior/**]

### Approach
- Generalize the existing `ExactSequenceSpec` authoring owner with typed forms for allowed/forbidden actions, required named occurrences, occurrence bounds, ordering, and adjacency.
- Lower every surface form into the existing checked Behavior declaration before validation, canonicalization, and evaluation.
- Keep exact action-sequence and exact-trace fixtures as explicit constructs; do not silently broaden them into ordering constraints.
- Use Lean elaboration/source diagnostics where context is needed and compile-failure guards at the author expression.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Behavior/Authoring.lean:9-67` — current typed exact-sequence lowering
- `model/Umpire/Behavior/Language.lean` — canonical checked declarations
- `model/Umpire/Behavior/Tests/Canonicalization.lean` — fingerprint equivalence tests
- `model/Temporal/Feature/Nexus2/AuthoringTests.lean:112-160` — authoring equivalence/diagnostic pattern

### Key context
- Ordering permits intervening Behavior-allowed occurrences; adjacency requires consecutive semantic occurrences.
- True union, general interleaving composition, and repetition remain outside this delivery.
## Acceptance
- [ ] Typed forms cover allowed/forbidden actions, required named occurrences, occurrence bounds, ordering, and adjacency by lowering to existing checked Behavior declarations.
- [ ] Equivalent surface and constructor forms have identical canonical declarations and fingerprints.
- [ ] Tests distinguish ordering from adjacency, exactness from permitted interleaving, and declared constraints from model-impossible occurrences.
- [ ] Existing exact sequence/trace regression fixtures and their bytes/fingerprints remain unchanged.
- [ ] Wrong references, contexts, and malformed bounds fail at the authored expression with stable focused diagnostics.
- [ ] No new form authorizes a transition absent from the Target, and no union/interleaving/repetition semantics are introduced.
- [ ] Focused Behavior tests and `make umpire-build-model` pass; final architecture documentation is owned by the qualification task.
## Done summary
Implemented typed BehaviorSpec/BehaviorConstraint lowering through the unchanged canonical Behavior checker. Existing ExactSequenceSpec lowering remains unchanged, and behavior% supports both named and inline specs with the existing source diagnostics.

Task: fn-78-typed-temporal-authoring-and-checked.3
Status: complete and ready for the user to commit; changes remain uncommitted as instructed.

Changed files:
- model/Umpire/Behavior/Authoring.lean
- model/Umpire/Behavior/Tests/Authoring.lean (new)
- model/Umpire/Behavior/Tests.lean

Coverage: R2/R9 constructor/surface canonical and fingerprint equality; literal pinned exact-sequence fingerprint; exact actions versus exact trace outcomes; before/inOrder permit allowed intervening actions while adjacent rejects them; a two-occurrence constraint checks but planning with two-step limits reports impossible on the one-transition Target. R6 compile guards cover malformed bounds, missing action/occurrence references, wrong reference kind, and wrong context. Inline exact/constraint authoring guards retain expected-type elaboration.

Trust: BehaviorSpec.declaration uses [propext]; BehaviorSpec.checked uses [propext, Classical.choice, Quot.sound], matching the existing authoring boundary. No added native_decide, custom axioms, or Target/Language semantics changes.

Baseline: make lint-code failed pre-edit (/tmp/fn78-task3-baseline-lint-code.log). Initial simultaneous model build/lint suffered missing .olean artifact races; subsequent model gates were serialized. Focused red/green test logs: /tmp/fn78-task3-red.log and /tmp/fn78-task3-focused-final.log.

Verification: final make umpire-build-model passed (483 jobs; /tmp/fn78-task3-build-final.log); LEAN_NUM_THREADS=2 make lint-model passed (/tmp/fn78-task3-lint-model-final.log). Focused Behavior suite and git diff --check passed. Go lint remains inherited red: /tmp/fn78-task3-lint-code.log (GOLANGCI_LINT_FIX=false make lint-code); no Go files changed by this task.

Risks/limits: Compile-time checking needs closed inputs; local-variable specs retain the existing Except-based runtime checker path. New constraints only narrow existing Target behavior. No general union, interleaving composition, repetition, or transition authority is added. Architecture documentation remains owned by qualification.

stage: impl-review - passed(gpt-6-astra at medium; verdict SHIP; no findings)


stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: make -f Makefile -f /tmp/fn78-task3.mk fn78-behavior-tests (passed; uses repository LEAN_LAKE to build Umpire.Behavior.Tests), git diff --check (passed), make lint-code (baseline failed pre-edit), GOLANGCI_LINT_FIX=false make lint-code (inherited failure; /tmp/fn78-task3-lint-code.log), make umpire-build-model (passed; /tmp/fn78-task3-build-final.log), LEAN_NUM_THREADS=2 make lint-model (passed; /tmp/fn78-task3-lint-model-final.log), flowctl codex impl-review fn-78.3 (gpt-6-astra at medium; verdict SHIP; no findings)
- PRs: