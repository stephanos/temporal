---
satisfies: [R1, R7]
---
# fn-255-thorough-framework-testing-generated.2 Generated differential harness with shrinking and fixture pinning

Touches: [model/umpire/dev/**, model/umpire/backends/native/tests/**, model/Cargo.toml]

## Description
Build the harness .1 chose (spec R1): generators for the admitted subset, three-way agreement, shrinking, and fixture pinning for every finding.

**Size:** M
**Files:** generator module or crate under `model/umpire/dev/` (per .1), tests, `generated_<n>` fixtures for findings.
**Touches:** [model/umpire/dev/**, model/umpire/backends/native/tests/**, model/Cargo.toml]

### Approach
- Cover the admitted subset in `model/README.md` and `model/umpire/README.md` (bounded optional values, World-selected assumptions, action inputs, checked arithmetic, `reject!`/`stay!`, facts), weighted by .1. Keep spaces small by construction.
- Agreement over starts, states, edges, outcomes, facts and claim verdicts, reusing `umpire-testing`'s existing comparison helpers. Any backend error, timeout or refusal fails the case with its reason.
- On failure, print the seed and the shrunk Model source. A helper writes it as a `generated_<n>` fixture with goldens.
- Bounds (spec R7): the generator never emits a Model above the size caps (states, edges, depth); a per-case timeout fails the case with its reason; case counts come from one committed budget table, not from environment defaults. proptest's own defaults (256 cases, unbounded shrink time) are overridden explicitly, including a shrink-iteration cap.
- Seeded-bug controls: a swapped comparison in emission and a dropped guard in the interpreter, behind test-only switches, are caught.

### Investigation targets
**Required:**
- `model/design/generated-differential.md`
- `model/umpire/dev/testing/src/backend.rs`, `interpreted.rs` — agreement helpers
- `model/umpire/dev/fixtures/fixtures/` — fixture layout

### Acceptance
- [ ] The configured case count passes with three-way agreement (two-way for tree-only cases).
- [ ] Both seeded-bug controls are caught and shrunk.
- [ ] Every real disagreement found is fixed or filed, and pinned as a fixture.
## Acceptance
- [ ] The configured case count passes with three-way agreement (two-way for tree-only cases).
- [ ] Both seeded-bug controls are caught and shrunk.
- [ ] Every real disagreement found is fixed or filed, and pinned as a fixture.

### Acceptance
- [ ] TBD
- [ ] Size caps, per-case timeout, shrink cap and case counts come from the committed budget table; a control shows an over-cap generation is rejected.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
