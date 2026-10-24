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
