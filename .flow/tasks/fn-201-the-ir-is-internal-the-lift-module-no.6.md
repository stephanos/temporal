---
satisfies: [R6]
---
# fn-201-the-ir-is-internal-the-lift-module-no.6 fn-192 recut: builtins held to the checker and each Scala emitter

## Description
Rewrite fn-192's plan so the builtins table, the checker's execution and each Scala emitter's translation are held to the list by one conformance test; delete the Go interpreter row and its task. Update MILESTONES' fn-192 section.

**Files:** .flow/specs/fn-192-*.md (set-plan), .flow/tasks/fn-192-*.md, MILESTONES.md, model/builtins.json conformance test.

**Quick commands:** `python3 <flowctl> spec show fn-192`

## Acceptance
- fn-192 names no Go interpreter.
- Conformance test fails on an operator with no emitter translation, naming the emitter.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
