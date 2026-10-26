---
satisfies: [R1, R7]
---
# fn-255-thorough-framework-testing-generated.2 Generated differential harness with shrinking and fixture pinning

Touches: [model/umpire/dev/**, model/umpire/backends/native/tests/**, model/umpire/backends/tlc/**, model/umpire/backends/tla/**, model/umpire/Cargo.toml, model/umpire/Cargo.lock]

## Description
Build the harness .1 chose (spec R1): generators for the admitted subset, three-way agreement, shrinking, and fixture pinning for every finding.

**Size:** M
**Files:** generator module or crate under `model/umpire/dev/` (per .1), tests, `generated_<n>` fixtures for findings.
**Touches:** [model/umpire/dev/**, model/umpire/backends/native/tests/**, model/umpire/backends/tlc/**, model/umpire/backends/tla/**, model/umpire/Cargo.toml, model/umpire/Cargo.lock]

### Approach
- Cover the admitted subset in `model/README.md` and `model/umpire/README.md` (bounded optional values, World-selected assumptions, action inputs, checked arithmetic, `reject!`/`stay!`, facts), weighted by .1. Keep spaces small by construction.
- Implement .1's complete start-set and labeled-transition comparison, including action identity, successor, outcome and facts, alongside states/verdicts. Extend TLC/emitter seams and `umpire-testing` helpers as needed; counterexample-only replay is insufficient.
- Require successful completion from each selected backend before using comparison helpers. Their matching-refusal success path must not be accepted here. Any backend error, timeout or refusal fails with its reason, including identical refusals on every backend.
- On failure, print the seed and the shrunk Model source. A helper writes it as a `generated_<n>` fixture with goldens.
- Bounds (spec R7): the generator never emits a Model above the size caps (states, edges, depth); a per-case timeout fails the case with its reason; case counts come from one committed budget table, not from environment defaults. proptest's own defaults (256 cases, unbounded shrink time) are overridden explicitly, including a shrink-iteration cap.
- Seeded-bug controls use test-only switches for a swapped comparison in emission, a dropped interpreter guard, and changes to an outcome and a fact that leave reached states/verdicts unchanged. All are caught and shrunk. A matching-refusal control fails the case.

### Investigation targets
**Required:**
- `model/design/generated-differential.md`
- `model/umpire/dev/testing/src/backend.rs`, `interpreted.rs` — agreement helpers
- `model/umpire/dev/fixtures/fixtures/` — fixture layout

### Tier agreement and evidence

Default mode records its selected native/interpreter checks and TLC skip explicitly. Required mode exercises the same bounded cases with actual TLC and fails on timeout/refusal/disagreement. Source generation remains required to cover the macro/front end; tree-only cases cannot claim that coverage. Freeze the case manifest and counts so backend agreement compares the same cases.

## Acceptance
- [ ] Required-mode source cases pass with compiled/interpreted/TLC complete starts, states, labeled edges, outcomes, facts and verdicts; any optional tree cases compare interpreter/TLC. Default-mode checks record their backend selection and TLC skip.
- [ ] Swapped-comparison, dropped-guard and outcome/fact-only controls are caught and shrunk; matching refusals fail.
- [ ] Every real disagreement found is fixed or filed, and pinned as a fixture.
- [ ] Size caps, per-case timeout, shrink cap and case counts come from the committed budget table; a control shows an over-cap generation is rejected.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
