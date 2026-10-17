---
satisfies: [R10]
---
# fn-190-models-are-plain-scala-state-as-vars.18 Keep the user reference free of Scala source-file references

## Description
Owner request: REFERENCE.md is user-facing and must not cite Scala implementation or test files. Remove definition-source pointers and fixture/source columns, retain API examples and behavioral restrictions, and preserve the concurrent Lifted documentation. No runtime change.

## Acceptance
- [x] REFERENCE.md contains no Scala source/test filename references.
- [x] API examples, operation/lint meaning and public-surface coverage are preserved.
- [x] ReferenceSurface tests and git diff --check pass.

## Done summary
Removed Scala implementation/test filenames, definition-source paragraphs and fixture/file columns from the user-facing DSL reference. Kept the operation/lint rules, all 16 fenced API examples and the concurrent Lifted documentation. Public types previously mentioned only in source citations now appear in the existing API appendix.

Source commit: 80c7b416b9. The commit also preserves the ongoing fn-189.5 Lifted reference additions; the rest of that worker's source remains uncommitted and untouched by this task.

Verification: zero Scala filename references or source/fixture columns, git diff --check clean, and final ReferenceSurfaceTest exit 0 with 4 passing tests (.flow/tmp/fn1895/reference-verified.log). The earlier combined check read an intermediate appendix and failed on six missing type names; after those names moved into the appendix, the focused final run passed. The 42 other combined-batch checks belong to fn-189.5 and are not claimed as this task's scope.

This documentation-only follow-up does not change runtime behavior or establish held IR/Go/Temporal proofs. No full model/compiler gate was needed for a prose-only change; the active worker retained the sole compiler lane and ran the focused coverage check. Retained pre-edit reference and structural checks are in .flow/tmp/fn19018/.

stage: impl-review - skipped(config: independent review deferred to the owner milestone batch)
stage: plan-sync - skipped(config: no downstream implementation shape changed)
## Evidence
- Commits: 80c7b416b9
- Tests: timeout 600 env UMPIRE_FRAMEWORK_BUILD=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1895 mise exec -- scala-cli run --test --server=false model/framework/check --main-class org.junit.runner.JUnitCore -- framework.check.lint.ReferenceSurfaceTest (exit 0; 4 tests), git diff --check (exit 0), No matches for Scala filename references, Defined in paragraphs, Fixture/File columns in REFERENCE.md, 16 fenced code examples unchanged against .flow/tmp/fn19018/reference-before.md
- PRs: