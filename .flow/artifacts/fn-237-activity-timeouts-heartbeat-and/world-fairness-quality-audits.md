### Correctness axis

## Quality Audit — Correctness axis: fn-237.6

### Summary

- Files changed: 31 in `model/` · Critical 0 · Should Fix 0 · Consider 0 · Ship: ✅ Ship, subject to required post-source gates.
- No reportable correctness findings in `52b2ed61b9..bd749e1808`. External planning commits receive no task credit.

### Evidence

- **walked**: authored guard/branch admission, selection before exact membership, conditional name reservation, unresolved refusal, retained-branch rebinding, and compiled progress metadata comparison with native override precedence.
- **executed**: diff/status inspection; later concurrent commits leave `model/` unchanged. No Cargo, gates or reviews run. Post-source canonical gate success remains **claimed** as pending by the dispatch.

### Test Budget

- Added Rust lines: 926 test / 493 implementation, including the fixture model; 1.88:1 before excluding test setup.
- No existing test files modified or assertions weakened.

### What's Good

- Source-only selectors, emitted fairness mutants, metadata mutations and five nonempty CFG families exercise distinct agreement boundaries.
- Small/large/repeated rebinding retains authored plans; exact WF emission remains with its existing owner.

Reviewer and writer use the same model family.

Artifact not written: the auditor’s developer contract prohibits all repository writes. The conductor can persist this report at `.flow/artifacts/fn-237-activity-timeouts-heartbeat-and/world-fairness-quality-correctness.md`.

### Standards axis

## Quality Audit — Standards axis: fn-237.6

### Summary

- Files changed: 32 task files · Should Fix 2 · Consider 0 · Blocking: none possible (standards axis)
- Scope: `52b2ed61b90e..bd749e1808`; concurrent planning changes excluded. Same-family reviewer, gpt-6.1-sol/high.

### Should Fix (High priority)

- **model/umpire/backends/native/src/interpret.rs:1034** (Conf 100): File crosses 1,000 lines, from 991 to 1,087 (executed, `git show`/`wc`) — extract World import/validation and its tests into a private sibling module.
- **model/README.md:416** (Conf 100): File crosses 1,000 lines, from 999 to 1,039 (executed, `git show`/`wc`) — move fixture/backend verification reference material into a linked guide, retaining the canonical authoring grammar here.

### What's Good

- Walked: one boxed conditional plan and shared primitive selector serve existing owners; native compiled callbacks remain independent.
- Cited: focused mutation tests reuse fixture/binding APIs; proof-boundary prose distinguishes source refusal, metadata disagreement, TLC replay and unevaluated callbacks.

Audit remained read-only; no tests or gates ran. Conductor must save this report to `.flow/artifacts/fn-237-activity-timeouts-heartbeat-and/world-fairness-quality-standards.md`; the auditor’s developer charter prohibits repository writes.

Correctness axis: 0 findings, worst tier none.
Standards axis: 2 findings, worst tier Should Fix.
