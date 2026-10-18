---
satisfies: [R6]
---
# fn-199-native-refinement-replacement-and.5 Verdict kinds: resource limit, unsupported, declaration error; Report coverage

## Description
Add the three verdict kinds, make `verifyAll` report every refinement, replacement and composed progress result, and keep the six samples' reports unchanged where they declare none.

**Files:** model/framework/checker/Result.scala, model/framework/checker/NativeCheck.scala, sample tests.

**Quick commands:** `mise exec -- scala-cli test --server=false model/framework/checker model/framework/samples --require-tests`

## Acceptance
- Every existing consumer of `Verdict` compiles unchanged.
- Sample reports byte-identical where no refinement or composition is declared.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
