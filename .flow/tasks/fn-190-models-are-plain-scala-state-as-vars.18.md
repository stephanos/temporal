---
satisfies: [R10]
---
# fn-190-models-are-plain-scala-state-as-vars.18 Keep the user reference free of Scala source-file references

## Description
Owner request: REFERENCE.md is user-facing and must not cite Scala implementation or test files. Remove definition-source pointers and fixture/source columns, retain API examples and behavioral restrictions, and preserve the concurrent Lifted documentation. No runtime change.

## Acceptance
- [ ] REFERENCE.md contains no Scala source/test filename references.
- [ ] API examples, operation/lint meaning and public-surface coverage are preserved.
- [ ] ReferenceSurface tests and git diff --check pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
